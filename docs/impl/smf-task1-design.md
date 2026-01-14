# SMF HTTP Server 初始化與 Router 註冊分析

## 1. HTTP Server 初始化與 Router 註冊位置

### 主要初始化流程

HTTP server 的初始化發生在 `internal/sbi/server.go` 中的 `NewServer` 函數。

在這個函數中，透過 `newRouter` 函數建立 Gin router 並註冊所有路由。

### Router 註冊機制

`newRouter` 函數負責註冊所有的 route groups，包含：

**註冊的 Router Groups：**

1. **Callback Router** (`/nsmf-callback`) - 用於接收來自 PCF/CHF 的通知
2. **UPI Router** (`/upi/v1`) - User Plane Integration APIs
3. **PDU Session Router** (`/nsmf-pdusession/v1`) - 標準 SBI 服務
4. **Event Exposure Router** (`/nsmf_event-exposure/v1`) - 事件訂閱服務
5. **OAM Router** (`/nsmf-oam/v1`) - 管理與維運 APIs

所有 URI prefix 定義在 `pkg/factory/config.go` 中。 

### Route 應用機制

Routes 透過 `applyRoutes` 輔助函數註冊到各個 router group 上。

每個 route 都使用 `Route` 結構定義。

## 2. 現有的 Handler/Route 命名與分層慣例

### 檔案與模組分層結構

**A. API 定義層** (`internal/sbi/api_*.go`)
- `api_callback.go` - Callback 相關的 routes
- `api_pdusession.go` - PDU Session 管理 routes
- `api_eventexposure.go` - Event Exposure 訂閱 routes
- `api_oam.go` - OAM 管理 routes
- `api_upi.go` - User Plane Integration routes

每個 API 檔案都有一個 `get*Routes()` 方法返回該服務的所有 routes。

**B. Handler 命名慣例**

Handler 函數統一使用 `HTTP` 前綴命名：
- `HTTPSmPolicyUpdateNotification`
- `HTTPChargingNotification`
- `HTTPPostSmContexts`
- `HTTPGetUEPDUSessionInfo` 

**C. Processor 層** (`internal/sbi/processor/`)

Processor 負責業務邏輯處理，Handler 會呼叫 Processor 的方法：
- `HandleSMPolicyUpdateNotify` 
- `HandleChargingNotification` 

Processor 結構體封裝了 SMF app 和 Consumer。 

**D. Consumer 層** (`internal/sbi/consumer/`)

Consumer 層負責呼叫其他 NF 的 API（作為 client 角色），每個 NF 有獨立的 service 檔案：
- `amf_service.go` - AMF client
- `chf_service.go` - CHF client
- `pcf_service.go` - PCF client
- `nrf_service.go` - NRF client
- `udm_service.go` - UDM client
- `smf_service.go` - SMF-to-SMF client

Consumer 結構體聚合所有 NF service clients。 

## 3. NWDAF 相關功能的最佳實作位置

### A. NWDAF Notify Callback Endpoint

**建議檔案：** `internal/sbi/api_callback.go`

**理由：**
1. 現有的 callback endpoints 都在此檔案中定義（PCF 和 CHF 的通知）
2. Callback routes 已經在 `newRouter` 中註冊為獨立的 group，使用 `/nsmf-callback` prefix
3. 遵循現有的 callback 處理模式

**實作方式：**
- 在 `getCallbackRoutes()` 中新增 NWDAF notification route
- 新增對應的 `HTTPNwdafNotification` handler
- Handler 呼叫 processor 層的業務邏輯處理函數

**Processor 處理邏輯建議檔案：** `internal/sbi/processor/notifier.go`

這個檔案專門處理各種 notification 的業務邏輯。 

### B. 手動觸發訂閱/取消訂閱入口

根據用途不同，有兩個選擇：

#### 選項 1：作為管理/維運功能（建議）

**建議檔案：** `internal/sbi/api_oam.go`

**理由：**
1. OAM router 專門用於管理與維運功能  
2. 已有類似的管理功能（查詢 PDU Session info、User Plane info）
3. 使用 `/nsmf-oam/v1` prefix，與標準 SBI 服務區隔
4. 通常需要額外的認證授權檢查 

**實作方式：**
```
POST /nsmf-oam/v1/nwdaf-subscriptions
DELETE /nsmf-oam/v1/nwdaf-subscriptions/:subscriptionId
GET /nsmf-oam/v1/nwdaf-subscriptions/:subscriptionId
```

#### 選項 2：作為標準 SBI 服務

**建議檔案：** 新建 `internal/sbi/api_nwdaf.go`

**理由：**
- 如果是供其他 NF 呼叫的標準化介面
- 需要在 `ServiceNameList` 配置中啟用（類似 Event Exposure 服務）
- 需要 OAuth 授權檢查

### C. NWDAF Consumer Client（發起訂閱的 Client 端）

**建議檔案：** 新建 `internal/sbi/consumer/nwdaf_service.go`

**理由：**
1. 遵循現有的 consumer pattern 
2. 與其他 NF service 的實作方式一致
3. 管理 NWDAF API clients 的生命週期和快取

**需要修改：** `internal/sbi/consumer/consumer.go`
- 在 Consumer 結構體中新增 `*nwdafService` 欄位
- 在 `NewConsumer` 中初始化 NWDAF service

## Notes

**架構模式總結：**

```
┌─────────────────────────────────────────┐
│  HTTP Request                            │
└──────────────┬──────────────────────────┘
               │
               ▼
┌─────────────────────────────────────────┐
│  Handler (api_*.go)                     │
│  - HTTP{MethodName}                     │
│  - 解析請求、驗證參數                      │
└──────────────┬──────────────────────────┘
               │
               ▼
┌─────────────────────────────────────────┐
│  Processor (processor/*.go)             │
│  - Handle{Operation}                    │
│  - 業務邏輯處理                           │
└──────────────┬──────────────────────────┘
               │
               ▼
┌─────────────────────────────────────────┐
│  Consumer (consumer/*_service.go)       │
│  - 呼叫其他 NF 的 APIs                   │
└─────────────────────────────────────────┘
```

**重要提醒：**
1. 所有受保護的 SBI 服務都需要經過 OAuth 授權檢查（通過 `util_oauth.NewRouterAuthorizationCheck`）
2. Callback endpoints 通常不需要授權，因為它們是由信任的 NF 呼叫
3. 新的服務需要在 `ServiceNameList` 配置中啟用才會被註冊
4. URI prefix 應在 `pkg/factory/config.go` 中定義常數
