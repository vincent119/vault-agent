# AWS Secrets Manager 操作指南與範例

本文件提供 AWS Secrets Manager 的基本操作指令，以及讓 Vault Agent 存取特定 Secret 所需的 IAM Policy 與角色綁定範例。

## 1. AWS Secrets Manager 基本操作

### 建立 Secret

使用 `create-secret` 將機密資料（以 JSON 格式為例）存入 Secrets Manager 中：

```bash
aws secretsmanager create-secret \
    --name "devops/project/vault-agent-secret" \
    --description "Vault Agent 測試用 Secret" \
    --secret-string '{"username":"admin","password":"[PASSWORD]"}' \
    --region ap-southeast-1
```

### 讀取 Secret

如果想在本地測試讀取 Secret 內容，請執行以下指令：

```bash
aws secretsmanager get-secret-value \
    --secret-id "devops/project/vault-agent-secret" \
    --region ap-southeast-1
```

---

## 2. AWS IAM 權限設定配置

為了讓部署在 Kubernetes (EKS) 裡面的 Vault Agent 能夠正確獲取 AWS Secrets Manager 裡面的機密，您必須先建立一份具備讀取權限的 IAM Policy。

### IAM Policy 範例

請建立（或在既有角色上附加以 Inline Policy 形式）以下的 IAM Policy，並確保 `Resource` ARNs 符合您的 AWS Account ID 與 Secret 名稱：

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "secretsmanager:GetSecretValue",
                "secretsmanager:DescribeSecret"
            ],
            "Resource": "arn:aws:secretsmanager:ap-southeast-1:<aws_account_id>:secret:devops/project/vault-agent-secret-*"
        }
    ]
}
```

*(請將 `<aws_account_id>` 替換為實際的 AWS 帳號 ID，並且將 resource path 對應到您的 Secret)*

### 將 IAM Policy 附加至 IAM Role (IRSA 或 EKS Node Role)

建立好 IAM Policy 後，請將此 Policy 綁定至提供 Vault Agent 運行的 IAM Role：

```bash
aws iam attach-role-policy \
    --role-name <your-eks-pod-or-node-role-name> \
    --policy-arn arn:aws:iam::<aws_account_id>:policy/<your-policy-name>
```

*(備註：若是 EKS 環境，強烈建議使用 [IRSA (IAM Roles for Service Accounts)](https://docs.aws.amazon.com/eks/latest/userguide/iam-roles-for-service-accounts.html) 機制為 Pod 賦予最小權限)*
