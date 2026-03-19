---
description: Kubernetes 應用程式除錯流程 (Pod Status, Logs, Events, Exec)
---

此工作流程將引導您排查與修復 Kubernetes 上的應用程式問題。

1. **檢查 Pod 狀態 (Pod Status)**

    首先確認 Pod 是否正常運作，或處於錯誤狀態 (如 CrashLoopBackOff, ImagePullBackOff)。

    ```bash
    kubectl get pods -o wide
    ```

    若發現問題 Pod，查看詳細資訊：

    ```bash
    kubectl describe pod <POD_NAME>
    ```

2. **查看應用程式日誌 (Logs)**

    檢查容器輸出的標準日誌，尋找錯誤訊息或堆疊追蹤 (Stack Trace)。

    ```bash
    kubectl logs <POD_NAME>
    # 若 Pod 包含多個容器，需指定容器名稱
    # kubectl logs <POD_NAME> -c <CONTAINER_NAME>
    ```

    若 Pod 不斷重啟，查看上一輪的崩潰日誌：

    ```bash
    kubectl logs <POD_NAME> --previous
    ```

3. **檢查叢集事件 (Events)**

    查看 Namespace 中的事件，了解排程、掛載或健康檢查失敗的原因。

    ```bash
    kubectl get events --sort-by=.lastTimestamp
    ```

4. **進入容器除錯 (Exec)**

    若需檢查檔案系統、環境變數或網路連線，可直接進入容器。

    ```bash
    kubectl exec -it <POD_NAME> -- sh
    # 或
    kubectl exec -it <POD_NAME> -- bash
    ```

    * **檢查環境變數**: `env`
    * **檢查網路連線**: `curl localhost:8080` 或 `nc -zv <HOST> <PORT>`

5. **本地端口轉發 (Port Forwarding)**

    將 Pod 端口轉發到本地，以便使用瀏覽器或 Postman 進行測試。

    ```bash
    kubectl port-forward <POD_NAME> 8080:8080
    ```

6. **資源使用量監控 (Top)**

    檢查 Pod 是否因 OOM (Out of Memory) 或 CPU 節流而導致效能問題。

    ```bash
    kubectl top pod <POD_NAME>
    ```
