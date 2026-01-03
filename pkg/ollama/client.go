package ollama

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

// Client wraps the Ollama API client
type Client struct {
	baseURL string
	model   string
	client  *http.Client
}

// NewClient creates a new Ollama client
func NewClient(baseURL, model string) *Client {
	return &Client{
		baseURL: baseURL,
		model:   model,
		client:  &http.Client{},
	}
}

// CategorizeAlert asks Ollama to categorize the alert and determine what debug data is needed
func (c *Client) CategorizeAlert(alert types.Alert) (string, error) {
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

	reqBody := types.OllamaRequest{
		Model:  c.model,
		Prompt: prompt,
		Stream: false,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "unknown", fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.client.Post(c.baseURL+"/api/generate", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "unknown", fmt.Errorf("failed to call Ollama API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "unknown", fmt.Errorf("Ollama API returned status %d: %s", resp.StatusCode, string(body))
	}

	var ollamaResp types.OllamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return "unknown", fmt.Errorf("failed to decode Ollama response: %w", err)
	}

	// Clean up the response - extract just the category word
	category := strings.TrimSpace(ollamaResp.Response)
	category = strings.ToLower(category)

	// Extract first line if multi-line response
	if idx := strings.Index(category, "\n"); idx != -1 {
		category = category[:idx]
		category = strings.TrimSpace(category)
	}

	// Remove anything after colon (in case model adds explanation)
	if idx := strings.Index(category, ":"); idx != -1 {
		category = category[:idx]
		category = strings.TrimSpace(category)
	}

	// Validate it's one of the expected categories
	validCategories := map[string]bool{
		"pod-crash": true, "pod-restart": true, "memory": true, "cpu": true,
		"disk": true, "network": true, "service": true, "hpa": true,
		"node": true, "deployment": true, "unknown": true,
	}

	if !validCategories[category] {
		log.Printf("Ollama returned invalid category '%s', using 'unknown'", ollamaResp.Response)
		return "unknown", nil
	}

	log.Printf("Successfully categorized as: %s", category)
	return category, nil
}

// AnalyzeDebugInfoStream sends debug information to Ollama for streaming analysis with learning from past feedback
// Reference: https://github.com/ollama/ollama/blob/main/docs/api.md
func (c *Client) AnalyzeDebugInfoStream(debugInfo string, pastFeedback []types.Feedback, updateFn func(chunk string)) error {
	// Build compact feedback context if available
	var feedbackContext string
	if len(pastFeedback) > 0 {
		feedbackContext = "\n=== PAST FEEDBACK ===\n"
		for i, fb := range pastFeedback {
			status := "✅ CORRECT"
			if !fb.IsCorrect {
				status = "❌ WRONG"
			}
			// Truncate analysis to first 200 chars
			analysis := fb.Analysis
			if len(analysis) > 200 {
				analysis = analysis[:200] + "..."
			}
			feedbackContext += fmt.Sprintf("%d. %s (%s): %s - %s\n", i+1, fb.AlertName, fb.Category, status, analysis)
		}
		feedbackContext += "\n"
	}

	prompt := fmt.Sprintf(`K8s SRE expert: Analyze this incident. Debug info is pre-filtered for this alert only.
%s
Provide:
1. Root Cause: Most likely cause
2. Key Evidence: Specific logs/errors/metrics
3. Impact: What's affected
4. Actions: 2-3 steps to fix
5. Prevention: 1-2 measures

Be concise. Only use provided data.%s

Debug Info:
%s

Analysis:`, feedbackContext,
		func() string {
			if len(pastFeedback) > 0 {
				return " Learn from past feedback - avoid repeating ❌ mistakes."
			}
			return ""
		}(), debugInfo)

	reqBody := types.OllamaRequest{
		Model:  c.model,
		Prompt: prompt,
		Stream: true, // Enable streaming
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.client.Post(c.baseURL+"/api/generate", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to call Ollama API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Ollama API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Read streaming response line by line
	decoder := json.NewDecoder(resp.Body)
	for {
		var streamResp types.OllamaResponse
		if err := decoder.Decode(&streamResp); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to decode stream response: %w", err)
		}

		// Call update function with each chunk
		if streamResp.Response != "" {
			updateFn(streamResp.Response)
		}

		// Break if done
		if streamResp.Done {
			break
		}
	}

	return nil
}

// AnalyzeDebugInfo sends debug information to Ollama for analysis with learning from past feedback (non-streaming)
// Reference: https://github.com/ollama/ollama/blob/main/docs/api.md
func (c *Client) AnalyzeDebugInfo(debugInfo string, pastFeedback []types.Feedback) (string, error) {
	var fullResponse strings.Builder

	err := c.AnalyzeDebugInfoStream(debugInfo, pastFeedback, func(chunk string) {
		fullResponse.WriteString(chunk)
	})

	if err != nil {
		return "", err
	}

	return fullResponse.String(), nil
}
