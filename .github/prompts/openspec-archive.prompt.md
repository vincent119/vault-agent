---
description: Archive a deployed OpenSpec change and update specs.
---

$ARGUMENTS
<!-- OPENSPEC:START -->
**指導原則**
- 優先採用簡單、最小化的實作方式，僅在明確要求或必要時才增加複雜度。
- 保持變更範圍緊密聚焦於請求的結果。
- 如需額外的 OpenSpec 慣例或說明，請參考 `openspec/AGENTS.md`（位於 `openspec/` 目錄內——如果看不到，請執行 `ls openspec` 或 `openspec-tw update`）。

**步驟**
1. 確定要封存的變更 ID：
   - 如果此提示已包含特定的變更 ID（例如在由 slash 命令參數填充的 `<ChangeId>` 區塊內），請在修剪空白後使用該值。
   - 如果對話籠統地引用了一個變更（例如透過標題或摘要），執行 `openspec-tw list` 以找出可能的 ID，分享相關候選項，並確認使用者的意圖。
   - 否則，檢視對話，執行 `openspec-tw list`，並詢問使用者要封存哪個變更；在繼續之前等待確認的變更 ID。
   - 如果仍無法識別單一變更 ID，請停止並告知使用者您尚無法封存任何內容。
2. 透過執行 `openspec-tw list`（或 `openspec-tw show <id>`）驗證變更 ID，如果變更遺失、已封存或其他方式尚未準備好封存，請停止。
3. 執行 `openspec-tw archive <id> --yes`，使 CLI 移動變更並應用規範更新而不提示（僅在純工具工作時使用 `--skip-specs`）。
4. 檢視命令輸出以確認目標規範已更新，且變更已進入 `changes/archive/`。
5. 使用 `openspec-tw validate --strict` 驗證，如有任何異常，使用 `openspec-tw show <id>` 檢查。

**參考**
- 在封存前使用 `openspec-tw list` 確認變更 ID。
- 使用 `openspec-tw list --specs` 檢查已更新的規範，並在交接前解決任何驗證問題。
<!-- OPENSPEC:END -->
