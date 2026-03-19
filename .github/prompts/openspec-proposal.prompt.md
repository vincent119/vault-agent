---
description: Scaffold a new OpenSpec change and validate strictly.
---

$ARGUMENTS
<!-- OPENSPEC:START -->
**指導原則**
- 優先採用簡單、最小化的實作方式，僅在明確要求或必要時才增加複雜度。
- 保持變更範圍緊密聚焦於請求的結果。
- 如需額外的 OpenSpec 慣例或說明，請參考 `openspec/AGENTS.md`（位於 `openspec/` 目錄內——如果看不到，請執行 `ls openspec` 或 `openspec-tw update`）。
- 在編輯檔案前，先識別任何模糊或不明確的細節，並提出必要的後續問題。

**步驟**
1. 檢視 `openspec/project.md`，執行 `openspec-tw list` 和 `openspec-tw list --specs`，並檢查相關程式碼或文件（例如透過 `rg`/`ls`），以基於當前行為建立提案；記錄任何需要釐清的缺漏。
2. 選擇一個唯一的動詞開頭的 `change-id`，並在 `openspec/changes/<id>/` 下建立 `proposal.md`、`tasks.md` 和 `design.md`（如有需要）。
3. 將變更對應到具體的功能或需求，將多範圍的工作分解為明確關聯和順序的規範差異。
4. 當解決方案跨越多個系統、引入新模式或需要在提交規範前討論權衡時，在 `design.md` 中記錄架構推理。
5. 在 `changes/<id>/specs/<capability>/spec.md` 中撰寫規範差異（每個功能一個資料夾），使用 `## ADDED|MODIFIED|REMOVED Requirements`，每個需求至少包含一個 `#### Scenario:`，情境內容必須使用 Gherkin 格式：`- **WHEN** 動作`、`- **THEN** 結果`、`- **AND** 附加條件`，並在相關時交叉引用相關功能。
6. 將 `tasks.md` 撰寫為有序的小型、可驗證工作項目清單，提供使用者可見的進度，包含驗證（測試、工具），並標示依賴或可並行的工作。
7. 使用 `openspec-tw validate <id> --strict` 驗證，並在分享提案前解決所有問題。

**參考**
- 當驗證失敗時，使用 `openspec-tw show <id> --json --deltas-only` 或 `openspec-tw show <spec> --type spec` 檢查詳細資訊。
- 在撰寫新需求前，使用 `rg -n "Requirement:|Scenario:" openspec/specs` 搜尋現有需求。
- 透過 `rg <keyword>`、`ls` 或直接讀取檔案來探索程式碼庫，使提案符合當前實作現況。
<!-- OPENSPEC:END -->
