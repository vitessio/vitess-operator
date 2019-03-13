package scripts

var (
	InitMySQLCreds = `
  set -ex
  creds=$(cat <<END_OF_COMMAND
  {
    "{{ .Cell.Spec.MySQLProtocol.Username }}": [
      {
        "UserData": "{{ .Cell.Spec.MySQLProtocol.Username }}",
        "Password": "$MYSQL_PASSWORD"
      }
    ],
    "vt_appdebug": []
  }
  END_OF_COMMAND
  )
  echo $creds > /mysqlcreds/creds.json
`
)
