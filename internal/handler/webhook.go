package handler

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/valentinpelus/k8flex/internal/processor"
	"github.com/valentinpelus/k8flex/pkg/types"
)

// WebhookHandler handles incoming Alertmanager webhooks
type WebhookHandler struct {
	processor *processor.AlertProcessor
}

// NewWebhookHandler creates a new webhook handler
func NewWebhookHandler(proc *processor.AlertProcessor) *WebhookHandler {
	return &WebhookHandler{
		processor: proc,
	}
}

// HandleWebhook processes incoming webhook requests
func (h *WebhookHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Failed to read request body: %v", err)
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var webhook types.AlertmanagerWebhook
	if err := json.Unmarshal(body, &webhook); err != nil {
		log.Printf("Failed to parse webhook: %v", err)
		http.Error(w, "Failed to parse webhook", http.StatusBadRequest)
		return
	}

	log.Printf("Received webhook with %d alerts, status: %s", len(webhook.Alerts), webhook.Status)

	// Process each alert asynchronously
	go func() {
		for _, alert := range webhook.Alerts {
			// Process if status is "firing" or empty (default to firing)
			if alert.Status == "firing" || alert.Status == "" {
				h.processor.ProcessAlert(alert)
			}
		}
	}()

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"accepted"}`))
}

// HandleHealth handles health check requests
func HandleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"healthy"}`))
}
