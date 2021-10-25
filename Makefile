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

LIBTRANSIT_DIRECTORY = libtransit

LIBTRANSIT_LIBRARY = ${LIBTRANSIT_DIRECTORY}/libtransit.so

LIBTRANSIT_HEADERS =	\
	${LIBTRANSIT_DIRECTORY}/libtransit.h	\
	${LIBTRANSIT_DIRECTORY}/transit.h

BUILD_HEADER_FILES = \
	${LIBTRANSIT_HEADERS}

BUILD_DYNAMIC_LIBRARIES = \
	${LIBTRANSIT_LIBRARY}

INSTALL_DIRECTORIES = \
	${INSTALL_BASE_DIRECTORY}/include/tcg	\
	${INSTALL_BASE_DIRECTORY}/lib

INSTALL_HEADER_FILES = $(addprefix ${INSTALL_BASE_DIRECTORY}/include/tcg/,$(notdir ${BUILD_HEADER_FILES}))

INSTALL_DYNAMIC_LIBRARIES = $(addprefix ${INSTALL_BASE_DIRECTORY}/lib/,$(notdir ${BUILD_DYNAMIC_LIBRARIES}))

# We currently specify "-g" to assist in debugging and possibly also in memory-leak detection.
CFLAGS = -std=c11 -g -D_REENTRANT -D_GNU_SOURCE -fPIC -Wall
CC = gcc ${CFLAGS}

.PHONY	: all

all	: ${LIBTRANSIT_LIBRARY}

.PHONY	: install

install	: ${INSTALL_HEADER_FILES} ${INSTALL_DYNAMIC_LIBRARIES} | ${INSTALL_DIRECTORIES}

# called with arguments:  install directory, build-file path
define INSTALLED_FILE_template =
$(1)/$(notdir $(2))	: $(2) | $(1)
	cp -p $$< $$@
endef

$(foreach path,${BUILD_HEADER_FILES},$(eval $(call INSTALLED_FILE_template,${INSTALL_BASE_DIRECTORY}/include/tcg,$(path))))

$(foreach path,${BUILD_DYNAMIC_LIBRARIES},$(eval $(call INSTALLED_FILE_template,${INSTALL_BASE_DIRECTORY}/lib,$(path))))

# For the ${LIBTRANSIT_HEADERS} and ${LIBTRANSIT_LIBRARY} targets, there are many more
# dependencies than we ought to be keeping track of here, and they are all tracked
# instead in the subsidiary Makefile where those targets are actually built.  So instead
# of just depending on ${LIBTRANSIT_SOURCE}, which is of course the principal source file
# involved (but by no means the only one used to create the library), we simply force an
# unconditional descent into the subdirectory and attempt to make there.  That will handle
# all the details of build dependencies, at the cost of one unconditional recursive make.

.PHONY	: ${LIBTRANSIT_DIRECTORY}

${LIBTRANSIT_HEADERS} ${LIBTRANSIT_LIBRARY}	: ${LIBTRANSIT_DIRECTORY}
	make -C ${LIBTRANSIT_DIRECTORY}

${BUILD_TARGET_DIRECTORY}	:
	mkdir -p $@

${INSTALL_BASE_DIRECTORY}/include/tcg	:
	mkdir -p $@

${INSTALL_BASE_DIRECTORY}/lib	:
	mkdir -p $@

.PHONY	: clean

clean	:
	rm -rf ${BUILD_TARGET_DIRECTORY}
	make -C libtransit clean

.PHONY	: realclean

realclean	:
	rm -rf bin pkg src ../src
	make -C libtransit realclean

.PHONY	: distclean

distclean	: realclean
	rm -rf ${INSTALL_BASE_DIRECTORY}
	make -C libtransit distclean
