package vitesscluster

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"

	vitessv1alpha2 "vitess.io/vitess-operator/pkg/apis/vitess/v1alpha2"
	"vitess.io/vitess-operator/pkg/normalizer"
)

var log = logf.Log.WithName("controller_vitesscluster")

// Add creates a new VitessCluster Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileVitessCluster{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("vitesscluster-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource VitessCluster
	err = c.Watch(&source.Kind{Type: &vitessv1alpha2.VitessCluster{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	for _, childType := range []runtime.Object{
		&vitessv1alpha2.VitessLockserver{},
		&vitessv1alpha2.VitessCell{},
		&vitessv1alpha2.VitessKeyspace{},
		&vitessv1alpha2.VitessShard{},
		&vitessv1alpha2.VitessTablet{},
	} {
		// Watch for changes to child type and requeue the owner VitessCluster
		err = c.Watch(&source.Kind{Type: childType}, &handler.EnqueueRequestForOwner{
			IsController: true,
			OwnerType:    &vitessv1alpha2.VitessCluster{},
		})
		if err != nil {
			return err
		}
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileVitessCluster{}

// ReconcileVitessCluster reconciles a VitessCluster object
type ReconcileVitessCluster struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a VitessCluster object and makes changes based on the state read
// and what is in the VitessCluster.Spec
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileVitessCluster) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling VitessCluster")

	// Fetch the VitessCluster instance
	cluster := &vitessv1alpha2.VitessCluster{}
	err := r.client.Get(context.TODO(), request.NamespacedName, cluster)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	n := normalizer.New(r.client)

	// Check
	if err := n.TestClusterSanity(cluster); err != nil {
		reqLogger.Error(err, "Cluster failed sanity test")
		return reconcile.Result{Requeue: false}, err
	}

	// Normalize
	if err := n.NormalizeCluster(cluster); err != nil {
		return reconcile.Result{Requeue: false}, err
	}

	// Validate
	if err := n.ValidateCluster(cluster); err != nil {
		reqLogger.Error(err, "Cluster failed validation")
		return reconcile.Result{Requeue: false}, err
	}

	// Reconcile
	if result, err := r.ReconcileClusterResources(cluster); err != nil {
		reqLogger.Info("Error reconciling cluster member resources")
		return result, err
	} else if result.Requeue {
		reqLogger.Info("Requeue after reconciling cluster member resources")
		return result, nil
	}

	// Status updates

	switch cluster.Phase() {
	// Set cluster status to Created if it's a new cluster
	case vitessv1alpha2.ClusterPhaseNone:
		r.SetClusterPhase(cluster, vitessv1alpha2.ClusterPhaseCreating)
	// Set a creating cluster status to Ready if all tablet sets are ready
	case vitessv1alpha2.ClusterPhaseCreating:
		if cluster.AllTabletsReady() {
			r.SetClusterPhase(cluster, vitessv1alpha2.ClusterPhaseReady)
		} else {
			// Requeue to re-check for readiness later
			reqLogger.Info("Cluster created but not ready. Will try again later.")
			return reconcile.Result{Requeue: true, RequeueAfter: 30 * time.Second}, nil
		}
	}

	// Nothing to do - don't reqeue
	reqLogger.Info("Skip reconcile: all managed services in sync")
	return reconcile.Result{}, nil
}

func (r *ReconcileVitessCluster) SetClusterPhase(cluster *vitessv1alpha2.VitessCluster, p vitessv1alpha2.ClusterPhase) error {
	log.Info(fmt.Sprintf("Setting VitessCluster to %s phase", p))

	// Get latest cluster
	foundCluster := &vitessv1alpha2.VitessCluster{}
	if err := r.client.Get(context.TODO(), types.NamespacedName{Name: cluster.GetName(), Namespace: cluster.GetNamespace()}, foundCluster); err != nil {
		return err
	}

	// set phase
	foundCluster.SetPhase(p)

	// update
	if err := r.client.Status().Update(context.TODO(), foundCluster); err != nil {
		log.Error(err, "Failed to update VitessCluster phase")
		return err
	}

	return nil
}
