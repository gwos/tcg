# Makefile for libtransit/ files

libtransit.h	:
	    # go tool cgo [cgo options] [-- compiler options] libtransit.go
	    export GOPATH=${HOME}; go build -buildmode=c-shared -o libtransit.so libtransit.go

clean	:
	rm -rf _obj libtransit.h libtransit.so
