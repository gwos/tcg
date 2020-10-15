#!/bin/bash

TCG_API_CONFIG=${TCG_API_CONFIG:-'http://localhost:8099/api/v1/config'}
GWOS_API_LOGIN=${GWOS_API_LOGIN:-'http://localhost:8080/api/auth/login'}
GWOS_WSUSERNAME=${GWOS_WSUSERNAME:-'RESTAPIACCESS'}
GWOS_WSPASSWORD=${GWOS_WSPASSWORD}

TOKEN=$(curl -s -X POST -d "user=${GWOS_WSUSERNAME}&password=${GWOS_WSPASSWORD}&gwos-app-name=gw8" "${GWOS_API_LOGIN}")
if [ $? != 0 ]; then
  echo "ERROR: No auth token"
  exit 1
fi

curl -s -X POST -d '{}' -H "GWOS-APP-NAME:gw8" -H "GWOS-API-TOKEN:${TOKEN}" -H 'Accept: application/json' ${TCG_API_CONFIG}
