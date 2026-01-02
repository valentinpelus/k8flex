package ollama

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

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

// AnalyzeDebugInfo sends debug information to Ollama for analysis
// Reference: https://github.com/ollama/ollama/blob/main/docs/api.md
func (c *Client) AnalyzeDebugInfo(debugInfo string) (string, error) {
	prompt := fmt.Sprintf(`You are a Kubernetes SRE expert analyzing an incident. Based on the following debug information, provide:

1. Root Cause Analysis: What is the most likely root cause of this issue?
2. Evidence: What specific evidence supports this conclusion?
3. Impact Assessment: What is the impact of this issue?
4. Recommended Actions: What immediate actions should the team take?
5. Prevention: What can be done to prevent this in the future?

Debug Information:
%s

Provide a clear, structured analysis:`, debugInfo)

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
