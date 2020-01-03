# Makefile for libtransit/ files

libtransit.h libtransit.so	: libtransit.go ../setup/config.go ../subseconds/milliseconds.go ../transit/*.go ../services/*.go ../log/*.go
	    # go tool cgo [cgo options] [-- compiler options] libtransit.go
	    # export GOPATH=${HOME}; go build -buildmode=c-shared -o libtransit.so libtransit.go
	    go build -buildmode=c-shared -o libtransit.so libtransit.go
	    chmod +x libtransit.so

clean	:
	rm -rf _obj libtransit.h libtransit.so
