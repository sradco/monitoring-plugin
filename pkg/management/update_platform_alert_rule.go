package management

import (
	"context"
	"fmt"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
)

func (c *client) getOriginalPlatformRule(ctx context.Context, namespace string, name string, alertRuleId string) (*monitoringv1.Rule, error) {
	pr, found, err := c.k8sClient.PrometheusRules().Get(ctx, namespace, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get PrometheusRule %s/%s: %w", namespace, name, err)
	}

	if !found {
		return nil, &NotFoundError{
			Resource:       "PrometheusRule",
			Id:             alertRuleId,
			AdditionalInfo: fmt.Sprintf("PrometheusRule %s/%s not found", namespace, name),
		}
	}

	for groupIdx := range pr.Spec.Groups {
		for ruleIdx := range pr.Spec.Groups[groupIdx].Rules {
			rule := &pr.Spec.Groups[groupIdx].Rules[ruleIdx]
			if ruleMatchesAlertRuleID(*rule, alertRuleId) {
				return rule, nil
			}
		}
	}

	return nil, &NotFoundError{
		Resource:       "AlertRule",
		Id:             alertRuleId,
		AdditionalInfo: fmt.Sprintf("in PrometheusRule %s/%s", namespace, name),
	}
}
