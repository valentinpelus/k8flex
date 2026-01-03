package knowledge

import (
	"time"

	"github.com/valentinpelus/k8flex/pkg/types"
)

// AlertCase represents a validated alert case stored in the knowledge base
type AlertCase struct {
	ID            string    `db:"id"`
	AlertName     string    `db:"alert_name"`
	Severity      string    `db:"severity"`
	Category      string    `db:"category"` // The emoji/category assigned by LLM
	Summary       string    `db:"summary"`
	Namespace     string    `db:"namespace"`
	PodName       string    `db:"pod_name"`
	ContainerName string    `db:"container_name"`
	Analysis      string    `db:"analysis"`   // Full LLM analysis
	DebugInfo     string    `db:"debug_info"` // Debug information collected
	Validated     bool      `db:"validated"`  // Whether this case was validated
	Embedding     []float32 `db:"embedding"`  // Vector embedding for similarity search
	CreatedAt     time.Time `db:"created_at"`
	UpdatedAt     time.Time `db:"updated_at"`
}

// SimilarCase represents a similar past case with similarity score
type SimilarCase struct {
	Case       *AlertCase
	Similarity float32 // Cosine similarity score (0-1)
}

// KnowledgeBaseConfig holds configuration for the knowledge base
type KnowledgeBaseConfig struct {
	DatabaseURL         string
	EmbeddingProvider   string // "openai", "gemini", "local"
	EmbeddingAPIKey     string
	EmbeddingModel      string
	SimilarityThreshold float32 // Minimum similarity score to consider (0.7-0.9 recommended)
	MaxSimilarCases     int     // Maximum number of similar cases to retrieve
}

// fromAlert creates an AlertCase from an Alert
func FromAlert(alert *types.Alert, category, analysis, debugInfo string) *AlertCase {
	now := time.Now()
	return &AlertCase{
		AlertName:     alert.Labels["alertname"],
		Severity:      alert.Labels["severity"],
		Category:      category,
		Summary:       alert.Annotations["summary"],
		Namespace:     alert.Labels["namespace"],
		PodName:       alert.Labels["pod"],
		ContainerName: alert.Labels["container"],
		Analysis:      analysis,
		DebugInfo:     debugInfo,
		Validated:     true, // Assuming all stored cases are validated
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

// GetSearchText returns a text representation for embedding generation
func (ac *AlertCase) GetSearchText() string {
	return ac.AlertName + " " + ac.Severity + " " + ac.Summary + " " +
		ac.Namespace + " " + ac.Analysis
}
