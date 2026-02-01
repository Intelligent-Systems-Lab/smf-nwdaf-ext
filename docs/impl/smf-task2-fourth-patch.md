# Task2 Patch 4 - Interfaces and TODOs (No V0 Behavior Change)

## Stubs Added
- Nsmf_EventExposure:
  - `GET /nsmf-event-exposure/v1/subscriptions/{subId}` now returns 501 ProblemDetails.
  - `PUT /nsmf-event-exposure/v1/subscriptions/{subId}` now returns 501 ProblemDetails.

## Future Extension Points
- TS 29.508:
  - Any UE / group UE subscriptions (anyUeInd, groupId).
  - Additional SMF event types beyond UPF_EVENT.
  - GET list and GET individual subscription bodies.
  - Replace/modify subscription semantics.
- TS 29.564:
  - ModifySubscription (PATCH).
  - Query/list subscriptions by UPF.

## V0 Behavior Confirmation
- Create/Delete behavior is unchanged.
- Only SUPI + USER_DATA_USAGE_MEASURES + PERIODIC + PER_SESSION are supported.
