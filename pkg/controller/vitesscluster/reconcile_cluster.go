package vitesscluster

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	vitessv1alpha2 "vitess.io/vitess-operator/pkg/apis/vitess/v1alpha2"
	lockserver_controller "vitess.io/vitess-operator/pkg/controller/vitesslockserver"
)

// ReconcileClusterResources should only be called against a fully-populated and verified VitessCluster object
func (r *ReconcileVitessCluster) ReconcileClusterResources(cluster *vitessv1alpha2.VitessCluster) (reconcile.Result, error) {
	if r, err := r.ReconcileClusterLockserver(cluster); err != nil || r.Requeue {
		return r, err
	}

	if r, err := r.ReconcileClusterTabletService(cluster); err != nil || r.Requeue {
		return r, err
	}

	for _, cell := range cluster.Cells() {
		if r, err := r.ReconcileCell(cell); err != nil || r.Requeue {
			return r, err
		}
	}

	for _, keyspace := range cluster.Keyspaces() {
		if r, err := r.ReconcileKeyspace(keyspace); err != nil || r.Requeue {
			return r, err
		}
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileVitessCluster) ReconcileClusterLockserver(cluster *vitessv1alpha2.VitessCluster) (reconcile.Result, error) {
	log.Info("Reconciling Embedded Lockserver")

	// Build a complete VitessLockserver
	lockserver := cluster.Spec.Lockserver.DeepCopy()

	if cluster.Status.Lockserver != nil {
		// If status is not empty, deepcopy it into the tmp object
		cluster.Status.Lockserver.DeepCopyInto(&lockserver.Status)
	}

	// Run it through the controller's reconcile func
	recResult, recErr := lockserver_controller.ReconcileObject(lockserver, log)

	// Split and store the spec and status in the parent VitessCluster
	cluster.Spec.Lockserver = lockserver.DeepCopy()
	cluster.Status.Lockserver = lockserver.Status.DeepCopy()

	// Using the  split client here breaks the cluster normalization
	// TODO Fix and re-enable

	// if err := r.client.Status().Update(context.TODO(), cluster); err != nil {
	// 	log.Error(err, "Failed to update VitessCluster status after lockserver change.")
	// 	return reconcile.Result{}, err
	// }

	return recResult, recErr
}

func (r *ReconcileVitessCluster) ReconcileClusterTabletService(cluster *vitessv1alpha2.VitessCluster) (reconcile.Result, error) {
	service, serviceErr := getServiceForClusterTablets(cluster)
	if serviceErr != nil {
		log.Error(serviceErr, "failed to generate service for VitessCluster tablets", "VitessCluster.Namespace", cluster.GetNamespace(), "VitessCluster.Name", cluster.GetNamespace())
		return reconcile.Result{}, serviceErr
	}
	foundService := &corev1.Service{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: service.GetName(), Namespace: service.GetNamespace()}, foundService)
	if err != nil && errors.IsNotFound(err) {
		controllerutil.SetControllerReference(cluster, service, r.scheme)
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

// getServiceForClusterTablets takes a vitess cluster and returns a headless service that will point to all of the cluster's tablets
func getServiceForClusterTablets(cluster *vitessv1alpha2.VitessCluster) (*corev1.Service, error) {
	labels := map[string]string{
		"app":       "vitess",
		"cluster":   cluster.GetName(),
		"component": "vttablet",
	}

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.GetTabletServiceName(),
			Namespace: cluster.GetNamespace(),
			Labels:    labels,
			Annotations: map[string]string{
				"service.alpha.kubernetes.io/tolerate-unready-endpoints": "true",
			},
		},
		Spec: corev1.ServiceSpec{
			ClusterIP:                corev1.ClusterIPNone,
			Selector:                 labels,
			Type:                     corev1.ServiceTypeClusterIP,
			PublishNotReadyAddresses: true,
			Ports: []corev1.ServicePort{
				{
					Name: "web",
					Port: 15002,
				},
				{
					Name: "grpc",
					Port: 16002,
				},
				// TODO: Configure ports below only if if ppm is enabled
				{
					Name: "query-data",
					Port: 42001,
				},
				{
					Name: "mysql-metrics",
					Port: 42002,
				},
			},
		},
	}

	// The error return is always nil right now, but it still returns one just
	// in case there are error states in the future
	return service, nil
}
