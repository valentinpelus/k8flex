# Integration Guide

K8flex supports webhooks from multiple popular alerting systems. Auto-detection identifies the source and converts alerts to the internal format automatically.

## Supported Alerting Systems

K8flex supports the following alerting systems out of the box:

- **Alertmanager** (Prometheus)
- **PagerDuty**
- **Grafana**
- **Datadog**
- **Opsgenie**
- **VictorOps** (Splunk On-Call)
- **New Relic**

### Configuring Enabled Adapters

By default, **all adapters are enabled**. To limit which alerting systems k8flex accepts, set the `ENABLED_ADAPTERS` environment variable:

```bash
# Enable only specific systems (comma-separated, case-insensitive)
export ENABLED_ADAPTERS=alertmanager,pagerduty,grafana

# In Kubernetes
kubectl set env deployment/k8flex-agent -n k8flex ENABLED_ADAPTERS=alertmanager,grafana

# In deployment.yaml
data:
  ENABLED_ADAPTERS: "alertmanager,pagerduty,grafana,datadog"
```

**Available adapter names:**
- `alertmanager`
- `pagerduty`
- `grafana`
- `datadog`
- `opsgenie`
- `victorops`
- `newrelic`

## Alerting System Setup

Each alerting system requires configuration to send webhooks to k8flex. Below are integration guides for each system.

## Alertmanager Integration

### Step 1: Update Alertmanager Configuration

Edit your Alertmanager ConfigMap to add the k8flex webhook receiver:

```bash
kubectl edit configmap alertmanager -n monitoring
```

Add this configuration:

```yaml
receivers:
  - name: 'k8flex-ai-debug'
    webhook_configs:
      - url: 'http://k8flex-agent.k8flex.svc.cluster.local:8080/webhook'
        send_resolved: false
        max_alerts: 10

route:
  receiver: 'default'
  group_by: ['alertname', 'cluster', 'service']
  group_wait: 10s
  group_interval: 10s
  repeat_interval: 12h
  routes:
    # Critical alerts go to AI debug
    - match:
        severity: critical
      receiver: 'k8flex-ai-debug'
      continue: true
    
    # Pod-related alerts
    - match_re:
        alertname: '(Pod.*|Container.*|Readiness.*|Liveness.*)'
      receiver: 'k8flex-ai-debug'
      continue: true
```

### Step 2: Reload Alertmanager

```bash
kubectl rollout restart statefulset alertmanager -n monitoring
```

### Step 3: Test the Integration

Send a test alert:

```bash
curl -XPOST 'http://localhost:9093/api/v2/alerts' \
  -H 'Content-Type: application/json' \
  -d '[
    {
      "labels": {
        "alertname": "TestPodIssue",
        "service": "test-service",
        "severity": "critical",
        "namespace": "default",
        "pod": "test-pod-abc123"
      },
      "annotations": {
        "summary": "Test alert for AI debugging",
        "description": "This is a test alert to verify k8flex integration"
      }
    }
  ]'
```

## Prometheus Rules for K8flex

Create alerts that provide all necessary labels for debugging:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: prometheus-k8flex-rules
  namespace: monitoring
data:
  k8flex-rules.yaml: |
    groups:
      - name: k8flex-pod-alerts
        interval: 30s
        rules:
          # Pod not ready
          - alert: PodNotReady
            expr: kube_pod_status_phase{phase!="Running",phase!="Succeeded"} == 1
            for: 5m
            labels:
              severity: warning
              namespace: "{{ $labels.namespace }}"
              pod: "{{ $labels.pod }}"
            annotations:
              summary: "Pod {{ $labels.namespace }}/{{ $labels.pod }} is not ready"
              description: "Pod has been in {{ $labels.phase }} state for more than 5 minutes"
          
          # Container restarts
          - alert: PodCrashLooping
            expr: rate(kube_pod_container_status_restarts_total[15m]) > 0
            for: 5m
            labels:
              severity: critical
              namespace: "{{ $labels.namespace }}"
              pod: "{{ $labels.pod }}"
              container: "{{ $labels.container }}"
            annotations:
              summary: "Pod {{ $labels.namespace }}/{{ $labels.pod }} is crash looping"
              description: "Container {{ $labels.container }} has restarted {{ $value }} times"
          
          # Pod pending
          - alert: PodPending
            expr: kube_pod_status_phase{phase="Pending"} == 1
            for: 10m
            labels:
              severity: warning
              namespace: "{{ $labels.namespace }}"
              pod: "{{ $labels.pod }}"
            annotations:
              summary: "Pod {{ $labels.namespace }}/{{ $labels.pod }} is pending"
              description: "Pod has been in Pending state for more than 10 minutes"
          
          # Container OOMKilled
          - alert: PodOOMKilled
            expr: kube_pod_container_status_last_terminated_reason{reason="OOMKilled"} == 1
            labels:
              severity: critical
              namespace: "{{ $labels.namespace }}"
              pod: "{{ $labels.pod }}"
              container: "{{ $labels.container }}"
            annotations:
              summary: "Container {{ $labels.container }} was OOMKilled"
              description: "Container in pod {{ $labels.namespace }}/{{ $labels.pod }} was killed due to out of memory"
          
          # Service has no endpoints
          - alert: ServiceNoEndpoints
            expr: kube_endpoint_address_available{endpoint!=""} == 0
            for: 5m
            labels:
              severity: warning
              namespace: "{{ $labels.namespace }}"
              service: "{{ $labels.endpoint }}"
            annotations:
              summary: "Service {{ $labels.namespace }}/{{ $labels.endpoint }} has no endpoints"
              description: "Service has had no available endpoints for more than 5 minutes"
```

## PagerDuty Integration

### Step 1: Configure Webhook in PagerDuty

1. Go to **Services** > Select your service > **Integrations** tab
2. Click **Add Integration** > **Generic Webhooks (v3)**
3. Enter webhook URL: `http://k8flex-agent.k8flex.svc.cluster.local:8080/webhook`
4. Select **Trigger** events you want to send
5. Save integration

### Step 2: Test Integration

Trigger a test incident in PagerDuty and verify k8flex receives it in the logs:

```bash
kubectl logs -n k8flex deployment/k8flex-agent | grep "PagerDuty"
```

**Note**: K8flex extracts Kubernetes metadata from PagerDuty custom details (namespace, pod, service).

---

## Grafana Integration

### Step 1: Configure Contact Point

1. Go to **Alerting** > **Contact points**
2. Click **New contact point**
3. Select **webhook** type
4. URL: `http://k8flex-agent.k8flex.svc.cluster.local:8080/webhook`
5. HTTP Method: `POST`
6. Save

### Step 2: Add to Notification Policy

Edit your notification policy to route alerts to the k8flex contact point.

### Step 3: Test

Send test notification from contact point configuration.

```bash
kubectl logs -n k8flex deployment/k8flex-agent | grep "Grafana"
```

---

## Datadog Integration

### Step 1: Create Webhook Integration

1. Go to **Integrations** > **Webhooks**
2. Click **New Webhook**
3. Name: `k8flex-debug`
4. URL: `http://k8flex-agent.k8flex.svc.cluster.local:8080/webhook`
5. Payload: Leave default (JSON)
6. Custom Headers: None required
7. Save

### Step 2: Configure Monitor Notifications

In your monitor configuration, add:

```
@webhook-k8flex-debug
```

### Step 3: Verify

```bash
kubectl logs -n k8flex deployment/k8flex-agent | grep "Datadog"
```

**Note**: K8flex extracts Kubernetes labels from Datadog tags (namespace, pod_name, kube_service).

---

## Opsgenie Integration

### Step 1: Create Outgoing Webhook

1. Go to **Settings** > **Integrations**
2. Click **Add integration** > **Outgoing Webhooks**
3. Name: `k8flex-debug`
4. Webhook URL: `http://k8flex-agent.k8flex.svc.cluster.local:8080/webhook`
5. Add conditions (e.g., Priority is Critical)
6. Enable **Alert Created** trigger
7. Save integration

### Step 2: Test

Create or trigger an alert matching your conditions.

```bash
kubectl logs -n k8flex deployment/k8flex-agent | grep "Opsgenie"
```

---

## VictorOps (Splunk On-Call) Integration

### Step 1: Configure Outbound Webhook

1. Go to **Settings** > **Outgoing Webhooks**
2. Click **Add Webhook**
3. Event: **Incident Triggered**
4. To: `http://k8flex-agent.k8flex.svc.cluster.local:8080/webhook`
5. Method: `POST`
6. Content Type: `application/json`
7. Save

### Step 2: Test

Trigger an incident and verify:

```bash
kubectl logs -n k8flex deployment/k8flex-agent | grep "VictorOps"
```

---

## New Relic Integration

### Step 1: Configure Webhook Notification Channel

1. Go to **Alerts & AI** > **Notification channels**
2. Click **New notification channel**
3. Channel type: **Webhook**
4. Channel name: `k8flex-debug`
5. Base URL: `http://k8flex-agent.k8flex.svc.cluster.local:8080/webhook`
6. Custom payload: Leave default
7. Save

### Step 2: Add to Alert Policy

Edit your alert policy and add the k8flex webhook channel.

### Step 3: Test

```bash
kubectl logs -n k8flex deployment/k8flex-agent | grep "New Relic"
```

---

## Multi-System Setup

You can integrate multiple alerting systems simultaneously. K8flex automatically detects the source and processes alerts accordingly.

**Example**: Use Alertmanager for infrastructure alerts, PagerDuty for on-call, and Datadog for application metrics - all feeding into the same k8flex instance.

### Kubernetes Label Requirements

For optimal analysis, ensure your alerts include these Kubernetes labels:

- `namespace`: Pod namespace
- `pod` or `pod_name`: Pod name
- `service` or `kube_service`: Service name (optional)
- `container`: Container name (optional)

Each alerting system adapter extracts these from different fields:

| System | Namespace | Pod | Service |
|--------|-----------|-----|---------|
| Alertmanager | `labels.namespace` | `labels.pod` | `labels.service` |
| PagerDuty | `custom_details.namespace` | `custom_details.pod` | `custom_details.service` |
| Grafana | `labels.namespace` | `labels.pod` | `labels.service` |
| Datadog | `tags.namespace` | `tags.pod_name` | `tags.kube_service` |
| Opsgenie | `details.namespace` | `details.pod` | `details.service` |
| VictorOps | `entity_id` | `state_message` | `entity_display_name` |
| New Relic | `entity.name` | (extracted) | (extracted) |

---

## Ollama Setup

### Deploy Ollama with Persistent Storage

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: ollama-data
  namespace: ollama
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 20Gi
---
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
          name: http
        volumeMounts:
        - name: data
          mountPath: /root/.ollama
        resources:
          requests:
            memory: "4Gi"
            cpu: "2"
          limits:
            memory: "8Gi"
            cpu: "4"
        livenessProbe:
          httpGet:
            path: /
            port: http
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /
            port: http
          initialDelaySeconds: 5
          periodSeconds: 5
      volumes:
      - name: data
        persistentVolumeClaim:
          claimName: ollama-data
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
    targetPort: http
    name: http
```

### Pull Models

```bash
# Pull llama2 (default)
kubectl exec -n ollama deployment/ollama -- ollama pull llama2

# Or use a smaller model for faster responses
kubectl exec -n ollama deployment/ollama -- ollama pull mistral

# Or use a larger model for better analysis
kubectl exec -n ollama deployment/ollama -- ollama pull llama2:70b
```

Update k8flex ConfigMap to use different model:

```bash
kubectl patch configmap k8flex-config -n k8flex --type merge -p '{"data":{"OLLAMA_MODEL":"mistral"}}'
kubectl rollout restart deployment k8flex-agent -n k8flex
```

## Monitoring K8flex

### View Logs

```bash
# Follow logs
kubectl logs -n k8flex deployment/k8flex-agent -f

# View recent logs
kubectl logs -n k8flex deployment/k8flex-agent --tail=100

# Logs from specific time
kubectl logs -n k8flex deployment/k8flex-agent --since=1h
```

### Metrics

Add ServiceMonitor for Prometheus (optional enhancement):

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: k8flex-agent
  namespace: k8flex
spec:
  selector:
    matchLabels:
      app: k8flex-agent
  endpoints:
  - port: http
    path: /metrics
    interval: 30s
```

## Troubleshooting

### K8flex agent not receiving alerts

1. Check Alertmanager configuration:
```bash
kubectl get configmap alertmanager -n monitoring -o yaml
```

2. Verify webhook URL is correct:
```bash
kubectl run -it --rm debug --image=curlimages/curl --restart=Never -- \
  curl http://k8flex-agent.k8flex.svc.cluster.local:8080/health
```

3. Check Alertmanager logs:
```bash
kubectl logs -n monitoring alertmanager-0 | grep k8flex
```

### K8flex cannot connect to Ollama

1. Verify Ollama is running:
```bash
kubectl get pods -n ollama
```

2. Test connectivity:
```bash
kubectl exec -n k8flex deployment/k8flex-agent -- \
  wget -qO- http://ollama.ollama.svc.cluster.local:11434/api/version
```

3. Check Ollama logs:
```bash
kubectl logs -n ollama deployment/ollama
```

### Permission errors

Verify RBAC is correctly configured:

```bash
kubectl auth can-i get pods --as=system:serviceaccount:k8flex:k8flex-agent -n default
kubectl auth can-i get pods/log --as=system:serviceaccount:k8flex:k8flex-agent -n default
```

## Advanced Configuration

### Using Different Models for Different Alert Severities

You can deploy multiple k8flex instances with different models:

```yaml
# Fast model for warnings
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: k8flex-agent-fast
  namespace: k8flex
spec:
  # ... same as main deployment but with:
  env:
    - name: OLLAMA_MODEL
      value: "mistral"
---
# Detailed model for critical alerts
apiVersion: apps/v1
kind: Deployment
metadata:
  name: k8flex-agent-detailed
  namespace: k8flex
spec:
  # ... with:
  env:
    - name: OLLAMA_MODEL
      value: "llama2:70b"
```

Update Alertmanager routes:

```yaml
routes:
  - match:
      severity: critical
    receiver: 'k8flex-detailed'
  - match:
      severity: warning
    receiver: 'k8flex-fast'
```

### Custom AI Prompts

Modify the `analyzeWithOllama` function in [main.go](main.go) to customize the AI analysis prompt for your specific needs.

## Performance Tuning

### For High Alert Volume

Increase replicas and resources:

```bash
kubectl scale deployment k8flex-agent -n k8flex --replicas=3
kubectl set resources deployment k8flex-agent -n k8flex \
  --limits=cpu=1,memory=1Gi \
  --requests=cpu=500m,memory=512Mi
```

### Optimize Log Collection

Reduce log tail lines in main.go:

```go
tailLines := int64(50)  // Instead of 100
```

## Security Best Practices

1. **Network Policies**: Restrict k8flex network access
2. **RBAC**: Already configured with minimal permissions
3. **Secrets**: If adding external integrations, use Kubernetes Secrets
4. **Audit**: Enable audit logging for k8flex actions

## Next Steps

- Set up persistent storage for analysis results
- Add webhook notifications (Slack, PagerDuty)
- Implement caching to reduce K8s API calls
- Add metrics and dashboards
- Create custom AI models trained on your infrastructure
