### 1) SMF 向 UPF 訂閱 User Data Usage events 範例

#### **Create Subscription (POST)**
*   **Resource URI:** `{apiRoot}/nupf-ee/v1/ee-subscriptions`
*   **Request Body (最小可行範例):**
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


*   **Response (201 Created):**
    *   **Status Code:** `201 Created`
    *   **Headers:** 包含 `Location` 標頭，指向新創建的訂閱資源 URI，例如：`{apiRoot}/nupf-ee/v1/ee-subscriptions/sub-001`。
    *   **Body:** 返回 `CreatedEventSubscription` 對象，確認訂閱詳情。

#### **Delete Subscription (DELETE)**
*   **Resource URI:** `{apiRoot}/nupf-ee/v1/ee-subscriptions/{subscriptionId}`
*   **行為:** 
    *   NF 服務消費者（SMF）發送 **DELETE** 請求來刪除現有的訂閱資源。
    *   **Response (204 No Content):** 若請求被接受並成功刪除，UPF 應回覆 `204 No Content` 狀態碼，且回應體為空。

---

### 2) 必填欄位與 Schema 路徑清單

根據 OpenAPI 定義，建立訂閱時 `CreateEventSubscription` 結構及其子結構中的必填欄位如下：

| 欄位名稱 (大小寫需精確) | Schema 內的 YAML 路徑 | 類型 / 列舉值示例 | 說明 |
| :--- | :--- | :--- | :--- |
| **subscription** | `#/components/schemas/CreateEventSubscription` | `UpfEventSubscription` | 訂閱主體內容 |
| **nfId** | `.../UpfEventSubscription` | `string` (UUID) | 創建訂閱的 NF 實例標識 |
| **eventList** | `.../UpfEventSubscription` | `array` | 請求訂閱的事件列表 |
| **type** | `.../UpfEventSubscription/eventList/items` | `USER_DATA_USAGE_MEASURES` | 事件類型 |
| **measurementTypes** | `.../UpfEventSubscription/eventList/items` | `VOLUME_MEASUREMENT`, `THROUGHPUT_MEASUREMENT` | 測量類型 |
| **eventNotifyUri** | `.../UpfEventSubscription` | `string` (Uri) | 接收通知的地址 |
| **notifyCorrelationId** | `.../UpfEventSubscription` | `string` | 通知關聯 ID |
| **eventReportingMode** | `.../UpfEventSubscription` | `UpfEventMode` | 報告觸發模式 |
| **trigger** | `.../UpfEventSubscription/eventReportingMode` | `PERIODIC` | 觸發方式 |

*註：`repPeriod` 在 `trigger` 為 `PERIODIC` 時為必填；`ueIpAddress` 對於 IP 類型 PDU Session 訂閱特定 UE 時為必填。*

---

### 3) 針對「使用 UE IP 當作 target」的正確欄位

根據來源，若訂閱目標是特定的 PDU Session，正確的 OpenAPI 欄位定義如下：

*   **正確欄位名稱：** **`ueIpAddress`**。
*   **型別：** **`IpAddr`**。
    *   根據引用自 TS 29.571 的定義，這是一個包含 IPv4 或 IPv6 地址的字串。
*   **正確路徑：** `#/components/schemas/UpfEventSubscription/properties/ueIpAddress`。
*   **條件性限制：**
    1.  當訂閱目標為**特定 UE 的 IP PDU Session** 時，必須包含此欄位。
    2.  此欄位與 `anyUe` (設為 true) 或 `supi` 是互斥的選擇，對於 IP PDU Session 應使用 `ueIpAddress`。
    3.  在事件通知（NotificationItem）中，對應回傳的欄位則會細分為 `ueIpv4Addr` 或 `ueIpv6Prefix`。