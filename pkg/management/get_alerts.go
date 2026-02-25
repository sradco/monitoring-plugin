package management

import (
	"context"
	"fmt"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/model/relabel"
	"k8s.io/apimachinery/pkg/types"

	alertrule "github.com/openshift/monitoring-plugin/pkg/alert_rule"
	"github.com/openshift/monitoring-plugin/pkg/k8s"
	"github.com/openshift/monitoring-plugin/pkg/managementlabels"
)

func (c *client) GetAlerts(ctx context.Context, req k8s.GetAlertsRequest) ([]k8s.PrometheusAlert, error) {
	alerts, err := c.k8sClient.PrometheusAlerts().GetAlerts(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get prometheus alerts: %w", err)
	}

	configs := c.k8sClient.RelabeledRules().Config()
	rules := c.k8sClient.RelabeledRules().List(ctx)

	result := make([]k8s.PrometheusAlert, 0, len(alerts))
	for _, alert := range alerts {
		// Only apply relabel configs for platform alerts. User workload alerts
		// already come from their own stack and should not be relabeled here.
		if alert.Labels[k8s.AlertSourceLabel] != k8s.AlertSourceUser {
			relabels, keep := relabel.Process(labels.FromMap(alert.Labels), configs...)
			if !keep {
				continue
			}
			alert.Labels = relabels.Map()
		}

		// Add calculated rule ID and source when not present (labels enrichment)
		c.setRuleIDAndSourceIfMissing(ctx, &alert, rules)

		// correlate alert -> base alert rule via subset matching against relabeled rules
		alertRuleId := alert.Labels[k8s.AlertRuleLabelId]

		bestRule, corrId := correlateAlertToRule(alert.Labels, rules)
		if corrId != "" {
			alertRuleId = corrId
		}
		if bestRule == nil && alertRuleId != "" {
			if rule, ok := c.k8sClient.RelabeledRules().Get(ctx, alertRuleId); ok {
				bestRule = &rule
			}
		}

		if bestRule != nil {
			if src := c.deriveAlertSource(bestRule.Labels); src != "" {
				alert.Labels[k8s.AlertSourceLabel] = src
			}
		}

		// keep label and optional enriched fields consistent
		if alert.Labels[k8s.AlertRuleLabelId] == "" && alertRuleId != "" {
			alert.Labels[k8s.AlertRuleLabelId] = alertRuleId
		}
		alert.AlertRuleId = alertRuleId

		if bestRule != nil && bestRule.Labels != nil {
			alert.PrometheusRuleNamespace = bestRule.Labels[k8s.PrometheusRuleLabelNamespace]
			alert.PrometheusRuleName = bestRule.Labels[k8s.PrometheusRuleLabelName]
			alert.AlertingRuleName = bestRule.Labels[managementlabels.AlertingRuleLabelName]
		}

		result = append(result, alert)
	}

	return result, nil
}

func (c *client) setRuleIDAndSourceIfMissing(ctx context.Context, alert *k8s.PrometheusAlert, rules []monitoringv1.Rule) {
	if alert.Labels[k8s.AlertRuleLabelId] == "" {
		for _, existing := range rules {
			if existing.Alert != alert.Labels[managementlabels.AlertNameLabel] {
				continue
			}
			if !ruleMatchesAlert(existing.Labels, alert.Labels) {
				continue
			}
			rid := alertrule.GetAlertingRuleId(&existing)
			alert.Labels[k8s.AlertRuleLabelId] = rid
			if alert.Labels[k8s.AlertSourceLabel] == "" {
				if src := c.deriveAlertSource(existing.Labels); src != "" {
					alert.Labels[k8s.AlertSourceLabel] = src
				}
			}
			break
		}
	}
	if alert.Labels[k8s.AlertSourceLabel] != "" {
		return
	}
	if rid := alert.Labels[k8s.AlertRuleLabelId]; rid != "" {
		if existing, ok := c.k8sClient.RelabeledRules().Get(ctx, rid); ok {
			if src := c.deriveAlertSource(existing.Labels); src != "" {
				alert.Labels[k8s.AlertSourceLabel] = src
			}
		}
	}
}

func ruleMatchesAlert(existingRuleLabels, alertLabels map[string]string) bool {
	existingBusiness := filterBusinessLabels(existingRuleLabels)
	for k, v := range existingBusiness {
		lv, ok := alertLabels[k]
		if !ok || lv != v {
			return false
		}
	}
	return true
}

// correlateAlertToRule tries to find the base alert rule for the given alert labels
// by subset-matching against relabeled rules.
func correlateAlertToRule(alertLabels map[string]string, rules []monitoringv1.Rule) (*monitoringv1.Rule, string) {
	// Determine best match: prefer rules with more labels (more specific)
	var (
		bestId         string
		bestRule       *monitoringv1.Rule
		bestLabelCount int
	)
	for i := range rules {
		rule := &rules[i]
		ruleLabels := sanitizeRuleLabels(rule.Labels)
		if isSubset(ruleLabels, alertLabels) {
			if len(ruleLabels) > bestLabelCount {
				bestLabelCount = len(ruleLabels)
				bestRule = rule
				bestId = rule.Labels[k8s.AlertRuleLabelId]
			}
		}
	}
	if bestRule == nil {
		return nil, ""
	}
	return bestRule, bestId
}

// sanitizeRuleLabels removes meta labels that will not be present on alerts
func sanitizeRuleLabels(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		if k == k8s.PrometheusRuleLabelNamespace || k == k8s.PrometheusRuleLabelName || k == k8s.AlertRuleLabelId {
			continue
		}
		out[k] = v
	}
	return out
}

// isSubset returns true if all key/value pairs in sub are present in sup
func isSubset(sub map[string]string, sup map[string]string) bool {
	for k, v := range sub {
		if sv, ok := sup[k]; !ok || sv != v {
			return false
		}
	}
	return true
}

func (c *client) deriveAlertSource(ruleLabels map[string]string) string {
	ns := ruleLabels[k8s.PrometheusRuleLabelNamespace]
	name := ruleLabels[k8s.PrometheusRuleLabelName]
	if ns == "" || name == "" {
		return ""
	}
	if c.IsPlatformAlertRule(types.NamespacedName{Namespace: ns, Name: name}) {
		return k8s.AlertSourcePlatform
	}
	return k8s.AlertSourceUser
}
