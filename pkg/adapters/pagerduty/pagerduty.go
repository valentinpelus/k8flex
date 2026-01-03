package pagerduty

import (
	"strings"
	"time"

	"github.com/valentinpelus/k8flex/pkg/types"
)

// Webhook represents a webhook from PagerDuty
// Reference: https://developer.pagerduty.com/docs/db0fa8c8984fc-overview
type Webhook struct {
	Messages []Message `json:"messages"`
}

type Message struct {
	ID         string     `json:"id"`
	Event      string     `json:"event"` // incident.trigger, incident.acknowledge, incident.resolve
	CreatedOn  time.Time  `json:"created_on"`
	Incident   Incident   `json:"incident"`
	LogEntries []LogEntry `json:"log_entries,omitempty"`
}

type Incident struct {
	ID             string                 `json:"id"`
	IncidentNumber int                    `json:"incident_number"`
	Title          string                 `json:"title"`
	Description    string                 `json:"description"`
	CreatedAt      time.Time              `json:"created_at"`
	Status         string                 `json:"status"`
	IncidentKey    string                 `json:"incident_key"`
	Service        Service                `json:"service"`
	Assignments    []interface{}          `json:"assignments"`
	Priority       interface{}            `json:"priority"`
	Urgency        string                 `json:"urgency"`
	CustomDetails  map[string]interface{} `json:"custom_details,omitempty"`
}

type Service struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Summary string `json:"summary"`
}

type LogEntry struct {
	Type    string `json:"type"`
	Summary string `json:"summary"`
}

// Adapter converts PagerDuty webhooks to k8flex alerts
type Adapter struct {
	Webhook Webhook
}

// ToAlert converts PagerDuty incidents to k8flex internal Alert format
func (a *Adapter) ToAlert() ([]types.Alert, error) {
	var alerts []types.Alert

	for _, msg := range a.Webhook.Messages {
		// Only process trigger events
		if msg.Event != "incident.trigger" {
			continue
		}

		incident := msg.Incident
		labels := make(map[string]string)
		annotations := make(map[string]string)

		// Extract labels from custom_details if available
		if incident.CustomDetails != nil {
			for k, v := range incident.CustomDetails {
				if str, ok := v.(string); ok {
					// Try to extract kubernetes-related fields
					switch strings.ToLower(k) {
					case "namespace", "k8s_namespace", "kubernetes_namespace":
						labels["namespace"] = str
					case "pod", "pod_name", "k8s_pod":
						labels["pod"] = str
					case "service", "service_name", "k8s_service":
						labels["service"] = str
					case "node", "node_name", "k8s_node":
						labels["node"] = str
					default:
						labels[k] = str
					}
				}
			}
		}

		// Add incident metadata to labels
		labels["incident_id"] = incident.ID
		labels["incident_key"] = incident.IncidentKey
		labels["urgency"] = incident.Urgency
		labels["service_name"] = incident.Service.Name

		// Add annotations
		annotations["summary"] = incident.Title
		annotations["description"] = incident.Description

		// Determine status
		status := "firing"
		if incident.Status == "resolved" {
			status = "resolved"
		}

		alerts = append(alerts, types.Alert{
			Status:      status,
			Labels:      labels,
			Annotations: annotations,
			StartsAt:    incident.CreatedAt,
			EndsAt:      time.Time{},
		})
	}

	return alerts, nil
}

// GetSource returns the source identifier
func (a *Adapter) GetSource() string {
	return "PagerDuty"
}
