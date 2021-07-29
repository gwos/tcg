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

all	: echo libtransit.h libtransit.so

echo:
	@echo Build info: ${BUILD_TAG} / ${BUILD_TIME}

# Build shared lib and set build info values with Go linker
libtransit.h libtransit.so	: libtransit.go ../config/config.go ../milliseconds/milliseconds.go ../transit/*.go ../services/*.go ../logger/*.go
	go build \
		-ldflags "-X 'github.com/gwos/tcg/config.buildTag=${BUILD_TAG}' -X 'github.com/gwos/tcg/config.buildTime=${BUILD_TIME}'" \
		-buildmode=c-shared -o libtransit.so libtransit.go
	chmod +x libtransit.so

clean	:
	rm -rf _obj libtransit.h libtransit.so
