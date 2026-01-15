# SMF Task1 第二版 Patch 紀錄（自我 code review + 修補）

本文件記錄本次對 Task1 實作的自我 code review 與修補成果，重點放在註解補齊、與 TS 29.520/contract 黃金 payload 的一致性、以及行為健壯性。

## 修改摘要（依檔案）

- `smf/internal/sbi/api_nwdaf_callback.go`
  - 檔頭補充用途與 TS 29.520 對應點（callback 204）
  - 對空陣列/反序列化失敗回 400 並記錄 log
- `smf/internal/sbi/processor/nwdaf_subscription.go`
  - 檔頭補充 Create(201+Location)/Callback(204)/Delete(204) 對應說明
  - `notifCorrId` 缺失時不拒絕，改以 log 警告與 fallback 行為
  - 強化 Location 解析（URL path + regex fallback）
  - callback 查找支援 `subscriptionId` fallback（當 `notifCorrId` 缺失）
  - delete 找不到 state 時回 404 並 log
  - `// TODO(V1)` 集中註記（單一檔案）
- `smf/internal/sbi/consumer/nwdaf_service.go`
  - 檔頭補充 TS 29.520 Create/Delete 對應說明
- `smf/internal/context/nwdaf_subscription.go`
  - 檔頭與資料結構不變量註解（key/index/fallback）
- `smf/internal/sbi/api_oam.go`
  - 增加 NWDAF OAM endpoints 註解

## TS 29.520 行為對齊清單

- CreateNWDAFEventsSubscription
  - 成功回 `201 Created` 且解析 `Location` header
  - `subscriptionId` 從 `Location` path 取得並寫入 in-memory store
- Notification callback
  - 接收 array of `NnwdafEventsSubscriptionNotification`
  - 正常回 `204 No Content`
  - 若 body 空陣列或格式錯誤回 `400 Bad Request`
- DeleteNWDAFEventsSubscription
  - 成功回 `204 No Content`
  - 本地 state 不存在時回 `404`（OAM 內部語義）

## 與 contract 黃金 payload 的對齊說明

依 `docs/contract/nnwdaf-uecomm-subscription.md`：

- Create payload（SMF → NWDAF）保持欄位與大小寫一致：
  - `eventSubscriptions[].event = "UE_COMMUNICATION"`
  - `eventSubscriptions[].tgtUe.supis`
  - `notificationURI`
  - `notifCorrId`
  - `evtReq`
- Notify payload（NWDAF → SMF）接受 **array**：
  - `subscriptionId`
  - `notifCorrId`
  - `eventNotifications[].event = "UE_COMMUNICATION"`
  - `eventNotifications[].ueComms`

若 `notifCorrId` 缺失（contract 為必填但需容錯），處理方式：
- 不 panic
- 以 `subscriptionId` 做 lookup fallback
- log warning 以利觀測

## 最小驗證方式（curl）

1) 建立訂閱（OAM → SMF）
```bash
curl -k -X POST https://127.0.0.2:8000/nsmf-oam/v1/nwdaf-subscriptions \
  -H 'Content-Type: application/json' \
  -d '{
    "supi": "imsi-208930000000001",
    "nwdafApiRoot": "http://127.0.0.1:9000",
    "notificationURI": "https://127.0.0.2:8000/nwdaf-callback",
    "notifCorrId": "corr-001",
    "evtReq": { "notifMethod": "PERIODIC", "repPeriod": 60 }
  }'
```

2) 通知（NWDAF → SMF callback）
```bash
curl -k -X POST https://127.0.0.2:8000/nwdaf-callback \
  -H 'Content-Type: application/json' \
  -d '[{
    "subscriptionId": "sub-0001",
    "notifCorrId": "corr-001",
    "eventNotifications": [{
      "event": "UE_COMMUNICATION",
      "timeStampGen": "2026-01-14T09:00:00Z",
      "ueComms": [{
        "ts": "2026-01-14T09:00:00Z",
        "commDur": 3600,
        "trafChar": { "ulVol": 1, "dlVol": 1 }
      }]
    }]
  }]'
```

3) 刪除訂閱（OAM → SMF）
```bash
curl -k -X DELETE https://127.0.0.2:8000/nsmf-oam/v1/nwdaf-subscriptions/sub-0001
```
