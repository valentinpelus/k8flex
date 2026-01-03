# Multi-LLM Provider Support

K8flex supports multiple LLM providers for alert analysis and categorization.

## Supported Providers

### 1. **Ollama** (Default)
Self-hosted, open-source LLM runtime. Great for local/on-premises deployments.

**Configuration:**
```yaml
config:
  llm:
    provider: "ollama"
  ollama:
    url: "http://ollama.ollama.svc.cluster.local:11434"
    model: "llama3"  # or llama2, mistral, mixtral, etc.
```

**Environment Variables:**
- `LLM_PROVIDER=ollama`
- `OLLAMA_URL=http://ollama.ollama.svc.cluster.local:11434`
- `OLLAMA_MODEL=llama3`

**Cost:** Free (self-hosted)

---

### 2. **OpenAI GPT**
Industry-leading models with excellent reasoning capabilities.

**Configuration:**
```yaml
config:
  llm:
    provider: "openai"
  openai:
    apiKey: "sk-proj-..."  # Your OpenAI API key
    model: "gpt-4-turbo-preview"
```

**Supported Models:**
- `gpt-4-turbo-preview` (Recommended - 128K context)
- `gpt-4` (8K context)
- `gpt-3.5-turbo` (16K context, cheaper)

**Environment Variables:**
- `LLM_PROVIDER=openai`
- `OPENAI_API_KEY=sk-proj-...`
- `OPENAI_MODEL=gpt-4-turbo-preview`

**Cost:** ~$0.01-0.03 per request (varies by model)

**Get API Key:** https://platform.openai.com/api-keys

---

### 3. **Anthropic Claude**
Advanced reasoning with strong technical analysis capabilities.

**Configuration:**
```yaml
config:
  llm:
    provider: "anthropic"
  anthropic:
    apiKey: "sk-ant-..."  # Your Anthropic API key
    model: "claude-3-5-sonnet-20241022"
```

**Supported Models:**
- `claude-3-5-sonnet-20241022` (Recommended - 200K context)
- `claude-3-opus-20240229` (Most capable, 200K context)
- `claude-3-sonnet-20240229` (Balanced, 200K context)

**Environment Variables:**
- `LLM_PROVIDER=anthropic` (or `claude`)
- `ANTHROPIC_API_KEY=sk-ant-...`
- `ANTHROPIC_MODEL=claude-3-5-sonnet-20241022`

**Cost:** ~$0.015-0.075 per request (varies by model)

**Get API Key:** https://console.anthropic.com/settings/keys

---

### 4. **Google Gemini**
Google's multimodal AI with strong reasoning.

**Configuration:**
```yaml
config:
  llm:
    provider: "gemini"
  gemini:
    apiKey: "AIza..."  # Your Google AI API key
    model: "gemini-1.5-pro"
```

**Supported Models:**
- `gemini-1.5-pro` (Recommended - 2M context)
- `gemini-pro` (1M context)

**Environment Variables:**
- `LLM_PROVIDER=gemini` (or `google`)
- `GEMINI_API_KEY=AIza...`
- `GEMINI_MODEL=gemini-1.5-pro`

**Cost:** ~$0.001-0.005 per request

**Get API Key:** https://makersuite.google.com/app/apikey

---

### 5. **AWS Bedrock** ⭐
Enterprise-grade AI service with multiple model providers. Ideal for AWS environments.

**Configuration:**
```yaml
config:
  llm:
    provider: "bedrock"
  bedrock:
    region: "us-east-1"
    model: "anthropic.claude-3-5-sonnet-20241022-v2:0"
```

**Supported Models:**
- `anthropic.claude-3-5-sonnet-20241022-v2:0` (Recommended)
- `anthropic.claude-3-opus-20240229-v1:0` (Most capable)
- `amazon.titan-text-express-v1` (Cost-effective)
- `ai21.j2-ultra-v1` (Alternative)

**Environment Variables:**
- `LLM_PROVIDER=bedrock` (or `aws`)
- `BEDROCK_REGION=us-east-1`
- `BEDROCK_MODEL=anthropic.claude-3-5-sonnet-20241022-v2:0`
- Uses IAM credentials (no API key needed!)

**Cost:** ~$0.003-0.015 per request (varies by model)

**Authentication:** Uses AWS IAM roles (IRSA for EKS) or instance profiles

**Setup Guide:** See [BEDROCK.md](BEDROCK.md) for detailed setup instructions

---

## Switching Providers

### Using Helm Values

Edit `helm/k8flex/values.yaml`:

```yaml
config:
  llm:
    provider: "openai"  # Change to: ollama, openai, anthropic, gemini, bedrock
  
  openai:
    apiKey: "sk-proj-YOUR-KEY-HERE"
    model: "gpt-4-turbo-preview"
```

Then upgrade:
```bash
helm upgrade k8flex ./helm/k8flex -n k8flex
```

### Using Environment Variables

Update your Kubernetes secret or deployment:

```bash
kubectl set env deployment/k8flex \
  -n k8flex \
  LLM_PROVIDER=anthropic \
  ANTHROPIC_API_KEY=sk-ant-YOUR-KEY \
  ANTHROPIC_MODEL=claude-3-5-sonnet-20241022
```

---

## Security Best Practices

### Store API Keys in Kubernetes Secrets

**Don't** put API keys directly in `values.yaml`:
```yaml
# ❌ BAD - Keys visible in Git
config:
  openai:
    apiKey: "sk-proj-actual-key-here"
```

**Do** use Kubernetes secrets:
```bash
# Create secret
kubectl create secret generic k8flex-llm-keys \
  --from-literal=OPENAI_API_KEY=sk-proj-YOUR-KEY \
  --from-literal=ANTHROPIC_API_KEY=sk-ant-YOUR-KEY \
  --from-literal=GEMINI_API_KEY=AIza-YOUR-KEY \
  -n k8flex

# Reference in deployment (already configured in templates/secret.yaml)
```

**For AWS Bedrock:** Use IAM roles instead of API keys (more secure):
```yaml
serviceAccount:
  annotations:
    eks.amazonaws.com/role-arn: arn:aws:iam::ACCOUNT:role/k8flex-bedrock
```

Then leave `apiKey` empty in values.yaml:
```yaml
config:
  openai:
    apiKey: ""  # Will be loaded from secret
```

---

## Provider Comparison

| Provider | Context Window | Speed | Cost/Req | Best For |
|----------|---------------|-------|----------|----------|
| **Ollama** | Varies (8K-128K) | Fast | Free | Self-hosted, privacy |
| **OpenAI GPT-4** | 128K | Medium | $0.01-0.03 | General purpose |
| **Claude** | 200K | Medium | $0.015-0.075 | Technical analysis |
| **Gemini** | 2M | Fast | $0.001-0.005 | Large context |
| **Bedrock** | Varies | Medium | $0.003-0.015 | AWS integration, compliance |

---

## Troubleshooting

### Check Current Provider
```bash
kubectl logs -n k8flex deployment/k8flex | grep "Using LLM provider"
# Should show: "Using LLM provider: OpenAI (gpt-4-turbo-preview)"
```

### Test Provider Connection
```bash
# Send test alert
curl -X POST http://k8flex.k8flex.svc.cluster.local:8080/webhook \
  -H "Authorization: Bearer YOUR-TOKEN" \
  -H "Content-Type: application/json" \
  -d @test-alert.json

# Check logs
kubectl logs -n k8flex deployment/k8flex --tail=50
```

### Common Errors

**"API key not configured"**
- Ensure API key is set in secret/configmap
- Verify secret is mounted to pod

**"Provider returned invalid category"**
- Model may need different prompt tuning
- Check model name is correct

**"Rate limit exceeded"**
- API provider rate limits hit
- Consider upgrading plan or switching provider

---

## Cost Optimization

### For High Volume (>1000 alerts/day)
1. **Use Ollama** - Free, self-hosted
2. **Use GPT-3.5-Turbo** - Cheapest commercial option

### For Low Volume (<100 alerts/day)
1. **Use GPT-4 or Claude** - Best quality
2. Cost is negligible (<$3/day)

### Hybrid Approach
- Use Ollama for categorization (fast, cheap)
- Use GPT-4/Claude for detailed analysis (quality)
- Implement in code by using different providers for different stages

---

## Examples

### Example 1: Use GPT-4 for Production
```yaml
config:
  llm:
    provider: "openai"
  openai:
    apiKey: ""  # Set in secret
    model: "gpt-4-turbo-preview"
```

### Example 2: Use Claude for Deep Analysis
```yaml
config:
  llm:
    provider: "anthropic"
  anthropic:
    apiKey: ""  # Set in secret
    model: "claude-3-5-sonnet-20241022"
```

### Example 3: Use Ollama for Cost Savings
```yaml
config:
  llm:
    provider: "ollama"
  ollama:
    url: "http://ollama.ollama.svc.cluster.local:11434"
    model: "llama3"
```
