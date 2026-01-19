# Task1 概觀：架構與資料流

本文件以中文整理 Task1 的「SMF 作為 consumer 訂閱 NWDAF UE_COMMUNICATION」架構，讓讀者不需閱讀 patch notes 即可理解全貌。  
來源引用：`docs/contract/nnwdaf-uecomm-subscription.md`、`docs/spec/TS29520_Nnwdaf_EventsSubscription.yaml`、`docs/impl/smf-task1-first-patch.md`、`docs/impl/smf-task1-second-patch.md`、`docs/impl/smf-task1-third-patch.md`。

## 面向 1：目的 / 故事線（UE Communication 訂閱）

目標是讓 SMF 作為 NWDAF 的 consumer，訂閱 UE Communication 分析事件，並在收到通知時回覆 204。  
完整流程為：  
1) SMF 發起 CreateNWDAFEventsSubscription（201 + Location）  
2) NWDAF 透過 notificationURI 回呼通知（204）  
3) SMF 依 subscriptionId 取消訂閱（204）  
參考：`docs/contract/nnwdaf-uecomm-subscription.md`「建立訂閱/事件通知/刪除訂閱」段落。

## 面向 2：介面與規格對齊（TS 29.520）

關鍵對齊點如下（以 TS 29.520 為基準）：  
- Create：`POST /nnwdaf-eventssubscription/v1/subscriptions`，回應 `201`，必須包含 `Location` header  
  參考：`docs/spec/TS29520_Nnwdaf_EventsSubscription.yaml` paths `/subscriptions` → `201` responses。  
- Notify callback：由 NWDAF 送到 `notificationURI`，Body 為 `array`，SMF 回 `204`  
  參考：`docs/spec/TS29520_Nnwdaf_EventsSubscription.yaml` callbacks `myNotification`。  
- Delete：`DELETE /nnwdaf-eventssubscription/v1/subscriptions/{subscriptionId}`，回 `204`  
  參考：`docs/spec/TS29520_Nnwdaf_EventsSubscription.yaml` paths `/subscriptions/{subscriptionId}` → delete。

Contract 黃金 payload 以 `docs/contract/nnwdaf-uecomm-subscription.md` 為準：  
- Create body 需包含 `eventSubscriptions[].event=UE_COMMUNICATION`、`tgtUe.supis`、`notificationURI`、`evtReq`；`notifCorrId` 為選填  
  參考：`docs/contract/nnwdaf-uecomm-subscription.md`「建立訂閱 (Create Subscription) / Request Body」。  
- Notify body 為 `array`，元素包含 `subscriptionId`、`notifCorrId`、`eventNotifications[]`、`ueComms[]`  
  參考：`docs/contract/nnwdaf-uecomm-subscription.md`「事件通知 (Notify) / Request Body」。

## 面向 3：SMF 內部模組與資料流

以下為 Task1 在 SMF 內部的主要模組與資料流（概念化）：  

1) OAM 觸發建立/刪除  
   - 入口：`internal/sbi/api_oam.go`  
   - 行為：接收 OAM request，轉交 processor。  
   參考：`docs/impl/smf-task1-first-patch.md`「OAM 管理入口」。

2) Processor 負責建/刪/通知處理  
   - 檔案：`internal/sbi/processor/nwdaf_subscription.go`  
   - Create：建立 NWDAF 訂閱、解析 `Location`、寫入記憶體 map  
   - Notify：接受 `array` 通知、用 `subscriptionId` / `notifCorrId` 做關聯  
   - Delete：以 `subscriptionId` 呼叫 NWDAF Delete 並清理 state  
   參考：`docs/impl/smf-task1-second-patch.md`「TS 29.520 行為對齊清單」。

3) Consumer 端呼叫 NWDAF  
   - 檔案：`internal/sbi/consumer/nwdaf_service.go`  
   - Create/ Delete 呼叫對應的 NWDAF API  
   參考：`docs/impl/smf-task1-first-patch.md`「實作位置與檔案」。

4) 訂閱狀態儲存  
   - 檔案：`internal/context/nwdaf_subscription.go`  
   - key：`subscriptionId`（主要）  
   - `notifCorrId` 若存在僅作為記錄用途  
   參考：`docs/impl/smf-task1-first-patch.md`「訂閱狀態（In-memory map）」。

5) NWDAF callback route  
   - 入口：`/nwdaf-callback`  
   - Handler：`internal/sbi/api_nwdaf_callback.go`  
   參考：`docs/impl/smf-task1-first-patch.md`「NWDAF 通知 callback」。
