### 1. 最小可行流程範例

此流程假設 NWDAF 作為服務消費者（NF Service Consumer），向作為生產者的 SMF 訂閱 `UPF_EVENT` 事件，該事件包含使用者數據用量測量（USER_DATA_USAGE_MEASURES）。

#### **A. 建立訂閱 (Create Subscription)**
*   **請求 (NWDAF → SMF):** `POST /nsmf-event-exposure/v1/subscriptions`
```json
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
*   **回應 (SMF → NWDAF):** `201 Created`
    *   **Header:** `Location: {apiRoot}/nsmf-event-exposure/v1/subscriptions/sub123`
    *   **Payload:** 回傳完整的 `NsmfEventExposure` 物件（含 SMF 分配的 `subId`）。

#### **B. 事件通知 (Notification)**
*   **請求 (SMF → NWDAF):** `POST {notifUri}`
```json
{
  "notifId": "NWDAF-SUB-001",
  "eventNotifs": [
    {
      "event": "UPF_EVENT",
      "timeStamp": "2026-01-16T00:00:00Z",
      "supi": "imsi-208930000000001"
    }
  ]
}

```
*   **回應 (NWDAF → SMF):** `204 No Content`（確認收到通知）。

#### **C. 刪除訂閱 (Delete Subscription)**
*   **請求 (NWDAF → SMF):** `DELETE /nsmf-event-exposure/v1/subscriptions/sub123`
*   **回應 (SMF → NWDAF):** `204 No Content`。

---

### 2. 必填欄位、Schema 路徑與 Stub 建議

根據 OpenAPI 組件定義（`#/components/schemas/`），訂閱請求的必填結構如下：

| 必填欄位 | YAML Schema 節點路徑 | 說明與 Stub 建議 |
| :--- | :--- | :--- |
| **notifId** | `NsmfEventExposure/properties/notifId` | NWDAF 分配的關聯 ID。**可 Stub/Hardcode**（例如 "ID-001"）。 |
| **notifUri** | `NsmfEventExposure/properties/notifUri` | 接收通知的端點。**可 Hardcode** 指向您的 Mock Server。 |
| **eventSubs** | `NsmfEventExposure/properties/eventSubs` | 訂閱事件陣列。**至少需含一個物件**。 |
| **event** | `EventSubscription/properties/event` | 必須設為 `UPF_EVENT` 以獲取 UPF 相關數據。 |

**其他欄位處理：**
*   **subId**: SMF 在 201 回應中生成的 ID，不應由 NWDAF 在 POST 請求中提供。
*   **supportedFeatures**: 若不涉及複雜協商，可先不填（Optional）。

---

### 3. UE 識別與過濾器欄位 (Event Filter)

針對「Single UE」情境與可選的過濾條件，相關欄位位於 `NsmfEventExposure` 物件的根路徑下：

*   **Single UE 最小識別應用：**
    *   **supi**: 路徑為 `#/components/schemas/NsmfEventExposure/properties/supi`。這是單一 UE 訂閱的最主要識別碼。
    *   **gpsi**: 路徑為 `#/components/schemas/NsmfEventExposure/properties/gpsi`。若無法取得 SUPI 時的可選方案。
    *   **pduSeId**: 路徑為 `#/components/schemas/NsmfEventExposure/properties/pduSeId`。若訂閱僅針對該 UE 的特定 PDU Session。

*   **可選過濾器 (Event Filters)：**
    *   **DNN 過濾**: 路徑為 `#/components/schemas/NsmfEventExposure/properties/dnn`。用於限定特定數據網路的用量報告。
    *   **S-NSSAI 過濾**: 路徑為 `#/components/schemas/NsmfEventExposure/properties/snssai`。用於限定特定網路切片的用量報告。

**備註：** 根據來源，`UPF_EVENT` 在本規範中專門用於獲取由 UPF 產生的 `USER_DATA_USAGE_MEASURES`（數據用量測量）與 `USER_DATA_USAGE_TRENDS`（數據用量趨勢）。