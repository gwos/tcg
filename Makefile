# Top-level Makefile for TNG code.

# export GOPATH := $(realpath .)
export GOPATH := $(realpath ..):$(realpath .)

# Here, we do not assume that the jansson library is available from an
# OS-provided package.  That's because such a package is likely to be out
# of date.  Instead, we download and install an appropriate package here.

JANSSON_VERSION = 2.12

JANSSON_BUILD_BASE_DIRECTORY = local

JANSSON_INSTALLED_BASE_DIRECTORY = /usr/local/groundwork/common

# Location of the Jansson library include files.
JANSSON_INCLUDE_DIRECTORY = ${JANSSON_BUILD_BASE_DIRECTORY}/include

# Location of the compiled Jansson library, for build purposes.
JANSSON_BUILD_LIB_DIRECTORY = ${JANSSON_BUILD_BASE_DIRECTORY}/lib

# Location of the compiled Jansson library, for production linking purposes.
# It will be up to external procedures to place the compiled library in this location.
JANSSON_INSTALLED_LIB_DIRECTORY = ${JANSSON_INSTALLED_BASE_DIRECTORY}/lib

JANSSON_LIBRARY = ${JANSSON_BUILD_LIB_DIRECTORY}/libjansson.so

KERNEL_NAME := $(shell uname -s)
ifeq ($(KERNEL_NAME),Linux)
    # Here we link at build time to our local copy of the Jansson library,
    # since our build machine won't have a copy of that library installed
    # in the location where that library will reside in production.  But
    # we set things up so the LIBTRANSITJSON_LIBRARY (libtransitjson.so)
    # to which the JANSSON_LINK_FLAGS get applied will refer at run time
    # to the production copy of the Jansson library.
    JANSSON_LINK_FLAGS = -Wl,-L${JANSSON_BUILD_LIB_DIRECTORY} -ljansson -Wl,-R${JANSSON_INSTALLED_LIB_DIRECTORY}
endif
ifeq ($(KERNEL_NAME),Darwin)
    # The linker -rpath option (-Wl,-R... as it may appear on the compiler commmand line
    # on other platforms) is apparently built into dynamic libraries on Darwin (MacOS),
    # so we can't use (but don't need) -Wl,-R... on this platform.  But that also means
    # our build is going to have to install the library in the final installed location
    # before we can link to it at build time -- and that part is not yet covered here.
    JANSSON_LINK_FLAGS = -Wl,-L${JANSSON_INSTALLED_LIB_DIRECTORY} -ljansson
endif

# The current definition here is a placeholder for whatever we actually
# want to use according to some sort of project-standard file tree.
BUILD_TARGET_DIRECTORY = build

CONVERT_GO_TO_C_BUILD_OBJECTS = \
	gotocjson/_c_code/convert_go_to_c.c	\
	gotocjson/_c_code/convert_go_to_c.h

MILLISECONDS_BUILD_OBJECTS = \
	${BUILD_TARGET_DIRECTORY}/milliseconds.c	\
	${BUILD_TARGET_DIRECTORY}/milliseconds.h

TRANSIT_BUILD_OBJECTS =	\
	${BUILD_TARGET_DIRECTORY}/transit.c		\
	${BUILD_TARGET_DIRECTORY}/transit.h

LIBTRANSITJSON_OBJECTS = \
	${BUILD_TARGET_DIRECTORY}/convert_go_to_c.o	\
	${BUILD_TARGET_DIRECTORY}/milliseconds.o	\
	${BUILD_TARGET_DIRECTORY}/transit.o

LIBTRANSITJSON_LIBRARY = ${BUILD_TARGET_DIRECTORY}/libtransitjson.so

# We currently specify "-g" to assist in debugging and possibly also in memory-leak detection.
CFLAGS = -std=c11 -g -D_REENTRANT -D_GNU_SOURCE -fPIC -Wall
CC = gcc $(CFLAGS)

all	: ${JANSSON_LIBRARY} ${LIBTRANSITJSON_LIBRARY}

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

${JANSSON_LIBRARY}	: jansson-${JANSSON_VERSION}/configure
	cd jansson-${JANSSON_VERSION}; ./configure --prefix=${PWD}/${JANSSON_BUILD_BASE_DIRECTORY}
	cd jansson-${JANSSON_VERSION}; make
	cd jansson-${JANSSON_VERSION}; make install

libtransit/libtransit.h	:
	cd libtransit ; make

${BUILD_TARGET_DIRECTORY}	:
	mkdir -p ${BUILD_TARGET_DIRECTORY}

gotocjson/gotocjson	: gotocjson/gotocjson.go
	cd gotocjson; make gotocjson

${MILLISECONDS_BUILD_OBJECTS}	: gotocjson/gotocjson milliseconds/milliseconds.go | ${BUILD_TARGET_DIRECTORY}
	gotocjson/gotocjson -o ${BUILD_TARGET_DIRECTORY} milliseconds/milliseconds.go

${TRANSIT_BUILD_OBJECTS}	: gotocjson/gotocjson transit/transit.go | ${BUILD_TARGET_DIRECTORY}
	gotocjson/gotocjson -o ${BUILD_TARGET_DIRECTORY} transit/transit.go

${BUILD_TARGET_DIRECTORY}/convert_go_to_c.o	: ${CONVERT_GO_TO_C_BUILD_OBJECTS} | ${BUILD_TARGET_DIRECTORY}
	${CC} -c gotocjson/_c_code/convert_go_to_c.c -o $@ -I${JANSSON_INCLUDE_DIRECTORY}

${BUILD_TARGET_DIRECTORY}/milliseconds.o	: ${MILLISECONDS_BUILD_OBJECTS}
	${CC} -c ${BUILD_TARGET_DIRECTORY}/milliseconds.c -o $@ -Igotocjson/_c_code -I${JANSSON_INCLUDE_DIRECTORY}

${BUILD_TARGET_DIRECTORY}/transit.o	: ${TRANSIT_BUILD_OBJECTS}
	${CC} -c ${BUILD_TARGET_DIRECTORY}/transit.c -o $@ -Igotocjson/_c_code -I${JANSSON_INCLUDE_DIRECTORY}

${LIBTRANSITJSON_LIBRARY}	: ${LIBTRANSITJSON_OBJECTS} ${JANSSON_LIBRARY}
	${LINK.c} -shared -o $@ -fPIC ${LIBTRANSITJSON_OBJECTS} ${JANSSON_LINK_FLAGS}

clean	:
	rm -rf BUILD
	cd gotocjson; make clean

realclean	:
	rm -rf jansson-${JANSSON_VERSION}
	rm -rf bin pkg src ../src
	cd gotocjson; make realclean

distclean	: realclean
	rm -rf ${JANSSON_BUILD_BASE_DIRECTORY} v${JANSSON_VERSION}.tar.gz jansson-${JANSSON_VERSION}.tar.gz 
	cd gotocjson; make distclean
