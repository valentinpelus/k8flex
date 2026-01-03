# Webhook Security Configuration

## Overview

The k8flex webhook API is protected with Bearer token authentication. Only requests with a valid token in the `Authorization` header will be processed.

## Configuration Steps

### 1. Generate a Secure Token

```bash
openssl rand -hex 32
```

Example output: `a3f8d9e2c1b4567890abcdef1234567890abcdef1234567890abcdef12345678`

### 2. Configure k8flex

Add the token to your `helm/k8flex/values.yaml`:

```yaml
webhook:
  authToken: "a3f8d9e2c1b4567890abcdef1234567890abcdef1234567890abcdef12345678"
```

Or set it via helmfile override:

```yaml
# helmfile.yaml
releases:
  - name: k8flex-agent
    values:
      - webhook:
          authToken: "a3f8d9e2c1b4567890abcdef1234567890abcdef1234567890abcdef12345678"
```

### 3. Configure Alertmanager

Update your Alertmanager configuration to send the token in the `Authorization` header:

```yaml
# k8s/app/alertmanager/values.yaml
config:
  receivers:
    - name: 'k8flex-ai-debug'
      webhook_configs:
        - url: 'http://k8flex-agent.k8flex.svc.cluster.local:8080/webhook'
          send_resolved: false
          http_config:
            authorization:
              credentials: "a3f8d9e2c1b4567890abcdef1234567890abcdef1234567890abcdef12345678"
```

### 4. Deploy Updates

```bash
# Deploy k8flex with new token
cd /Users/valentinpelus/Documents/workspace/k8flex
./deploy.sh

# Update Alertmanager configuration
helm upgrade alertmanager prometheus-community/alertmanager \
  --namespace monitoring \
  --values /Users/valentinpelus/Documents/workspace/k8s/app/alertmanager/values.yaml
```

## Testing

### With Valid Token

```bash
TOKEN="your-token-here"

curl -X POST http://k8flex.k8flex.svc.cluster.local:8080/webhook \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "alerts": [{
      "status": "firing",
      "labels": {
        "alertname": "TestAlert",
        "namespace": "default",
        "pod": "test-pod"
      },
      "annotations": {
        "summary": "Test alert"
      }
    }]
  }'
```

**Expected:** Alert processed successfully

### Without Token

```bash
curl -X POST http://k8flex.k8flex.svc.cluster.local:8080/webhook \
  -H "Content-Type: application/json" \
  -d '{"alerts": [...]}'
```

**Expected:** `401 Unauthorized: Missing Authorization header`

### With Invalid Token

```bash
curl -X POST http://k8flex.k8flex.svc.cluster.local:8080/webhook \
  -H "Authorization: Bearer invalid-token" \
  -H "Content-Type: application/json" \
  -d '{"alerts": [...]}'
```

**Expected:** `401 Unauthorized: Invalid token`

## Security Best Practices

1. **Generate Strong Tokens**: Use `openssl rand -hex 32` or similar cryptographically secure method
2. **Store Securely**: Never commit tokens to git. Use Kubernetes secrets or external secret managers
3. **Rotate Regularly**: Change tokens periodically (recommended: every 90 days)
4. **Use Different Tokens**: Use separate tokens for different environments (dev/staging/prod)
5. **Limit Access**: Only share tokens with authorized services (Alertmanager, monitoring tools)

## Troubleshooting

### Logs show authentication errors

Check k8flex logs:
```bash
kubectl logs -n k8flex deployment/k8flex | grep -i "unauthorized\|auth"
```

### Alertmanager can't send alerts

1. Verify token matches in both configurations
2. Check Alertmanager logs for HTTP 401 responses
3. Test webhook directly with curl

### Emergency: Disable Authentication

If you need to temporarily disable authentication:

```yaml
# helm/k8flex/values.yaml
webhook:
  authToken: ""  # Empty = disabled (NOT RECOMMENDED FOR PRODUCTION)
```

**Warning:** This allows anyone to send alerts to your system!

## Environment Variable

The token is injected as the `WEBHOOK_AUTH_TOKEN` environment variable from the Kubernetes secret.

To view current configuration (base64 encoded):
```bash
kubectl get secret k8flex-agent-secrets -n k8flex -o jsonpath='{.data.WEBHOOK_AUTH_TOKEN}' | base64 -d
```
