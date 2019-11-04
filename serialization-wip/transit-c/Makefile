# Makefile for transit-c/ files

# libjansson.so might be present on the build system already because of
# dependencies of other programs, not related to our work here.  But that
# copy might well be out of date.  So we use linker flags to specify and
# lock down the version that we care about, in the binary that we build.
LDFLAGS = -Wl,-L`realpath ../local/lib` -Wl,-R`realpath ../local/lib` -ljansson

INCLUDES = -I../local/include -I../libtransit

CFLAGS = ${INCLUDES} -m64 -std=gnu11 ${LDFLAGS}

default	: all

all	: test_transit test_libtransit

test_transit	: ../libtransit/libtransit.h test_transit.o transit_json.o

test_libtransit	: ../libtransit/libtransit.h test_libtransit.o
	cc -o test_libtransit test_libtransit.o -ldl

transit_json.o	: transit_json.c transit_json.h

../libtransit/libtransit.h	:
	cd ../libtransit ; make

test	: test_transit
	test_transit

clean	:
	rm -f *.o

realclean	: clean
	rm -f test_transit test_libtransit
