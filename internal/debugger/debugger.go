package debugger

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/valentinpelus/k8flex/pkg/kubernetes"
	"github.com/valentinpelus/k8flex/pkg/types"
)

// Debugger handles gathering debug information for alerts
type Debugger struct {
	k8sClient *kubernetes.Client
}

// New creates a new debugger
func New(k8sClient *kubernetes.Client) *Debugger {
	return &Debugger{
		k8sClient: k8sClient,
	}
}

// GatherDebugInfo collects contextually relevant debug information based on alert category
func (d *Debugger) GatherDebugInfo(ctx context.Context, alert types.Alert, category string) string {
	namespace := alert.Labels["namespace"]
	podName := alert.Labels["pod"]
	serviceName := alert.Labels["service"]

	var debugInfo strings.Builder

	d.writeHeader(&debugInfo, alert, namespace)

	// Always gather namespace events - they provide crucial context for any alert
	d.gatherNamespaceEvents(ctx, &debugInfo, namespace)

	// Gather debug info based on the category determined by Ollama
	log.Printf("Gathering debug info for category: %s", category)

	switch category {
	case "pod-crash", "pod-restart":
		// Pod issues: logs, pod details, resource constraints
		d.gatherPodLogs(ctx, &debugInfo, namespace, podName)
		d.gatherPodDetails(ctx, &debugInfo, namespace, podName)
		d.gatherResourceInfo(ctx, &debugInfo, namespace, podName)

	case "memory", "cpu", "disk":
		// Resource issues: pod resources, node capacity
		d.gatherPodDetails(ctx, &debugInfo, namespace, podName)
		d.gatherResourceInfo(ctx, &debugInfo, namespace, podName)
		d.gatherNodeResources(ctx, &debugInfo, namespace, podName)

	case "network":
		// Network issues: service info, network policies, pod network
		d.gatherServiceInfo(ctx, &debugInfo, namespace, serviceName)
		d.gatherNetworkInfo(ctx, &debugInfo, namespace, podName)
		if podName != "" {
			d.gatherPodDetails(ctx, &debugInfo, namespace, podName)
		}

	case "service":
		// Service issues: service info, endpoints
		d.gatherServiceInfo(ctx, &debugInfo, namespace, serviceName)
		if podName != "" {
			d.gatherPodDetails(ctx, &debugInfo, namespace, podName)
		}

	case "hpa", "deployment":
		// HPA/Deployment issues: resource metrics, pod details (not node capacity)
		d.gatherResourceInfo(ctx, &debugInfo, namespace, podName)
		if podName != "" {
			d.gatherPodDetails(ctx, &debugInfo, namespace, podName)
		}

	case "node":
		// Node issues: node status, node resources
		if podName != "" {
			d.gatherPodDetails(ctx, &debugInfo, namespace, podName)
			d.gatherNodeStatus(ctx, &debugInfo, namespace, podName)
			d.gatherNodeResources(ctx, &debugInfo, namespace, podName)
		}

	default:
		// Unknown alert type: gather pod and service basics
		log.Printf("Unknown category '%s', gathering basic info", category)
		if podName != "" {
			d.gatherPodLogs(ctx, &debugInfo, namespace, podName)
			d.gatherPodDetails(ctx, &debugInfo, namespace, podName)
		}
		if serviceName != "" {
			d.gatherServiceInfo(ctx, &debugInfo, namespace, serviceName)
		}
	}

	return debugInfo.String()
}

// writeHeader writes the debug report header
func (d *Debugger) writeHeader(debugInfo *strings.Builder, alert types.Alert, namespace string) {
	debugInfo.WriteString("=== AI-Powered Debug Analysis ===\n")
	debugInfo.WriteString(fmt.Sprintf("Alert: %s\n", alert.Labels["alertname"]))
	debugInfo.WriteString(fmt.Sprintf("Severity: %s\n", alert.Labels["severity"]))
	debugInfo.WriteString(fmt.Sprintf("Namespace: %s\n", namespace))
	debugInfo.WriteString(fmt.Sprintf("Time: %s\n\n", alert.StartsAt.Format(time.RFC3339)))

	if summary := alert.Annotations["summary"]; summary != "" {
		debugInfo.WriteString(fmt.Sprintf("Summary: %s\n", summary))
	}
	if description := alert.Annotations["description"]; description != "" {
		debugInfo.WriteString(fmt.Sprintf("Description: %s\n\n", description))
	}
}

// gatherPodLogs retrieves and appends pod logs
func (d *Debugger) gatherPodLogs(ctx context.Context, debugInfo *strings.Builder, namespace, podName string) {
	if podName == "" {
		return
	}

	log.Printf("Fetching logs for pod: %s/%s", namespace, podName)
	logs, err := d.k8sClient.GetPodLogs(ctx, namespace, podName, 100)
	if err != nil {
		debugInfo.WriteString(fmt.Sprintf("=== Pod Logs ===\nError fetching logs: %v\n\n", err))
	} else {
		debugInfo.WriteString(fmt.Sprintf("=== Pod Logs (last 100 lines) ===\n%s\n\n", logs))
	}
}

// gatherPodDetails retrieves and appends pod description
func (d *Debugger) gatherPodDetails(ctx context.Context, debugInfo *strings.Builder, namespace, podName string) {
	if podName == "" {
		return
	}

	log.Printf("Describing pod: %s/%s", namespace, podName)
	desc, err := d.k8sClient.DescribePod(ctx, namespace, podName)
	if err != nil {
		debugInfo.WriteString(fmt.Sprintf("=== Pod Details ===\nError describing pod: %v\n\n", err))
	} else {
		debugInfo.WriteString(fmt.Sprintf("=== Pod Details ===\n%s\n\n", desc))
	}
}

// gatherNamespaceEvents retrieves and appends namespace events
func (d *Debugger) gatherNamespaceEvents(ctx context.Context, debugInfo *strings.Builder, namespace string) {
	log.Printf("Fetching events in namespace: %s", namespace)
	events, err := d.k8sClient.GetNamespaceEvents(ctx, namespace, 50)
	if err != nil {
		debugInfo.WriteString(fmt.Sprintf("=== Recent Events ===\nError fetching events: %v\n\n", err))
	} else {
		debugInfo.WriteString(fmt.Sprintf("=== Recent Events ===\n%s\n\n", events))
	}
}

// gatherServiceInfo retrieves and appends service information
func (d *Debugger) gatherServiceInfo(ctx context.Context, debugInfo *strings.Builder, namespace, serviceName string) {
	if serviceName == "" {
		return
	}

	log.Printf("Checking service: %s/%s", namespace, serviceName)
	svcCheck, err := d.k8sClient.CheckService(ctx, namespace, serviceName)
	if err != nil {
		debugInfo.WriteString(fmt.Sprintf("=== Service Check ===\nError checking service: %v\n\n", err))
	} else {
		debugInfo.WriteString(fmt.Sprintf("=== Service Check ===\n%s\n\n", svcCheck))
	}
}

// gatherNetworkInfo retrieves and appends network information
func (d *Debugger) gatherNetworkInfo(ctx context.Context, debugInfo *strings.Builder, namespace, podName string) {
	if podName == "" {
		return
	}

	log.Printf("Checking network for pod: %s/%s", namespace, podName)
	netCheck, err := d.k8sClient.CheckPodNetwork(ctx, namespace, podName)
	if err != nil {
		debugInfo.WriteString(fmt.Sprintf("=== Network Check ===\nError checking network: %v\n\n", err))
	} else {
		debugInfo.WriteString(fmt.Sprintf("=== Network Check ===\n%s\n\n", netCheck))
	}
}

// gatherResourceInfo retrieves and appends resource information
func (d *Debugger) gatherResourceInfo(ctx context.Context, debugInfo *strings.Builder, namespace, podName string) {
	if podName == "" {
		return
	}

	log.Printf("Checking resources for pod: %s/%s", namespace, podName)
	metrics, err := d.k8sClient.CheckPodResources(ctx, namespace, podName)
	if err != nil {
		debugInfo.WriteString(fmt.Sprintf("=== Resource Metrics ===\nError checking resources: %v\n\n", err))
	} else {
		debugInfo.WriteString(fmt.Sprintf("=== Resource Metrics ===\n%s\n\n", metrics))
	}
}

// gatherNodeResources retrieves and appends node resource information
func (d *Debugger) gatherNodeResources(ctx context.Context, debugInfo *strings.Builder, namespace, podName string) {
	if podName == "" {
		return
	}

	log.Printf("Checking node resources for pod: %s/%s", namespace, podName)
	nodeResources, err := d.k8sClient.CheckNodeResources(ctx, namespace, podName)
	if err != nil {
		debugInfo.WriteString(fmt.Sprintf("=== Node Resource Metrics ===\nError checking node resources: %v\n\n", err))
	} else {
		debugInfo.WriteString(fmt.Sprintf("=== Node Resource Metrics ===\n%s\n\n", nodeResources))
	}
}

// gatherNodeStatus retrieves and appends node status information
func (d *Debugger) gatherNodeStatus(ctx context.Context, debugInfo *strings.Builder, namespace, podName string) {
	if podName == "" {
		return
	}

	log.Printf("Checking node status for pod: %s/%s", namespace, podName)
	nodeStatus, err := d.k8sClient.CheckNodeStatus(ctx, namespace, podName)
	if err != nil {
		debugInfo.WriteString(fmt.Sprintf("=== Node Status ===\nError checking node status: %v\n\n", err))
	} else {
		debugInfo.WriteString(fmt.Sprintf("=== Node Status ===\n%s\n\n", nodeStatus))
	}
}
