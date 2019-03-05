package v1alpha2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.
// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file

// VitessKeyspaceSpec defines the desired state of VitessKeyspace
type VitessKeyspaceSpec struct {
	Defaults *VitessShardOptions `json:"defaults"`

	Shards []*VitessShard `json:"shards"`

	ShardSelector []ResourceSelector `json:"shardSelector,omitempty"`

	// parent is unexported on purpose.
	// It should only be used during processing and never stored
	parent VitessKeyspaceParents
}

type VitessKeyspaceParents struct {
	Cluster *VitessCluster
}

type VitessBatchOptions struct {
	Count int64 `json:"count"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VitessKeyspace is the Schema for the vitesskeyspaces API
// +k8s:openapi-gen=true
type VitessKeyspace struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec VitessKeyspaceSpec `json:"spec,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VitessKeyspaceList contains a list of VitessKeyspace
type VitessKeyspaceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VitessKeyspace `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VitessKeyspace{}, &VitessKeyspaceList{})
}
