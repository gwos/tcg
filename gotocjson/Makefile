#
# Makefile for gotocjson
#
# This is the Makefile for a tool that analyzes a Go source file, identifies all
# the struct typedefs, enumeration typdefs, and enumeration constants, and writes
# out C code that supports equivalent enumerations and structs along with associated
# JSON marshalling and unmarshalling that matches the JSON processing (as tweaked by
# field tags) in the original Go code.  This tool may find use when exchanging data
# between Go and C contexts.

# The testdata/enum.go file is used for ad-hoc testing of the gotocjson tool,
# during development.

# The copies of setup.go, subseconds.go, and transport.go in the testdata directory are NOT
# production code.  Instead, they are local test copies of config.go, milliseconds.go, and
# transit.go, intended to freeze stable versions of enumerations and data structures that
# we can reliably use for testing the gotocjson tool ehile developint it.

# The _c_code/convert_go_to_c.h and _c_code/convert_go_to_c.c files comprise an integral
# part of the implementation of the gotocjson tool.  They provide routines which are not
# derived from converted Go code, but which are necessary to support the converted Go code.

# The Jansson library is used for parsing and generating JSON strings from C-language
# code.  We use a locally-built copy of this library because it is fairly likely to be
# more up-to-date than a system-provided copy.

# Location of the Jansson library include files.
JANSSON_INCLUDE_DIRECTORY = ../local/include

# Location of the compiled Jansson library, for linking purposes.
JANSSON_LIB_DIRECTORY = ../local/lib

# JSON test files are expected to be named as "json.{package}.{structure}.{item}",
# where the {item} is a two-digit number that allows for multiple JSON files to be
# used to test the same structure.
JSON_TEST_FILES = testdata/json.transport.*
# JSON_TEST_FILES = testdata/json.transport.InventoryResource.01

KERNEL_NAME := $(shell uname -s)
ifeq ($(KERNEL_NAME),Linux)
    JANSSON_LINK_FLAGS += -Wl,-L${JANSSON_LIB_DIRECTORY} -ljansson -Wl,-R${JANSSON_LIB_DIRECTORY}
endif
ifeq ($(KERNEL_NAME),Darwin)
    # The linker -rpath option (-Wl,-R... as it may appear on the compiler commmand line
    # on other platforms) is apparently built into dynamic libraries on Darwin (MacOS),
    # so we can't use (but don't need) -Wl,-R... on this platform.
    JANSSON_LINK_FLAGS += -Wl,-L${JANSSON_LIB_DIRECTORY} -ljansson
endif

# The current definition here is a placeholder for whatever we actually
# want to use according to some sort of project-standard file tree.
BUILD_TARGET_DIRECTORY = BUILD

# We currently specify "-g" to assist in debugging and possibly also in memory-leak detection.
CFLAGS = -std=c11 -g
CC = gcc $(CFLAGS)

# Principal targets here are:
#
#     make test       (compiles code, runs unit tests)
#     make check      (runs unit tests under valgrind to check for memory leaks)
#     make fullcheck  (checks for memory leak with full detail of where they occur)

default	: all

all	: gotocjson

${BUILD_TARGET_DIRECTORY}	:
	mkdir -p ${BUILD_TARGET_DIRECTORY}

test	: ${BUILD_TARGET_DIRECTORY}/unittest
	${BUILD_TARGET_DIRECTORY}/unittest

# To test each individual JSON file, we must identify the nature of that file and compile
# specific code that understands the particular data structure involved, calling just the
# conversion routines that are tailored for that one data structure.  To make it easier to
# add more test cases over time, here we dynamically construct the C test code.  That way,
# you don't have have to edit any C code yourself.  All you have to do is provide a new
# JSON-string test file using an appropriate filename.
tests	: ${JSON_TEST_FILES} _c_code/testjson.c ${BUILD_TARGET_DIRECTORY}/convert_go_to_c.o ${BUILD_TARGET_DIRECTORY}/setup.o ${BUILD_TARGET_DIRECTORY}/subseconds.o ${BUILD_TARGET_DIRECTORY}/transport.o Makefile
	for json_file in ${JSON_TEST_FILES}; do			\
	    echo "================================================================================================";	\
	    echo Processing json test file $$json_file;		\
	    package=$${json_file#testdata/json.};		\
	    structure=$${package#*.};				\
	    item=$${structure#*.};				\
	    package=$${package%%.*};				\
	    structure=$${structure%%.*};			\
	    # echo "package   = '$$package'";			\
	    # echo "structure = '$$structure'";			\
	    # echo "item      = '$$item'";			\
	    ${CC} -c _c_code/testjson.c -o ${BUILD_TARGET_DIRECTORY}/testjson.o -DPackage=$$package -DStructure=$$structure -I${BUILD_TARGET_DIRECTORY} -I. -I${JANSSON_INCLUDE_DIRECTORY};	\
	    ${CC} -o ${BUILD_TARGET_DIRECTORY}/testjson ${BUILD_TARGET_DIRECTORY}/convert_go_to_c.o ${BUILD_TARGET_DIRECTORY}/setup.o ${BUILD_TARGET_DIRECTORY}/subseconds.o ${BUILD_TARGET_DIRECTORY}/transport.o ${BUILD_TARGET_DIRECTORY}/testjson.o ${JANSSON_LINK_FLAGS};	\
	    if ! ${BUILD_TARGET_DIRECTORY}/testjson $$json_file; then			\
	        echo "ERROR:  Test of $$json_file failed!";	\
		exit 1;						\
	    fi;							\
	done

gotocjson	: gotocjson.go
	go build -o gotocjson gotocjson.go

# Run the set of unit tests while looking for full detail on memory leaks.
fullcheck	: ${BUILD_TARGET_DIRECTORY}/unittest
	valgrind --leak-check=full ${BUILD_TARGET_DIRECTORY}/unittest

# Run the set of unit tests while looking for basic info on memory leaks.
check	: ${BUILD_TARGET_DIRECTORY}/unittest
	valgrind ${BUILD_TARGET_DIRECTORY}/unittest

${BUILD_TARGET_DIRECTORY}/unittest	: ${BUILD_TARGET_DIRECTORY}/setup.o ${BUILD_TARGET_DIRECTORY}/subseconds.o ${BUILD_TARGET_DIRECTORY}/transport.o ${BUILD_TARGET_DIRECTORY}/unittest.o ${BUILD_TARGET_DIRECTORY}/convert_go_to_c.o Makefile | ${BUILD_TARGET_DIRECTORY}
	${CC} -o ${BUILD_TARGET_DIRECTORY}/unittest ${BUILD_TARGET_DIRECTORY}/setup.o ${BUILD_TARGET_DIRECTORY}/subseconds.o ${BUILD_TARGET_DIRECTORY}/transport.o ${BUILD_TARGET_DIRECTORY}/unittest.o ${BUILD_TARGET_DIRECTORY}/convert_go_to_c.o ${JANSSON_LINK_FLAGS}

${BUILD_TARGET_DIRECTORY}/unittest.o	: _c_code/unittest.c _c_code/convert_go_to_c.h ${BUILD_TARGET_DIRECTORY}/setup.h ${BUILD_TARGET_DIRECTORY}/subseconds.h ${BUILD_TARGET_DIRECTORY}/transport.h
	${CC} -c _c_code/unittest.c -o $@ -I${BUILD_TARGET_DIRECTORY} -I. -I${JANSSON_INCLUDE_DIRECTORY}

${BUILD_TARGET_DIRECTORY}/convert_go_to_c.o	: _c_code/convert_go_to_c.h _c_code/convert_go_to_c.c
	${CC} -c _c_code/convert_go_to_c.c -o $@ -I${JANSSON_INCLUDE_DIRECTORY}

${BUILD_TARGET_DIRECTORY}/setup.h ${BUILD_TARGET_DIRECTORY}/setup.c	: gotocjson testdata/setup.go | ${BUILD_TARGET_DIRECTORY}
	./gotocjson -d -o ${BUILD_TARGET_DIRECTORY} testdata/setup.go > ,setup-diagnostics

${BUILD_TARGET_DIRECTORY}/subseconds.h ${BUILD_TARGET_DIRECTORY}/subseconds.c	: gotocjson testdata/subseconds.go | ${BUILD_TARGET_DIRECTORY}
	./gotocjson -d -o ${BUILD_TARGET_DIRECTORY} testdata/subseconds.go > ,subseconds-diagnostics

${BUILD_TARGET_DIRECTORY}/transport.h ${BUILD_TARGET_DIRECTORY}/transport.c	: gotocjson testdata/transport.go | ${BUILD_TARGET_DIRECTORY}
	./gotocjson -d -o ${BUILD_TARGET_DIRECTORY} testdata/transport.go > ,transport-diagnostics

${BUILD_TARGET_DIRECTORY}/setup.o	: ${BUILD_TARGET_DIRECTORY}/setup.h ${BUILD_TARGET_DIRECTORY}/setup.c _c_code/convert_go_to_c.h
	${CC} -c ${BUILD_TARGET_DIRECTORY}/setup.c -o $@ -I_c_code -I${JANSSON_INCLUDE_DIRECTORY}

${BUILD_TARGET_DIRECTORY}/subseconds.o	: ${BUILD_TARGET_DIRECTORY}/subseconds.h ${BUILD_TARGET_DIRECTORY}/subseconds.c _c_code/convert_go_to_c.h
	${CC} -c ${BUILD_TARGET_DIRECTORY}/subseconds.c -o $@ -I_c_code -I${JANSSON_INCLUDE_DIRECTORY}

${BUILD_TARGET_DIRECTORY}/transport.o	: ${BUILD_TARGET_DIRECTORY}/setup.h ${BUILD_TARGET_DIRECTORY}/subseconds.h ${BUILD_TARGET_DIRECTORY}/transport.h ${BUILD_TARGET_DIRECTORY}/transport.c _c_code/convert_go_to_c.h
	${CC} -c ${BUILD_TARGET_DIRECTORY}/transport.c -o $@ -I${BUILD_TARGET_DIRECTORY} -I_c_code -I${JANSSON_INCLUDE_DIRECTORY}

${BUILD_TARGET_DIRECTORY}/transport_main.o	: _c_code/transport_main.c _c_code/convert_go_to_c.h ${BUILD_TARGET_DIRECTORY}/setup.h ${BUILD_TARGET_DIRECTORY}/subseconds.h ${BUILD_TARGET_DIRECTORY}/transport.h
	${CC} -c _c_code/transport_main.c -o $@ -I${BUILD_TARGET_DIRECTORY} -I. -I${JANSSON_INCLUDE_DIRECTORY}

# A kind of sample application.  Here just to elucidate the linking aspects of such a build,
# not to produce an actively useful binary program.
transport	: ${BUILD_TARGET_DIRECTORY}/transport

${BUILD_TARGET_DIRECTORY}/transport	: ${BUILD_TARGET_DIRECTORY}/convert_go_to_c.o ${BUILD_TARGET_DIRECTORY}/setup.o ${BUILD_TARGET_DIRECTORY}/subseconds.o ${BUILD_TARGET_DIRECTORY}/transport.o ${BUILD_TARGET_DIRECTORY}/transport_main.o
	${CC} -o $@ ${BUILD_TARGET_DIRECTORY}/convert_go_to_c.o ${BUILD_TARGET_DIRECTORY}/setup.o ${BUILD_TARGET_DIRECTORY}/subseconds.o ${BUILD_TARGET_DIRECTORY}/transport.o ${BUILD_TARGET_DIRECTORY}/transport_main.o ${JANSSON_LINK_FLAGS}

${BUILD_TARGET_DIRECTORY}/enum.h ${BUILD_TARGET_DIRECTORY}/enum.c	: gotocjson testdata/enum.go
	./gotocjson -d -o ${BUILD_TARGET_DIRECTORY} testdata/enum.go > ,enum-diagnostics

enumtest	: ${BUILD_TARGET_DIRECTORY}/enum.h ${BUILD_TARGET_DIRECTORY}/enum.c

clean	:
	rm -f ${BUILD_TARGET_DIRECTORY}/*.o \
	    ${BUILD_TARGET_DIRECTORY}/setup.h ${BUILD_TARGET_DIRECTORY}/setup.c \
	    ${BUILD_TARGET_DIRECTORY}/subseconds.h ${BUILD_TARGET_DIRECTORY}/subseconds.c \
	    ${BUILD_TARGET_DIRECTORY}/transport.h ${BUILD_TARGET_DIRECTORY}/transport.c \
	    ${BUILD_TARGET_DIRECTORY}/enum.h ${BUILD_TARGET_DIRECTORY}/enum.c

realclean	: clean
	rm -f gotocjson ,setup-diagnostics ,enum-diagnostics ,subseconds-diagnostics ,transport-diagnostics \
	    ${BUILD_TARGET_DIRECTORY}/unittest ${BUILD_TARGET_DIRECTORY}/testjson ${BUILD_TARGET_DIRECTORY}/transport

distclean	: realclean
