#!/usr/bin/env bash

__build_libtransit() {
    go build -buildmode=c-shared -o /src/libtransit/libtransit.so /src/libtransit/libtransit.go
    export LIBTRANSIT=/src/libtransit/libtransit.so
}

__go_test() {
    go test $@
}

__build_libtransit

__go_test $@
