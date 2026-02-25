# Classification Overrides: Proposed Change Summary

## What is changing

Replace ConfigMap-based classification storage with labels stored directly on the rule resource, using AlertRelabelConfig (ARC) CRs only for rules that cannot be directly modified.

## Before and After

| Rule type | Management | Before (ConfigMap) | After (Labels + ARC) |
|---|---|---|---|
| User-defined PrometheusRule | Unmanaged | ConfigMap in plugin namespace | Labels directly on the PrometheusRule |
| User-defined PrometheusRule | Operator-managed | ConfigMap in plugin namespace | ARC in `openshift-user-workload-monitoring` |
| User-defined PrometheusRule | GitOps-managed | ConfigMap in plugin namespace | Block (user adds labels in Git) |
| Platform via AlertingRule | Unmanaged | ConfigMap in plugin namespace | Labels directly on the AlertingRule |
| Platform via AlertingRule | Operator-managed | ConfigMap in plugin namespace | ARC in `openshift-monitoring` |
| Platform via AlertingRule | GitOps-managed | ConfigMap in plugin namespace | Block (user adds labels in Git) |
| Platform without AlertingRule | Operator-managed | ConfigMap in plugin namespace | ARC in `openshift-monitoring` |

### Storage comparison

| Aspect | Before | After |
|---|---|---|
| Where classification is stored | ConfigMaps in plugin namespace | Labels on the rule itself (or ARC for operator-managed) |
| New k8s interfaces | `ConfigMapInterface` (3 methods) | None |
| New files | 5 files | 0 (reuses existing label update paths) |
| Classification visibility | API response only | Everywhere (kubectl, Prometheus, Alertmanager) |
| ARC model | N/A | Per-alert-rule (same as existing label changes) |
| Provenance tracking | Implicit (separate data store) | `openshift_io_alert_rule_classification_managed_by` label |

## Pros

1. **Simpler implementation** -- removes 5 files and the ConfigMap subsystem; classification reuses the existing label update code paths
2. **Classification visible beyond the API** -- labels on the rule are visible to kubectl, Prometheus, Alertmanager, and external tools (Grafana, PagerDuty)
3. **No new k8s interfaces** -- no `ConfigMapInterface`, no ConfigMap mock in tests
4. **Consistent with label changes** -- classification follows the exact same branching as other label updates (direct for editable rules, ARC for operator-managed, block for GitOps)
5. **Clear provenance** -- `managed_by` label explicitly tracks whether the classification was set via the API
6. **Zero migration** -- new feature, no existing data to migrate; existing clusters with custom ARCs or alerts are unaffected

## Cons

1. **`_from` (dynamic derivation) limited** -- `componentFrom`/`layerFrom` is only supported for ARC-based rules (operator-managed platform). Directly editable rules support only static component/layer. This covers the main use case (CVO-style platform alerts).
2. **RBAC addition for user-workload ARCs** -- operator-managed user-defined PrometheusRules require a Role + RoleBinding for ARCs in `openshift-user-workload-monitoring` (one namespace, not cluster-wide)
3. **GitOps-managed rules blocked** -- users must add classification labels in Git. ARC side-channel for GitOps rules can be added later if requested.
4. **No provenance distinction for external label setters** -- if someone sets `openshift_io_alert_rule_component` via kubectl or their own ARC (without `managed_by`), the API treats it as "rule-defined." Functionally correct but the UI won't show it as a "user override."

## RBAC changes needed

A Role + RoleBinding in `openshift-user-workload-monitoring` for the monitoring-plugin ServiceAccount:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: monitoring-plugin-arc-manager
  namespace: openshift-user-workload-monitoring
rules:
  - apiGroups: ["monitoring.openshift.io"]
    resources: ["alertrelabelconfigs"]
    verbs: ["get", "list", "watch", "create", "update", "delete"]
```

This covers operator-managed user-defined PrometheusRules. All other rule types require no new RBAC.

## Impact on existing clusters

None. Classification is only activated when a user explicitly classifies a rule via the API. Existing ARCs, PrometheusRules, and alerts are unaffected.
