---
description: 通用專案發布流程 (測試 -> 標記 -> 推送)
---

此工作流程將引導您完成專案新版本的發布過程。適用於 Go, Python, Node.js 等各類專案。

1. **檢查 Git 狀態**

    在發布前確保您的工作目錄是乾淨的。

    ```bash
    git status --porcelain
    ```

    如果有變更，請在繼續之前提交 (commit) 或暫存 (stash) 它們。

2. **執行測試**

    執行所有測試以確保發布版本穩定。請根據專案語言選擇對應指令：

    * **Go**:

        ```bash
        go test -v ./...
        ```

    * **Python**:

        ```bash
        # 假設使用 pytest
        pytest
        ```

    * **Node.js**:

        ```bash
        npm test
        ```

    * **Bash (Shell Script)**:

        ```bash
        # 語法檢查
        bash -n script.sh
        # 或使用 bats 測試框架
        # bats test/
        ```

    * **YAML**:

        ```bash
        # 使用 yamllint 進行語法驗證
        yamllint .
        ```

3. **決定版本**

    檢查目前的標籤以決定下一個版號。

    ```bash
    git tag --sort=-v:refname | head -n 5
    ```

    *詢問使用者：「下一個版本標籤應該是什麼？（例如：v1.0.1）」*

4. **建立 Git 標籤 (Tag)**

    在本地建立標籤。

    ```bash
    # 將 <VERSION> 替換為使用者提供的版本
    git tag <VERSION>
    ```

5. **推送到遠端**

    將新標籤推送到遠端儲存庫。

    ```bash
    git push origin <VERSION>
    ```

6. **驗證**

    確認標籤已在遠端（可選）。

    ```bash
    git ls-remote --tags origin | grep <VERSION>
    ```
