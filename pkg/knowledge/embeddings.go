package knowledge

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// EmbeddingGenerator generates vector embeddings for text
type EmbeddingGenerator interface {
	Generate(ctx context.Context, text string) ([]float32, error)
}

// OpenAIEmbeddings generates embeddings using OpenAI API
type OpenAIEmbeddings struct {
	apiKey string
	model  string
	client *http.Client
}

// NewOpenAIEmbeddings creates a new OpenAI embeddings generator
func NewOpenAIEmbeddings(apiKey, model string) *OpenAIEmbeddings {
	if model == "" {
		model = "text-embedding-3-small" // Default to smaller, faster model
	}
	return &OpenAIEmbeddings{
		apiKey: apiKey,
		model:  model,
		client: &http.Client{},
	}
}

type openAIEmbeddingRequest struct {
	Input string `json:"input"`
	Model string `json:"model"`
}

type openAIEmbeddingResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
}

// Generate creates an embedding vector for the given text
func (e *OpenAIEmbeddings) Generate(ctx context.Context, text string) ([]float32, error) {
	reqBody := openAIEmbeddingRequest{
		Input: text,
		Model: e.model,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/embeddings", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e.apiKey)

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call OpenAI API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("OpenAI API returned status %d: %s", resp.StatusCode, string(body))
	}

	var embResp openAIEmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&embResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(embResp.Data) == 0 {
		return nil, fmt.Errorf("no embeddings returned")
	}

	return embResp.Data[0].Embedding, nil
}

// GeminiEmbeddings generates embeddings using Google Gemini API
type GeminiEmbeddings struct {
	apiKey string
	model  string
	client *http.Client
}

// NewGeminiEmbeddings creates a new Gemini embeddings generator
func NewGeminiEmbeddings(apiKey, model string) *GeminiEmbeddings {
	if model == "" {
		model = "embedding-001" // Default Gemini embedding model
	}
	return &GeminiEmbeddings{
		apiKey: apiKey,
		model:  model,
		client: &http.Client{},
	}
}

type geminiEmbeddingRequest struct {
	Requests []struct {
		Model   string `json:"model"`
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"requests"`
}

type geminiEmbeddingResponse struct {
	Embeddings []struct {
		Values []float32 `json:"values"`
	} `json:"embeddings"`
}

// Generate creates an embedding vector for the given text
func (e *GeminiEmbeddings) Generate(ctx context.Context, text string) ([]float32, error) {
	reqBody := geminiEmbeddingRequest{
		Requests: []struct {
			Model   string `json:"model"`
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		}{
			{
				Model: e.model,
				Content: struct {
					Parts []struct {
						Text string `json:"text"`
					} `json:"parts"`
				}{
					Parts: []struct {
						Text string `json:"text"`
					}{
						{Text: text},
					},
				},
			},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:batchEmbedContents?key=%s", e.model, e.apiKey)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call Gemini API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Gemini API returned status %d: %s", resp.StatusCode, string(body))
	}

	var embResp geminiEmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&embResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(embResp.Embeddings) == 0 {
		return nil, fmt.Errorf("no embeddings returned")
	}

	return embResp.Embeddings[0].Values, nil
}
