# Top-level Makefile for TNG code.

# export GOPATH := $(realpath .)
export GOPATH := $(realpath ..):$(realpath .)

# The current definition here is a placeholder for whatever we actually
# want to use according to some sort of project-standard file tree.
BUILD_TARGET_DIRECTORY = build

# We need a place to store header files as a final result of our code
# construction, where we can specify some parent directory for header
# files that is not either "build" (which is totally non-descriptive)
# or "include" (which also is not specific to this package).
INSTALL_BASE_DIRECTORY = install

CONVERT_GO_TO_C_BUILD_OBJECTS = \
	gotocjson/_c_code/convert_go_to_c.c	\
	gotocjson/_c_code/convert_go_to_c.h

CONFIG_BUILD_OBJECTS = \
	${BUILD_TARGET_DIRECTORY}/setup.c	\
	${BUILD_TARGET_DIRECTORY}/setup.h

MILLISECONDS_BUILD_OBJECTS = \
	${BUILD_TARGET_DIRECTORY}/subseconds.c	\
	${BUILD_TARGET_DIRECTORY}/subseconds.h

TRANSIT_BUILD_OBJECTS =	\
	${BUILD_TARGET_DIRECTORY}/transit.c		\
	${BUILD_TARGET_DIRECTORY}/transit.h

LIBTRANSITJSON_OBJECTS = \
	${BUILD_TARGET_DIRECTORY}/convert_go_to_c.o	\
	${BUILD_TARGET_DIRECTORY}/setup.o		\
	${BUILD_TARGET_DIRECTORY}/subseconds.o	\
	${BUILD_TARGET_DIRECTORY}/transit.o

LIBTRANSIT_DIRECTORY = libtransit

LIBTRANSIT_SOURCE = ${LIBTRANSIT_DIRECTORY}/libtransit.go

LIBTRANSIT_HEADER = ${LIBTRANSIT_DIRECTORY}/libtransit.h

LIBTRANSIT_LIBRARY = ${LIBTRANSIT_DIRECTORY}/libtransit.so

LIBTRANSITJSON_LIBRARY = ${BUILD_TARGET_DIRECTORY}/libtransitjson.so

BUILD_HEADER_FILES = \
	${LIBTRANSIT_HEADER}				\
	gotocjson/_c_code/convert_go_to_c.h		\
	${BUILD_TARGET_DIRECTORY}/setup.h		\
	${BUILD_TARGET_DIRECTORY}/subseconds.h	\
	${BUILD_TARGET_DIRECTORY}/transit.h

BUILD_DYNAMIC_LIBRARIES = \
	${LIBTRANSIT_LIBRARY}		\
	${LIBTRANSITJSON_LIBRARY}

INSTALL_DIRECTORIES = \
	${INSTALL_BASE_DIRECTORY}/include/tng	\
	${INSTALL_BASE_DIRECTORY}/lib

INSTALL_HEADER_FILES = $(addprefix ${INSTALL_BASE_DIRECTORY}/include/tng/,$(notdir ${BUILD_HEADER_FILES}))

INSTALL_DYNAMIC_LIBRARIES = $(addprefix ${INSTALL_BASE_DIRECTORY}/lib/,$(notdir ${BUILD_DYNAMIC_LIBRARIES}))

# We currently specify "-g" to assist in debugging and possibly also in memory-leak detection.
CFLAGS = -std=c11 -g -D_REENTRANT -D_GNU_SOURCE -fPIC -Wall
CC = gcc ${CFLAGS}

all	: ${LIBTRANSIT_LIBRARY} ${LIBTRANSITJSON_LIBRARY}

.PHONY	: install

install	: ${INSTALL_HEADER_FILES} ${INSTALL_DYNAMIC_LIBRARIES} | ${INSTALL_DIRECTORIES}

# called with arguments:  install directory, build-file path
define INSTALLED_FILE_template =
$(1)/$(notdir $(2))	: $(2) | $(1)
	cp -p $$< $$@
endef

$(foreach path,${BUILD_HEADER_FILES},$(eval $(call INSTALLED_FILE_template,${INSTALL_BASE_DIRECTORY}/include/tng,$(path))))

$(foreach path,${BUILD_DYNAMIC_LIBRARIES},$(eval $(call INSTALLED_FILE_template,${INSTALL_BASE_DIRECTORY}/lib,$(path))))

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

${LIBTRANSIT_HEADER} ${LIBTRANSIT_LIBRARY}	: ${LIBTRANSIT_SOURCE}
	cd ${LIBTRANSIT_DIRECTORY}; make

${BUILD_TARGET_DIRECTORY}	:
	mkdir -p ${BUILD_TARGET_DIRECTORY}

${INSTALL_BASE_DIRECTORY}/include/tng	:
	mkdir -p ${INSTALL_BASE_DIRECTORY}/include/tng

${INSTALL_BASE_DIRECTORY}/lib	:
	mkdir -p ${INSTALL_BASE_DIRECTORY}/lib

gotocjson/gotocjson	: gotocjson/gotocjson.go
	cd gotocjson; make gotocjson

${CONFIG_BUILD_OBJECTS}	: gotocjson/gotocjson setup/config.go | ${BUILD_TARGET_DIRECTORY}
	gotocjson/gotocjson -o ${BUILD_TARGET_DIRECTORY} setup/config.go

${MILLISECONDS_BUILD_OBJECTS}	: gotocjson/gotocjson subseconds/milliseconds.go | ${BUILD_TARGET_DIRECTORY}
	gotocjson/gotocjson -o ${BUILD_TARGET_DIRECTORY} subseconds/milliseconds.go

${TRANSIT_BUILD_OBJECTS}	: gotocjson/gotocjson transit/transit.go | ${BUILD_TARGET_DIRECTORY}
	gotocjson/gotocjson -o ${BUILD_TARGET_DIRECTORY} transit/transit.go

${BUILD_TARGET_DIRECTORY}/convert_go_to_c.o	: ${CONVERT_GO_TO_C_BUILD_OBJECTS} | ${BUILD_TARGET_DIRECTORY}
	${CC} -c gotocjson/_c_code/convert_go_to_c.c -o $@

${BUILD_TARGET_DIRECTORY}/setup.o	: ${CONFIG_BUILD_OBJECTS}
	${CC} -c ${BUILD_TARGET_DIRECTORY}/setup.c -o $@ -Igotocjson/_c_code

${BUILD_TARGET_DIRECTORY}/subseconds.o	: ${MILLISECONDS_BUILD_OBJECTS}
	${CC} -c ${BUILD_TARGET_DIRECTORY}/subseconds.c -o $@ -Igotocjson/_c_code

${BUILD_TARGET_DIRECTORY}/transit.o	: ${TRANSIT_BUILD_OBJECTS}
	${CC} -c ${BUILD_TARGET_DIRECTORY}/transit.c -o $@ -Igotocjson/_c_code

${LIBTRANSITJSON_LIBRARY}	: ${LIBTRANSITJSON_OBJECTS}
	${LINK.c} -shared -o $@ -fPIC ${LIBTRANSITJSON_OBJECTS} -ljansson

.PHONY	: clean

clean	:
	rm -rf ${BUILD_TARGET_DIRECTORY}
	cd gotocjson; make clean

.PHONY	: realclean

realclean	:
	rm -rf bin pkg src ../src
	cd gotocjson; make realclean

.PHONY	: distclean

distclean	: realclean
	rm -rf ${INSTALL_BASE_DIRECTORY}
	cd gotocjson; make distclean
