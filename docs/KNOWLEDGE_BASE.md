# Knowledge Base - Vector Database Integration

## Overview

The k8flex Knowledge Base feature stores validated alert cases with vector embeddings in a PostgreSQL database (with pgvector extension), enabling semantic similarity search for faster and more accurate alert analysis.

## How It Works

1. **Alert Processing**: When an alert arrives, k8flex searches the knowledge base for similar past cases
2. **Similarity Matching**: Uses vector embeddings to find semantically similar alerts (not just keyword matching)
3. **Context Enhancement**: Includes similar past cases in the LLM prompt to improve analysis quality
4. **Feedback Loop**: When users validate an analysis (✅ reaction in Slack), it's stored in the knowledge base
5. **Continuous Learning**: The knowledge base grows over time, improving response quality and speed

## Architecture

```
┌─────────────┐      ┌──────────────────┐      ┌─────────────────┐
│   Alert     │──1──▶│  Search Similar  │──2──▶│   PostgreSQL    │
│   Arrives   │      │  Cases (Vector)  │      │   + pgvector    │
└─────────────┘      └──────────────────┘      └─────────────────┘
                              │                          │
                              ▼                          │
                     ┌──────────────────┐               │
                     │  LLM Analysis    │               │
                     │  (with context)  │               │
                     └──────────────────┘               │
                              │                          │
                              ▼                          │
                     ┌──────────────────┐               │
                     │ User Validation  │               │
                     │ (✅ in Slack)    │──3──▶Store────┘
                     └──────────────────┘
```

## Setup

### 1. Database Setup (PostgreSQL with pgvector)

#### Option A: AWS RDS PostgreSQL

1. Create an RDS PostgreSQL instance (15.x or later recommended)
2. Enable pgvector extension:
   ```sql
   CREATE EXTENSION IF NOT EXISTS vector;
   ```
3. Run the migration script:
   ```bash
   psql -h your-rds-endpoint.rds.amazonaws.com -U k8flex -d k8flex < deployments/migrations/001_init_knowledge_base.sql
   ```

#### Option B: Self-Hosted PostgreSQL

1. Install PostgreSQL 15+ with pgvector:
   ```bash
   # Using Docker
   docker run -d \
     --name k8flex-postgres \
     -e POSTGRES_USER=k8flex \
     -e POSTGRES_PASSWORD=yourpassword \
     -e POSTGRES_DB=k8flex \
     -p 5432:5432 \
     pgvector/pgvector:pg15
   ```

2. Run the migration:
   ```bash
   psql -h localhost -U k8flex -d k8flex < deployments/migrations/001_init_knowledge_base.sql
   ```

### 2. Configure Embeddings Provider

The knowledge base needs an embeddings API to convert text into vectors. Choose one:

#### OpenAI (Recommended)
- **Model**: `text-embedding-3-small` (1536 dimensions, fast and cheap)
- **Cost**: ~$0.02 per 1M tokens
- **Setup**: Use your OpenAI API key (same as LLM provider or separate)

#### Google Gemini
- **Model**: `embedding-001` (768 dimensions)
- **Cost**: Free tier available
- **Setup**: Use your Gemini API key

### 3. Helm Configuration

Update your `values.yaml`:

```yaml
# Enable knowledge base
knowledgeBase:
  enabled: true
  
  # PostgreSQL connection (use secret for production)
  databaseUrl: "postgresql://k8flex:password@k8flex-kb.xxxxx.us-east-1.rds.amazonaws.com:5432/k8flex?sslmode=require"
  
  # Embedding provider
  embeddingProvider: "openai"  # or "gemini"
  embeddingApiKey: ""  # Leave empty to use LLM provider API key
  embeddingModel: "text-embedding-3-small"
  
  # Similarity settings
  similarityThreshold: 0.75  # 0.7-0.9 recommended (higher = stricter matching)
  maxResults: 5              # Number of similar cases to retrieve
```

### 4. Secure Secrets (Production)

Instead of putting the database URL in values.yaml, use a Kubernetes secret:

```bash
# Create secret
kubectl create secret generic k8flex-kb-secret \
  --namespace k8flex \
  --from-literal=KB_DATABASE_URL='postgresql://user:pass@host:5432/db?sslmode=require'

# Update deployment to use the secret
kubectl edit deployment k8flex -n k8flex
```

Add to pod spec:
```yaml
env:
  - name: KB_DATABASE_URL
    valueFrom:
      secretKeyRef:
        name: k8flex-kb-secret
        key: KB_DATABASE_URL
```

## Usage

### Automatic Learning

Once configured, the system automatically:
1. Searches for similar cases before analyzing new alerts
2. Stores validated cases when users react with ✅ in Slack
3. Uses historical knowledge to improve future analyses

### Manual Validation

You can manually validate cases via the feedback API:
```bash
curl -X POST http://k8flex:8080/api/feedback \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "alert_name": "PodCrashLooping",
    "category": "🔄",
    "analysis": "...",
    "is_correct": true
  }'
```

### Monitoring

Check knowledge base statistics:
```sql
-- View stats
SELECT * FROM alert_cases_stats;

-- Top categories
SELECT category, COUNT(*) as count 
FROM alert_cases 
WHERE validated = true 
GROUP BY category 
ORDER BY count DESC;

-- Recent cases
SELECT alert_name, category, created_at, similarity_score
FROM alert_cases 
WHERE validated = true 
ORDER BY created_at DESC 
LIMIT 10;
```

## Configuration Reference

| Environment Variable | Description | Default |
|---------------------|-------------|---------|
| `KB_ENABLED` | Enable knowledge base | `false` |
| `KB_DATABASE_URL` | PostgreSQL connection string | - |
| `KB_EMBEDDING_PROVIDER` | Embeddings API (`openai`, `gemini`) | `openai` |
| `KB_EMBEDDING_API_KEY` | API key for embeddings (optional) | Uses LLM key |
| `KB_EMBEDDING_MODEL` | Embedding model name | `text-embedding-3-small` |
| `KB_SIMILARITY_THRESHOLD` | Minimum similarity (0-1) | `0.75` |
| `KB_MAX_RESULTS` | Max similar cases to retrieve | `5` |

## Tuning

### Similarity Threshold

- **0.9+**: Very strict - only nearly identical cases
- **0.75-0.85**: Recommended - similar issues and patterns
- **0.6-0.7**: Loose - includes broader related cases
- **<0.6**: Too loose - may include unrelated cases

### Embedding Models

#### OpenAI
- `text-embedding-3-small` (1536 dims): Fast, cheap, good quality ✅
- `text-embedding-3-large` (3072 dims): Higher quality, slower, 3x cost
- `text-embedding-ada-002` (1536 dims): Legacy model

#### Gemini
- `embedding-001` (768 dims): Free tier, good quality

## Cost Estimation

### OpenAI Embeddings (text-embedding-3-small)

- **Per Alert**: 2 embeddings (search query + storage) ≈ 500 tokens
- **Cost**: $0.02 per 1M tokens = $0.00001 per alert
- **Monthly (1000 alerts)**: ~$0.01
- **Monthly (100,000 alerts)**: ~$1.00

### Storage (PostgreSQL)

- **Per Case**: ~5KB (metadata + 1536-dim vector)
- **10,000 cases**: ~50MB
- **100,000 cases**: ~500MB
- **RDS cost**: $0.10-0.50/GB/month

## Troubleshooting

### "pgvector extension not installed"
```sql
-- Connect to your database
CREATE EXTENSION IF NOT EXISTS vector;
```

### "Failed to generate embedding"
- Check API key is valid
- Verify network connectivity to embedding API
- Check API rate limits

### "No similar cases found"
- Lower `similarityThreshold` (try 0.65)
- Verify cases are being stored (check `validated = true`)
- Check embedding dimensions match (1536 for OpenAI small)

### Performance Issues

For large knowledge bases (>100k cases):
```sql
-- Check index usage
EXPLAIN ANALYZE 
SELECT * FROM alert_cases 
WHERE embedding <=> '[0.1, 0.2, ...]'::vector < 0.25;

-- Rebuild HNSW index if needed
REINDEX INDEX CONCURRENTLY idx_alert_cases_embedding;
```

## Best Practices

1. **Start Small**: Begin with `enabled: false`, test with a small database first
2. **Monitor Costs**: Track embedding API usage in your provider dashboard
3. **Tune Threshold**: Start at 0.75, adjust based on result quality
4. **Regular Backups**: Backup your PostgreSQL database regularly
5. **Clean Old Data**: Archive or delete very old cases that may no longer be relevant
6. **Validate Quality**: Review stored cases periodically to ensure quality

## Database Schema

```sql
CREATE TABLE alert_cases (
    id VARCHAR(36) PRIMARY KEY,
    alert_name VARCHAR(255) NOT NULL,
    severity VARCHAR(50),
    category VARCHAR(100) NOT NULL,    -- Emoji/category
    summary TEXT,
    namespace VARCHAR(255),
    pod_name VARCHAR(255),
    container_name VARCHAR(255),
    analysis TEXT NOT NULL,
    debug_info TEXT,
    validated BOOLEAN DEFAULT true,
    embedding vector(1536),            -- Vector for similarity search
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

-- Indexes for performance
CREATE INDEX idx_alert_cases_embedding ON alert_cases 
USING hnsw (embedding vector_cosine_ops);
```

## Future Enhancements

- [ ] Support for other embedding providers (AWS Bedrock, local models)
- [ ] Automated case deduplication
- [ ] Case aging/deprecation for outdated solutions
- [ ] Multi-cluster knowledge sharing
- [ ] Export/import knowledge base for backup/migration
- [ ] Web UI for browsing and managing cases
- [ ] A/B testing different similarity thresholds
- [ ] Integration with incident management systems

## Support

For issues or questions:
1. Check logs: `kubectl logs -n k8flex deployment/k8flex | grep -i "knowledge\|KB_"`
2. Verify database connectivity
3. Test embeddings API separately
4. Review configuration in deployed pod: `kubectl exec -n k8flex deployment/k8flex -- env | grep KB_`
