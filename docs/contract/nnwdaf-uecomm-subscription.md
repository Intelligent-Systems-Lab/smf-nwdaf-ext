根據 3GPP TS 29.520 與 TS 23.288 原始文件，關於「SMF 作為 consumer 訂閱 **UE Communication** 分析」的最小可行 Request/Response 範例及必填欄位說明如下：

### 一、 建立訂閱 (Create Subscription)

**1. 請求範例 (Request)**
*   **HTTP Method:** POST
*   **Resource URI:** `{apiRoot}/nnwdaf-eventssubscription/v1/subscriptions`
*   **Request Body (JSON):**
```json
{
  "eventSubscriptions": [
    {
      "event": "UE_COMMUNICATION",
      "tgtUe": {
        "supis": ["imsi-208930000000001"]
      }
    }
  ],
  "notificationURI": "http://<SMF_HOST>:<PORT>/nwdaf-callback",
  "notifCorrId": "my-correlation-001",
  "evtReq": {
    "notifMethod": "PERIODIC",
    "repPeriod": 60
  }
}
```

**2. 回應範例 (Response)**
*   **Status Code:** `201 Created`
*   **Location Header:** `{apiRoot}/nnwdaf-eventssubscription/v1/subscriptions/sub123`
*   **Response Body:** 同 Request Body，但包含由 NWDAF 分配的狀態。

**3. 必填欄位與 YAML Schema 路徑**
*   **notificationURI**: 接收通知的 URI。
    *   路徑：`#/components/schemas/NnwdafEventsSubscription/properties/notificationURI`。
*   **eventSubscriptions**: 訂閱事件列表。
    *   路徑：`#/components/schemas/NnwdafEventsSubscription/properties/eventSubscriptions`。
*   **event**: 事件識別碼，必須為 `UE_COMMUNICATION`。
    *   路徑：`#/components/schemas/EventSubscription/properties/event`。
*   **tgtUe**: 目標 UE 資訊（SMF 訂閱時須提供 SUPI 或 Internal-Group-Id）。
    *   路徑：`#/components/schemas/EventSubscription/properties/tgtUe`。
*   **notifCorrId**: 選填；SMF 會接受缺失並以 `subscriptionId` 作為後續關聯依據。

---

### 二、 刪除訂閱 (Delete Subscription)

**1. 請求範例 (Request)**
*   **HTTP Method:** DELETE
*   **Resource URI:** `{apiRoot}/nnwdaf-eventssubscription/v1/subscriptions/{subscriptionId}`
*   **Request Body:** 無 (n/a)。

**2. 回應範例 (Response)**
*   **Status Code:** `204 No Content`。

---

### 三、 事件通知 (Notify)

**1. 請求範例 (Request - 由 NWDAF 發起)**
*   **HTTP Method:** POST
*   **Resource URI:** `{notificationURI}` (建立訂閱時由 SMF 提供)
*   **Request Body (JSON):**
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

**2. 回應範例 (Response - 由 SMF 回覆)**
*   **Status Code:** `204 No Content`。

**3. 必填欄位與 YAML Schema 路徑**
*   **subscriptionId**: 訂閱識別碼。
    *   路徑：`#/components/schemas/NnwdafEventsSubscriptionNotification/properties/subscriptionId`。
*   **eventNotifications**: 已發生的事件清單。
    *   路徑：`#/components/schemas/NnwdafEventsSubscriptionNotification/properties/eventNotifications`。
*   **event**: 通知之事件類型。
    *   路徑：`#/components/schemas/EventNotification/properties/event`。
*   **ueComms**: 當事件為 `UE_COMMUNICATION` 時必須包含此欄位。
    *   路徑：`#/components/schemas/EventNotification/properties/ueComms`。

---

### 四、 Location Header 補充說明

*   **內容：** Location Header 必須包含**新建立之個別訂閱資源的完整 URI**。
*   **URI 格式：** `{apiRoot}/nnwdaf-eventssubscription/<apiVersion>/subscriptions/{subscriptionId}`。
    *   其中 `{apiRoot}` 依據 TS 29.501 定義。
    *   `<apiVersion>` 在當前版本為 `v1`。
    *   `{subscriptionId}` 為 NWDAF 分配給該訂閱的唯一識別字串。
