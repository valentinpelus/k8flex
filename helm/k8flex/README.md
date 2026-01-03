# K8flex Helm Chart

This Helm chart deploys the K8flex AI-powered Kubernetes debugging agent with support for multiple alerting systems.

## Features

- 🤖 **AI-Powered Analysis**: Multi-LLM support (Ollama, OpenAI, Claude, Gemini, Bedrock)
- 🔔 **Multi-Provider Webhooks**: Alertmanager, PagerDuty, Grafana, Datadog, Opsgenie, VictorOps, New Relic
- 💬 **Slack Integration**: Real-time streaming with threading support
- 📚 **Knowledge Base**: Optional vector database for similar case retrieval
- 👍 **Feedback System**: Learn from user ratings
- 🔒 **Production Ready**: RBAC, secrets management, resource limits

## Prerequisites

- Kubernetes 1.19+
- Helm 3.0+
- At least one LLM provider configured (Ollama, OpenAI, Claude, Gemini, or AWS Bedrock)
- At least one alerting system (Alertmanager, PagerDuty, Grafana, etc.)

## Installation

### Quick Install

```bash
helm install k8flex ./helm/k8flex --namespace k8flex --create-namespace
```

### Install with Specific Alerting Systems

```bash
# Only Alertmanager
helm install k8flex ./helm/k8flex \
  --namespace k8flex \
  --create-namespace \
  --set adapters.enabled="alertmanager"

# Multiple systems
helm install k8flex ./helm/k8flex \
  --namespace k8flex \
  --create-namespace \
  --set adapters.enabled="alertmanager,pagerduty,grafana"
```

### Install with Slack Bot Token (Threading Support)

```bash
helm install k8flex ./helm/k8flex \
  --namespace k8flex \
  --create-namespace \
  --set slack.botToken="xoxb-YOUR-BOT-TOKEN" \
  --set slack.channelId="C01234567890"
```

### Install with Slack Webhook (No Threading)

```bash
helm install k8flex ./helm/k8flex \
  --namespace k8flex \
  --create-namespace \
  --set slack.webhookUrl="https://hooks.slack.com/services/..."
```

### Install with Custom Values

```bash
helm install k8flex ./helm/k8flex \
  --namespace k8flex \
  --create-namespace \
  --values custom-values.yaml
```

## Configuration

The following table lists the configurable parameters of the K8flex chart and their default values.

### Core Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `replicaCount` | Number of replicas | `1` |
| `image.repository` | Image repository | `k8flex-agent` |
| `image.tag` | Image tag | `latest` |
| `image.pullPolicy` | Image pull policy | `IfNotPresent` |
| `service.type` | Service type | `ClusterIP` |
| `service.port` | Service port | `8080` |

### LLM Provider Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `config.llm.provider` | LLM provider (ollama, openai, anthropic, gemini, bedrock) | `ollama` |
| `config.ollama.url` | Ollama API URL | `http://ollama.ollama.svc.cluster.local:11434` |
| `config.ollama.model` | Ollama model to use | `llama3` |
| `config.openai.apiKey` | OpenAI API key (set in secrets.yaml) | `""` |
| `config.openai.model` | OpenAI model | `gpt-4-turbo-preview` |
| `config.anthropic.apiKey` | Anthropic API key (set in secrets.yaml) | `""` |
| `config.anthropic.model` | Claude model | `claude-3-5-sonnet-20241022` |
| `config.gemini.apiKey` | Google Gemini API key (set in secrets.yaml) | `""` |
| `config.gemini.model` | Gemini model | `gemini-1.5-pro` |
| `config.bedrock.region` | AWS Bedrock region | `us-east-1` |
| `config.bedrock.model` | Bedrock model ID | `anthropic.claude-3-5-sonnet-20241022-v2:0` |

### Alerting System Adapters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `adapters.enabled` | Comma-separated list of enabled adapters or empty for all | `""` (all enabled) |

**Available adapters:**
- `alertmanager` - Prometheus Alertmanager
- `pagerduty` - PagerDuty incidents  
- `grafana` - Grafana Alerting
- `datadog` - Datadog Monitors
- `opsgenie` - Opsgenie alerts
- `victorops` - VictorOps (Splunk On-Call)
- `newrelic` - New Relic Alerts

**Examples:**
- `adapters.enabled: "alertmanager"` - Only Alertmanager
- `adapters.enabled: "alertmanager,pagerduty,grafana"` - Multiple systems
- `adapters.enabled: ""` - All adapters (default)

### Slack Integration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `slack.botToken` | Slack Bot token for threading (set in secrets.yaml) | `""` |
| `slack.channelId` | Slack channel ID | `""` |
| `slack.workspaceId` | Slack workspace ID for links | `""` |
| `slack.webhookUrl` | Slack webhook URL (alternative, no threading) | `""` |

### Knowledge Base

| Parameter | Description | Default |
|-----------|-------------|---------|
| `knowledgeBase.enabled` | Enable knowledge base | `false` |
| `knowledgeBase.databaseUrl` | PostgreSQL connection URL (set in secrets.yaml) | `""` |
| `knowledgeBase.embeddingProvider` | Embedding provider (openai, gemini) | `openai` |
| `knowledgeBase.embeddingModel` | Embedding model | `text-embedding-3-small` |
| `knowledgeBase.similarityThreshold` | Similarity threshold (0.0-1.0) | `0.75` |
| `knowledgeBase.maxResults` | Max similar cases to retrieve | `5` |

### Resources

| Parameter | Description | Default |
|-----------|-------------|---------|
| `resources.limits.cpu` | CPU limit | `500m` |
| `resources.limits.memory` | Memory limit | `512Mi` |
| `resources.requests.cpu` | CPU request | `100m` |
| `resources.requests.memory` | Memory request | `128Mi` |

### RBAC & Security

| Parameter | Description | Default |
|-----------|-------------|---------|
| `rbac.create` | Create RBAC resources | `true` |
| `serviceAccount.create` | Create service account | `true` |
| `webhook.authToken` | Webhook authentication token (set in secrets.yaml) | `""` |

## Examples

### Single Alerting System (Alertmanager Only)

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

slack:
  botToken: "xoxb-YOUR-BOT-TOKEN"
  channelId: "C01234567890"
```

Deploy:
```bash
helm install k8flex ./helm/k8flex -f alertmanager-only.yaml -n k8flex --create-namespace
```

### Multi-Provider Setup

```yaml
# multi-provider.yaml
adapters:
  # Enable Alertmanager for infrastructure, PagerDuty for on-call, Grafana for apps
  enabled: "alertmanager,pagerduty,grafana"

config:
  llm:
    provider: "openai"
  openai:
    model: "gpt-4-turbo-preview"
    # apiKey set in secrets.yaml

slack:
  botToken: "xoxb-YOUR-BOT-TOKEN"
  channelId: "C01234567890"
  workspaceId: "T01234567"
```

Deploy with secrets:
```bash
# First, create secrets.yaml with API keys
helm install k8flex ./helm/k8flex \
  -f multi-provider.yaml \
  -f secrets.yaml \
  -n k8flex --create-namespace
```

### Enterprise Setup (All Systems)

```yaml
# enterprise.yaml
replicaCount: 3

adapters:
  # Enable all alerting systems (default)
  enabled: ""

config:
  llm:
    provider: "anthropic"
  anthropic:
    model: "claude-3-5-sonnet-20241022"
    # apiKey set in secrets.yaml

knowledgeBase:
  enabled: true
  embeddingProvider: "openai"
  embeddingModel: "text-embedding-3-small"
  # databaseUrl set in secrets.yaml

resources:
  limits:
    cpu: 1000m
    memory: 1Gi
  requests:
    cpu: 500m
    memory: 512Mi

affinity:
  podAntiAffinity:
    preferredDuringSchedulingIgnoredDuringExecution:
      - weight: 100
        podAffinityTerm:
          labelSelector:
            matchExpressions:
              - key: app.kubernetes.io/name
                operator: In
                values:
                  - k8flex
          topologyKey: kubernetes.io/hostname
```

## Upgrading

### Enable Additional Alerting Systems

```bash
# Add PagerDuty and Grafana to existing Alertmanager setup
helm upgrade k8flex ./helm/k8flex \
  --namespace k8flex \
  --set adapters.enabled="alertmanager,pagerduty,grafana"
```

### Switch to All Adapters

```bash
# Enable all alerting systems
helm upgrade k8flex ./helm/k8flex \
  --namespace k8flex \
  --set adapters.enabled=""
```

### Update LLM Provider

```bash
# Switch from Ollama to OpenAI
helm upgrade k8flex ./helm/k8flex \
  --namespace k8flex \
  --set config.llm.provider="openai" \
  --set config.openai.apiKey="sk-..."
```

## Uninstallation

```bash
helm uninstall k8flex --namespace k8flex
```

To also delete the namespace:
```bash
kubectl delete namespace k8flex
```

## Verifying the Installation

```bash
# Check deployment status
helm status k8flex -n k8flex

# View pods
kubectl get pods -n k8flex

# View logs
kubectl logs -n k8flex deployment/k8flex-k8flex-agent -f

# Test health endpoint
kubectl exec -n k8flex deployment/k8flex-k8flex-agent -- wget -qO- http://localhost:8080/health
```

## Integration with Alerting Systems

After installing, configure your alerting systems to send webhooks to K8flex.

### Alertmanager (Prometheus)

```yaml
# alertmanager.yaml
receivers:
  - name: 'k8flex-ai-debug'
    webhook_configs:
      - url: 'http://k8flex-k8flex.k8flex.svc.cluster.local:8080/webhook'
        send_resolved: false

route:
  routes:
    - match:
        severity: critical
      receiver: 'k8flex-ai-debug'
      continue: true
```

### PagerDuty

1. Go to **Services** > Select service > **Integrations**
2. Add **Generic Webhooks (v3)**
3. URL: `http://k8flex-k8flex.k8flex.svc.cluster.local:8080/webhook`
4. Enable **Trigger** events

### Grafana

1. Go to **Alerting** > **Contact points**
2. Add **webhook** type
3. URL: `http://k8flex-k8flex.k8flex.svc.cluster.local:8080/webhook`

### Datadog

1. **Integrations** > **Webhooks**
2. Name: `k8flex-debug`
3. URL: `http://k8flex-k8flex.k8flex.svc.cluster.local:8080/webhook`
4. Add `@webhook-k8flex-debug` to monitor notifications

### Opsgenie

1. **Settings** > **Integrations** > **Outgoing Webhooks**
2. URL: `http://k8flex-k8flex.k8flex.svc.cluster.local:8080/webhook`
3. Trigger: **Alert Created**

### VictorOps (Splunk On-Call)

1. **Settings** > **Outgoing Webhooks**
2. Event: **Incident Triggered**
3. URL: `http://k8flex-k8flex.k8flex.svc.cluster.local:8080/webhook`

### New Relic

1. **Alerts & AI** > **Notification channels**
2. Type: **Webhook**
3. URL: `http://k8flex-k8flex.k8flex.svc.cluster.local:8080/webhook`

See [../../docs/INTEGRATION.md](../../docs/INTEGRATION.md) for detailed setup instructions.

## Troubleshooting

### Check Logs

```bash
kubectl logs -n k8flex -l app.kubernetes.io/name=k8flex
```

### Verify Configuration

```bash
kubectl get configmap -n k8flex k8flex-k8flex-config -o yaml
```

### Check RBAC

```bash
kubectl auth can-i get pods --as=system:serviceaccount:k8flex:k8flex-agent --all-namespaces
```

### Test Slack Integration

```bash
# Check if Slack is configured
kubectl get secret -n k8flex k8flex-k8flex-secrets -o yaml
```

## Advanced Configuration

### Using External Secrets Operator

```yaml
# values.yaml
slack:
  # Don't set credentials in values
  botToken: ""
  channelId: ""

# Create ExternalSecret separately
---
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: k8flex-slack-credentials
  namespace: k8flex
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: vault-backend
    kind: SecretStore
  target:
    name: k8flex-k8flex-secrets
  data:
    - secretKey: SLACK_BOT_TOKEN
      remoteRef:
        key: slack/k8flex
        property: bot_token
```

### Custom Resource Limits by Node

```yaml
# Use node selector and taints
nodeSelector:
  workload-type: ai-debugging

tolerations:
  - key: "ai-workload"
    operator: "Equal"
    value: "true"
    effect: "NoSchedule"
```

## Support

For issues and questions:
- GitHub Issues: https://github.com/yourorg/k8flex/issues
- Documentation: See README.md in project root
