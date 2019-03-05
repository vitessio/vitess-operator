package v1alpha2

import (
	"strings"
)

func (cluster *VitessCluster) Cells() []*VitessCell {
	return cluster.Spec.Cells
}

func (cluster *VitessCluster) EmbedCellCopy(cell *VitessCell) {
	cluster.Spec.Cells = append(cluster.Spec.Cells, cell.DeepCopy())
}

func (cluster *VitessCluster) Keyspaces() []*VitessKeyspace {
	return cluster.Spec.Keyspaces
}

func (cluster *VitessCluster) EmbedKeyspaceCopy(keyspace *VitessKeyspace) {
	cluster.Spec.Keyspaces = append(cluster.Spec.Keyspaces, keyspace.DeepCopy())
}

func (cluster *VitessCluster) Shards() []*VitessShard {
	var shards []*VitessShard
	for _, keyspace := range cluster.Keyspaces() {
		shards = append(shards, keyspace.Shards()...)
	}
	return shards
}

func (cluster *VitessCluster) Tablets() []*VitessTablet {
	var tablets []*VitessTablet
	for _, shard := range cluster.Shards() {
		tablets = append(tablets, shard.Tablets()...)
	}
	return tablets
}

func (cluster *VitessCluster) Lockserver() *VitessLockserver {
	return cluster.Spec.Lockserver
}

func (cluster *VitessCluster) GetCellByID(cellID string) *VitessCell {
	for _, cell := range cluster.Cells() {
		if cell.GetName() == cellID {
			return cell
		}
	}

	return nil
}

func (cluster *VitessCluster) GetScopedName(extra ...string) string {
	return strings.Join(append(
		[]string{
			cluster.GetName(),
		},
		extra...), "-")
}

func (cluster *VitessCluster) GetTabletServiceName() string {
	return cluster.GetScopedName("tab")
}

func (cluster *VitessCluster) Phase() ClusterPhase {
	return cluster.Status.Phase
}

func (cluster *VitessCluster) SetPhase(p ClusterPhase) {
	cluster.Status.Phase = p
}

func (cluster *VitessCluster) InPhase(p ClusterPhase) bool {
	return cluster.Status.Phase == p
}

func (cluster *VitessCluster) AllTabletsReady() bool {
	for _, tablet := range cluster.Tablets() {
		if !tablet.InPhase(TabletPhaseReady) {
			return false
		}
	}
	return true
}
