package knowledge

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq" // PostgreSQL driver
)

// KnowledgeBase manages storage and retrieval of alert cases
type KnowledgeBase struct {
	db                  *sql.DB
	embeddings          EmbeddingGenerator
	similarityThreshold float32
	maxSimilarCases     int
}

// NewKnowledgeBase creates a new knowledge base instance
func NewKnowledgeBase(config *KnowledgeBaseConfig) (*KnowledgeBase, error) {
	if config.DatabaseURL == "" {
		return nil, fmt.Errorf("database URL is required")
	}

	db, err := sql.Open("postgres", config.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Initialize embeddings generator
	var embedGen EmbeddingGenerator
	switch config.EmbeddingProvider {
	case "openai":
		if config.EmbeddingAPIKey == "" {
			return nil, fmt.Errorf("OpenAI API key is required for embedding provider")
		}
		embedGen = NewOpenAIEmbeddings(config.EmbeddingAPIKey, config.EmbeddingModel)
	case "gemini":
		if config.EmbeddingAPIKey == "" {
			return nil, fmt.Errorf("Gemini API key is required for embedding provider")
		}
		embedGen = NewGeminiEmbeddings(config.EmbeddingAPIKey, config.EmbeddingModel)
	default:
		return nil, fmt.Errorf("unsupported embedding provider: %s", config.EmbeddingProvider)
	}

	// Set defaults
	similarityThreshold := config.SimilarityThreshold
	if similarityThreshold == 0 {
		similarityThreshold = 0.75 // Default to 75% similarity
	}

	maxSimilarCases := config.MaxSimilarCases
	if maxSimilarCases == 0 {
		maxSimilarCases = 5 // Default to top 5 similar cases
	}

	return &KnowledgeBase{
		db:                  db,
		embeddings:          embedGen,
		similarityThreshold: similarityThreshold,
		maxSimilarCases:     maxSimilarCases,
	}, nil
}

// Close closes the database connection
func (kb *KnowledgeBase) Close() error {
	if kb.db != nil {
		return kb.db.Close()
	}
	return nil
}

// Store saves a validated alert case to the knowledge base
func (kb *KnowledgeBase) Store(ctx context.Context, alertCase *AlertCase) error {
	// Generate embedding for the case
	searchText := alertCase.GetSearchText()
	embedding, err := kb.embeddings.Generate(ctx, searchText)
	if err != nil {
		return fmt.Errorf("failed to generate embedding: %w", err)
	}

	// Generate UUID if not set
	if alertCase.ID == "" {
		alertCase.ID = uuid.New().String()
	}

	// Store in database
	query := `
		INSERT INTO alert_cases (
			id, alert_name, severity, category, summary, namespace, 
			pod_name, container_name, analysis, debug_info, validated, 
			embedding, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		ON CONFLICT (id) DO UPDATE SET
			category = EXCLUDED.category,
			analysis = EXCLUDED.analysis,
			debug_info = EXCLUDED.debug_info,
			validated = EXCLUDED.validated,
			embedding = EXCLUDED.embedding,
			updated_at = EXCLUDED.updated_at
	`

	_, err = kb.db.ExecContext(ctx, query,
		alertCase.ID,
		alertCase.AlertName,
		alertCase.Severity,
		alertCase.Category,
		alertCase.Summary,
		alertCase.Namespace,
		alertCase.PodName,
		alertCase.ContainerName,
		alertCase.Analysis,
		alertCase.DebugInfo,
		alertCase.Validated,
		pgvectorString(embedding), // Convert to pgvector format
		alertCase.CreatedAt,
		alertCase.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to store alert case: %w", err)
	}

	log.Printf("Stored alert case: %s (category: %s)", alertCase.AlertName, alertCase.Category)
	return nil
}

// FindSimilar finds similar cases from the knowledge base
func (kb *KnowledgeBase) FindSimilar(ctx context.Context, searchText string) ([]*SimilarCase, error) {
	// Generate embedding for the search text
	embedding, err := kb.embeddings.Generate(ctx, searchText)
	if err != nil {
		return nil, fmt.Errorf("failed to generate search embedding: %w", err)
	}

	// Query for similar cases using cosine similarity
	// Using pgvector's <=> operator for cosine distance (1 - similarity)
	query := `
		SELECT 
			id, alert_name, severity, category, summary, namespace,
			pod_name, container_name, analysis, debug_info, validated,
			created_at, updated_at,
			1 - (embedding <=> $1::vector) as similarity
		FROM alert_cases
		WHERE validated = true
			AND 1 - (embedding <=> $1::vector) >= $2
		ORDER BY embedding <=> $1::vector
		LIMIT $3
	`

	rows, err := kb.db.QueryContext(ctx, query,
		pgvectorString(embedding),
		kb.similarityThreshold,
		kb.maxSimilarCases,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query similar cases: %w", err)
	}
	defer rows.Close()

	var similarCases []*SimilarCase
	for rows.Next() {
		var ac AlertCase
		var similarity float32

		err := rows.Scan(
			&ac.ID,
			&ac.AlertName,
			&ac.Severity,
			&ac.Category,
			&ac.Summary,
			&ac.Namespace,
			&ac.PodName,
			&ac.ContainerName,
			&ac.Analysis,
			&ac.DebugInfo,
			&ac.Validated,
			&ac.CreatedAt,
			&ac.UpdatedAt,
			&similarity,
		)
		if err != nil {
			log.Printf("Warning: failed to scan row: %v", err)
			continue
		}

		similarCases = append(similarCases, &SimilarCase{
			Case:       &ac,
			Similarity: similarity,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	log.Printf("Found %d similar cases (threshold: %.2f)", len(similarCases), kb.similarityThreshold)
	return similarCases, nil
}

// GetStats returns statistics about the knowledge base
func (kb *KnowledgeBase) GetStats(ctx context.Context) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Total cases
	var totalCases int
	err := kb.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM alert_cases WHERE validated = true").Scan(&totalCases)
	if err != nil {
		return nil, fmt.Errorf("failed to get total cases: %w", err)
	}
	stats["total_cases"] = totalCases

	// Cases by category
	categoryQuery := `
		SELECT category, COUNT(*) as count 
		FROM alert_cases 
		WHERE validated = true 
		GROUP BY category 
		ORDER BY count DESC
	`
	rows, err := kb.db.QueryContext(ctx, categoryQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to get category stats: %w", err)
	}
	defer rows.Close()

	categoryCounts := make(map[string]int)
	for rows.Next() {
		var category string
		var count int
		if err := rows.Scan(&category, &count); err != nil {
			continue
		}
		categoryCounts[category] = count
	}
	stats["by_category"] = categoryCounts

	// Latest case timestamp
	var latestCase time.Time
	err = kb.db.QueryRowContext(ctx, "SELECT MAX(created_at) FROM alert_cases WHERE validated = true").Scan(&latestCase)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get latest case: %w", err)
	}
	if !latestCase.IsZero() {
		stats["latest_case"] = latestCase
	}

	return stats, nil
}

// pgvectorString converts a float32 slice to pgvector format string
func pgvectorString(embedding []float32) string {
	if len(embedding) == 0 {
		return "[]"
	}

	result := "["
	for i, val := range embedding {
		if i > 0 {
			result += ","
		}
		result += fmt.Sprintf("%f", val)
	}
	result += "]"
	return result
}
