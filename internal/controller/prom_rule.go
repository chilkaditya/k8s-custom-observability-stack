package controller

import (
	"fmt"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func buildPrometheusRule(
	deploymentName string,
	cpuThreshold int32,
	memoryThreshold int32,
) *monitoringv1.PrometheusRule {

	cpuAlertName := fmt.Sprintf("%sHighCPU", deploymentName)
	memoryAlertName := fmt.Sprintf("%sHighMemory", deploymentName)

	duration := monitoringv1.Duration("5m")

	return &monitoringv1.PrometheusRule{
		Spec: monitoringv1.PrometheusRuleSpec{
			Groups: []monitoringv1.RuleGroup{
				{
					Name: deploymentName + ".rules",

					Rules: []monitoringv1.Rule{
						{
							Alert: cpuAlertName,

							Expr: intstr.FromString(
								fmt.Sprintf(
									`100 * sum(rate(container_cpu_usage_seconds_total{pod=~"%s-.*"}[5m])) > %d`,
									deploymentName,
									cpuThreshold,
								),
							),

							For: &duration,

							Labels: map[string]string{
								"severity": "warning",
							},

							Annotations: map[string]string{
								"summary": fmt.Sprintf(
									"High CPU usage for %s",
									deploymentName,
								),

								"description": fmt.Sprintf(
									"CPU usage for %s is above %d%%",
									deploymentName,
									cpuThreshold,
								),
							},
						},

						{
							Alert: memoryAlertName,

							Expr: intstr.FromString(
								fmt.Sprintf(
									`100 * sum(container_memory_working_set_bytes{pod=~"%s-.*"}) / sum(kube_pod_container_resource_limits{pod=~"%s-.*", resource="memory"}) > %d`,
									deploymentName,
									deploymentName,
									memoryThreshold,
								),
							),

							For: &duration,

							Labels: map[string]string{
								"severity": "warning",
							},

							Annotations: map[string]string{
								"summary": fmt.Sprintf(
									"High memory usage for %s",
									deploymentName,
								),

								"description": fmt.Sprintf(
									"Memory usage for %s is above %d%%",
									deploymentName,
									memoryThreshold,
								),
							},
						},
					},
				},
			},
		},
	}
}
