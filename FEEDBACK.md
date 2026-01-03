# Feedback System

## Overview

K8flex includes an intelligent feedback system that learns from human evaluations to improve future incident analyses.

## Features

### 1. Automatic Reaction Detection

The system periodically checks Slack threads for emoji reactions on analysis messages:

- **✅ / :white_check_mark:** - Marks analysis as correct
- **❌ / :x:** - Marks analysis as incorrect

**How it works:**
- Checks every 30 seconds for new reactions
- Automatically records feedback when detected
- Sends confirmation message in thread
- Feedback is stored in `/data/feedback.json`
- Used to improve future analyses

### 2. Similar Incident Linking

When analyzing a new alert, if a similar incident was previously analyzed:
- The analysis includes a link to the past Slack thread
- Allows users to see historical context
- Learn from previous resolutions

**Example:**
```
=== PAST FEEDBACK ===
1. KubernetesPodOOMKilled (pod-crash): ✅ CORRECT - Root cause was memory limit too low...
   (See: https://your-workspace.slack.com/archives/C01234567/p1704211234567890)
```

## Configuration

### Required Slack Bot Scopes

Your Slack bot **MUST** have these scopes to enable feedback detection:

**Required Scopes:**
- `chat:write` - Post messages
- `chat:write.public` - Post to public channels
- `reactions:read` - **CRITICAL**: Read emoji reactions (for feedback detection)

**How to add scopes:**
1. Go to https://api.slack.com/apps
2. Select your app
3. Click "OAuth & Permissions" (left sidebar)
4. Scroll to "Bot Token Scopes"
5. Add `reactions:read` if missing
6. Click "Reinstall to Workspace" at the top
7. Copy the new bot token and update your deployment

### Required Environment Variables

```bash
# Slack Bot Token (required for reaction detection)
SLACK_BOT_TOKEN=xoxb-your-bot-token

# Slack Channel ID
SLACK_CHANNEL_ID=C01234567

# Slack Workspace ID (for building thread links)
SLACK_WORKSPACE_ID=T01234567
```

### Finding Your Workspace ID

Your workspace ID is in your Slack URL:
```
https://YOUR-WORKSPACE.slack.com
         ^^^^^^^^^^^^^^
```

Or run this Slack API call:
```bash
curl -H "Authorization: Bearer $SLACK_BOT_TOKEN" \
  https://slack.com/api/team.info
```

### Helm Configuration

```yaml
slack:
  botToken: "xoxb-your-bot-token"
  channelId: "C01234567"
  workspaceId: "T01234567"  # Optional but recommended for thread links
```

## Usage

### 1. React to Analysis Messages

After K8flex posts an analysis:
1. Read the analysis
2. React with ✅ if correct or ❌ if incorrect
3. System automatically detects and records feedback
4. Confirmation message posted in thread

### 2. View Past Incidents

When a similar alert occurs:
- Check the analysis for "PAST FEEDBACK" section
- Click the Slack link to see previous thread
- Review past resolution steps

### 3. Manual Feedback (Alternative)

If automatic detection doesn't work, use API endpoint:
```bash
curl -X POST http://k8flex:8080/api/feedback \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $WEBHOOK_AUTH_TOKEN" \
  -d '{
    "alert_name": "KubernetesPodOOMKilled",
    "category": "pod-crash",
    "is_correct": true
  }'
```

## Feedback Storage

### Location
- File: `/data/feedback.json`
- Persistent volume recommended

### Structure
```json
{
  "feedbacks": [
    {
      "timestamp": "2026-01-03T12:34:56Z",
      "alert_name": "KubernetesPodOOMKilled",
      "category": "pod-crash",
      "namespace": "production",
      "summary": "Pod was OOMKilled",
      "analysis": "Root cause: memory limit too low...",
      "is_correct": true,
      "slack_thread": "1704211234.567890",
      "labels": {
        "alertname": "KubernetesPodOOMKilled",
        "severity": "critical"
      }
    }
  ]
}
```

## Reaction Checker

### Behavior
- Runs every 30 seconds
- Checks all pending feedback (up to 24 hours old)
- Removes old entries automatically
- Minimal API calls (only when reactions exist)

### Logs
```
2026/01/03 12:34:56 Started reaction checker - polling every 30 seconds
2026/01/03 12:35:26 Recorded ✅ feedback for alert 'KubernetesPodOOMKilled' via reaction
```

## Best Practices

### For Users
1. **Be timely**: React within 24 hours while context is fresh
2. **Be honest**: Incorrect feedback is valuable for learning
3. **Check links**: Review similar past incidents before reacting
4. **Add context**: Reply in thread with additional details if needed

### For Admins
1. **Enable bot token**: Required for reaction detection
2. **Set workspace ID**: Enables thread linking
3. **Persistent storage**: Mount `/data` volume for feedback persistence
4. **Monitor logs**: Check for reaction checker errors

## Troubleshooting

### Reactions Not Detected
- **CHECK SCOPES FIRST**: Verify bot has `reactions:read` scope
  - Go to https://api.slack.com/apps → Your App → OAuth & Permissions
  - Under "Bot Token Scopes", ensure `reactions:read` is listed
  - If missing, add it and reinstall app to workspace
- **CHECK BOT IS IN CHANNEL**: Invite bot to your channel
  - In Slack, go to the channel
  - Type: `/invite @your-bot-name`
  - Or click channel name → Integrations → Add apps
- Check logs for "missing_scope" or "not_in_channel" errors
- Verify `SLACK_BOT_TOKEN` is set correctly
- Check logs for API errors

### Thread Links Not Working
- Set `SLACK_WORKSPACE_ID` environment variable
- Verify workspace ID format (starts with T)
- Check Slack URL structure

### Feedback Not Persisted
- Ensure `/data` volume is mounted
- Check file permissions on `/data/feedback.json`
- Verify JSON file format is valid

## Statistics

View feedback statistics via logs:
```bash
kubectl logs -n k8flex deployment/k8flex | grep -i feedback
```

Example output:
```
Including 1 past feedback example for learning
Recorded ✅ feedback for alert 'KubernetesPodOOMKilled' (category: pod-crash)
```

## Privacy & Security

- Feedback stored locally (not sent to external services)
- Contains alert metadata and analysis text
- No sensitive pod data included
- Stored in cluster only
- Can be deleted/reset anytime by clearing `/data/feedback.json`
