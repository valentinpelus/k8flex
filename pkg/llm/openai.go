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

// OpenAIProvider implements the Provider interface for OpenAI's GPT models
type OpenAIProvider struct {
	apiKey string
	model  string
	client *http.Client
}

// NewOpenAIProvider creates a new OpenAI provider
func NewOpenAIProvider(apiKey, model string) *OpenAIProvider {
	if model == "" {
		model = "gpt-4-turbo-preview" // Default to GPT-4 Turbo
	}
	return &OpenAIProvider{
		apiKey: apiKey,
		model:  model,
		client: &http.Client{},
	}
}

// Name returns the provider name
func (p *OpenAIProvider) Name() string {
	return fmt.Sprintf("OpenAI (%s)", p.model)
}

// OpenAI API structures
type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIRequest struct {
	Model    string          `json:"model"`
	Messages []openAIMessage `json:"messages"`
	Stream   bool            `json:"stream"`
}

type openAIChoice struct {
	Delta struct {
		Content string `json:"content"`
	} `json:"delta"`
	Message struct {
		Content string `json:"content"`
	} `json:"message"`
	FinishReason string `json:"finish_reason"`
}

type openAIResponse struct {
	Choices []openAIChoice `json:"choices"`
}

// CategorizeAlert asks OpenAI to categorize the alert
func (p *OpenAIProvider) CategorizeAlert(alert types.Alert) (string, error) {
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

	reqBody := openAIRequest{
		Model: p.model,
		Messages: []openAIMessage{
			{Role: "user", Content: prompt},
		},
		Stream: false,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "unknown", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "unknown", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return "unknown", fmt.Errorf("failed to call OpenAI API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "unknown", fmt.Errorf("OpenAI API returned status %d: %s", resp.StatusCode, string(body))
	}

	var openAIResp openAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&openAIResp); err != nil {
		return "unknown", fmt.Errorf("failed to decode OpenAI response: %w", err)
	}

	if len(openAIResp.Choices) == 0 {
		return "unknown", fmt.Errorf("OpenAI returned no choices")
	}

	// Clean up the response
	category := strings.TrimSpace(openAIResp.Choices[0].Message.Content)
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
		log.Printf("OpenAI returned invalid category '%s', using 'unknown'", openAIResp.Choices[0].Message.Content)
		return "unknown", nil
	}

	log.Printf("Successfully categorized as: %s", category)
	return category, nil
}

// AnalyzeDebugInfoStream performs streaming analysis
func (p *OpenAIProvider) AnalyzeDebugInfoStream(debugInfo string, pastFeedback []types.Feedback, updateFn func(chunk string)) error {
	prompt := BuildAnalysisPrompt(debugInfo, pastFeedback)

	reqBody := openAIRequest{
		Model: p.model,
		Messages: []openAIMessage{
			{Role: "system", Content: "You are an expert Kubernetes SRE analyzing production incidents."},
			{Role: "user", Content: prompt},
		},
		Stream: true,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to call OpenAI API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("OpenAI API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Read streaming response
	for {
		var line string
		// Read line by line (SSE format: "data: {...}")
		chunk := make([]byte, 4096)
		n, err := resp.Body.Read(chunk)
		if err != nil && err != io.EOF {
			return fmt.Errorf("failed to read stream: %w", err)
		}
		if n == 0 {
			break
		}

		line = string(chunk[:n])
		// Split by newlines for SSE format
		for _, l := range strings.Split(line, "\n") {
			l = strings.TrimSpace(l)
			if l == "" || l == "data: [DONE]" {
				continue
			}
			if !strings.HasPrefix(l, "data: ") {
				continue
			}

			jsonStr := strings.TrimPrefix(l, "data: ")
			var streamResp openAIResponse
			if err := json.Unmarshal([]byte(jsonStr), &streamResp); err != nil {
				continue // Skip malformed chunks
			}

			if len(streamResp.Choices) > 0 {
				content := streamResp.Choices[0].Delta.Content
				if content != "" {
					updateFn(content)
				}

				if streamResp.Choices[0].FinishReason != "" {
					return nil
				}
			}
		}
	}

	return nil
}

// AnalyzeDebugInfo performs non-streaming analysis
func (p *OpenAIProvider) AnalyzeDebugInfo(debugInfo string, pastFeedback []types.Feedback) (string, error) {
	var fullResponse strings.Builder

	err := p.AnalyzeDebugInfoStream(debugInfo, pastFeedback, func(chunk string) {
		fullResponse.WriteString(chunk)
	})

	if err != nil {
		return "", err
	}

	return fullResponse.String(), nil
}
