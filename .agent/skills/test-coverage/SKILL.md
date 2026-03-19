---
description: 執行多語言測試並產生覆蓋率報告 (Go, Python, Node.js)
---

此工作流程將引導您執行各語言專案的測試並分析覆蓋率。

1. **Go (Golang)**

    * **執行測試與收集覆蓋率**:

        ```bash
        go test -v -coverprofile=coverage.out ./...
        ```

    * **顯示終端機摘要**:

        ```bash
        go tool cover -func=coverage.out
        ```

    * **產生 HTML 報告**:

        ```bash
        go tool cover -html=coverage.out -o coverage.html
        ```

2. **Python**

    假設使用 `pytest` 與 `pytest-cov` 套件。

    * **執行測試並產生報告**:

        ```bash
        # 需安裝 pytest-cov
        pytest --cov=. --cov-report=term --cov-report=html
        ```

    * **報告位置**:
        * 終端機顯示摘要
        * 詳細 HTML 報告產生於 `htmlcov/index.html`

3. **Node.js**

    假設使用 `Jest` 測試框架。

    * **執行測試**:

        ```bash
        # 確保 package.json 設定 "test": "jest"
        npm test -- --coverage
        ```

    * **報告位置**:
        * 終端機顯示摘要
        * 詳細 HTML 報告產生於 `coverage/lcov-report/index.html`

4. **結果分析提示**

    * **整體覆蓋率**: 檢查報告中的總體覆蓋率百分比（通常建議 > 80%）。
    * **熱點分析**: 重點檢查核心商業邏輯或高風險模組的覆蓋情況。
    * **清理**: 測試完成後可刪除產生的測試報告檔案 (如 `coverage.out`, `htmlcov/`, `coverage/`)。
