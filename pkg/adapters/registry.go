package adapters

import (
	"encoding/json"
	"fmt"

	"github.com/valentinpelus/k8flex/pkg/adapters/alertmanager"
	"github.com/valentinpelus/k8flex/pkg/adapters/datadog"
	"github.com/valentinpelus/k8flex/pkg/adapters/grafana"
	"github.com/valentinpelus/k8flex/pkg/adapters/newrelic"
	"github.com/valentinpelus/k8flex/pkg/adapters/opsgenie"
	"github.com/valentinpelus/k8flex/pkg/adapters/pagerduty"
	"github.com/valentinpelus/k8flex/pkg/adapters/victorops"
	"github.com/valentinpelus/k8flex/pkg/types"
)

// AlertAdapter interface for converting various alert formats to internal Alert type
type AlertAdapter interface {
	ToAlert() ([]types.Alert, error)
	GetSource() string
}

// Registry manages enabled alerting system adapters
type Registry struct {
	enabledAdapters map[string]bool
}

// NewRegistry creates a new adapter registry with specified enabled adapters
// Pass adapter names: "alertmanager", "pagerduty", "grafana", "datadog", "opsgenie", "victorops", "newrelic"
// If no adapters specified, all are enabled by default
func NewRegistry(enabledAdapters []string) *Registry {
	registry := &Registry{
		enabledAdapters: make(map[string]bool),
	}

	// If no adapters specified, enable all
	if len(enabledAdapters) == 0 {
		enabledAdapters = []string{
			"alertmanager",
			"pagerduty",
			"grafana",
			"datadog",
			"opsgenie",
			"victorops",
			"newrelic",
		}
	}

	for _, adapter := range enabledAdapters {
		registry.enabledAdapters[adapter] = true
	}

	return registry
}

// IsEnabled checks if an adapter is enabled
func (r *Registry) IsEnabled(adapterName string) bool {
	return r.enabledAdapters[adapterName]
}

// DetectAndConvert attempts to detect the alert source and convert to internal format
// Returns the alerts, source name, and any error
func (r *Registry) DetectAndConvert(body []byte) ([]types.Alert, string, error) {
	var errors []string

	// Try Alertmanager
	if r.IsEnabled("alertmanager") {
		if alerts, source, err := r.tryAlertmanager(body); err == nil {
			return alerts, source, nil
		} else {
			errors = append(errors, fmt.Sprintf("alertmanager: %v", err))
		}
	}

	// Try PagerDuty
	if r.IsEnabled("pagerduty") {
		if alerts, source, err := r.tryPagerDuty(body); err == nil {
			return alerts, source, nil
		} else {
			errors = append(errors, fmt.Sprintf("pagerduty: %v", err))
		}
	}

	// Try Grafana
	if r.IsEnabled("grafana") {
		if alerts, source, err := r.tryGrafana(body); err == nil {
			return alerts, source, nil
		} else {
			errors = append(errors, fmt.Sprintf("grafana: %v", err))
		}
	}

	// Try Datadog
	if r.IsEnabled("datadog") {
		if alerts, source, err := r.tryDatadog(body); err == nil {
			return alerts, source, nil
		} else {
			errors = append(errors, fmt.Sprintf("datadog: %v", err))
		}
	}

	// Try Opsgenie
	if r.IsEnabled("opsgenie") {
		if alerts, source, err := r.tryOpsgenie(body); err == nil {
			return alerts, source, nil
		} else {
			errors = append(errors, fmt.Sprintf("opsgenie: %v", err))
		}
	}

	// Try VictorOps
	if r.IsEnabled("victorops") {
		if alerts, source, err := r.tryVictorOps(body); err == nil {
			return alerts, source, nil
		} else {
			errors = append(errors, fmt.Sprintf("victorops: %v", err))
		}
	}

	// Try New Relic
	if r.IsEnabled("newrelic") {
		if alerts, source, err := r.tryNewRelic(body); err == nil {
			return alerts, source, nil
		} else {
			errors = append(errors, fmt.Sprintf("newrelic: %v", err))
		}
	}

	return nil, "", fmt.Errorf("failed to parse webhook from any enabled source: %v", errors)
}

func (r *Registry) tryAlertmanager(body []byte) ([]types.Alert, string, error) {
	var webhook alertmanager.Webhook
	if err := json.Unmarshal(body, &webhook); err != nil {
		return nil, "", err
	}

	// Validate it's an Alertmanager webhook by checking required fields
	if webhook.GroupKey == "" && len(webhook.Alerts) == 0 {
		return nil, "", fmt.Errorf("not an Alertmanager webhook")
	}

	adapter := &alertmanager.Adapter{Webhook: webhook}
	alerts, err := adapter.ToAlert()
	return alerts, adapter.GetSource(), err
}

func (r *Registry) tryPagerDuty(body []byte) ([]types.Alert, string, error) {
	var webhook pagerduty.Webhook
	if err := json.Unmarshal(body, &webhook); err != nil {
		return nil, "", err
	}

	// Validate it's a PagerDuty webhook
	if len(webhook.Messages) == 0 {
		return nil, "", fmt.Errorf("not a PagerDuty webhook")
	}

	adapter := &pagerduty.Adapter{Webhook: webhook}
	alerts, err := adapter.ToAlert()
	return alerts, adapter.GetSource(), err
}

func (r *Registry) tryGrafana(body []byte) ([]types.Alert, string, error) {
	var webhook grafana.Webhook
	if err := json.Unmarshal(body, &webhook); err != nil {
		return nil, "", err
	}

	// Validate it's a Grafana webhook by checking for Grafana-specific fields
	if webhook.Title == "" && webhook.Message == "" {
		return nil, "", fmt.Errorf("not a Grafana webhook")
	}

	adapter := &grafana.Adapter{Webhook: webhook}
	alerts, err := adapter.ToAlert()
	return alerts, adapter.GetSource(), err
}

func (r *Registry) tryDatadog(body []byte) ([]types.Alert, string, error) {
	var webhook datadog.Webhook
	if err := json.Unmarshal(body, &webhook); err != nil {
		return nil, "", err
	}

	// Validate it's a Datadog webhook
	if webhook.EventType == "" && webhook.AlertType == "" {
		return nil, "", fmt.Errorf("not a Datadog webhook")
	}

	adapter := &datadog.Adapter{Webhook: webhook}
	alerts, err := adapter.ToAlert()
	return alerts, adapter.GetSource(), err
}

func (r *Registry) tryOpsgenie(body []byte) ([]types.Alert, string, error) {
	var webhook opsgenie.Webhook
	if err := json.Unmarshal(body, &webhook); err != nil {
		return nil, "", err
	}

	// Validate it's an Opsgenie webhook
	if webhook.Action == "" && webhook.Alert.AlertID == "" {
		return nil, "", fmt.Errorf("not an Opsgenie webhook")
	}

	adapter := &opsgenie.Adapter{Webhook: webhook}
	alerts, err := adapter.ToAlert()
	return alerts, adapter.GetSource(), err
}

func (r *Registry) tryVictorOps(body []byte) ([]types.Alert, string, error) {
	var webhook victorops.Webhook
	if err := json.Unmarshal(body, &webhook); err != nil {
		return nil, "", err
	}

	// Validate it's a VictorOps webhook
	if webhook.EntityID == "" && webhook.MessageType == "" {
		return nil, "", fmt.Errorf("not a VictorOps webhook")
	}

	adapter := &victorops.Adapter{Webhook: webhook}
	alerts, err := adapter.ToAlert()
	return alerts, adapter.GetSource(), err
}

func (r *Registry) tryNewRelic(body []byte) ([]types.Alert, string, error) {
	var webhook newrelic.Webhook
	if err := json.Unmarshal(body, &webhook); err != nil {
		return nil, "", err
	}

	// Validate it's a New Relic webhook
	if webhook.ConditionName == "" && webhook.PolicyName == "" {
		return nil, "", fmt.Errorf("not a New Relic webhook")
	}

	adapter := &newrelic.Adapter{Webhook: webhook}
	alerts, err := adapter.ToAlert()
	return alerts, adapter.GetSource(), err
}
