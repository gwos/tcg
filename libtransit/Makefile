# Makefile for libtransit/ files

# Define the build info used by Go linker
BUILD_TAG           := ${BUILD_TAG}
ifeq ($(BUILD_TAG),)
    BUILD_TAG       := ${TRAVIS_TAG}
endif
ifeq ($(BUILD_TAG),)
    BUILD_TAG       := ${COMMIT_HASH}
endif
ifeq ($(BUILD_TAG),)
    BUILD_TAG       := 8.x
endif

BUILD_TIME          := ${BUILD_TIME}
ifeq ($(BUILD_TIME),)
    BUILD_TIME      := $$(date -u +"%Y-%m-%dT%H:%M:%SZ")
endif

INCLUDES = -I./

# We use "-g" because it helps with running under gdb when we experience segfaults.
CFLAGS = ${INCLUDES} -m64 -std=gnu11 -g

all		: echo \
    transit.h \
    libtransit.h libtransit.so\
    libtransit_compat.h \
	examples

echo	:
	@echo Build info: ${BUILD_TAG} / ${BUILD_TIME}

run_examples: examples
	@echo Running examples
	LIBTRANSIT=./libtransit.so ./example_dl.run

examples	: libtransit.so libtransit.h transit.h ./examples/c/*.c
	${CC} ${CFLAGS} -o example_dl.run ./examples/c/example_dl.c -ldl

# Build header with constants definitions
transit.h	: ../sdk/transit/*.go
	go run ./tools/hgen/ --folders ../sdk/transit

# Build shared lib with build info
libtransit.h libtransit.so	: ../*/*.go ../*/*/*.go
	go build \
		-ldflags "-X 'github.com/gwos/tcg/config.buildTag=${BUILD_TAG}' \
			-X 'github.com/gwos/tcg/config.buildTime=${BUILD_TIME}'" \
		-buildmode=c-shared -o libtransit.so ./
	chmod +x libtransit.so

# Build combined header for compatibility
libtransit_compat.h	: libtransit.h transit.h
	cat transit.h > libtransit_compat.h
	echo >> libtransit_compat.h
	cat libtransit.h >> libtransit_compat.h

clean	:
	rm -rf *.h *.so *.run

distclean: clean

realclean: clean
