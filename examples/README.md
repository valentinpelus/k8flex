# K8flex Webhook Examples

This directory contains example webhook payloads for all supported alerting systems and a test script to verify integrations.

## Supported Alerting Systems

- **Alertmanager** (Prometheus) - `webhooks/alertmanager.json`
- **PagerDuty** - `webhooks/pagerduty.json`
- **Grafana** - `webhooks/grafana.json`
- **Datadog** - `webhooks/datadog.json`
- **Opsgenie** - `webhooks/opsgenie.json`
- **VictorOps** (Splunk On-Call) - `webhooks/victorops.json`
- **New Relic** - `webhooks/newrelic.json`

## Quick Test

Test all webhook integrations:

```bash
# Set k8flex URL (default: http://localhost:8080/webhook)
export K8FLEX_URL=http://k8flex-agent.k8flex.svc.cluster.local:8080/webhook

# Run test script
./test-webhooks.sh
```

## Test Individual Webhooks

### Alertmanager

```bash
curl -X POST http://localhost:8080/webhook \
  -H "Content-Type: application/json" \
  -d @webhooks/alertmanager.json
```

### PagerDuty

```bash
curl -X POST http://localhost:8080/webhook \
  -H "Content-Type: application/json" \
  -d @webhooks/pagerduty.json
```

### Grafana

```bash
curl -X POST http://localhost:8080/webhook \
  -H "Content-Type: application/json" \
  -d @webhooks/grafana.json
```

### Datadog

```bash
curl -X POST http://localhost:8080/webhook \
  -H "Content-Type: application/json" \
  -d @webhooks/datadog.json
```

### Opsgenie

```bash
curl -X POST http://localhost:8080/webhook \
  -H "Content-Type: application/json" \
  -d @webhooks/opsgenie.json
```

### VictorOps

```bash
curl -X POST http://localhost:8080/webhook \
  -H "Content-Type: application/json" \
  -d @webhooks/victorops.json
```

### New Relic

```bash
curl -X POST http://localhost:8080/webhook \
  -H "Content-Type: application/json" \
  -d @webhooks/newrelic.json
```

## Configuring Enabled Adapters

By default, k8flex supports all alerting systems. To limit which systems are enabled, set the `ENABLED_ADAPTERS` environment variable:

### Enable Only Specific Adapters

```bash
# Only Alertmanager and Grafana
export ENABLED_ADAPTERS=alertmanager,grafana

# Only PagerDuty
export ENABLED_ADAPTERS=pagerduty

# Multiple systems
export ENABLED_ADAPTERS=alertmanager,pagerduty,grafana,datadog
```

### In Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: k8flex-agent
  namespace: k8flex
spec:
  template:
    spec:
      containers:
      - name: k8flex-agent
        image: k8flex-agent:latest
        env:
        - name: ENABLED_ADAPTERS
          value: "alertmanager,pagerduty,grafana"
```

### In Docker

```bash
docker run -e ENABLED_ADAPTERS=alertmanager,grafana k8flex-agent:latest
```

## Adapter Names

Use these exact names (case-insensitive) in `ENABLED_ADAPTERS`:

- `alertmanager`
- `pagerduty`
- `grafana`
- `datadog`
- `opsgenie`
- `victorops`
- `newrelic`

## Verifying Logs

After sending a webhook, check k8flex logs to see which adapter was detected:

```bash
kubectl logs -n k8flex deployment/k8flex-agent --tail=20

# Expected output:
# Received Alertmanager webhook with 1 alerts
# Received PagerDuty webhook with 1 alerts
# etc.
```

## Customizing Payloads

Each JSON file contains a realistic example with Kubernetes metadata. Customize the following fields for your use case:

### Critical Fields for k8flex

- `namespace`: Kubernetes namespace
- `pod`: Pod name
- `service`: Service name (optional)
- `container`: Container name (optional)

These fields must be present in the webhook payload for k8flex to properly debug the issue. Each adapter extracts these from different locations:

| System | Namespace Location | Pod Location | Service Location |
|--------|-------------------|--------------|------------------|
| Alertmanager | `labels.namespace` | `labels.pod` | `labels.service` |
| PagerDuty | `custom_details.namespace` | `custom_details.pod` | `custom_details.service` |
| Grafana | `labels.namespace` | `labels.pod` | `labels.service` |
| Datadog | `tags` (namespace:value) | `tags` (pod_name:value) | `tags` (kube_service:value) |
| Opsgenie | `details.namespace` | `details.pod` | `details.service` |
| VictorOps | Parsed from `entity_id` | Parsed from `entity_id` / `state_message` | N/A |
| New Relic | `targets[].labels.namespace` | Parsed from `targets[].name` | N/A |

## Troubleshooting

### Webhook Not Detected

If k8flex cannot detect your webhook source:

1. Verify the adapter is enabled in `ENABLED_ADAPTERS`
2. Check the JSON structure matches the expected format
3. Review k8flex logs for detailed error messages
4. Compare your payload with the example in this directory

### Missing Kubernetes Metadata

If k8flex cannot extract namespace/pod information:

1. Ensure your alerting system includes these fields in the webhook
2. Check the adapter code in `pkg/adapters/<system>/<system>.go`
3. Add custom labels/tags to your alerts in the alerting system configuration

### Multiple Alerting Systems

K8flex can handle webhooks from multiple systems simultaneously. The adapter is automatically detected based on the JSON structure.

## Contributing

To add support for a new alerting system:

1. Create `pkg/adapters/<system>/<system>.go` with webhook types and adapter
2. Update `pkg/adapters/registry.go` to include the new adapter
3. Add example payload to `examples/webhooks/<system>.json`
4. Update this README with integration instructions
