package normalizer

import (
	"context"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	// "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	// "sigs.k8s.io/controller-runtime/pkg/reconcile"
	// logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"

	vitessv1alpha2 "vitess.io/vitess-operator/pkg/apis/vitess/v1alpha2"
)

var (
	testNamespace   = "vitess"
	testClusterName = "vitess-operator"

	// simple labels for all resources
	testLabels = map[string]string{
		"app": "yes",
	}

	// simple selector for all resources
	testSel = []vitessv1alpha2.ResourceSelector{
		{
			Key:      "app",
			Operator: vitessv1alpha2.ResourceSelectorOpIn,
			Values:   []string{"yes"},
		},
	}
)

func TestSanity(t *testing.T) {
	// Set the logger to development mode for verbose logs.
	// logf.SetLogger(logf.ZapLogger(true))

	// Define a minimal cluster which matches one of the cells above
	cluster := &vitessv1alpha2.VitessCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testClusterName,
			Namespace: testNamespace,
		},
		Spec: vitessv1alpha2.VitessClusterSpec{
			LockserverRef: &corev1.LocalObjectReference{
				Name: "lockserver",
			},
		},
	}

	// Populate the client with initial data
	objs := []runtime.Object{
		cluster,
		&vitessv1alpha2.VitessLockserver{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "lockserver",
				Namespace: testNamespace,
			},
		},
	}

	// Register operator types with the runtime scheme.
	s := scheme.Scheme
	s.AddKnownTypes(vitessv1alpha2.SchemeGroupVersion, &vitessv1alpha2.VitessCluster{})
	s.AddKnownTypes(vitessv1alpha2.SchemeGroupVersion, &vitessv1alpha2.VitessClusterList{})
	s.AddKnownTypes(vitessv1alpha2.SchemeGroupVersion, &vitessv1alpha2.VitessCell{})
	s.AddKnownTypes(vitessv1alpha2.SchemeGroupVersion, &vitessv1alpha2.VitessCellList{})
	s.AddKnownTypes(vitessv1alpha2.SchemeGroupVersion, &vitessv1alpha2.VitessTablet{})
	s.AddKnownTypes(vitessv1alpha2.SchemeGroupVersion, &vitessv1alpha2.VitessTabletList{})
	s.AddKnownTypes(vitessv1alpha2.SchemeGroupVersion, &vitessv1alpha2.VitessShard{})
	s.AddKnownTypes(vitessv1alpha2.SchemeGroupVersion, &vitessv1alpha2.VitessShardList{})
	s.AddKnownTypes(vitessv1alpha2.SchemeGroupVersion, &vitessv1alpha2.VitessKeyspace{})
	s.AddKnownTypes(vitessv1alpha2.SchemeGroupVersion, &vitessv1alpha2.VitessKeyspaceList{})
	s.AddKnownTypes(vitessv1alpha2.SchemeGroupVersion, &vitessv1alpha2.VitessLockserver{})
	s.AddKnownTypes(vitessv1alpha2.SchemeGroupVersion, &vitessv1alpha2.VitessLockserverList{})

	// Create a fake client to mock API calls.
	client := fake.NewFakeClient(objs...)

	n := New(client)

	// Call the normalize function for the cluster
	if err := n.NormalizeCluster(cluster); err != nil {
		t.Fatalf("Error normalizing cluster: %s", err)
	}

	// Ensure that all matched objects were embedded properly
	if err := n.TestClusterSanity(cluster); err == nil {
		t.Fatalf("Cluster passed sanity test and shouldn't have")
	}
}

func TestValidation(t *testing.T) {
	// Set the logger to development mode for verbose logs.
	// logf.SetLogger(logf.ZapLogger(true))

	tests := []struct {
		obj        runtime.Object
		missingErr ValidationError
	}{
		{
			&vitessv1alpha2.VitessLockserver{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cluster-lockserver",
					Namespace: testNamespace,
					Labels:    testLabels,
				},
			},
			ValidationErrorNoLockserverForCluster,
		},
		{
			&vitessv1alpha2.VitessCell{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cell",
					Namespace: testNamespace,
					Labels:    testLabels,
				},
				Spec: vitessv1alpha2.VitessCellSpec{
					LockserverRef: &corev1.LocalObjectReference{
						Name: "cell-lockserver",
					},
				},
			},
			ValidationErrorNoCells,
		},
		{
			&vitessv1alpha2.VitessLockserver{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cell-lockserver",
					Namespace: testNamespace,
					Labels:    testLabels,
				},
			},
			ValidationErrorNoLockserverForCell,
		},
		{
			&vitessv1alpha2.VitessKeyspace{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "keyspace",
					Namespace: testNamespace,
					Labels:    testLabels,
				},
				Spec: vitessv1alpha2.VitessKeyspaceSpec{
					ShardSelector: testSel,
				},
			},
			ValidationErrorNoKeyspaces,
		},
		{
			&vitessv1alpha2.VitessShard{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "shard",
					Namespace: testNamespace,
					Labels:    testLabels,
				},
				Spec: vitessv1alpha2.VitessShardSpec{
					TabletSelector: testSel,
				},
			},
			ValidationErrorNoShards,
		},
		{
			&vitessv1alpha2.VitessTablet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tablet",
					Namespace: testNamespace,
					Labels:    testLabels,
				},
				Spec: vitessv1alpha2.VitessTabletSpec{
					TabletID: 101,
					CellID:   "cell",
				},
			},
			ValidationErrorNoTablets,
		},
	}

	// Register operator types with the runtime scheme.
	s := scheme.Scheme
	s.AddKnownTypes(vitessv1alpha2.SchemeGroupVersion, &vitessv1alpha2.VitessCluster{})
	s.AddKnownTypes(vitessv1alpha2.SchemeGroupVersion, &vitessv1alpha2.VitessClusterList{})
	s.AddKnownTypes(vitessv1alpha2.SchemeGroupVersion, &vitessv1alpha2.VitessCell{})
	s.AddKnownTypes(vitessv1alpha2.SchemeGroupVersion, &vitessv1alpha2.VitessCellList{})
	s.AddKnownTypes(vitessv1alpha2.SchemeGroupVersion, &vitessv1alpha2.VitessTablet{})
	s.AddKnownTypes(vitessv1alpha2.SchemeGroupVersion, &vitessv1alpha2.VitessTabletList{})
	s.AddKnownTypes(vitessv1alpha2.SchemeGroupVersion, &vitessv1alpha2.VitessShard{})
	s.AddKnownTypes(vitessv1alpha2.SchemeGroupVersion, &vitessv1alpha2.VitessShardList{})
	s.AddKnownTypes(vitessv1alpha2.SchemeGroupVersion, &vitessv1alpha2.VitessKeyspace{})
	s.AddKnownTypes(vitessv1alpha2.SchemeGroupVersion, &vitessv1alpha2.VitessKeyspaceList{})
	s.AddKnownTypes(vitessv1alpha2.SchemeGroupVersion, &vitessv1alpha2.VitessLockserver{})
	s.AddKnownTypes(vitessv1alpha2.SchemeGroupVersion, &vitessv1alpha2.VitessLockserverList{})

	// Create a fake client to mock API calls.
	client := fake.NewFakeClient()

	n := New(client)

	// loop through and add objs one at a time
	// Cluster should not be valid until all objs have been added
	for _, test := range tests {
		cluster := &vitessv1alpha2.VitessCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testClusterName,
				Namespace: testNamespace,
			},
			Spec: vitessv1alpha2.VitessClusterSpec{
				CellSelector:     testSel,
				KeyspaceSelector: testSel,
			},
		}

		// handle special case for cluster lockserverRef
		if test.missingErr != ValidationErrorNoLockserverForCluster {
			cluster.Spec.LockserverRef = &corev1.LocalObjectReference{
				Name: "cluster-lockserver",
			}
		}

		// check for expected error when obj is missing
		if err := n.NormalizeCluster(cluster); err != nil {
			t.Fatalf("Error normalizing cluster: %s", err)
		}

		if err := n.ValidateCluster(cluster); err != test.missingErr {
			t.Fatalf("Wrong error for missing resource, got: '%s'; expected: '%s'", err, test.missingErr)
		}

		// add obj
		// t.Logf("Creating obj of kind: %s", test.obj.(metav1.Object).GetName())
		if err := client.Create(context.Background(), test.obj); err != nil {
			t.Fatalf("Error creating object: %s", err)
		}

		// redeclare empty cluster
		cluster = &vitessv1alpha2.VitessCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testClusterName,
				Namespace: testNamespace,
			},
			Spec: vitessv1alpha2.VitessClusterSpec{
				LockserverRef: &corev1.LocalObjectReference{
					Name: "cluster-lockserver",
				},
				CellSelector:     testSel,
				KeyspaceSelector: testSel,
			},
		}

		// Make sure there is a different error
		if err := n.NormalizeCluster(cluster); err != nil {
			t.Fatalf("Error normalizing cluster: %s", err)
		}

		if err := n.ValidateCluster(cluster); err == test.missingErr {
			t.Fatalf("Wrong error for missing resource, got: '%s' again; expected new error", err)
		}
	}

}

func TestValidateTabletHostnameSizeLimit(t *testing.T) {
	cluster := &vitessv1alpha2.VitessCluster{}
	cell := &vitessv1alpha2.VitessCell{}
	keyspace := &vitessv1alpha2.VitessKeyspace{}
	shard := &vitessv1alpha2.VitessShard{}
	tablet := &vitessv1alpha2.VitessTablet{}

	tablet.SetParentCluster(cluster)
	tablet.SetParentCell(cell)
	tablet.SetParentKeyspace(keyspace)
	tablet.SetParentShard(shard)

	baseLen := getMaxExpectedTabletHostLength(tablet)

	tests := []struct {
		numChars int
		expected ValidationError
	}{
		{
			(MaxTabletHostnameLength - baseLen) - 1, // one under max
			nil,
		},
		{
			MaxTabletHostnameLength - baseLen, // exactly max
			ValidationErrorTabletNameTooLong,
		},
	}

	n := New(fake.NewFakeClient())

	for _, tc := range tests {
		// increase the final hostname by the test size
		tablet.Keyspace().Name = strings.Repeat("x", tc.numChars)
		t.Logf("%s", tablet.GetStatefulSetName())
		err := n.ValidateTablet(tablet)
		if err != tc.expected {
			t.Errorf("Unexpected error: Got: %s; Expected: %s", err, tc.expected)
		}
	}
}

// TestSaneNormalAndValidCluster makes sure that a perfect cluster works as expected
func TestSaneNormalAndValidCluster(t *testing.T) {
	// Set the logger to development mode for verbose logs.
	// logf.SetLogger(logf.ZapLogger(true))

	// Define a minimal cluster which matches one of the cells above
	cluster := &vitessv1alpha2.VitessCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testClusterName,
			Namespace: testNamespace,
		},
		Spec: vitessv1alpha2.VitessClusterSpec{
			LockserverRef: &corev1.LocalObjectReference{
				Name: "cluster-lockserver",
			},
			CellSelector:     testSel,
			KeyspaceSelector: testSel,
		},
	}

	// Populate the client with initial data
	objs := []runtime.Object{
		&vitessv1alpha2.VitessLockserver{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "cluster-lockserver",
				Namespace: testNamespace,
				Labels:    testLabels,
			},
		},
		&vitessv1alpha2.VitessLockserver{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "cell-lockserver",
				Namespace: testNamespace,
				Labels:    testLabels,
			},
		},
		&vitessv1alpha2.VitessCell{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "cell",
				Namespace: testNamespace,
				Labels:    testLabels,
			},
			Spec: vitessv1alpha2.VitessCellSpec{
				LockserverRef: &corev1.LocalObjectReference{
					Name: "cell-lockserver",
				},
			},
		},
		&vitessv1alpha2.VitessKeyspace{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "keyspace",
				Namespace: testNamespace,
				Labels:    testLabels,
			},
			Spec: vitessv1alpha2.VitessKeyspaceSpec{
				ShardSelector: testSel,
			},
		},
		&vitessv1alpha2.VitessShard{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "shard",
				Namespace: testNamespace,
				Labels:    testLabels,
			},
			Spec: vitessv1alpha2.VitessShardSpec{
				TabletSelector: testSel,
			},
		},
		&vitessv1alpha2.VitessTablet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "tablet",
				Namespace: testNamespace,
				Labels:    testLabels,
			},
			Spec: vitessv1alpha2.VitessTabletSpec{
				TabletID: 101,
				CellID:   "cell",
			},
		},
		cluster,
	}

	// Register operator types with the runtime scheme.
	s := scheme.Scheme
	s.AddKnownTypes(vitessv1alpha2.SchemeGroupVersion, &vitessv1alpha2.VitessCluster{})
	s.AddKnownTypes(vitessv1alpha2.SchemeGroupVersion, &vitessv1alpha2.VitessClusterList{})
	s.AddKnownTypes(vitessv1alpha2.SchemeGroupVersion, &vitessv1alpha2.VitessCell{})
	s.AddKnownTypes(vitessv1alpha2.SchemeGroupVersion, &vitessv1alpha2.VitessCellList{})
	s.AddKnownTypes(vitessv1alpha2.SchemeGroupVersion, &vitessv1alpha2.VitessTablet{})
	s.AddKnownTypes(vitessv1alpha2.SchemeGroupVersion, &vitessv1alpha2.VitessTabletList{})
	s.AddKnownTypes(vitessv1alpha2.SchemeGroupVersion, &vitessv1alpha2.VitessShard{})
	s.AddKnownTypes(vitessv1alpha2.SchemeGroupVersion, &vitessv1alpha2.VitessShardList{})
	s.AddKnownTypes(vitessv1alpha2.SchemeGroupVersion, &vitessv1alpha2.VitessKeyspace{})
	s.AddKnownTypes(vitessv1alpha2.SchemeGroupVersion, &vitessv1alpha2.VitessKeyspaceList{})
	s.AddKnownTypes(vitessv1alpha2.SchemeGroupVersion, &vitessv1alpha2.VitessLockserver{})
	s.AddKnownTypes(vitessv1alpha2.SchemeGroupVersion, &vitessv1alpha2.VitessLockserverList{})

	// Create a fake client to mock API calls.
	client := fake.NewFakeClient(objs...)

	n := New(client)

	// Check Sanity
	if err := n.TestClusterSanity(cluster); err != nil {
		t.Fatalf("Cluster Sanity Test failed: %s", err)
	}

	// Call the normalize function for the cluster
	if err := n.NormalizeCluster(cluster); err != nil {
		t.Fatalf("Error normalizing cluster: %s", err)
	}

	// Ensure that all matched objects were embedded properly
	if err := n.ValidateCluster(cluster); err != nil {
		t.Fatalf("Cluster Sanity Test failed: %s", err)
	}

	// Test Parenting
	for _, keyspace := range cluster.Keyspaces() {
		shards := keyspace.Shards()
		if len(shards) == 0 {
			t.Fatalf("No embedded shards from keyspace after normalization")
		}

		for _, shard := range shards {
			tablets := shard.Tablets()
			if len(tablets) == 0 {
				t.Fatalf("No embedded tablets from shard after normalization")
			}
		}
	}

	// Child tests from the top down
	lockserver := cluster.Lockserver()
	if lockserver == nil {
		t.Errorf("No embeddded lockserver from cluster after normalization")
	}

	cells := cluster.Cells()
	if len(cells) == 0 {
		t.Errorf("No embedded cells from cluster after normalization")
	}

	shards := cluster.Shards()
	if len(shards) == 0 {
		t.Errorf("No embedded shards from cluster after normalization")
	}

	tablets := cluster.Tablets()
	if len(tablets) == 0 {
		t.Errorf("No embedded tablets from cluster after normalization")
	}

	keyspaces := cluster.Keyspaces()
	if len(keyspaces) == 0 {
		t.Errorf("No embedded keyspaces from cluster after normalization")
	}

	for _, keyspace := range keyspaces {
		shards := keyspace.Shards()
		if len(shards) == 0 {
			t.Errorf("No embedded shards from keyspace after normalization")
		}

		for _, shard := range shards {
			tablets := shard.Tablets()
			if len(tablets) == 0 {
				t.Errorf("No embedded tablets from shard after normalization")
			}
		}
	}

	// Parent tests from the bottom up

	// every tablet should have a parent cell, cluster, keyspace, and shard
	for _, tablet := range tablets {
		if tablet.Cell() == nil {
			t.Errorf("No parent cell in tablet after normalization")
		}
		if tablet.Cluster() == nil {
			t.Errorf("No parent cluster in tablet after normalization")
		}
		if tablet.Keyspace() == nil {
			t.Errorf("No parent keyspace in tablet after normalization")
		}
		if tablet.Shard() == nil {
			t.Errorf("No parent shard in tablet after normalization")
		}
		if tablet.Lockserver() == nil {
			t.Errorf("No lockserver in tablet after normalization")
		} else if tablet.Lockserver().GetName() != "cell-lockserver" {
			t.Errorf("Wrong lockserver in tablet after normalization. Should be 'cell-lockserver', not %s", tablet.Lockserver().GetName())
		}
	}

	// every shard should have a parent keyspace and cluster
	for _, shard := range shards {
		if shard.Keyspace() == nil {
			t.Errorf("No parent keyspace in shard after normalization")
		}

		if shard.Cluster() == nil {
			t.Errorf("No parent cluster in shard after normalization")
		}
	}

	// every keyspace should have a parent cluster
	for _, keyspace := range keyspaces {
		if keyspace.Cluster() == nil {
			t.Errorf("No parent cluster in keyspace after normalization")
		}
	}

	// every cell should have a parent cluster
	for _, cell := range cells {
		if cell.Cluster() == nil {
			t.Errorf("No parent cluster in cell after normalization")
		}
	}
}
