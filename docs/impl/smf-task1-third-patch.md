# SMF Task1 第三版 Patch 紀錄（參數化 + 靜態檢查）

本次重點在於將 NWDAF 訂閱相關參數從 hardcode 改為 `smfcfg.yaml` 可配置，並補上最小重試/backoff 設定，同時確保靜態檢查與建置可通過。

## Config 參數化（YAML schema 範例）

新增可選區塊：`configuration.nwdafSubscription`（不設定不影響既有行為）。

```yaml
configuration:
  nwdafSubscription:
    defaultNwdafApiRoot: http://127.0.0.1:9000
    defaultNotificationURI: http://127.0.0.2:8000/nwdaf-callback
    defaultNotifCorrId: corr-001
    defaultRepPeriod: 60
    retryTimes: 2
    retryInterval: 1s
```

對應行為：
- OAM request 若提供欄位，優先採用；未提供時，使用上述預設值
- `notifCorrId` 若 request 與 config 都未提供，會自動產生 UUID（並 log warning）
- `repPeriod` 若 OAM 未提供且 config 有預設，會補到 `evtReq.notifMethod=PERIODIC` 與 `evtReq.repPeriod`
- `retryTimes` 與 `retryInterval` 用於 Create/Delete 的最小重試

## 本次修改影響範圍

- 新增 config schema：
  - `smf/pkg/factory/config.go` 增加 `NwdafSubscription` struct 與 `Configuration` 欄位
  - `smf/smfcfg.yaml` 增加 `nwdafSubscription` 範例
- 訂閱流程參數解析：
  - `smf/internal/sbi/processor/nwdaf_subscription.go` 加入預設值解析與 retry/backoff

## 回滾方式

- 若需回滾，移除或注解 `configuration.nwdafSubscription` 區塊即可回到舊行為
- 若需完全還原，移除：
  - `smf/pkg/factory/config.go` 中的 `NwdafSubscription` struct 與欄位
  - `smf/internal/sbi/processor/nwdaf_subscription.go` 內的預設解析與 retry

## Commenting pass

- 在 Task1-modified 與 Task1-new 檔案中新增簡明註解，以釐清 NWDAF UE_COMMUNICATION 流程（建立 → 通知 → 刪除）。
- 文件化設定預設值、回呼路由、訂閱狀態索引，以及重試/退避意圖。
- 已更新的檔案:
  - `smf/pkg/factory/config.go`
  - `smf/internal/sbi/server.go`
  - `smf/internal/sbi/api_oam.go`
  - `smf/internal/sbi/api_nwdaf_callback.go`
  - `smf/internal/sbi/consumer/nwdaf_service.go`
  - `smf/internal/sbi/consumer/consumer.go`
  - `smf/internal/context/nwdaf_subscription.go`
  - `smf/internal/context/context.go`
  - `smf/internal/sbi/processor/nwdaf_subscription.go`
