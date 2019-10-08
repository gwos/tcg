# Makefile for libtransit/ files

libtransit.h	:
	    # go tool cgo [cgo options] [-- compiler options] libtransit.go
	    # go tool cgo libtransit.go
	    export GOPATH=${HOME}; go build -buildmode=c-shared libtransit.go

clean	:
	rm -rf _obj libtransit.h libtransit
