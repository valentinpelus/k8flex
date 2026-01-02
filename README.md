# K8flex - AI-Powered Kubernetes Debug Agent

An intelligent debugging agent that receives Alertmanager webhooks and performs automated first-level analysis of Kubernetes issues using AI.

## Features

- **Webhook Receiver**: Accepts Alertmanager webhook notifications
- **Automated Debugging**: Performs comprehensive Kubernetes resource analysis:
  - Pod log collection (last 100 lines)
  - Pod description and status
  - Recent namespace events
  - Service and endpoint checks
  - Network policy analysis
  - Resource usage and limits
- **AI Analysis**: Uses Ollama to synthesize findings and deduce root causes
- **Slack Integration**: Posts alerts and AI analysis to Slack channels with threading
- **Label-Driven**: Extracts all parameters from alert labels (no fake data)

## Architecture

```
Alertmanager → K8flex Agent → Kubernetes API
                    ↓
                Ollama API
                    ↓
         AI Analysis → Slack (optional)
                    ↓
                  Logs
```

## Prerequisites

- Kubernetes cluster with:
  - Prometheus & Alertmanager installed
  - Ollama deployed (see deployment guide below)
- kubectl configured
- Docker (for building image)

## Quick Start

### 1. Build and Deploy

```bash
# Build the Docker image
docker build -t k8flex-agent:latest .

# For Kind/Minikube, load the image
kind load docker-image k8flex-agent:latest
# or
minikube image load k8flex-agent:latest

# Deploy to Kubernetes
kubectl apply -f k8s/deployment.yaml
```

### 2. Configure Alertmanager

Add the k8flex webhook receiver to your Alertmanager configuration:

```yaml
receivers:
  - name: 'k8flex-ai-debug'
    webhook_configs:
      - url: 'http://k8flex-agent.k8flex.svc.cluster.local:8080/webhook'
        send_resolved: false

route:
  routes:
    - match:
        severity: critical
      receiver: 'k8flex-ai-debug'
      continue: true  # Also send to other receivers
```

Or apply the example configuration:

```bash
kubectl apply -f k8s/alertmanager-config.yaml
```

### 3. Deploy Ollama (if not already installed)

```bash
# Example Ollama deployment
kubectl create namespace ollama

kubectl apply -f - <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ollama
  namespace: ollama
spec:
  replicas: 1
  selector:
    matchLabels:
      app: ollama
  template:
    metadata:
      labels:
        app: ollama
    spec:
      containers:
      - name: ollama
        image: ollama/ollama:latest
        ports:
        - containerPort: 11434
        resources:
          requests:
            memory: "4Gi"
            cpu: "2"
          limits:
            memory: "8Gi"
            cpu: "4"
---
apiVersion: v1
kind: Service
metadata:
  name: ollama
  namespace: ollama
spec:
  selector:
    app: ollama
  ports:
  - port: 11434
    targetPort: 11434
EOF

# Pull the model
kubectl exec -n ollama deployment/ollama -- ollama pull llama2
```

### 4. Test with a Sample Alert

```bash
kubectl run test-pod --image=nginx --restart=Never
kubectl delete pod test-pod

# Or send a webhook directly
curl -XPOST 'http://k8flex-agent.k8flex.svc.cluster.local:8080/webhook' \
  -H 'Content-Type: application/json' \
  -d '{
    "version": "4",
    "groupKey": "test",
    "status": "firing",
    "alerts": [{
      "status": "firing",
      "labels": {
        "alertname": "PodCrashLooping",
        "severity": "critical",
        "namespace": "default",
        "pod": "test-pod"
      },
      "annotations": {
        "summary": "Pod is crash looping",
        "description": "The pod has restarted multiple times"
      },
      "startsAt": "2026-01-02T10:00:00Z"
    }]
  }'
```

## Configuration

Environment variables (configured in ConfigMap):

| Variable | Default | Description |
|----------|---------|-------------|
| `OLLAMA_URL` | `http://ollama.ollama.svc.cluster.local:11434` | Ollama API endpoint |
| `OLLAMA_MODEL` | `llama2` | Ollama model to use |
| `PORT` | `8080` | HTTP server port |

## Alert Label Requirements

The agent extracts debugging parameters from alert labels:

- `namespace` (required): Kubernetes namespace
- `pod` (optional): Pod name to debug
- `service` (optional): Service name to check
- `alertname`: Alert identifier
- `severity`: Alert severity

Example alert with proper labels:

```yaml
- alert: PodNotReady
  expr: kube_pod_status_phase{phase!="Running"} == 1
  labels:
    severity: warning
    namespace: "{{ $labels.namespace }}"
    pod: "{{ $labels.pod }}"
  annotations:
    summary: "Pod {{ $labels.pod }} is not ready"
```

## Debugging Functions

The agent performs these checks automatically:

1. **Pod Logs**: Last 100 lines from the pod
2. **Pod Description**: Status, conditions, container states
3. **Events**: Recent events in the namespace
4. **Service Check**: Service configuration and endpoints
5. **Network Check**: Pod IPs and network policies
6. **Resource Check**: CPU/memory requests, limits, and probes

## AI Analysis

The agent sends all collected data to Ollama with a structured prompt requesting:

1. **Root Cause Analysis**: Most likely cause
2. **Evidence**: Supporting data
3. **Impact Assessment**: What's affected
4. **Recommended Actions**: Immediate steps
5. **Prevention**: Future recommendations

## Monitoring

Check agent logs:

```bash
kubectl logs -n k8flex deployment/k8flex-agent -f
```

Health check:

```bash
kubectl exec -n k8flex deployment/k8flex-agent -- wget -qO- http://localhost:8080/health
```

## Development

### Local Testing

```bash
# Run locally (requires kubeconfig)
go run main.go

# Test webhook
curl -XPOST 'http://localhost:8080/webhook' \
  -H 'Content-Type: application/json' \
  -d @test-alert.json
```

### Building

```bash
# Download dependencies
go mod download

# Build
go build -o k8flex-agent .
```

## Security Considerations

- The agent requires read-only access to cluster resources (RBAC configured)
- Does not access Secret data, only metadata
- Runs with minimal privileges using ServiceAccount
- All communication within cluster uses internal DNS

## References

- [Alertmanager Webhook Config](https://prometheus.io/docs/alerting/latest/configuration/#webhook_config)
- [Kubernetes client-go](https://pkg.go.dev/k8s.io/client-go)
- [Ollama API Documentation](https://github.com/ollama/ollama/blob/main/docs/api.md)
- [Kubernetes RBAC](https://kubernetes.io/docs/reference/access-authn-authz/rbac/)
- [Slack Incoming Webhooks](https://api.slack.com/messaging/webhooks)

## Documentation

- **[README.md](README.md)** - This file, main documentation
- **[QUICKSTART.md](QUICKSTART.md)** - Quick reference guide
- **[INTEGRATION.md](INTEGRATION.md)** - Alertmanager and Prometheus setup
- **[SLACK_SETUP.md](SLACK_SETUP.md)** - Slack integration guide
- **[SLACK_INTEGRATION.md](SLACK_INTEGRATION.md)** - Slack feature summary

## Scripts

- **[setup-slack.sh](setup-slack.sh)** - Configure Slack webhook URL
- **[test-integration.sh](test-integration.sh)** - Test complete setup
- **[deploy.sh](deploy.sh)** - Build and deploy script

## License

MIT

## Contributing

Contributions welcome! Please ensure all parameters are extracted from alert labels and avoid hardcoded values.
