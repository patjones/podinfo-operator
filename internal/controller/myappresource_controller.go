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
	"reflect"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	myapigroupv1beta1 "github.com/patjones/podinfo-operator/api/v1beta1"
	"github.com/pingcap/errors"
)

// MyAppResourceReconciler reconciles a MyAppResource object
type MyAppResourceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func createDeployment(myappresource myapigroupv1beta1.MyAppResource, r MyAppResourceReconciler) *appsv1.Deployment {
	labels := map[string]string{
		"app":                          myappresource.Name,
		"app.kubernetes.io/name":       myappresource.Name,
		"app.kubernetes.io/managed-by": "podinfo-operator",
	}

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
			Value: "tcp://" + myappresource.Name + "-redis:6379",
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
									corev1.ResourceCPU: resource.MustParse(myappresource.Spec.Resources.CpuRequest),
								},
							},
							Env: environmentVars,
						},
					},
				},
			},
		},
	}

	ctrl.SetControllerReference(&myappresource, dep, r.Scheme)
	return dep
}

func createService(myappresource myapigroupv1beta1.MyAppResource, r MyAppResourceReconciler) *corev1.Service {

	labels := map[string]string{
		"app":                          myappresource.Name,
		"app.kubernetes.io/name":       myappresource.Name,
		"app.kubernetes.io/managed-by": "podinfo-operator",
	}

	svc := &corev1.Service{
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
	ctrl.SetControllerReference(&myappresource, svc, r.Scheme)
	return svc
}

func createRedisDeployment(myappresource myapigroupv1beta1.MyAppResource, r MyAppResourceReconciler) *appsv1.Deployment {

	labels := map[string]string{
		"app":                          myappresource.Name + "-redis",
		"app.kubernetes.io/name":       myappresource.Name + "-redis",
		"app.kubernetes.io/managed-by": "podinfo-operator",
	}

	replicas := int32(1)
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      myappresource.Name + "-redis",
			Namespace: myappresource.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,

			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},

			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "redis",
							Image: "redis:5.0.4",
							Ports: []corev1.ContainerPort{
								{
									Name:          "redis",
									ContainerPort: 6379,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									Exec: &corev1.ExecAction{
										Command: []string{"redis-cli", "ping"},
									},
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "redis-data",
									MountPath: "/var/lib/redis",
								},
								{
									Name:      "redis-config",
									MountPath: "redis-master",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "redis-data",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
						{
							Name: "redis-config",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: myappresource.Name + "-redis",
									},
								},
							},
						},
					},
				},
			},
		},
	}
	ctrl.SetControllerReference(&myappresource, dep, r.Scheme)
	return dep
}

func createRedisService(myappresource myapigroupv1beta1.MyAppResource, r MyAppResourceReconciler) *corev1.Service {

	labels := map[string]string{
		"app":                          myappresource.Name + "-redis",
		"app.kubernetes.io/name":       myappresource.Name + "-redis",
		"app.kubernetes.io/managed-by": "podinfo-operator",
	}

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      myappresource.Name + "-redis",
			Namespace: myappresource.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports: []corev1.ServicePort{
				{
					Name:       "redis",
					Protocol:   corev1.ProtocolTCP,
					Port:       6379,
					TargetPort: intstr.FromInt(6379),
				},
			},
		},
	}
	ctrl.SetControllerReference(&myappresource, svc, r.Scheme)
	return svc
}

func createRedisConfigMap(myappresource myapigroupv1beta1.MyAppResource, r MyAppResourceReconciler) *corev1.ConfigMap {
	labels := map[string]string{
		"app":                          myappresource.Name + "-redis",
		"app.kubernetes.io/name":       myappresource.Name + "-redis",
		"app.kubernetes.io/managed-by": "podinfo-operator",
	}

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      myappresource.Name + "-redis",
			Namespace: myappresource.Namespace,
			Labels:    labels,
		},
		Data: map[string]string{
			"redis.conf": `
				maxmemory 64mb
				maxmemory-policy allkeys-lru
				save ""
				appendonly no
			`,
		},
	}
	ctrl.SetControllerReference(&myappresource, cm, r.Scheme)
	return cm
}

//+kubebuilder:rbac:groups=my.api.group.patjones.io,resources=myappresources,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=my.api.group.patjones.io,resources=myappresources/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=my.api.group.patjones.io,resources=myappresources/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;delete;create;update;patch
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;delete;create;update;patch
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;delete;create;update;patch

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

	var myappresource myapigroupv1beta1.MyAppResource
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
			}
			log.Log.Info("Deployment created successfully", "deployment", deployment)
		}
	} else {
		log.Log.Info("Deployment already exists, checking for changes", "deployment", deployment)
		updatedDeployment := createDeployment(myappresource, *r)
		if !reflect.DeepEqual(deployment.Spec, updatedDeployment.Spec) {
			deployment.Spec = updatedDeployment.Spec
			err = r.Update(ctx, &deployment)
			if err != nil {
				log.Log.Error(err, "Error updating Deployment for MyAppResource", "deployment", deployment)
			}
			log.Log.Info("Deployment updated successfully", "deployment", deployment)
		}
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
			}
		}
	}

	if myappresource.Spec.Redis.Enabled {
		//Manage Redis Deployment
		redisDeployment := appsv1.Deployment{}
		err = r.Get(ctx, client.ObjectKey{Namespace: myappresource.ObjectMeta.Namespace, Name: myappresource.ObjectMeta.Name + "-redis"}, &redisDeployment)
		if err != nil {
			if errors.IsNotFound(err) {
				log.Log.Info("Creating Redis Deployment", "Namespace", myappresource.ObjectMeta.Namespace, "Name", myappresource.ObjectMeta.Name+"-redis")
				redisDeployment = *createRedisDeployment(myappresource, *r)
				err = r.Create(ctx, &redisDeployment)
				if err != nil {
					log.Log.Error(err, "Error creating Deployment for Redis", "deployment", redisDeployment)
				}
			}
		}

		//Manage Redis Service
		redisService := corev1.Service{}
		err = r.Get(ctx, client.ObjectKey{Namespace: myappresource.ObjectMeta.Namespace, Name: myappresource.ObjectMeta.Name + "-redis"}, &redisService)
		if err != nil {
			if errors.IsNotFound(err) {
				log.Log.Info("Creating Redis Service", "Namespace", myappresource.ObjectMeta.Namespace, "Name", myappresource.ObjectMeta.Name+"-redis")
				redisService = *createRedisService(myappresource, *r)
				err = r.Create(ctx, &redisService)
				if err != nil {
					log.Log.Error(err, "Error creating Service for Redis", "service", redisService)
				}
			}
		}

		//Manage Redis ConfigMap
		redisConfigMap := corev1.ConfigMap{}
		err = r.Get(ctx, client.ObjectKey{Namespace: myappresource.ObjectMeta.Namespace, Name: myappresource.ObjectMeta.Name + "-redis"}, &redisConfigMap)
		if err != nil {
			if errors.IsNotFound(err) {
				log.Log.Info("Creating Redis ConfigMap", "Namespace", myappresource.ObjectMeta.Namespace, "Name", myappresource.ObjectMeta.Name+"-redis")
				redisConfigMap = *createRedisConfigMap(myappresource, *r)
				err = r.Create(ctx, &redisConfigMap)
				if err != nil {
					log.Log.Error(err, "Error creating ConfigMap for Redis", "configmap", redisConfigMap)
				}
			}
		}
	} else {
		err = r.Get(ctx, client.ObjectKey{Namespace: myappresource.ObjectMeta.Namespace, Name: myappresource.ObjectMeta.Name + "-redis"}, &appsv1.Deployment{})
		if err == nil {
			log.Log.Info("Redis is disabled, cleaning up Redis resources")
			err = r.Delete(ctx, &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Namespace: myappresource.ObjectMeta.Namespace, Name: myappresource.ObjectMeta.Name + "-redis"}})
			if err != nil {
				log.Log.Error(err, "Error deleting Redis Deployment")
			}
			err = r.Delete(ctx, &corev1.Service{ObjectMeta: metav1.ObjectMeta{Namespace: myappresource.ObjectMeta.Namespace, Name: myappresource.ObjectMeta.Name + "-redis"}})
			if err != nil {
				log.Log.Error(err, "Error deleting Redis Service")
			}
			err = r.Delete(ctx, &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Namespace: myappresource.ObjectMeta.Namespace, Name: myappresource.ObjectMeta.Name + "-redis"}})
			if err != nil {
				log.Log.Error(err, "Error deleting Redis ConfigMap")
			}
		}
	}

	//TODO if not enabled check redis was previously deployed and clean it up

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MyAppResourceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&myapigroupv1beta1.MyAppResource{}).
		Complete(r)
}
