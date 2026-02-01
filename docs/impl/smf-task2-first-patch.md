# Task2 Patch 1 - Nsmf_EventExposure Create/Delete

## Touched Files
- `smf/internal/sbi/api_eventexposure.go`
- `smf/internal/context/nsmf_event_exposure_store.go`

## Endpoints and Status Codes
- `POST /nsmf-event-exposure/v1/subscriptions`
  - `201 Created` (Location header set to `{apiRoot}/nsmf-event-exposure/v1/subscriptions/{subId}`)
  - `400 Bad Request` (ProblemDetails for invalid/missing fields)
- `DELETE /nsmf-event-exposure/v1/subscriptions/{subId}`
  - `204 No Content` (subscription removed)
  - `404 Not Found` (ProblemDetails when subscription is missing)
  - `400 Bad Request` (ProblemDetails if subId path is empty)

## Validation Rules (Create)
- `supi` is required (V0 single UE only).
- `notifId` is required and stored for later binding to UPF `notifyCorrelationId`.
- `notifUri` is required (accepted but not used in V0).
- `eventSubs` must be present and each `event` must be `UPF_EVENT`.
- `upfEvents` must include an item with `type` = `USER_DATA_USAGE_MEASURES`.
- `measurementTypes` must exist and contain at least one item; only
  `VOLUME_MEASUREMENT` and `THROUGHPUT_MEASUREMENT` are allowed.
- `granularityOfMeasurement` must be present and set to `PER_SESSION`.
- `bundledEventNotifyUri` is required (missing -> ProblemDetails).

## Curl Examples

Create:
```bash
curl -X POST \
  http://127.0.0.2:8000/nsmf-event-exposure/v1/subscriptions \
  -H 'Content-Type: application/json' \
  -d '{
    "supi": "imsi-208930000000001",
    "notifUri": "http://nwdaf.example.com/nsmf-ee/v1/notify",
    "notifId": "nwdaf-uecom-0001",
    "eventSubs": [
      {
        "event": "UPF_EVENT",
        "upfEvents": [
          {
            "type": "USER_DATA_USAGE_MEASURES",
            "measurementTypes": [
              "VOLUME_MEASUREMENT",
              "THROUGHPUT_MEASUREMENT"
            ],
            "granularityOfMeasurement": "PER_SESSION"
          }
        ],
        "bundlingAllowed": true,
        "bundledEventNotifyUri": "http://127.0.0.1:8080/collector/upf-notify"
      }
    ],
    "notifMethod": "PERIODIC",
    "repPeriod": 10
  }'
```

Delete:
```bash
curl -X DELETE \
  http://127.0.0.2:8000/nsmf-event-exposure/v1/subscriptions/sub-123
```
