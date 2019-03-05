package vitesscluster

import (
	"context"
	"fmt"
	"reflect"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	vitessv1alpha2 "vitess.io/vitess-operator/pkg/apis/vitess/v1alpha2"
	"vitess.io/vitess-operator/pkg/util/scripts"
)

func (r *ReconcileVitessCluster) ReconcileTablet(tablet *vitessv1alpha2.VitessTablet) (reconcile.Result, error) {
	log.Info("Reconciling Tablet", "Namespace", tablet.GetNamespace(), "VitessCluster.Name", tablet.Cluster().GetName(), "Tablet.Name", tablet.GetName())

	if r, err := r.ReconcileTabletResources(tablet); err != nil {
		log.Error(err, "Failed to reconcile tablet statefulset", "Namespace", tablet.GetName(), "VitessCluster.Name", tablet.Cluster().GetName(), "Tablet.Name", tablet.GetName())
		return r, err
	} else if r.Requeue {
		return r, err
	}

	// Create an init job for tablets of type replica
	// TODO replace this with direct election via the operator
	if tablet.Spec.Type == vitessv1alpha2.TabletTypeReplica {
		if r, err := r.ReconcileReplicaTabletInitJob(tablet); err != nil {
			log.Error(err, "Failed to reconcile replica tablet master init job", "Namespace", tablet.GetName(), "VitessCluster.Name", tablet.Cluster().GetName(), "Tablet.Name", tablet.GetName())
			return r, err
		} else if r.Requeue {
			return r, err
		}
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileVitessCluster) ReconcileTabletResources(tablet *vitessv1alpha2.VitessTablet) (reconcile.Result, error) {
	statefulSet, statefulSetErr := getStatefulSetForTablet(tablet)
	if statefulSetErr != nil {
		log.Error(statefulSetErr, "failed to generate StatefulSet for VitessTablet", "VitessTablet.Namespace", tablet.GetNamespace(), "VitessTablet.Name", tablet.GetNamespace())
		return reconcile.Result{}, statefulSetErr
	}

	foundStatefulSet := &appsv1.StatefulSet{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: statefulSet.GetName(), Namespace: tablet.Cluster().GetNamespace()}, foundStatefulSet)
	if err != nil && errors.IsNotFound(err) {
		controllerutil.SetControllerReference(tablet.Cluster(), statefulSet, r.scheme)

		err = r.client.Create(context.TODO(), statefulSet)
		if err != nil {
			return reconcile.Result{}, err
		}
	} else if err != nil {
		log.Error(err, "failed to get StatefulSet")
		return reconcile.Result{}, err
	} else {
		// This is a cheap way to detect changes and it works for now. However it is not perfect
		// it will always detect changes because of the defaulting values that get placed
		// on a statefulset when it is created in the cluster. Those values are not set in the
		// generated statefulset so it is always different. The extra updates are harmless and don't actually
		// trigger statefulset upgrades.
		// TODO more exact diff detection
		if !reflect.DeepEqual(foundStatefulSet.Spec.Template, statefulSet.Spec.Template) ||
			!reflect.DeepEqual(foundStatefulSet.Spec.Replicas, statefulSet.Spec.Replicas) ||
			!reflect.DeepEqual(foundStatefulSet.Spec.UpdateStrategy, statefulSet.Spec.UpdateStrategy) {
			log.Info("Updating statefulSet for tablet", "Namespace", tablet.GetNamespace(), "VitessCluster.Name", tablet.Cluster().GetName(), "Tablet.Name", tablet.GetName())

			// Update foundStatefulSet with changable fields from the generated StatefulSet

			// Only Template, replicas and updateStrategy may be updated on existing StatefulSet spec
			statefulSet.Spec.Template.DeepCopyInto(&foundStatefulSet.Spec.Template)
			statefulSet.Spec.Replicas = foundStatefulSet.Spec.Replicas
			statefulSet.Spec.UpdateStrategy.DeepCopyInto(&foundStatefulSet.Spec.UpdateStrategy)

			err = r.client.Update(context.TODO(), foundStatefulSet)
			if err != nil {
				return reconcile.Result{}, err
			}
		}

		// Set the tablet status based on the StatefulSet status
		// this is for use by the VitessCluster controller later
		if foundStatefulSet.Status.Replicas == foundStatefulSet.Status.ReadyReplicas {
			tablet.SetPhase(vitessv1alpha2.TabletPhaseReady)
		}
	}

	return reconcile.Result{}, nil
}

func getStatefulSetForTablet(tablet *vitessv1alpha2.VitessTablet) (*appsv1.StatefulSet, error) {
	selfLabels := map[string]string{
		"tabletname": tablet.GetName(),
		"app":        "vitess",
		"cluster":    tablet.Cluster().GetName(),
		"cell":       tablet.Cell().GetName(),
		"keyspace":   tablet.Keyspace().GetName(),
		"shard":      tablet.Shard().GetName(),
		"component":  "vttablet",
		"type":       string(tablet.Spec.Type),
	}

	vtgateLabels := map[string]string{
		"app":       "vitess",
		"cluster":   tablet.Cluster().GetName(),
		"cell":      tablet.Cell().GetName(),
		"component": "vtgate",
	}

	sameClusterTabletLabels := map[string]string{
		"app":       "vitess",
		"cluster":   tablet.Cluster().GetName(),
		"component": "vttablet",
	}

	sameShardTabletLabels := map[string]string{
		"app":       "vitess",
		"cluster":   tablet.Cluster().GetName(),
		"cell":      tablet.Cell().GetName(),
		"keyspace":  tablet.Keyspace().GetName(),
		"shard":     tablet.Shard().GetName(),
		"component": "vttablet",
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
							MatchLabels: vtgateLabels,
						},
						TopologyKey: "kubernetes.io/hostname",
					},
				},
			},
		},
		PodAntiAffinity: &corev1.PodAntiAffinity{
			PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
				// Hard preference to avoid running on the same host as another tablet in the same shard/keyspace
				{
					Weight: 100,
					PodAffinityTerm: corev1.PodAffinityTerm{
						LabelSelector: &metav1.LabelSelector{
							MatchLabels: sameShardTabletLabels,
						},
						TopologyKey: "kubernetes.io/hostname",
					},
				},
				// Soft preference to avoid running on the same host as another tablet in the same cluster
				{
					Weight: 10,
					PodAffinityTerm: corev1.PodAffinityTerm{
						LabelSelector: &metav1.LabelSelector{
							MatchLabels: sameClusterTabletLabels,
						},
						TopologyKey: "kubernetes.io/hostname",
					},
				},
			},
		},
	}

	dbContainers, dbInitContainers, err := GetTabletMysqlContainers(tablet)
	if err != nil {
		return nil, err
	}

	vttabletContainers, vttabletInitContainers, err := GetTabletVTTabletContainers(tablet)
	if err != nil {
		return nil, err
	}

	// build containers
	containers := []corev1.Container{}
	containers = append(containers, dbContainers...)
	containers = append(containers, vttabletContainers...)

	// build initcontainers
	initContainers := []corev1.Container{}
	initContainers = append(initContainers, dbInitContainers...)
	initContainers = append(initContainers, vttabletInitContainers...)

	// setup volume requests
	volumeRequests := make(corev1.ResourceList)
	volumeRequests[corev1.ResourceStorage] = resource.MustParse("10Gi")

	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tablet.GetStatefulSetName(),
			Namespace: tablet.Cluster().GetNamespace(),
			Labels:    selfLabels,
		},
		Spec: appsv1.StatefulSetSpec{
			//PodManagementPolicy: appsv1.PodManagementPolicyParallel{},
			PodManagementPolicy: appsv1.ParallelPodManagement,
			Replicas:            tablet.GetReplicas(),
			Selector: &metav1.LabelSelector{
				MatchLabels: selfLabels,
			},
			UpdateStrategy: appsv1.StatefulSetUpdateStrategy{
				Type: appsv1.RollingUpdateStatefulSetStrategyType,
			},
			ServiceName: tablet.Cluster().GetTabletServiceName(),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: selfLabels,
				},
				Spec: corev1.PodSpec{
					Affinity:       affinity,
					Containers:     containers,
					InitContainers: initContainers,
					Volumes: []corev1.Volume{
						{
							Name: "vt",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
					SecurityContext: &corev1.PodSecurityContext{
						FSGroup:   getInt64Ptr(2000),
						RunAsUser: getInt64Ptr(1000),
					},
					TerminationGracePeriodSeconds: getInt64Ptr(60000000),
				},
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "vtdataroot",
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
						Resources: corev1.ResourceRequirements{
							Requests: volumeRequests,
						},
					},
				},
			},
		},
	}, nil
}

func GetTabletMysqlContainers(tablet *vitessv1alpha2.VitessTablet) (containers []corev1.Container, initContainers []corev1.Container, err error) {
	mysql := tablet.GetMySQLContainer()
	if mysql == nil {
		return containers, initContainers, fmt.Errorf("No database container configuration found")
	}

	dbScripts := scripts.NewContainerScriptGenerator("mysql", tablet)
	if err := dbScripts.Generate(); err != nil {
		return containers, initContainers, fmt.Errorf("Error generating DB container scripts: %s", err)
	}

	initContainers = append(initContainers,
		corev1.Container{
			Name:            "init-mysql",
			Image:           "vitess/mysqlctld:helm-1.0.3", // TODO get this from a crd w/default
			ImagePullPolicy: corev1.PullIfNotPresent,
			Command:         []string{"bash"},
			Args: []string{
				"-c",
				dbScripts.Init,
			},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "vtdataroot",
					MountPath: "/vtdataroot",
				},
				{
					Name:      "vt",
					MountPath: "/vttmp",
				},
			},
		})

	containers = append(containers, corev1.Container{
		Name:            "mysql",
		Image:           mysql.Image,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Command:         []string{"bash"},
		Args: []string{
			"-c",
			dbScripts.Start,
		},
		Lifecycle: &corev1.Lifecycle{
			PreStop: &corev1.Handler{
				Exec: &corev1.ExecAction{
					Command: []string{
						"bash",
						"-c",
						dbScripts.PreStop,
					},
				},
			},
		},
		ReadinessProbe: &corev1.Probe{
			Handler: corev1.Handler{
				Exec: &corev1.ExecAction{
					Command: []string{
						"mysqladmin",
						"ping",
						"-uroot",
						"--socket=/vtdataroot/tabletdata/mysql.sock",
					},
				},
			},
			InitialDelaySeconds: 60,
			TimeoutSeconds:      10,
			PeriodSeconds:       10,
			SuccessThreshold:    1,
			FailureThreshold:    3,
		},
		Resources: mysql.Resources,
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "vtdataroot",
				MountPath: "/vtdataroot",
			},
			{
				Name:      "vt",
				MountPath: "/vt",
			},
		},
		Env: []corev1.EnvVar{
			{
				Name:  "VTROOT",
				Value: "/vt",
			},
			{
				Name:  "VTDATAROOT",
				Value: "/vtdataroot",
			},
			{
				Name:  "GOBIN",
				Value: "/vt/bin",
			},
			{
				Name:  "VT_MYSQL_ROOT",
				Value: "/usr",
			},
			{
				Name:  "PKG_CONFIG_PATH",
				Value: "/vt/lib",
			},
			{
				Name:  "VT_DB_FLAVOR",
				Value: mysql.DBFlavor,
			},
		},
	})

	return
}

func GetTabletVTTabletContainers(tablet *vitessv1alpha2.VitessTablet) (containers []corev1.Container, initContainers []corev1.Container, err error) {
	vttablet := tablet.GetVTTabletContainer()
	if vttablet == nil {
		err = fmt.Errorf("No database container configuration found")
		return
	}

	vtScripts := scripts.NewContainerScriptGenerator("vttablet", tablet)
	if err = vtScripts.Generate(); err != nil {
		err = fmt.Errorf("Error generating DB container scripts: %s", err)
		return
	}

	initContainers = append(initContainers,
		corev1.Container{
			Name:            "init-vttablet",
			Image:           "vitess/vtctl:helm-1.0.3", // TODO get this from a crd w/default
			ImagePullPolicy: corev1.PullIfNotPresent,
			Command:         []string{"bash"},
			Args: []string{
				"-c",
				vtScripts.Init,
			},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "vtdataroot",
					MountPath: "/vtdataroot",
				},
			},
		})

	containers = append(containers,
		corev1.Container{
			Name:            "vttablet",
			Image:           vttablet.Image,
			ImagePullPolicy: corev1.PullIfNotPresent,
			Command:         []string{"bash"},
			Args: []string{
				"-c",
				vtScripts.Start,
			},
			Lifecycle: &corev1.Lifecycle{
				PreStop: &corev1.Handler{
					Exec: &corev1.ExecAction{
						Command: []string{
							"bash",
							"-c",
							vtScripts.PreStop,
						},
					},
				},
			},
			ReadinessProbe: &corev1.Probe{
				Handler: corev1.Handler{
					HTTPGet: &corev1.HTTPGetAction{
						Path:   "/debug/health",
						Port:   intstr.FromInt(15002),
						Scheme: corev1.URISchemeHTTP,
					},
				},
				InitialDelaySeconds: 60,
				TimeoutSeconds:      10,
				PeriodSeconds:       10,
				SuccessThreshold:    1,
				FailureThreshold:    3,
			},
			LivenessProbe: &corev1.Probe{
				Handler: corev1.Handler{
					HTTPGet: &corev1.HTTPGetAction{
						Path:   "/debug/status",
						Port:   intstr.FromInt(15002),
						Scheme: corev1.URISchemeHTTP,
					},
				},
				InitialDelaySeconds: 60,
				TimeoutSeconds:      10,
				PeriodSeconds:       10,
				SuccessThreshold:    1,
				FailureThreshold:    3,
			},
			Ports: []corev1.ContainerPort{
				{
					ContainerPort: 15002,
					Name:          "web",
					Protocol:      corev1.ProtocolTCP,
				},
				{
					ContainerPort: 16002,
					Name:          "grpc",
					Protocol:      corev1.ProtocolTCP,
				},
			},
			Resources: corev1.ResourceRequirements{
				// Limits:   corev1.ResourceList{},
				// Requests: corev1.ResourceList{},
			},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "vtdataroot",
					MountPath: "/vtdataroot",
				},
			},
			Env: []corev1.EnvVar{
				{
					Name:  "VTROOT",
					Value: "/vt",
				},
				{
					Name:  "VTDATAROOT",
					Value: "/vtdataroot",
				},
				{
					Name:  "GOBIN",
					Value: "/vt/bin",
				},
				{
					Name:  "VT_MYSQL_ROOT",
					Value: "/usr",
				},
				{
					Name:  "PKG_CONFIG_PATH",
					Value: "/vt/lib",
				},
				{
					Name:  "VT_DB_FLAVOR",
					Value: vttablet.DBFlavor,
				},
			},
		},
		corev1.Container{
			Name:            "logrotate",
			Image:           "vitess/logrotate:helm-1.0.4", // TODO get this from a crd w/default
			ImagePullPolicy: corev1.PullIfNotPresent,
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "vtdataroot",
					MountPath: "/vtdataroot",
				},
			},
		})

	// add log containers with a slice of filename + containername slices
	for _, logtype := range [][]string{
		{"general", "general"},
		{"error", "error"},
		{"slow-query", "slow"},
	} {
		containers = append(containers, corev1.Container{
			Name:            logtype[1] + "-log",
			Image:           "vitess/logtail:helm-1.0.4", // TODO get this from a crd w/default
			ImagePullPolicy: corev1.PullIfNotPresent,
			Env: []corev1.EnvVar{
				{
					Name:  "TAIL_FILEPATH",
					Value: fmt.Sprintf("/vtdataroot/tabletdata/%s.log", logtype[0]),
				},
			},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "vtdataroot",
					MountPath: "/vtdataroot",
				},
			},
		})
	}

	return
}

func (r *ReconcileVitessCluster) ReconcileReplicaTabletInitJob(tablet *vitessv1alpha2.VitessTablet) (reconcile.Result, error) {
	job, jobErr := GetReplicaTabletInitMasterJob(tablet)
	if jobErr != nil {
		log.Error(jobErr, "failed to generate master elect job for replica VitessTablet", "VitessTablet.Namespace", tablet.GetNamespace(), "VitessTablet.Name", tablet.GetNamespace())
		return reconcile.Result{}, jobErr
	}

	found := &batchv1.Job{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: job.GetName(), Namespace: job.GetNamespace()}, found)
	if err != nil && errors.IsNotFound(err) {
		controllerutil.SetControllerReference(tablet.Cluster(), job, r.scheme)
		err = r.client.Create(context.TODO(), job)
		if err != nil {
			return reconcile.Result{}, err
		}
		// Job created successfully - return and requeue
		return reconcile.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "failed to get Job")
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func GetReplicaTabletInitMasterJob(tablet *vitessv1alpha2.VitessTablet) (*batchv1.Job, error) {
	jobName := tablet.GetScopedName("init-replica-master")

	scripts := scripts.NewContainerScriptGenerator("init_replica_master", tablet)
	if err := scripts.Generate(); err != nil {
		return nil, err
	}

	jobLabels := map[string]string{
		"app":                "vitess",
		"cluster":            tablet.Cluster().GetName(),
		"keyspace":           tablet.Keyspace().GetName(),
		"shard":              tablet.Shard().GetName(),
		"component":          "vttablet-replica-elector",
		"initShardMasterJob": "true",
		"job-name":           jobName,
	}

	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: tablet.Cluster().GetNamespace(),
			Labels:    jobLabels,
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: getInt32Ptr(1),
			Completions:  getInt32Ptr(1),
			Parallelism:  getInt32Ptr(1),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: jobLabels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "init-master",
							Image: "vitess/vtctlclient:helm-1.0.3", // TODO use CRD w/default
							Command: []string{
								"bash",
							},
							Args: []string{
								"-c",
								scripts.Start,
							},
						},
					},
					RestartPolicy: corev1.RestartPolicyOnFailure,
				},
			},
		},
	}, nil
}
