#!/usr/bin/env bash

__grab_jar () {
    echo "grabing jar..."
    fname=collagerest-common-8.0.0-SNAPSHOT.jar
    mkdir -p gw-transit/lib
    fpath=/src/groundwork/gw-server/target/lib/
    dc_id=$(docker create groundworkdevelopment/groundwork:master)
    docker cp "${dc_id}":${fpath}${fname} gw-transit/lib/
    docker rm -v "$dc_id" > /dev/null
}

cd $(dirname "$0")

ARGS=$@

if [ -z "$ARGS" ]; then
    echo "$(basename "$0"): packages not specified"
    echo "imply testing all packages"
    ARGS='-v ./...'
fi

case "$ARGS" in
    *"integration"*|*"./..."*) __grab_jar ;;
esac

docker run -it --rm --network host -v "${PWD}":/src groundworkdevelopment/tng ./docker_cmd.sh $ARGS
