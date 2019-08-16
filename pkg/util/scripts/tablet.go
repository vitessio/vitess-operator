package scripts

const (
	VTTabletInitTemplate = `
set -ex
# Split pod name (via hostname) into prefix and ordinal index.
hostname=$(hostname -s)
[[ $hostname =~ ^(.+)-([0-9]+)$ ]] || exit 1
pod_prefix=${BASH_REMATCH[1]}
pod_index=${BASH_REMATCH[2]}

# Prepend cell name since tablet UIDs must be globally unique.
uid_name=zone1-$pod_prefix

# Take MD5 hash of cellname-podprefix.
uid_hash=$(echo -n $uid_name | md5sum | awk "{print \$1}")

# Take first 24 bits of hash, convert to decimal.
# Shift left 2 decimal digits, add in index.
tablet_uid=$((16#${uid_hash:0:6} * 100 + $pod_index))

# Save UID for other containers to read.
echo $tablet_uid > /vtdataroot/tabletdata/tablet-uid

# Tell MySQL what hostname to report in SHOW SLAVE HOSTS.
echo report-host=$hostname.{{ .Cluster.Name }}-tab > /vtdataroot/tabletdata/report-host.cnf

# Orchestrator looks there, so it should match -tablet_hostname above.

# make sure that etcd is initialized
eval exec /vt/bin/vtctl $(cat <<END_OF_COMMAND
{{- if eq .LocalLockserver.Spec.Type "etcd2" }}
  -topo_implementation="etcd2"
  -topo_global_root="{{ .LocalLockserver.Spec.Etcd2.Path }}"
  -topo_global_server_address="{{ .LocalLockserver.Spec.Etcd2.Address }}"
{{- end }}
  -logtostderr=true
  -stderrthreshold=0
  UpdateCellInfo
  -server_address="{{ .LocalLockserver.Spec.Etcd2.Address }}"
  "{{ .Cell.Name }}"
END_OF_COMMAND
)
`

	VTTabletStartTemplate = `
set -ex

if [ "$VT_DB_FLAVOR" = "percona" ]; then
  MYSQL_FLAVOR=Percona

elif [ "$VT_DB_FLAVOR" = "mysql" ]; then
  MYSQL_FLAVOR=MySQL56

elif [ "$VT_DB_FLAVOR" = "mysql56" ]; then
  MYSQL_FLAVOR=MySQL56

elif [ "$VT_DB_FLAVOR" = "maria" ]; then
  MYSQL_FLAVOR=MariaDB

elif [ "$VT_DB_FLAVOR" = "mariadb" ]; then
  MYSQL_FLAVOR=MariaDB

elif [ "$VT_DB_FLAVOR" = "mariadb103" ]; then
  MYSQL_FLAVOR=MariaDB103

fi

export MYSQL_FLAVOR
export EXTRA_MY_CNF="/vtdataroot/tabletdata/report-host.cnf:/vt/config/mycnf/rbr.cnf"

{{ if eq .LocalLockserver.Spec.Type "etcd2" }}
eval exec /vt/bin/vttablet $(cat <<END_OF_COMMAND
  -topo_implementation="etcd2"
  -topo_global_server_address="{{ .LocalLockserver.Spec.Etcd2.Address }}"
  -topo_global_root="{{ .LocalLockserver.Spec.Etcd2.Path }}"
  -logtostderr
  -port=15002
  -grpc_port=16002
  -service_map="grpc-queryservice,grpc-tabletmanager,grpc-updatestream"
  -tablet_dir="tabletdata"
  -tablet-path="{{ .Cell.Name }}-$(cat /vtdataroot/tabletdata/tablet-uid)"
  -tablet_hostname="$(hostname).{{ .Cluster.Name }}-tab"
  -init_keyspace="{{ .Keyspace.Name }}"
  -init_shard="{{ .Shard.Spec.KeyRange }}"
  -init_tablet_type="{{ .Tablet.Spec.Type }}"
  -init_db_name_override="{{ .Keyspace.Name }}"
  -v=7
  -health_check_interval="5s"
  -mysqlctl_socket="/vtdataroot/mysqlctl.sock"
  -enable_replication_reporter
END_OF_COMMAND
)
{{ end }}
`

	// TODO move the actual reparenting work to the operator
	VTTabletPreStopTemplate = `
set -x

VTCTLD_SVC=vtctld.default:15999
VTCTL_EXTRA_FLAGS=()

master_alias_json=$(/vt/bin/vtctlclient ${VTCTL_EXTRA_FLAGS[@]} -server $VTCTLD_SVC GetShard {{ .Keyspace.Name }}/{{ .Shard.Spec.KeyRange }})
master_cell=$(jq -r '.master_alias.cell' <<< "$master_alias_json")
master_uid=$(jq -r '.master_alias.uid' <<< "$master_alias_json")
master_alias=$master_cell-$master_uid

current_uid=$(cat /vtdataroot/tabletdata/tablet-uid)
current_alias=zone1-$current_uid

if [ $master_alias != $current_alias ]; then
    # since this isn't the master, there's no reason to reparent
    exit
fi

# TODO: add more robust health checks to make sure that we don't initiate a reparent
# if there isn't a healthy enough replica to take over
# - seconds behind master
# - use GTID_SUBTRACT

RETRY_COUNT=0
MAX_RETRY_COUNT=100000

# retry reparenting
until [ $DONE_REPARENTING ]; do

  # reparent before shutting down
  /vt/bin/vtctlclient ${VTCTL_EXTRA_FLAGS[@]} -server $VTCTLD_SVC PlannedReparentShard -keyspace_shard={{ .Keyspace.Name }}/{{ .Shard.Spec.KeyRange }} -avoid_master=$current_alias

  # if PlannedReparentShard succeeded, then don't retry
  if [ $? -eq 0 ]; then
    DONE_REPARENTING=true

  # if we've reached the max retry count, exit unsuccessfully
  elif [ $RETRY_COUNT -eq $MAX_RETRY_COUNT ]; then
    exit 1

  # otherwise, increment the retry count and sleep for 10 seconds
  else
    let RETRY_COUNT=RETRY_COUNT+1
    sleep 10
  fi

done

# delete the current tablet from topology. Not strictly necessary, but helps to prevent
# edge cases where there are two masters
/vt/bin/vtctlclient ${VTCTL_EXTRA_FLAGS[@]} -server $VTCTLD_SVC DeleteTablet $current_alias
`
)
