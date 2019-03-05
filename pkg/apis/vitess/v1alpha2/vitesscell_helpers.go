package v1alpha2

import (
	"strings"
)

func (cell *VitessCell) SetParentCluster(cluster *VitessCluster) {
	cell.Spec.parent.Cluster = cluster
}

func (cell *VitessCell) Cluster() *VitessCluster {
	return cell.Spec.parent.Cluster
}

func (cell *VitessCell) Lockserver() *VitessLockserver {
	return cell.Spec.Lockserver
}

func (cell *VitessCell) GetScopedName(extra ...string) string {
	return strings.Join(append(
		[]string{
			cell.Cluster().GetName(),
			cell.GetName(),
		},
		extra...), "-")
}
