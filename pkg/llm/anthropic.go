package llm

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

// AnthropicProvider implements the Provider interface for Anthropic's Claude models
type AnthropicProvider struct {
	apiKey string
	model  string
	client *http.Client
}

// NewAnthropicProvider creates a new Anthropic provider
func NewAnthropicProvider(apiKey, model string) *AnthropicProvider {
	if model == "" {
		model = "claude-3-5-sonnet-20241022" // Default to Claude 3.5 Sonnet
	}
	return &AnthropicProvider{
		apiKey: apiKey,
		model:  model,
		client: &http.Client{},
	}
}

// Name returns the provider name
func (p *AnthropicProvider) Name() string {
	return fmt.Sprintf("Anthropic (%s)", p.model)
}

// Anthropic API structures
type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicRequest struct {
	Model     string             `json:"model"`
	Messages  []anthropicMessage `json:"messages"`
	MaxTokens int                `json:"max_tokens"`
	Stream    bool               `json:"stream"`
}

type anthropicContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type anthropicDelta struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type anthropicStreamEvent struct {
	Type         string                 `json:"type"` // "message_start", "content_block_start", "content_block_delta", "content_block_stop", "message_delta", "message_stop"
	Message      *anthropicMessageData  `json:"message,omitempty"`
	ContentBlock *anthropicContentBlock `json:"content_block,omitempty"`
	Delta        *anthropicDelta        `json:"delta,omitempty"`
}

type anthropicMessageData struct {
	ID      string                  `json:"id"`
	Type    string                  `json:"type"`
	Role    string                  `json:"role"`
	Content []anthropicContentBlock `json:"content"`
}

type anthropicResponse struct {
	ID      string                  `json:"id"`
	Type    string                  `json:"type"`
	Role    string                  `json:"role"`
	Content []anthropicContentBlock `json:"content"`
}

// CategorizeAlert asks Claude to categorize the alert
func (p *AnthropicProvider) CategorizeAlert(alert types.Alert) (string, error) {
	alertName := alert.Labels["alertname"]
	severity := alert.Labels["severity"]
	summary := alert.Annotations["summary"]
	description := alert.Annotations["description"]

	prompt := fmt.Sprintf(`K8s SRE: Categorize this alert. Pick the MOST SPECIFIC match.

Alert: %s
Severity: %s
Summary: %s
Description: %s

Categories (priority order):
1. hpa - HPA/autoscaling issues (keywords: autoscal, hpa, scale up, scale down, horizontal, replica target, autoscaling)
2. pod-crash - Pods crashing (keywords: crashloop, backoff, exit code, terminated, failed)
3. pod-restart - Excessive restarts (keywords: restart count, restarting frequently)
4. memory - Memory issues (keywords: oom, memory limit, memory pressure)
5. cpu - CPU issues (keywords: cpu throttl, high cpu, cpu limit)
6. disk - Storage issues (keywords: disk full, pvc, volume, filesystem)
7. network - Network issues (keywords: connection timeout, dns, unreachable)
8. service - Service/LB issues (keywords: endpoint, load balanc, ingress)
9. node - Node issues (keywords: node not ready, taint, cordon, node pressure)
10. deployment - Deployment issues (keywords: rollout, deployment unavailable)
11. unknown - None match

Rules:
- If "autoscal" or "hpa" appears anywhere → hpa
- If "crashloop" or "backoff" → pod-crash
- If just "restart" with high count → pod-restart
- Pick the FIRST matching category

Response: ONE word only

Category:`, alertName, severity, summary, description)

	reqBody := anthropicRequest{
		Model: p.model,
		Messages: []anthropicMessage{
			{Role: "user", Content: prompt},
		},
		MaxTokens: 100,
		Stream:    false,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "unknown", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(jsonData))
	if err != nil {
		return "unknown", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := p.client.Do(req)
	if err != nil {
		return "unknown", fmt.Errorf("failed to call Anthropic API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "unknown", fmt.Errorf("Anthropic API returned status %d: %s", resp.StatusCode, string(body))
	}

	var anthropicResp anthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&anthropicResp); err != nil {
		return "unknown", fmt.Errorf("failed to decode Anthropic response: %w", err)
	}

	if len(anthropicResp.Content) == 0 {
		return "unknown", fmt.Errorf("Anthropic returned no content")
	}

	// Clean up the response
	category := strings.TrimSpace(anthropicResp.Content[0].Text)
	category = strings.ToLower(category)

	// Extract first line if multi-line response
	if idx := strings.Index(category, "\n"); idx != -1 {
		category = category[:idx]
		category = strings.TrimSpace(category)
	}

	// Remove anything after colon
	if idx := strings.Index(category, ":"); idx != -1 {
		category = category[:idx]
		category = strings.TrimSpace(category)
	}

	// Validate category
	validCategories := map[string]bool{
		"pod-crash": true, "pod-restart": true, "memory": true, "cpu": true,
		"disk": true, "network": true, "service": true, "hpa": true,
		"node": true, "deployment": true, "unknown": true,
	}

	if !validCategories[category] {
		log.Printf("Anthropic returned invalid category '%s', using 'unknown'", anthropicResp.Content[0].Text)
		return "unknown", nil
	}

	log.Printf("Successfully categorized as: %s", category)
	return category, nil
}

// AnalyzeDebugInfoStream performs streaming analysis
func (p *AnthropicProvider) AnalyzeDebugInfoStream(debugInfo string, pastFeedback []types.Feedback, updateFn func(chunk string)) error {
	prompt := BuildAnalysisPrompt(debugInfo, pastFeedback)

	reqBody := anthropicRequest{
		Model: p.model,
		Messages: []anthropicMessage{
			{Role: "user", Content: prompt},
		},
		MaxTokens: 4096,
		Stream:    true,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to call Anthropic API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Anthropic API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Read streaming response (SSE format)
	buf := make([]byte, 4096)
	for {
		n, err := resp.Body.Read(buf)
		if err != nil && err != io.EOF {
			return fmt.Errorf("failed to read stream: %w", err)
		}
		if n == 0 {
			break
		}

		// Parse SSE events
		data := string(buf[:n])
		for _, line := range strings.Split(data, "\n") {
			line = strings.TrimSpace(line)
			if line == "" || !strings.HasPrefix(line, "data: ") {
				continue
			}

			jsonStr := strings.TrimPrefix(line, "data: ")
			var event anthropicStreamEvent
			if err := json.Unmarshal([]byte(jsonStr), &event); err != nil {
				continue // Skip malformed events
			}

			// Handle content_block_delta events (contain actual text)
			if event.Type == "content_block_delta" && event.Delta != nil && event.Delta.Text != "" {
				updateFn(event.Delta.Text)
			}

			// Handle message_stop (end of stream)
			if event.Type == "message_stop" {
				return nil
			}
		}
	}

	return nil
}

// AnalyzeDebugInfo performs non-streaming analysis
func (p *AnthropicProvider) AnalyzeDebugInfo(debugInfo string, pastFeedback []types.Feedback) (string, error) {
	var fullResponse strings.Builder

	err := p.AnalyzeDebugInfoStream(debugInfo, pastFeedback, func(chunk string) {
		fullResponse.WriteString(chunk)
	})

	if err != nil {
		return "", err
	}

	return fullResponse.String(), nil
}
