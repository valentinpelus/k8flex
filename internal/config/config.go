package config

import (
	"os"
	"strconv"
)

// Config holds all application configuration
type Config struct {
	Port             string
	LLMProvider      string // "ollama", "openai", "anthropic", "gemini", "bedrock"
	OllamaURL        string
	OllamaModel      string
	OpenAIAPIKey     string
	OpenAIModel      string
	AnthropicAPIKey  string
	AnthropicModel   string
	GeminiAPIKey     string
	GeminiModel      string
	BedrockRegion    string
	BedrockModel     string
	SlackWebhookURL  string
	SlackBotToken    string
	SlackChannelID   string
	SlackWorkspaceID string
	WebhookAuthToken string
	// Knowledge Base Configuration
	KnowledgeBaseEnabled     bool
	KnowledgeBaseDatabaseURL string
	KnowledgeBaseEmbedding   string  // "openai" or "gemini"
	KnowledgeBaseAPIKey      string  // API key for embedding provider
	KnowledgeBaseModel       string  // Embedding model name
	KnowledgeBaseSimilarity  float64 // Similarity threshold (0-1)
	KnowledgeBaseMaxResults  int     // Max similar cases to retrieve
}

// LoadConfig loads configuration from environment variables
func LoadConfig() *Config {
	return &Config{
		Port:             getEnv("PORT", "8080"),
		LLMProvider:      getEnv("LLM_PROVIDER", "ollama"),
		OllamaURL:        getEnv("OLLAMA_URL", "http://ollama.ollama.svc.cluster.local:11434"),
		OllamaModel:      getEnv("OLLAMA_MODEL", "llama3"),
		OpenAIAPIKey:     getEnv("OPENAI_API_KEY", ""),
		OpenAIModel:      getEnv("OPENAI_MODEL", "gpt-4-turbo-preview"),
		AnthropicAPIKey:  getEnv("ANTHROPIC_API_KEY", ""),
		AnthropicModel:   getEnv("ANTHROPIC_MODEL", "claude-3-5-sonnet-20241022"),
		GeminiAPIKey:     getEnv("GEMINI_API_KEY", ""),
		GeminiModel:      getEnv("GEMINI_MODEL", "gemini-1.5-pro"),
		BedrockRegion:    getEnv("BEDROCK_REGION", "us-east-1"),
		BedrockModel:     getEnv("BEDROCK_MODEL", "anthropic.claude-3-5-sonnet-20241022-v2:0"),
		SlackWebhookURL:  getEnv("SLACK_WEBHOOK_URL", ""),
		SlackBotToken:    getEnv("SLACK_BOT_TOKEN", ""),
		SlackChannelID:   getEnv("SLACK_CHANNEL_ID", ""),
		SlackWorkspaceID: getEnv("SLACK_WORKSPACE_ID", ""),
		WebhookAuthToken: getEnv("WEBHOOK_AUTH_TOKEN", ""),
		// Knowledge Base
		KnowledgeBaseEnabled:     getEnv("KB_ENABLED", "false") == "true",
		KnowledgeBaseDatabaseURL: getEnv("KB_DATABASE_URL", ""),
		KnowledgeBaseEmbedding:   getEnv("KB_EMBEDDING_PROVIDER", "openai"),
		KnowledgeBaseAPIKey:      getEnv("KB_EMBEDDING_API_KEY", ""),
		KnowledgeBaseModel:       getEnv("KB_EMBEDDING_MODEL", "text-embedding-3-small"),
		KnowledgeBaseSimilarity:  getEnvFloat("KB_SIMILARITY_THRESHOLD", 0.75),
		KnowledgeBaseMaxResults:  getEnvInt("KB_MAX_RESULTS", 5),
	}
}

// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvFloat gets a float environment variable with a default value
func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if f, err := strconv.ParseFloat(value, 64); err == nil {
			return f
		}
	}
	return defaultValue
}

// getEnvInt gets an int environment variable with a default value
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultValue
}
