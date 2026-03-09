# Event Exposure Contract (EES Branch)

## Audience

- Users integrating NWDAF with this SMF branch
- Maintainers validating branch-specific API behavior

## Scope

This document defines the implemented contract in `ees-only`, not the full TS 29.508/29.564 surface.

## Nsmf_EventExposure (SMF SBI)

Base path:
- `/nsmf-event-exposure/v1/subscriptions`

Implemented operations:
- `POST /nsmf-event-exposure/v1/subscriptions`
- `DELETE /nsmf-event-exposure/v1/subscriptions/{subId}`

Not implemented in this branch:
- `GET /nsmf-event-exposure/v1/subscriptions/{subId}`
- `PUT /nsmf-event-exposure/v1/subscriptions/{subId}`

### Create Subscription

Required request characteristics:
- single-UE mode only (`supi` required)
- `eventSubs` must include `UPF_EVENT`
- `upfEvents` must include `USER_DATA_USAGE_MEASURES`
- `granularityOfMeasurement` must be `PER_SESSION`
- measurement types currently accepted:
  - `VOLUME_MEASUREMENT`
  - `THROUGHPUT_MEASUREMENT`

Response behavior:
- `201 Created` on success
- `Location` header is set to the created Nsmf subscription resource
- returns `ProblemDetails` on validation/resolution/upstream failures

State behavior:
- SMF resolves SUPI -> SMContext -> UE IP + selected UPF apiRoot first
- SMF calls UPF create subscription
- SMF stores local subscription state only after UPF create succeeds
- UPF target is single-valued per subscription (resolved from `SMContext.SelectedUPF`)

### Delete Subscription

Behavior:
- if local subscription exists, SMF attempts UPF DELETE first when linkage is available
- local state is always deleted afterward (deterministic local cleanup)
- if subscription is missing, returns `404 ProblemDetails`

Response behavior:
- `204 No Content` when local subscription existed and was removed

## Nupf_EventExposure Mapping (SMF -> UPF)

UPF endpoint used:
- `POST {nupfEeApiRoot}/nupf-ee/v1/ee-subscriptions`
- `DELETE {Location returned by UPF}`

Mapping rules:
- Nsmf `notifId` -> UPF `notifyCorrelationId`
- Nsmf `bundledEventNotifyUri` -> UPF `eventNotifyUri`
- SUPI-resolved UE IP -> UPF `ueIpAddress`
- event type fixed to `USER_DATA_USAGE_MEASURES` in current branch scope
- current target selection uses only `SelectedUPF`; no fan-out to multiple UPFs on one path

Reporting mode:
- trigger fixed to `PERIODIC`
- `repPeriod` precedence:
  - request `repPeriod`
  - fallback to `configuration.urrPeriod`
  - fallback default `10`

## Config Contract Additions

Top-level configuration:
- `nupfEeNfId` (optional UUID)
- `nupfEeReqTimeout` (duration)
- `nupfEeMaxRetries` (int)

Per-UPF node configuration:
- `userplaneInformation.upNodes.<UPF>.nupfEeApiRoot` (required for UPF cascade behavior)

## Out Of Scope

- any-UE / group-UE subscription semantics
- full event catalog beyond branch-fixed UPF usage event flow
- persistent subscription storage backend
- full GET/PUT lifecycle semantics
- multi-target subscription in chained topologies (for example `AN -> I-UPF -> PSA-UPF`)

## Reference Specs

- `docs/spec/TS29508_Nsmf_EventExposure.yaml`
- `docs/spec/TS29564_Nupf_EventExposure.yaml`
- `docs/spec/TS29571_CommonData.yaml`
