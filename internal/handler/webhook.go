package handler

import (
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/valentinpelus/k8flex/internal/processor"
	"github.com/valentinpelus/k8flex/pkg/adapters"
)

// WebhookHandler handles incoming webhooks from multiple alerting systems
type WebhookHandler struct {
	processor *processor.AlertProcessor
	registry  *adapters.Registry
}

// NewWebhookHandler creates a new webhook handler with configurable adapters
// Reads ENABLED_ADAPTERS environment variable (comma-separated list)
// Example: ENABLED_ADAPTERS=alertmanager,pagerduty,grafana
// If not set, all adapters are enabled by default
func NewWebhookHandler(proc *processor.AlertProcessor) *WebhookHandler {
	var enabledAdapters []string

	// Read enabled adapters from environment variable
	if adaptersEnv := os.Getenv("ENABLED_ADAPTERS"); adaptersEnv != "" {
		enabledAdapters = strings.Split(strings.ToLower(adaptersEnv), ",")
		for i := range enabledAdapters {
			enabledAdapters[i] = strings.TrimSpace(enabledAdapters[i])
		}
		log.Printf("Enabled adapters: %v", enabledAdapters)
	} else {
		log.Printf("No ENABLED_ADAPTERS set, all adapters enabled by default")
	}

	return &WebhookHandler{
		processor: proc,
		registry:  adapters.NewRegistry(enabledAdapters),
	}
}

// HandleWebhook processes incoming webhook requests from various alerting systems
// Supports: Alertmanager, PagerDuty, Grafana, Datadog, Opsgenie, VictorOps, New Relic
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

	// Auto-detect alert source and convert to standard Alert format
	alerts, source, err := h.registry.DetectAndConvert(body)
	if err != nil {
		log.Printf("Failed to parse webhook from any enabled source: %v", err)
		http.Error(w, "Failed to parse webhook", http.StatusBadRequest)
		return
	}

	log.Printf("Received %s webhook with %d alerts", source, len(alerts))

	// Process each alert asynchronously
	go func() {
		for _, alert := range alerts {
			// Process if status is "firing" or empty (default to firing)
			if alert.Status == "firing" || alert.Status == "" {
				h.processor.ProcessAlert(alert)
			}
		}
	}()

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"accepted","source":"` + source + `"}`))
}

// HandleHealth handles health check requests
func HandleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"healthy"}`))
}
