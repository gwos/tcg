#!/usr/bin/env bash

cd $(dirname "$0")

fname=collagerest-common-8.0.0-SNAPSHOT.jar
fpath=/src/groundwork/gw-server/target/lib/
dc_id=$(docker create groundworkdevelopment/groundwork:master)
docker cp "${dc_id}":${fpath}${fname} gw-transit/lib/
docker rm -v "$dc_id"

docker run -it --rm --network host -v "${PWD}":/src groundworkdevelopment/tng ./entrypoint.sh
