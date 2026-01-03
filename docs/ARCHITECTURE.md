# K8flex Architecture & Workflow

## System Architecture

```
                              ┌─────────────────┐
Alertmanager ───webhook──────▶│   K8flex Agent  │
                              └────────┬────────┘
                                       │
                    ┌──────────────────┼──────────────────┐
                    │                  │                  │
                    ▼                  ▼                  ▼
            ┌──────────────┐   ┌─────────────┐   ┌──────────────┐
            │ Kubernetes   │   │  LLM API    │   │  Knowledge   │
            │     API      │   │  (Ollama/   │   │  Base (KB)   │
            │              │   │  OpenAI/    │   │  PostgreSQL  │
            │              │   │  Claude/    │   │  + pgvector  │
            │              │   │  Gemini)    │   │  (Optional)  │
            └──────────────┘   └─────────────┘   └──────────────┘
                    │                  │                  │
                    └──────────────────┼──────────────────┘
                                       │
                                       ▼
                              ┌─────────────────┐
                              │  Slack (with    │
                              │  threading &    │
                              │  feedback)      │
                              └─────────────────┘
                                       │
                                       ▼
                              ┌─────────────────┐
                              │  Feedback Loop  │
                              │  (Learn & Store)│
                              └─────────────────┘
```

## Complete Workflow

### 1️⃣ Alert Reception
```
Alertmanager sends webhook → K8flex validates auth token (if configured)
                          → Extracts labels (namespace, pod, service, etc.)
```

**Details:**
- Webhook endpoint: `/webhook`
- Validates `WEBHOOK_AUTH_TOKEN` if configured
- Parses alert labels and annotations
- Validates required fields (namespace)

### 2️⃣ AI Categorization
```
Alert labels + annotations → LLM categorizes alert type
                          → Determines debug scope (pod/service/node/network/resource)
```

**Categories:**
- `pod-crash`, `pod-restart`, `pod-pending` - Pod issues
- `service-down`, `endpoint-missing` - Service issues
- `node-not-ready`, `disk-pressure` - Node issues
- `network-policy`, `dns-issues` - Network issues
- `oom-killed`, `cpu-throttling` - Resource issues

**Purpose:** Gather only relevant debug data to reduce noise

### 3️⃣ Knowledge Search (if enabled)
```
Generate embedding of alert → Search vector database for similar cases
                          → Retrieve top N similar validated solutions
                          → Include in context for LLM
```

**Process:**
1. Generate embedding from alert name + severity + summary
2. Perform cosine similarity search in PostgreSQL pgvector
3. Return cases above similarity threshold (default 0.75)
4. Include top N cases (default 5) in LLM prompt

### 4️⃣ Targeted Debug Collection
```
Based on category → Gather only relevant K8s resources
                 → Pod logs, events, descriptions, network policies, etc.
                 → Retrieve past feedback for similar alerts
```

**Data Gathered by Category:**

**Pod Issues:**
- Last 100 lines of logs from all containers
- Pod description (status, conditions, container states)
- Recent namespace events (last 20)
- Resource requests/limits/usage

**Service Issues:**
- Service configuration and endpoints
- Pod IPs and connectivity status
- Endpoint ready/not-ready counts

**Node Issues:**
- Node status and conditions
- Allocatable resources
- Node-level events
- Pods running on node

**Network Issues:**
- Network policies (ingress/egress rules)
- CoreDNS status and logs
- Service discovery status

**Resource Issues:**
- Current CPU/memory consumption
- Configured requests and limits
- QoS class classification
- Recent resource patterns

### 5️⃣ Streaming AI Analysis
```
Send to LLM provider → Stream response in real-time
                    → Update Slack message every ~3 seconds
                    → User sees analysis develop progressively
```

**Streaming Process:**
1. Post initial alert to Slack (if configured)
2. Start LLM streaming request
3. Buffer chunks and update Slack message every 10 chunks
4. Show "🔄 Analysis in progress..." during streaming
5. Final update with "✅ Analysis Complete"
6. Add feedback instructions

**LLM Prompt Structure:**
1. System role with debugging expertise
2. Similar past cases (if found in knowledge base)
3. Past feedback from similar alerts
4. Complete debug information
5. Request structured analysis:
   - Root cause
   - Evidence
   - Impact assessment
   - Recommended actions
   - Prevention measures

### 6️⃣ Feedback Loop
```
User reacts with ✅/❌ → System detects reaction automatically
                      → Records feedback with timestamp
                      → If ✅, stores in knowledge base (if enabled)
                      → Future analyses use this learning
```

**Feedback Detection:**
- Background checker runs every 30 seconds
- Queries Slack API for reactions on analysis messages
- Detects ✅ (correct) or ❌ (incorrect) reactions
- Records feedback with alert metadata
- Stores to `/data/feedback.json`
- If ✅ and KB enabled, stores to knowledge base

**Feedback Storage:**
```json
{
  "timestamp": "2026-01-04T10:00:00Z",
  "alert_name": "PodCrashLooping",
  "category": "pod-crash",
  "is_correct": true,
  "analysis": "Root cause: missing ConfigMap...",
  "slack_thread": "1704364800.123456"
}
```

### 7️⃣ Continuous Improvement
```
More feedback → Better categorization → More accurate analyses
            → Knowledge base grows → Faster incident resolution
            → Similar issues resolved instantly
```

**Learning Loop:**
1. Human validates analysis with reaction
2. Feedback influences future similar alerts
3. Validated cases stored in vector database
4. Similar future incidents get instant context
5. Analysis quality improves over time

## Component Details

### Debugger Module
**Location:** `internal/debugger/`

**Responsibilities:**
- Kubernetes API client management
- Category-based debug information gathering
- Pod logs collection
- Event retrieval
- Resource status checks

### LLM Provider Module
**Location:** `pkg/llm/`

**Components:**
- `provider.go` - Interface definition
- `ollama.go` - Ollama implementation
- `openai.go` - OpenAI implementation
- `anthropic.go` - Anthropic implementation
- `gemini.go` - Gemini implementation
- `bedrock.go` - AWS Bedrock implementation
- `factory.go` - Provider factory

**Interface:**
```go
type Provider interface {
    Name() string
    CategorizeAlert(alert Alert) (string, error)
    AnalyzeDebugInfo(debugInfo string, pastFeedback []Feedback) (string, error)
    AnalyzeDebugInfoStream(debugInfo string, pastFeedback []Feedback, updateFn func(string)) error
}
```

### Feedback Module
**Location:** `pkg/feedback/`

**Responsibilities:**
- Store feedback to JSON file
- Retrieve relevant past feedback
- Track accuracy statistics
- Filter by category and alert name

### Knowledge Base Module
**Location:** `pkg/knowledge/`

**Components:**
- `store.go` - PostgreSQL + pgvector storage
- `embeddings.go` - OpenAI/Gemini embeddings
- `types.go` - Data structures

**Schema:**
```sql
CREATE TABLE alert_cases (
    id UUID PRIMARY KEY,
    alert_name TEXT,
    severity TEXT,
    category TEXT,
    namespace TEXT,
    pod_name TEXT,
    analysis TEXT,
    debug_info TEXT,
    validated BOOLEAN,
    similarity_score FLOAT,
    embedding vector(1536), -- or 768 for Gemini
    created_at TIMESTAMP,
    validated_at TIMESTAMP
);

CREATE INDEX ON alert_cases USING ivfflat (embedding vector_cosine_ops);
```

### Slack Module
**Location:** `pkg/slack/`

**Features:**
- Webhook posting
- Bot token API integration
- Threaded messages
- Real-time message updates
- Reaction detection
- Historical thread links

## Data Flow

### Alert Processing Data Flow
```
Alert JSON
    ↓
Parse Labels/Annotations
    ↓
Category = LLM.CategorizeAlert(alert)
    ↓
SimilarCases = KB.FindSimilar(alert) [if enabled]
    ↓
DebugInfo = Debugger.GatherDebugInfo(alert, category)
    ↓
PastFeedback = FeedbackManager.GetRelevant(category, alert_name)
    ↓
Analysis = LLM.AnalyzeDebugInfoStream(debugInfo, pastFeedback)
    ↓
Slack.SendAlert(alert)
Slack.StreamAnalysis(analysis, thread)
    ↓
WaitForFeedback()
    ↓
IF feedback == ✅:
    KB.Store(alert, analysis, debugInfo) [if enabled]
```

### Knowledge Base Data Flow
```
Validated Analysis
    ↓
GenerateEmbedding(alert_text)
    ↓
Store(alert_case, embedding) → PostgreSQL
    ↓
Future Similar Alert
    ↓
GenerateEmbedding(new_alert_text)
    ↓
CosineSimilaritySearch(embedding) → Similar Cases
    ↓
Include in LLM Context
```

## Scalability Considerations

### Horizontal Scaling
- K8flex agent is stateless (except feedback file)
- Can run multiple replicas
- Feedback file needs shared storage (PVC) or database
- Knowledge base is centralized (PostgreSQL)

### Performance
- Streaming reduces perceived latency
- Knowledge base search is O(log n) with ivfflat index
- Parallel alert processing supported
- LLM provider is the bottleneck

### Resource Requirements
**Minimum:**
- CPU: 100m
- Memory: 128Mi

**Recommended:**
- CPU: 500m
- Memory: 512Mi

**With Knowledge Base:**
- PostgreSQL: 1 CPU, 2Gi memory
- Storage: 10Gi+

## Security Architecture

### Authentication Flow
```
Alertmanager
    ↓ (Authorization: Bearer TOKEN)
K8flex validates WEBHOOK_AUTH_TOKEN
    ↓ (if valid)
Process Alert
```

### RBAC Permissions
- `get`, `list` - Pods, Services, Endpoints, Events, Nodes
- `get` - ConfigMaps, NetworkPolicies (metadata only)
- No `create`, `update`, `delete` permissions
- No Secret data access (only metadata)

### Secret Management
- LLM API keys stored as Kubernetes Secrets
- Slack tokens stored as Secrets
- Database credentials via Secrets
- Environment variables reference secrets

## Monitoring & Observability

### Metrics Endpoint
- `/health` - Health check endpoint
- Returns 200 OK if healthy

### Logs
**Structured logging includes:**
- Alert name and namespace
- Category determined
- Similar cases found (count, top similarity)
- Feedback recorded (correct/incorrect)
- Errors with context

**Example:**
```
Processing alert: PodCrashLooping
Asking claude-3-5-sonnet to categorize alert: PodCrashLooping
Claude categorized alert as: pod-crash
Found 2 similar cases in knowledge base (top similarity: 87.3%)
Starting streaming analysis from claude-3-5-sonnet
Analysis complete for PodCrashLooping
Recorded ✅ feedback for alert 'PodCrashLooping' (category: pod-crash)
```

### Health Checks
```bash
# Liveness probe
kubectl exec -n k8flex deployment/k8flex-agent -- \
  wget -qO- http://localhost:8080/health

# Readiness probe
kubectl exec -n k8flex deployment/k8flex-agent -- \
  wget -qO- http://localhost:8080/health
```

## Future Architecture Enhancements

### Planned Features
- **Multi-cluster support**: Aggregate alerts from multiple clusters
- **Custom playbooks**: Execute automated remediation actions
- **Web UI**: Browse knowledge base and feedback history
- **Metrics export**: Prometheus metrics for feedback accuracy
- **Alert deduplication**: Prevent duplicate analyses
- **Webhook fanout**: Send to multiple endpoints

### Scalability Roadmap
- Redis for distributed feedback storage
- Kafka for event streaming
- Distributed tracing (OpenTelemetry)
- Rate limiting per LLM provider
- Caching layer for similar cases
