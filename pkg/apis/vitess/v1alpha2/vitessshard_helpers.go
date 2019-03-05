package v1alpha2

import (
	"strings"
)

func (shard *VitessShard) Cluster() *VitessCluster {
	return shard.Spec.parent.Cluster
}

func (shard *VitessShard) SetParentCluster(cluster *VitessCluster) {
	shard.Spec.parent.Cluster = cluster
}

func (shard *VitessShard) Keyspace() *VitessKeyspace {
	return shard.Spec.parent.Keyspace
}

func (shard *VitessShard) SetParentKeyspace(keyspace *VitessKeyspace) {
	shard.Spec.parent.Keyspace = keyspace
}

func (shard *VitessShard) Tablets() []*VitessTablet {
	return shard.Spec.Tablets
}

func (shard *VitessShard) EmbedTabletCopy(tablet *VitessTablet) {
	shard.Spec.Tablets = append(shard.Spec.Tablets, tablet.DeepCopy())
}

// GetTabletContainers satisfies ConfigProvider
func (shard *VitessShard) GetTabletContainers() *TabletContainers {
	if shard.Spec.Defaults != nil {
		return shard.Spec.Defaults.Containers
	}
	return nil
}

func (shard *VitessShard) GetScopedName(extra ...string) string {
	return strings.Join(append(
		[]string{
			shard.Keyspace().GetScopedName(),
		},
		extra...), "-")
}
