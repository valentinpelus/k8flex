# K8flex Quick Reference

## Quick Start

```bash
# 1. Build and deploy
make deploy-kind  # or make deploy-minikube

# 2. Configure Alertmanager (add to your alertmanager config)
receivers:
  - name: 'k8flex-ai-debug'
    webhook_configs:
      - url: 'http://k8flex-agent.k8flex.svc.cluster.local:8080/webhook'

# 3. Deploy Ollama
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
EOF

# 4. Pull model
kubectl exec -n ollama deployment/ollama -- ollama pull llama2

# 5. (Optional) Setup Slack notifications
./setup-slack.sh 'https://hooks.slack.com/services/YOUR/WEBHOOK/URL'

# 6. Test
kubectl apply -f test-alert.json
```

## Slack Setup (Optional)

```bash
# Get Slack webhook URL from https://api.slack.com/apps
# Then run:
./setup-slack.sh 'https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXX'

# Or manually:
kubectl create secret generic k8flex-secrets \
  --from-literal=SLACK_WEBHOOK_URL='YOUR_WEBHOOK_URL' \
  -n k8flex

kubectl rollout restart deployment k8flex-agent -n k8flex
```

See [SLACK_SETUP.md](SLACK_SETUP.md) for detailed Slack integration guide.

## Common Commands

```bash
# View logs
kubectl logs -n k8flex deployment/k8flex-agent -f

# Check health
kubectl exec -n k8flex deployment/k8flex-agent -- \
  wget -qO- http://localhost:8080/health

# Test locally
go run main.go
curl -XPOST http://localhost:8080/webhook -d @test-alert.json

# Rebuild and redeploy
make clean deploy-kind

# Scale up
kubectl scale deployment k8flex-agent -n k8flex --replicas=3
```

## Alert Label Requirements

**Required:**
- `namespace`: Kubernetes namespace

**Optional but recommended:**
- `pod`: Pod name
- `service`: Service name
- `alertname`: Alert identifier
- `severity`: Alert severity

## Example Alert

```json
{
  "labels": {
    "alertname": "PodNotReady",
    "severity": "critical",
    "namespace": "production",
    "pod": "api-server-xyz",
    "service": "api-server"
  },
  "annotations": {
    "summary": "Pod is not ready",
    "description": "Failed readiness checks"
  }
}
```

## Configuration

Environment variables in ConfigMap:

```yaml
OLLAMA_URL: "http://ollama.ollama.svc.cluster.local:11434"
OLLAMA_MODEL: "llama2"  # or "mistral", "llama2:70b"
PORT: "8080"
```

## Troubleshooting

### No alerts received
```bash
# Check Alertmanager config
kubectl get configmap alertmanager -n monitoring -o yaml | grep k8flex

# Test connectivity
kubectl run test --rm -it --image=curlimages/curl -- \
  curl http://k8flex-agent.k8flex.svc.cluster.local:8080/health
```

### Can't connect to Ollama
```bash
# Verify Ollama
kubectl get pods -n ollama
kubectl logs -n ollama deployment/ollama

# Test from k8flex pod
kubectl exec -n k8flex deployment/k8flex-agent -- \
  wget -qO- http://ollama.ollama.svc.cluster.local:11434
```

### Permission errors
```bash
# Check RBAC
kubectl auth can-i get pods \
  --as=system:serviceaccount:k8flex:k8flex-agent \
  --all-namespaces
```

## Architecture

```
┌─────────────┐
│ Prometheus  │
└─────┬───────┘
      │ alerts
      ▼
┌─────────────────┐      ┌──────────────┐
│  Alertmanager   │─────▶│ K8flex Agent │
└─────────────────┘ POST └──────┬───────┘
                                 │
                    ┌────────────┼────────────┐
                    │            │            │
                    ▼            ▼            ▼
            ┌──────────┐  ┌──────────┐  ┌─────────┐
            │   K8s    │  │  Ollama  │  │  Logs   │
            │   API    │  │   API    │  │         │
            └──────────┘  └──────────┘  └─────────┘
```

## What K8flex Does

1. ✅ Receives webhook from Alertmanager
2. ✅ Extracts namespace/pod/service from labels
3. ✅ Fetches pod logs (last 100 lines)
4. ✅ Describes pod status and conditions
5. ✅ Lists recent namespace events
6. ✅ Checks service endpoints
7. ✅ Analyzes network policies
8. ✅ Reviews resource limits
9. ✅ Sends to Ollama for AI analysis
10. ✅ Logs complete analysis with root cause

## Files

- [main.go](main.go) - Main application
- [go.mod](go.mod) - Go dependencies
- [Dockerfile](Dockerfile) - Container image
- [k8s/deployment.yaml](k8s/deployment.yaml) - K8s manifests
- [k8s/alertmanager-config.yaml](k8s/alertmanager-config.yaml) - Alertmanager config
- [README.md](README.md) - Full documentation
- [INTEGRATION.md](INTEGRATION.md) - Integration guide
- [Makefile](Makefile) - Build commands
- [deploy.sh](deploy.sh) - Deployment script
- [test-alert.json](test-alert.json) - Test payload

## API References Used

All code references official documentation:

- [Alertmanager Webhook](https://prometheus.io/docs/alerting/latest/configuration/#webhook_config)
- [Kubernetes client-go](https://pkg.go.dev/k8s.io/client-go)
- [Ollama API](https://github.com/ollama/ollama/blob/main/docs/api.md)
- [Kubernetes RBAC](https://kubernetes.io/docs/reference/access-authn-authz/rbac/)

No fake placeholders. All parameters from alert labels.
