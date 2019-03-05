package scripts

var (
	InitReplicaMaster = `
set -ex

VTCTLD_SVC={{ .Cluster.Name }}-{{ .Cell.Name }}-vtctld.{{ .Cluster.Namespace }}:15999
SECONDS=0
TIMEOUT_SECONDS=600
VTCTL_EXTRA_FLAGS=()

# poll every 5 seconds to see if vtctld is ready
until vtctlclient ${VTCTL_EXTRA_FLAGS[@]} -server $VTCTLD_SVC ListAllTablets {{ .Cell.Name }} > /dev/null 2>&1; do
  if (( $SECONDS > $TIMEOUT_SECONDS )); then
    echo "timed out waiting for vtctlclient to be ready"
    exit 1
  fi
  sleep 5
done

until [ $TABLETS_READY ]; do
  # get all the tablets in the current cell
  cellTablets="$(vtctlclient ${VTCTL_EXTRA_FLAGS[@]} -server $VTCTLD_SVC ListAllTablets {{ .Cell.Name }})"

  # filter to only the tablets in our current shard
  shardTablets=$( echo "$cellTablets" | grep -w '{{ .Cluster.Name }}-{{ .Cell.Name }}-{{ .Keyspace.Name }}-{{ .Shard.Name }}' || : )

  # check for a master tablet from the ListAllTablets call
  masterTablet=$( echo "$shardTablets" | awk '$4 == "master" {print $1}')
  if [ $masterTablet ]; then
      echo "'$masterTablet' is already the master tablet, exiting without running InitShardMaster"
      exit
  fi

  # check for a master tablet from the GetShard call
  master_alias=$(vtctlclient ${VTLCTL_EXTRA_FLAGS[@]} -server $VTCTLD_SVC GetShard {{ .Keyspace.Name }}/{{ .Shard.Spec.KeyRange }} | jq '.master_alias.uid')
  if [ "$master_alias" != "null" -a "$master_alias" != "" ]; then
      echo "'$master_alias' is already the master tablet, exiting without running InitShardMaster"
      exit
  fi

  # count the number of newlines for the given shard to get the tablet count
  tabletCount=$( echo "$shardTablets" | wc | awk '{print $1}')

  # check to see if the tablet count equals the expected tablet count
  if [ $tabletCount == 2 ]; then
    TABLETS_READY=true
  else
    if (( $SECONDS > $TIMEOUT_SECONDS )); then
      echo "timed out waiting for tablets to be ready"
      exit 1
    fi

    # wait 5 seconds for vttablets to continue getting ready
    sleep 5
  fi

done

# find the tablet id for the "-replica-0" stateful set for a given cell, keyspace and shard
tablet_id=$( echo "$shardTablets" | grep -w '{{ .ScopedName }}-replica-0' | awk '{print $1}')

# initialize the shard master
until vtctlclient ${VTCTL_EXTRA_FLAGS[@]} -server $VTCTLD_SVC InitShardMaster -force {{ .Keyspace.Name }}/{{ .Shard.Spec.KeyRange }} $tablet_id; do
  if (( $SECONDS > $TIMEOUT_SECONDS )); then
    echo "timed out waiting for InitShardMaster to succeed"
    exit 1
  fi
  sleep 5
done
`
)
