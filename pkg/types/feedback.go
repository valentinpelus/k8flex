package types

import "time"

// Feedback represents human feedback on an analysis
type Feedback struct {
	Timestamp   time.Time         `json:"timestamp"`
	AlertName   string            `json:"alert_name"`
	Category    string            `json:"category"`
	Namespace   string            `json:"namespace"`
	Summary     string            `json:"summary"`
	Analysis    string            `json:"analysis"`
	IsCorrect   bool              `json:"is_correct"`   // true for ✅, false for ❌
	SlackThread string            `json:"slack_thread"` // For reference
	Labels      map[string]string `json:"labels"`       // Alert labels for context
}

// FeedbackStore holds all collected feedback
type FeedbackStore struct {
	Feedbacks []Feedback `json:"feedbacks"`
}
