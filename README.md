# K8flex - AI-Powered Kubernetes Debug Agent

AI-powered incident response agent that receives webhooks from popular alerting systems and performs automated Kubernetes debugging. Learns from feedback and maintains a knowledge base for faster resolution.

## Features

- **Multi-Source Webhooks**: Alertmanager, PagerDuty, Grafana, Datadog, Opsgenie, VictorOps, New Relic
- **Automated Debugging**: Gathers logs, events, pod status, services, and network policies from Kubernetes
- **Multi-LLM Support**: Ollama (self-hosted), OpenAI, Anthropic Claude, Google Gemini, or AWS Bedrock
- **Real-Time Streaming**: Analysis streams progressively to Slack as it develops
- **Learning System**: Rate analyses with ✅/❌ in Slack; system learns from feedback
- **Knowledge Base** (Optional): PostgreSQL + pgvector for semantic search of past incidents
- **Slack Integration**: Threaded conversations with historical context links

## Quick Start

### 1. Choose LLM Provider

**Ollama (Self-hosted)**
```bash
export LLM_PROVIDER=ollama
export OLLAMA_URL=http://ollama.ollama.svc.cluster.local:11434
export OLLAMA_MODEL=llama3
```

**OpenAI / Claude / Gemini**
```bash
export LLM_PROVIDER=openai  # or anthropic, gemini, bedrock
export OPENAI_API_KEY=sk-...
export OPENAI_MODEL=gpt-4-turbo-preview
```

See [LLM_PROVIDERS.md](docs/LLM_PROVIDERS.md) for all options.

### 2. Deploy

```bash
docker build -t k8flex-agent:latest .
kubectl apply -f k8s/deployment.yaml
```

### 3. Configure Alerting System

K8flex supports webhooks from multiple alerting systems (Alertmanager, PagerDuty, Grafana, Datadog, Opsgenie, VictorOps, New Relic). Choose the one(s) you use:

**Alertmanager (Prometheus):**

```yaml
receivers:
  - name: 'k8flex-ai-debug'
    webhook_configs:
      - url: 'http://k8flex-agent.k8flex.svc.cluster.local:8080/webhook'
```

**Optional: Limit Enabled Adapters**

By default, all alerting systems are supported. To enable only specific ones:

```bash
export ENABLED_ADAPTERS=alertmanager,pagerduty,grafana
```

See [Adapter Configuration](docs/ADAPTER_CONFIGURATION.md) for all options.

Full setup for each system: [INTEGRATION.md](docs/INTEGRATION.md)

### 4. Optional: Slack Integration

```bash
export SLACK_BOT_TOKEN=xoxb-...
export SLACK_CHANNEL_ID=C01234567
```

Required scopes: `chat:write`, `chat:write.public`, `reactions:read`  
Details: [SLACK_SETUP.md](docs/SLACK_SETUP.md)

### 5. Optional: Knowledge Base

```bash
export KB_ENABLED=true
export KB_DATABASE_URL="postgresql://user:pass@host:5432/k8flex"
export KB_EMBEDDING_PROVIDER=openai
```

Setup: [KNOWLEDGE_BASE.md](docs/KNOWLEDGE_BASE.md)

## How It Works

1. Alertmanager sends webhook → K8flex receives alert
2. AI categorizes alert type (pod/service/node/network/resource)
3. System searches knowledge base for similar past cases (if enabled)
4. Gathers targeted Kubernetes debug information
5. AI analyzes and streams results to Slack in real-time
6. Users rate analysis with ✅/❌ reactions
7. Validated solutions stored for future incidents

## Configuration

### Key Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `LLM_PROVIDER` | `ollama` | `ollama`, `openai`, `anthropic`, `gemini`, `bedrock` |
| `OLLAMA_URL` | `http://ollama.ollama.svc.cluster.local:11434` | Ollama endpoint |
| `OLLAMA_MODEL` | `llama3` | Model name |
| `OPENAI_API_KEY` | - | OpenAI API key |
| `SLACK_BOT_TOKEN` | - | Slack bot token (for advanced features) |
| `SLACK_CHANNEL_ID` | - | Slack channel ID |
| `KB_ENABLED` | `false` | Enable knowledge base |
| `KB_DATABASE_URL` | - | PostgreSQL connection string |
| `WEBHOOK_AUTH_TOKEN` | - | Webhook authentication token |

Full reference: See **Complete Configuration Reference** section below or [Configuration Documentation](docs/).

### Slack Scopes Required

For feedback system and threading:
- `chat:write` - Post messages
- `chat:write.public` - Post to public channels
- `reactions:read` - Detect emoji reactions

## Alert Requirements

Alerts must include these labels:
- `namespace` (required): Kubernetes namespace
- `pod` (optional): Pod name
- `service` (optional): Service name
- `alertname`: Alert identifier
- `severity`: Alert severity

Example:
```yaml
- alert: PodNotReady
  expr: kube_pod_status_phase{phase!="Running"} == 1
  labels:
    namespace: "{{ $labels.namespace }}"
    pod: "{{ $labels.pod }}"
    severity: warning
```

## Development

```bash
# Local testing
go run main.go

# Test webhook
curl -XPOST 'http://localhost:8080/webhook' \
  -H 'Content-Type: application/json' \
  -d @test-alert.json

# Build
go build -o k8flex-agent .
```

## Documentation

- **[INTEGRATION.md](docs/INTEGRATION.md)** - Alertmanager/Prometheus setup
- **[ARCHITECTURE.md](docs/ARCHITECTURE.md)** - Complete architecture and workflow
- **[QUICKSTART.md](docs/QUICKSTART.md)** - Quick reference and examples
- **[USE_CASES.md](docs/USE_CASES.md)** - Use cases, benefits, and best practices
- **[LLM_PROVIDERS.md](docs/LLM_PROVIDERS.md)** - All LLM provider configs
- **[SLACK_SETUP.md](docs/SLACK_SETUP.md)** - Slack bot configuration
- **[FEEDBACK.md](docs/FEEDBACK.md)** - Feedback system details
- **[KNOWLEDGE_BASE.md](docs/KNOWLEDGE_BASE.md)** - Vector database setup
- **[WEBHOOK_SECURITY.md](docs/WEBHOOK_SECURITY.md)** - Webhook authentication

## Complete Configuration Reference

<details>
<summary>Click to expand full environment variable list</summary>

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP server port |
| `LLM_PROVIDER` | `ollama` | LLM provider |
| `OLLAMA_URL` | `http://ollama.ollama.svc.cluster.local:11434` | Ollama endpoint |
| `OLLAMA_MODEL` | `llama3` | Ollama model |
| `OPENAI_API_KEY` | - | OpenAI API key |
| `OPENAI_MODEL` | `gpt-4-turbo-preview` | OpenAI model |
| `ANTHROPIC_API_KEY` | - | Anthropic API key |
| `ANTHROPIC_MODEL` | `claude-3-5-sonnet-20241022` | Anthropic model |
| `GEMINI_API_KEY` | - | Gemini API key |
| `GEMINI_MODEL` | `gemini-1.5-pro` | Gemini model |
| `BEDROCK_REGION` | `us-east-1` | AWS region |
| `BEDROCK_MODEL` | `anthropic.claude-3-5-sonnet-20241022-v2:0` | Bedrock model ARN |
| `SLACK_WEBHOOK_URL` | - | Slack webhook (basic) |
| `SLACK_BOT_TOKEN` | - | Slack bot token (advanced) |
| `SLACK_CHANNEL_ID` | - | Slack channel ID |
| `SLACK_WORKSPACE_ID` | - | Workspace ID for thread links |
| `WEBHOOK_AUTH_TOKEN` | - | Webhook auth token |
| `KB_ENABLED` | `false` | Enable knowledge base |
| `KB_DATABASE_URL` | - | PostgreSQL URL |
| `KB_EMBEDDING_PROVIDER` | `openai` | `openai` or `gemini` |
| `KB_EMBEDDING_API_KEY` | - | Embedding API key |
| `KB_EMBEDDING_MODEL` | `text-embedding-3-small` | Embedding model |
| `KB_SIMILARITY_THRESHOLD` | `0.75` | Similarity threshold (0-1) |
| `KB_MAX_RESULTS` | `5` | Max similar cases |

</details>

## License

MIT

## Contributing

Contributions welcome! Ensure parameters are extracted from alert labels.
