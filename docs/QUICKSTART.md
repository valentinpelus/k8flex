# K8flex Quick Reference

## Quick Start

```bash
# 1. Choose LLM Provider
export LLM_PROVIDER=ollama  # or openai, anthropic, gemini, bedrock
export OLLAMA_URL=http://ollama.ollama.svc.cluster.local:11434
export OLLAMA_MODEL=llama3

# 2. Build and deploy
docker build -t k8flex-agent:latest .
kubectl apply -f k8s/deployment.yaml

# 3. Configure Alertmanager (add to your alertmanager config)
receivers:
  - name: 'k8flex-ai-debug'
    webhook_configs:
      - url: 'http://k8flex-agent.k8flex.svc.cluster.local:8080/webhook'

# 4. Deploy Ollama (if using self-hosted)
kubectl create namespace ollama
kubectl apply -f k8s/ollama-deployment.yaml
kubectl exec -n ollama deployment/ollama -- ollama pull llama3

# 5. (Optional) Setup Slack
export SLACK_BOT_TOKEN=xoxb-...
export SLACK_CHANNEL_ID=C01234567

# 6. Test
kubectl apply -f test-alert.json
```

### Resource Issues (oom-killed, cpu-throttling)
- Current resource usage
- Resource limits
- QoS class
- Historical patterns

## Slack Setup (Optional)

**Basic (Webhook only):**
```bash
export SLACK_WEBHOOK_URL=https://hooks.slack.com/services/YOUR/WEBHOOK/URL
```

**Advanced (Bot token for threading, feedback, streaming):**
```bash
export SLACK_BOT_TOKEN=xoxb-...
export SLACK_CHANNEL_ID=C01234567
export SLACK_WORKSPACE_ID=T01234567  # Optional, for thread links
```

**Required Bot Scopes:**
- `chat:write` - Post messages
- `chat:write.public` - Post to public channels
- `reactions:read` - Detect feedback reactions

See [SLACK_SETUP.md](SLACK_SETUP.md) for detailed setup.

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
docker build -t k8flex-agent:latest .
kubectl rollout restart deployment k8flex-agent -n k8flex

# Scale up
kubectl scale deployment k8flex-agent -n k8flex --replicas=3
```

## Monitoring

**View feedback statistics in logs:**
```
Recorded ✅ feedback for alert 'PodCrashLooping' (category: pod-crash)
Feedback stats: Total=45, Correct=38 (84%), Incorrect=7 (16%)
```

**Knowledge base stats (if enabled):**
```
Found 3 similar cases in knowledge base (top similarity: 87.3%)
Knowledge Base Stats: total_cases=127, avg_similarity=0.82
```

## Configuration Examples

### Ollama (Self-hosted)
```bash
LLM_PROVIDER=ollama
OLLAMA_URL=http://ollama.ollama.svc.cluster.local:11434
OLLAMA_MODEL=llama3
```

### OpenAI
```bash
LLM_PROVIDER=openai
OPENAI_API_KEY=sk-proj-...
OPENAI_MODEL=gpt-4-turbo-preview
```

### Anthropic Claude
```bash
LLM_PROVIDER=anthropic
ANTHROPIC_API_KEY=sk-ant-...
ANTHROPIC_MODEL=claude-3-5-sonnet-20241022
```

### Google Gemini
```bash
LLM_PROVIDER=gemini
GEMINI_API_KEY=AIza...
GEMINI_MODEL=gemini-1.5-pro
```

### Knowledge Base
```bash
KB_ENABLED=true
KB_DATABASE_URL=postgresql://user:pass@host:5432/k8flex
KB_EMBEDDING_PROVIDER=openai
KB_EMBEDDING_API_KEY=sk-...
KB_SIMILARITY_THRESHOLD=0.75
KB_MAX_RESULTS=5
```

## Troubleshooting

Your Prometheus alerts must include these labels:

**Required:**
- `namespace` - Kubernetes namespace

**Optional (for targeted debugging):**
- `pod` - Pod name
- `service` - Service name
- `alertname` - Alert identifier
- `severity` - Alert severity level

**Example Alert:**
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

## What K8flex Debugs

K8flex automatically categorizes alerts and gathers relevant information:

### Pod Issues (pod-crash, pod-restart, pod-pending)
- Last 100 lines of pod logs (all containers)
- Pod status and conditions
- Container states and restart counts
- Recent namespace events
- Resource requests/limits

### Service Issues (service-down, endpoint-missing)
- Service configuration
- Endpoint status (ready/not-ready)
- Pod IPs and connectivity
- Network policies

### Node Issues (node-not-ready, disk-pressure)
- Node status and conditions
- Available resources
- Node events
- Pods on the node

### Network Issues (network-policy, dns-issues)
- Network policies (ingress/egress)
- DNS configuration
- Service endpoints

### Resource Issues (oom-killed, cpu-throttling)
- Current resource usage
- Resource limits
- QoS class
- Historical patterns

## Troubleshooting

**No analysis generated:**
- Check LLM provider is accessible
- Verify API keys are correct
- Check logs for errors

**Slack not working:**
- Verify webhook URL or bot token
- Check bot scopes (chat:write, reactions:read)
- Test with `curl` to Slack API

**Knowledge base not finding similar cases:**
- Check PostgreSQL is running
- Verify embedding provider API key
- Adjust similarity threshold (lower = more results)

**Feedback not detected:**
- Ensure bot has `reactions:read` scope
- Check Slack workspace ID is set
- Verify bot is in the channel

## Quick Links

- **[ARCHITECTURE.md](ARCHITECTURE.md)** - Complete system architecture and workflow
- **[LLM_PROVIDERS.md](LLM_PROVIDERS.md)** - All LLM provider configurations
- **[FEEDBACK.md](FEEDBACK.md)** - Feedback system details
- **[KNOWLEDGE_BASE.md](KNOWLEDGE_BASE.md)** - Vector database setup
- **[SLACK_SETUP.md](SLACK_SETUP.md)** - Detailed Slack configuration
- **[WEBHOOK_SECURITY.md](WEBHOOK_SECURITY.md)** - Webhook authentication

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
