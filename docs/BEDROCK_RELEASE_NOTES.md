# AWS Bedrock LLM Provider

AWS Bedrock support has been added to k8flex! 

## What was added

- **New Provider**: `bedrock` - AWS Bedrock integration with Claude, Titan, and other models
- **IAM Authentication**: Uses AWS IAM roles (no API keys needed)
- **Streaming Support**: Real-time analysis updates via Bedrock streaming API
- **Multi-Region**: Support for all AWS regions where Bedrock is available

## Quick Start

### 1. Configure Helm Values

```yaml
config:
  llm:
    provider: "bedrock"
  bedrock:
    region: "us-east-1"
    model: "anthropic.claude-3-5-sonnet-20241022-v2:0"
```

### 2. Set up IAM Role (EKS)

```bash
# Create IAM policy with Bedrock permissions
aws iam create-policy \
  --policy-name K8flexBedrockAccess \
  --policy-document '{
    "Version": "2012-10-17",
    "Statement": [{
      "Effect": "Allow",
      "Action": [
        "bedrock:InvokeModel",
        "bedrock:InvokeModelWithResponseStream"
      ],
      "Resource": "arn:aws:bedrock:*::foundation-model/*"
    }]
  }'

# Create service account with IAM role
eksctl create iamserviceaccount \
  --name=k8flex-agent \
  --namespace=k8flex \
  --cluster=your-cluster \
  --attach-policy-arn=arn:aws:iam::ACCOUNT:policy/K8flexBedrockAccess \
  --approve
```

### 3. Deploy

```bash
helm upgrade --install k8flex ./helm/k8flex -n k8flex
```

## Available Models

### Anthropic Claude (Recommended)
- `anthropic.claude-3-5-sonnet-20241022-v2:0` - Best balance ($0.003/1K tokens)
- `anthropic.claude-3-opus-20240229-v1:0` - Most capable ($0.015/1K tokens)
- `anthropic.claude-3-haiku-20240307-v1:0` - Fastest, cheapest ($0.00025/1K tokens)

### Amazon Titan
- `amazon.titan-text-express-v1` - Cost-effective ($0.0008/1K tokens)
- `amazon.titan-text-lite-v1` - Cheapest ($0.0003/1K tokens)

### AI21 Labs
- `ai21.j2-ultra-v1` - Alternative provider ($0.0188/1K tokens)

## Benefits

✅ **No API keys to manage** - Uses IAM authentication
✅ **Enterprise-grade security** - Compliant with AWS standards
✅ **Data privacy** - Data stays in your AWS account
✅ **Cost-effective** - Pay only for what you use
✅ **Multi-region** - Deploy in any supported AWS region

## Documentation

- **Full setup guide**: [docs/BEDROCK.md](../docs/BEDROCK.md)
- **Provider comparison**: [docs/LLM_PROVIDERS.md](../docs/LLM_PROVIDERS.md)

## Code Changes

- Added `pkg/llm/bedrock.go` - Bedrock provider implementation
- Updated `internal/config/config.go` - Added Bedrock configuration
- Updated `internal/app/app.go` - Added Bedrock provider initialization
- Updated Helm templates - Added Bedrock environment variables
- Added comprehensive documentation

## Next Steps

1. Enable Bedrock in AWS Console
2. Request model access
3. Set up IAM role for your cluster
4. Update Helm values
5. Deploy and enjoy!

For detailed instructions, see [docs/BEDROCK.md](../docs/BEDROCK.md).
