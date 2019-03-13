package vitesscluster

import (
	"strings"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	vitessv1alpha2 "vitess.io/vitess-operator/pkg/apis/vitess/v1alpha2"
	// "vitess.io/vitess-operator/pkg/normalizer"
)

func TestGetCellVTGateResources(t *testing.T) {

	// Define a minimal cluster
	cluster := &vitessv1alpha2.VitessCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testcluster",
			Namespace: "vitess",
		},
		Spec: vitessv1alpha2.VitessClusterSpec{
			Lockserver: &vitessv1alpha2.VitessLockserver{
				Spec: vitessv1alpha2.VitessLockserverSpec{
					Type: vitessv1alpha2.LockserverTypeEtcd2,
					Etcd2: &vitessv1alpha2.Etcd2Lockserver{
						Address:    "global-lockserver:8080",
						Path: "/global",
					},
				},
			},
		},
	}

	// Define a basic cell
	cell := &vitessv1alpha2.VitessCell{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "zone0",
			Namespace: "vitess",
		},
		Spec: vitessv1alpha2.VitessCellSpec{
			Lockserver: &vitessv1alpha2.VitessLockserver{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cell-lockserver",
				},
				Spec: vitessv1alpha2.VitessLockserverSpec{},
			},
		},
	}

	cell.SetParentCluster(cluster)

	// Get the resources
	deployment, service, err := GetCellVTGateResources(cell)

	if err != nil {
		t.Errorf("Got error generating vtgate resources for cell: %s", err)
	}

	// Validate basic returns
	if deployment == nil {
		t.Error("Got nil vtgate deployment for cell")
	}

	if service == nil {
		t.Error("Got nil vtgate service for cell")
	}

	// Test no mysql protocol

	if vtGateServiceHasMySQLPort(service) {
		t.Error("vtgate service had mysql port set and shouldn't have")
	}

	if vtGateDeploymentHasMySQLOpts(deployment, "-mysql_auth") {
		t.Error("vtgate deployment had mysql auth flags set and shouldn't have")
	}

	// Test mysql protocol with explict auth disable
	cell.Spec.MySQLProtocol = &vitessv1alpha2.VitessCellMySQLProtocol{
		AuthType: vitessv1alpha2.VitessMySQLAuthTypeNone,
	}

	deployment, service, err = GetCellVTGateResources(cell)

	if err != nil {
		t.Errorf("Got error generating vtgate resources for cell with mysql and no auth: %s", err)
	}

	if !vtGateServiceHasMySQLPort(service) {
		t.Error("vtgate service did not have mysql port set")
	}

	if !vtGateDeploymentHasMySQLOpts(deployment, "-mysql_auth_server_impl=\"none\"") {
		t.Error("vtgate deployment did not have mysql no auth flag")
	}

	// Test mysql protocol with static auth
	cell.Spec.MySQLProtocol = &vitessv1alpha2.VitessCellMySQLProtocol{
		Username:          "test",
		PasswordSecretRef: &corev1.SecretKeySelector{},
	}

	deployment, service, err = GetCellVTGateResources(cell)

	if err != nil {
		t.Errorf("Got error generating vtgate resources for cell with mysql and basic auth: %s", err)
	}

	if !vtGateServiceHasMySQLPort(service) {
		t.Error("vtgate service did not have mysql port set")
	}

	if !vtGateDeploymentHasMySQLOpts(deployment, "-mysql_auth_server_impl=\"static\"") {
		t.Error("vtgate deployment did not have mysql static auth flag")
	}
}

func vtGateServiceHasMySQLPort(service *corev1.Service) bool {
	for _, port := range service.Spec.Ports {
		if port.Name == "mysql" {
			return true
		}
	}
	return false
}

func vtGateDeploymentHasMySQLOpts(deployment *appsv1.Deployment, optstr string) bool {
	if strings.Contains(deployment.Spec.Template.Spec.Containers[0].Args[1], optstr) {
		return true
	}

	return false
}
