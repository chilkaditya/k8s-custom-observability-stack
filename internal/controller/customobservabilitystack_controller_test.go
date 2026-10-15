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

	appsv1 "chilkaditya.me/k8s-observability-op/api/v1"
	appsapi "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("CustomObservabilityStack Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default", // TODO(user):Modify as needed
		}
		customobservabilitystack := &appsv1.CustomObservabilityStack{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind CustomObservabilityStack")
			err := k8sClient.Get(ctx, typeNamespacedName, customobservabilitystack)
			if err != nil && errors.IsNotFound(err) {
				resource := &appsv1.CustomObservabilityStack{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					// TODO(user): Specify other spec details if needed.
					Spec: appsv1.CustomObservabilityStackSpec{
						TargetDeployment: "nginx-deployment",
						TargetNamespace:  "default",
						CPUThreshold:     75,
						MemoryThreshold:  80,
						EnableDashboard:  true,
						MetricsPort:      "http",
						MetricsPath:      "/metrics",
					},
				}
				deployment := &appsapi.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "nginx-deployment",
						Namespace: "default",
					},
					Spec: appsapi.DeploymentSpec{
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"app": "nginx",
							},
						},
						Template: corev1.PodTemplateSpec{
							ObjectMeta: metav1.ObjectMeta{
								Labels: map[string]string{
									"app": "nginx",
								},
							},
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{
									{
										Name:  "nginx",
										Image: "nginx:latest",
										Ports: []corev1.ContainerPort{
											{
												Name:          "http",
												ContainerPort: 80,
											},
										},
									},
								},
							},
						},
					},
				}

				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
				By("creating the target Deployment")
				Expect(k8sClient.Create(ctx, deployment)).To(Succeed())
			}
		})

		AfterEach(func() {
			// TODO(user): Cleanup logic after each test, like removing the resource instance.
			resource := &appsv1.CustomObservabilityStack{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance CustomObservabilityStack")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
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
		})
	})
})
