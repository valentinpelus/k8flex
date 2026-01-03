package app

import (
	"log"

	"github.com/valentinpelus/k8flex/internal/config"
	"github.com/valentinpelus/k8flex/internal/debugger"
	"github.com/valentinpelus/k8flex/internal/processor"
	"github.com/valentinpelus/k8flex/pkg/feedback"
	"github.com/valentinpelus/k8flex/pkg/kubernetes"
	"github.com/valentinpelus/k8flex/pkg/ollama"
	"github.com/valentinpelus/k8flex/pkg/slack"
)

// App holds all application dependencies
type App struct {
	Config          *config.Config
	K8sClient       *kubernetes.Client
	OllamaClient    *ollama.Client
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

	// Initialize Ollama client
	ollamaClient := ollama.NewClient(cfg.OllamaURL, cfg.OllamaModel)

	// Initialize Slack client
	slackClient := slack.NewClient(cfg.SlackWebhookURL, cfg.SlackBotToken, cfg.SlackChannelID)

	// Initialize feedback manager
	feedbackManager := feedback.NewManager("/data/feedback.json")

	// Initialize debugger
	dbg := debugger.New(k8sClient)

	// Initialize alert processor
	alertProcessor := processor.NewAlertProcessor(dbg, ollamaClient, slackClient, feedbackManager)

	// Log feedback stats
	total, correct, incorrect := feedbackManager.GetStats()
	if total > 0 {
		log.Printf("Loaded %d feedback entries (%d correct, %d incorrect)", total, correct, incorrect)
	}

	return &App{
		Config:          cfg,
		K8sClient:       k8sClient,
		OllamaClient:    ollamaClient,
		SlackClient:     slackClient,
		FeedbackManager: feedbackManager,
		AlertProcessor:  alertProcessor,
	}, nil
}

// LogStartupInfo logs application startup information
func (a *App) LogStartupInfo() {
	log.Printf("Starting K8flex AI Debug Agent on port %s", a.Config.Port)
	log.Printf("Ollama URL: %s, Model: %s", a.Config.OllamaURL, a.Config.OllamaModel)

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
