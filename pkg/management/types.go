package management

import (
	"context"

	"github.com/openshift/monitoring-plugin/pkg/k8s"
)

// Client is the interface for managing alert rules
type Client interface {
	// GetAlerts retrieves Prometheus alerts
	GetAlerts(ctx context.Context, req k8s.GetAlertsRequest) ([]k8s.PrometheusAlert, error)

	// GetAlertingHealth retrieves alerting health details
	GetAlertingHealth(ctx context.Context) (k8s.AlertingHealth, error)
}
