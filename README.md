# SMF EES Extension Branch (`ees-only`)

This branch extends `free5gc/smf` `main` for EES-driven NWDAF/SMF/UPF event exposure workflows and multi-AN deployments.

## Scope

Base reference:
- Upstream: `free5gc/smf` `upstream/main`
- Branch: `ees-only`

Primary goals in this branch:
- Implement usable `Nsmf_EventExposure` create/delete flows.
- Cascade Nsmf subscriptions to UPF `Nupf_EventExposure` subscriptions.
- Add resolver/store plumbing for SUPI -> SMContext -> UPF mapping.
- Add EES-oriented configuration and topology behavior.

## Delta Vs Upstream Main

Compared with `upstream/main`, this branch adds these functional extensions:
- Full `Nsmf_EventExposure` handler logic instead of 501 stubs.
- SMF -> UPF subscription create/delete HTTP client for `Nupf_EventExposure`.
- In-memory subscription state store for lifecycle management.
- SUPI resolver for selecting UE IP and target UPF API root.
- New config fields for UPF event exposure API root and client policy.
- Multi-AN ingress selection fix for disconnected AN-UPF topologies.
- URR volume-threshold guard: trigger is enabled only when `threshold > 0`.

## Implemented Extensions

### 1) Nsmf Event Exposure API

File:
- `internal/sbi/api_eventexposure.go`

Implemented behavior:
- `POST /nsmf-event-exposure/v1/subscriptions`
- `DELETE /nsmf-event-exposure/v1/subscriptions/{subId}`
- strict request validation with `ProblemDetails`
- deterministic `Location` generation for created Nsmf subscription
- local state persisted only after successful UPF subscription creation
- delete flow always performs local cleanup even if UPF delete fails

Current non-goals in this branch:
- GET/PUT subscription management remain unsupported.
- No persistence backend (store is process-local memory).

### 2) Nupf Event Exposure Client

File:
- `internal/sbi/nupf_eventexposure_client.go`

Implemented behavior:
- create UPF subscription via `POST /nupf-ee/v1/ee-subscriptions`
- delete UPF subscription via `DELETE` against returned Location
- configurable timeout/retry policy
- transport and upstream error mapping to `ProblemDetails`

### 3) Resolver and Subscription Store

Files:
- `internal/context/nupf_event_exposure_resolver.go`
- `internal/context/nsmf_event_exposure_store.go`

Implemented behavior:
- resolve a single active SMContext from SUPI
- extract UE IP and selected UPF API root for NUPF operations
- keep Nsmf <-> Upf subscription linkage in memory for delete cascade

### 4) Configuration Model Extensions

Files:
- `pkg/factory/config.go`
- `smfcfg.yaml`

Added fields:
- `configuration.nupfEeNfId`
- `configuration.nupfEeReqTimeout`
- `configuration.nupfEeMaxRetries`
- `configuration.userplaneInformation.upNodes.<UPF>.nupfEeApiRoot`

Also aligned event exposure URI prefix to:
- `/nsmf-event-exposure/v1/subscriptions`

### 5) User Plane Behavior Extensions

File:
- `internal/context/user_plane_information.go`

Implemented behavior:
- supports multi-AN source selection for UPF/path selection logic
- avoids map-order-dependent AN choice under disconnected topologies

Regression coverage:
- `internal/context/user_plane_information_test.go`

### 6) URR Guard For EES Test Compatibility

File:
- `internal/context/pfcp_rules.go`

Behavior:
- `NewVolumeThreshold()` only sets `Volth` when `threshold > 0`.

Note:
- This guard was added for this project team's own extended `UPF-EES` test requirements.

## Quick Configuration Checklist

1. Enable `nsmf-event-exposure` in `serviceNameList`.
2. For each UPF node used by EES, set `nupfEeApiRoot`.
3. Optionally tune `nupfEeReqTimeout` and `nupfEeMaxRetries`.
4. If needed, set explicit `nupfEeNfId`; otherwise SMF `nfInstanceId` is used.

## Documentation

See:
- `docs/README.md`
- `docs/contract/event-exposure-contract.md`
- `docs/impl/implementation-guide.md`
- `docs/impl/operations-and-troubleshooting.md`
- `docs/spec/TS29508_Nsmf_EventExposure.yaml`
- `docs/spec/TS29564_Nupf_EventExposure.yaml`
- `docs/spec/TS29571_CommonData.yaml`

## Maintainer Notes

This branch intentionally focuses on Task2/EES event exposure flow and related operability fixes. It is not a full replacement for all TS 29.508/29.564 features.
