package scripts

var (
	MySQLInitTemplate = `
set -ex
# set up the directories vitess needs
mkdir -p /vttmp/bin
mkdir -p /vtdataroot/tabletdata

# copy necessary assets to the volumeMounts
cp /vt/bin/mysqlctld /vttmp/bin/
cp /bin/busybox /vttmp/bin/
cp -R /vt/config /vttmp/

# make sure the log files exist
touch /vtdataroot/tabletdata/error.log
touch /vtdataroot/tabletdata/slow-query.log
touch /vtdataroot/tabletdata/general.log

# remove the old socket file if it is still around
rm -f /vtdataroot/tabletdata/mysql.sock
`

	MySQLStartTemplate = `
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



eval exec /vt/bin/mysqlctld $(cat <<END_OF_COMMAND
  -logtostderr=true
  -stderrthreshold=0
  -tablet_dir "tabletdata"
  -tablet_uid "$(cat /vtdataroot/tabletdata/tablet-uid)"
  -socket_file "/vtdataroot/mysqlctl.sock"
  -init_db_sql_file "/vt/config/init_db.sql"
END_OF_COMMAND
)
`
	MySQLPreStopTemplate = `
set -x

# block shutting down mysqlctld until vttablet shuts down first
until [ $VTTABLET_GONE ]; do

  # poll every 5 seconds to see if vttablet is still running
  /vttmp/bin/busybox wget --spider localhost:15002/debug/vars

  if [ $? -ne 0 ]; then
    VTTABLET_GONE=true
  fi

  sleep 5
done
`
)
