/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	apimeta "k8s.io/apimachinery/pkg/api/meta"

	appsv1 "chilkaditya.me/k8s-observability-op/api/v1"
	appsapi "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("CustomObservabilityStack Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		const (
			testNamespace      = "default"
			testDeploymentName = "nginx-deployment"
			testAppLabel       = "nginx"
		)

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: testNamespace, // TODO(user):Modify as needed
		}
		// customobservabilitystack := &appsv1.CustomObservabilityStack{}

		BeforeEach(func() {
			By("creating the target Deployment if it does not exist")

			deploymentKey := types.NamespacedName{
				Name:      testDeploymentName,
				Namespace: testNamespace,
			}

			deployment := &appsapi.Deployment{}
			err := k8sClient.Get(ctx, deploymentKey, deployment)

			if errors.IsNotFound(err) {
				deployment = &appsapi.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      testDeploymentName,
						Namespace: testNamespace,
					},
					Spec: appsapi.DeploymentSpec{
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{"app": testAppLabel},
						},
						Template: corev1.PodTemplateSpec{
							ObjectMeta: metav1.ObjectMeta{
								Labels: map[string]string{"app": testAppLabel},
							},
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{{
									Name:  testAppLabel,
									Image: "nginx:latest",
									Ports: []corev1.ContainerPort{{
										Name:          "http",
										ContainerPort: 80,
									}},
								}},
							},
						},
					},
				}
				Expect(k8sClient.Create(ctx, deployment)).To(Succeed())
			} else {
				Expect(err).NotTo(HaveOccurred())
			}

			By("creating the CustomObservabilityStack if it does not exist")

			resource := &appsv1.CustomObservabilityStack{}
			err = k8sClient.Get(ctx, typeNamespacedName, resource)

			if errors.IsNotFound(err) {
				resource = &appsv1.CustomObservabilityStack{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: testNamespace,
					},
					Spec: appsv1.CustomObservabilityStackSpec{
						TargetDeployment: testDeploymentName,
						TargetNamespace:  testNamespace,
						CPUThreshold:     75,
						MemoryThreshold:  80,
						EnableDashboard:  true,
						MetricsPort:      "http",
						MetricsPath:      "/metrics",
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			} else {
				Expect(err).NotTo(HaveOccurred())
			}
		})

		AfterEach(func() {
			By("Cleaning up the generated ConfigMap")

			configMap := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "customobservabilitystack-sample-dashboard",
					Namespace: testNamespace,
				},
			}
			err := k8sClient.Delete(ctx, configMap)
			Expect(errors.IsNotFound(err) || err == nil).To(BeTrue())

			By("Cleaning up the PrometheusRule")

			promRule := &monitoringv1.PrometheusRule{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "customobservabilitystack-sample-rules",
					Namespace: testNamespace,
				},
			}
			err = k8sClient.Delete(ctx, promRule)
			Expect(errors.IsNotFound(err) || err == nil).To(BeTrue())

			By("Cleaning up the ServiceMonitor")

			serviceMonitor := &monitoringv1.ServiceMonitor{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "customobservabilitystack-sample-monitor",
					Namespace: testNamespace,
				},
			}
			err = k8sClient.Delete(ctx, serviceMonitor)
			Expect(errors.IsNotFound(err) || err == nil).To(BeTrue())

			By("Cleaning up the target Deployment")

			deployment := &appsapi.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testDeploymentName,
					Namespace: testNamespace,
				},
			}
			err = k8sClient.Delete(ctx, deployment)
			Expect(errors.IsNotFound(err) || err == nil).To(BeTrue())

			By("Cleaning up the CustomObservabilityStack")

			resource := &appsv1.CustomObservabilityStack{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: testNamespace,
				},
			}
			err = k8sClient.Delete(ctx, resource)
			Expect(errors.IsNotFound(err) || err == nil).To(BeTrue())
		})

		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &CustomObservabilityStackReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			// TODO(user): Add more specific assertions depending on your controller's reconciliation logic.
			// Example: If you expect a certain status condition after reconciliation, verify it here.

			By("Checking that the Grafana dashboard ConfigMap exists")

			configMap := &corev1.ConfigMap{}
			configMapKey := types.NamespacedName{
				Name:      "test-resource-dashboard",
				Namespace: testNamespace,
			}

			Expect(k8sClient.Get(ctx, configMapKey, configMap)).
				To(Succeed())
			Expect(configMap.Data).To(HaveKey("dashboard.json"))
			Expect(configMap.Labels).To(HaveKeyWithValue("grafana_dashboard", "1"))

			By("Checking that the PrometheusRule exists")

			promRule := &monitoringv1.PrometheusRule{}
			promRuleKey := types.NamespacedName{
				Name:      "test-resource-rules",
				Namespace: testNamespace,
			}

			Expect(k8sClient.Get(ctx, promRuleKey, promRule)).
				To(Succeed())

			By("Checking that the ServiceMonitor exists")

			serviceMonitor := &monitoringv1.ServiceMonitor{}
			serviceMonitorKey := types.NamespacedName{
				Name:      "test-resource-monitor",
				Namespace: testNamespace,
			}

			Expect(k8sClient.Get(ctx, serviceMonitorKey, serviceMonitor)).
				To(Succeed())

			By("Checking that the CustomObservabilityStack is Ready")

			reconciledStack := &appsv1.CustomObservabilityStack{}
			Expect(k8sClient.Get(ctx, typeNamespacedName, reconciledStack)).
				To(Succeed())

			readyCondition := apimeta.FindStatusCondition(
				reconciledStack.Status.Conditions,
				"Ready",
			)

			Expect(readyCondition).NotTo(BeNil())
			Expect(readyCondition.Status).To(Equal(metav1.ConditionTrue))
			Expect(readyCondition.Reason).To(Equal("ReconciliationsSucceeded"))

		})
	})
})
