package datadog

import (
	"strings"
	"time"

	"github.com/valentinpelus/k8flex/pkg/types"
)

// Webhook represents a webhook from Datadog
// Reference: https://docs.datadoghq.com/integrations/webhooks/
type Webhook struct {
	ID              string    `json:"id"`
	Title           string    `json:"title"`
	Body            string    `json:"body"`
	LastUpdated     time.Time `json:"last_updated"`
	EventType       string    `json:"event_type"`
	AlertTransition string    `json:"alert_transition"`
	AlertType       string    `json:"alert_type"`
	Priority        string    `json:"priority"`
	Org             Org       `json:"org"`
	Tags            []string  `json:"tags"`
	AlertScope      []string  `json:"alert_scope"`
	URL             string    `json:"url"`
	AggregationKey  string    `json:"aggregation_key"`
}

type Org struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Adapter converts Datadog webhooks to k8flex alerts
type Adapter struct {
	Webhook Webhook
}

// ToAlert converts Datadog alerts to k8flex internal Alert format
func (a *Adapter) ToAlert() ([]types.Alert, error) {
	labels := make(map[string]string)
	annotations := make(map[string]string)

	// Extract labels from tags
	for _, tag := range a.Webhook.Tags {
		parts := strings.SplitN(tag, ":", 2)
		if len(parts) == 2 {
			key := parts[0]
			value := parts[1]

			// Map common Kubernetes tags
			switch key {
			case "namespace", "kube_namespace":
				labels["namespace"] = value
			case "pod_name", "pod":
				labels["pod"] = value
			case "kube_service", "service":
				labels["service"] = value
			case "kube_deployment", "deployment":
				labels["deployment"] = value
			case "cluster_name", "cluster":
				labels["cluster"] = value
			case "container_name", "container":
				labels["container"] = value
			default:
				labels[key] = value
			}
		}
	}

	// Add annotations
	annotations["summary"] = a.Webhook.Title
	annotations["description"] = a.Webhook.Body
	annotations["alert_type"] = a.Webhook.AlertType
	annotations["priority"] = a.Webhook.Priority

	// Determine status
	status := "firing"
	if a.Webhook.AlertTransition == "Recovered" {
		status = "resolved"
	}

	alert := types.Alert{
		Status:       status,
		Labels:       labels,
		Annotations:  annotations,
		StartsAt:     a.Webhook.LastUpdated,
		EndsAt:       time.Time{},
		GeneratorURL: a.Webhook.URL,
	}

	return []types.Alert{alert}, nil
}

// GetSource returns the source identifier
func (a *Adapter) GetSource() string {
	return "Datadog"
}
