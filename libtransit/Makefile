# Makefile for libtransit/ files

all	: libtransit.h libtransit.so

libtransit.h libtransit.so	: libtransit.go ../config/config.go ../milliseconds/milliseconds.go ../transit/*.go ../services/*.go ../log/*.go
	    # go tool cgo [cgo options] [-- compiler options] libtransit.go
	    # export GOPATH=${HOME}; go build -buildmode=c-shared -o libtransit.so libtransit.go
	    go build -buildmode=c-shared -o libtransit.so libtransit.go
	    chmod +x libtransit.so

clean	:
	rm -rf _obj libtransit.h libtransit.so
