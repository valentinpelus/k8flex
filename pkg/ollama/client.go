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

	prompt := fmt.Sprintf(`You are a Kubernetes SRE expert. Categorize this alert to determine what debug information is needed.

Alert Information:
- Name: %s
- Severity: %s
- Summary: %s
- Description: %s

Respond with ONLY ONE of these categories (exactly as written):
- pod-crash: Pod is crashing or in CrashLoopBackOff
- pod-restart: Pod is restarting frequently
- memory: Memory/OOM related issues
- cpu: CPU usage or throttling issues
- disk: Disk/storage related issues
- network: Network connectivity or DNS issues
- service: Service endpoint or load balancing issues
- hpa: HorizontalPodAutoscaler scaling issues
- node: Node-level issues (pressure, taints, etc)
- deployment: Deployment rollout or replica issues
- unknown: Cannot determine from the information

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
	}

	// Validate it's one of the expected categories
	validCategories := map[string]bool{
		"pod-crash": true, "pod-restart": true, "memory": true, "cpu": true,
		"disk": true, "network": true, "service": true, "hpa": true,
		"node": true, "deployment": true, "unknown": true,
	}

	if !validCategories[category] {
		log.Printf("Ollama returned invalid category '%s', using 'unknown'", category)
		return "unknown", nil
	}

	return category, nil
}

// AnalyzeDebugInfo sends debug information to Ollama for analysis
// Reference: https://github.com/ollama/ollama/blob/main/docs/api.md
func (c *Client) AnalyzeDebugInfo(debugInfo string) (string, error) {
	prompt := fmt.Sprintf(`You are a Kubernetes SRE expert analyzing an incident. The debug information provided has been pre-filtered to include ONLY the data relevant to this specific alert.

Analyze the information and provide a FOCUSED response with:

1. Root Cause: Identify the most likely root cause based on the provided evidence
2. Key Evidence: Cite specific log lines, error messages, or metrics that support your conclusion
3. Impact: Describe the actual impact of this issue
4. Immediate Actions: List 2-3 concrete steps to resolve the issue now
5. Prevention: Suggest 1-2 preventive measures

Keep your response concise and actionable. Focus ONLY on insights directly related to this alert. Do not speculate about information not provided.

Debug Information:
%s

Analysis:`, debugInfo)

	reqBody := types.OllamaRequest{
		Model:  c.model,
		Prompt: prompt,
		Stream: false,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.client.Post(c.baseURL+"/api/generate", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to call Ollama API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Ollama API returned status %d: %s", resp.StatusCode, string(body))
	}

	var ollamaResp types.OllamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return "", fmt.Errorf("failed to decode Ollama response: %w", err)
	}

	return ollamaResp.Response, nil
}
