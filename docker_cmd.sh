#!/usr/bin/env bash

__build_libtransit() {
  go build -buildmode=c-shared -o /src/libtransit/libtransit.so /src/libtransit/libtransit.go
  export LIBTRANSIT=/src/libtransit/libtransit.so
}

__go_test() {
  go test -v ./"$PACKAGE_NAME"
}

__getopts() {
  PACKAGE_NAME=${1}
}

__getopts $@

__build_libtransit

__go_test
