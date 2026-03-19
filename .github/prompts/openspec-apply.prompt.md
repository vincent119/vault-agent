---
description: Implement an approved OpenSpec change and keep tasks in sync.
---

$ARGUMENTS
<!-- OPENSPEC:START -->
**指導原則**
- 優先採用簡單、最小化的實作方式，僅在明確要求或必要時才增加複雜度。
- 保持變更範圍緊密聚焦於請求的結果。
- 如需額外的 OpenSpec 慣例或說明，請參考 `openspec/AGENTS.md`（位於 `openspec/` 目錄內——如果看不到，請執行 `ls openspec` 或 `openspec-tw update`）。

**步驟**
將這些步驟作為 TODO 追蹤，並逐一完成。
1. 閱讀 `changes/<id>/proposal.md`、`design.md`（如有）和 `tasks.md`，以確認範圍和驗收標準。
2. 依序執行任務，保持編輯最小化並專注於請求的變更。
3. 在更新狀態前確認完成——確保 `tasks.md` 中的每個項目都已完成。
4. 在所有工作完成後更新檢查清單，使每個任務都標記為 `- [x]` 並反映實際狀況。
5. 需要額外上下文時，參考 `openspec-tw list` 或 `openspec-tw show <item>`。

**參考**
- 如果在實作時需要提案的額外上下文，請使用 `openspec-tw show <id> --json --deltas-only`。
<!-- OPENSPEC:END -->
