package slack

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/valentinpelus/k8flex/pkg/types"
)

// Client wraps the Slack API client
type Client struct {
	webhookURL  string
	botToken    string
	channelID   string
	workspaceID string // Added for building Slack links
	client      *http.Client
}

// NewClient creates a new Slack client
func NewClient(webhookURL, botToken, channelID string) *Client {
	return &Client{
		webhookURL: webhookURL,
		botToken:   botToken,
		channelID:  channelID,
		client:     &http.Client{},
	}
}

// IsConfigured checks if Slack notifications are configured
func (c *Client) IsConfigured() bool {
	return c.webhookURL != "" || (c.botToken != "" && c.channelID != "")
}

// HasBotToken checks if Bot token is configured for threading support
func (c *Client) HasBotToken() bool {
	return c.botToken != "" && c.channelID != ""
}

// SendAlert sends an alert to Slack and returns the thread timestamp
func (c *Client) SendAlert(alert types.Alert) (string, error) {
	if c.HasBotToken() {
		return c.sendAlertWithBot(alert)
	} else if c.webhookURL != "" {
		return c.sendAlertWithWebhook(alert)
	}
	return "", nil
}

// SendAnalysis sends the analysis to Slack as a threaded reply
func (c *Client) SendAnalysis(alert types.Alert, analysis string, threadTS string) error {
	if c.HasBotToken() && threadTS != "" {
		_, err := c.sendAnalysisWithBot(alert, analysis, threadTS)
		return err
	} else if c.webhookURL != "" {
		return c.sendAnalysisWithWebhook(alert, analysis, threadTS)
	}
	return nil
}

// SendAnalysisInThread sends analysis in a thread and returns the message timestamp for updates
func (c *Client) SendAnalysisInThread(alert types.Alert, analysis string, threadTS string) (string, error) {
	if c.HasBotToken() && threadTS != "" {
		return c.sendAnalysisWithBot(alert, analysis, threadTS)
	} else if c.webhookURL != "" {
		err := c.sendAnalysisWithWebhook(alert, analysis, threadTS)
		return "", err
	}
	return "", nil
}

// sendAlertWithBot sends an alert using the Slack Bot token API
// Reference: https://api.slack.com/methods/chat.postMessage
func (c *Client) sendAlertWithBot(alert types.Alert) (string, error) {
	severity := alert.Labels["severity"]
	message := c.buildAlertMessage(alert, severity)
	message.Channel = c.channelID

	return c.postMessage(message)
}

// sendAlertWithWebhook sends an alert using Slack incoming webhook
func (c *Client) sendAlertWithWebhook(alert types.Alert) (string, error) {
	severity := alert.Labels["severity"]
	message := c.buildAlertMessage(alert, severity)

	jsonData, err := json.Marshal(message)
	if err != nil {
		return "", fmt.Errorf("failed to marshal Slack message: %w", err)
	}

	resp, err := c.client.Post(c.webhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to send to Slack: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Slack API returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Warning: Could not read Slack response: %v", err)
		return "", nil
	}

	// Incoming webhooks typically just return "ok"
	if strings.TrimSpace(string(body)) == "ok" {
		log.Printf("Alert sent to Slack successfully")
		return "", nil
	}

	var slackResp types.SlackResponse
	if err := json.Unmarshal(body, &slackResp); err != nil {
		log.Printf("Warning: Could not parse Slack response: %v", err)
		return "", nil
	}

	if !slackResp.OK {
		return "", fmt.Errorf("Slack error: %s", slackResp.Error)
	}

	return slackResp.TS, nil
}

// sendAnalysisWithBot sends analysis using the Slack Bot token API and returns message timestamp
func (c *Client) sendAnalysisWithBot(_ types.Alert, analysis string, threadTS string) (string, error) {
	message := types.SlackMessage{
		Channel:     c.channelID,
		ThreadTS:    threadTS,
		UnfurlLinks: false,
		Blocks: []types.SlackBlock{
			{
				Type: "section",
				Text: &types.SlackTextObject{
					Type: "mrkdwn",
					Text: "*🔍 AI Debug Analysis*",
				},
			},
			{
				Type: "divider",
			},
			{
				Type: "section",
				Text: &types.SlackTextObject{
					Type: "mrkdwn",
					Text: truncateForSlack(ConvertMarkdownToSlack(analysis), 2900),
				},
			},
		},
	}

	return c.postMessage(message)
}

// sendAnalysisWithWebhook sends analysis using Slack incoming webhook
func (c *Client) sendAnalysisWithWebhook(alert types.Alert, analysis string, threadTS string) error {
	message := types.SlackMessage{
		UnfurlLinks: false,
		Blocks: []types.SlackBlock{
			{
				Type: "section",
				Text: &types.SlackTextObject{
					Type: "mrkdwn",
					Text: fmt.Sprintf("*🔍 AI Debug Analysis Complete*\nAlert: `%s`", alert.Labels["alertname"]),
				},
			},
			{
				Type: "divider",
			},
			{
				Type: "section",
				Text: &types.SlackTextObject{
					Type: "mrkdwn",
					Text: fmt.Sprintf("```\n%s\n```", truncateForSlack(analysis, 2900)),
				},
			},
		},
	}

	if threadTS != "" {
		message.ThreadTS = threadTS
	}

	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal Slack message: %w", err)
	}

	resp, err := c.client.Post(c.webhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send to Slack: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Slack API returned status %d: %s", resp.StatusCode, string(body))
	}

	log.Printf("Analysis sent to Slack for alert: %s", alert.Labels["alertname"])
	return nil
}

// postMessage sends a message using the Slack chat.postMessage API
func (c *Client) postMessage(message types.SlackMessage) (string, error) {
	jsonData, err := json.Marshal(message)
	if err != nil {
		return "", fmt.Errorf("failed to marshal Slack message: %w", err)
	}

	req, err := http.NewRequest("POST", "https://slack.com/api/chat.postMessage", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.botToken))

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send to Slack: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var slackResp types.SlackResponse
	if err := json.Unmarshal(body, &slackResp); err != nil {
		return "", fmt.Errorf("failed to parse Slack response: %w", err)
	}

	if !slackResp.OK {
		return "", fmt.Errorf("Slack error: %s", slackResp.Error)
	}

	log.Printf("Message sent to Slack, thread_ts: %s", slackResp.TS)
	return slackResp.TS, nil
}

// buildAlertMessage creates a Slack message with blocks for an alert
func (c *Client) buildAlertMessage(alert types.Alert, severity string) types.SlackMessage {
	message := types.SlackMessage{
		UnfurlLinks: false,
		Blocks: []types.SlackBlock{
			{
				Type: "header",
				Text: &types.SlackTextObject{
					Type: "plain_text",
					Text: fmt.Sprintf("🚨 %s", alert.Labels["alertname"]),
				},
			},
			{
				Type: "section",
				Fields: []types.SlackTextObject{
					{
						Type: "mrkdwn",
						Text: fmt.Sprintf("*Severity:*\n%s", severity),
					},
					{
						Type: "mrkdwn",
						Text: fmt.Sprintf("*Namespace:*\n%s", alert.Labels["namespace"]),
					},
				},
			},
		},
	}

	// Add pod info if available
	if pod := alert.Labels["pod"]; pod != "" {
		message.Blocks = append(message.Blocks, types.SlackBlock{
			Type: "section",
			Fields: []types.SlackTextObject{
				{
					Type: "mrkdwn",
					Text: fmt.Sprintf("*Pod:*\n`%s`", pod),
				},
				{
					Type: "mrkdwn",
					Text: fmt.Sprintf("*Service:*\n%s", alert.Labels["service"]),
				},
			},
		})
	}

	// Add annotations
	if summary := alert.Annotations["summary"]; summary != "" {
		message.Blocks = append(message.Blocks, types.SlackBlock{
			Type: "section",
			Text: &types.SlackTextObject{
				Type: "mrkdwn",
				Text: fmt.Sprintf("*Summary:*\n%s", summary),
			},
		})
	}

	if description := alert.Annotations["description"]; description != "" {
		message.Blocks = append(message.Blocks, types.SlackBlock{
			Type: "section",
			Text: &types.SlackTextObject{
				Type: "mrkdwn",
				Text: fmt.Sprintf("*Description:*\n%s", description),
			},
		})
	}

	// Add timestamp
	message.Blocks = append(message.Blocks, types.SlackBlock{
		Type: "context",
		Elements: []types.SlackTextObject{
			{
				Type: "mrkdwn",
				Text: fmt.Sprintf("Started: %s", alert.StartsAt.Format("2006-01-02 15:04:05 MST")),
			},
		},
	})

	// Add divider and status message
	message.Blocks = append(message.Blocks,
		types.SlackBlock{Type: "divider"},
		types.SlackBlock{
			Type: "section",
			Text: &types.SlackTextObject{
				Type: "mrkdwn",
				Text: "🤖 _AI debugging in progress..._",
			},
		},
	)

	return message
}

// truncateForSlack truncates text to fit within Slack message limits
func truncateForSlack(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "\n... (truncated)"
}

// ConvertMarkdownToSlack converts standard Markdown to Slack's mrkdwn format
func ConvertMarkdownToSlack(text string) string {
	// Convert **bold** to *bold*
	text = strings.ReplaceAll(text, "**", "*")
	// Convert numbered lists with bold headers (e.g., "1. **Action:**" to "1. *Action:*")
	// Already handled by the above replacement
	return text
}

// UpdateMessage updates an existing Slack message (requires Bot token)
func (c *Client) UpdateMessage(messageTS, newText string) error {
	if !c.HasBotToken() {
		return fmt.Errorf("Bot token required for message updates")
	}

	updatePayload := map[string]interface{}{
		"channel": c.channelID,
		"ts":      messageTS,
		"text":    truncateForSlack(ConvertMarkdownToSlack(newText), 3000),
	}

	jsonData, err := json.Marshal(updatePayload)
	if err != nil {
		return fmt.Errorf("failed to marshal update payload: %w", err)
	}

	req, err := http.NewRequest("POST", "https://slack.com/api/chat.update", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.botToken)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to update message: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var slackResp types.SlackResponse
	if err := json.Unmarshal(body, &slackResp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if !slackResp.OK {
		return fmt.Errorf("Slack error: %s", slackResp.Error)
	}

	return nil
}

// GetMessageReactions retrieves reactions on a specific message
func (c *Client) GetMessageReactions(messageTS string) ([]string, error) {
	if !c.HasBotToken() {
		return nil, fmt.Errorf("Bot token required for getting reactions")
	}

	url := fmt.Sprintf("https://slack.com/api/reactions.get?channel=%s&timestamp=%s", c.channelID, messageTS)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.botToken)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get reactions: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result struct {
		OK      bool   `json:"ok"`
		Error   string `json:"error,omitempty"`
		Message struct {
			Reactions []struct {
				Name string `json:"name"`
			} `json:"reactions"`
		} `json:"message"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if !result.OK {
		// If error is "message_not_found" or "no_reaction", return empty list
		if result.Error == "message_not_found" || result.Error == "no_reaction" {
			return []string{}, nil
		}
		// Special handling for not_in_channel error
		if result.Error == "not_in_channel" {
			return nil, fmt.Errorf("bot not in channel - invite bot to channel with: /invite @bot-name (channel: %s)", c.channelID)
		}
		return nil, fmt.Errorf("Slack error: %s (channel: %s, ts: %s)", result.Error, c.channelID, messageTS)
	}

	reactions := make([]string, 0, len(result.Message.Reactions))
	for _, r := range result.Message.Reactions {
		reactions = append(reactions, r.Name)
	}

	return reactions, nil
}

// ReplyToThread sends a message as a reply in a thread
func (c *Client) ReplyToThread(threadTS, text string) error {
	if !c.HasBotToken() {
		return fmt.Errorf("Bot token required for thread replies")
	}

	message := types.SlackMessage{
		Channel:  c.channelID,
		ThreadTS: threadTS,
		Text:     text,
	}

	_, err := c.postMessage(message)
	return err
}

// GetChannelID returns the configured channel ID
func (c *Client) GetChannelID() string {
	return c.channelID
}

// GetWorkspaceID returns the workspace ID (extracted from team info or set manually)
func (c *Client) GetWorkspaceID() string {
	// If not set, try to extract from first API call or return empty
	// For now, this needs to be configured via environment variable
	return c.workspaceID
}

// SetWorkspaceID sets the workspace ID for building Slack links
func (c *Client) SetWorkspaceID(workspaceID string) {
	c.workspaceID = workspaceID
}

// ValidateScopes checks if the bot token has required scopes for feedback detection
func (c *Client) ValidateScopes() error {
	if !c.HasBotToken() {
		return fmt.Errorf("bot token not configured")
	}

	// Call auth.test to verify token and get bot info
	req, err := http.NewRequest("GET", "https://slack.com/api/auth.test", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.botToken)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to validate token: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result struct {
		OK    bool   `json:"ok"`
		Error string `json:"error,omitempty"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if !result.OK {
		return fmt.Errorf("token validation failed: %s", result.Error)
	}

	// Try a test call to reactions.get to check if scope exists
	// We use a fake timestamp, expecting either success or message_not_found (which means scope is OK)
	testURL := fmt.Sprintf("https://slack.com/api/reactions.get?channel=%s&timestamp=0000000000.000000", c.channelID)
	req, _ = http.NewRequest("GET", testURL, nil)
	req.Header.Set("Authorization", "Bearer "+c.botToken)

	resp, err = c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to check scopes: %w", err)
	}
	defer resp.Body.Close()

	body, _ = io.ReadAll(resp.Body)

	var scopeResult struct {
		OK    bool   `json:"ok"`
		Error string `json:"error,omitempty"`
	}

	if err := json.Unmarshal(body, &scopeResult); err != nil {
		return fmt.Errorf("failed to parse scope check: %w", err)
	}

	// If error is "missing_scope", the bot doesn't have reactions:read
	if scopeResult.Error == "missing_scope" {
		return fmt.Errorf("missing required scope 'reactions:read' - add it at https://api.slack.com/apps")
	}

	// Other errors like "message_not_found" are OK - it means the scope exists
	return nil
}
