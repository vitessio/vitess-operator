package v1alpha2

import (
	"strings"
)

func (keyspace *VitessKeyspace) SetParentCluster(cluster *VitessCluster) {
	keyspace.Spec.parent.Cluster = cluster
}

func (keyspace *VitessKeyspace) Cluster() *VitessCluster {
	return keyspace.Spec.parent.Cluster
}

// GetTabletContainers satisfies ConfigProvider
func (keyspace *VitessKeyspace) GetTabletContainers() *TabletContainers {
	if keyspace.Spec.Defaults != nil {
		return keyspace.Spec.Defaults.Containers
	}
	return nil
}

func (keyspace *VitessKeyspace) Shards() []*VitessShard {
	return keyspace.Spec.Shards
}

func (keyspace *VitessKeyspace) EmbedShardCopy(shard *VitessShard) {
	keyspace.Spec.Shards = append(keyspace.Spec.Shards, shard.DeepCopy())
}

func (keyspace *VitessKeyspace) GetScopedName(extra ...string) string {
	return strings.Join(append(
		[]string{
			keyspace.Cluster().GetScopedName(),
		},
		extra...), "-")
}
