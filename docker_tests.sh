#!/usr/bin/env bash

cd $(dirname "$0")

__getopts() {
  PACKAGE_NAME=${1}
}

__getopts $@

if [ -z "$PACKAGE_NAME" ]; then
  echo "docker_tests.sh: <package_name> not specified"
  exit 1
fi

if [ "$PACKAGE_NAME" == "integration" ]; then
  fname=collagerest-common-8.0.0-SNAPSHOT.jar
  mkdir -p gw-transit/lib
  fpath=/src/groundwork/gw-server/target/lib/
  dc_id=$(docker create groundworkdevelopment/groundwork:master)
  docker cp "${dc_id}":${fpath}${fname} gw-transit/lib/
  docker rm -v "$dc_id"
fi

docker run -it --rm --network host -v "${PWD}":/src groundworkdevelopment/tng ./docker_cmd.sh "$PACKAGE_NAME"
