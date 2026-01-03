package newrelic

import (
	"strings"
	"time"

	"github.com/valentinpelus/k8flex/pkg/types"
)

// Webhook represents a webhook from New Relic
// Reference: https://docs.newrelic.com/docs/alerts-applied-intelligence/notifications/notification-integrations/
type Webhook struct {
	AccountID              int64    `json:"account_id"`
	AccountName            string   `json:"account_name"`
	ConditionID            int64    `json:"condition_id"`
	ConditionName          string   `json:"condition_name"`
	CurrentState           string   `json:"current_state"`
	Details                string   `json:"details"`
	EventType              string   `json:"event_type"`
	IncidentAcknowledgeURL string   `json:"incident_acknowledge_url"`
	IncidentID             int64    `json:"incident_id"`
	IncidentURL            string   `json:"incident_url"`
	Owner                  string   `json:"owner"`
	PolicyName             string   `json:"policy_name"`
	PolicyURL              string   `json:"policy_url"`
	RunbookURL             string   `json:"runbook_url"`
	Severity               string   `json:"severity"`
	Targets                []Target `json:"targets"`
	Timestamp              int64    `json:"timestamp"`
	TimestampUTCString     string   `json:"timestamp_utc_string"`
	ViolationCallbackURL   string   `json:"violation_callback_url"`
	ViolationChartURL      string   `json:"violation_chart_url"`
}

type Target struct {
	ID      string                 `json:"id"`
	Name    string                 `json:"name"`
	Link    string                 `json:"link"`
	Product string                 `json:"product"`
	Type    string                 `json:"type"`
	Labels  map[string]interface{} `json:"labels"`
}

// Adapter converts New Relic webhooks to k8flex alerts
type Adapter struct {
	Webhook Webhook
}

// ToAlert converts New Relic alerts to k8flex internal Alert format
func (a *Adapter) ToAlert() ([]types.Alert, error) {
	labels := make(map[string]string)
	annotations := make(map[string]string)

	// Extract labels from targets
	for _, target := range a.Webhook.Targets {
		// Try to extract namespace/pod from target name
		// Format: namespace:pod-name:container-name
		if target.Product == "KUBERNETES" {
			parts := strings.Split(target.Name, ":")
			if len(parts) >= 2 {
				labels["namespace"] = parts[0]
				labels["pod"] = parts[1]
				if len(parts) >= 3 {
					labels["container"] = parts[2]
				}
			}
		}

		// Extract labels from target.Labels
		if target.Labels != nil {
			for k, v := range target.Labels {
				if str, ok := v.(string); ok {
					labels[k] = str
				}
			}
		}
	}

	// Add metadata
	labels["policy_name"] = a.Webhook.PolicyName
	labels["condition_name"] = a.Webhook.ConditionName
	labels["severity"] = strings.ToLower(a.Webhook.Severity)

	// Add annotations
	annotations["summary"] = a.Webhook.ConditionName
	annotations["description"] = a.Webhook.Details
	annotations["runbook_url"] = a.Webhook.RunbookURL
	annotations["incident_url"] = a.Webhook.IncidentURL

	// Determine status
	status := "firing"
	if a.Webhook.CurrentState == "closed" {
		status = "resolved"
	}

	alert := types.Alert{
		Status:       status,
		Labels:       labels,
		Annotations:  annotations,
		StartsAt:     time.Unix(a.Webhook.Timestamp/1000, 0),
		EndsAt:       time.Time{},
		GeneratorURL: a.Webhook.IncidentURL,
	}

	return []types.Alert{alert}, nil
}

// GetSource returns the source identifier
func (a *Adapter) GetSource() string {
	return "New Relic"
}
