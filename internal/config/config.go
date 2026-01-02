package config

import "os"

// Config holds all application configuration
type Config struct {
	Port             string
	OllamaURL        string
	OllamaModel      string
	SlackWebhookURL  string
	SlackBotToken    string
	SlackChannelID   string
	WebhookAuthToken string
}

// LoadConfig loads configuration from environment variables
func LoadConfig() *Config {
	return &Config{
		Port:             getEnv("PORT", "8080"),
		OllamaURL:        getEnv("OLLAMA_URL", "http://ollama.ollama.svc.cluster.local:11434"),
		OllamaModel:      getEnv("OLLAMA_MODEL", "llama2"),
		SlackWebhookURL:  getEnv("SLACK_WEBHOOK_URL", ""),
		SlackBotToken:    getEnv("SLACK_BOT_TOKEN", ""),
		SlackChannelID:   getEnv("SLACK_CHANNEL_ID", ""),
		WebhookAuthToken: getEnv("WEBHOOK_AUTH_TOKEN", ""),
	}
}

// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
