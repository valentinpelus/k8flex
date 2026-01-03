# Integration Guide

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
