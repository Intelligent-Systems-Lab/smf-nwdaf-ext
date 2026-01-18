# SMF Task2 實作計畫（NWDAF->SMF Nsmf_EventExposure + 級聯 SMF->UPF Nupf_EventExposure）

以下僅為分析與計畫輸出，不修改程式碼實作。

## 1) 最小檔案觸碰清單（含新增）

**既有檔案（計畫修改）**
- `smf/internal/sbi/api_eventexposure.go`：實作 Nsmf_EventExposure 的 Create/Delete/Get/Replace handler（本任務聚焦 Create/Delete）。
- `smf/internal/sbi/server.go`：無需改動邏輯，僅確認路由入口（已有）。
- `smf/internal/sbi/consumer/consumer.go`：掛接新的 UPF Event Exposure consumer。
- `smf/internal/context/context.go`：掛接 Task2 訂閱狀態 store（全域）。
- `smf/internal/context/user_plane_information.go`：解析 UPF EventExposure apiRoot 設定到 UPF/UPNode 結構。
- `smf/pkg/factory/config.go`：擴充 UPF 節點設定，增加 Nupf EventExposure apiRoot 欄位（或等價的 UPF SBI 位址欄位）。
- `smf/smfcfg.yaml`：示範新增 UPF EventExposure apiRoot 設定。

**新增檔案**
- `smf/internal/sbi/processor/event_exposure_subscription.go`：實作 Create/Delete 主流程與錯誤對映。
- `smf/internal/sbi/consumer/upf_eventexposure_service.go`：封裝 SMF->UPF Nupf_EventExposure Create/Delete。
- `smf/internal/context/nsmf_eventexposure_subscription.go`：Nsmf 訂閱與 UPF 子訂閱狀態 store。
- `smf/docs/impl/smf-task2-plan.md`：本計畫文件（已建立）。

## 2) Nsmf_EventExposure 路由/Handler 入口

- Router group 註冊：`smf/internal/sbi/server.go` 中 `factory.SmfEventExposureResUriPrefix`（目前為 `/nsmf_event-exposure/v1` 的實作常數）。
- 路由定義：`smf/internal/sbi/api_eventexposure.go` 的 `getEventExposureRoutes()`。
- 入口 handler（現為 stub）：
  - `HTTPCreateIndividualSubcription` -> `POST /subscriptions`
  - `HTTPDeleteIndividualSubcription` -> `DELETE /subscriptions/:subId`
  - `HTTPGetIndividualSubcription` -> `GET /subscriptions/:subId`
  - `HTTPReplaceIndividualSubcription` -> `PUT /subscriptions/:subId`

## 3) SMF 內部資料路徑（SUPI -> PDU Session -> UE IP -> UPF -> apiRoot）

1. **訂閱請求解析**：從 `NsmfEventExposure` 讀取 `supi` 或 `pduSeId`（優先使用 `pduSeId` 精準定位）。
2. **定位 SMContext**：
   - 若包含 `supi + pduSeId`：呼叫 `smf/internal/context/sm_context.go` 的 `GetSMContextById(supi, pduSeId)`。
   - 若僅有 `supi`：需新增一個遍歷 `smContextPool` 的查找函式（依 `Supi` 及可選 `DNN/S-NSSAI` 篩選，取 Active 的 session）。
3. **取得 UE IP**：`smContext.PDUAddress`（若為空，視為無有效 PDU Session）。
4. **確定選用 UPF**：`smContext.SelectedUPF`（`*context.UPNode`），透過 `SelectedUPF.UPF` 取得 UPF 識別。
5. **取得 UPF apiRoot**：
   - 建議在 `smfcfg.yaml` 的 UPF 節點中新增 `eventExposureApiRoot`（或 `sbiApiRoot`）欄位。
   - `user_plane_information.go` 解析後附加到 `UPNode` 或 `UPF` 結構上。
6. **級聯呼叫 UPF**：使用 `Nupf_EventExposure` 的 `CreateSubscription/DeleteSubscription`，填入 `ueIpAddress`、`eventList`、`eventNotifyUri`、`notifyCorrelationId`。

## 4) In-memory 狀態設計（對齊 Task1 相關關聯模式）

**核心索引**
- `nsmfSubId -> NsmfSubscriptionState`
- `nsmfSubId -> UpfSubscriptionState`

**NsmfSubscriptionState（訂閱請求資訊）**
- `nsmfSubId`
- `supi`
- `pduSeId`
- `notifId`
- `notifUri`
- `eventSubs`（包含 `upfEvents` 原樣保留）
- `dnn` / `snssai`
- `createdAt`（可選）

**UpfSubscriptionState（級聯 UPF 訂閱結果）**
- `nsmfSubId`
- `upfSubId`
- `upfLocation`（UPF 201 Location）
- `selectedUpf`（UPF name/NodeID）
- `upfApiRoot`

**關聯模式借鏡 Task1**
- 依 Task1 的 `subscriptionId -> state` 與 `subscriptionId + notifCorrId` 方式，為 Task2 保留 `nsmfSubId` 與 `notifId` 的組合索引，以便後續通知或聯動清理。

## 5) 錯誤策略（無有效 PDU Session / UE IP）

- **缺失 SUPI / pduSeId**：回傳 `400 Bad Request`（`ProblemDetails`，使用 `openapi.ProblemDetailsMalformedReqSyntax` 風格）。
- **找不到 SMContext**：回傳 `404 Not Found`（`ProblemDetails`，使用 `openapi.ProblemDetailsDataNotFound`，detail 指明 `SMContext not found`）。
- **SMContext 存在但無 UE IP**：回傳 `404 Not Found`（`ProblemDetails`，detail 指明 `UE IP not allocated`）。
- **UPF apiRoot 缺失或解析失敗**：回傳 `502 Bad Gateway`（`ProblemDetailsSystemFailure` 或 `ProblemDetailsDataNotFound` 視實作約束）。

以上回應碼符合 TS 29.508 中對 `POST /subscriptions` 允許的 `400/404/5xx` 族群，且採用 CommonData `ProblemDetails` 結構。

## 6) Patch 序列（Patch1..PatchN）與驗收標準

**Patch1: 狀態與設定骨架**
- 內容：新增 Task2 訂閱 store；擴充 UPF 設定結構新增 `eventExposureApiRoot`；初始化 store。
- 驗收：`go test`/`go build` 通過（不觸發編譯錯誤）；`smfcfg.yaml` 可載入新欄位。

**Patch2: UPF EventExposure consumer**
- 內容：新增 `upf_eventexposure_service.go`；封裝 `CreateSubscription/DeleteSubscription`；處理 Location 解析。
- 驗收：consumer 具備 `SendCreate/SendDelete`；錯誤可對映為 `ProblemDetails` 或 `error` 回傳。

**Patch3: Nsmf_EventExposure Create/Delete 主流程**
- 內容：實作 `HTTPCreateIndividualSubcription` / `HTTPDeleteIndividualSubcription`；建立 Nsmf 訂閱 -> 解析 SMContext -> 級聯 UPF 訂閱 -> 記錄雙層狀態 -> 回傳 `201 + Location`；刪除時級聯 UPF 刪除並清理狀態 -> 回傳 `204`。
- 驗收：POST/DELETE 不再回傳 501；成功時 Location 與 body 符合 TS 29.508；失敗情境回傳 ProblemDetails。

**Patch4: 路徑補強與一致性**
- 內容：補充 `Get`/`Replace` 的最小行為（可 `501` 或明確 `405`）；增加日誌欄位（supi、nsmfSubId、upfSubId、http_status）。
- 驗收：日誌可追蹤訂閱生命週期；與 Task1 的日誌風格一致。
