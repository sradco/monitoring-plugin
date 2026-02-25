package management

import (
	"context"

	"github.com/openshift/monitoring-plugin/pkg/k8s"
)

// Client is the interface for managing alert rules
type Client interface {
	// GetAlertingHealth retrieves alerting health details
	GetAlertingHealth(ctx context.Context) (k8s.AlertingHealth, error)
}
