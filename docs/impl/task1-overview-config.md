# Task1 概觀：設定項目

本文件整理 Task1 相關設定（若未設定則不影響原有功能）。  
來源引用：`docs/impl/smf-task1-third-patch.md`、`docs/impl/smf-task1-first-patch.md`。

## 設定區塊

Task1 使用 `configuration.nwdafSubscription` 進行預設值設定。  
參考：`docs/impl/smf-task1-third-patch.md`「Config 參數化（YAML schema 範例）」。

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

## 欄位說明與覆蓋規則

- `defaultNwdafApiRoot`  
  - NWDAF apiRoot 預設值  
  - OAM request 有帶 `nwdafApiRoot` 時，以 OAM 為準  
  - 參考：`docs/impl/smf-task1-third-patch.md`

- `defaultNotificationURI`  
  - 通知 callback 預設值（含 `/nwdaf-callback` path）  
  - OAM request 有帶 `notificationURI` 時，以 OAM 為準  
  - 參考：`docs/impl/smf-task1-third-patch.md`

- `defaultNotifCorrId`  
  - notification correlation id 預設值  
  - OAM request 有帶 `notifCorrId` 時，以 OAM 為準  
  - 參考：`docs/impl/smf-task1-third-patch.md`

- `defaultRepPeriod`  
  - `evtReq.repPeriod` 預設值（PERIODIC）  
  - OAM request 有帶 `evtReq` 或 `repPeriod` 時，以 OAM 為準  
  - 參考：`docs/impl/smf-task1-third-patch.md`

- `retryTimes` / `retryInterval`  
  - Create/Delete 最小重試次數與間隔  
  - OAM request 有帶 retry 參數時，以 OAM 為準  
  - 參考：`docs/impl/smf-task1-third-patch.md`

## 與 contract 的關係

設定僅影響「預設值」，不修改 contract 黃金 payload 的欄位與結構。  
建立/通知 payload 必須遵循：`docs/contract/nnwdaf-uecomm-subscription.md`。

