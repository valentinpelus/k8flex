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
ANALYSIS RULES:
1. Base ALL conclusions on the Debug Info below - cite specific evidence
2. You MAY make logical inferences from the provided metrics and logs
3. Cross-reference patterns with past incidents (see feedback above) if similar
4. Quote actual log lines, errors, or metric values when citing evidence
5. If data is incomplete, state what's missing instead of inventing details
6. Use your K8s expertise to interpret the data, but DO NOT fabricate scenarios
7. Also consider severity and alert status in your impact assessment
8. Provide clear, actionable remediation steps based on evidence
9. Suggest prevention measures to avoid recurrence
10. Be concise and structured in your response
11. Keep in mind that the alert name and description may not cover all aspects of the issue
12. If the alert name is misleading, rely on the debug info for accurate analysis but in the same moment try to align with the alert context
13. Distinguish between:
   - OBSERVED (what debug data shows NOW): "Pod status is Running" ✓
   - PAST EVENTS (from logs/events): "Pod was OOMKilled 5min ago" ✓  
   - INFERENCE (logical conclusion): "This COULD indicate..." ✓
   - FABRICATION (not in data): "Pod has been terminated" ✗
14. Use conditional language for inferences: "may have", "could be", "likely", "suggests"
15. Check actual pod/node STATUS before claiming current state
16. Quote specific log lines, errors, or metrics when citing evidence

Example:
- WRONG: "The pod has been terminated" (if status shows Running)
- RIGHT: "The pod experienced an OOMKill event (see logs), but current status shows Running"

Provide analysis using this format (use *text* for bold, not **text**):

*Root Cause:* Most likely cause based on evidence (cite specific metrics/logs)
*Key Evidence:* Quote ACTUAL lines from debug info
*Impact:* What's affected (based on provided status/metrics)
*Actions:*
• Step 1 (specific to the evidence found)
• Step 2 (actionable based on data)
• Step 3 (resolves identified issue)
*Prevention:*
• Measure 1 (prevents recurrence of this root cause)
• Measure 2 (improves monitoring/detection)

Use bullet points (•). Use *bold* for headers. Ground everything in provided data.%s

Debug Info:
%s

Analysis:`, feedbackContext,
		func() string {
			if len(pastFeedback) > 0 {
				return " Apply lessons from past feedback - use similar patterns if applicable."
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
