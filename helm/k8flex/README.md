# K8flex Helm Chart

This Helm chart deploys the K8flex AI-powered Kubernetes debugging agent.

## Prerequisites

- Kubernetes 1.19+
- Helm 3.0+
- Ollama deployed in your cluster
- Alertmanager configured

## Installation

### Quick Install

```bash
helm install k8flex ./helm/k8flex --namespace k8flex --create-namespace
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

| Parameter | Description | Default |
|-----------|-------------|---------|
| `replicaCount` | Number of replicas | `1` |
| `image.repository` | Image repository | `k8flex-agent` |
| `image.tag` | Image tag | `latest` |
| `image.pullPolicy` | Image pull policy | `IfNotPresent` |
| `service.type` | Service type | `ClusterIP` |
| `service.port` | Service port | `8080` |
| `config.ollama.url` | Ollama API URL | `http://ollama.ollama.svc.cluster.local:11434` |
| `config.ollama.model` | Ollama model to use | `llama3` |
| `slack.botToken` | Slack Bot token for threading | `""` |
| `slack.channelId` | Slack channel ID | `""` |
| `slack.webhookUrl` | Slack webhook URL (alternative) | `""` |
| `resources.limits.cpu` | CPU limit | `500m` |
| `resources.limits.memory` | Memory limit | `512Mi` |
| `resources.requests.cpu` | CPU request | `100m` |
| `resources.requests.memory` | Memory request | `128Mi` |
| `rbac.create` | Create RBAC resources | `true` |
| `serviceAccount.create` | Create service account | `true` |

## Examples

### Development Environment

```yaml
# dev-values.yaml
image:
  pullPolicy: Always
  tag: "dev"

resources:
  limits:
    cpu: 200m
    memory: 256Mi
  requests:
    cpu: 50m
    memory: 64Mi

config:
  ollama:
    url: "http://ollama.development.svc.cluster.local:11434"
    model: "mistral"  # Smaller, faster model
```

Deploy:
```bash
helm install k8flex ./helm/k8flex -f dev-values.yaml -n k8flex --create-namespace
```

### Production Environment

```yaml
# prod-values.yaml
replicaCount: 3

image:
  tag: "1.0.0"

resources:
  limits:
    cpu: 1000m
    memory: 1Gi
  requests:
    cpu: 500m
    memory: 512Mi

config:
  ollama:
    url: "http://ollama.production.svc.cluster.local:11434"
    model: "llama2:70b"  # Larger, more accurate model

slack:
  botToken: "xoxb-production-token"
  channelId: "C12345678"

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

Deploy:
```bash
helm install k8flex ./helm/k8flex -f prod-values.yaml -n k8flex --create-namespace
```

## Upgrading

### Update Configuration

```bash
helm upgrade k8flex ./helm/k8flex \
  --namespace k8flex \
  --set config.ollama.model="llama3"
```

### Update Slack Credentials

```bash
helm upgrade k8flex ./helm/k8flex \
  --namespace k8flex \
  --set slack.botToken="xoxb-NEW-TOKEN" \
  --set slack.channelId="C98765432"
```

### Upgrade with New Image

```bash
helm upgrade k8flex ./helm/k8flex \
  --namespace k8flex \
  --set image.tag="1.1.0"
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

## Integration with Alertmanager

After installing, configure Alertmanager to send webhooks to K8flex:

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
