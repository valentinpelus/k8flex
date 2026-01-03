package processor

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/valentinpelus/k8flex/internal/debugger"
	"github.com/valentinpelus/k8flex/pkg/feedback"
	"github.com/valentinpelus/k8flex/pkg/knowledge"
	"github.com/valentinpelus/k8flex/pkg/llm"
	"github.com/valentinpelus/k8flex/pkg/slack"
	"github.com/valentinpelus/k8flex/pkg/types"
)

// PendingFeedback tracks analysis waiting for user reaction
type PendingFeedback struct {
	Alert      types.Alert
	Category   string
	Analysis   string
	ThreadTS   string
	AnalysisTS string // The message timestamp for the analysis
	Timestamp  time.Time
}

// AlertProcessor handles the processing of alerts
type AlertProcessor struct {
	debugger        *debugger.Debugger
	llmProvider     llm.Provider
	slackClient     *slack.Client
	feedbackManager *feedback.Manager
	knowledgeBase   *knowledge.KnowledgeBase
	pendingFeedback map[string]*PendingFeedback // Key: analysis message TS
	pendingMutex    sync.RWMutex
}

// NewAlertProcessor creates a new alert processor
func NewAlertProcessor(dbg *debugger.Debugger, llmProvider llm.Provider, slackClient *slack.Client, feedbackMgr *feedback.Manager, kb *knowledge.KnowledgeBase) *AlertProcessor {
	processor := &AlertProcessor{
		debugger:        dbg,
		llmProvider:     llmProvider,
		slackClient:     slackClient,
		feedbackManager: feedbackMgr,
		knowledgeBase:   kb,
		pendingFeedback: make(map[string]*PendingFeedback),
	}

	// Start background reaction checker if Slack is configured
	if slackClient.IsConfigured() && slackClient.HasBotToken() {
		go processor.reactionChecker()
	}

	return processor
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

	// Phase 1: Ask LLM provider to categorize the alert
	log.Printf("Asking %s to categorize alert: %s", p.llmProvider.Name(), alert.Labels["alertname"])
	category, err := p.llmProvider.CategorizeAlert(alert)
	if err != nil {
		log.Printf("Error categorizing alert: %v, using 'unknown'", err)
		category = "unknown"
	}
	log.Printf("%s categorized alert as: %s", p.llmProvider.Name(), category)

	// Phase 2: Search knowledge base for similar cases (if enabled)
	ctx := context.Background()
	var similarCases []*knowledge.SimilarCase
	if p.knowledgeBase != nil {
		searchText := fmt.Sprintf("%s %s %s",
			alert.Labels["alertname"],
			alert.Labels["severity"],
			alert.Annotations["summary"])
		var err error
		similarCases, err = p.knowledgeBase.FindSimilar(ctx, searchText)
		if err != nil {
			log.Printf("Warning: failed to search knowledge base: %v", err)
		} else if len(similarCases) > 0 {
			log.Printf("Found %d similar cases in knowledge base (top similarity: %.2f%%)",
				len(similarCases), similarCases[0].Similarity*100)
		}
	}

	// Phase 3: Gather only relevant debug information based on category
	debugInfo := p.debugger.GatherDebugInfo(ctx, alert, category)

	// Get past feedback for similar alerts to improve analysis (limit to 1 to reduce prompt size)
	pastFeedback := p.feedbackManager.GetRelevantFeedback(category, alert.Labels["alertname"], 1)
	if len(pastFeedback) > 0 {
		log.Printf("Including %d past feedback example for learning", len(pastFeedback))
		// Enhance feedback with Slack links if available
		for i := range pastFeedback {
			if pastFeedback[i].SlackThread != "" && p.slackClient.HasBotToken() {
				channelID := p.slackClient.GetChannelID()
				workspaceID := p.slackClient.GetWorkspaceID()
				if workspaceID != "" {
					slackLink := fmt.Sprintf("https://%s.slack.com/archives/%s/p%s",
						workspaceID, channelID, strings.ReplaceAll(pastFeedback[i].SlackThread, ".", ""))
					pastFeedback[i].Summary += fmt.Sprintf(" (See: %s)", slackLink)
				}
			}
		}
	}

	// Add similar cases context to prompt if available
	if len(similarCases) > 0 {
		similarCasesText := "\n\n=== SIMILAR PAST CASES (from Knowledge Base) ===\n"
		for i, sc := range similarCases {
			if i >= 3 { // Limit to top 3 to avoid prompt bloat
				break
			}
			similarCasesText += fmt.Sprintf("\n%d. [%.0f%% similar] %s - Category: %s\n",
				i+1, sc.Similarity*100, sc.Case.AlertName, sc.Case.Category)
			// Truncate analysis to first 150 chars
			analysis := sc.Case.Analysis
			if len(analysis) > 150 {
				analysis = analysis[:150] + "..."
			}
			similarCasesText += fmt.Sprintf("   Previous Analysis: %s\n", analysis)
		}
		similarCasesText += "\nUse these similar cases to inform your analysis if patterns match.\n"
		debugInfo = debugInfo + similarCasesText
	}

	// Phase 4: Stream analysis from LLM provider with real-time Slack updates
	log.Printf("Starting streaming analysis from %s", p.llmProvider.Name())

	var fullAnalysis strings.Builder
	var analysisMessageTS string // Track the THREAD message timestamp for updates (not the parent)
	updateCount := 0

	err = p.llmProvider.AnalyzeDebugInfoStream(debugInfo, pastFeedback, func(chunk string) {
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
			ts, err := p.slackClient.SendAnalysisInThread(alert, analysisWithInstructions, slackThreadTS)
			if err != nil {
				log.Printf("Failed to send analysis to Slack thread: %v", err)
			} else {
				analysisMessageTS = ts
				log.Printf("Analysis posted to Slack thread: %s", slackThreadTS)
			}
		}

		// Store pending feedback with the analysis message timestamp
		if analysisMessageTS != "" {
			p.storePendingFeedback(alert, category, analysis, slackThreadTS, analysisMessageTS)
		}
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
func (p *AlertProcessor) storePendingFeedback(alert types.Alert, category, analysis, threadTS, analysisTS string) {
	p.pendingMutex.Lock()
	defer p.pendingMutex.Unlock()

	p.pendingFeedback[analysisTS] = &PendingFeedback{
		Alert:      alert,
		Category:   category,
		Analysis:   analysis,
		ThreadTS:   threadTS,
		AnalysisTS: analysisTS,
		Timestamp:  time.Now(),
	}

	log.Printf("Stored pending feedback for message: %s (thread: %s)", analysisTS, threadTS)
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

	// Store in feedback manager
	if err := p.feedbackManager.RecordFeedback(feedback); err != nil {
		return err
	}

	// If feedback is positive and knowledge base is enabled, store the case
	if isCorrect && p.knowledgeBase != nil {
		debugInfo := "" // We don't have debug info in manual feedback, but could add it
		alertCase := knowledge.FromAlert(&alert, category, analysis, debugInfo)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := p.knowledgeBase.Store(ctx, alertCase); err != nil {
			log.Printf("Warning: failed to store case in knowledge base: %v", err)
			// Don't fail the feedback recording if knowledge base storage fails
		} else {
			log.Printf("✅ Stored validated case in knowledge base: %s (%s)", alertCase.AlertName, alertCase.Category)
		}
	}

	return nil
}

// reactionChecker periodically checks for emoji reactions on analysis messages
func (p *AlertProcessor) reactionChecker() {
	ticker := time.NewTicker(30 * time.Second) // Check every 30 seconds
	defer ticker.Stop()

	log.Printf("Started reaction checker - polling every 30 seconds")

	for range ticker.C {
		p.checkPendingReactions()
	}
}

// checkPendingReactions checks all pending feedback for reactions
func (p *AlertProcessor) checkPendingReactions() {
	p.pendingMutex.RLock()
	pendingList := make([]*PendingFeedback, 0, len(p.pendingFeedback))
	for _, pending := range p.pendingFeedback {
		pendingList = append(pendingList, pending)
	}
	p.pendingMutex.RUnlock()

	for _, pending := range pendingList {
		// Skip if too old (older than 24 hours)
		if time.Since(pending.Timestamp) > 24*time.Hour {
			p.pendingMutex.Lock()
			delete(p.pendingFeedback, pending.AnalysisTS)
			p.pendingMutex.Unlock()
			continue
		}

		// Check for reactions
		reactions, err := p.slackClient.GetMessageReactions(pending.AnalysisTS)
		if err != nil {
			log.Printf("Error checking reactions for %s: %v", pending.AnalysisTS, err)
			continue
		}

		// Check for ✅ or ❌ reactions
		var isCorrect *bool
		for _, reaction := range reactions {
			if reaction == "white_check_mark" || reaction == "coche_blanche" || reaction == "heavy_check_mark" {
				val := true
				isCorrect = &val
				break
			} else if reaction == "x" || reaction == "cross" || reaction == "negative_squared_cross_mark" {
				val := false
				isCorrect = &val
				break
			}
		}

		if isCorrect != nil {
			// Record feedback
			feedback := types.Feedback{
				Timestamp:   time.Now(),
				AlertName:   pending.Alert.Labels["alertname"],
				Category:    pending.Category,
				Namespace:   pending.Alert.Labels["namespace"],
				Summary:     pending.Alert.Annotations["summary"],
				Analysis:    pending.Analysis,
				IsCorrect:   *isCorrect,
				SlackThread: pending.ThreadTS,
				Labels:      pending.Alert.Labels,
			}

			if err := p.feedbackManager.RecordFeedback(feedback); err != nil {
				log.Printf("Error recording feedback: %v", err)
			} else {
				emoji := "✅"
				if !*isCorrect {
					emoji = "❌"
				}

				// If feedback is positive and knowledge base is enabled, store the case
				if *isCorrect && p.knowledgeBase != nil {
					debugInfo := "" // We could enhance this by storing debug info in PendingFeedback
					alertCase := knowledge.FromAlert(&pending.Alert, pending.Category, pending.Analysis, debugInfo)

					ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
					defer cancel()

					if err := p.knowledgeBase.Store(ctx, alertCase); err != nil {
						log.Printf("Warning: failed to store case in knowledge base: %v", err)
					} else {
						log.Printf("✅ Stored validated case in knowledge base: %s (%s)", alertCase.AlertName, alertCase.Category)
					}
				}

				// Notify user that feedback was recorded
				confirmMsg := fmt.Sprintf("_Thank you! Your feedback (%s) has been recorded and will help improve future analyses._", emoji)
				if err := p.slackClient.ReplyToThread(pending.ThreadTS, confirmMsg); err != nil {
					log.Printf("Error sending confirmation: %v", err)
				}
				log.Printf("Recorded %s feedback for alert '%s' via reaction", emoji, pending.Alert.Labels["alertname"])
			}

			// Remove from pending
			p.pendingMutex.Lock()
			delete(p.pendingFeedback, pending.AnalysisTS)
			p.pendingMutex.Unlock()
		}
	}
}
