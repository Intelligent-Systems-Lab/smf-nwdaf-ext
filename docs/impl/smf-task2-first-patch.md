# SMF Task2 Patch1 實作說明（Nsmf_EventExposure 訂閱端點）

## 觸碰檔案清單

- `smf/pkg/factory/config.go`
- `smf/internal/context/context.go`
- `smf/internal/context/nsmf_eventexposure_subscription.go`
- `smf/internal/sbi/api_eventexposure.go`

## 新增/啟用的端點

- `POST /nsmf-event-exposure/v1/subscriptions`
- `DELETE /nsmf-event-exposure/v1/subscriptions/{subId}`

> `GET/PUT` 仍維持未實作（回 501）。

## Contract 範例（照 docs/contract/nsmf-eventexposure-subscription.md）

**Create Subscription**

```bash
curl -k -X POST https://smf.5gc.net/nsmf-event-exposure/v1/subscriptions \
  -H 'Content-Type: application/json' \
  -d '{
    "notifId": "nwdaf-uecom-0001",
    "notifUri": "http://nwdaf.example.com/nsmf-ee/v1/notify",
    "supi": "imsi-208930000000001",
    "nfId": "550e8400-e29b-41d4-a716-446655440000",
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
        ]
      }
    ]
  }'
```

**Delete Subscription**

```bash
curl -k -X DELETE https://smf.5gc.net/nsmf-event-exposure/v1/subscriptions/sub-123
```

## 已知限制

- 尚未實作 SMF->UPF 級聯訂閱/退訂（Patch2 才會補上）。
- 僅支援單一 UE：要求 `supi` 存在，且 `eventSubs` 需包含 `UPF_EVENT` 與 `USER_DATA_USAGE_MEASURES`。
