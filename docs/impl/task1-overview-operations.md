# Task1 概觀：操作與驗證

本文件提供最小操作流程與排查方式，請以 contract 黃金 payload 為準。  
來源引用：`docs/contract/nnwdaf-uecomm-subscription.md`、`docs/spec/TS29520_Nnwdaf_EventsSubscription.yaml`、`docs/impl/smf-task1-first-patch.md`、`docs/impl/smf-task1-second-patch.md`、`docs/impl/smf-task1-third-patch.md`。

## 新手快速上手（從零開始）

如果你是第一次 clone 此 SMF repo，建議按以下順序確認環境與位置：

1) 進入 SMF 目錄

```bash
cd smf
```

2) 確認主要設定檔

- 預設配置：`smfcfg.yaml`
- NWDAF 相關預設值：`configuration.nwdafSubscription`
  - 參考：`docs/impl/task1-overview-config.md`

3) 準備 NWDAF 端點與 callback

- 需要一個可連線的 NWDAF API Root（例如 `http://127.0.0.1:9000`）
- SMF callback URI 需可被 NWDAF 連回（例如 `http://<SMF_HOST>:<PORT>/nwdaf-callback`）
  - 參考：`docs/contract/nnwdaf-uecomm-subscription.md`

4) 啟動 SMF（示例）

```bash
go run ./cmd/main.go -c smfcfg.yaml
```

5) 啟動後，使用終端機手動訂閱 / 取消訂閱

- 訂閱（OAM → SMF）

```bash
curl -X POST http://127.0.0.2:8000/nsmf-oam/v1/nwdaf-subscriptions \
  -H 'Content-Type: application/json' \
  -d '{
    "supi": "imsi-208930000000001",
    "nwdafApiRoot": "http://127.0.0.1:9000",
    "notificationURI": "http://127.0.0.2:8000/nwdaf-callback",
    "evtReq": {
      "notifMethod": "PERIODIC",
      "repPeriod": 60
    }
  }'
```

- 取消訂閱（OAM → SMF）

```bash
curl -X DELETE http://127.0.0.2:8000/nsmf-oam/v1/nwdaf-subscriptions/<subscriptionId>
```

完成以上步驟後，可進入下方「最小流程」直接驗證 Create/Notify/Delete。

## 最小流程（依 contract payload）

### 1) OAM 觸發建立訂閱（SMF → NWDAF）

Endpoint（OAM）：  
`POST /nsmf-oam/v1/nwdaf-subscriptions`

Request body（請保持欄位與大小寫一致；`notifCorrId` 選填）：  
參考：`docs/contract/nnwdaf-uecomm-subscription.md`「建立訂閱 / Request Body」。

```json
{
  "supi": "imsi-208930000000001",
  "nwdafApiRoot": "http://127.0.0.1:9000",
  "notificationURI": "http://<SMF_HOST>:<PORT>/nwdaf-callback",
  "notifCorrId": "my-correlation-001",
  "evtReq": {
    "notifMethod": "PERIODIC",
    "repPeriod": 60
  }
}
```

預期行為：  
- SMF 對 NWDAF 發起 Create，成功回 `201` 並在 response header 帶 `Location`
  - 參考：`docs/spec/TS29520_Nnwdaf_EventsSubscription.yaml` `/subscriptions` → `201`

### 2) NWDAF 回呼通知（Notify）

Endpoint（SMF callback）：  
`POST {notificationURI}`（由 Create 中的 `notificationURI` 決定）

Request body（必須是 array）：  
參考：`docs/contract/nnwdaf-uecomm-subscription.md`「事件通知 / Request Body」。

```json
[
  {
    "subscriptionId": "sub-0001",
    "notifCorrId": "my-correlation-001",
    "eventNotifications": [
      {
        "event": "UE_COMMUNICATION",
        "timeStampGen": "2026-01-14T09:00:00Z",
        "ueComms": [
          {
            "ts": "2026-01-14T09:00:00Z",
            "commDur": 3600,
            "trafChar": {
              "ulVol": 1048576,
              "dlVol": 5242880
            }
          }
        ]
      }
    ]
  }
]
```

預期行為：  
- SMF 回 `204 No Content`
  - 參考：`docs/spec/TS29520_Nnwdaf_EventsSubscription.yaml` callbacks `myNotification` → `204`
- `notifCorrId` 若缺失，SMF 仍可依 `subscriptionId` 完成關聯

### 3) OAM 觸發刪除訂閱（SMF → NWDAF）

Endpoint（OAM）：  
`DELETE /nsmf-oam/v1/nwdaf-subscriptions/{subscriptionId}`

預期行為：  
- SMF 呼叫 NWDAF Delete 並回 `204 No Content`
  - 參考：`docs/spec/TS29520_Nnwdaf_EventsSubscription.yaml` `/subscriptions/{subscriptionId}` → delete `204`

## 應看到的 log（重點欄位）

依 Task1 patch notes，log 會包含以下欄位：  
- `supi`
- `subscription_id`
- `notif_corr_id`
- `http_status`
參考：`docs/impl/smf-task1-first-patch.md`「記錄行為（Logging）」

## 常見錯誤與排查

1) Create 回 `400` 或 `502`
   - 檢查 `notificationURI` 與 `nwdafApiRoot` 是否正確
   - 確認 NWDAF 端是否可達
   - 參考：`docs/impl/smf-task1-second-patch.md`「TS 29.520 行為對齊清單」

2) Callback 回 `400`
   - Body 不是 array 或為空陣列
   - 參考：`docs/impl/smf-task1-second-patch.md`「Notification callback」

3) Delete 回 `404`
   - SMF 本地 state 查不到 `subscriptionId`
   - 參考：`docs/impl/smf-task1-second-patch.md`「DeleteNWDAFEventsSubscription」
