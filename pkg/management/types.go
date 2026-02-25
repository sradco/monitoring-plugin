package management

import (
	"context"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"

	"github.com/openshift/monitoring-plugin/pkg/k8s"
)

// Client is the interface for managing alert rules
type Client interface {
	// ListRules lists all alert rules in the specified PrometheusRule resource
	ListRules(ctx context.Context, prOptions PrometheusRuleOptions, arOptions AlertRuleOptions) ([]monitoringv1.Rule, error)

	// CreateUserDefinedAlertRule creates a new user-defined alert rule
	CreateUserDefinedAlertRule(ctx context.Context, alertRule monitoringv1.Rule, prOptions PrometheusRuleOptions) (alertRuleId string, err error)

	// DeleteUserDefinedAlertRuleById deletes a user-defined alert rule by its ID
	DeleteUserDefinedAlertRuleById(ctx context.Context, alertRuleId string) error

	// CreatePlatformAlertRule creates a new platform alert rule
	CreatePlatformAlertRule(ctx context.Context, alertRule monitoringv1.Rule) (alertRuleId string, err error)

	// GetAlerts retrieves Prometheus alerts
	GetAlerts(ctx context.Context, req k8s.GetAlertsRequest) ([]k8s.PrometheusAlert, error)
	// GetRules retrieves Prometheus alerting rules and active alerts
	GetRules(ctx context.Context, req k8s.GetRulesRequest) ([]k8s.PrometheusRuleGroup, error)

	// GetAlertingHealth retrieves alerting health details
	GetAlertingHealth(ctx context.Context) (k8s.AlertingHealth, error)
}

// PrometheusRuleOptions specifies options for selecting PrometheusRule resources and groups
type PrometheusRuleOptions struct {
	// Name of the PrometheusRule resource where the alert rule will be added/listed from
	Name string `json:"prometheusRuleName"`

	// Namespace of the PrometheusRule resource where the alert rule will be added/listed from
	Namespace string `json:"prometheusRuleNamespace"`

	// GroupName of the RuleGroup within the PrometheusRule resource
	GroupName string `json:"groupName"`
}

type AlertRuleOptions struct {
	// Name filters alert rules by alert name
	Name string `json:"name,omitempty"`

	// Source filters alert rules by source type (platform or user-defined)
	Source string `json:"source,omitempty"`

	// Labels filters alert rules by arbitrary label key-value pairs
	Labels map[string]string `json:"labels,omitempty"`
}
