package llm

import (
	"fmt"

	"github.com/valentinpelus/k8flex/pkg/types"
)

// BuildAnalysisPrompt creates the analysis prompt shared across all providers
func BuildAnalysisPrompt(debugInfo string, pastFeedback []types.Feedback) string {
	// Build compact feedback context if available
	var feedbackContext string
	if len(pastFeedback) > 0 {
		feedbackContext = "\n=== PAST FEEDBACK ===\n"
		for i, fb := range pastFeedback {
			status := "✅ CORRECT"
			if !fb.IsCorrect {
				status = "❌ WRONG"
			}
			// Truncate analysis to first 200 chars
			analysis := fb.Analysis
			if len(analysis) > 200 {
				analysis = analysis[:200] + "..."
			}
			feedbackContext += fmt.Sprintf("%d. %s (%s): %s - %s\n", i+1, fb.AlertName, fb.Category, status, analysis)
		}
		feedbackContext += "\n"
	}

	prompt := fmt.Sprintf(`K8s SRE expert: Analyze this incident. Debug info is pre-filtered for this alert only.
%s
ANALYSIS RULES:
1. Base ALL conclusions on the Debug Info below - cite specific evidence
2. You MAY make logical inferences from the provided metrics and logs
3. Cross-reference patterns with past incidents (see feedback above) if similar
4. Quote actual log lines, errors, or metric values when citing evidence
5. If data is incomplete, state what's missing instead of inventing details
6. Use your K8s expertise to interpret the data, but DO NOT fabricate scenarios
7. Also consider severity and alert status in your impact assessment
8. Provide clear, actionable remediation steps based on evidence
9. Suggest prevention measures to avoid recurrence
10. Be concise and structured in your response
11. Keep in mind that the alert name and description may not cover all aspects of the issue
12. If the alert name is misleading, rely on the debug info for accurate analysis but in the same moment try to align with the alert context
13. Distinguish between:
   - OBSERVED (what debug data shows NOW): "Pod status is Running" ✓
   - PAST EVENTS (from logs/events): "Pod was OOMKilled 5min ago" ✓  
   - INFERENCE (logical conclusion): "This COULD indicate..." ✓
   - FABRICATION (not in data): "Pod has been terminated" ✗
14. Use conditional language for inferences: "may have", "could be", "likely", "suggests"
15. Check actual pod/node STATUS before claiming current state
16. Quote specific log lines, errors, or metrics when citing evidence

Example:
- WRONG: "The pod has been terminated" (if status shows Running)
- RIGHT: "The pod experienced an OOMKill event (see logs), but current status shows Running"

Provide analysis using this format (use *text* for bold, not **text**):

*Root Cause:* Most likely cause based on evidence (cite specific metrics/logs)
*Key Evidence:* Quote ACTUAL lines from debug info
*Impact:* What's affected (based on provided status/metrics)
*Actions:*
• Step 1 (specific to the evidence found)
• Step 2 (actionable based on data)
• Step 3 (resolves identified issue)
*Prevention:*
• Measure 1 (prevents recurrence of this root cause)
• Measure 2 (improves monitoring/detection)

Use bullet points (•). Use *bold* for headers. Ground everything in provided data.%s

Debug Info:
%s

Analysis:`, feedbackContext,
		func() string {
			if len(pastFeedback) > 0 {
				return " Apply lessons from past feedback - use similar patterns if applicable."
			}
			return ""
		}(), debugInfo)

	return prompt
}
