package normalizer

import (
	"errors"
	"fmt"
)

type ValidationError error

var (
	ValidationErrorNoLockserverForCluster ValidationError = errors.New("No Lockserver in Cluster")
	ValidationErrorNoLockserverForCell    ValidationError = errors.New("No Lockserver in Cell")

	ValidationErrorNoCells     ValidationError = errors.New("No Cells in Cluster")
	ValidationErrorNoShards    ValidationError = errors.New("No Shards in Cluster")
	ValidationErrorNoTablets   ValidationError = errors.New("No Tablets in Cluster")
	ValidationErrorNoKeyspaces ValidationError = errors.New("No Keyspaces in Cluster")

	ValidationErrorOverlappingKeyrange ValidationError = errors.New("Multiple shards provided with the same keyrange")

	ValidationErrorNoCellForTablet   ValidationError = errors.New("No Cell for Tablet")
	ValidationErrorTabletNameTooLong ValidationError = errors.New("Tablet name is too long and would break mysql replication")
)

var ClientError = errors.New("Client Error")

func NewClientError(err error) error {
	return fmt.Errorf("Client Error: %s", err)
}
