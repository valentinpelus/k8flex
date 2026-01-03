# Adapter Configuration Guide

This guide explains how to configure which alerting systems k8flex supports.

## Overview

K8flex can receive webhooks from multiple alerting systems simultaneously. Each system has its own adapter that converts the vendor-specific webhook format to k8flex's internal alert format.

By default, **all adapters are enabled**. You can optionally restrict which adapters are active using the `ENABLED_ADAPTERS` environment variable.

## Supported Adapters

| Adapter Name | Alerting System | Auto-Detection |
|--------------|----------------|----------------|
| `alertmanager` | Prometheus Alertmanager | ✅ |
| `pagerduty` | PagerDuty | ✅ |
| `grafana` | Grafana Alerting | ✅ |
| `datadog` | Datadog Monitors | ✅ |
| `opsgenie` | Opsgenie | ✅ |
| `victorops` | VictorOps (Splunk On-Call) | ✅ |
| `newrelic` | New Relic Alerts | ✅ |

## Configuration Methods

### Method 1: Enable All Adapters (Default)

If `ENABLED_ADAPTERS` is not set, all adapters are enabled:

```bash
# No configuration needed - all adapters active by default
```

### Method 2: Enable Specific Adapters

Set `ENABLED_ADAPTERS` to a comma-separated list of adapter names:

```bash
export ENABLED_ADAPTERS=alertmanager,pagerduty,grafana
```

**Note:** Adapter names are case-insensitive. Spaces are trimmed automatically.

### Method 3: Enable Single Adapter

To accept webhooks from only one system:

```bash
export ENABLED_ADAPTERS=alertmanager
```

## Configuration by Deployment Method

### Docker

```bash
docker run \
  -e ENABLED_ADAPTERS=alertmanager,grafana \
  -p 8080:8080 \
  k8flex-agent:latest
```

### Docker Compose

```yaml
version: '3.8'
services:
  k8flex:
    image: k8flex-agent:latest
    environment:
      ENABLED_ADAPTERS: "alertmanager,pagerduty,grafana"
    ports:
      - "8080:8080"
```

### Kubernetes - ConfigMap

Edit `k8s/deployment.yaml`:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: k8flex-config
  namespace: k8flex
data:
  ENABLED_ADAPTERS: "alertmanager,pagerduty,grafana"
```

Then apply:

```bash
kubectl apply -f k8s/deployment.yaml
kubectl rollout restart deployment/k8flex-agent -n k8flex
```

### Kubernetes - kubectl set env

Update running deployment:

```bash
kubectl set env deployment/k8flex-agent -n k8flex \
  ENABLED_ADAPTERS=alertmanager,grafana

# Verify
kubectl get deployment k8flex-agent -n k8flex -o jsonpath='{.spec.template.spec.containers[0].env}'
```

### Kubernetes - Helm (if using Helm chart)

```yaml
# values.yaml
config:
  enabledAdapters: "alertmanager,pagerduty,grafana,datadog"
```

## Use Cases

### Scenario 1: Single Alerting System

**Use Case:** You only use Prometheus Alertmanager

```bash
ENABLED_ADAPTERS=alertmanager
```

**Benefits:**
- Faster webhook processing (fewer adapters to try)
- Clearer logs (no attempted parsing from disabled adapters)
- Security: Rejects webhooks from unauthorized sources

### Scenario 2: Hybrid Cloud

**Use Case:** On-premise Prometheus + cloud-based PagerDuty

```bash
ENABLED_ADAPTERS=alertmanager,pagerduty
```

### Scenario 3: Multi-Vendor Monitoring

**Use Case:** Different teams use different tools

```bash
ENABLED_ADAPTERS=alertmanager,grafana,datadog
```

**Benefits:**
- Infrastructure team uses Prometheus/Alertmanager
- Application team uses Grafana
- SRE team uses Datadog
- All feed into same k8flex instance

### Scenario 4: Migration Period

**Use Case:** Migrating from Alertmanager to Grafana

```bash
# During migration, accept both
ENABLED_ADAPTERS=alertmanager,grafana

# After migration complete
ENABLED_ADAPTERS=grafana
```

### Scenario 5: Enterprise Stack

**Use Case:** Full observability stack

```bash
# Enable everything
ENABLED_ADAPTERS=alertmanager,pagerduty,grafana,datadog,opsgenie,victorops,newrelic
# Or simply omit ENABLED_ADAPTERS to enable all by default
```

## How Auto-Detection Works

When k8flex receives a webhook:

1. **Request arrives** at `/webhook` endpoint
2. **Registry checks** which adapters are enabled
3. **Tries each enabled adapter** in sequence:
   - Alertmanager
   - PagerDuty
   - Grafana
   - Datadog
   - Opsgenie
   - VictorOps
   - New Relic
4. **First successful parse** wins
5. **Converts to internal format** and processes alert

**Performance:** Detection is fast. Each adapter validates specific JSON fields unique to that system.

## Adapter Structure

Each adapter is organized in its own package:

```
pkg/adapters/
├── alertmanager/
│   └── alertmanager.go      # Webhook types + adapter
├── pagerduty/
│   └── pagerduty.go         # Webhook types + adapter
├── grafana/
│   └── grafana.go           # Webhook types + adapter
├── datadog/
│   └── datadog.go           # Webhook types + adapter
├── opsgenie/
│   └── opsgenie.go          # Webhook types + adapter
├── victorops/
│   └── victorops.go         # Webhook types + adapter
├── newrelic/
│   └── newrelic.go          # Webhook types + adapter
└── registry.go              # Adapter registry + auto-detection
```

**Benefits:**
- Clean separation of concerns
- Easy to add new adapters
- Testable in isolation
- Clear ownership per alerting system

## Verification

### Check Enabled Adapters

View k8flex startup logs to see which adapters are enabled:

```bash
kubectl logs -n k8flex deployment/k8flex-agent | grep "Enabled adapters"

# Output examples:
# Enabled adapters: [alertmanager pagerduty grafana]
# No ENABLED_ADAPTERS set, all adapters enabled by default
```

### Test Webhook Detection

Send test webhook and check logs:

```bash
# Send test
curl -X POST http://localhost:8080/webhook \
  -H "Content-Type: application/json" \
  -d @examples/webhooks/alertmanager.json

# Check logs
kubectl logs -n k8flex deployment/k8flex-agent --tail=5

# Expected output:
# Received Alertmanager webhook with 1 alerts
```

### Test Disabled Adapter

If you send a webhook from a disabled adapter:

```bash
# ENABLED_ADAPTERS=alertmanager (Grafana disabled)

curl -X POST http://localhost:8080/webhook \
  -H "Content-Type: application/json" \
  -d @examples/webhooks/grafana.json

# Response: 400 Bad Request
# Logs: Failed to parse webhook from any enabled source
```

## Troubleshooting

### Adapter Not Detecting Webhook

**Problem:** k8flex returns 400 error

**Solutions:**

1. **Verify adapter is enabled:**
   ```bash
   kubectl get configmap k8flex-config -n k8flex -o jsonpath='{.data.ENABLED_ADAPTERS}'
   ```

2. **Check webhook format:**
   Compare your webhook with examples in `examples/webhooks/`

3. **Review logs:**
   ```bash
   kubectl logs -n k8flex deployment/k8flex-agent --tail=50
   ```

4. **Test with example payload:**
   ```bash
   curl -X POST http://localhost:8080/webhook \
     -H "Content-Type: application/json" \
     -d @examples/webhooks/<system>.json
   ```

### Wrong Adapter Detected

**Problem:** k8flex detects the wrong source

**Solutions:**

1. **Disable conflicting adapters:**
   ```bash
   # If Grafana is being detected as Alertmanager
   ENABLED_ADAPTERS=grafana
   ```

2. **Check webhook payload structure:**
   Ensure it matches the expected format for your system

3. **Report issue:**
   If detection is incorrect, this may be a bug. Check adapter validation logic in `pkg/adapters/<system>/<system>.go`

### Performance Concerns

**Problem:** Webhook processing seems slow

**Solutions:**

1. **Limit enabled adapters:**
   ```bash
   # Only enable what you use
   ENABLED_ADAPTERS=alertmanager
   ```

2. **Monitor detection attempts:**
   Each disabled adapter saves one JSON parsing attempt

3. **Check metrics:**
   ```bash
   kubectl logs -n k8flex deployment/k8flex-agent | grep "Received .* webhook"
   ```

## Security Considerations

### Limit Attack Surface

Enable only the adapters you actually use:

```bash
# Instead of accepting all webhook formats
ENABLED_ADAPTERS=alertmanager  # Only accept Alertmanager
```

**Benefits:**
- Reduces code paths that process external input
- Clearer audit trail of accepted sources
- Prevents accidental webhook acceptance from unauthorized systems

### Network Policies

Combine with Kubernetes NetworkPolicies:

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: k8flex-ingress
  namespace: k8flex
spec:
  podSelector:
    matchLabels:
      app: k8flex-agent
  policyTypes:
  - Ingress
  ingress:
  # Only allow from Alertmanager namespace
  - from:
    - namespaceSelector:
        matchLabels:
          name: monitoring
    ports:
    - protocol: TCP
      port: 8080
```

## Best Practices

1. **Be Explicit:** Set `ENABLED_ADAPTERS` even if you want all adapters
   ```bash
   ENABLED_ADAPTERS=alertmanager,pagerduty,grafana,datadog,opsgenie,victorops,newrelic
   ```

2. **Document Your Choice:** Comment why you enabled specific adapters
   ```yaml
   data:
     # We use Prometheus for infra and PagerDuty for on-call rotation
     ENABLED_ADAPTERS: "alertmanager,pagerduty"
   ```

3. **Test After Changes:** Verify webhooks still work after changing configuration
   ```bash
   ./examples/test-webhooks.sh
   ```

4. **Monitor Logs:** Watch for unexpected webhook sources
   ```bash
   kubectl logs -n k8flex deployment/k8flex-agent -f | grep "Received.*webhook"
   ```

5. **Version Control:** Track `ENABLED_ADAPTERS` in your deployment manifests

## Migration Path

### From Old to New Adapter System

If upgrading from a previous version that only supported Alertmanager:

1. **No action needed if using Alertmanager only:**
   - Old behavior is preserved
   - Alertmanager webhooks continue to work

2. **To add new systems:**
   ```bash
   # Add PagerDuty support
   ENABLED_ADAPTERS=alertmanager,pagerduty
   ```

3. **Configuration is backward compatible:**
   - Old deployments work without changes
   - New adapters are opt-in via configuration

## Reference

- Example webhooks: [`examples/webhooks/`](../examples/webhooks/)
- Integration guides: [`docs/INTEGRATION.md`](INTEGRATION.md)
- Test script: [`examples/test-webhooks.sh`](../examples/test-webhooks.sh)
- Adapter source code: [`pkg/adapters/`](../pkg/adapters/)
