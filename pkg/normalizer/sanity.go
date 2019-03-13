package normalizer

import (
	"fmt"

	vitessv1alpha2 "vitess.io/vitess-operator/pkg/apis/vitess/v1alpha2"
)

func (n *Normalizer) TestClusterSanity(cluster *vitessv1alpha2.VitessCluster) error {
	// Lockserver and LockserverRef are mutuallly exclusive
	if cluster.Spec.Lockserver != nil && cluster.Spec.LockserverRef != nil {
		return fmt.Errorf("Cannot specify both a lockserver and lockserverRef")
	}

	return nil
}
