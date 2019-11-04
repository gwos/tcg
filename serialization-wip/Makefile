# Top-level Makefile for TNG code.

# export GOPATH := $(realpath .)
export GOPATH := $(realpath ..):$(realpath .)

# Here, we do not assume that the jansson library is available from an
# OS-provided package.  That's because such a package is likely to be out
# of date.  Instead, we download and install an appropriate package here.

JANSSON_VERSION = 2.12

all	: local/lib/libjansson.so libtransit/libtransit.h

# Fetch all third-party Go packages needed either directly or indirectly
# by the TNG software.
get	:
	mkdir -p ../src
	# [ -h src ] || ln -s ../src
	mkdir -p ../src/github.com/gwos
	[ -h ../src/github.com/gwos/tng ] || ln -s ../../../tng ../src/github.com/gwos/tng
	# FIX MAJOR:  It is not yet clear that the next step here will get us exactly
	# what we want:  nothing changed with respect to the checked-out and possibly
	# locally-modified files of the branch of github.com/gwos/tng that we are
	# already using.  All we want is to have it analyze the dependencies and
	# pull those in to make them available for our TNG builds.
	go get github.com/gwos/tng
	go get github.com/nats-io/go-nats-streaming
	go get github.com/nats-io/nats-streaming-server/server

# For no good reason, the upstream code does not follow the universal
# standard for naming the release tarball after the top-level directory
# that contains it.  So we must play a game here to clean that up.
jansson-${JANSSON_VERSION}.tar.gz	:
	wget https://github.com/akheron/jansson/archive/v${JANSSON_VERSION}.tar.gz
	mv v${JANSSON_VERSION}.tar.gz jansson-${JANSSON_VERSION}.tar.gz

jansson-${JANSSON_VERSION}/configure	: jansson-${JANSSON_VERSION}.tar.gz
	rm -rf jansson-${JANSSON_VERSION}
	tar xfz jansson-${JANSSON_VERSION}.tar.gz
	cd jansson-${JANSSON_VERSION}; autoreconf -i

local/lib/libjansson.so	: jansson-${JANSSON_VERSION}/configure
	cd jansson-${JANSSON_VERSION}; ./configure --prefix=${PWD}/local
	cd jansson-${JANSSON_VERSION}; make
	cd jansson-${JANSSON_VERSION}; make install

libtransit/libtransit.h	:
	cd libtransit ; make

clean	:
	rm -rf jansson-${JANSSON_VERSION}

realclean	:
	rm -rf bin pkg src ../src

distclean	: clean
	rm -rf local v${JANSSON_VERSION}.tar.gz jansson-${JANSSON_VERSION}.tar.gz 
