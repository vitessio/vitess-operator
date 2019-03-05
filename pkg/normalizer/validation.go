package normalizer

import (
	"strings"

	vitessv1alpha2 "vitess.io/vitess-operator/pkg/apis/vitess/v1alpha2"
)

const (
	// If a tablet's hostname ever goes over 60 chars then it will not be able
	// to boostrap properly because it will truncate the master hostname and replication will fail.
	// See the chart at the bottom of https://dev.mysql.com/doc/refman/8.0/en/change-master-to.html
	MaxTabletHostnameLength = 60

	// allow up to 99 replicas. Statefulsets can go higher, but it's not likely for this use case
	MaxTabletOrdinalLength = 2
)

func (n *Normalizer) ValidateCluster(cluster *vitessv1alpha2.VitessCluster) error {
	if cluster.Lockserver() == nil {
		return ValidationErrorNoLockserverForCluster
	}

	if len(cluster.Cells()) == 0 {
		return ValidationErrorNoCells
	}

	for _, cell := range cluster.Cells() {
		if cell.Lockserver() == nil {
			return ValidationErrorNoLockserverForCell
		}
	}

	if len(cluster.Keyspaces()) == 0 {
		return ValidationErrorNoKeyspaces
	}

	if len(cluster.Shards()) == 0 {
		return ValidationErrorNoShards
	}

	// check for overlapping keyranges
	for _, shard := range cluster.Shards() {
		// store matched keyranges
		keyranges := make(map[string]struct{})

		// if keyrange string is already in the map then it is a duplicate
		if _, ok := keyranges[shard.Spec.KeyRange.String()]; ok {
			return ValidationErrorOverlappingKeyrange
		}

		// set keyrange string as existing
		keyranges[shard.Spec.KeyRange.String()] = struct{}{}
	}

	if len(cluster.Tablets()) == 0 {
		return ValidationErrorNoTablets
	}

	for _, tablet := range cluster.Tablets() {
		if tablet.Cell() == nil {
			return ValidationErrorNoCellForTablet
		}
	}

	return nil
}

func (n *Normalizer) ValidateTablet(tablet *vitessv1alpha2.VitessTablet) error {
	if getMaxExpectedTabletHostLength(tablet) >= MaxTabletHostnameLength {
		return ValidationErrorTabletNameTooLong
	}

	return nil
}

// getMaxExpectedTabletHostLength returns the maximum possible hostname of
// this tablet given the max oridinal length allowed
func getMaxExpectedTabletHostLength(tablet *vitessv1alpha2.VitessTablet) int {
	return len(strings.Join([]string{
		tablet.GetStatefulSetName(),
		"-",
		strings.Repeat("9", MaxTabletOrdinalLength),
		".",
		tablet.Cluster().GetTabletServiceName(),
	}, ""))
}
