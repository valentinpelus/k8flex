package llm

import (
	"fmt"
	"log"
)

// Factory creates LLM providers based on configuration
type Factory struct {
	config Config
}

// NewFactory creates a new provider factory
func NewFactory(config Config) *Factory {
	return &Factory{config: config}
}

// CreateProvider creates the configured LLM provider
func (f *Factory) CreateProvider() (Provider, error) {
	switch f.config.Provider {
	case "ollama":
		if f.config.OllamaURL == "" {
			return nil, fmt.Errorf("ollama URL not configured")
		}
		if f.config.OllamaModel == "" {
			f.config.OllamaModel = "llama3" // Default model
		}
		log.Printf("Using Ollama provider with model %s at %s", f.config.OllamaModel, f.config.OllamaURL)
		return NewOllamaProvider(f.config.OllamaURL, f.config.OllamaModel), nil

	case "openai":
		if f.config.OpenAIAPIKey == "" {
			return nil, fmt.Errorf("openAI API key not configured")
		}
		if f.config.OpenAIModel == "" {
			f.config.OpenAIModel = "gpt-4-turbo-preview" // Default model
		}
		log.Printf("Using OpenAI provider with model %s", f.config.OpenAIModel)
		return NewOpenAIProvider(f.config.OpenAIAPIKey, f.config.OpenAIModel), nil

	case "anthropic", "claude":
		if f.config.AnthropicAPIKey == "" {
			return nil, fmt.Errorf("anthropic API key not configured")
		}
		if f.config.AnthropicModel == "" {
			f.config.AnthropicModel = "claude-3-5-sonnet-20241022" // Default model
		}
		log.Printf("Using Anthropic provider with model %s", f.config.AnthropicModel)
		return NewAnthropicProvider(f.config.AnthropicAPIKey, f.config.AnthropicModel), nil

	case "gemini", "google":
		if f.config.GeminiAPIKey == "" {
			return nil, fmt.Errorf("gemini API key not configured")
		}
		if f.config.GeminiModel == "" {
			f.config.GeminiModel = "gemini-1.5-pro" // Default model
		}
		log.Printf("Using Google Gemini provider with model %s", f.config.GeminiModel)
		return NewGeminiProvider(f.config.GeminiAPIKey, f.config.GeminiModel), nil

	case "bedrock", "aws":
		if f.config.BedrockRegion == "" {
			f.config.BedrockRegion = "us-east-1" // Default region
		}
		if f.config.BedrockModel == "" {
			f.config.BedrockModel = "anthropic.claude-3-5-sonnet-20241022-v2:0" // Default model
		}
		log.Printf("Using AWS Bedrock provider with model %s in %s", f.config.BedrockModel, f.config.BedrockRegion)
		return NewBedrockProvider(f.config.BedrockRegion, f.config.BedrockModel)

	default:
		return nil, fmt.Errorf("unknown provider: %s (supported: ollama, openai, anthropic, gemini, bedrock)", f.config.Provider)
	}
}
