package scripts

const (
	VtCtldStart = `eval exec /vt/bin/vtctld $(cat <<END_OF_COMMAND
  -cell={{ .Cell.Name }}
  -web_dir="/vt/web/vtctld"
  -web_dir2="/vt/web/vtctld2/app"
  -workflow_manager_init
  -workflow_manager_use_election
  -logtostderr=true
  -stderrthreshold=0
  -port=15000
  -grpc_port=15999
  -service_map="grpc-vtctl"
  {{- if eq .LocalLockserver.Spec.Type "etcd2" }}
  -topo_implementation="etcd2"
  -topo_global_server_address="{{ .LocalLockserver.Spec.Etcd2.Address }}"
  -topo_global_root="{{ .LocalLockserver.Spec.Etcd2.Path }}"
  {{- end }}
END_OF_COMMAND
)
`
)
