# SMF Task1 移除 notifCorrId 依賴（相容性變更）

本文件說明 Task1 移除 notifCorrId 依賴的行為變更與相容性策略。

## 變更摘要

- Subscribe/Notify 流程不再要求 `notifCorrId`。  
- `subscriptionId` 成為主要關聯鍵；若 `notifCorrId` 存在，僅用於輔助紀錄。  
- 仍符合 TS 29.520：Create 201 + Location、Notify 204、Delete 204。

## 相容性說明

- 既有 payload 中仍可包含 `notifCorrId`，SMF 會接收並記錄。  
- 若 `notifCorrId` 缺失，SMF 會以 `subscriptionId` 進行回呼關聯，不影響流程。  
- 參考：`docs/contract/nnwdaf-uecomm-subscription.md` 內新增說明（notifCorrId 選填）。

## 影響範圍

- Request building：僅在 `notifCorrId` 明確提供時傳入 NWDAF  
- Callback handler：只依 `subscriptionId` 查找本地 state  
- In-memory store：以 `subscriptionId` 作為主索引  
- Logging：僅在 `notifCorrId` 存在時記錄該欄位

