# SMF Task1 測試變更摘要

本文件記錄 Task1 新增的單元測試與覆蓋範圍。

## 新增檔案

- `internal/sbi/processor/nwdaf_subscription_test.go`
  - 驗證 Subscribe payload 組成符合 contract  
  - 驗證 Location header 解析穩健性

- `internal/sbi/api_nwdaf_callback_test.go`
  - 驗證 Notify callback handler 接收 array 回 204  
  - 驗證 invalid payload 回 400

## 覆蓋行為

對應 TS 29.520 重點：  
- Create (201 + Location)  
- Notify callback (204)  

Delete (204) 尚未建立獨立 unit test，可視需求擴充。

