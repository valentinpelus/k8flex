package processor

import (
	"context"
	"fmt"
	"log"

	"github.com/valentinpelus/k8flex/internal/debugger"
	"github.com/valentinpelus/k8flex/pkg/ollama"
	"github.com/valentinpelus/k8flex/pkg/slack"
	"github.com/valentinpelus/k8flex/pkg/types"
)

// AlertProcessor handles the processing of alerts
type AlertProcessor struct {
	debugger     *debugger.Debugger
	ollamaClient *ollama.Client
	slackClient  *slack.Client
}

// NewAlertProcessor creates a new alert processor
func NewAlertProcessor(dbg *debugger.Debugger, ollamaClient *ollama.Client, slackClient *slack.Client) *AlertProcessor {
	return &AlertProcessor{
		debugger:     dbg,
		ollamaClient: ollamaClient,
		slackClient:  slackClient,
	}
}

// ProcessAlert processes a single alert
func (p *AlertProcessor) ProcessAlert(alert types.Alert) {
	log.Printf("Processing alert: %s", alert.Labels["alertname"])

	// Extract parameters from alert labels
	namespace := alert.Labels["namespace"]

	if namespace == "" {
		log.Printf("Alert %s missing namespace label, skipping", alert.Labels["alertname"])
		return
	}

	// Send alert to Slack FIRST before starting debug work
	var slackThreadTS string
	if p.slackClient.IsConfigured() {
		ts, err := p.slackClient.SendAlert(alert)
		if err != nil {
			log.Printf("Failed to send alert to Slack: %v", err)
		} else {
			slackThreadTS = ts
			log.Printf("Alert sent to Slack successfully")
		}
	}

	// Phase 1: Ask Ollama to categorize the alert
	log.Printf("Asking Ollama to categorize alert: %s", alert.Labels["alertname"])
	category, err := p.ollamaClient.CategorizeAlert(alert)
	if err != nil {
		log.Printf("Error categorizing alert: %v, using 'unknown'", err)
		category = "unknown"
	}
	log.Printf("Ollama categorized alert as: %s", category)

	// Phase 2: Gather only relevant debug information based on category
	ctx := context.Background()
	debugInfo := p.debugger.GatherDebugInfo(ctx, alert, category)

	// Phase 3: Analyze with Ollama using the targeted debug info
	log.Printf("Sending debug info to Ollama for analysis")
	analysis, err := p.ollamaClient.AnalyzeDebugInfo(debugInfo)
	if err != nil {
		log.Printf("Error analyzing with Ollama: %v", err)
		analysis = fmt.Sprintf("Error: %v", err)
	}

	// Log the complete analysis
	log.Printf("\n=== COMPLETE ANALYSIS FOR %s ===\n%s\n=== AI ANALYSIS ===\n%s\n=== END ===\n",
		alert.Labels["alertname"], debugInfo, analysis)

	// Send analysis to Slack thread
	if p.slackClient.IsConfigured() && slackThreadTS != "" {
		if err := p.slackClient.SendAnalysis(alert, analysis, slackThreadTS); err != nil {
			log.Printf("Failed to send analysis to Slack thread: %v", err)
		} else {
			log.Printf("Analysis posted to Slack thread: %s", slackThreadTS)
		}
	} else if p.slackClient.IsConfigured() {
		// If no thread ID, send as separate message
		if err := p.slackClient.SendAnalysis(alert, analysis, ""); err != nil {
			log.Printf("Failed to send analysis to Slack: %v", err)
		} else {
			log.Printf("Analysis sent to Slack")
		}
	}
}
