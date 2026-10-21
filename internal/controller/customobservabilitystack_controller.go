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
	"fmt"
	"reflect"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"

	meta "k8s.io/apimachinery/pkg/api/meta"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	appsv1 "chilkaditya.me/k8s-observability-op/api/v1"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	kappsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	controllerutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// CustomObservabilityStackReconciler reconciles a CustomObservabilityStack object
type CustomObservabilityStackReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=apps.chilkaditya.me,resources=customobservabilitystacks,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps.chilkaditya.me,resources=customobservabilitystacks/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps.chilkaditya.me,resources=customobservabilitystacks/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the CustomObservabilityStack object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.20.4/pkg/reconcile
func (r *CustomObservabilityStackReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := logf.FromContext(ctx)

	// --------------------------------------------------
	// 1. Fetch our CustomObservabilityStack CR
	// --------------------------------------------------

	var myobsStack appsv1.CustomObservabilityStack

	if err := r.Get(ctx, req.NamespacedName, &myobsStack); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	logger.Info(
		"Reconcile custom observability stack",
		"name", myobsStack.Name,
		"namespace", myobsStack.Namespace,
	)

	// --------------------------------------------------
	// 2. Fetch the target Deployment
	// --------------------------------------------------

	deployment := &kappsv1.Deployment{}

	objKey := types.NamespacedName{
		Name:      myobsStack.Spec.TargetDeployment,
		Namespace: myobsStack.Spec.TargetNamespace,
	}
	err := r.Get(ctx, objKey, deployment)
	if err != nil {
		if apierrors.IsNotFound(err) {
			message := fmt.Sprintf(
				"Deployment %s/%s was not found",
				myobsStack.Spec.TargetNamespace,
				myobsStack.Spec.TargetDeployment,
			)
			if statusErr := r.setReadyCondition(
				ctx,
				&myobsStack,
				metav1.ConditionFalse,
				"TargetDeploymentNotFound",
				message,
			); statusErr != nil {
				return ctrl.Result{}, statusErr
			}
			return ctrl.Result{
				RequeueAfter: 30 * time.Second,
			}, nil
		}

		if statusErr := r.setReadyCondition(
			ctx,
			&myobsStack,
			metav1.ConditionFalse,
			"DeploymentLookupFailed",
			err.Error(),
		); statusErr != nil {
			return ctrl.Result{}, statusErr
		}
		return ctrl.Result{}, err
	}

	logger.Info(
		"Deployment found",
		"Deployment name", deployment.Name,
	)

	// --------------------------------------------------
	// 3. Define the desired ConfigMap
	// --------------------------------------------------

	ConfigMapName := myobsStack.Name + "-dashboard"

	dashboardJSON, err := buildGrafanaDashboard(
		myobsStack.Spec.TargetDeployment,
	)
	if err != nil {
		return ctrl.Result{}, err
	}

	desiredConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ConfigMapName,
			Namespace: myobsStack.Spec.TargetNamespace,
			Labels: map[string]string{
				"grafana_dashboard": "1",
			},
		},
		Data: map[string]string{
			"dashboard.json": dashboardJSON,
		},
	}

	if err := controllerutil.SetControllerReference(
		&myobsStack,
		desiredConfigMap,
		r.Scheme,
	); err != nil {
		return ctrl.Result{}, err
	}

	existingConfigMap := &corev1.ConfigMap{}
	configMapKey := types.NamespacedName{
		Name:      ConfigMapName,
		Namespace: myobsStack.Spec.TargetNamespace,
	}

	err = r.Get(ctx, configMapKey, existingConfigMap)

	if apierrors.IsNotFound(err) {
		logger.Info(
			"Config map not found",
			"configmap", ConfigMapName,
		)
		if err := r.Create(ctx, desiredConfigMap); err != nil {
			if apierrors.IsAlreadyExists(err) {
				logger.Info(
					"Config map already exists; will reconcile again",
					"configmap", ConfigMapName,
				)
				return ctrl.Result{RequeueAfter: time.Second}, nil
			}
			return ctrl.Result{}, err
		}

		logger.Info(
			"ConfigMap created",
			"ConfigMap", ConfigMapName,
		)

	} else if err != nil {

		return ctrl.Result{}, err
	} else {

		logger.Info(
			"ConfigMap found",
			"ConfigMap", ConfigMapName,
		)

		if existingConfigMap.Data["dashboard.json"] != dashboardJSON {
			existingConfigMap.Data["dashboard.json"] = dashboardJSON

			if err := r.Update(ctx, existingConfigMap); err != nil {
				return ctrl.Result{}, err
			}

			logger.Info(
				"ConfigMap updated",
				"ConfigMap", ConfigMapName,
			)
		}
		logger.Info(
			"ConfigMap unchanged",
			"ConfigMap", ConfigMapName,
		)
	}

	// --------------------------------------------------
	// 3. Define the desired prometheus rule
	// --------------------------------------------------

	desiredPromRule := buildPrometheusRule(
		myobsStack.Spec.TargetDeployment,
		myobsStack.Spec.CPUThreshold,
		myobsStack.Spec.MemoryThreshold,
	)

	desiredPromRule.Name = myobsStack.Name + "-rules"
	desiredPromRule.Namespace = myobsStack.Spec.TargetNamespace

	if err := controllerutil.SetControllerReference(
		&myobsStack,
		desiredPromRule,
		r.Scheme,
	); err != nil {
		return ctrl.Result{}, err
	}

	existingPromRule := &monitoringv1.PrometheusRule{}

	rulekey := types.NamespacedName{
		Name:      desiredPromRule.Name,
		Namespace: desiredPromRule.Namespace,
	}

	err = r.Get(ctx, rulekey, existingPromRule)

	if apierrors.IsNotFound(err) {
		logger.Info(
			"Prometheus rule not found",
			"Prometheus rule", desiredPromRule.Name,
		)

		if err := r.Create(ctx, desiredPromRule); err != nil {
			if apierrors.IsAlreadyExists(err) {
				logger.Info(
					"Prometheus rule already exists; will reconcile again",
					"Prometheus rule", desiredPromRule.Name,
				)
				return ctrl.Result{RequeueAfter: time.Second}, nil
			}
			return ctrl.Result{}, err
		}
		logger.Info(
			"Prometheus rule created",
			"Prometheus rule", desiredPromRule.Name,
		)

	} else if err != nil {

		return ctrl.Result{}, err
	} else {

		logger.Info(
			"Prometheus rule found",
			"PrometheusRule", existingPromRule.Name,
		)

		if !reflect.DeepEqual(existingPromRule.Spec, desiredPromRule.Spec) {
			existingPromRule.Spec = desiredPromRule.Spec

			if err := r.Update(ctx, existingPromRule); err != nil {
				return ctrl.Result{}, err
			}

			logger.Info(
				"Prometheus rule updated",
				"Prometheus rule", existingPromRule.Name,
			)
		}
		logger.Info(
			"Prometheus rule unchanged",
			"Prometheus rule", existingPromRule.Name,
		)
	}

	// --------------------------------------------------
	// 3. Define the desired service monitor
	// --------------------------------------------------

	desiredServiceMonitor := buildServiceMonitor(
		myobsStack.Spec.TargetDeployment,
		myobsStack.Spec.MetricsPort,
		myobsStack.Spec.MetricsPath,
	)

	desiredServiceMonitor.Name =
		myobsStack.Name + "-monitor"

	desiredServiceMonitor.Namespace =
		myobsStack.Spec.TargetNamespace

	if err := controllerutil.SetControllerReference(
		&myobsStack,
		desiredServiceMonitor,
		r.Scheme,
	); err != nil {
		return ctrl.Result{}, err
	}

	existingServiceMonitor := &monitoringv1.ServiceMonitor{}

	ServiceMonitorKey := types.NamespacedName{
		Name:      desiredServiceMonitor.Name,
		Namespace: desiredServiceMonitor.Namespace,
	}

	err = r.Get(ctx, ServiceMonitorKey, existingServiceMonitor)

	if apierrors.IsNotFound(err) {

		logger.Info(
			"ServiceMonitor Not found",
			"ServiceMonitor",
			desiredServiceMonitor.Name,
		)

		if err := r.Create(ctx, desiredServiceMonitor); err != nil {
			if apierrors.IsAlreadyExists(err) {
				logger.Info(
					"ServiceMonitor already exists; will reconcile again",
					"ServiceMonitor", desiredServiceMonitor.Name,
				)
				return ctrl.Result{RequeueAfter: time.Second}, nil
			}
			return ctrl.Result{}, err
		}

		logger.Info(
			"ServiceMonitor created",
			"ServiceMonitor",
			desiredServiceMonitor.Name,
		)

	} else if err != nil {

		return ctrl.Result{}, err
	} else {

		logger.Info(
			"ServiceMonitor Found",
			"ServiceMonitor", desiredServiceMonitor.Name,
		)

		if !reflect.DeepEqual(existingServiceMonitor.Spec, desiredServiceMonitor.Spec) {

			existingServiceMonitor.Spec =
				desiredServiceMonitor.Spec

			if err := r.Update(
				ctx,
				existingServiceMonitor,
			); err != nil {
				return ctrl.Result{}, err
			}

			logger.Info(
				"ServiceMonitor updated",
				"ServiceMonitor",
				desiredServiceMonitor.Name,
			)
		} else {
			logger.Info(
				"ServiceMonitor unchanged",
				"ServiceMonitor", desiredServiceMonitor.Name,
			)
		}
	}

	if err := r.setReadyCondition(
		ctx,
		&myobsStack,
		metav1.ConditionTrue,
		"ReconciliationsSucceeded",
		"Dashboard, alert rules, and ServiceMonitor are configured",
	); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *CustomObservabilityStackReconciler) setReadyCondition(
	ctx context.Context,
	stack *appsv1.CustomObservabilityStack,
	status metav1.ConditionStatus,
	reason string,
	message string,
) error {
	key := types.NamespacedName{
		Name:      stack.Name,
		Namespace: stack.Namespace,
	}

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// Fetch the latest version on every retry.
		latest := &appsv1.CustomObservabilityStack{}
		if err := r.Get(ctx, key, latest); err != nil {
			return err
		}

		// Preserve the original status for comparison.
		originalStatus := latest.Status.DeepCopy()

		condition := metav1.Condition{
			Type:               "Ready",
			Status:             status,
			Reason:             reason,
			Message:            message,
			ObservedGeneration: latest.Generation,
		}
		meta.SetStatusCondition(&latest.Status.Conditions, condition)
		latest.Status.ObservedGeneration = latest.Generation

		statusChanged := !reflect.DeepEqual(originalStatus.Conditions, latest.Status.Conditions) || latest.Status.ObservedGeneration != originalStatus.ObservedGeneration

		if !statusChanged {
			return nil
		}

		now := metav1.Now()
		latest.Status.LastReconcileTime = &now

		return r.Status().Update(ctx, latest)
	})
}

// SetupWithManager sets up the controller with the Manager.
func (r *CustomObservabilityStackReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.CustomObservabilityStack{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&monitoringv1.PrometheusRule{}).
		Owns(&monitoringv1.ServiceMonitor{}).
		Named("customobservabilitystack").
		Complete(r)
}
