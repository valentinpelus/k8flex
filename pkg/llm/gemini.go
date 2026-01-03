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

// GeminiProvider implements the Provider interface for Google's Gemini models
type GeminiProvider struct {
	apiKey string
	model  string
	client *http.Client
}

// NewGeminiProvider creates a new Gemini provider
func NewGeminiProvider(apiKey, model string) *GeminiProvider {
	if model == "" {
		model = "gemini-1.5-pro" // Default to Gemini 1.5 Pro
	}
	return &GeminiProvider{
		apiKey: apiKey,
		model:  model,
		client: &http.Client{},
	}
}

// Name returns the provider name
func (p *GeminiProvider) Name() string {
	return fmt.Sprintf("Google Gemini (%s)", p.model)
}

// Gemini API structures
type geminiContent struct {
	Parts []geminiPart `json:"parts"`
	Role  string       `json:"role,omitempty"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiRequest struct {
	Contents         []geminiContent         `json:"contents"`
	GenerationConfig *geminiGenerationConfig `json:"generationConfig,omitempty"`
}

type geminiGenerationConfig struct {
	Temperature     float64 `json:"temperature,omitempty"`
	MaxOutputTokens int     `json:"maxOutputTokens,omitempty"`
}

type geminiCandidate struct {
	Content geminiContent `json:"content"`
}

type geminiResponse struct {
	Candidates []geminiCandidate `json:"candidates"`
}

// CategorizeAlert asks Gemini to categorize the alert
func (p *GeminiProvider) CategorizeAlert(alert types.Alert) (string, error) {
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

	reqBody := geminiRequest{
		Contents: []geminiContent{
			{
				Parts: []geminiPart{{Text: prompt}},
			},
		},
		GenerationConfig: &geminiGenerationConfig{
			MaxOutputTokens: 100,
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "unknown", fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", p.model, p.apiKey)
	resp, err := p.client.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "unknown", fmt.Errorf("failed to call Gemini API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "unknown", fmt.Errorf("Gemini API returned status %d: %s", resp.StatusCode, string(body))
	}

	var geminiResp geminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		return "unknown", fmt.Errorf("failed to decode Gemini response: %w", err)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return "unknown", fmt.Errorf("Gemini returned no content")
	}

	// Clean up the response
	category := strings.TrimSpace(geminiResp.Candidates[0].Content.Parts[0].Text)
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
		log.Printf("Gemini returned invalid category '%s', using 'unknown'", geminiResp.Candidates[0].Content.Parts[0].Text)
		return "unknown", nil
	}

	log.Printf("Successfully categorized as: %s", category)
	return category, nil
}

// AnalyzeDebugInfoStream performs streaming analysis
func (p *GeminiProvider) AnalyzeDebugInfoStream(debugInfo string, pastFeedback []types.Feedback, updateFn func(chunk string)) error {
	prompt := BuildAnalysisPrompt(debugInfo, pastFeedback)

	reqBody := geminiRequest{
		Contents: []geminiContent{
			{
				Parts: []geminiPart{{Text: prompt}},
			},
		},
		GenerationConfig: &geminiGenerationConfig{
			MaxOutputTokens: 4096,
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// Use streamGenerateContent endpoint for streaming
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:streamGenerateContent?key=%s&alt=sse", p.model, p.apiKey)
	resp, err := p.client.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to call Gemini API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Gemini API returned status %d: %s", resp.StatusCode, string(body))
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
			var streamResp geminiResponse
			if err := json.Unmarshal([]byte(jsonStr), &streamResp); err != nil {
				continue // Skip malformed chunks
			}

			if len(streamResp.Candidates) > 0 {
				content := streamResp.Candidates[0].Content
				if len(content.Parts) > 0 && content.Parts[0].Text != "" {
					updateFn(content.Parts[0].Text)
				}
			}
		}
	}

	return nil
}

// AnalyzeDebugInfo performs non-streaming analysis
func (p *GeminiProvider) AnalyzeDebugInfo(debugInfo string, pastFeedback []types.Feedback) (string, error) {
	var fullResponse strings.Builder

	err := p.AnalyzeDebugInfoStream(debugInfo, pastFeedback, func(chunk string) {
		fullResponse.WriteString(chunk)
	})

	if err != nil {
		return "", err
	}

	return fullResponse.String(), nil
}
