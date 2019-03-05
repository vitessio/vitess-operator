package v1alpha2

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.
// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file

// VitessClusterSpec defines the desired state of VitessCluster
type VitessClusterSpec struct {
	Lockserver *VitessLockserver `json:"lockserver,omitempty"`

	LockserverRef *corev1.LocalObjectReference `json:"lockserverRef,omitempty"`

	Cells []*VitessCell `json:"cells,omitempty"`

	CellSelector []ResourceSelector `json:"cellSelector,omitempty"`

	Keyspaces []*VitessKeyspace `json:"keyspaces,omitempty"`

	KeyspaceSelector []ResourceSelector `json:"keyspaceSelector,omitempty"`
}

// VitessClusterStatus defines the observed state of VitessCluster
type VitessClusterStatus struct {
	Phase ClusterPhase `json:"phase,omitempty"`

	Reason string `json:"reason,omitempty"`

	Message string `json:"reason,omitempty"`

	Conditions []VitessClusterCondition `json:"conditions,omitempty"`

	Lockserver *VitessLockserverStatus `json:"lockserver,omitempty"`
}

type ClusterPhase string

const (
	ClusterPhaseNone     ClusterPhase = ""
	ClusterPhaseCreating ClusterPhase = "Creating"
	ClusterPhaseReady    ClusterPhase = "Ready"
)

type VitessClusterCondition struct {
	// Type of cluster condition.
	Type ClusterConditionType `json:"type"`

	// Status of the condition, one of True, False, Unknown.
	Status corev1.ConditionStatus `json:"status"`

	// The last time this condition was updated.
	LastUpdateTime string `json:"lastUpdateTime,omitempty"`

	// Last time the condition transitioned from one status to another.
	LastTransitionTime string `json:"lastTransitionTime,omitempty"`

	// The reason for the condition's last transition.
	Reason string `json:"reason,omitempty"`

	// A human readable message indicating details about the transition.
	Message string `json:"message,omitempty"`
}

type ClusterConditionType string

const (
	VitessClusterConditionAvailable  ClusterConditionType = "Available"
	VitessClusterConditionRecovering ClusterConditionType = "Recovering"
	VitessClusterConditionScaling    ClusterConditionType = "Scaling"
	VitessClusterConditionUpgrading  ClusterConditionType = "Upgrading"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VitessCluster is the Schema for the vitessclusters API
// +k8s:openapi-gen=true
type VitessCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VitessClusterSpec   `json:"spec,omitempty"`
	Status VitessClusterStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VitessClusterList contains a list of VitessCluster
type VitessClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VitessCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VitessCluster{}, &VitessClusterList{})
}
