/*
Copyright 2024.

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

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	myapigroupv1alpha1 "github.com/patjones/podinfo-operator/api/v1alpha1"
	"github.com/pingcap/errors"
)

// MyAppResourceReconciler reconciles a MyAppResource object
type MyAppResourceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func createDeployment(myappresource myapigroupv1alpha1.MyAppResource, r MyAppResourceReconciler) *appsv1.Deployment {
	labels := map[string]string{
		"app": myappresource.Name,
	}

	// ...

	environmentVars := []corev1.EnvVar{
		{
			Name:  "PODINFO_UI_COLOR",
			Value: myappresource.Spec.UI.Color,
		},
		{
			Name:  "PODINFO_UI_MESSAGE",
			Value: myappresource.Spec.UI.Message,
		},
	}
	if myappresource.Spec.Redis.Enabled {
		environmentVars = append(environmentVars, corev1.EnvVar{
			Name:  "PODINFO_CACHE_SERVER",
			Value: "tcp://localhost:6379",
		})
	}

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      myappresource.Name,
			Namespace: myappresource.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &myappresource.Spec.ReplicaCount,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},

			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "podinfo",
							Image: myappresource.Spec.Image.Repository + ":" + myappresource.Spec.Image.Tag,
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/healthz",
										Port: intstr.FromInt(9898),
									},
								},
							},
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									corev1.ResourceMemory: resource.MustParse(myappresource.Spec.Resources.MemoryLimit),
								},
								Requests: corev1.ResourceList{
									corev1.ResourceCPU: resource.MustParse(myappresource.Spec.Resources.CpuReuest),
								},
							},
							Env: environmentVars,
						},
						{
							Name:  "redis",
							Image: "redis:alpine",
						},
					},
				},
			},
		},
	}

	ctrl.SetControllerReference(&myappresource, dep, r.Scheme)
	return dep
}

func createService(myappresource myapigroupv1alpha1.MyAppResource, r MyAppResourceReconciler) *corev1.Service {

	labels := map[string]string{
		"app": myappresource.Name,
	}

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      myappresource.Name,
			Namespace: myappresource.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Protocol:   corev1.ProtocolTCP,
					Port:       80,
					TargetPort: intstr.FromInt(9898),
				},
			},
		},
	}
	return service
}

//+kubebuilder:rbac:groups=my.api.group,resources=myappresources,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=my.api.group,resources=myappresources/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=my.api.group,resources=myappresources/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the MyAppResource object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.17.0/pkg/reconcile
func (r *MyAppResourceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	var myappresource myapigroupv1alpha1.MyAppResource
	if err := r.Get(ctx, req.NamespacedName, &myappresource); err != nil {
		log.Log.Error(err, "unable to fetch MyAppResource")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	log.Log.Info("MyAppResource", "Name", myappresource.ObjectMeta.Name)

	//Manage Deployment
	deployment := appsv1.Deployment{}
	err := r.Get(ctx, client.ObjectKey{Namespace: myappresource.ObjectMeta.Namespace, Name: myappresource.ObjectMeta.Name}, &deployment)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Log.Info("Creating MyAppResource Deployment", "Namespace", myappresource.ObjectMeta.Namespace, "Name", myappresource.ObjectMeta.Name)
			deployment = *createDeployment(myappresource, *r)
			err = r.Create(ctx, &deployment)
			if err != nil {

				log.Log.Error(err, "Error creating Deployment for MyAppResource", "deployment", deployment)
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}
		log.Log.Error(err, "unable to fetch Deployment for MyAppResource", "deployment", deployment)
		return ctrl.Result{}, err
	}

	//Manage Service
	service := corev1.Service{}
	err = r.Get(ctx, client.ObjectKey{Namespace: myappresource.ObjectMeta.Namespace, Name: myappresource.ObjectMeta.Name}, &service)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Log.Info("Creating MyAppResource Service", "Namespace", myappresource.ObjectMeta.Namespace, "Name", myappresource.ObjectMeta.Name)
			service = *createService(myappresource, *r)
			err = r.Create(ctx, &service)
			if err != nil {
				log.Log.Error(err, "Error creating Service for MyAppResource", "service", service)
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}
		log.Log.Error(err, "unable to fetch Service for MyAppResource", "service", service)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MyAppResourceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&myapigroupv1alpha1.MyAppResource{}).
		Complete(r)
}
