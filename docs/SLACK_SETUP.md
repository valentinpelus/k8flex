# Slack Integration Setup Guide

This guide explains how to configure Slack notifications for k8flex, including posting alerts and AI analysis as threaded messages.

## Features

- 🚨 **Alert Notifications**: Immediate alert posting to Slack channel
- 🧵 **Threaded Analysis**: AI debug analysis posted as thread reply
- 🎨 **Rich Formatting**: Color-coded severity, structured blocks
- 📊 **Detailed Info**: Pod, namespace, service details in alert

## Prerequisites

- Slack workspace with admin access
- Kubernetes cluster with k8flex deployed

## Step 1: Create Slack Webhook

### Option A: Incoming Webhooks (Simpler, but no threading)

1. Go to your Slack workspace settings
2. Navigate to **Apps** → **Manage Apps**
3. Search for "Incoming Webhooks" and click **Add to Slack**
4. Select the channel where you want alerts (e.g., `#k8s-alerts`)
5. Click **Add Incoming Webhooks Integration**
6. Copy the **Webhook URL** (looks like: `https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXX`)

**Note**: Incoming webhooks don't support threading. Analysis will be posted as separate messages.

### Option B: Slack App with Bot Token (Full Features + Threading)

For full threading support, create a Slack App:

1. Go to https://api.slack.com/apps
2. Click **Create New App** → **From scratch**
3. Name it "K8flex Debug Agent" and select your workspace
4. In **OAuth & Permissions**:
   - Add these Bot Token Scopes:
     - `chat:write` - Post messages
     - `chat:write.public` - Post to public channels
   - Click **Install to Workspace**
   - Copy the **Bot User OAuth Token** (starts with `xoxb-`)

5. Get your channel ID:
   - Open Slack, right-click your channel → **View channel details**
   - Copy the Channel ID from the bottom

6. For threaded replies, you'll need to use the Slack API instead of webhook URL

## Step 2: Configure K8flex

### Using Incoming Webhook (Recommended for Quick Setup)

```bash
# Create/update the secret with your Slack webhook URL
kubectl create secret generic k8flex-secrets \
  --from-literal=SLACK_WEBHOOK_URL='https://hooks.slack.com/services/YOUR/WEBHOOK/URL' \
  --namespace k8flex \
  --dry-run=client -o yaml | kubectl apply -f -
```

### Or edit the deployment directly:

```bash
kubectl edit secret k8flex-secrets -n k8flex
```

Add your webhook URL (base64 encoded):

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: k8flex-secrets
  namespace: k8flex
type: Opaque
stringData:
  SLACK_WEBHOOK_URL: "https://hooks.slack.com/services/YOUR/WEBHOOK/URL"
```

## Step 3: Restart K8flex

```bash
kubectl rollout restart deployment k8flex-agent -n k8flex
```

## Step 4: Test the Integration

Send a test alert:

```bash
curl -XPOST 'http://localhost:9093/api/v2/alerts' \
  -H 'Content-Type: application/json' \
  -d '[
    {
      "labels": {
        "alertname": "TestSlackIntegration",
        "severity": "warning",
        "namespace": "default",
        "pod": "test-pod",
        "service": "test-service"
      },
      "annotations": {
        "summary": "Testing Slack integration",
        "description": "This is a test to verify Slack notifications are working"
      }
    }
  ]'
```

You should see:
1. Initial alert message in your Slack channel
2. Follow-up message with AI analysis (threaded if using Bot token)

## Message Format

### Alert Message

The initial alert includes:
- 🚨 Alert name as header
- Severity (color-coded)
- Namespace and Pod information
- Service details
- Summary and description from annotations
- Timestamp
- "AI debugging in progress..." status

### Analysis Message

The AI analysis includes:
- 🔍 Header indicating analysis complete
- Complete debug findings
- Root cause analysis
- Recommended actions
- Formatted in code blocks for readability

## Verification

Check k8flex logs to verify Slack integration:

```bash
kubectl logs -n k8flex deployment/k8flex-agent -f
```

You should see:
```
Slack notifications: enabled
Alert sent to Slack successfully
Analysis sent to Slack for alert: TestSlackIntegration
```

## Troubleshooting

### Alerts not appearing in Slack

1. Verify webhook URL is correct:
```bash
kubectl get secret k8flex-secrets -n k8flex -o jsonpath='{.data.SLACK_WEBHOOK_URL}' | base64 -d
```

2. Test webhook manually:
```bash
WEBHOOK_URL=$(kubectl get secret k8flex-secrets -n k8flex -o jsonpath='{.data.SLACK_WEBHOOK_URL}' | base64 -d)

curl -X POST "$WEBHOOK_URL" \
  -H 'Content-Type: application/json' \
  -d '{"text":"Test from k8flex"}'
```

3. Check k8flex logs for errors:
```bash
kubectl logs -n k8flex deployment/k8flex-agent | grep -i slack
```

### Threading not working

Incoming webhooks don't support threading. You need to:
- Use a Slack App with Bot token
- Implement the `chat.postMessage` API endpoint

### Messages truncated

Long analyses are automatically truncated to fit Slack's 3000 character limit per block. Full details are always available in k8flex logs:

```bash
kubectl logs -n k8flex deployment/k8flex-agent | grep "COMPLETE ANALYSIS"
```

## Advanced Configuration

### Custom Message Format

Edit the `sendAlertToSlack` function in [main.go](main.go) to customize message formatting.

### Different Channels for Different Severities

Create multiple webhook URLs for different channels:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: k8flex-secrets
  namespace: k8flex
stringData:
  SLACK_WEBHOOK_URL: "https://hooks.slack.com/services/CRITICAL/CHANNEL"
  SLACK_WEBHOOK_URL_WARNING: "https://hooks.slack.com/services/WARNING/CHANNEL"
```

Then modify the code to use different URLs based on severity.

### Disable Slack for Specific Alerts

Alerts without certain labels can skip Slack notification by adding conditions in the code:

```go
if alert.Labels["notify_slack"] == "false" {
    return "", nil
}
```

## Example Output

### Alert Message:
```
🚨 PodCrashLooping

Severity: critical
Namespace: production

Pod: api-server-xyz-123
Service: api-server

Summary: Pod is crash looping
Description: Container has restarted 5 times in the last 10 minutes

Started: Jan 2, 2026 at 10:30 AM

🤖 AI debugging in progress...
```

### Analysis Reply:
```
🔍 AI Debug Analysis Complete
────────────────────────────────

Root Cause Analysis:
The pod is experiencing an OOMKill condition. Container memory usage 
exceeded the configured limit of 512Mi.

Evidence:
- Last terminated reason: OOMKilled
- Container restart count: 5
- Memory limit: 512Mi
- No memory request configured

Recommended Actions:
1. Increase memory limit to 1Gi
2. Add memory request of 768Mi
3. Review application memory usage patterns
4. Consider enabling memory profiling

Prevention:
- Set appropriate memory requests/limits
- Implement resource monitoring
- Enable horizontal pod autoscaling
```

## Security Best Practices

1. **Use Secrets**: Never commit webhook URLs to git
2. **Rotate URLs**: Regularly rotate webhook URLs
3. **Limit Permissions**: Use minimal Slack app permissions
4. **Monitor Usage**: Track webhook usage in Slack audit logs

## References

- [Slack Incoming Webhooks](https://api.slack.com/messaging/webhooks)
- [Slack Block Kit](https://api.slack.com/block-kit)
- [Slack Message Formatting](https://api.slack.com/reference/surfaces/formatting)
