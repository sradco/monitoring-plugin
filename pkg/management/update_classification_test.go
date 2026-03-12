package management_test

import (
	"context"
	"errors"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	alertrule "github.com/openshift/monitoring-plugin/pkg/alert_rule"
	"github.com/openshift/monitoring-plugin/pkg/k8s"
	"github.com/openshift/monitoring-plugin/pkg/management"
	"github.com/openshift/monitoring-plugin/pkg/management/testutils"
	"github.com/openshift/monitoring-plugin/pkg/managementlabels"
)

var _ = Describe("UpdateAlertRuleClassification", func() {
	var (
		ctx     context.Context
		mockK8s *testutils.MockClient
		client  management.Client

		platformNamespace = "openshift-monitoring"
		userNamespace     = "my-app"
		ruleName          = "my-rules"
	)

	// platformOriginal is the canonical rule as it exists in the PrometheusRule on disk.
	platformOriginal := monitoringv1.Rule{
		Alert: "CannotRetrieveUpdates",
		Labels: map[string]string{
			"severity": "warning",
		},
	}
	platformRuleId := alertrule.GetAlertingRuleId(&platformOriginal)

	userOriginal := monitoringv1.Rule{
		Alert: "HighLatency",
		Labels: map[string]string{
			"severity": "warning",
		},
	}
	userRuleId := alertrule.GetAlertingRuleId(&userOriginal)

	makePlatformRelabeled := func() monitoringv1.Rule {
		return monitoringv1.Rule{
			Alert: platformOriginal.Alert,
			Labels: map[string]string{
				k8s.AlertRuleLabelId:             platformRuleId,
				k8s.PrometheusRuleLabelNamespace: platformNamespace,
				k8s.PrometheusRuleLabelName:      ruleName,
				"severity":                       "warning",
			},
		}
	}

	makeUserRelabeled := func() monitoringv1.Rule {
		return monitoringv1.Rule{
			Alert: userOriginal.Alert,
			Labels: map[string]string{
				k8s.AlertRuleLabelId:             userRuleId,
				k8s.PrometheusRuleLabelNamespace: userNamespace,
				k8s.PrometheusRuleLabelName:      ruleName,
				"severity":                       "warning",
			},
		}
	}

	makePR := func(ns, name string, rules ...monitoringv1.Rule) *monitoringv1.PrometheusRule {
		return &monitoringv1.PrometheusRule{
			ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: name},
			Spec: monitoringv1.PrometheusRuleSpec{
				Groups: []monitoringv1.RuleGroup{{Name: "default", Rules: rules}},
			},
		}
	}

	BeforeEach(func() {
		ctx = context.Background()
		mockK8s = &testutils.MockClient{}
		client = management.New(ctx, mockK8s)
	})

	Context("validation", func() {
		It("returns ValidationError when ruleId is empty", func() {
			err := client.UpdateAlertRuleClassification(ctx, management.UpdateRuleClassificationRequest{})
			Expect(err).To(HaveOccurred())
			var ve *management.ValidationError
			Expect(errors.As(err, &ve)).To(BeTrue())
		})

		It("returns ValidationError on invalid layer", func() {
			rule := makePlatformRelabeled()
			mockK8s.RelabeledRulesFunc = func() k8s.RelabeledRulesInterface {
				return &testutils.MockRelabeledRulesInterface{
					GetFunc: func(ctx context.Context, id string) (monitoringv1.Rule, bool) {
						return rule, true
					},
				}
			}
			bad := "invalid"
			err := client.UpdateAlertRuleClassification(ctx, management.UpdateRuleClassificationRequest{
				RuleId:   platformRuleId,
				Layer:    &bad,
				LayerSet: true,
			})
			Expect(err).To(HaveOccurred())
			var ve *management.ValidationError
			Expect(errors.As(err, &ve)).To(BeTrue())
		})

		It("returns ValidationError on invalid component", func() {
			rule := makePlatformRelabeled()
			mockK8s.RelabeledRulesFunc = func() k8s.RelabeledRulesInterface {
				return &testutils.MockRelabeledRulesInterface{
					GetFunc: func(ctx context.Context, id string) (monitoringv1.Rule, bool) {
						return rule, true
					},
				}
			}
			empty := ""
			err := client.UpdateAlertRuleClassification(ctx, management.UpdateRuleClassificationRequest{
				RuleId:       platformRuleId,
				Component:    &empty,
				ComponentSet: true,
			})
			Expect(err).To(HaveOccurred())
			var ve *management.ValidationError
			Expect(errors.As(err, &ve)).To(BeTrue())
		})

		It("returns ValidationError on invalid component_from", func() {
			rule := makePlatformRelabeled()
			mockK8s.RelabeledRulesFunc = func() k8s.RelabeledRulesInterface {
				return &testutils.MockRelabeledRulesInterface{
					GetFunc: func(ctx context.Context, id string) (monitoringv1.Rule, bool) {
						return rule, true
					},
				}
			}
			bad := "bad-label"
			err := client.UpdateAlertRuleClassification(ctx, management.UpdateRuleClassificationRequest{
				RuleId:           platformRuleId,
				ComponentFrom:    &bad,
				ComponentFromSet: true,
			})
			Expect(err).To(HaveOccurred())
			var ve *management.ValidationError
			Expect(errors.As(err, &ve)).To(BeTrue())
		})

		It("returns ValidationError on invalid layer_from", func() {
			rule := makePlatformRelabeled()
			mockK8s.RelabeledRulesFunc = func() k8s.RelabeledRulesInterface {
				return &testutils.MockRelabeledRulesInterface{
					GetFunc: func(ctx context.Context, id string) (monitoringv1.Rule, bool) {
						return rule, true
					},
				}
			}
			bad := "1layer"
			err := client.UpdateAlertRuleClassification(ctx, management.UpdateRuleClassificationRequest{
				RuleId:       platformRuleId,
				LayerFrom:    &bad,
				LayerFromSet: true,
			})
			Expect(err).To(HaveOccurred())
			var ve *management.ValidationError
			Expect(errors.As(err, &ve)).To(BeTrue())
		})
	})

	It("returns NotFoundError when the base rule cannot be found", func() {
		mockK8s.RelabeledRulesFunc = func() k8s.RelabeledRulesInterface {
			return &testutils.MockRelabeledRulesInterface{
				GetFunc: func(ctx context.Context, id string) (monitoringv1.Rule, bool) {
					return monitoringv1.Rule{}, false
				},
			}
		}
		val := "cluster"
		err := client.UpdateAlertRuleClassification(ctx, management.UpdateRuleClassificationRequest{
			RuleId:   "missing",
			Layer:    &val,
			LayerSet: true,
		})
		Expect(err).To(HaveOccurred())
		var nf *management.NotFoundError
		Expect(errors.As(err, &nf)).To(BeTrue())
		Expect(nf.Resource).To(Equal("AlertRule"))
	})

	It("treats empty payload as a no-op", func() {
		rule := makePlatformRelabeled()
		mockK8s.RelabeledRulesFunc = func() k8s.RelabeledRulesInterface {
			return &testutils.MockRelabeledRulesInterface{
				GetFunc: func(ctx context.Context, id string) (monitoringv1.Rule, bool) {
					return rule, true
				},
			}
		}
		err := client.UpdateAlertRuleClassification(ctx, management.UpdateRuleClassificationRequest{RuleId: platformRuleId})
		Expect(err).NotTo(HaveOccurred())
	})

	Context("platform rules (ARC path)", func() {
		BeforeEach(func() {
			mockK8s.NamespaceFunc = func() k8s.NamespaceInterface {
				return &testutils.MockNamespaceInterface{
					MonitoringNamespaces: map[string]bool{platformNamespace: true},
				}
			}
		})

		It("creates an ARC with classification relabel configs", func() {
			relabeled := makePlatformRelabeled()
			pr := makePR(platformNamespace, ruleName, platformOriginal)

			mockK8s.RelabeledRulesFunc = func() k8s.RelabeledRulesInterface {
				return &testutils.MockRelabeledRulesInterface{
					GetFunc: func(ctx context.Context, id string) (monitoringv1.Rule, bool) {
						if id == platformRuleId {
							return relabeled, true
						}
						return monitoringv1.Rule{}, false
					},
				}
			}
			prStore := &testutils.MockPrometheusRuleInterface{
				PrometheusRules: map[string]*monitoringv1.PrometheusRule{
					platformNamespace + "/" + ruleName: pr,
				},
			}
			mockK8s.PrometheusRulesFunc = func() k8s.PrometheusRuleInterface { return prStore }

			arcStore := &testutils.MockAlertRelabelConfigInterface{}
			mockK8s.AlertRelabelConfigsFunc = func() k8s.AlertRelabelConfigInterface { return arcStore }

			component := "networking"
			layer := "cluster"
			err := client.UpdateAlertRuleClassification(ctx, management.UpdateRuleClassificationRequest{
				RuleId:       platformRuleId,
				Component:    &component,
				ComponentSet: true,
				Layer:        &layer,
				LayerSet:     true,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(arcStore.AlertRelabelConfigs).To(HaveLen(1))

			for _, arc := range arcStore.AlertRelabelConfigs {
				Expect(arc.Labels[managementlabels.ARCLabelPrometheusRuleNameKey]).To(Equal(ruleName))
				Expect(arc.Labels[managementlabels.ARCLabelAlertNameKey]).To(Equal("CannotRetrieveUpdates"))
				Expect(arc.Annotations[managementlabels.ARCAnnotationAlertRuleIDKey]).To(Equal(platformRuleId))

				hasComponent := false
				hasLayer := false
				hasManagedBy := false
				for _, rc := range arc.Spec.Configs {
					if rc.Action == "Replace" && rc.TargetLabel == k8s.AlertRuleClassificationComponentKey {
						Expect(rc.Replacement).To(Equal("networking"))
						hasComponent = true
					}
					if rc.Action == "Replace" && rc.TargetLabel == k8s.AlertRuleClassificationLayerKey {
						Expect(rc.Replacement).To(Equal("cluster"))
						hasLayer = true
					}
					if rc.Action == "Replace" && rc.TargetLabel == managementlabels.ClassificationManagedByKey {
						hasManagedBy = true
					}
				}
				Expect(hasComponent).To(BeTrue(), "ARC should have component replace config")
				Expect(hasLayer).To(BeTrue(), "ARC should have layer replace config")
				Expect(hasManagedBy).To(BeTrue(), "ARC should have managed-by replace config")
			}
		})

		It("creates ARC with component_from and layer_from relabel configs", func() {
			relabeled := makePlatformRelabeled()
			pr := makePR(platformNamespace, ruleName, platformOriginal)

			mockK8s.RelabeledRulesFunc = func() k8s.RelabeledRulesInterface {
				return &testutils.MockRelabeledRulesInterface{
					GetFunc: func(ctx context.Context, id string) (monitoringv1.Rule, bool) {
						if id == platformRuleId {
							return relabeled, true
						}
						return monitoringv1.Rule{}, false
					},
				}
			}
			prStore := &testutils.MockPrometheusRuleInterface{
				PrometheusRules: map[string]*monitoringv1.PrometheusRule{
					platformNamespace + "/" + ruleName: pr,
				},
			}
			mockK8s.PrometheusRulesFunc = func() k8s.PrometheusRuleInterface { return prStore }

			arcStore := &testutils.MockAlertRelabelConfigInterface{}
			mockK8s.AlertRelabelConfigsFunc = func() k8s.AlertRelabelConfigInterface { return arcStore }

			componentFrom := "namespace"
			layerFrom := "tier"
			err := client.UpdateAlertRuleClassification(ctx, management.UpdateRuleClassificationRequest{
				RuleId:           platformRuleId,
				ComponentFrom:    &componentFrom,
				ComponentFromSet: true,
				LayerFrom:        &layerFrom,
				LayerFromSet:     true,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(arcStore.AlertRelabelConfigs).To(HaveLen(1))

			for _, arc := range arcStore.AlertRelabelConfigs {
				hasComponentFrom := false
				hasLayerFrom := false
				for _, rc := range arc.Spec.Configs {
					if rc.Action == "Replace" && rc.TargetLabel == k8s.AlertRuleClassificationComponentFromKey {
						Expect(rc.Replacement).To(Equal("namespace"))
						hasComponentFrom = true
					}
					if rc.Action == "Replace" && rc.TargetLabel == k8s.AlertRuleClassificationLayerFromKey {
						Expect(rc.Replacement).To(Equal("tier"))
						hasLayerFrom = true
					}
				}
				Expect(hasComponentFrom).To(BeTrue())
				Expect(hasLayerFrom).To(BeTrue())
			}
		})
	})

	Context("user-defined rules", func() {
		BeforeEach(func() {
			mockK8s.NamespaceFunc = func() k8s.NamespaceInterface {
				return &testutils.MockNamespaceInterface{
					MonitoringNamespaces: map[string]bool{platformNamespace: true},
				}
			}
		})

		It("returns NotAllowedError when ENABLE_USER_WORKLOAD_ARCS is disabled", func() {
			relabeled := makeUserRelabeled()
			mockK8s.RelabeledRulesFunc = func() k8s.RelabeledRulesInterface {
				return &testutils.MockRelabeledRulesInterface{
					GetFunc: func(ctx context.Context, id string) (monitoringv1.Rule, bool) {
						if id == userRuleId {
							return relabeled, true
						}
						return monitoringv1.Rule{}, false
					},
				}
			}

			component := "team_a"
			err := client.UpdateAlertRuleClassification(ctx, management.UpdateRuleClassificationRequest{
				RuleId:       userRuleId,
				Component:    &component,
				ComponentSet: true,
			})
			Expect(err).To(HaveOccurred())
			var na *management.NotAllowedError
			Expect(errors.As(err, &na)).To(BeTrue())
		})

		It("creates ARC in openshift-user-workload-monitoring when flag is enabled", func() {
			os.Setenv("ENABLE_USER_WORKLOAD_ARCS", "true")
			DeferCleanup(func() { os.Unsetenv("ENABLE_USER_WORKLOAD_ARCS") })
			client = management.New(ctx, mockK8s)

			relabeled := makeUserRelabeled()
			pr := makePR(userNamespace, ruleName, userOriginal)

			mockK8s.RelabeledRulesFunc = func() k8s.RelabeledRulesInterface {
				return &testutils.MockRelabeledRulesInterface{
					GetFunc: func(ctx context.Context, id string) (monitoringv1.Rule, bool) {
						if id == userRuleId {
							return relabeled, true
						}
						return monitoringv1.Rule{}, false
					},
				}
			}
			prStore := &testutils.MockPrometheusRuleInterface{
				PrometheusRules: map[string]*monitoringv1.PrometheusRule{
					userNamespace + "/" + ruleName: pr,
				},
			}
			mockK8s.PrometheusRulesFunc = func() k8s.PrometheusRuleInterface { return prStore }

			arcStore := &testutils.MockAlertRelabelConfigInterface{}
			mockK8s.AlertRelabelConfigsFunc = func() k8s.AlertRelabelConfigInterface { return arcStore }

			component := "team_a"
			layer := "namespace"
			err := client.UpdateAlertRuleClassification(ctx, management.UpdateRuleClassificationRequest{
				RuleId:       userRuleId,
				Component:    &component,
				ComponentSet: true,
				Layer:        &layer,
				LayerSet:     true,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(arcStore.AlertRelabelConfigs).To(HaveLen(1))

			for _, arc := range arcStore.AlertRelabelConfigs {
				Expect(arc.Namespace).To(Equal(k8s.UserWorkloadMonitoringNamespace))
				Expect(arc.Annotations[managementlabels.ARCAnnotationAlertRuleIDKey]).To(Equal(userRuleId))

				hasComponent := false
				hasLayer := false
				for _, rc := range arc.Spec.Configs {
					if rc.Action == "Replace" && rc.TargetLabel == k8s.AlertRuleClassificationComponentKey {
						Expect(rc.Replacement).To(Equal("team_a"))
						hasComponent = true
					}
					if rc.Action == "Replace" && rc.TargetLabel == k8s.AlertRuleClassificationLayerKey {
						Expect(rc.Replacement).To(Equal("namespace"))
						hasLayer = true
					}
				}
				Expect(hasComponent).To(BeTrue())
				Expect(hasLayer).To(BeTrue())
			}
		})
	})
})

var _ = Describe("ApplyDynamicClassification", func() {
	It("returns defaults when no _from labels set", func() {
		c, l := management.ApplyDynamicClassification(nil, nil, "comp", "cluster")
		Expect(c).To(Equal("comp"))
		Expect(l).To(Equal("cluster"))
	})

	It("uses component_from when set", func() {
		ruleLabels := map[string]string{
			k8s.AlertRuleClassificationComponentFromKey: "name",
		}
		alertLabels := map[string]string{
			"name": "dns",
		}
		c, _ := management.ApplyDynamicClassification(ruleLabels, alertLabels, "default", "cluster")
		Expect(c).To(Equal("dns"))
	})

	It("uses layer_from when set", func() {
		ruleLabels := map[string]string{
			k8s.AlertRuleClassificationLayerFromKey: "tier",
		}
		alertLabels := map[string]string{
			"tier": "Cluster",
		}
		_, l := management.ApplyDynamicClassification(ruleLabels, alertLabels, "comp", "namespace")
		Expect(l).To(Equal("cluster"))
	})

	It("falls back to defaults when _from label points to empty alert label", func() {
		ruleLabels := map[string]string{
			k8s.AlertRuleClassificationComponentFromKey: "missing_label",
		}
		c, _ := management.ApplyDynamicClassification(ruleLabels, map[string]string{}, "fallback", "cluster")
		Expect(c).To(Equal("fallback"))
	})
})
