package victorops

import (
	"strings"
	"time"

	"github.com/valentinpelus/k8flex/pkg/types"
)

// Webhook represents a webhook from VictorOps (Splunk On-Call)
// Reference: https://help.victorops.com/knowledge-base/custom-outbound-webhooks/
type Webhook struct {
	EntityID          string `json:"entity_id"`
	EntityDisplayName string `json:"entity_display_name"`
	StateStartTime    int64  `json:"state_start_time"`
	StateMessage      string `json:"state_message"`
	MonitoringTool    string `json:"monitoring_tool"`
	MessageType       string `json:"message_type"`
	EntityState       string `json:"entity_state"`
	AlertURL          string `json:"alert_url"`
	CurrentPhase      string `json:"current_phase"`
	VOOrganizationID  string `json:"vo_organization_id"`
	VOUUID            string `json:"vo_uuid"`
}

// Adapter converts VictorOps webhooks to k8flex alerts
type Adapter struct {
	Webhook Webhook
}

// ToAlert converts VictorOps alerts to k8flex internal Alert format
func (a *Adapter) ToAlert() ([]types.Alert, error) {
	labels := make(map[string]string)
	annotations := make(map[string]string)

	// Try to extract namespace/pod from entity_id (format: namespace/pod-name)
	if strings.Contains(a.Webhook.EntityID, "/") {
		parts := strings.SplitN(a.Webhook.EntityID, "/", 2)
		if len(parts) == 2 {
			labels["namespace"] = parts[0]
			labels["pod"] = parts[1]
		}
	}

	// Try to extract pod from state_message
	stateMsg := a.Webhook.StateMessage
	if strings.Contains(strings.ToLower(stateMsg), "pod") {
		words := strings.Fields(stateMsg)
		for i, word := range words {
			if strings.ToLower(word) == "pod" && i+1 < len(words) {
				labels["pod"] = words[i+1]
			}
		}
	}

	// Add metadata
	labels["entity_id"] = a.Webhook.EntityID
	labels["entity_state"] = a.Webhook.EntityState
	labels["message_type"] = a.Webhook.MessageType
	labels["monitoring_tool"] = a.Webhook.MonitoringTool

	// Add annotations
	annotations["summary"] = a.Webhook.EntityDisplayName
	annotations["description"] = a.Webhook.StateMessage
	annotations["current_phase"] = a.Webhook.CurrentPhase

	// Determine status
	status := "firing"
	if a.Webhook.EntityState == "OK" || a.Webhook.EntityState == "RECOVERY" {
		status = "resolved"
	}

	alert := types.Alert{
		Status:       status,
		Labels:       labels,
		Annotations:  annotations,
		StartsAt:     time.Unix(a.Webhook.StateStartTime, 0),
		EndsAt:       time.Time{},
		GeneratorURL: a.Webhook.AlertURL,
	}

	return []types.Alert{alert}, nil
}

// GetSource returns the source identifier
func (a *Adapter) GetSource() string {
	return "VictorOps"
}
