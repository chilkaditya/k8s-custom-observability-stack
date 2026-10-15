package controller

import (
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func buildServiceMonitor(
	deploymentName string,
	metricsPort string,
	metricsPath string,
) *monitoringv1.ServiceMonitor {

	return &monitoringv1.ServiceMonitor{
		Spec: monitoringv1.ServiceMonitorSpec{
			Selector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": deploymentName,
				},
			},

			Endpoints: []monitoringv1.Endpoint{
				{
					Port: metricsPort,
					Path: metricsPath,
				},
			},
		},
	}
}
