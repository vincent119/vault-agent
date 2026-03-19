# AWS Secrets Manager Guide & Examples

This document provides basic AWS CLI commands for operating AWS Secrets Manager, as well as example IAM policies and role attachments needed for Vault Agent to securely access specific secrets.

## 1. AWS Secrets Manager Basic Operations

### Creating a Secret

Use the `create-secret` command to store confidential data (e.g., in JSON format) into Secrets Manager:

```bash
aws secretsmanager create-secret \
    --name "devops/project/vault-agent-secret" \
    --description "Secret for Vault Agent testing" \
    --secret-string '{"username":"admin","password":"[PASSWORD]"}' \
    --region ap-southeast-1
```

### Retrieving a Secret

To test retrieving the secret value locally, execute the following command:

```bash
aws secretsmanager get-secret-value \
    --secret-id "devops/project/vault-agent-secret" \
    --region ap-southeast-1
```

---

## 2. AWS IAM Permissions Configuration

To allow the Vault Agent running inside Kubernetes (EKS) to correctly fetch secrets from AWS Secrets Manager, you must define an IAM Policy with read permissions.

### IAM Policy Example

Create (or attach as an inline policy) the following IAM Policy. Make sure the `Resource` ARNs match your AWS Account ID and the Secret name:

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

*(Note: Replace `<aws_account_id>` with your actual AWS account ID, and ensure the resource path matches your secret)*

### Attaching the Policy to an IAM Role (IRSA or EKS Node Role)

Once the IAM Policy is structured, attach it to the IAM Role assumed by the Vault Agent:

```bash
aws iam attach-role-policy \
    --role-name <your-eks-pod-or-node-role-name> \
    --policy-arn arn:aws:iam::<aws_account_id>:policy/<your-policy-name>
```

*(Note: For EKS environments, it is highly recommended to use the [IRSA (IAM Roles for Service Accounts)](https://docs.aws.amazon.com/eks/latest/userguide/iam-roles-for-service-accounts.html) mechanism to grant the Pod least-privilege access)*
