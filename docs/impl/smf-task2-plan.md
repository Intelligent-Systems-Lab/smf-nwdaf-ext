# Task2 Patch 0 Plan (No Code Changes)

This document is the concrete implementation plan for Task2 (NWDAF -> SMF -> UPF event exposure subscription), based on the authoritative inputs and the fixed semantics provided.

## A) File Touch List (Exact Paths)

### Existing files to modify in later patches
- `smf/internal/sbi/api_eventexposure.go` (implement POST/GET/PUT/DELETE handlers)
- `smf/internal/sbi/server.go` (route registration already present; confirm wiring and add comments if needed)
- `smf/pkg/factory/config.go` (may extend config model for UPF NUPF API roots)
- `smf/smfcfg.yaml` (add per-UPF Nupf Event Exposure apiRoot if needed)
- `smf/internal/context/sm_context.go` (add helper lookup by SUPI; add subscription state pointer)
- `smf/internal/context/upf.go` (extend UPF/UPNode or config mapping to carry NUPF apiRoot)
- `smf/internal/sbi/consumer/consumer.go` (register NUPF client service)

### New files (proposed)
- `smf/internal/context/nwdaf_event_subscriptions.go` (in-memory state model + mutex + indexes)
- `smf/internal/sbi/consumer/nupf_eventexposure_service.go` (NUPF Event Exposure client wrapper)
- `smf/internal/sbi/processor/event_exposure.go` (shared logic for create/delete subscription)
- `smf/docs/impl/smf-task2-plan.md` (this plan)

## B) Where to Register Nsmf_EventExposure Routes on SMF Startup

- Routing is already registered in `smf/internal/sbi/server.go` inside `newRouter()` when `ServiceNameList` includes `models.ServiceName_NSMF_EVENT_EXPOSURE`.
- The router group uses `factory.SmfEventExposureResUriPrefix` (`/nsmf-event-exposure/v1`) from `smf/pkg/factory/config.go`.
- Ensure `smf/smfcfg.yaml` includes `nsmf-event-exposure` in `configuration.serviceNameList` (currently present).

## C) SMF-Internal Resolution Path (SUPI -> Session -> UE IP -> UPF apiRoot)

1) **Parse and validate request:**
   - Require `supi` (or `pduSeId` if future extension is desired; Task2 assumes SUPI path).
   - Require `eventSubs` with `event=UPF_EVENT` and `upfEvents` content.
   - Enforce fixed semantics: `bundledEventNotifyUri` is required, `eventNotifyUri` must equal it, `notifId` binds to `notifyCorrelationId`.

2) **Resolve SMContext:**
   - If `pduSeId` is present, use `smf/internal/context.GetSMContextById(supi, pduSeId)`.
   - If `pduSeId` is not present, add a helper that scans `smContextPool` for an active session matching `Supi` (return error if none or multiple ambiguous sessions).

3) **Extract UE IP:**
   - Use `smContext.PDUAddress` (must be non-nil). Reject if missing.

4) **Select UPF:**
   - Use `smContext.SelectedUPF` (must be non-nil). This already represents the selected anchor UPF from `SelectUPFAndAllocUEIP()`.

5) **Resolve NUPF apiRoot:**
   - **Proposed config-based mapping**: extend UPF config in `smf/smfcfg.yaml` to include `nupfEventExposureApiRoot` per UPF node.
   - Attach this apiRoot to the runtime UPF/UPNode object during `InitSmfContext`, so it can be read from `smContext.SelectedUPF`.
   - If apiRoot is missing, reject with a 500 (configuration error) or 502 (upstream not available), depending on error strategy.

6) **Build NUPF subscription:**
   - `ueIpAddress` := `smContext.PDUAddress`.
   - `notifyCorrelationId` := `notifId` from Nsmf request.
   - `eventNotifyUri` := `bundledEventNotifyUri` from Nsmf request (fixed semantic).
   - `eventReportingMode` := map from `notifMethod`/`repPeriod`.

## D) In-Memory State Model (with Mutex) and Key Strategy

Create a global store (owned by SMF context) to keep Nsmf and Nupf subscription bindings.

### Proposed structs
- `type NwdafNsmfSubscription struct {`
  - `NsmfSubId string`
  - `Supi string`
  - `NotifId string`
  - `RepPeriod int32`
  - `NotifMethod models.NotificationMethod`
  - `MeasurementTypes []models.MeasurementType`
  - `Granularity models.GranularityOfMeasurement`
  - `BundledEventNotifyUri string`
  - `EventSubs []models.EventSubscription`
  - `CreatedAt time.Time`
- `}`

- `type NwdafUpfBinding struct {`
  - `NsmfSubId string`
  - `UpfSubId string`
  - `UpfLocation string`
  - `SelectedUpfApiRoot string`
  - `UpfNodeName string`
  - `CreatedAt time.Time`
- `}`

- `type NwdafSubStore struct {`
  - `mu sync.RWMutex`
  - `nsmfById map[string]*NwdafNsmfSubscription`  
  - `upfByNsmfId map[string]*NwdafUpfBinding`  
  - `nsmfByNotifId map[string]string // notifId -> nsmfSubId (optional index)`
- `}`

### Key strategy
- Primary key: `nsmfSubId`.
- Binding key required by Task2 semantics: `notifId` (Nsmf) == `notifyCorrelationId` (Nupf). Keep an optional `notifId -> nsmfSubId` index for quick reverse lookups.
- Deletion should remove all indexes in a single locked operation.

## E) Error Mapping Strategy (TS29571 ProblemDetails)

Use `application/problem+json` with `ProblemDetails` and `invalidParams` where applicable.

1) **Invalid/missing required fields**
   - Missing `bundledEventNotifyUri`, missing `eventSubs`, missing `notifId`, missing `supi`, `event != UPF_EVENT`, or `upfEvents` empty -> **400 Bad Request**.
   - Include `invalidParams` entries with `param` paths (e.g., `eventSubs[0].bundledEventNotifyUri`).
   - If `eventNotifyUri` in UPF subscription would not equal `bundledEventNotifyUri` -> **400 Bad Request** (violates fixed semantic).

2) **No active session / UE IP not found**
   - `GetSMContextById` returns nil, or `smContext.PDUAddress` is nil -> **404 Not Found**.
   - Detail should indicate that no active PDU session/UE IP is available for the SUPI.

3) **UPF subscribe failure**
   - Transport errors/timeouts or UPF 5xx -> **502 Bad Gateway**.
   - UPF 4xx -> **502 Bad Gateway** unless the failure is clearly caused by client input (then **400** with explicit detail). The plan favors a conservative 502 to avoid exposing UPF-specific semantics to NWDAF.
   - If apiRoot/config missing -> **500 Internal Server Error** (SMF misconfiguration).

## F) Patch Sequence (Patch1..Patch4) with Acceptance Criteria

### Patch1: State store + request validation skeleton
- Add `NwdafSubStore` to SMF context and initialize at startup.
- Implement request parsing helpers (no network calls yet) and enforce fixed semantics:
  - `bundledEventNotifyUri` required.
  - `eventNotifyUri == bundledEventNotifyUri`.
  - `notifId` binding key.
- Acceptance criteria:
  - Unit tests or handler stubs validate errors for missing/invalid fields.
  - No behavioral change to existing SMF functions outside the new paths.

### Patch2: SUPI -> SMContext -> UE IP -> UPF apiRoot resolution
- Add helper to find SMContext by SUPI (and optional pduSeId) and extract UE IP + SelectedUPF.
- Add apiRoot mapping to UPF config and load into runtime structures.
- Acceptance criteria:
  - For a known active SUPI, resolution returns `PDUAddress`, `SelectedUPF`, and `SelectedUpfApiRoot`.
  - For unknown/idle SUPI, returns 404 ProblemDetails.

### Patch3: NUPF subscription create + state persistence
- Add NUPF client wrapper and create subscription call.
- Store `nsmfSubId -> NsmfSub` and `nsmfSubId -> UpfBinding` atomically.
- Respond `201 Created` with Location and Nsmf subscription body (per TS 29.508).
- Acceptance criteria:
  - NUPF subscription request uses `notifyCorrelationId = notifId` and `eventNotifyUri = bundledEventNotifyUri`.
  - SMF does NOT send any notification to NWDAF (only UPF does).

### Patch4: Delete/Get/Replace handlers
- Implement DELETE to remove SMF state and call UPF delete.
- Implement GET to return stored Nsmf subscription.
- Implement PUT replace (optional: for Task2 V0, return 501/409 if not supported; or fully replace with re-subscribe).
- Acceptance criteria:
  - DELETE returns 204 on success and clears local state.
  - GET returns 200 with stored subscription.
  - Replace behavior is defined and documented (either not supported or fully implemented).

---

## Fixed Semantics Checklist (Must Hold in All Patches)
- UPF Notify goes directly to NWDAF; SMF does not relay.
- `notifUri` accepted but unused in V0.
- `bundledEventNotifyUri` is required; reject if missing.
- UPF `eventNotifyUri` must equal `bundledEventNotifyUri`.
- `notifId` (Nsmf) must map to `notifyCorrelationId` (Nupf).
