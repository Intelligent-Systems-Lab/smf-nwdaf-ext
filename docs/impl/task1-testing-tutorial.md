# Task1 測試教學（Unit Tests）

本文件說明如何執行 Task1 的單元測試，以及常見失敗原因的判讀。  
測試目標依據：`docs/contract/nnwdaf-uecomm-subscription.md`、`docs/spec/TS29520_Nnwdaf_EventsSubscription.yaml`。

## 前置需求

- Go 版本：請符合 `smf/go.mod` 中的版本要求（目前為 Go 1.25.5）。
- 已完成 `go env` 基本設定，並能執行 `go test`。

## 步驟一：跑全部測試（包含 Task1）

在 `smf/` 目錄下執行：

```bash
go test ./...
```

## 步驟二：只跑 Task1 相關測試

1) NWDAF subscription processor 的測試：

```bash
go test ./internal/sbi/processor -run TestHandleOAMCreateNwdafSubscription_BuildsContractPayload
```

2) NWDAF callback handler 的測試：

```bash
go test ./internal/sbi -run TestHTTPNwdafEventsNotification
```

## 常見失敗與判讀方式

1) **JSON shape mismatch**  
   - 可能原因：payload 欄位名稱大小寫不符 contract  
   - 參考：`docs/contract/nnwdaf-uecomm-subscription.md`「建立訂閱 / 事件通知」

2) **Location parsing failure**  
   - 可能原因：Location header 格式與 `.../subscriptions/{subscriptionId}` 不一致  
   - 參考：`docs/spec/TS29520_Nnwdaf_EventsSubscription.yaml` → `201 Location`

3) **Wrong status code**  
   - callback 非 array / 空陣列 → 應回 400  
   - 正常 callback → 應回 204  
   - 參考：`docs/spec/TS29520_Nnwdaf_EventsSubscription.yaml` callbacks

## TS 29.520 行為對應表（測試覆蓋）

- **Create 201 + Location**  
  - 對應測試：`TestHandleOAMCreateNwdafSubscription_BuildsContractPayload`  
  - 重點：驗證 request 內容與 contract，並檢查 Location 可解析

- **Notify callback 204**  
  - 對應測試：`TestHTTPNwdafEventsNotification_OK`  
  - 重點：array payload 正確回 204

- **Delete 204**  
  - Delete 行為由 handler 及 client 邏輯覆蓋，尚未額外建立單獨 unit test  
  - 可依後續需求擴充

