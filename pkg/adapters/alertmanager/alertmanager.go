package alertmanager

import (
	"time"

	"github.com/valentinpelus/k8flex/pkg/types"
)

// Webhook represents a webhook from Prometheus Alertmanager
// Reference: https://prometheus.io/docs/alerting/latest/configuration/#webhook_config
type Webhook struct {
	Version           string            `json:"version"`
	GroupKey          string            `json:"groupKey"`
	TruncatedAlerts   int               `json:"truncatedAlerts"`
	Status            string            `json:"status"`
	Receiver          string            `json:"receiver"`
	GroupLabels       map[string]string `json:"groupLabels"`
	CommonLabels      map[string]string `json:"commonLabels"`
	CommonAnnotations map[string]string `json:"commonAnnotations"`
	ExternalURL       string            `json:"externalURL"`
	Alerts            []Alert           `json:"alerts"`
}

type Alert struct {
	Status       string            `json:"status"`
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	StartsAt     time.Time         `json:"startsAt"`
	EndsAt       time.Time         `json:"endsAt"`
	GeneratorURL string            `json:"generatorURL"`
	Fingerprint  string            `json:"fingerprint"`
}

// Adapter converts Alertmanager webhooks to k8flex alerts
type Adapter struct {
	Webhook Webhook
}

// ToAlert converts Alertmanager alerts to k8flex internal Alert format
func (a *Adapter) ToAlert() ([]types.Alert, error) {
	var alerts []types.Alert

	for _, alert := range a.Webhook.Alerts {
		alerts = append(alerts, types.Alert{
			Status:       alert.Status,
			Labels:       alert.Labels,
			Annotations:  alert.Annotations,
			StartsAt:     alert.StartsAt,
			EndsAt:       alert.EndsAt,
			GeneratorURL: alert.GeneratorURL,
		})
	}

	return alerts, nil
}

// GetSource returns the source identifier
func (a *Adapter) GetSource() string {
	return "Alertmanager"
}
