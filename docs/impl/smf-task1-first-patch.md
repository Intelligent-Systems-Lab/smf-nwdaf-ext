# SMF Task1 第一版 Patch 行為整理（最小可跑通）

本文整理第一版 patch 的行為與改動範圍，對應「SMF 作為 Consumer → 訂閱/取消訂閱 NWDAF UE_COMMUNICATION」及 callback handler 的最小可跑通實作。

## 目標與範圍

- 在 SMF 內提供本機管理觸發點（OAM）來建立/刪除 NWDAF 訂閱
- SMF 作為 client 呼叫 NWDAF `CreateNWDAFEventsSubscription`，處理 `201 + Location` 並將 subscriptionId 存入記憶體
- 提供 `notificationURI` 對應的 callback handler（接收通知後回 204）
- 刪除訂閱時使用 Location 中的 subscriptionId 呼叫 NWDAF Delete，並清除記憶體狀態
- 全程記錄可追蹤 log（supi / subscriptionId / notifCorrId / http status）
- 不實作 OAuth2（依 TS 29.520 security `{}` 跑通）
- 不做多 UE / 多 UPF 或 filter 推導，必要時 hardcode/stub，集中在單一檔案標示 `// TODO(V1)`

## 新增/修改的 HTTP 入口

### OAM 管理入口（本機觸發點）

- `POST /nsmf-oam/v1/nwdaf-subscriptions`
  - 用來觸發 SMF 建立 NWDAF UE_COMMUNICATION 訂閱
- `DELETE /nsmf-oam/v1/nwdaf-subscriptions/:subscriptionId`
  - 用來觸發 SMF 刪除訂閱
- `GET /nsmf-oam/v1/nwdaf-subscriptions/:subscriptionId`
  - 用來查詢記憶體中的訂閱狀態（簡易確認用）

### NWDAF 通知 callback

- `POST /nwdaf-callback`
  - 作為 `notificationURI`，接收 NWDAF 通知，回 `204 No Content`

## 訂閱狀態（In-memory map）

為了能在 callback/刪除時快速定位訂閱，新增記憶體存放結構：

- 主 key：`(supi, subscriptionId, notifCorrId)`
- 輔助索引：
  - `subscriptionId -> key`
  - `subscriptionId + notifCorrId -> key`

狀態內容包含：
- `supi`
- `subscriptionId`
- `notifCorrId`
- `notificationURI`
- `nwdafApiRoot`

> 設計理由：Delete 只給 subscriptionId，因此必須能由 subscriptionId 反查；callback 內含 subscriptionId + notifCorrId，可由複合索引快速找到 supi。

## NWDAF 訂閱流程（Create / Delete）

### Create Subscription（SMF → NWDAF）

- 建立 payload：`event=UE_COMMUNICATION`，`tgtUe.supis=[supi]`，`notificationURI`，`notifCorrId`，`evtReq`
- 呼叫 NWDAF `POST /nnwdaf-eventssubscription/v1/subscriptions`
- 期望回應：
  - HTTP `201 Created`
  - `Location` header 含 `{apiRoot}/nnwdaf-eventssubscription/v1/subscriptions/{subscriptionId}`
- 從 `Location` 解析 `subscriptionId`，並寫入 in-memory
- 回覆 OAM 端 `201`，同時回傳 state 與 `Location`（便於調試）

### Delete Subscription（SMF → NWDAF）

- OAM 呼叫 `DELETE /nsmf-oam/v1/nwdaf-subscriptions/:subscriptionId`
- SMF 從 map 取出對應 state，取 `nwdafApiRoot`
- 呼叫 NWDAF `DELETE /nnwdaf-eventssubscription/v1/subscriptions/{subscriptionId}`
- 若成功，從 in-memory 移除該 subscription
- 回覆 OAM `204 No Content`

## Callback 處理流程（Notify）

- endpoint：`POST /nwdaf-callback`
- 解析 body：`[]NnwdafEventsSubscriptionNotification`
- 逐筆印出 log（含 subscriptionId / notifCorrId / supi / http status）
- 回覆 `204 No Content`

> 此版只做最小可跑通，未做 payload 驗證與 UE_COMMUNICATION 內容解析。

## 記錄行為（Logging）

所有 Create/Delete/Notify 都會記錄：
- `supi`
- `subscriptionId`
- `notifCorrId`
- `http_status`

方便在 SMF 端追蹤訂閱生命週期。

## 實作位置與檔案

- `smf/internal/sbi/api_oam.go`
  - 新增 OAM 入口 `POST/DELETE/GET /nwdaf-subscriptions`
- `smf/internal/sbi/api_nwdaf_callback.go`
  - 新增 callback handler `POST /nwdaf-callback`
- `smf/internal/sbi/processor/nwdaf_subscription.go`
  - Create/Delete/Notify 業務邏輯（含 `// TODO(V1)`）
- `smf/internal/sbi/consumer/nwdaf_service.go`
  - NWDAF client 呼叫 Create/Delete
- `smf/internal/sbi/consumer/consumer.go`
  - 注入 `nwdafService`
- `smf/internal/context/nwdaf_subscription.go`
  - 訂閱狀態 in-memory store
- `smf/internal/context/context.go`
  - 初始化 `NwdafSubs`
- `smf/internal/sbi/server.go`
  - 註冊 `/nwdaf-callback`
- `smf/pkg/factory/config.go`
  - 新增 `NwdafCallbackUriPrefix = "/nwdaf-callback"`

## 最小測試流程（示例）

1) 啟動 SMF
```bash
cd smf
go run ./cmd/main.go -c config/smfcfg.yaml
```

2) 觸發建立訂閱（OAM）
```bash
curl -k -X POST https://127.0.0.2:8000/nsmf-oam/v1/nwdaf-subscriptions \
  -H 'Content-Type: application/json' \
  -d '{
    "supi": "imsi-208930000000001",
    "nwdafApiRoot": "http://127.0.0.1:9000",
    "notificationURI": "https://127.0.0.2:8000/nwdaf-callback",
    "notifCorrId": "corr-001"
  }'
```

3) NWDAF 送通知（callback）
```bash
curl -k -X POST https://127.0.0.2:8000/nwdaf-callback \
  -H 'Content-Type: application/json' \
  -d '[{
    "subscriptionId": "sub-0001",
    "notifCorrId": "corr-001",
    "eventNotifications": [{
      "event": "UE_COMMUNICATION",
      "ueComms": [{}]
    }]
  }]'
```

4) 刪除訂閱（OAM）
```bash
curl -k -X DELETE https://127.0.0.2:8000/nsmf-oam/v1/nwdaf-subscriptions/sub-0001
```

## 待辦（V1 後）

- 依 spec 驗證 eventNotifications/ueComms 欄位
- 支援多 UE、多訂閱、filter 推導
- 將 hardcode/stub 換成真正的資料來源
- 加上 OAuth2（依 TS 29.520 security）
