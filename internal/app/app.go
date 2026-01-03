package app

import (
	"log"

	"github.com/valentinpelus/k8flex/internal/config"
	"github.com/valentinpelus/k8flex/internal/debugger"
	"github.com/valentinpelus/k8flex/internal/processor"
	"github.com/valentinpelus/k8flex/pkg/feedback"
	"github.com/valentinpelus/k8flex/pkg/knowledge"
	"github.com/valentinpelus/k8flex/pkg/kubernetes"
	"github.com/valentinpelus/k8flex/pkg/llm"
	"github.com/valentinpelus/k8flex/pkg/slack"
)

// App holds all application dependencies
type App struct {
	Config          *config.Config
	K8sClient       *kubernetes.Client
	LLMProvider     llm.Provider
	SlackClient     *slack.Client
	FeedbackManager *feedback.Manager
	AlertProcessor  *processor.AlertProcessor
}

// New initializes a new application with all dependencies
func New() (*App, error) {
	// Load configuration
	cfg := config.LoadConfig()

	// Initialize Kubernetes client
	clientset, err := kubernetes.GetClientset()
	if err != nil {
		return nil, err
	}
	k8sClient := kubernetes.NewClient(clientset)

	// Initialize LLM provider based on configuration
	var llmProvider llm.Provider
	llmConfig := llm.Config{
		Provider:        cfg.LLMProvider,
		OllamaURL:       cfg.OllamaURL,
		OllamaModel:     cfg.OllamaModel,
		OpenAIAPIKey:    cfg.OpenAIAPIKey,
		OpenAIModel:     cfg.OpenAIModel,
		AnthropicAPIKey: cfg.AnthropicAPIKey,
		AnthropicModel:  cfg.AnthropicModel,
		GeminiAPIKey:    cfg.GeminiAPIKey,
		GeminiModel:     cfg.GeminiModel,
		BedrockRegion:   cfg.BedrockRegion,
		BedrockModel:    cfg.BedrockModel,
	}

	// For Ollama, use the new OllamaProvider from llm package
	if llmConfig.Provider == "ollama" || llmConfig.Provider == "" {
		llmProvider = llm.NewOllamaProvider(cfg.OllamaURL, cfg.OllamaModel)
	} else {
		factory := llm.NewFactory(llmConfig)
		llmProvider, err = factory.CreateProvider()
		if err != nil {
			return nil, err
		}
	}

	log.Printf("Using LLM provider: %s", llmProvider.Name())

	// Initialize Slack client
	slackClient := slack.NewClient(cfg.SlackWebhookURL, cfg.SlackBotToken, cfg.SlackChannelID)
	if cfg.SlackWorkspaceID != "" {
		slackClient.SetWorkspaceID(cfg.SlackWorkspaceID)
		log.Printf("Slack workspace ID configured: %s", cfg.SlackWorkspaceID)
	}

	// Validate Slack bot scopes if bot token is configured
	if cfg.SlackBotToken != "" && cfg.SlackChannelID != "" {
		if err := slackClient.ValidateScopes(); err != nil {
			log.Printf("WARNING: Slack bot scope validation failed: %v", err)
			log.Printf("Feedback detection requires 'reactions:read' scope. Add it at https://api.slack.com/apps")
		} else {
			log.Printf("Slack bot scopes validated successfully")
		}
	}

	// Initialize feedback manager
	feedbackManager := feedback.NewManager("/data/feedback.json")

	// Initialize knowledge base (if enabled)
	var knowledgeBase *knowledge.KnowledgeBase
	if cfg.KnowledgeBaseEnabled {
		if cfg.KnowledgeBaseDatabaseURL == "" {
			log.Printf("WARNING: Knowledge base enabled but KB_DATABASE_URL not configured")
		} else {
			// Determine API key for embeddings
			embeddingAPIKey := cfg.KnowledgeBaseAPIKey
			if embeddingAPIKey == "" {
				// Fallback to LLM provider API key if not separately configured
				switch cfg.KnowledgeBaseEmbedding {
				case "openai":
					embeddingAPIKey = cfg.OpenAIAPIKey
				case "gemini":
					embeddingAPIKey = cfg.GeminiAPIKey
				}
			}

			kbConfig := &knowledge.KnowledgeBaseConfig{
				DatabaseURL:         cfg.KnowledgeBaseDatabaseURL,
				EmbeddingProvider:   cfg.KnowledgeBaseEmbedding,
				EmbeddingAPIKey:     embeddingAPIKey,
				EmbeddingModel:      cfg.KnowledgeBaseModel,
				SimilarityThreshold: float32(cfg.KnowledgeBaseSimilarity),
				MaxSimilarCases:     cfg.KnowledgeBaseMaxResults,
			}

			var err error
			knowledgeBase, err = knowledge.NewKnowledgeBase(kbConfig)
			if err != nil {
				log.Printf("WARNING: Failed to initialize knowledge base: %v", err)
				log.Printf("Continuing without knowledge base support")
			} else {
				log.Printf("✅ Knowledge base enabled: %s embeddings, similarity threshold: %.2f",
					cfg.KnowledgeBaseEmbedding, cfg.KnowledgeBaseSimilarity)
			}
		}
	}

	// Initialize debugger
	dbg := debugger.New(k8sClient)

	// Initialize alert processor
	alertProcessor := processor.NewAlertProcessor(dbg, llmProvider, slackClient, feedbackManager, knowledgeBase)

	// Log feedback stats
	total, correct, incorrect := feedbackManager.GetStats()
	if total > 0 {
		log.Printf("Loaded %d feedback entries (%d correct, %d incorrect)", total, correct, incorrect)
	}

	return &App{
		Config:          cfg,
		K8sClient:       k8sClient,
		LLMProvider:     llmProvider,
		SlackClient:     slackClient,
		FeedbackManager: feedbackManager,
		AlertProcessor:  alertProcessor,
	}, nil
}

// LogStartupInfo logs application startup information
func (a *App) LogStartupInfo() {
	log.Printf("Starting K8flex AI Debug Agent on port %s", a.Config.Port)
	log.Printf("LLM Provider: %s", a.LLMProvider.Name())

	if a.Config.WebhookAuthToken != "" {
		log.Printf("Webhook authentication: enabled (Bearer token required)")
	} else {
		log.Printf("Webhook authentication: disabled (WARNING: anyone can send alerts)")
	}

	if a.SlackClient.HasBotToken() {
		log.Printf("Slack notifications: enabled (Bot token with threading support)")
	} else if a.SlackClient.IsConfigured() {
		log.Printf("Slack notifications: enabled (Webhook - no threading)")
	} else {
		log.Printf("Slack notifications: disabled")
	}
}
