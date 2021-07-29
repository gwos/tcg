# Top-level Makefile for TCG code.

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

GENERIC_DATATYPES_BUILD_OBJECTS = \
	${BUILD_TARGET_DIRECTORY}/generic_datatypes.c	\
	${BUILD_TARGET_DIRECTORY}/generic_datatypes.h

TIME_BUILD_OBJECTS = \
	${BUILD_TARGET_DIRECTORY}/time.c	\
	${BUILD_TARGET_DIRECTORY}/time.h

CONFIG_BUILD_OBJECTS = \
	${BUILD_TARGET_DIRECTORY}/config.c	\
	${BUILD_TARGET_DIRECTORY}/config.h

MILLISECONDS_BUILD_OBJECTS = \
	${BUILD_TARGET_DIRECTORY}/milliseconds.c	\
	${BUILD_TARGET_DIRECTORY}/milliseconds.h

TRANSIT_BUILD_OBJECTS =	\
	${BUILD_TARGET_DIRECTORY}/transit.c		\
	${BUILD_TARGET_DIRECTORY}/transit.h

LIBTRANSITJSON_OBJECTS = \
	${BUILD_TARGET_DIRECTORY}/convert_go_to_c.o	\
	${BUILD_TARGET_DIRECTORY}/generic_datatypes.o	\
	${BUILD_TARGET_DIRECTORY}/config.o		\
	${BUILD_TARGET_DIRECTORY}/milliseconds.o	\
	${BUILD_TARGET_DIRECTORY}/transit.o

LIBTRANSIT_DIRECTORY = libtransit

LIBTRANSIT_SOURCE = ${LIBTRANSIT_DIRECTORY}/libtransit.go

LIBTRANSIT_HEADER = ${LIBTRANSIT_DIRECTORY}/libtransit.h

LIBTRANSIT_LIBRARY = ${LIBTRANSIT_DIRECTORY}/libtransit.so

LIBTRANSITJSON_LIBRARY = ${BUILD_TARGET_DIRECTORY}/libtransitjson.so

BUILD_HEADER_FILES = \
	${LIBTRANSIT_HEADER}				\
	gotocjson/_c_code/convert_go_to_c.h		\
	${BUILD_TARGET_DIRECTORY}/time.h		\
	${BUILD_TARGET_DIRECTORY}/generic_datatypes.h	\
	${BUILD_TARGET_DIRECTORY}/config.h		\
	${BUILD_TARGET_DIRECTORY}/milliseconds.h	\
	${BUILD_TARGET_DIRECTORY}/transit.h

BUILD_DYNAMIC_LIBRARIES = \
	${LIBTRANSIT_LIBRARY}		\
	${LIBTRANSITJSON_LIBRARY}

INSTALL_DIRECTORIES = \
	${INSTALL_BASE_DIRECTORY}/include/tcg	\
	${INSTALL_BASE_DIRECTORY}/lib

INSTALL_HEADER_FILES = $(addprefix ${INSTALL_BASE_DIRECTORY}/include/tcg/,$(notdir ${BUILD_HEADER_FILES}))

INSTALL_DYNAMIC_LIBRARIES = $(addprefix ${INSTALL_BASE_DIRECTORY}/lib/,$(notdir ${BUILD_DYNAMIC_LIBRARIES}))

# We currently specify "-g" to assist in debugging and possibly also in memory-leak detection.
CFLAGS = -std=c11 -g -D_REENTRANT -D_GNU_SOURCE -fPIC -Wall
CC = gcc ${CFLAGS}

.PHONY	: all

all	: ${LIBTRANSIT_LIBRARY} ${LIBTRANSITJSON_LIBRARY}

.PHONY	: install

install	: ${INSTALL_HEADER_FILES} ${INSTALL_DYNAMIC_LIBRARIES} | ${INSTALL_DIRECTORIES}

# called with arguments:  install directory, build-file path
define INSTALLED_FILE_template =
$(1)/$(notdir $(2))	: $(2) | $(1)
	cp -p $$< $$@
endef

$(foreach path,${BUILD_HEADER_FILES},$(eval $(call INSTALLED_FILE_template,${INSTALL_BASE_DIRECTORY}/include/tcg,$(path))))

$(foreach path,${BUILD_DYNAMIC_LIBRARIES},$(eval $(call INSTALLED_FILE_template,${INSTALL_BASE_DIRECTORY}/lib,$(path))))

# Fetch all third-party Go packages needed either directly or indirectly
# by the TCG software.
get	:
	mkdir -p ../src
	# [ -h src ] || ln -s ../src
	mkdir -p ../src/github.com/gwos
	[ -h ../src/github.com/gwos/tcg ] || ln -s ../../../tcg ../src/github.com/gwos/tcg
	# FIX MAJOR:  It is not yet clear that the next step here will get us exactly
	# what we want:  nothing changed with respect to the checked-out and possibly
	# locally-modified files of the branch of github.com/gwos/tcg that we are
	# already using.  All we want is to have it analyze the dependencies and
	# pull those in to make them available for our TCG builds.
	go get github.com/gwos/tcg
	go get github.com/nats-io/go-nats-streaming
	go get github.com/nats-io/nats-streaming-server/server

# For the ${LIBTRANSIT_HEADER} and ${LIBTRANSIT_LIBRARY} targets, there are many more
# dependencies than we ought to be keeping track of here, and they are all tracked
# instead in the subsidiary Makefile where those targets are actually built.  So instead
# of just depending on ${LIBTRANSIT_SOURCE}, which is of course the principal source file
# involved (but by no means the only one used to create the library), we simply force an
# unconditional descent into the subdirectory and attempt to make there.  That will handle
# all the details of build dependencies, at the cost of one unconditional recursive make.

.PHONY	: ${LIBTRANSIT_DIRECTORY}

${LIBTRANSIT_HEADER} ${LIBTRANSIT_LIBRARY}	: ${LIBTRANSIT_DIRECTORY}
	make -C ${LIBTRANSIT_DIRECTORY}

${BUILD_TARGET_DIRECTORY}	:
	mkdir -p $@

${INSTALL_BASE_DIRECTORY}/include/tcg	:
	mkdir -p $@

${INSTALL_BASE_DIRECTORY}/lib	:
	mkdir -p $@

gotocjson/gotocjson	: gotocjson/gotocjson.go
	make -C gotocjson gotocjson

${GENERIC_DATATYPES_BUILD_OBJECTS}	: gotocjson/gotocjson gotocjson/generic_datatypes/generic_datatypes.go | ${BUILD_TARGET_DIRECTORY}
	gotocjson/gotocjson -g -o ${BUILD_TARGET_DIRECTORY} gotocjson/generic_datatypes/generic_datatypes.go

${TIME_BUILD_OBJECTS}	: gotocjson/gotocjson time/time.go | ${BUILD_TARGET_DIRECTORY}
	gotocjson/gotocjson -o ${BUILD_TARGET_DIRECTORY} time/time.go

${CONFIG_BUILD_OBJECTS}	: gotocjson/gotocjson ${TIME_BUILD_OBJECTS} logger/logger.go ${TRANSIT_BUILD_OBJECTS} config/config.go | ${BUILD_TARGET_DIRECTORY}
	gotocjson/gotocjson -o ${BUILD_TARGET_DIRECTORY} config/config.go

${MILLISECONDS_BUILD_OBJECTS}	: gotocjson/gotocjson ${TIME_BUILD_OBJECTS} milliseconds/milliseconds.go | ${BUILD_TARGET_DIRECTORY}
	gotocjson/gotocjson -o ${BUILD_TARGET_DIRECTORY} milliseconds/milliseconds.go

${TRANSIT_BUILD_OBJECTS}	: gotocjson/gotocjson ${MILLISECONDS_BUILD_OBJECTS} transit/transit.go | ${BUILD_TARGET_DIRECTORY}
	gotocjson/gotocjson -o ${BUILD_TARGET_DIRECTORY} transit/transit.go

${BUILD_TARGET_DIRECTORY}/convert_go_to_c.o	: ${CONVERT_GO_TO_C_BUILD_OBJECTS} | ${BUILD_TARGET_DIRECTORY}
	${CC} -c gotocjson/_c_code/convert_go_to_c.c -o $@

${BUILD_TARGET_DIRECTORY}/generic_datatypes.o	: ${GENERIC_DATATYPES_BUILD_OBJECTS}
	${CC} -c ${BUILD_TARGET_DIRECTORY}/generic_datatypes.c -o $@ -Igotocjson/_c_code

${BUILD_TARGET_DIRECTORY}/config.o	: ${TIME_BUILD_OBJECTS} ${TRANSIT_BUILD_OBJECTS} ${CONFIG_BUILD_OBJECTS} ${BUILD_TARGET_DIRECTORY}/generic_datatypes.h
	${CC} -c ${BUILD_TARGET_DIRECTORY}/config.c -o $@ -Igotocjson/_c_code

${BUILD_TARGET_DIRECTORY}/milliseconds.o	: ${TIME_BUILD_OBJECTS} ${MILLISECONDS_BUILD_OBJECTS} ${BUILD_TARGET_DIRECTORY}/generic_datatypes.h
	${CC} -c ${BUILD_TARGET_DIRECTORY}/milliseconds.c -o $@ -Igotocjson/_c_code

${BUILD_TARGET_DIRECTORY}/transit.o	: ${MILLISECONDS_BUILD_OBJECTS} ${TRANSIT_BUILD_OBJECTS} ${BUILD_TARGET_DIRECTORY}/generic_datatypes.h
	${CC} -c ${BUILD_TARGET_DIRECTORY}/transit.c -o $@ -Igotocjson/_c_code

${LIBTRANSITJSON_LIBRARY}	: ${LIBTRANSITJSON_OBJECTS}
	${LINK.c} -shared -o $@ -fPIC ${LIBTRANSITJSON_OBJECTS} -ljansson

.PHONY	: clean

clean	:
	rm -rf ${BUILD_TARGET_DIRECTORY}
	make -C gotocjson clean

.PHONY	: realclean

realclean	:
	rm -rf bin pkg src ../src
	make -C gotocjson realclean

.PHONY	: distclean

distclean	: realclean
	rm -rf ${INSTALL_BASE_DIRECTORY}
	make -C gotocjson distclean
