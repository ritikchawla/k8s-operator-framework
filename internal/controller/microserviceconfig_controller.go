/*
Copyright 2025.

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
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	configv1alpha1 "github.com/ritikchawla/k8s-operator-framework/api/v1alpha1"
	networkingv1alpha3 "istio.io/api/networking/v1alpha3"
	istioclientv1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
)

// MicroserviceConfigReconciler reconciles a MicroserviceConfig object
type MicroserviceConfigReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=config.ritikchawla.dev,resources=microserviceconfigs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=config.ritikchawla.dev,resources=microserviceconfigs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=config.ritikchawla.dev,resources=microserviceconfigs/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.istio.io,resources=virtualservices;destinationrules,verbs=get;list;watch;create;update;patch;delete

func (r *MicroserviceConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Fetch the MicroserviceConfig instance
	microserviceConfig := &configv1alpha1.MicroserviceConfig{}
	err := r.Get(ctx, req.NamespacedName, microserviceConfig)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Initialize the status if it's nil
	if microserviceConfig.Status.Conditions == nil {
		microserviceConfig.Status.Conditions = []metav1.Condition{}
	}

	// Create or update the Deployment
	deployment := &appsv1.Deployment{}
	err = r.Get(ctx, types.NamespacedName{Name: microserviceConfig.Name, Namespace: microserviceConfig.Namespace}, deployment)
	if err != nil && errors.IsNotFound(err) {
		// Create new deployment
		deployment = r.constructDeployment(microserviceConfig)
		if err := controllerutil.SetControllerReference(microserviceConfig, deployment, r.Scheme); err != nil {
			return ctrl.Result{}, err
		}
		if err := r.Create(ctx, deployment); err != nil {
			log.Error(err, "Failed to create Deployment")
			return ctrl.Result{}, err
		}
	} else if err == nil {
		// Update existing deployment
		deployment = r.updateDeployment(deployment, microserviceConfig)
		if err := r.Update(ctx, deployment); err != nil {
			log.Error(err, "Failed to update Deployment")
			return ctrl.Result{}, err
		}
	}

	// Create or update the Service
	service := &corev1.Service{}
	err = r.Get(ctx, types.NamespacedName{Name: microserviceConfig.Name, Namespace: microserviceConfig.Namespace}, service)
	if err != nil && errors.IsNotFound(err) {
		// Create new service
		service = r.constructService(microserviceConfig)
		if err := controllerutil.SetControllerReference(microserviceConfig, service, r.Scheme); err != nil {
			return ctrl.Result{}, err
		}
		if err := r.Create(ctx, service); err != nil {
			log.Error(err, "Failed to create Service")
			return ctrl.Result{}, err
		}
	}

	// Handle Istio configuration if enabled
	if microserviceConfig.Spec.Istio != nil && microserviceConfig.Spec.Istio.Enabled {
		if err := r.reconcileIstioConfig(ctx, microserviceConfig); err != nil {
			log.Error(err, "Failed to reconcile Istio configuration")
			return ctrl.Result{}, err
		}
	}

	// Update status
	microserviceConfig.Status.Phase = "Running"
	microserviceConfig.Status.LastUpdated = &metav1.Time{Time: time.Now()}
	if err := r.Status().Update(ctx, microserviceConfig); err != nil {
		log.Error(err, "Failed to update MicroserviceConfig status")
		return ctrl.Result{}, err
	}

	// Requeue after 10 minutes to check GitOps sync status
	return ctrl.Result{RequeueAfter: time.Minute * 10}, nil
}

func (r *MicroserviceConfigReconciler) constructDeployment(config *configv1alpha1.MicroserviceConfig) *appsv1.Deployment {
	replicas := int32(1)
	if config.Spec.Service.Replicas != nil {
		replicas = *config.Spec.Service.Replicas
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.Name,
			Namespace: config.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": config.Name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": config.Name,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  config.Name,
							Image: config.Spec.Service.Image,
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: config.Spec.Service.Port,
								},
							},
						},
					},
				},
			},
		},
	}

	// Add resource requirements if specified
	if config.Spec.Service.Resources != nil {
		deployment.Spec.Template.Spec.Containers[0].Resources = corev1.ResourceRequirements{
			Requests: corev1.ResourceList{},
			Limits:   corev1.ResourceList{},
		}
		if config.Spec.Service.Resources.CPU != "" {
			deployment.Spec.Template.Spec.Containers[0].Resources.Requests[corev1.ResourceCPU] = resource.MustParse(config.Spec.Service.Resources.CPU)
			deployment.Spec.Template.Spec.Containers[0].Resources.Limits[corev1.ResourceCPU] = resource.MustParse(config.Spec.Service.Resources.CPU)
		}
		if config.Spec.Service.Resources.Memory != "" {
			deployment.Spec.Template.Spec.Containers[0].Resources.Requests[corev1.ResourceMemory] = resource.MustParse(config.Spec.Service.Resources.Memory)
			deployment.Spec.Template.Spec.Containers[0].Resources.Limits[corev1.ResourceMemory] = resource.MustParse(config.Spec.Service.Resources.Memory)
		}
	}

	// Add environment variables if specified
	if len(config.Spec.Service.Env) > 0 {
		envVars := make([]corev1.EnvVar, len(config.Spec.Service.Env))
		for i, env := range config.Spec.Service.Env {
			envVars[i] = corev1.EnvVar{
				Name:  env.Name,
				Value: env.Value,
			}
		}
		deployment.Spec.Template.Spec.Containers[0].Env = envVars
	}

	return deployment
}

func (r *MicroserviceConfigReconciler) updateDeployment(existing *appsv1.Deployment, config *configv1alpha1.MicroserviceConfig) *appsv1.Deployment {
	// Update image and port
	existing.Spec.Template.Spec.Containers[0].Image = config.Spec.Service.Image
	existing.Spec.Template.Spec.Containers[0].Ports[0].ContainerPort = config.Spec.Service.Port

	// Update replicas if specified
	if config.Spec.Service.Replicas != nil {
		existing.Spec.Replicas = config.Spec.Service.Replicas
	}

	// Update resources if specified
	if config.Spec.Service.Resources != nil {
		existing.Spec.Template.Spec.Containers[0].Resources = corev1.ResourceRequirements{
			Requests: corev1.ResourceList{},
			Limits:   corev1.ResourceList{},
		}
		if config.Spec.Service.Resources.CPU != "" {
			existing.Spec.Template.Spec.Containers[0].Resources.Requests[corev1.ResourceCPU] = resource.MustParse(config.Spec.Service.Resources.CPU)
			existing.Spec.Template.Spec.Containers[0].Resources.Limits[corev1.ResourceCPU] = resource.MustParse(config.Spec.Service.Resources.CPU)
		}
		if config.Spec.Service.Resources.Memory != "" {
			existing.Spec.Template.Spec.Containers[0].Resources.Requests[corev1.ResourceMemory] = resource.MustParse(config.Spec.Service.Resources.Memory)
			existing.Spec.Template.Spec.Containers[0].Resources.Limits[corev1.ResourceMemory] = resource.MustParse(config.Spec.Service.Resources.Memory)
		}
	}

	// Update environment variables if specified
	if len(config.Spec.Service.Env) > 0 {
		envVars := make([]corev1.EnvVar, len(config.Spec.Service.Env))
		for i, env := range config.Spec.Service.Env {
			envVars[i] = corev1.EnvVar{
				Name:  env.Name,
				Value: env.Value,
			}
		}
		existing.Spec.Template.Spec.Containers[0].Env = envVars
	}

	return existing
}

func (r *MicroserviceConfigReconciler) constructService(config *configv1alpha1.MicroserviceConfig) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.Name,
			Namespace: config.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": config.Name,
			},
			Ports: []corev1.ServicePort{
				{
					Port:       config.Spec.Service.Port,
					TargetPort: intstr.FromInt(int(config.Spec.Service.Port)),
					Protocol:   corev1.ProtocolTCP,
				},
			},
		},
	}
}

func (r *MicroserviceConfigReconciler) reconcileIstioConfig(ctx context.Context, config *configv1alpha1.MicroserviceConfig) error {
	if config.Spec.Istio.VirtualService != nil {
		virtualService := &istioclientv1alpha3.VirtualService{
			ObjectMeta: metav1.ObjectMeta{
				Name:      config.Name,
				Namespace: config.Namespace,
			},
			Spec: networkingv1alpha3.VirtualService{
				Hosts:    config.Spec.Istio.VirtualService.Hosts,
				Gateways: config.Spec.Istio.VirtualService.Gateways,
			},
		}

		// Create or update VirtualService
		err := r.Get(ctx, types.NamespacedName{Name: config.Name, Namespace: config.Namespace}, virtualService)
		if err != nil && errors.IsNotFound(err) {
			if err := controllerutil.SetControllerReference(config, virtualService, r.Scheme); err != nil {
				return fmt.Errorf("failed to set controller reference: %w", err)
			}
			if err := r.Create(ctx, virtualService); err != nil {
				return fmt.Errorf("failed to create VirtualService: %w", err)
			}
		} else if err == nil {
			if err := r.Update(ctx, virtualService); err != nil {
				return fmt.Errorf("failed to update VirtualService: %w", err)
			}
		} else {
			return fmt.Errorf("failed to get VirtualService: %w", err)
		}
	}

	if config.Spec.Istio.DestinationRule != nil {
		destinationRule := &istioclientv1alpha3.DestinationRule{
			ObjectMeta: metav1.ObjectMeta{
				Name:      config.Name,
				Namespace: config.Namespace,
			},
			Spec: networkingv1alpha3.DestinationRule{
				Host: config.Name,
			},
		}

		if config.Spec.Istio.DestinationRule.TrafficPolicy != nil {
			destinationRule.Spec.TrafficPolicy = &networkingv1alpha3.TrafficPolicy{
				LoadBalancer: &networkingv1alpha3.LoadBalancerSettings{},
			}
		}

		// Create or update DestinationRule
		err := r.Get(ctx, types.NamespacedName{Name: config.Name, Namespace: config.Namespace}, destinationRule)
		if err != nil && errors.IsNotFound(err) {
			if err := controllerutil.SetControllerReference(config, destinationRule, r.Scheme); err != nil {
				return fmt.Errorf("failed to set controller reference: %w", err)
			}
			if err := r.Create(ctx, destinationRule); err != nil {
				return fmt.Errorf("failed to create DestinationRule: %w", err)
			}
		} else if err == nil {
			if err := r.Update(ctx, destinationRule); err != nil {
				return fmt.Errorf("failed to update DestinationRule: %w", err)
			}
		} else {
			return fmt.Errorf("failed to get DestinationRule: %w", err)
		}
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MicroserviceConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&configv1alpha1.MicroserviceConfig{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&istioclientv1alpha3.VirtualService{}).
		Owns(&istioclientv1alpha3.DestinationRule{}).
		Complete(r)
}
