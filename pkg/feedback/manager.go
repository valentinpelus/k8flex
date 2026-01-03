package feedback

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/valentinpelus/k8flex/pkg/types"
)

// Manager handles feedback storage and retrieval
type Manager struct {
	filePath string
	store    *types.FeedbackStore
	mu       sync.RWMutex
}

// NewManager creates a new feedback manager
func NewManager(filePath string) *Manager {
	m := &Manager{
		filePath: filePath,
		store:    &types.FeedbackStore{Feedbacks: []types.Feedback{}},
	}

	// Load existing feedback
	if err := m.load(); err != nil {
		log.Printf("No existing feedback file, starting fresh: %v", err)
	}

	return m
}

// RecordFeedback stores human feedback about an analysis
func (m *Manager) RecordFeedback(feedback types.Feedback) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.store.Feedbacks = append(m.store.Feedbacks, feedback)

	log.Printf("Recorded %s feedback for alert '%s' (category: %s)",
		feedbackEmoji(feedback.IsCorrect), feedback.AlertName, feedback.Category)

	return m.save()
}

// GetRelevantFeedback retrieves past feedback for similar alerts
func (m *Manager) GetRelevantFeedback(category string, alertName string, limit int) []types.Feedback {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var relevant []types.Feedback

	// First, try to find feedback for the same alert name
	for i := len(m.store.Feedbacks) - 1; i >= 0 && len(relevant) < limit; i-- {
		fb := m.store.Feedbacks[i]
		if fb.AlertName == alertName {
			relevant = append(relevant, fb)
		}
	}

	// If not enough, add feedback for the same category
	if len(relevant) < limit {
		for i := len(m.store.Feedbacks) - 1; i >= 0 && len(relevant) < limit; i-- {
			fb := m.store.Feedbacks[i]
			if fb.Category == category && fb.AlertName != alertName {
				// Check if not already included
				found := false
				for _, r := range relevant {
					if r.Timestamp == fb.Timestamp {
						found = true
						break
					}
				}
				if !found {
					relevant = append(relevant, fb)
				}
			}
		}
	}

	return relevant
}

// GetStats returns feedback statistics
func (m *Manager) GetStats() (total, correct, incorrect int) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	total = len(m.store.Feedbacks)
	for _, fb := range m.store.Feedbacks {
		if fb.IsCorrect {
			correct++
		} else {
			incorrect++
		}
	}

	return total, correct, incorrect
}

// load reads feedback from disk
func (m *Manager) load() error {
	data, err := os.ReadFile(m.filePath)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, m.store)
}

// save writes feedback to disk
func (m *Manager) save() error {
	data, err := json.MarshalIndent(m.store, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal feedback: %w", err)
	}

	return os.WriteFile(m.filePath, data, 0644)
}

func feedbackEmoji(isCorrect bool) string {
	if isCorrect {
		return "✅"
	}
	return "❌"
}
