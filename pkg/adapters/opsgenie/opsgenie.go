package opsgenie

import (
	"strings"
	"time"

	"github.com/valentinpelus/k8flex/pkg/types"
)

// Webhook represents a webhook from Opsgenie
// Reference: https://docs.opsgenie.com/docs/webhook-integration
type Webhook struct {
	Source struct {
		Name string `json:"name"`
		Type string `json:"type"`
	} `json:"source"`
	Alert struct {
		AlertID     string                 `json:"alertId"`
		Message     string                 `json:"message"`
		Status      string                 `json:"status"`
		Tags        []string               `json:"tags"`
		TinyID      string                 `json:"tinyId"`
		Entity      string                 `json:"entity"`
		Alias       string                 `json:"alias"`
		CreatedAt   int64                  `json:"createdAt"`
		UpdatedAt   int64                  `json:"updatedAt"`
		Username    string                 `json:"username"`
		UserID      string                 `json:"userId"`
		Description string                 `json:"description"`
		Priority    string                 `json:"priority"`
		Details     map[string]interface{} `json:"details"`
	} `json:"alert"`
	Action          string `json:"action"`
	IntegrationID   string `json:"integrationId"`
	IntegrationName string `json:"integrationName"`
}

// Adapter converts Opsgenie webhooks to k8flex alerts
type Adapter struct {
	Webhook Webhook
}

// ToAlert converts Opsgenie alerts to k8flex internal Alert format
func (a *Adapter) ToAlert() ([]types.Alert, error) {
	labels := make(map[string]string)
	annotations := make(map[string]string)

	// Extract labels from tags
	for _, tag := range a.Webhook.Alert.Tags {
		parts := strings.SplitN(tag, ":", 2)
		if len(parts) == 2 {
			labels[parts[0]] = parts[1]
		} else {
			labels[tag] = "true"
		}
	}

	// Extract labels from details
	if a.Webhook.Alert.Details != nil {
		for k, v := range a.Webhook.Alert.Details {
			if str, ok := v.(string); ok {
				switch strings.ToLower(k) {
				case "namespace", "k8s_namespace":
					labels["namespace"] = str
				case "pod", "pod_name":
					labels["pod"] = str
				case "service", "service_name":
					labels["service"] = str
				case "cluster":
					labels["cluster"] = str
				default:
					labels[k] = str
				}
			}
		}
	}

	// Add annotations
	annotations["summary"] = a.Webhook.Alert.Message
	annotations["description"] = a.Webhook.Alert.Description
	annotations["priority"] = a.Webhook.Alert.Priority
	annotations["entity"] = a.Webhook.Alert.Entity

	// Determine status
	status := "firing"
	if a.Webhook.Alert.Status == "closed" {
		status = "resolved"
	}

	alert := types.Alert{
		Status:      status,
		Labels:      labels,
		Annotations: annotations,
		StartsAt:    time.Unix(a.Webhook.Alert.CreatedAt/1000, 0),
		EndsAt:      time.Time{},
	}

	return []types.Alert{alert}, nil
}

// GetSource returns the source identifier
func (a *Adapter) GetSource() string {
	return "Opsgenie"
}
