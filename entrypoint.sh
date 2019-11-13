#!/usr/bin/env bash

__build_libtransit () {
    go build -buildmode=c-shared -o libtransit/libtransit.so libtransit/libtransit.go
    ln -s -T /src/libtransit/libtransit.so /src/gw-transit/src/main/resources
    ln -s -T /src/libtransit/libtransit.so /src/gw-transit/src/test/resources
}

__build_libtransit
