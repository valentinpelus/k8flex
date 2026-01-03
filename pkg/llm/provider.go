package llm

import "github.com/valentinpelus/k8flex/pkg/types"

// Provider defines the interface for LLM providers (Ollama, OpenAI, Claude, Gemini)
type Provider interface {
	// CategorizeAlert analyzes an alert and returns its category
	CategorizeAlert(alert types.Alert) (string, error)

	// AnalyzeDebugInfoStream performs streaming analysis with real-time updates
	// updateFn is called with each chunk of the response
	AnalyzeDebugInfoStream(debugInfo string, pastFeedback []types.Feedback, updateFn func(chunk string)) error

	// AnalyzeDebugInfo performs non-streaming analysis and returns the full response
	AnalyzeDebugInfo(debugInfo string, pastFeedback []types.Feedback) (string, error)

	// Name returns the provider name (for logging)
	Name() string
}

// Config holds common configuration for LLM providers
type Config struct {
	Provider string // "ollama", "openai", "anthropic", "gemini"

	// Ollama-specific
	OllamaURL   string
	OllamaModel string

	// OpenAI-specific
	OpenAIAPIKey string
	OpenAIModel  string // e.g., "gpt-4", "gpt-4-turbo", "gpt-3.5-turbo"

	// Anthropic-specific
	AnthropicAPIKey string
	AnthropicModel  string // e.g., "claude-3-opus-20240229", "claude-3-sonnet-20240229"

	// Gemini-specific
	GeminiAPIKey string
	GeminiModel  string // e.g., "gemini-pro", "gemini-pro-vision"

	// AWS Bedrock-specific
	BedrockRegion string // e.g., "us-east-1", "us-west-2"
	BedrockModel  string // e.g., "anthropic.claude-3-5-sonnet-20241022-v2:0", "amazon.titan-text-express-v1"
}
