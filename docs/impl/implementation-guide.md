# Implementation Guide

## Audience

- Maintainers extending or debugging the EES branch implementation

## Scope

Implementation architecture for:
- Nsmf create/delete lifecycle
- Nsmf <-> Nupf cascade behavior
- resolver and in-memory state model
- multi-AN user-plane selection fix

## Code Map

Nsmf API handlers:
- `internal/sbi/api_eventexposure.go`

Nupf HTTP client:
- `internal/sbi/nupf_eventexposure_client.go`

Resolver and state store:
- `internal/context/nupf_event_exposure_resolver.go`
- `internal/context/nsmf_event_exposure_store.go`

Config model and defaults:
- `pkg/factory/config.go`
- `smfcfg.yaml`

User-plane selection logic:
- `internal/context/user_plane_information.go`
- `internal/context/user_plane_information_test.go`

URR threshold guard:
- `internal/context/pfcp_rules.go`

## Lifecycle Sequence

### Create (`POST /subscriptions`)

1. Parse request body and validate branch constraints.
2. Resolve SUPI to one active `SMContext`.
3. Extract `PDUAddress` and `SelectedUPF.NupfEeApiRoot`.
4. Build UPF request payload and call UPF create API.
5. Persist Nsmf subscription state with UPF linkage.
6. Return `201 Created` with Nsmf `Location`.

Key invariant:
- local subscription state is not persisted before UPF create success.
- only one UPF target is used per subscription (`SelectedUPF`).

### Current Topology Limitation

Current behavior:
- SMF resolves a single target via `SMContext.SelectedUPF`.
- subscription cascade is sent only to that one UPF.

Implication:
- in a chained path such as `AN -> I-UPF -> PSA-UPF`, this branch does not subscribe both UPFs.
- multi-UPF fan-out/coordination is not implemented in current scope.

### Delete (`DELETE /subscriptions/{subId}`)

1. Lookup local state by `subId`.
2. If UPF linkage exists, call UPF delete.
3. Delete local state regardless of UPF delete result.
4. Return `204 No Content` when local state existed.

Key invariant:
- local cleanup is deterministic when local subscription exists.

## State Model

Stored state fields include:
- Nsmf ID and request identity (`subId`, `supi`, `notifId`, `notifUri`)
- selected measurement parameters
- resolved target (`ueIpAddress`, `selectedUpfApiRoot`)
- UPF linkage (`upfSubscriptionId`, `upfSubscriptionLocation`)
- creation timestamp

Storage characteristics:
- in-memory map with RW lock
- process-local and non-persistent

## Multi-AN Selection Behavior

Problem addressed:
- map iteration order could pick an unrelated AN source in multi-AN disconnected topologies.

Implemented approach:
- evaluate candidate AN sources deterministically
- aggregate reachable anchor UPFs across AN sources
- avoid relying on random map ordering

Outcome:
- UE sessions in multi-gNB/multi-UPF disconnected topologies can select the correct UPF path.

## URR Threshold Guard

Behavior:
- `NewVolumeThreshold()` enables `Volth` only when `threshold > 0`.

Project-specific note:
- this was added for this team's extended `UPF-EES` test requirements.

## Extension Points

If future scope expands:
- add GET/PUT lifecycle handling in `api_eventexposure.go`
- add persistence backend for subscription state
- support additional event and filter variants
- add richer resolver keys when SUPI has multiple active contexts by design
- add multi-UPF subscription targeting for chained data paths
