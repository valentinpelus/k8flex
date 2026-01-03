# AWS Bedrock Integration Guide

This guide explains how to configure k8flex to use AWS Bedrock as the LLM provider.

## What is AWS Bedrock?

AWS Bedrock provides access to foundation models from leading AI companies through a single API. It includes:
- **Anthropic Claude** (Claude 3 Opus, Sonnet, Haiku)
- **Amazon Titan**
- **AI21 Labs Jurassic**
- **Cohere Command**
- **Meta Llama 2**
- **Stability AI**

Benefits:
- ✅ Enterprise-grade security and compliance
- ✅ No API keys to manage (uses IAM)
- ✅ Pay-per-use pricing
- ✅ Data privacy (stays in your AWS account)
- ✅ Multi-region availability

---

## Prerequisites

### 1. Enable Bedrock in AWS Console

1. Go to [AWS Bedrock Console](https://console.aws.amazon.com/bedrock/)
2. Select your region (e.g., `us-east-1`)
3. Navigate to **Model access**
4. Request access to models you want to use:
   - ✅ **Anthropic Claude 3.5 Sonnet** (recommended)
   - ✅ Anthropic Claude 3 Opus (most capable)
   - ✅ Amazon Titan Text Express (cost-effective)

Access is usually granted immediately for most models.

---

## Setup for EKS (Recommended)

### Step 1: Create IAM Role for Bedrock Access

Create an IAM policy with Bedrock permissions:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "bedrock:InvokeModel",
        "bedrock:InvokeModelWithResponseStream"
      ],
      "Resource": [
        "arn:aws:bedrock:*::foundation-model/*"
      ]
    }
  ]
}
```

Save as `bedrock-policy.json` and create the policy:

```bash
aws iam create-policy \
  --policy-name K8flexBedrockAccess \
  --policy-document file://bedrock-policy.json
```

### Step 2: Create IAM Role for Service Account (IRSA)

Replace `ACCOUNT_ID`, `REGION`, and `CLUSTER_NAME`:

```bash
# Set variables
ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
REGION="us-east-1"
CLUSTER_NAME="your-eks-cluster"
NAMESPACE="k8flex"

# Create OIDC provider for your cluster (if not already created)
eksctl utils associate-iam-oidc-provider \
  --cluster=$CLUSTER_NAME \
  --region=$REGION \
  --approve

# Create IAM role with trust policy for service account
eksctl create iamserviceaccount \
  --name=k8flex-agent \
  --namespace=$NAMESPACE \
  --cluster=$CLUSTER_NAME \
  --region=$REGION \
  --attach-policy-arn=arn:aws:iam::${ACCOUNT_ID}:policy/K8flexBedrockAccess \
  --approve \
  --override-existing-serviceaccounts
```

This creates:
- IAM role: `eksctl-CLUSTER_NAME-addon-iamserviceaccount-k8flex-k8flex-agent`
- Service account annotation: `eks.amazonaws.com/role-arn`

### Step 3: Configure Helm Values

Edit `helm/k8flex/values.yaml`:

```yaml
config:
  llm:
    provider: "bedrock"
  
  bedrock:
    region: "us-east-1"  # Your AWS region
    model: "anthropic.claude-3-5-sonnet-20241022-v2:0"

serviceAccount:
  create: true
  name: "k8flex-agent"
  # This will be auto-configured by eksctl above
  # annotations:
  #   eks.amazonaws.com/role-arn: arn:aws:iam::ACCOUNT_ID:role/eksctl-...
```

### Step 4: Deploy

```bash
helm upgrade --install k8flex ./helm/k8flex \
  --namespace k8flex \
  --create-namespace
```

---

## Setup for Non-EKS (Using AWS Credentials)

If you're not using EKS or prefer using AWS credentials directly:

### Option 1: Use AWS Credentials Secret

Create a Kubernetes secret with AWS credentials:

```bash
kubectl create secret generic aws-credentials \
  --from-literal=AWS_ACCESS_KEY_ID=AKIA... \
  --from-literal=AWS_SECRET_ACCESS_KEY=... \
  --from-literal=AWS_REGION=us-east-1 \
  -n k8flex
```

Update deployment to use the secret:

```yaml
# In helm/k8flex/templates/deployment.yaml
containers:
  - name: k8flex
    env:
      - name: AWS_ACCESS_KEY_ID
        valueFrom:
          secretKeyRef:
            name: aws-credentials
            key: AWS_ACCESS_KEY_ID
      - name: AWS_SECRET_ACCESS_KEY
        valueFrom:
          secretKeyRef:
            name: aws-credentials
            key: AWS_SECRET_ACCESS_KEY
      - name: AWS_REGION
        valueFrom:
          secretKeyRef:
            name: aws-credentials
            key: AWS_REGION
```

### Option 2: Use EC2 Instance Profile

If running on EC2 instances, attach the IAM role directly to the instances:

```bash
# Attach policy to EC2 instance role
aws iam attach-role-policy \
  --role-name your-node-instance-role \
  --policy-arn arn:aws:iam::${ACCOUNT_ID}:policy/K8flexBedrockAccess
```

No additional configuration needed - the AWS SDK will automatically use the instance profile.

---

## Available Bedrock Models

### Anthropic Claude Models (Recommended)

| Model ID | Name | Context | Best For | Cost/1K tokens |
|----------|------|---------|----------|----------------|
| `anthropic.claude-3-5-sonnet-20241022-v2:0` | Claude 3.5 Sonnet | 200K | **Best balance** | $0.003 |
| `anthropic.claude-3-opus-20240229-v1:0` | Claude 3 Opus | 200K | Most capable | $0.015 |
| `anthropic.claude-3-sonnet-20240229-v1:0` | Claude 3 Sonnet | 200K | Fast, balanced | $0.003 |
| `anthropic.claude-3-haiku-20240307-v1:0` | Claude 3 Haiku | 200K | Fastest, cheapest | $0.00025 |

### Amazon Titan Models

| Model ID | Name | Context | Cost/1K tokens |
|----------|------|---------|----------------|
| `amazon.titan-text-express-v1` | Titan Text Express | 8K | $0.0008 |
| `amazon.titan-text-lite-v1` | Titan Text Lite | 4K | $0.0003 |

### AI21 Labs Models

| Model ID | Name | Context | Cost/1K tokens |
|----------|------|---------|----------------|
| `ai21.j2-ultra-v1` | Jurassic-2 Ultra | 8K | $0.0188 |
| `ai21.j2-mid-v1` | Jurassic-2 Mid | 8K | $0.0125 |

---

## Configuration Examples

### Example 1: Claude 3.5 Sonnet (Recommended)

```yaml
config:
  llm:
    provider: "bedrock"
  bedrock:
    region: "us-east-1"
    model: "anthropic.claude-3-5-sonnet-20241022-v2:0"
```

### Example 2: Amazon Titan (Cost-Effective)

```yaml
config:
  llm:
    provider: "bedrock"
  bedrock:
    region: "us-west-2"
    model: "amazon.titan-text-express-v1"
```

### Example 3: Claude 3 Opus (Maximum Quality)

```yaml
config:
  llm:
    provider: "bedrock"
  bedrock:
    region: "eu-west-1"
    model: "anthropic.claude-3-opus-20240229-v1:0"
```

---

## Region Availability

Bedrock is available in these regions:

- **US**: `us-east-1`, `us-west-2`
- **Europe**: `eu-west-1` (Ireland), `eu-central-1` (Frankfurt)
- **Asia Pacific**: `ap-southeast-1` (Singapore), `ap-northeast-1` (Tokyo)

Check current availability: https://docs.aws.amazon.com/bedrock/latest/userguide/bedrock-regions.html

---

## Troubleshooting

### "AccessDeniedException: User is not authorized"

**Cause**: IAM role lacks Bedrock permissions

**Fix**:
```bash
# Verify IAM role has the policy attached
aws iam list-attached-role-policies --role-name your-role-name

# Attach the policy if missing
aws iam attach-role-policy \
  --role-name your-role-name \
  --policy-arn arn:aws:iam::ACCOUNT_ID:policy/K8flexBedrockAccess
```

### "ValidationException: The provided model identifier is invalid"

**Cause**: Model ID incorrect or model not available in your region

**Fix**:
- Check model ID spelling
- Verify model is available in your region
- Request model access in Bedrock console

### "ResourceNotFoundException: Could not find model"

**Cause**: Model access not granted

**Fix**:
1. Go to Bedrock Console → Model access
2. Request access to the model
3. Wait for approval (usually instant)

### "ThrottlingException: Rate exceeded"

**Cause**: Too many requests to Bedrock

**Fix**:
- Reduce alert frequency
- Use a more economical model (Haiku instead of Opus)
- Request quota increase in Service Quotas console

### Check Pod Logs

```bash
kubectl logs -n k8flex deployment/k8flex --tail=50
```

Look for:
- "Using LLM provider: AWS Bedrock (model-id)"
- Any AWS-related error messages

### Verify IAM Role

```bash
# Check service account annotation
kubectl get sa k8flex-agent -n k8flex -o yaml | grep role-arn

# Should show:
# eks.amazonaws.com/role-arn: arn:aws:iam::ACCOUNT_ID:role/...
```

---

## Cost Optimization

### For High Volume (>1000 alerts/day)

Use **Claude 3 Haiku** or **Titan Text Express**:
```yaml
bedrock:
  model: "anthropic.claude-3-haiku-20240307-v1:0"  # ~$0.25/day
```

### For Low Volume (<100 alerts/day)

Use **Claude 3.5 Sonnet** (best quality):
```yaml
bedrock:
  model: "anthropic.claude-3-5-sonnet-20241022-v2:0"  # ~$3/day
```

### Cost Comparison (1000 alerts/day)

| Model | Input Tokens | Output Tokens | Daily Cost |
|-------|--------------|---------------|------------|
| Claude 3 Haiku | 2000 | 1000 | ~$0.75 |
| Titan Express | 2000 | 1000 | ~$2.40 |
| Claude 3.5 Sonnet | 2000 | 1000 | ~$9.00 |
| Claude 3 Opus | 2000 | 1000 | ~$45.00 |

---

## Security Best Practices

1. **Use IRSA (IAM Roles for Service Accounts)**
   - No credentials in code or configs
   - Automatic credential rotation
   - Fine-grained permissions

2. **Principle of Least Privilege**
   ```json
   {
     "Effect": "Allow",
     "Action": [
       "bedrock:InvokeModel",
       "bedrock:InvokeModelWithResponseStream"
     ],
     "Resource": [
       "arn:aws:bedrock:us-east-1::foundation-model/anthropic.claude-3-5-sonnet-20241022-v2:0"
     ]
   }
   ```

3. **Enable CloudTrail Logging**
   - Monitor Bedrock API calls
   - Detect unusual patterns

4. **Use VPC Endpoints** (Optional)
   - Keep traffic within AWS network
   - Reduce data egress costs

---

## Monitoring

### CloudWatch Metrics

Bedrock automatically publishes metrics:
- `InvocationLatency` - Response time
- `InvocationThrottles` - Rate limit hits
- `InvocationClientErrors` - Client errors (4xx)
- `InvocationServerErrors` - Server errors (5xx)

### Cost Tracking

Enable Cost Explorer and create alerts:
```bash
# View Bedrock costs
aws ce get-cost-and-usage \
  --time-period Start=2026-01-01,End=2026-01-31 \
  --granularity DAILY \
  --metrics BlendedCost \
  --filter file://bedrock-filter.json
```

**bedrock-filter.json**:
```json
{
  "Dimensions": {
    "Key": "SERVICE",
    "Values": ["Amazon Bedrock"]
  }
}
```

---

## Migration from Other Providers

### From Ollama
```yaml
# Before (Ollama)
config:
  llm:
    provider: "ollama"
  ollama:
    url: "http://ollama.svc:11434"
    model: "llama3"

# After (Bedrock)
config:
  llm:
    provider: "bedrock"
  bedrock:
    region: "us-east-1"
    model: "anthropic.claude-3-5-sonnet-20241022-v2:0"
```

### From OpenAI
```yaml
# Before (OpenAI)
config:
  llm:
    provider: "openai"
  openai:
    apiKey: "sk-..."
    model: "gpt-4-turbo-preview"

# After (Bedrock)
config:
  llm:
    provider: "bedrock"
  bedrock:
    region: "us-east-1"
    model: "anthropic.claude-3-5-sonnet-20241022-v2:0"
```

No code changes required - just update the configuration and redeploy!
