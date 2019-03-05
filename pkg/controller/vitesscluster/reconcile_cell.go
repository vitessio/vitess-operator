package vitesscluster

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	vitessv1alpha2 "vitess.io/vitess-operator/pkg/apis/vitess/v1alpha2"
	"vitess.io/vitess-operator/pkg/util/scripts"
)

func (r *ReconcileVitessCluster) ReconcileCell(cell *vitessv1alpha2.VitessCell) (reconcile.Result, error) {
	log.Info("Reconciling Cell", "Namespace", cell.GetNamespace(), "VitessCluster.Name", cell.Cluster().GetName(), "Cell.Name", cell.GetName())

	if r, err := r.ReconcileCellVTctld(cell); err != nil {
		log.Error(err, "Failed to reconcile vtctl", "Namespace", cell.GetName(), "VitessCluster.Name", cell.Cluster().GetName(), "Cell.Name", cell.GetName())
		return r, err
	} else if r.Requeue {
		return r, err
	}

	if r, err := r.ReconcileCellVTGate(cell); err != nil {
		log.Error(err, "Failed to reconcile vtgate", "Namespace", cell.GetName(), "VitessCluster.Name", cell.Cluster().GetName(), "Cell.Name", cell.GetName())
		return r, err
	} else if r.Requeue {
		return r, err
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileVitessCluster) ReconcileCellVTctld(cell *vitessv1alpha2.VitessCell) (reconcile.Result, error) {
	deploy, service, deployErr := GetCellVTctldResources(cell)
	if deployErr != nil {
		log.Error(deployErr, "failed to generate Vtctld Deployment for VitessCell", "VitessCell.Namespace", cell.GetNamespace(), "VitessCell.Name", cell.GetNamespace())
		return reconcile.Result{}, deployErr
	}

	foundDeployment := &appsv1.Deployment{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: deploy.GetName(), Namespace: deploy.GetNamespace()}, foundDeployment)
	if err != nil && errors.IsNotFound(err) {
		controllerutil.SetControllerReference(cell.Cluster(), deploy, r.scheme)
		err = r.client.Create(context.TODO(), deploy)
		if err != nil {
			return reconcile.Result{}, err
		}
	} else if err != nil {
		log.Error(err, "failed to get Deployment")
		return reconcile.Result{}, err
	}

	foundService := &corev1.Service{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: service.GetName(), Namespace: service.GetNamespace()}, foundService)
	if err != nil && errors.IsNotFound(err) {
		controllerutil.SetControllerReference(cell.Cluster(), service, r.scheme)
		err = r.client.Create(context.TODO(), service)
		if err != nil {
			return reconcile.Result{}, err
		}
	} else if err != nil {
		log.Error(err, "failed to get Service")
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil

}

func GetCellVTctldResources(cell *vitessv1alpha2.VitessCell) (*appsv1.Deployment, *corev1.Service, error) {
	name := cell.GetScopedName("vtctld")

	scripts := scripts.NewContainerScriptGenerator("vtctld", cell)
	if err := scripts.Generate(); err != nil {
		return nil, nil, err
	}

	labels := map[string]string{
		"app":       "vitess",
		"cluster":   cell.Cluster().GetName(),
		"cell":      cell.GetName(),
		"component": "vtctld",
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cell.Cluster().GetNamespace(),
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			ProgressDeadlineSeconds: getInt32Ptr(1),
			Replicas:                getInt32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "vtctld",
							Image: "vitess/vtctld:helm-1.0.3", // TODO use CRD w/default
							Command: []string{
								"bash",
							},
							Args: []string{
								"-c",
								scripts.Start,
							},
							LivenessProbe: &corev1.Probe{
								Handler: corev1.Handler{
									HTTPGet: &corev1.HTTPGetAction{
										Path:   "/debug/status",
										Port:   intstr.FromInt(15000),
										Scheme: corev1.URISchemeHTTP,
									},
								},
								InitialDelaySeconds: 30,
								TimeoutSeconds:      5,
								PeriodSeconds:       10,
								SuccessThreshold:    1,
								FailureThreshold:    3,
							},
							ReadinessProbe: &corev1.Probe{
								Handler: corev1.Handler{
									HTTPGet: &corev1.HTTPGetAction{
										Path:   "/debug/health",
										Port:   intstr.FromInt(15000),
										Scheme: corev1.URISchemeHTTP,
									},
								},
								InitialDelaySeconds: 30,
								TimeoutSeconds:      5,
								PeriodSeconds:       10,
								SuccessThreshold:    1,
								FailureThreshold:    3,
							},
						},
					},
					SecurityContext: &corev1.PodSecurityContext{
						FSGroup:   getInt64Ptr(2000),
						RunAsUser: getInt64Ptr(1000),
					},
				},
			},
		},
	}

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cell.Cluster().GetNamespace(),
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Type:     corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{
				{
					Name: "web",
					Port: 15000,
				},
				{
					Name: "grpc",
					Port: 15999,
				},
			},
		},
	}

	return deployment, service, nil
}

func (r *ReconcileVitessCluster) ReconcileCellVTGate(cell *vitessv1alpha2.VitessCell) (reconcile.Result, error) {
	deploy, service, deployErr := GetCellVTGateResources(cell)
	if deployErr != nil {
		log.Error(deployErr, "failed to generate VTGate Deployment for VitessCell", "VitessCell.Namespace", cell.GetNamespace(), "VitessCell.Name", cell.GetNamespace())
		return reconcile.Result{}, deployErr
	}

	foundDeployment := &appsv1.Deployment{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: deploy.GetName(), Namespace: deploy.GetNamespace()}, foundDeployment)
	if err != nil && errors.IsNotFound(err) {
		controllerutil.SetControllerReference(cell.Cluster(), deploy, r.scheme)
		err = r.client.Create(context.TODO(), deploy)
		if err != nil {
			return reconcile.Result{}, err
		}
	} else if err != nil {
		log.Error(err, "failed to get Deployment")
		return reconcile.Result{}, err
	}

	foundService := &corev1.Service{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: service.GetName(), Namespace: service.GetNamespace()}, foundService)
	if err != nil && errors.IsNotFound(err) {
		controllerutil.SetControllerReference(cell.Cluster(), service, r.scheme)
		err = r.client.Create(context.TODO(), service)
		if err != nil {
			return reconcile.Result{}, err
		}
	} else if err != nil {
		log.Error(err, "failed to get Service")
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil

}

func GetCellVTGateResources(cell *vitessv1alpha2.VitessCell) (*appsv1.Deployment, *corev1.Service, error) {
	name := cell.GetScopedName("vtgate")

	scriptGen := scripts.NewContainerScriptGenerator("vtgate", cell)
	if err := scriptGen.Generate(); err != nil {
		return nil, nil, err
	}

	vtgateLabels := map[string]string{
		"app":       "vitess",
		"cluster":   cell.Cluster().GetName(),
		"cell":      cell.GetName(),
		"component": "vtgate",
	}

	vttabletLabels := map[string]string{
		"app":       "vitess",
		"cluster":   cell.Cluster().GetName(),
		"cell":      cell.GetName(),
		"component": "vttabletLabels",
	}

	// Build affinity
	affinity := &corev1.Affinity{
		PodAffinity: &corev1.PodAffinity{
			// Prefer to run on the same host as a vtgate pod
			PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
				{
					Weight: 10,
					PodAffinityTerm: corev1.PodAffinityTerm{
						LabelSelector: &metav1.LabelSelector{
							MatchLabels: vttabletLabels,
						},
						TopologyKey: "kubernetes.io/hostname",
					},
				},
			},
		},
		PodAntiAffinity: &corev1.PodAntiAffinity{
			PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
				{
					Weight: 100,
					PodAffinityTerm: corev1.PodAffinityTerm{
						LabelSelector: &metav1.LabelSelector{
							MatchLabels: vtgateLabels,
						},
						TopologyKey: "kubernetes.io/hostname",
					},
				},
			},
		},
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cell.Cluster().GetNamespace(),
			Labels:    vtgateLabels,
		},
		Spec: appsv1.DeploymentSpec{
			ProgressDeadlineSeconds: getInt32Ptr(600),
			Replicas:                getInt32Ptr(2),
			Selector: &metav1.LabelSelector{
				MatchLabels: vtgateLabels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: vtgateLabels,
				},
				Spec: corev1.PodSpec{
					Affinity: affinity,
					Containers: []corev1.Container{
						{
							Name:  "vtgate",
							Image: "vitess/vtgate:helm-1.0.3", // TODO use CRD w/default
							Command: []string{
								"bash",
							},
							Args: []string{
								"-c",
								scriptGen.Start,
							},
							LivenessProbe: &corev1.Probe{
								Handler: corev1.Handler{
									HTTPGet: &corev1.HTTPGetAction{
										Path:   "/debug/status",
										Port:   intstr.FromInt(15001),
										Scheme: corev1.URISchemeHTTP,
									},
								},
								InitialDelaySeconds: 30,
								TimeoutSeconds:      5,
								PeriodSeconds:       10,
								SuccessThreshold:    1,
								FailureThreshold:    3,
							},
							ReadinessProbe: &corev1.Probe{
								Handler: corev1.Handler{
									HTTPGet: &corev1.HTTPGetAction{
										Path:   "/debug/health",
										Port:   intstr.FromInt(15001),
										Scheme: corev1.URISchemeHTTP,
									},
								},
								InitialDelaySeconds: 30,
								TimeoutSeconds:      5,
								PeriodSeconds:       10,
								SuccessThreshold:    1,
								FailureThreshold:    3,
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									MountPath: "/mysqlcreds",
									Name:      "creds",
								},
							},
						},
					},
					SecurityContext: &corev1.PodSecurityContext{
						FSGroup:   getInt64Ptr(2000),
						RunAsUser: getInt64Ptr(1000),
					},
					Volumes: []corev1.Volume{
						{
							Name: "creds",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
				},
			},
		},
	}

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cell.Cluster().GetNamespace(),
			Labels:    vtgateLabels,
		},
		Spec: corev1.ServiceSpec{
			Selector: vtgateLabels,
			Type:     corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{
				{
					Name: "web",
					Port: 15001,
				},
				{
					Name: "grpc",
					Port: 15991,
				},
			},
		},
	}

	if cell.Spec.MySQLProtocol != nil {
		// Add Service Port
		service.Spec.Ports = append(service.Spec.Ports, corev1.ServicePort{
			Name: "mysql",
			Port: 3306,
		})

		// Setup credential init container
		if cell.Spec.MySQLProtocol.PasswordSecretRef != nil {
			scriptGen := scripts.NewContainerScriptGenerator("init-mysql-creds", cell)
			if err := scriptGen.Generate(); err != nil {
				return nil, nil, err
			}

			// Add deployment initContainer to bootstrap creds
			deployment.Spec.Template.Spec.InitContainers = append(deployment.Spec.Template.Spec.InitContainers, corev1.Container{
				Name:  "init-mysql-creds",
				Image: "vitess/vtgate:helm-1.0.3", // TODO use CRD w/default
				Env: []corev1.EnvVar{
					{
						Name: "MYSQL_PASSWORD",
						ValueFrom: &corev1.EnvVarSource{
							SecretKeyRef: cell.Spec.MySQLProtocol.PasswordSecretRef,
						},
					},
				},
				Command: []string{
					"bash",
				},
				Args: []string{
					"-c",
					scriptGen.Start,
				},
				VolumeMounts: []corev1.VolumeMount{
					{
						MountPath: "/mysqlcreds",
						Name:      "creds",
					},
				},
			})
		}
	}

	return deployment, service, nil
}
