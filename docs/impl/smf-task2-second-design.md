根據 free5gc/smf 代碼庫的分析，以下是支援 Task2（SMF→UPF Nupf_EventExposure subscribe）的內部資訊來源：

## 1) 如何從 SUPI 找到對應 PDU Session / UE IP

**關鍵函式與 struct：**  in `sm_context.go:361-371`, `sm_context.go:261-268` 

**SMContext 核心結構：** in `sm_context.go:112-135` 

**使用方式：** 透過 `GetSMContextById(supi, pduSessionId)` 即可取得包含 `PDUAddress`（UE IP）的完整 session context。

---

## 2) 如何從 PDU Session / DNN / S-NSSAI 推斷使用哪個 UPF

**UPF 選擇參數結構：** in `upf.go:93-99` 

**UPF 選擇核心邏輯：** in `user_plane_information.go:859-903`

**UPF 匹配檢查：** in `user_plane_information.go:48-64`

**UPNode 結構（含 NodeID）：** in `user_plane_information.go:37-46`

**使用方式：** SMContext 建立時會呼叫 `SelectUPFAndAllocUEIP()`，結果存於 `smContext.SelectedUPF`，可從中取得 UPF NodeID 與 PFCP endpoint。

---

## 3) SMF 與 UPF 溝通（PFCP/路由）相關的 context/selector

**PFCP Session Context 儲存位置：** in `sm_context.go:166`

**PFCP Session Context 結構定義：** in `pfcp_session_context.go:26-31`

**PFCP 訊息傳送函式：** in `send.go:126-150`

**PFCP Session 啟動流程：** in `datapath.go:31-102`

**關鍵檔案結構：**
- **Context 管理：** `internal/context/sm_context.go` - SMContext.PFCPContext（map[NodeIP]→PFCPSessionContext）
- **PFCP 通訊：** `internal/pfcp/message/send.go` - 各種 PFCP 訊息傳送函式
- **UPF 狀態：** `internal/context/upf.go` - UPF struct 含 NodeID、Addr、SNssaiInfos
- **Session 建立：** `internal/sbi/processor/datapath.go` - ActivateUPFSession 統籌所有 UPF 的 PFCP session

---

## 4) 最適合的 Hook 點建議

### 建議 Hook 點 1：Event Exposure API Handler（推薦） 
in `api_eventexposure.go:56-58`

**理由：**
1. 此處是 NWDAF→SMF 訂閱的入口點，目前為 stub 實作（StatusNotImplemented）
2. 可在此接收訂閱請求後，解析 SUPI/PDU Session，查詢對應 SMContext，再向關聯的 UPF 發起訂閱
3. 完全獨立於 PDU Session 建立/修改主流程，不會影響現有邏輯

### 建議 Hook 點 2：PDU Session 建立後（次選）
in `pdu_session.go:258` 

**理由：**
1. 在 `ActivateUPFSession` 呼叫後，PFCP session 已建立，UPF 資訊完整可用
2. 可在此處檢查是否有待處理的 event subscription，批次發起 SMF→UPF 訂閱
3. 類似現有的 Event Exposure notification 機制（notifier.go）

### 建議 Hook 點 3：SM Policy Update 通知後（參考） 
in `notifier.go:114-118`

**理由：**
1. 展示了完整的 Event Exposure notification 發送流程（含 EARLY/LATE 通知）
2. 可參考此模式，在 UPF session 更新時同步更新 SMF→UPF 訂閱
3. 適合處理動態路徑變更時的訂閱維護

---

## Notes

**重要提醒：**
- 目前 SMF 的 Event Exposure API（`api_eventexposure.go`）僅有 stub 實作，需要完整實作訂閱管理邏輯
- SMF→UPF 的 Nupf_EventExposure 訂閱需要額外開發（目前 PFCP 層未包含此功能）
- 建議在 SMContext 中新增欄位來追蹤 event subscriptions 與對應的 UPF subscription IDs
- 可參考 `UpPathChgEarlyNotification` / `UpPathChgLateNotification` 的設計模式來管理訂閱狀態
