package kubernetes

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Client wraps the Kubernetes clientset
type Client struct {
	clientset *kubernetes.Clientset
}

// NewClient creates a new Kubernetes client
func NewClient(clientset *kubernetes.Clientset) *Client {
	return &Client{
		clientset: clientset,
	}
}

// GetPodLogs retrieves the logs for a pod
// Reference: https://pkg.go.dev/k8s.io/client-go/kubernetes/typed/core/v1#PodInterface
func (c *Client) GetPodLogs(ctx context.Context, namespace, podName string, tailLines int64) (string, error) {
	req := c.clientset.CoreV1().Pods(namespace).GetLogs(podName, &corev1.PodLogOptions{
		TailLines: &tailLines,
	})

	logs, err := req.Stream(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get logs: %w", err)
	}
	defer logs.Close()

	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, logs); err != nil {
		return "", fmt.Errorf("failed to read logs: %w", err)
	}

	return buf.String(), nil
}

// DescribePod retrieves detailed information about a pod
// Reference: https://pkg.go.dev/k8s.io/client-go/kubernetes/typed/core/v1#PodInterface
func (c *Client) DescribePod(ctx context.Context, namespace, podName string) (string, error) {
	pod, err := c.clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get pod: %w", err)
	}

	var desc strings.Builder
	desc.WriteString(fmt.Sprintf("Name: %s\n", pod.Name))
	desc.WriteString(fmt.Sprintf("Namespace: %s\n", pod.Namespace))
	desc.WriteString(fmt.Sprintf("Phase: %s\n", pod.Status.Phase))
	desc.WriteString(fmt.Sprintf("Node: %s\n", pod.Spec.NodeName))
	desc.WriteString(fmt.Sprintf("IP: %s\n", pod.Status.PodIP))
	desc.WriteString(fmt.Sprintf("Start Time: %s\n", pod.Status.StartTime))

	desc.WriteString("\nContainer Statuses:\n")
	for _, cs := range pod.Status.ContainerStatuses {
		desc.WriteString(fmt.Sprintf("  - %s: Ready=%v, RestartCount=%d\n", cs.Name, cs.Ready, cs.RestartCount))
		if cs.State.Waiting != nil {
			desc.WriteString(fmt.Sprintf("    Waiting: %s - %s\n", cs.State.Waiting.Reason, cs.State.Waiting.Message))
		}
		if cs.State.Terminated != nil {
			desc.WriteString(fmt.Sprintf("    Terminated: %s - %s (Exit Code: %d)\n",
				cs.State.Terminated.Reason, cs.State.Terminated.Message, cs.State.Terminated.ExitCode))
		}
	}

	desc.WriteString("\nConditions:\n")
	for _, cond := range pod.Status.Conditions {
		desc.WriteString(fmt.Sprintf("  - %s: %s (%s)\n", cond.Type, cond.Status, cond.Reason))
		if cond.Message != "" {
			desc.WriteString(fmt.Sprintf("    Message: %s\n", cond.Message))
		}
	}

	return desc.String(), nil
}

// GetNamespaceEvents retrieves recent events in a namespace
// Reference: https://pkg.go.dev/k8s.io/client-go/kubernetes/typed/core/v1#EventInterface
func (c *Client) GetNamespaceEvents(ctx context.Context, namespace string, limit int64) (string, error) {
	events, err := c.clientset.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{
		Limit: limit,
	})
	if err != nil {
		return "", fmt.Errorf("failed to get events: %w", err)
	}

	if len(events.Items) == 0 {
		return "No recent events found", nil
	}

	var eventDesc strings.Builder
	for _, event := range events.Items {
		eventDesc.WriteString(fmt.Sprintf("[%s] %s %s/%s: %s - %s\n",
			event.LastTimestamp.Format("15:04:05"),
			event.Type,
			event.InvolvedObject.Kind,
			event.InvolvedObject.Name,
			event.Reason,
			event.Message))
	}

	return eventDesc.String(), nil
}

// CheckService retrieves information about a service and its endpoints
// Reference: https://pkg.go.dev/k8s.io/client-go/kubernetes/typed/core/v1#ServiceInterface
func (c *Client) CheckService(ctx context.Context, namespace, serviceName string) (string, error) {
	svc, err := c.clientset.CoreV1().Services(namespace).Get(ctx, serviceName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get service: %w", err)
	}

	var svcDesc strings.Builder
	svcDesc.WriteString(fmt.Sprintf("Service: %s\n", svc.Name))
	svcDesc.WriteString(fmt.Sprintf("Type: %s\n", svc.Spec.Type))
	svcDesc.WriteString(fmt.Sprintf("ClusterIP: %s\n", svc.Spec.ClusterIP))

	svcDesc.WriteString("Ports:\n")
	for _, port := range svc.Spec.Ports {
		svcDesc.WriteString(fmt.Sprintf("  - %s: %d/%s -> %d\n", port.Name, port.Port, port.Protocol, port.TargetPort.IntVal))
	}

	svcDesc.WriteString("Selector:\n")
	for k, v := range svc.Spec.Selector {
		svcDesc.WriteString(fmt.Sprintf("  %s: %s\n", k, v))
	}

	// Get endpoints
	endpoints, err := c.clientset.CoreV1().Endpoints(namespace).Get(ctx, serviceName, metav1.GetOptions{})
	if err != nil {
		svcDesc.WriteString(fmt.Sprintf("Failed to get endpoints: %v\n", err))
	} else {
		svcDesc.WriteString("\nEndpoints:\n")
		if len(endpoints.Subsets) == 0 {
			svcDesc.WriteString("  WARNING: No endpoints available!\n")
		} else {
			for _, subset := range endpoints.Subsets {
				for _, addr := range subset.Addresses {
					svcDesc.WriteString(fmt.Sprintf("  - %s", addr.IP))
					if addr.TargetRef != nil {
						svcDesc.WriteString(fmt.Sprintf(" (Pod: %s)", addr.TargetRef.Name))
					}
					svcDesc.WriteString("\n")
				}
			}
		}
	}

	return svcDesc.String(), nil
}

// CheckPodNetwork retrieves network information for a pod
func (c *Client) CheckPodNetwork(ctx context.Context, namespace, podName string) (string, error) {
	pod, err := c.clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get pod: %w", err)
	}

	var netDesc strings.Builder
	netDesc.WriteString(fmt.Sprintf("Pod IP: %s\n", pod.Status.PodIP))
	netDesc.WriteString(fmt.Sprintf("Host IP: %s\n", pod.Status.HostIP))

	// Check network policies in namespace
	netPolicies, err := c.clientset.NetworkingV1().NetworkPolicies(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		netDesc.WriteString(fmt.Sprintf("Failed to get network policies: %v\n", err))
	} else {
		netDesc.WriteString(fmt.Sprintf("\nNetwork Policies: %d\n", len(netPolicies.Items)))
		for _, np := range netPolicies.Items {
			netDesc.WriteString(fmt.Sprintf("  - %s\n", np.Name))
		}
	}

	return netDesc.String(), nil
}

// CheckPodResources retrieves resource information for a pod
func (c *Client) CheckPodResources(ctx context.Context, namespace, podName string) (string, error) {
	pod, err := c.clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get pod: %w", err)
	}

	var resDesc strings.Builder
	resDesc.WriteString("Container Resources:\n")
	for _, container := range pod.Spec.Containers {
		resDesc.WriteString(fmt.Sprintf("\nContainer: %s\n", container.Name))

		if len(container.Resources.Requests) > 0 {
			resDesc.WriteString("  Requests:\n")
			for k, v := range container.Resources.Requests {
				resDesc.WriteString(fmt.Sprintf("    %s: %s\n", k, v.String()))
			}
		}

		if len(container.Resources.Limits) > 0 {
			resDesc.WriteString("  Limits:\n")
			for k, v := range container.Resources.Limits {
				resDesc.WriteString(fmt.Sprintf("    %s: %s\n", k, v.String()))
			}
		}

		// Check liveness and readiness probes
		if container.LivenessProbe != nil {
			resDesc.WriteString("  Liveness Probe: Configured\n")
		}
		if container.ReadinessProbe != nil {
			resDesc.WriteString("  Readiness Probe: Configured\n")
		}
	}

	return resDesc.String(), nil
}

// CheckNodeResources retrieves resource information for a node on which a pod is running
func (c *Client) CheckNodeResources(ctx context.Context, namespace, podName string) (string, error) {
	// Get pod to extract node name
	pod, err := c.clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get pod: %w", err)
	}

	if pod.Spec.NodeName == "" {
		return "Pod is not scheduled to any node yet", nil
	}

	// Get node information
	node, err := c.clientset.CoreV1().Nodes().Get(ctx, pod.Spec.NodeName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get node: %w", err)
	}

	var resDesc strings.Builder
	resDesc.WriteString(fmt.Sprintf("Node: %s\n", node.Name))

	resDesc.WriteString("Capacity:\n")
	for k, v := range node.Status.Capacity {
		resDesc.WriteString(fmt.Sprintf("  %s: %s\n", k, v.String()))
	}

	resDesc.WriteString("Allocatable:\n")
	for k, v := range node.Status.Allocatable {
		resDesc.WriteString(fmt.Sprintf("  %s: %s\n", k, v.String()))
	}

	return resDesc.String(), nil
}

// CheckNodeStatus retrieves node conditions and status for the node on which a pod is running
func (c *Client) CheckNodeStatus(ctx context.Context, namespace, podName string) (string, error) {
	// Get pod to extract node name
	pod, err := c.clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get pod: %w", err)
	}

	if pod.Spec.NodeName == "" {
		return "Pod is not scheduled to any node yet", nil
	}

	// Get node information
	node, err := c.clientset.CoreV1().Nodes().Get(ctx, pod.Spec.NodeName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get node: %w", err)
	}

	var statusDesc strings.Builder
	statusDesc.WriteString(fmt.Sprintf("Node: %s\n", node.Name))

	statusDesc.WriteString("Conditions:\n")
	for _, cond := range node.Status.Conditions {
		statusDesc.WriteString(fmt.Sprintf("  - %s: %s (%s)\n", cond.Type, cond.Status, cond.Reason))
		if cond.Message != "" {
			statusDesc.WriteString(fmt.Sprintf("    Message: %s\n", cond.Message))
		}
	}

	return statusDesc.String(), nil
}
