# K8flex Helm Chart Configuration Examples

This document provides comprehensive configuration examples for various deployment scenarios.

## Table of Contents

- [Basic Configurations](#basic-configurations)
- [Alerting System Configurations](#alerting-system-configurations)
- [LLM Provider Configurations](#llm-provider-configurations)
- [Production Configurations](#production-configurations)
- [Security & Secrets Management](#security--secrets-management)

## Basic Configurations

### Minimal Installation (Alertmanager + Ollama)

```yaml
# minimal.yaml
adapters:
  enabled: "alertmanager"

config:
  llm:
    provider: "ollama"
  ollama:
    url: "http://ollama.ollama.svc.cluster.local:11434"
    model: "llama3"

slack:
  botToken: "xoxb-YOUR-TOKEN"
  channelId: "C01234567890"
```

Install:
```bash
helm install k8flex ./helm/k8flex -f minimal.yaml -n k8flex --create-namespace
```

### All Features Enabled

```yaml
# full-featured.yaml
replicaCount: 2

adapters:
  enabled: ""  # All adapters

config:
  llm:
    provider: "openai"
  openai:
    model: "gpt-4-turbo-preview"
    # apiKey set in secrets.yaml

knowledgeBase:
  enabled: true
  embeddingProvider: "openai"
  embeddingModel: "text-embedding-3-small"
  # databaseUrl set in secrets.yaml

slack:
  botToken: "xoxb-YOUR-TOKEN"
  channelId: "C01234567890"
  workspaceId: "T01234567"

persistence:
  enabled: true
  size: 5Gi

resources:
  limits:
    cpu: 1000m
    memory: 1Gi
  requests:
    cpu: 500m
    memory: 512Mi
```

## Alerting System Configurations

### Alertmanager Only

```yaml
# alertmanager-only.yaml
adapters:
  enabled: "alertmanager"

config:
  llm:
    provider: "ollama"
  ollama:
    url: "http://ollama.ollama.svc.cluster.local:11434"
    model: "llama3"
```

Use case: Pure Prometheus/Alertmanager infrastructure

### PagerDuty + Alertmanager

```yaml
# pagerduty-alertmanager.yaml
adapters:
  enabled: "alertmanager,pagerduty"

config:
  llm:
    provider: "openai"
  openai:
    model: "gpt-4-turbo-preview"
```

Use case: On-call rotation (PagerDuty) + infrastructure monitoring (Alertmanager)

### Multi-Cloud Setup

```yaml
# multi-cloud.yaml
adapters:
  # On-prem: Alertmanager
  # AWS: Datadog
  # GCP: Grafana
  enabled: "alertmanager,datadog,grafana"

config:
  llm:
    provider: "bedrock"
  bedrock:
    region: "us-east-1"
    model: "anthropic.claude-3-5-sonnet-20241022-v2:0"

serviceAccount:
  annotations:
    eks.amazonaws.com/role-arn: arn:aws:iam::123456789012:role/k8flex-bedrock-role
```

### Enterprise Stack (All Systems)

```yaml
# enterprise.yaml
replicaCount: 3

adapters:
  enabled: ""  # All: alertmanager, pagerduty, grafana, datadog, opsgenie, victorops, newrelic

config:
  llm:
    provider: "anthropic"
  anthropic:
    model: "claude-3-5-sonnet-20241022"

knowledgeBase:
  enabled: true
  embeddingProvider: "openai"

resources:
  limits:
    cpu: 2000m
    memory: 2Gi
  requests:
    cpu: 1000m
    memory: 1Gi

affinity:
  podAntiAffinity:
    requiredDuringSchedulingIgnoredDuringExecution:
      - labelSelector:
          matchExpressions:
            - key: app.kubernetes.io/name
              operator: In
              values:
                - k8flex
        topologyKey: kubernetes.io/hostname
```

## LLM Provider Configurations

### Self-Hosted Ollama

```yaml
# ollama.yaml
config:
  llm:
    provider: "ollama"
  ollama:
    url: "http://ollama.ollama.svc.cluster.local:11434"
    model: "llama3"  # or mistral, llama2:70b, etc.

resources:
  limits:
    cpu: 500m
    memory: 512Mi
```

Benefits:
- No external API costs
- Data privacy (everything on-prem)
- Offline capable

### OpenAI GPT-4

```yaml
# openai.yaml
config:
  llm:
    provider: "openai"
  openai:
    model: "gpt-4-turbo-preview"
    # apiKey in secrets.yaml

resources:
  limits:
    cpu: 200m  # Lower since no local inference
    memory: 256Mi
```

Benefits:
- Latest GPT-4 capabilities
- Fast response times
- No GPU required

### Anthropic Claude

```yaml
# claude.yaml
config:
  llm:
    provider: "anthropic"
  anthropic:
    model: "claude-3-5-sonnet-20241022"
    # apiKey in secrets.yaml

knowledgeBase:
  enabled: true
  embeddingProvider: "openai"  # Claude doesn't have embeddings API

resources:
  limits:
    cpu: 200m
    memory: 256Mi
```

Benefits:
- Excellent reasoning
- Large context window (200k tokens)
- Good at technical analysis

### AWS Bedrock

```yaml
# bedrock.yaml
config:
  llm:
    provider: "bedrock"
  bedrock:
    region: "us-east-1"
    model: "anthropic.claude-3-5-sonnet-20241022-v2:0"

serviceAccount:
  create: true
  annotations:
    eks.amazonaws.com/role-arn: arn:aws:iam::123456789012:role/k8flex-bedrock-role

resources:
  limits:
    cpu: 200m
    memory: 256Mi
```

Benefits:
- AWS native integration
- IAM-based security
- No API key management
- Multiple model options

### Google Gemini

```yaml
# gemini.yaml
config:
  llm:
    provider: "gemini"
  gemini:
    model: "gemini-1.5-pro"
    # apiKey in secrets.yaml

knowledgeBase:
  enabled: true
  embeddingProvider: "gemini"
  embeddingModel: "embedding-001"

resources:
  limits:
    cpu: 200m
    memory: 256Mi
```

Benefits:
- Long context (1M+ tokens)
- Built-in embeddings
- Cost-effective

## Production Configurations

### High Availability

```yaml
# ha.yaml
replicaCount: 3

adapters:
  enabled: "alertmanager,pagerduty,grafana"

config:
  llm:
    provider: "openai"
  openai:
    model: "gpt-4-turbo-preview"

resources:
  limits:
    cpu: 1000m
    memory: 1Gi
  requests:
    cpu: 500m
    memory: 512Mi

affinity:
  podAntiAffinity:
    requiredDuringSchedulingIgnoredDuringExecution:
      - labelSelector:
          matchExpressions:
            - key: app.kubernetes.io/name
              operator: In
              values:
                - k8flex
        topologyKey: kubernetes.io/hostname

topologySpreadConstraints:
  - maxSkew: 1
    topologyKey: topology.kubernetes.io/zone
    whenUnsatisfiable: DoNotSchedule
    labelSelector:
      matchLabels:
        app.kubernetes.io/name: k8flex
```

### Performance Optimized

```yaml
# performance.yaml
replicaCount: 2

adapters:
  enabled: "alertmanager,grafana"

config:
  llm:
    provider: "openai"
  openai:
    model: "gpt-3.5-turbo"  # Faster, cheaper

resources:
  limits:
    cpu: 500m
    memory: 512Mi
  requests:
    cpu: 250m
    memory: 256Mi

# No knowledge base for faster responses
knowledgeBase:
  enabled: false

persistence:
  enabled: false  # Ephemeral feedback storage
```

### Cost Optimized

```yaml
# cost-optimized.yaml
replicaCount: 1

adapters:
  enabled: "alertmanager"

config:
  llm:
    provider: "ollama"
  ollama:
    url: "http://ollama.ollama.svc.cluster.local:11434"
    model: "mistral"  # Smaller, faster model

resources:
  limits:
    cpu: 200m
    memory: 256Mi
  requests:
    cpu: 100m
    memory: 128Mi

knowledgeBase:
  enabled: false

persistence:
  enabled: false
```

## Security & Secrets Management

### Using SOPS (Recommended)

Create `secrets.yaml`:

```yaml
# secrets.yaml
config:
  openai:
    apiKey: "sk-..."
  anthropic:
    apiKey: "sk-ant-..."
  gemini:
    apiKey: "..."

knowledgeBase:
  databaseUrl: "postgresql://user:pass@host:5432/db?sslmode=require"

slack:
  botToken: "xoxb-..."
  
webhook:
  authToken: "your-webhook-auth-token"
```

Encrypt:
```bash
sops --encrypt --in-place secrets.yaml
```

Install:
```bash
helm secrets install k8flex ./helm/k8flex \
  -f values.yaml \
  -f secrets.yaml \
  -n k8flex --create-namespace
```

### Using External Secrets Operator

```yaml
# values.yaml - no secrets
config:
  llm:
    provider: "openai"
  openai:
    apiKey: ""  # Will be injected

slack:
  botToken: ""  # Will be injected
```

```yaml
# external-secret.yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: k8flex-api-keys
  namespace: k8flex
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: vault-backend
    kind: SecretStore
  target:
    name: k8flex-k8flex-secrets
    creationPolicy: Owner
  data:
    - secretKey: OPENAI_API_KEY
      remoteRef:
        key: k8flex/openai
        property: api_key
    - secretKey: SLACK_BOT_TOKEN
      remoteRef:
        key: k8flex/slack
        property: bot_token
```

### Using Sealed Secrets

```bash
# Create secret
kubectl create secret generic k8flex-secrets \
  --from-literal=OPENAI_API_KEY=sk-... \
  --from-literal=SLACK_BOT_TOKEN=xoxb-... \
  --dry-run=client -o yaml | \
  kubeseal -o yaml > sealed-secrets.yaml

# Apply sealed secret
kubectl apply -f sealed-secrets.yaml
```

## Testing Configurations

### Validate Configuration

```bash
# Dry run
helm install k8flex ./helm/k8flex \
  -f your-values.yaml \
  -n k8flex \
  --dry-run --debug
```

### Template Rendering

```bash
# See what will be deployed
helm template k8flex ./helm/k8flex \
  -f your-values.yaml \
  -n k8flex
```

### Upgrade with Diff

```bash
# See what will change
helm diff upgrade k8flex ./helm/k8flex \
  -f your-values.yaml \
  -n k8flex
```

## Common Patterns

### Dev/Staging/Prod Separation

```
helm/k8flex/
├── values.yaml           # Base values
├── values-dev.yaml       # Dev overrides
├── values-staging.yaml   # Staging overrides
└── values-prod.yaml      # Prod overrides
```

Install per environment:
```bash
# Dev
helm install k8flex ./helm/k8flex \
  -f values.yaml \
  -f values-dev.yaml \
  -n k8flex-dev

# Prod
helm install k8flex ./helm/k8flex \
  -f values.yaml \
  -f values-prod.yaml \
  -f secrets-prod.yaml \
  -n k8flex-prod
```

### GitOps with ArgoCD/Flux

```yaml
# argocd-application.yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: k8flex
  namespace: argocd
spec:
  project: default
  source:
    repoURL: https://github.com/yourorg/k8flex
    targetRevision: main
    path: helm/k8flex
    helm:
      valueFiles:
        - values.yaml
        - values-prod.yaml
  destination:
    server: https://kubernetes.default.svc
    namespace: k8flex
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
```

## Custom Analysis Prompts

### Security-Focused Prompt

```yaml
# security-prompt.yaml
config:
  llm:
    provider: "anthropic"
  anthropic:
    model: "claude-3-5-sonnet-20241022"
  
  analysisPrompt: |
    SECURITY INCIDENT ANALYSIS - Kubernetes Cluster
    {FEEDBACK_CONTEXT}
    
    Analysis Framework:
    1. Identify security vulnerabilities and attack vectors
    2. Assess compliance and regulatory impact
    3. Determine data exposure risk
    4. Check for privilege escalation
    5. Review network security implications
    
    Debug Information:
    {DEBUG_INFO}
    {FEEDBACK_INSTRUCTION}
    
    Required Output Format:
    *Security Risk:* [CRITICAL/HIGH/MEDIUM/LOW]
    *Attack Vector:* [Description with evidence]
    *Data Exposure:* [What data is at risk]
    *Compliance Impact:* [SOC2/HIPAA/PCI-DSS implications]
    *Immediate Actions:* 
    • [Security containment step 1]
    • [Security containment step 2]
    *Security Hardening:*
    • [Prevention measure 1]
    • [Prevention measure 2]

adapters:
  enabled: "alertmanager,datadog"

slack:
  botToken: "xoxb-YOUR-TOKEN"
  channelId: "C-SECURITY-OPS"
```

### Brief & Actionable Prompt

```yaml
# brief-prompt.yaml
config:
  llm:
    provider: "openai"
  openai:
    model: "gpt-4-turbo-preview"
  
  analysisPrompt: |
    Quick Incident Analysis
    {FEEDBACK_CONTEXT}
    Debug: {DEBUG_INFO}
    {FEEDBACK_INSTRUCTION}
    
    Format (be concise):
    *Problem:* [1-2 sentences]
    *Fix:* [3 bullet points max]
    *Prevention:* [1 recommendation]

adapters:
  enabled: "grafana"
```

### Custom Domain Expert Prompt

```yaml
# fintech-prompt.yaml
config:
  llm:
    provider: "anthropic"
  anthropic:
    model: "claude-3-5-sonnet-20241022"
  
  analysisPrompt: |
    Financial Services Kubernetes Incident Analysis
    {FEEDBACK_CONTEXT}
    
    Context: This is a payment processing platform with strict SLA requirements.
    
    Critical Evaluation Points:
    1. Transaction processing impact (quote actual error rates from metrics)
    2. Database consistency implications
    3. Regulatory compliance (PCI-DSS, SOX)
    4. Customer-facing service availability
    5. Financial data integrity
    
    Debug Information:
    {DEBUG_INFO}
    {FEEDBACK_INSTRUCTION}
    
    Response Template:
    *Business Impact:* [Revenue/SLA/Customer impact with numbers]
    *Root Cause:* [Technical cause with evidence]
    *Transaction Safety:* [Are in-flight transactions safe?]
    *Recovery Steps:*
    • [Step 1 - most critical]
    • [Step 2]
    • [Step 3]
    *Compliance Report:* [What to report to compliance team]
    *Prevention:* [Technical + process improvements]
    
    Tone: Professional, precise, cite specific metrics.

adapters:
  enabled: "pagerduty,datadog,opsgenie"

slack:
  botToken: "xoxb-YOUR-TOKEN"
  channelId: "C-PAYMENTS-OPS"
  workspaceId: "T01234567"

knowledgeBase:
  enabled: true
  embeddingProvider: "openai"
```

### Multi-Language Prompt

```yaml
# french-prompt.yaml
config:
  llm:
    provider: "gemini"
  gemini:
    model: "gemini-1.5-pro"
  
  analysisPrompt: |
    Analyse d'Incident Kubernetes
    {FEEDBACK_CONTEXT}
    
    Instructions:
    1. Identifier la cause racine avec des preuves
    2. Évaluer l'impact métier
    3. Fournir des étapes de remédiation
    4. Suggérer des mesures préventives
    
    Informations de debug:
    {DEBUG_INFO}
    {FEEDBACK_INSTRUCTION}
    
    Format de réponse:
    *Cause Racine:* ...
    *Impact:* ...
    *Actions Correctives:* ...
    *Prévention:* ...

adapters:
  enabled: "alertmanager"
```

### Minimal Prompt (Override Default)

```yaml
# minimal-prompt.yaml
config:
  analysisPrompt: |
    {DEBUG_INFO}
    {FEEDBACK_CONTEXT}
    {FEEDBACK_INSTRUCTION}
    Analyze and fix.
```

**Note:** The minimal prompt gives the LLM maximum freedom but may produce inconsistent output. Use the default prompt or customize it with structure for best results.

## Troubleshooting

### View Current Configuration

```bash
helm get values k8flex -n k8flex
```

### Check Rendered Manifests

```bash
helm get manifest k8flex -n k8flex
```

### Rollback Configuration

```bash
helm rollback k8flex -n k8flex
```

## References

- [Main README](../README.md)
- [Adapter Configuration](../../docs/ADAPTER_CONFIGURATION.md)
- [Integration Guide](../../docs/INTEGRATION.md)
- [SOPS Usage](docs/SOPS_USAGE.md)
