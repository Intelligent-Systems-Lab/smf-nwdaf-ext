# Task2 Patch 3 - Nsmf Delete Cascade to UPF

## Sequence
1) Nsmf DELETE `/nsmf-event-exposure/v1/subscriptions/{subId}`.
2) Lookup local subscription state by `subId`.
3) If `upfLocation` is present, issue UPF DELETE against that Location (or apiRoot + id).
4) Always delete local state regardless of UPF delete outcome.
5) Return `204 No Content` to NWDAF on successful local cleanup.

## Idempotency Notes
- If the UPF subscription is already gone or UPF delete fails, SMF logs WARN with status/body summary.
- Local cleanup is deterministic and always performed once the Nsmf subscription exists.
- If the Nsmf subscription does not exist, SMF returns ProblemDetails 404.

## Logging
- Delete request logs: `nsmf_sub_id`.
- Cascade delete logs: `upf_sub_id`, `upf_location`, `selected_upf_api_root`, and UPF HTTP status/body summary.
