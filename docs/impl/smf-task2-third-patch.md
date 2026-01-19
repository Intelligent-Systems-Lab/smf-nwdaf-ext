# SMF Task2 Patch3 實作說明（級聯 SMF->UPF 退訂）

## 退訂級聯流程

1) NWDAF 呼叫 `DELETE /nsmf-event-exposure/v1/subscriptions/{subId}`  
2) SMF 取得 `nsmfSubId` 對應狀態（含 `upfLocation`/`upfSubId`）  
3) 若有 UPF 綁定，SMF 呼叫 UPF `DELETE /nupf-ee/v1/ee-subscriptions/{subscriptionId}`  
4) 不論 UPF 回應成功與否，SMF 都會清理本地記憶體狀態並回 `204 No Content`

## Idempotency 行為

- 若 `nsmfSubId` 不存在，視為已刪除，仍回 `204`。  
- 重複刪除不會造成 crash；清理動作可重入。

## 已知限制

- V0 僅支援單一 UE/單一 PDU Session；多 Session/多 UPF 仍未處理。  
- UPF 退訂結果僅記錄於日誌，未回傳給 NWDAF（依 204 回應規格）。  
