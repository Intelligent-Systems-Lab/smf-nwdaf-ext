# SMF Task2 Patch2 實作說明（級聯 SMF->UPF Nupf_EventExposure）

## Resolver 邏輯摘要

- 入口：`smf/internal/sbi/api_eventexposure.go`
- 解析流程：
  - 優先使用 `pduSeId`：`smf/internal/context/sm_context.go` 的 `GetSMContextById(supi, pduSeId)`
  - 否則使用 SUPI + 可選 DNN/S-NSSAI：`smf/internal/context/sm_context_lookup.go` 的 `GetSMContextBySupi`
- 取得 UE IP：`smContext.PDUAddress`
- 取得選擇的 UPF：`smContext.SelectedUPF`
- UPF apiRoot：從 `smfcfg.yaml` 的 `nupfEventExposure.defaultUpfApiRoot` 取得

若找不到 SMContext 或 UE IP，直接回 `ProblemDetails`，不進行 UPF 訂閱。

## UPF 訂閱 Payload 範例（與 contract 一致）

```json
{
  "subscription": {
    "nfId": "550e8400-e29b-41d4-a716-446655440000",
    "ueIpAddress": "10.10.0.1",
    "eventList": [
      {
        "type": "USER_DATA_USAGE_MEASURES",
        "measurementTypes": [
          "VOLUME_MEASUREMENT",
          "THROUGHPUT_MEASUREMENT"
        ],
        "granularityOfMeasurement": "PER_SESSION"
      }
    ],
    "eventNotifyUri": "http://nwdaf.example.com/nupf-ee/v1/notify",
    "notifyCorrelationId": "nwdaf-uecom-0001",
    "eventReportingMode": {
      "trigger": "PERIODIC",
      "repPeriod": 10
    }
  }
}
```

## 失敗行為

- 無 SMContext：回 `404` ProblemDetails（`SMContext not found`）。
- 無 UE IP：回 `404` ProblemDetails（`UE IP not allocated`）。
- UPF apiRoot 或 `nwdafUpfNotifyUri` 未設定：回 `500` ProblemDetails（SystemFailure）。
- UPF CreateSubscription 非 `201`：回 `500` ProblemDetails（SystemFailure），不建立 SMF 訂閱狀態。
