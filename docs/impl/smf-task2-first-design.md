# Nsmf_EventExposure 相關落點分析

## 1) 路由/Handler 註冊位置

### Server 初始化與 Router Group 註冊
**`internal/sbi/server.go`**
- 在 `newRouter()` 函式中，當 `ServiceNameList` 包含 `models.ServiceName_NSMF_EVENT_EXPOSURE` 時，會註冊 Event Exposure 路由
- 建立 router group: `router.Group(factory.SmfEventExposureResUriPrefix)` (即 `/nsmf-event-exposure/v1`)
- 套用 OAuth 授權檢查 middleware 

### URI Prefix 定義
**`pkg/factory/config.go`**
- 常數 `SmfEventExposureResUriPrefix = "/nsmf_event-exposure/v1"` 

### 路由定義與 Handler
**`internal/sbi/api_eventexposure.go`**
- `getEventExposureRoutes()`: 定義 4 個 RESTful endpoints
  - `POST /subscriptions` → `HTTPCreateIndividualSubcription`
  - `DELETE /subscriptions/:subId` → `HTTPDeleteIndividualSubcription`
  - `GET /subscriptions/:subId` → `HTTPGetIndividualSubcription`
  - `PUT /subscriptions/:subId` → `HTTPReplaceIndividualSubcription`
- **注意**：所有 handler 目前皆回傳 `http.StatusNotImplemented` (501) 

## 2) Subscription Resource 資料結構與儲存方式

### 資料結構定義
**`internal/context/sm_context.go`**

**EventExposureNotification struct**:
- 包裝 OpenAPI model `models.NsmfEventExposureNotification`
- 額外新增 `Uri` 欄位用於記錄通知目標

### 儲存方式 (In-Memory)
**`internal/context/sm_context.go`**

**SMContext struct 中的儲存欄位**:
- `UpPathChgEarlyNotification map[string]*EventExposureNotification` - 儲存 EARLY 類型通知
- `UpPathChgLateNotification map[string]*EventExposureNotification` - 儲存 LATE 類型通知
- Key 格式: `Uri + NotifId`
- **儲存方式**: In-memory，儲存在各個 SMContext 物件中，沒有使用 DB 

**初始化**:
- 在 `NewSMContext()` 中初始化這兩個 map 

## 3) Create/Delete Subscription 既有實作模式

### 參考實作模式 (PDU Session 為例)
**`internal/sbi/processor/pdu_session.go`**

**201 Created + Location Header 模式**:
- 建構 Location header: 包含 protocol、host、path 和 resource ID
- 使用 `c.Header("Location", location)` 設定 header
- 回傳 `http.StatusCreated` (201) 與 response body 

**204 No Content 模式**:
**`internal/sbi/processor/notifier.go`**
- 在 charging notification 中使用
- 成功處理後直接回傳 `c.Status(http.StatusNoContent)` 

### 目前 Event Exposure Subscription 實作狀態
**`internal/sbi/api_eventexposure.go`**
- **Create/Delete/Get/Replace 皆未實作**，全部回傳 `StatusNotImplemented` 

## 4) 已支援的 Event Exposure 事件

### 已實作事件: UP_PATH_CH (User Plane Path Change)

**事件建構與儲存**:
**`internal/context/sm_context.go`**

**`BuildUpPathChgEventExposureNotification()` 方法**:
- 接收 `chgEvent *models.UpPathChgEvent`, `srcRoute`, `tgtRoute`
- 建立 `SmfEventExposureEventNotification` with `Event: models.SmfEvent_UP_PATH_CH`
- 根據 `DnaiChgType` (EARLY/LATE) 分別儲存到對應的 notification map 

**事件發送**:
**`SendUpPathChgNotification()` 方法**:
- 根據 chgType (EARLY/LATE) 選擇對應的 notification map
- 使用 callback 發送通知後從 map 中刪除 

### 觸發機制
**`internal/context/sm_context_policy.go`**

**`checkUpPathChangeEvt()` 函式**:
- 在 `ApplyPccRules()` 中被呼叫
- 比較 source 和 target `TrafficControlData` 的 `RouteToLocs`
- 當路由改變時，呼叫 `BuildUpPathChgEventExposureNotification()` 建立通知 

**TrafficControlData 結構**:
**`internal/context/traffic_control_data.go`**
- 包裝 `models.TrafficControlData` (來自 PCF policy decision)
- 包含 `RouteToLocs` 和 `UpPathChgEvent` 欄位 

### 事件通知實際發送
**`internal/sbi/processor/notifier.go`**

**`SendUpPathChgEventExposureNotification()` 函式**:
- 使用 `EventExposure.NewAPIClient()` 建立 client
- 透過 `CreateIndividualSubcriptionMyNotificationPost()` 發送通知到 NEF/AF
- 處理各種錯誤情況 

**呼叫時機**:
- PDU Session 建立過程中: `HandlePDUSessionSMContextCreate`
- SM Policy 更新時: `HandleSMPolicyUpdateNotify`
- 先發送 EARLY notification，執行 UPF session activation，再發送 LATE notification 

---

## Notes

### 如何新增新事件類型或擴充 subscription body

**目前架構分析**:
1. **Subscription 管理尚未實作** - 目前沒有實際的 subscription 儲存機制，通知 URI 來自 PCF policy decision 中的 `UpPathChgEvent.NotificationUri`

2. **若要實作 subscription 功能，建議步驟**:
   - 定義 subscription 儲存結構 (可參考 `EventExposureNotification` 的設計)
   - 在 SMF context 或 global context 中維護 subscription pool
   - 實作 `api_eventexposure.go` 中的 4 個 handler 函式
   - Create 時產生 subscription ID，儲存 subscription，回傳 201 + Location
   - Delete 時移除 subscription，回傳 204

3. **擴充新事件類型**:
   - 參考 `UP_PATH_CH` 的實作模式
   - 在 SMContext 中新增對應的 notification map
   - 實作類似 `BuildUpPathChgEventExposureNotification()` 的方法
   - 在適當的業務邏輯觸發點呼叫建構與發送方法
   - 使用 `SendUpPathChgEventExposureNotification()` 或建立新的發送函式

4. **當前限制**: Event Exposure subscription API 雖已定義路由，但完全未實作 (501)。通知功能僅針對 UP path change 且 notification URI 由 PCF 提供，非由 SMF 自行管理 subscription。