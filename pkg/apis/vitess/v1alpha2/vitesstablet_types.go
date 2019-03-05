package v1alpha2

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.
// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file

// VitessTabletSpec defines the desired state of VitessTablet
type VitessTabletSpec struct {
	TabletID int64 `json:"tabletID"`

	Replicas *int32 `json:"replicas"`

	CellID string `json:"cellID"`

	Type TabletType `json:"type"`

	Datastore TabletDatastore `json:"datastore"`

	Containers *TabletContainers `json:"containers"`

	VolumeClaim *corev1.PersistentVolumeClaimVolumeSource `json:"volumeclaim, omitempty"`

	Credentials *TabletCredentials `json:"credentials,omitempty"`

	// parent is unexported on purpose.
	// It should only be used during processing and never stored
	parent VitessTabletParents
}

type VitessTabletParents struct {
	Cluster  *VitessCluster
	Cell     *VitessCell
	Keyspace *VitessKeyspace
	Shard    *VitessShard
}

type TabletType string

const (
	TabletTypeMaster   TabletType = "master"
	TabletTypeReplica  TabletType = "replica"
	TabletTypeReadOnly TabletType = "readonly"
	TabletTypeBackup   TabletType = "backup"
	TabletTypeRestore  TabletType = "restore"
	TabletTypeDrained  TabletType = "drained"
)

const TabletTypeDefault TabletType = TabletTypeReplica

type TabletDatastore struct {
	Type TabletDatastoreType `json:"type"`
}

type TabletDatastoreType string

const (
	TabletDatastoreTypeLocal TabletDatastoreType = "local"
)

const TabletDatastoreTypeDefault TabletDatastoreType = TabletDatastoreTypeLocal

type TabletCredentials struct {
	// SecretRef points a Secret resource which contains the credentials
	// +optional
	SecretRef *corev1.SecretReference `json:"secretRef,omitempty" protobuf:"bytes,4,opt,name=secretRef"`
}

// status is for internal use only. If it was exported then it would dirty-up the
// tablet objects embedded in other resources and would result in mixed status and spec data
// it is here for use by the VitessCluster object and its controller
type VitessTabletStatus struct {
	Phase TabletPhase `json:"-"`
}

type TabletPhase string

const (
	TabletPhaseNone  TabletPhase = ""
	TabletPhaseReady TabletPhase = "Ready"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VitessTablet is the Schema for the vitesstablets API
// +k8s:openapi-gen=true
type VitessTablet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec VitessTabletSpec `json:"spec,omitempty"`

	// internal use only. See struct def for details
	status VitessTabletStatus `json:"-"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VitessTabletList contains a list of VitessTablet
type VitessTabletList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VitessTablet `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VitessTablet{}, &VitessTabletList{})
}
