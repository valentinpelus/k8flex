package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	brtypes "github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
	"github.com/valentinpelus/k8flex/pkg/types"
)

// BedrockProvider implements the Provider interface for AWS Bedrock
type BedrockProvider struct {
	client *bedrockruntime.Client
	model  string
	region string
}

// NewBedrockProvider creates a new AWS Bedrock provider
func NewBedrockProvider(region, model string) (*BedrockProvider, error) {
	if region == "" {
		region = "us-east-1" // Default region
	}
	if model == "" {
		model = "anthropic.claude-3-5-sonnet-20241022-v2:0" // Default model
	}

	// Load AWS credentials from environment/IAM role
	cfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := bedrockruntime.NewFromConfig(cfg)

	return &BedrockProvider{
		client: client,
		model:  model,
		region: region,
	}, nil
}

// Name returns the provider name
func (p *BedrockProvider) Name() string {
	return fmt.Sprintf("AWS Bedrock (%s)", p.model)
}

// Bedrock request/response structures (using Claude's format on Bedrock)
type bedrockClaudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type bedrockClaudeRequest struct {
	Messages         []bedrockClaudeMessage `json:"messages"`
	MaxTokens        int                    `json:"max_tokens"`
	Temperature      float64                `json:"temperature,omitempty"`
	AnthropicVersion string                 `json:"anthropic_version"`
}

type bedrockClaudeContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type bedrockClaudeResponse struct {
	ID      string                      `json:"id"`
	Type    string                      `json:"type"`
	Role    string                      `json:"role"`
	Content []bedrockClaudeContentBlock `json:"content"`
}

// CategorizeAlert asks Bedrock to categorize the alert
func (p *BedrockProvider) CategorizeAlert(alert types.Alert) (string, error) {
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

	reqBody := bedrockClaudeRequest{
		Messages: []bedrockClaudeMessage{
			{Role: "user", Content: prompt},
		},
		MaxTokens:        100,
		Temperature:      0.0,
		AnthropicVersion: "bedrock-2023-05-31",
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "unknown", fmt.Errorf("failed to marshal request: %w", err)
	}

	ctx := context.Background()
	resp, err := p.client.InvokeModel(ctx, &bedrockruntime.InvokeModelInput{
		ModelId:     aws.String(p.model),
		ContentType: aws.String("application/json"),
		Accept:      aws.String("application/json"),
		Body:        jsonData,
	})

	if err != nil {
		return "unknown", fmt.Errorf("failed to call Bedrock API: %w", err)
	}

	var bedrockResp bedrockClaudeResponse
	if err := json.Unmarshal(resp.Body, &bedrockResp); err != nil {
		return "unknown", fmt.Errorf("failed to decode Bedrock response: %w", err)
	}

	if len(bedrockResp.Content) == 0 {
		return "unknown", fmt.Errorf("Bedrock returned no content")
	}

	// Clean up the response
	category := strings.TrimSpace(bedrockResp.Content[0].Text)
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
		log.Printf("Bedrock returned invalid category '%s', using 'unknown'", bedrockResp.Content[0].Text)
		return "unknown", nil
	}

	log.Printf("Successfully categorized as: %s", category)
	return category, nil
}

// AnalyzeDebugInfoStream performs streaming analysis
func (p *BedrockProvider) AnalyzeDebugInfoStream(debugInfo string, pastFeedback []types.Feedback, updateFn func(chunk string)) error {
	prompt := BuildAnalysisPrompt(debugInfo, pastFeedback)

	reqBody := bedrockClaudeRequest{
		Messages: []bedrockClaudeMessage{
			{Role: "user", Content: prompt},
		},
		MaxTokens:        4096,
		Temperature:      0.0,
		AnthropicVersion: "bedrock-2023-05-31",
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	ctx := context.Background()

	// Use InvokeModelWithResponseStream for streaming
	resp, err := p.client.InvokeModelWithResponseStream(ctx, &bedrockruntime.InvokeModelWithResponseStreamInput{
		ModelId:     aws.String(p.model),
		ContentType: aws.String("application/json"),
		Accept:      aws.String("application/json"),
		Body:        jsonData,
	})

	if err != nil {
		return fmt.Errorf("failed to call Bedrock streaming API: %w", err)
	}

	// Process the streaming response
	stream := resp.GetStream()
	defer stream.Close()

	for event := range stream.Events() {
		switch v := event.(type) {
		case *brtypes.ResponseStreamMemberChunk:
			// Parse the chunk
			var chunkResp struct {
				Type  string `json:"type"`
				Delta struct {
					Type string `json:"type"`
					Text string `json:"text"`
				} `json:"delta"`
			}

			if err := json.Unmarshal(v.Value.Bytes, &chunkResp); err != nil {
				log.Printf("Warning: failed to parse chunk: %v", err)
				continue
			}

			// Send text chunks to the callback
			if chunkResp.Type == "content_block_delta" && chunkResp.Delta.Text != "" {
				updateFn(chunkResp.Delta.Text)
			}

		default:
			// Ignore other event types (errors will be caught by stream.Err())
		}
	}

	if err := stream.Err(); err != nil {
		return fmt.Errorf("stream error: %w", err)
	}

	return nil
}

// AnalyzeDebugInfo performs non-streaming analysis
func (p *BedrockProvider) AnalyzeDebugInfo(debugInfo string, pastFeedback []types.Feedback) (string, error) {
	var fullResponse strings.Builder

	err := p.AnalyzeDebugInfoStream(debugInfo, pastFeedback, func(chunk string) {
		fullResponse.WriteString(chunk)
	})

	if err != nil {
		return "", err
	}

	return fullResponse.String(), nil
}
