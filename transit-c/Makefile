# Makefile for transit-c/ files

JANSSON_BUILD_BASE_DIRECTORY = $(realpath ../local)

JANSSON_INSTALLED_BASE_DIRECTORY = /usr/local/groundwork/common

# Location of the compiled Jansson library, for build purposes.
JANSSON_BUILD_LIB_DIRECTORY = ${JANSSON_BUILD_BASE_DIRECTORY}/lib

# Location of the compiled Jansson library, for production linking purposes.
# It will be up to external procedures to place the compiled library in this location.
JANSSON_INSTALLED_LIB_DIRECTORY = ${JANSSON_INSTALLED_BASE_DIRECTORY}/lib

JANSSON_LIBRARY = ${JANSSON_BUILD_LIB_DIRECTORY}/libjansson.so

# libjansson.so might be present on the build system already because of
# dependencies of other programs, not related to our work here.  But that
# copy might well be out of date.  So we use linker flags to specify and
# lock down the version that we care about, in the binary that we build.
KERNEL_NAME := $(shell uname -s)
ifeq ($(KERNEL_NAME),Linux)
    # Here we link at build time to our local copy of the Jansson library,
    # since our build machine won't have a copy of that library installed
    # in the location where that library will reside in production.  But
    # we set things up so the LIBTRANSITJSON_LIBRARY (libtransitjson.so)
    # to which the JANSSON_LINK_FLAGS get applied will refer at run time
    # to the production copy of the Jansson library.
    JANSSON_LINK_FLAGS = -Wl,-L${JANSSON_BUILD_LIB_DIRECTORY} -ljansson -Wl,-R${JANSSON_BUILD_LIB_DIRECTORY}
endif
ifeq ($(KERNEL_NAME),Darwin)
    # The linker -rpath option (-Wl,-R... as it may appear on the compiler commmand line
    # on other platforms) is apparently built into dynamic libraries on Darwin (MacOS),
    # so we can't use (but don't need) -Wl,-R... on this platform.  But that also means
    # our build is going to have to install the library in the final installed location
    # before we can link to it at build time -- and that part is not yet covered here.
    JANSSON_LINK_FLAGS = -Wl,-L${JANSSON_BUILD_LIB_DIRECTORY} -ljansson
endif

INCLUDES = -I${JANSSON_BUILD_BASE_DIRECTORY}/include -I../libtransit

# We use "-g" because it helps with running under gdb when we experience segfaults.
CFLAGS = ${INCLUDES} -m64 -std=gnu11 ${JANSSON_LINK_FLAGS} -g

default	: all

# test_transit isn't working yet, so there's no sense in holding up our
# ability to easily run test_libtransit in the meantime.
# all	: test_transit test_libtransit
all	: test_libtransit

test_transit	: ../libtransit/libtransit.h test_transit.o transit_json.o

test_libtransit	: ../libtransit/libtransit.h test_libtransit.o
	cc -o test_libtransit test_libtransit.o -ldl

transit_json.o	: transit_json.c transit_json.h

../libtransit/libtransit.h	:
	cd ../libtransit ; make

# test_transit isn't working yet, so there's no sense in holding up our
# ability to easily run test_libtransit in the meantime.
# test	: test_transit test_libtransit
test	: test_libtransit
	# test_transit
	test_libtransit

clean	:
	rm -f *.o

realclean	: clean
	rm -f test_transit test_libtransit
