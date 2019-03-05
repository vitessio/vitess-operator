package normalizer

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"

	vitessv1alpha2 "vitess.io/vitess-operator/pkg/apis/vitess/v1alpha2"
)

var log = logf.Log.WithName("normalizer")

type Normalizer struct {
	client client.Client
}

func New(client client.Client) *Normalizer {
	return &Normalizer{
		client: client,
	}
}

func (n *Normalizer) NormalizeCluster(cluster *vitessv1alpha2.VitessCluster) error {
	if err := n.NormalizeClusterLockserver(cluster); err != nil {
		return err
	}

	if err := n.NormalizeClusterCells(cluster); err != nil {
		return err
	}

	if err := n.NormalizeClusterKeyspaces(cluster); err != nil {
		return err
	}

	// if err := n.NormalizeClusterTabletParentage(cluster); err != nil {
	// 	return err
	// }

	return nil
}

func (n *Normalizer) NormalizeClusterLockserver(cluster *vitessv1alpha2.VitessCluster) error {
	// Populate the embedded lockserver spec from Ref if given
	if cluster.Spec.LockserverRef != nil {
		ls := &vitessv1alpha2.VitessLockserver{}
		err := n.client.Get(context.TODO(), types.NamespacedName{Name: cluster.Spec.LockserverRef.Name, Namespace: cluster.GetNamespace()}, ls)
		if err != nil {
			return NewClientError(err)
		}

		// Since Lockserver and Lockserver Ref are mutually-exclusive, it should be safe
		// to simply populate the Lockserver struct member with a pointer to the fetched lockserver
		cluster.Spec.Lockserver = ls
	}

	return nil
}

func (n *Normalizer) NormalizeClusterCells(cluster *vitessv1alpha2.VitessCluster) error {
	if len(cluster.Spec.CellSelector) != 0 {
		cellList := &vitessv1alpha2.VitessCellList{}
		if err := n.ListFromSelectors(context.TODO(), cluster.Spec.CellSelector, cellList); err != nil {
			return fmt.Errorf("Error getting cells for cluster %s", err)
		}

		log.Info(fmt.Sprintf("VitessCluster's cellSelector matched %d cells", len(cellList.Items)))
		for _, cell := range cellList.Items {
			cluster.EmbedCellCopy(&cell)
		}
	}

	for _, cell := range cluster.Cells() {
		cell.SetParentCluster(cluster)

		n.NormalizeCellLockserver(cell)
	}

	return nil
}

func (n *Normalizer) NormalizeCellLockserver(cell *vitessv1alpha2.VitessCell) error {
	// Populate the embedded lockserver spec from Ref if given
	if cell.Spec.LockserverRef != nil {
		ls := &vitessv1alpha2.VitessLockserver{}
		err := n.client.Get(context.TODO(), types.NamespacedName{Name: cell.Spec.LockserverRef.Name, Namespace: cell.Cluster().GetNamespace()}, ls)
		if err != nil {
			return NewClientError(err)
		}

		// Since Lockserver and Lockserver Ref are mutually-exclusive, it should be safe
		// to simply populate the Lockserver struct member with a pointer to the fetched lockserver
		cell.Spec.Lockserver = ls
	}

	return nil
}

func (n *Normalizer) NormalizeClusterKeyspaces(cluster *vitessv1alpha2.VitessCluster) error {
	if len(cluster.Spec.KeyspaceSelector) != 0 {
		keyspaceList := &vitessv1alpha2.VitessKeyspaceList{}
		if err := n.ListFromSelectors(context.TODO(), cluster.Spec.KeyspaceSelector, keyspaceList); err != nil {
			return fmt.Errorf("Error getting keyspaces for cluster %s", err)
		}

		log.Info(fmt.Sprintf("VitessCluster's keyspaceSelector matched %d keyspaces", len(keyspaceList.Items)))
		for _, keyspace := range keyspaceList.Items {
			cluster.EmbedKeyspaceCopy(&keyspace)
		}
	}

	for _, keyspace := range cluster.Keyspaces() {
		keyspace.SetParentCluster(cluster)

		if err := n.NormalizeClusterKeyspaceShards(cluster, keyspace); err != nil {
			return err
		}
	}

	return nil
}

func (n *Normalizer) NormalizeClusterKeyspaceShards(cluster *vitessv1alpha2.VitessCluster, keyspace *vitessv1alpha2.VitessKeyspace) error {
	shardList := &vitessv1alpha2.VitessShardList{}
	err := n.ListFromSelectors(context.TODO(), keyspace.Spec.ShardSelector, shardList)
	if err != nil {
		return fmt.Errorf("Error getting shards for keyspace %s", err)
	}

	log.Info(fmt.Sprintf("VitessKeyspace's shardSelector matched %d shards", len(shardList.Items)))
	for _, shard := range shardList.Items {
		keyspace.EmbedShardCopy(&shard)
	}

	for _, shard := range keyspace.Shards() {
		shard.SetParentCluster(keyspace.Cluster())
		shard.SetParentKeyspace(keyspace)

		if err := n.NormalizeClusterShardTablets(cluster, shard); err != nil {
			return err
		}
	}

	return nil
}

func (n *Normalizer) NormalizeClusterShardTablets(cluster *vitessv1alpha2.VitessCluster, shard *vitessv1alpha2.VitessShard) error {
	tabletList := &vitessv1alpha2.VitessTabletList{}
	err := n.ListFromSelectors(context.TODO(), shard.Spec.TabletSelector, tabletList)
	if err != nil {
		return fmt.Errorf("Error getting tablets for shard %s", err)
	}

	log.Info(fmt.Sprintf("VitessShard's tabletSelector matched %d tablets", len(tabletList.Items)))
	for _, tablet := range tabletList.Items {
		shard.EmbedTabletCopy(&tablet)
	}

	for _, tablet := range shard.Tablets() {
		tablet.SetParentCluster(cluster)
		tablet.SetParentCell(cluster.GetCellByID(tablet.Spec.CellID))
		tablet.SetParentKeyspace(shard.Keyspace())
		tablet.SetParentShard(shard)
	}

	return nil
}

func (n *Normalizer) ListFromSelectors(ctx context.Context, rSels []vitessv1alpha2.ResourceSelector, retList runtime.Object) error {
	labelSelector, err := ResourceSelectorsAsLabelSelector(rSels)
	if err == nil {
		err := n.client.List(ctx, &client.ListOptions{LabelSelector: labelSelector}, retList)
		if err != nil {
			return err
		}
		return nil
	}
	return err
}

// ResourceSelectorsAsLabelSelector converts the []ResourceSelector api type into a struct that implements
// labels.Selector.
func ResourceSelectorsAsLabelSelector(rSels []vitessv1alpha2.ResourceSelector) (labels.Selector, error) {
	if len(rSels) == 0 {
		return labels.Nothing(), nil
	}

	selector := labels.NewSelector()
	for _, expr := range rSels {
		var op selection.Operator
		switch expr.Operator {
		case vitessv1alpha2.ResourceSelectorOpIn:
			op = selection.In
		case vitessv1alpha2.ResourceSelectorOpNotIn:
			op = selection.NotIn
		case vitessv1alpha2.ResourceSelectorOpExists:
			op = selection.Exists
		case vitessv1alpha2.ResourceSelectorOpDoesNotExist:
			op = selection.DoesNotExist
		default:
			return nil, fmt.Errorf("%q is not a valid resource selector operator", expr.Operator)
		}
		r, err := labels.NewRequirement(expr.Key, op, expr.Values)
		if err != nil {
			return nil, err
		}
		selector = selector.Add(*r)
	}
	return selector, nil
}
