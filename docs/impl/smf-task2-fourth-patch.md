# SMF Task2 Patch4 實作說明（介面骨架擴充）

## 介面列舉與 Stub 概覽

**Nsmf_EventExposure（TS 29.508）**
- 已實作：`POST /subscriptions`、`DELETE /subscriptions/{subId}`
- Stub（回 501）：`GET /subscriptions/{subId}`、`PUT /subscriptions/{subId}`
- 以介面註解 `NsmfEventExposureHandler` 列舉完整 handler 面向

**Nupf_EventExposure（TS 29.564）**
- 已實作：`POST /ee-subscriptions`、`DELETE /ee-subscriptions/{subscriptionId}`
- Stub（未實作）：`PATCH /ee-subscriptions/{subscriptionId}`
- 以介面註解 `NupfEventExposureOperations` 列舉完整操作面向

## Capabilities 說明區塊

在 `smf/internal/sbi/api_eventexposure.go` 與 `smf/internal/sbi/consumer/upf_eventexposure_service.go` 皆加入 V0 功能範圍註解，明確標示已實作與暫緩項目。

## 後續工作（不影響 V0 行為）

- 實作 `GET /subscriptions/{subId}` 與 `PUT /subscriptions/{subId}` 的查詢與替換邏輯
- 實作 `PATCH /ee-subscriptions/{subscriptionId}` 以支援 UPF 訂閱更新
