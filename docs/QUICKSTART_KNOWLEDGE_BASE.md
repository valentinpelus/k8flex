# Quick Start: Knowledge Base

Enable vector database knowledge base in 5 minutes.

## Prerequisites

- PostgreSQL 15+ with pgvector extension
- OpenAI or Gemini API key

## Setup

### 1. Initialize Database

```bash
# Set your database credentials
export DB_HOST=your-postgres-host
export DB_PORT=5432
export DB_USER=k8flex
export DB_PASSWORD=yourpassword
export DB_NAME=k8flex

# Run setup script
./scripts/setup-knowledge-base.sh
```

### 2. Update Helm Values

```yaml
# values.yaml
knowledgeBase:
  enabled: true
  databaseUrl: "postgresql://k8flex:yourpassword@your-host:5432/k8flex?sslmode=require"
  embeddingProvider: "openai"  # or "gemini"
  # embeddingApiKey: ""  # Optional: uses LLM API key by default
  embeddingModel: "text-embedding-3-small"
  similarityThreshold: 0.75
  maxResults: 5
```

### 3. Deploy

```bash
helm upgrade k8flex ./helm/k8flex -n k8flex --values values.yaml
```

### 4. Verify

```bash
# Check logs
kubectl logs -n k8flex deployment/k8flex | grep -i knowledge

# Expected output:
# ✅ Knowledge base enabled: openai embeddings, similarity threshold: 0.75
```

## How It Works

1. **Alert arrives** → Search for similar past cases
2. **LLM analyzes** with historical context included
3. **User validates** with ✅ reaction in Slack
4. **Case stored** in vector database
5. **Future alerts** benefit from this knowledge

## Example

**First Alert (PodCrashLooping)**:
```
Alert: PodCrashLooping
Found: 0 similar cases
Analysis: [Full debug analysis]
User: ✅ (validates in Slack)
→ Case stored in knowledge base
```

**Second Similar Alert**:
```
Alert: PodCrashLooping (different pod)
Found: 1 similar case (92% similarity)
Similar case: "Previous crash due to OOM, increased memory limits"
Analysis: [Uses historical context for faster, better analysis]
```

## Cost

**Per 1000 alerts/month**:
- Embeddings: $0.01
- Database: ~$15 (RDS db.t3.micro)
- **Total: ~$15/month**

## Tuning

### Similarity Threshold

- **0.75** (default): Good balance
- **0.85**: Stricter matching (fewer but more relevant)
- **0.65**: Looser matching (more cases, less relevant)

Test and adjust based on your results.

## Troubleshooting

### "No similar cases found"
- Lower threshold: `similarityThreshold: 0.65`
- Check cases exist: `SELECT COUNT(*) FROM alert_cases;`

### "Failed to connect to database"
- Verify connection string
- Check network/firewall rules
- Test with: `psql "$KB_DATABASE_URL"`

### "Failed to generate embedding"
- Verify API key is correct
- Check API rate limits
- Test embedding API separately

## Production Checklist

- [ ] Use Kubernetes secrets for database URL
- [ ] Enable SSL mode in connection string (`?sslmode=require`)
- [ ] Set up database backups
- [ ] Monitor embedding API usage
- [ ] Review and tune similarity threshold
- [ ] Set up database connection pooling
- [ ] Configure resource limits appropriately

## Learn More

- Full documentation: [docs/KNOWLEDGE_BASE.md](docs/KNOWLEDGE_BASE.md)
- Implementation details: [KNOWLEDGE_BASE_IMPLEMENTATION.md](KNOWLEDGE_BASE_IMPLEMENTATION.md)
- Example config: [examples/values-with-knowledge-base.yaml](examples/values-with-knowledge-base.yaml)

## Disable (if needed)

No data loss - just set:
```yaml
knowledgeBase:
  enabled: false
```

All stored cases remain in the database for future use.
