package processor

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/valentinpelus/k8flex/internal/debugger"
	"github.com/valentinpelus/k8flex/pkg/feedback"
	"github.com/valentinpelus/k8flex/pkg/ollama"
	"github.com/valentinpelus/k8flex/pkg/slack"
	"github.com/valentinpelus/k8flex/pkg/types"
)

// AlertProcessor handles the processing of alerts
type AlertProcessor struct {
	debugger        *debugger.Debugger
	ollamaClient    *ollama.Client
	slackClient     *slack.Client
	feedbackManager *feedback.Manager
}

// NewAlertProcessor creates a new alert processor
func NewAlertProcessor(dbg *debugger.Debugger, ollamaClient *ollama.Client, slackClient *slack.Client, feedbackMgr *feedback.Manager) *AlertProcessor {
	return &AlertProcessor{
		debugger:        dbg,
		ollamaClient:    ollamaClient,
		slackClient:     slackClient,
		feedbackManager: feedbackMgr,
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

	// Get past feedback for similar alerts to improve analysis (limit to 1 to reduce prompt size)
	pastFeedback := p.feedbackManager.GetRelevantFeedback(category, alert.Labels["alertname"], 1)
	if len(pastFeedback) > 0 {
		log.Printf("Including %d past feedback example for learning", len(pastFeedback))
	}

	// Phase 3: Stream analysis from Ollama with real-time Slack updates
	log.Printf("Starting streaming analysis from Ollama")

	var fullAnalysis strings.Builder
	var analysisMessageTS string // Track the THREAD message timestamp for updates (not the parent)
	updateCount := 0

	err = p.ollamaClient.AnalyzeDebugInfoStream(debugInfo, pastFeedback, func(chunk string) {
		fullAnalysis.WriteString(chunk)
		updateCount++

		// Update Slack every 10 chunks or when we have substantial content
		if p.slackClient.IsConfigured() && slackThreadTS != "" && updateCount%10 == 0 {
			currentAnalysis := fullAnalysis.String()
			if analysisMessageTS == "" {
				// First update - send initial message IN THE THREAD and capture its timestamp
				analysisMsg := "🔄 *Analysis in progress...*\n\n" + currentAnalysis
				ts, sendErr := p.slackClient.SendAnalysisInThread(alert, analysisMsg, slackThreadTS)
				if sendErr == nil {
					analysisMessageTS = ts // Save the thread message timestamp for future updates
					log.Printf("Started streaming analysis in thread message: %s", analysisMessageTS)
				}
			} else {
				// Update the THREAD message (not the parent alert message)
				analysisMsg := "🔄 *Analysis in progress...*\n\n" + currentAnalysis
				p.slackClient.UpdateMessage(analysisMessageTS, analysisMsg)
			}
		}
	})

	analysis := fullAnalysis.String()
	if err != nil {
		log.Printf("Error analyzing with Ollama: %v", err)
		analysis = fmt.Sprintf("Error: %v", err)
	}

	// Log the complete analysis
	log.Printf("\n=== COMPLETE ANALYSIS FOR %s ===\n%s\n=== AI ANALYSIS ===\n%s\n=== END ===\n",
		alert.Labels["alertname"], debugInfo, analysis)

	// Send final analysis to Slack thread
	if p.slackClient.IsConfigured() && slackThreadTS != "" {
		// Add feedback instructions
		analysisWithInstructions := "✅ *Analysis Complete*\n\n" + analysis + "\n\n_💡 Rate this analysis: React with ✅ if correct or ❌ if incorrect to help improve future debugging_"

		if analysisMessageTS != "" {
			// Update the existing streaming message with final analysis
			if err := p.slackClient.UpdateMessage(analysisMessageTS, analysisWithInstructions); err != nil {
				log.Printf("Failed to update final analysis in Slack: %v", err)
			} else {
				log.Printf("Final analysis updated in Slack thread message: %s", analysisMessageTS)
			}
		} else {
			// No streaming message exists, send as new message in thread
			if err := p.slackClient.SendAnalysis(alert, analysisWithInstructions, slackThreadTS); err != nil {
				log.Printf("Failed to send analysis to Slack thread: %v", err)
			} else {
				log.Printf("Analysis posted to Slack thread: %s", slackThreadTS)
			}
		}

		// Store pending feedback (will be updated when human reacts)
		p.storePendingFeedback(alert, category, analysis, slackThreadTS)
	} else if p.slackClient.IsConfigured() {
		// If no thread ID, send as separate message
		analysisWithInstructions := analysis + "\n\n_💡 Rate this analysis: React with ✅ if correct or ❌ if incorrect to help improve future debugging_"

		if err := p.slackClient.SendAnalysis(alert, analysisWithInstructions, ""); err != nil {
			log.Printf("Failed to send analysis to Slack: %v", err)
		} else {
			log.Printf("Analysis sent to Slack")
		}
	}
}

// storePendingFeedback stores analysis info for future feedback collection
func (p *AlertProcessor) storePendingFeedback(alert types.Alert, category, analysis, slackThread string) {
	// This creates a placeholder - in a real system, you'd implement Slack Events API
	// to listen for reactions and update this feedback
	log.Printf("Analysis ready for feedback on thread: %s", slackThread)
	log.Printf("To provide feedback manually, use the feedback API endpoint")
}

// RecordManualFeedback allows manual feedback recording (can be called from API endpoint)
func (p *AlertProcessor) RecordManualFeedback(alert types.Alert, category, analysis, slackThread string, isCorrect bool) error {
	feedback := types.Feedback{
		Timestamp:   time.Now(),
		AlertName:   alert.Labels["alertname"],
		Category:    category,
		Namespace:   alert.Labels["namespace"],
		Summary:     alert.Annotations["summary"],
		Analysis:    analysis,
		IsCorrect:   isCorrect,
		SlackThread: slackThread,
		Labels:      alert.Labels,
	}

	return p.feedbackManager.RecordFeedback(feedback)
}
