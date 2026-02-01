# Task2 Patch 2 - Nsmf -> Nupf Cascading Subscription

## Resolver Path
1) `supi` from Nsmf Create request.
2) `GetSMContextBySupi(supi)` scans SMF context pool.
3) From SMContext:
   - `PDUAddress` -> `ueIpAddress`.
   - `SelectedUPF` -> `NupfEeApiRoot`.
4) If any resolution step fails, respond with ProblemDetails and do not store the Nsmf subscription.

## Mapping Rules
- `notifId` (Nsmf) -> `notifyCorrelationId` (UPF)
- `bundledEventNotifyUri` (Nsmf) -> `eventNotifyUri` (UPF)
- UPF event list is fixed to `USER_DATA_USAGE_MEASURES`.
- `granularityOfMeasurement` is `PER_SESSION`.

## Error Behavior
- No active session / missing UE IP / missing selected UPF -> `404` ProblemDetails.
- Multiple active sessions for SUPI -> `409` ProblemDetails.
- Missing UPF apiRoot configuration -> `500` ProblemDetails.
- UPF subscription failure -> `502` ProblemDetails.

## Notes
- The UPF request body is wrapped with `{ "subscription": { ... } }` as required by TS 29.564.
- `repPeriod` precedence: request value if present; otherwise SMF config `urrPeriod`; fallback to `10` seconds.
- `nfId` source: `configuration.nupfEeNfId` if configured, else `configuration.nfInstanceId`.
- Optional timeouts/retries: `configuration.nupfEeReqTimeout`, `configuration.nupfEeMaxRetries`.
