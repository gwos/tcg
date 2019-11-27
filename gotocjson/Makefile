#
# Makefile for gotocjson
#
# This is the Makefile for a tool that analyzes a Go source file, identifies all
# the struct typedefs, enumeration typdefs, and enumeration constants, and writes
# out C code that supports equivalent enumerations and structs along with associated
# JSON marshalling and unmarshalling that matches the JSON processing (as tweaked by
# field tags) in the original Go code.  This tool may find use when exchanging data
# between Go and C contexts.

# The enum.go file in this directory is used for ad-hoc testing of the gotocjson tool,
# during development.

# The copies of setup.go, subseconds.go, and transport.go in this directory are NOT the
# production copies.  Instead, they are test copies, intended to freeze stable versions of
# enumerations and data structures that we can reliably use for testing the gotocjson tool
# ehile developint it.

# The convert_go_to_c.h and convert_go_to_c.c files comprise an integral part of the
# implementation of the gotocjson tool.  They provide routines which are not derived
# from converted Go code, but which are necessary to support the converted Go code.

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
JSON_TEST_FILES = json.transport.*
# JSON_TEST_FILES = json.transport.InventoryResource.01

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

test	: unittest
	./unittest

# To test each individual JSON file, we must identify the nature of that file and compile
# specific code that understands the particular data structure involved, calling just the
# conversion routines that are tailored for that one data structure.  To make it easier to
# add more test cases over time, here we dynamically construct the C test code.  That way,
# you don't have have to edit any C code yourself.  All you have to do is provide a new
# JSON-string test file using an appropriate filename.
tests	: ${JSON_TEST_FILES} testjson.c convert_go_to_c.o setup.o subseconds.o transport.o Makefile
	for json_file in ${JSON_TEST_FILES}; do			\
	    echo "================================================================================================";	\
	    echo Processing json test file $$json_file;		\
	    package=$${json_file#json.};			\
	    structure=$${package#*.};				\
	    item=$${structure#*.};				\
	    package=$${package%%.*};				\
	    structure=$${structure%%.*};			\
	    # echo "package   = '$$package'";			\
	    # echo "structure = '$$structure'";			\
	    # echo "item      = '$$item'";			\
	    ${CC} -c testjson.c -DPackage=$$package -DStructure=$$structure -I${JANSSON_INCLUDE_DIRECTORY};		\
	    ${CC} -o testjson convert_go_to_c.o setup.o subseconds.o transport.o testjson.o ${JANSSON_LINK_FLAGS};	\
	    if ! testjson $$json_file; then			\
	        echo "ERROR:  Test of $$json_file failed!";	\
		exit 1;						\
	    fi;							\
	done

gotocjson	: gotocjson.go
	go build -o gotocjson gotocjson.go

# Run the set of unit tests while looking for full detail on memory leaks.
fullcheck	: unittest
	valgrind --leak-check=full ./unittest

# Run the set of unit tests while looking for basic info on memory leaks.
check	: unittest
	valgrind ./unittest

unittest	: setup.o subseconds.o transport.o unittest.o convert_go_to_c.o Makefile
	${CC} -o unittest setup.o subseconds.o transport.o unittest.o convert_go_to_c.o ${JANSSON_LINK_FLAGS}

unittest.o	: unittest.c convert_go_to_c.h setup.h subseconds.h transport.h
	${CC} -c unittest.c -I${JANSSON_INCLUDE_DIRECTORY}

convert_go_to_c.o	: convert_go_to_c.h convert_go_to_c.c
	${CC} -c convert_go_to_c.c -I${JANSSON_INCLUDE_DIRECTORY}

setup.h setup.c	: gotocjson setup.go
	./gotocjson -d setup.go > ,setup-diagnostics

subseconds.h subseconds.c	: gotocjson subseconds.go
	./gotocjson -d subseconds.go > ,subseconds-diagnostics

transport.h transport.c	: gotocjson transport.go
	./gotocjson -d transport.go > ,transport-diagnostics

setup.o	: setup.h setup.c convert_go_to_c.h
	${CC} -c setup.c -I${JANSSON_INCLUDE_DIRECTORY}

subseconds.o	: subseconds.h subseconds.c convert_go_to_c.h
	${CC} -c subseconds.c -I${JANSSON_INCLUDE_DIRECTORY}

transport.o	: setup.h subseconds.h transport.h transport.c convert_go_to_c.h
	${CC} -c transport.c -I${JANSSON_INCLUDE_DIRECTORY}

transport_main.o	: transport_main.c
	${CC} -c transport_main.c -I${JANSSON_INCLUDE_DIRECTORY}

# A kind of sample application.  Here just to elucidate the linking aspects of such a build,
# not to produce an actively useful binary program.
transport	: convert_go_to_c.o setup.o subseconds.o transport.o transport_main.o
	${CC} -o transport convert_go_to_c.o setup.o subseconds.o transport.o transport_main.o ${JANSSON_LINK_FLAGS}

enum.h enum.c	: gotocjson enum.go
	./gotocjson -d enum.go > ,enum-diagnostics

clean	:
	rm -f gotocjson *.o setup.h setup.c subseconds.h subseconds.c transport.h transport.c enum.h enum.c

realclean	: clean
	rm -f ,setup-diagnostics ,enum-diagnostics ,subseconds-diagnostics ,transport-diagnostics unittest transport testjson
