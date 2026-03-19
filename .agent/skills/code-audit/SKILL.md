---
description: 執行多語言程式碼品質與合規性檢查 (Go, Python, Bash, YAML)
---

此工作流程將引導您完成專案的程式碼品質檢查與規範驗證。

1. **Go 語言檢查 (Golang)**

    若專案包含 Go 程式碼，請執行以下檢查：

    * **格式與 Vet**:

        ```bash
        go fmt ./...
        go vet ./...
        ```

    * **Lint (GolangCI-Lint)**:

        ```bash
        golangci-lint run ./...
        ```

2. **Python 語言檢查**

    若專案包含 Python 程式碼 (`.py`)，請執行：

    * **Lint & Format Check (Ruff)**:

        ```bash
        ruff check .
        ruff format --check .
        ```

3. **Shell Script 檢查 (Bash)**

    若專案包含 Shell Scripts (`.sh`)，請執行：

    * **Static Analysis (ShellCheck)**:

        ```bash
        find . -type f -name "*.sh" -not -path "*/node_modules/*" -exec shellcheck {} +
        ```

4. **設定檔檢查 (YAML)**

    若專案包含 YAML 檔案 (`.yaml`, `.yml`)，請執行：

    * **Lint (yamllint)**:

        ```bash
        yamllint .
        ```

5. **自動修復 (Auto-fix)**

    針對檢測到的問題，若工具支援自動修復，可嘗試執行：

    * **Go**: `go fmt ./...`
    * **Python**: `ruff check --fix .` 與 `ruff format .`

6. **產出報告 (Report Generation)**

    請依照以下 Markdown 格式產出審計報告 (Artifact)：

    ```markdown
    # Code Audit Report

    **日期**: {YYYY-MM-DD}
    **專案**: {Project Name}
    **結果**: {✅ 通過 / ⚠️ 警告 / ❌ 失敗}

    ## 執行摘要
    {簡短總結整體結果}

    ## 詳細檢測項目

    ### 1. Go 語言檢查 (若適用)
    *   **Format**: {✅ 通過 / ❌ 失敗 (附原因)}
    *   **Vet**: {✅ 通過 / ❌ 失敗 (附原因)}
    *   **GolangCI-Lint**: {✅ 通過 / ❌ 發現 X 個問題}

    ### 2. 其他語言檢查 (若適用)
    *   **Python (Ruff)**: {✅ 通過 / ❌ 失敗}
    *   **Bash (ShellCheck)**: {✅ 通過 / ❌ 失敗}
    *   **YAML (yamllint)**: {✅ 通過 / ❌ 失敗}

    ### 3. 專案規範手動驗證 (Go)
    | 檢查項目 | 狀態 | 備註 |
    | :--- | :--- | :--- |
    | Package 宣告 | {✅/❌} | {說明} |
    | Imports 分組 | {✅/❌} | {說明} |
    | Error Handling | {✅/❌} | {說明} |
    | Context 使用 | {✅/❌} | {說明} |
    | Naming | {✅/❌} | {說明} |
    ```
