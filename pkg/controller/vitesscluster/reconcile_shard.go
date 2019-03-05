package vitesscluster

import (
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	vitessv1alpha2 "vitess.io/vitess-operator/pkg/apis/vitess/v1alpha2"
)

func (r *ReconcileVitessCluster) ReconcileShard(shard *vitessv1alpha2.VitessShard) (reconcile.Result, error) {
	log.Info("Reconciling Shard", "Namespace", shard.GetNamespace(), "VitessCluster.Name", shard.Cluster().GetName(), "Shard.Name", shard.GetName())

	// Reconcile all shard tablets
	for _, tablet := range shard.Tablets() {
		if result, err := r.ReconcileTablet(tablet); err != nil {
			return result, err
		}
	}

	return reconcile.Result{}, nil
}
