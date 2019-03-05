package scripts

import (
	"bytes"
	"fmt"
	"text/template"

	"k8s.io/apimachinery/pkg/runtime"

	vitessv1alpha2 "vitess.io/vitess-operator/pkg/apis/vitess/v1alpha2"
)

type ContainerScriptGenerator struct {
	ContainerType string
	Object        runtime.Object
	Init          string
	Start         string
	PreStop       string
}

func NewContainerScriptGenerator(containerType string, obj runtime.Object) *ContainerScriptGenerator {
	return &ContainerScriptGenerator{
		ContainerType: containerType,
		Object:        obj,
	}
}

func (csg *ContainerScriptGenerator) Generate() error {
	var err error
	switch csg.ContainerType {
	case "vttablet":
		csg.Init, err = csg.getTemplatedScript("vttabletinit", VTTabletInitTemplate)
		if err != nil {
			return err
		}
		csg.Start, err = csg.getTemplatedScript("vttabletstart", VTTabletStartTemplate)
		if err != nil {
			return err
		}
		csg.PreStop, err = csg.getTemplatedScript("vttabletPreStop", VTTabletPreStopTemplate)
		if err != nil {
			return err
		}
	case "mysql":
		csg.Init, err = csg.getTemplatedScript("mysqlinit", MySQLInitTemplate)
		if err != nil {
			return err
		}
		csg.Start, err = csg.getTemplatedScript("mysqlstart", MySQLStartTemplate)
		if err != nil {
			return err
		}
		csg.PreStop, err = csg.getTemplatedScript("mysqlPreStop", MySQLPreStopTemplate)
		if err != nil {
			return err
		}
	case "init_replica_master":
		csg.Start, err = csg.getTemplatedScript("init_replica_master", InitReplicaMaster)
		if err != nil {
			return err
		}
	case "vtctld":
		csg.Start, err = csg.getTemplatedScript("vtctld", VtCtldStart)
		if err != nil {
			return err
		}
	case "vtgate":
		csg.Start, err = csg.getTemplatedScript("vtgate", VTGateStart)
		if err != nil {
			return err
		}
	case "init-mysql-creds":
		csg.Start, err = csg.getTemplatedScript("init-mysql-creds", InitMySQLCreds)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("Unsupported container type: %s", csg.ContainerType)
	}

	return nil
}

func (csg *ContainerScriptGenerator) getTemplatedScript(name string, templateStr string) (string, error) {
	tmpl, err := template.New(name).Parse(templateStr)
	if err != nil {
		return "", err
	}

	// if tablet, ok := csg.Object.(*vitessv1alpha2.VitessTablet); ok {
	// 	return getTemplatedScriptForTablet(name, templateStr)
	// }

	// if cell, ok := csg.Object.(*vitessv1alpha2.VitessTablet); ok {
	// 	return getTemplatedScriptForTablet(name, templateStr)
	// }
	// }

	// func (csg *ContainerScriptGenerator) getTemplatedScriptForTablet(name string, templateStr string) (string, error) {
	// Params are different depending on the resource type

	// For simplicity, the tablet and all parent objects are passed to the template.
	// This is safe while the templates are hard-coded. But if templates are ever made
	// end-user configurable could would potentially expose too much data and would need to be sanitized
	var params map[string]interface{}

	// Configure tablet params
	if tablet, ok := csg.Object.(*vitessv1alpha2.VitessTablet); ok {
		params = map[string]interface{}{
			"LocalLockserver":  tablet.Lockserver(),
			"GlobalLockserver": tablet.Cluster().Lockserver(),
			"Cluster":          tablet.Cluster(),
			"Cell":             tablet.Cell(),
			"Keyspace":         tablet.Keyspace(),
			"Shard":            tablet.Shard(),
			"Tablet":           tablet,
			"ScopedName":       tablet.GetScopedName(),
		}
	}

	// Configure shard params
	if cell, ok := csg.Object.(*vitessv1alpha2.VitessCell); ok {
		params = map[string]interface{}{
			"LocalLockserver":  cell.Lockserver(),
			"GlobalLockserver": cell.Cluster().Lockserver(),
			"Cluster":          cell.Cluster(),
			"Cell":             cell,
			"ScopedName":       cell.GetScopedName(),
		}
	}

	var out bytes.Buffer
	err = tmpl.Execute(&out, params)
	if err != nil {
		return "", err
	}

	return out.String(), nil
}
