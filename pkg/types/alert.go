package types

import "time"

// AlertmanagerWebhook represents the webhook payload from Alertmanager
// Reference: https://prometheus.io/docs/alerting/latest/configuration/#webhook_config
type AlertmanagerWebhook struct {
	Version  string  `json:"version"`
	GroupKey string  `json:"groupKey"`
	Status   string  `json:"status"`
	Alerts   []Alert `json:"alerts"`
}

// Alert represents a single alert from Alertmanager
type Alert struct {
	Status       string            `json:"status"`
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	StartsAt     time.Time         `json:"startsAt"`
	EndsAt       time.Time         `json:"endsAt"`
	GeneratorURL string            `json:"generatorURL"`
}

// DebugResult contains all debug information gathered for an alert
type DebugResult struct {
	Alert           Alert
	PodLogs         string
	PodDescription  string
	EventsList      string
	ServiceCheck    string
	NetworkCheck    string
	ResourceMetrics string
	Analysis        string
}
