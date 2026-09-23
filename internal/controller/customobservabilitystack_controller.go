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
	"reflect"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

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
	if err := r.Get(ctx, objKey, deployment); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
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
			return ctrl.Result{}, err
		}

		logger.Info(
			"ConfigMap created",
			"ConfigMap", ConfigMapName,
		)
		return ctrl.Result{}, nil
	}

	if err != nil {
		return ctrl.Result{}, err
	}

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
			return ctrl.Result{}, err
		}
		logger.Info(
			"Prometheus rule created",
			"Prometheus rule", desiredPromRule.Name,
		)
		return ctrl.Result{}, nil
	}

	if err != nil {
		return ctrl.Result{}, err
	}

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

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CustomObservabilityStackReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.CustomObservabilityStack{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&monitoringv1.PrometheusRule{}).
		Named("customobservabilitystack").
		Complete(r)
}
