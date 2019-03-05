package v1alpha2

import (
	// "fmt"
	"strconv"
	"strings"
)

// GetTabletContainers satisfies ConfigProvider
func (tablet *VitessTablet) GetTabletContainers() *TabletContainers {
	return tablet.Spec.Containers
}

func (tablet *VitessTablet) SetParentCluster(cluster *VitessCluster) {
	tablet.Spec.parent.Cluster = cluster
}

func (tablet *VitessTablet) SetParentCell(cell *VitessCell) {
	tablet.Spec.parent.Cell = cell
}

func (tablet *VitessTablet) SetParentKeyspace(keyspace *VitessKeyspace) {
	tablet.Spec.parent.Keyspace = keyspace
}

func (tablet *VitessTablet) SetParentShard(shard *VitessShard) {
	tablet.Spec.parent.Shard = shard
}

func (tablet *VitessTablet) Lockserver() *VitessLockserver {
	return tablet.Cell().Lockserver()
}

func (tablet *VitessTablet) Cluster() *VitessCluster {
	return tablet.Spec.parent.Cluster
}

func (tablet *VitessTablet) Cell() *VitessCell {
	return tablet.Spec.parent.Cell
}

func (tablet *VitessTablet) Keyspace() *VitessKeyspace {
	return tablet.Spec.parent.Keyspace
}

func (tablet *VitessTablet) Shard() *VitessShard {
	return tablet.Spec.parent.Shard
}

func (tablet *VitessTablet) GetStatefulSetName() string {
	return tablet.GetScopedName(string(tablet.Spec.Type))
}

func (tablet *VitessTablet) GetScopedName(extra ...string) string {
	return strings.Join(append(
		[]string{
			tablet.Cluster().GetName(),
			tablet.Cell().GetName(),
			tablet.Keyspace().GetName(),
			tablet.Shard().GetName(),
		},
		extra...), "-")
}

func (tablet *VitessTablet) GetReplicas() *int32 {
	if tablet.Spec.Replicas != nil {
		return tablet.Spec.Replicas
	}

	if tablet.Shard().Spec.Defaults != nil && tablet.Shard().Spec.Defaults.Replicas != nil {
		return tablet.Shard().Spec.Defaults.Replicas
	}

	var def int32
	return &def
}

func (tablet *VitessTablet) GetMySQLContainer() *MySQLContainer {
	// Inheritance order, with most specific first
	providers := []ConfigProvider{
		tablet,
		tablet.Spec.parent.Shard,
		tablet.Spec.parent.Keyspace,
	}

	for _, p := range providers {
		if containers := p.GetTabletContainers(); containers != nil && containers.MySQL != nil {
			// TODO get defaults from full range of providers
			if containers.MySQL.DBFlavor == "" && containers.DBFlavor != "" {
				containers.MySQL.DBFlavor = containers.DBFlavor
			}
			if containers.MySQL.DBFlavor == "" {
				containers.MySQL.DBFlavor = "mysql56"
			}
			return containers.MySQL
		}
	}
	return nil
}

func (tablet *VitessTablet) GetVTTabletContainer() *VTTabletContainer {
	// Inheritance order, with most specific first
	providers := []ConfigProvider{
		tablet,
		tablet.Shard(),
		tablet.Keyspace(),
	}

	for _, p := range providers {
		if containers := p.GetTabletContainers(); containers != nil && containers.VTTablet != nil {
			// TODO get defaults from full range of providers
			if containers.VTTablet.DBFlavor == "" && containers.DBFlavor != "" {
				containers.VTTablet.DBFlavor = containers.DBFlavor
			}
			if containers.VTTablet.DBFlavor == "" {
				containers.VTTablet.DBFlavor = "mysql56"
			}
			return containers.VTTablet
		}
	}
	return nil
}

func (tablet *VitessTablet) GetTabletID() string {
	return strconv.FormatInt(tablet.Spec.TabletID, 10)
}

func (tablet *VitessTablet) Phase() TabletPhase {
	return tablet.status.Phase
}

func (tablet *VitessTablet) SetPhase(p TabletPhase) {
	tablet.status.Phase = p
}

func (tablet *VitessTablet) InPhase(p TabletPhase) bool {
	return tablet.status.Phase == p
}
