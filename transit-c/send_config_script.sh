#!/bin/bash

API_USER='RESTAPIACCESS'
API_URI='http://c1.bluesunrise.com/api'
CONFIG_API_URI='http://localhost:8099/api/v1/config'
API_PASSWORD='***REMOVED***'
#API_PASSWORD='***REMOVED***'

TOKEN=$(curl -s -X POST -d "user=${API_USER}&password=${API_PASSWORD}&gwos-app-name=gw8" "${API_URI}/auth/login")
if [ $? != 0 ]; then
  echo "ERROR: No auth token"
  exit 1
else
  _=$(curl -s -X POST -d '{}' -H "GWOS-APP-NAME:gw8" -H "GWOS-API-TOKEN:${TOKEN}" -H 'Accept: application/json' ${CONFIG_API_URI})
fi
