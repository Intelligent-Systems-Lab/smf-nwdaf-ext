根據您提供的 TS 29.508 與 TS 23.502 文件，以下是針對 NWDAF 向 SMF 訂閱並取得 UPF 使用者資料用量（User Data Usage）的最小可行流程與分析。

### 1. 最小可行流程 (Minimum Viable Workflow)

根據規範，NWDAF 需使用 `Nsmf_EventExposure_Subscribe` 服務操作向 SMF 進行訂閱。針對 `UPF_EVENT`（包含 User Data Usage Measures），SMF 通常會協助在 UPF 建立訂閱，但資料通知通常由 UPF 直接發送給 NWDAF（Direct Notification）。

#### A. Create Subscription (POST) Request/Response

NWDAF 向 SMF 發送訂閱請求。

*   **HTTP Request:**
    *   **Method:** `POST`
    *   **URI:** `{apiRoot}/nsmf-event-exposure/v1/subscriptions`
    *   **Headers:** `Content-Type: application/json`

```http
POST /nsmf-event-exposure/v1/subscriptions HTTP/2
Host: smf.5gc.net
Content-Type: application/json

{
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
}
```
*註：`upfEvents` 內部的結構定義於 TS 29.564，在此依據 TS 29.508 上下文 與 TS 23.502 填入示意值。`nfId` 為訂閱 `UPF_EVENT` 時的條件必填欄位。*

*   **HTTP Response:**
    *   **Status Code:** `201 Created`
    *   **Headers:** `Location: {apiRoot}/nsmf-event-exposure/v1/subscriptions/{subId}`

```http
HTTP/2 201 Created
Location: https://smf.5gc.net/nsmf-event-exposure/v1/subscriptions/sub-123
Content-Type: application/json

{
  "subId": "sub-123",
  "supi": "supi-1234567890",
  "notifId": "nwdaf-correlation-id-001",
  "notifUri": "https://nwdaf.example.com/notify/upf-data",
  "eventSubs": [
    {
      "event": "UPF_EVENT"
    }
  ]
}
```

#### B. Notify (Notification)

根據 TS 29.508 第 4.2.3.2 節 Note 3，顯式訂閱 `UPF_EVENT` 意味著 **"直接由 UPF 進行通知 (direct notification from the UPF)"**。因此，SMF 不會發送包含 User Data Usage 的 `Nsmf_EventExposure_Notify` Payload。資料流是 `UPF -> NWDAF`。

若因特殊實作或錯誤情境 SMF 需要發送通知（例如通知訂閱被移除），其最小 Payload 如下：

*   **HTTP Request (SMF -> NWDAF):**
    *   **Method:** `POST`
    *   **URI:** `{notifUri}` (即 `https://nwdaf.example.com/notify/upf-data`)

```http
POST /notify/upf-data HTTP/2
Host: nwdaf.example.com
Content-Type: application/json

{
  "notifId": "nwdaf-correlation-id-001",
  "eventNotifs": [
    {
      "event": "UPF_EVENT",
      "timeStamp": "2025-09-24T10:00:00Z"
    }
  ]
}
```

*   **HTTP Response (NWDAF -> SMF - Ack):**
    *   **Status Code:** `204 No Content`

```http
HTTP/2 204 No Content
```

#### C. Delete Subscription (DELETE) Request/Response

NWDAF 取消訂閱。

*   **HTTP Request:**
    *   **Method:** `DELETE`
    *   **URI:** `{apiRoot}/nsmf-event-exposure/v1/subscriptions/{subId}`

```http
DELETE /nsmf-event-exposure/v1/subscriptions/sub-123 HTTP/2
Host: smf.5gc.net
```

*   **HTTP Response:**
    *   **Status Code:** `204 No Content`

```http
HTTP/2 204 No Content
```

---

### 2. 欄位需求與 Schema 路徑 (TS 29.508 YAML)

以下列出建立訂閱 (`NsmfEventExposure`) 時的關鍵欄位。路徑基於 `components/schemas/NsmfEventExposure`。

*   **必填欄位 (Mandatory / Conditional Mandatory):**
    *   `notifId`: **(必填)** 用於關聯通知 ID。
        *   Schema: `properties/notifId`
    *   `notifUri`: **(必填)** 通知回傳的 URI。
        *   Schema: `properties/notifUri`
    *   `eventSubs`: **(必填)** 訂閱的事件列表。
        *   Schema: `properties/eventSubs` (Array of `EventSubscription`)
    *   `eventSubs[].event`: **(必填)** 設為 `"UPF_EVENT"`。
    *   `nfId`: **(條件必填)** 當 `event` 為 `UPF_EVENT` 時，此欄位為必填，代表建立訂閱的 NF 實例 ID。
        *   Schema: `properties/nfId`
    *   **UE 識別 (三擇一)**:
        *   `supi`: 用於 Single UE。
        *   `gpsi`: 用於 Single UE。
        *   `groupId` / `anyUeInd`: 用於群組或任意 UE。

*   **可 Stub / Hardcode 的欄位:**
    *   `notifId`: 可產生任意唯一字串 (e.g., `"uuid-notif-001"`).
    *   `notifUri`: 可固定為測試接收端點 (e.g., `"http://localhost:8080/callback"`).
    *   `nfId`: 可固定為 NWDAF 的 UUID (e.g., `"35368538-4089-4b2a-9a99-923304240000"`).
    *   `eventSubs[].upfEvents`: 此內容透傳給 UPF，若僅測試 SMF 介面流程，內容可先 Stub 為空物件或依據 29.564 的最小結構。

---

### 3. Single UE 識別與 Event Filter 位置

*   **Single UE 最小識別:**
    *   **欄位:** `supi` (Subscription Permanent Identifier)。
    *   **YAML 路徑:** `NsmfEventExposure.properties.supi`。
    *   **說明:** 若訂閱針對特定 UE，必須包含 `supi` 或 `gpsi`。對於 UPF Data Usage，通常使用 SUPI。

*   **可選 Event Filter (DNN / S-NSSAI) 位置:**
    *   這些欄位位於 `NsmfEventExposure` 的根層級，作為全域過濾器使用。
    *   **DNN:**
        *   **欄位:** `dnn`。
        *   **YAML 路徑:** `NsmfEventExposure.properties.dnn`。
    *   **S-NSSAI:**
        *   **欄位:** `snssai`。
        *   **YAML 路徑:** `NsmfEventExposure.properties.snssai`。

**注意事項:**
雖然 `EventSubscription` (在 `eventSubs` 陣列內) 也有 `dnn` 和 `snssai` 欄位 用於特定事件的過濾（例如 `ENERGY_USAGE_DATA`），但針對 `UPF_EVENT` 與 `UPF User Data Usage`，TS 23.502 的過濾器範例顯示這些參數通常作為匹配條件。在 OpenAPI 定義中，通常建議將適用於整個訂閱範圍的 Filter 放在根層級 (`NsmfEventExposure`)。