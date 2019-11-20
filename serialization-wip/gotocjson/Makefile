#
# Makefile for gotocjson
#
# This is the Makefile for a tool that analyzes a Go source file, identifies all
# the struct typedefs, enumeration typdefs, and enumeration constants, and writes
# out C code that supports equivalent enumerations and structs along with associated
# JSON marshalling and unmarshalling that matches the JSON processing (as tweaked by
# field tags) in the original Go code.  This tool may find use when exchanging data
# between Go and C contexts.

# The Jansson library is used for parsing and generating JSON strings from C-language
# code.  We use a locally-built copy of this library because it is fairly likely to be
# more up-to-date than a system-provided copy.

# Location of the Jansson library include files.
JANSSON_INCLUDE_DIRECTORY = ../local/include

# Location of the compiled Jansson library, for linking purposes.
JANSSON_LIB_DIRECTORY = ../local/lib

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

gotocjson	: gotocjson.go
	go build -o gotocjson gotocjson.go

fullcheck	: unittest
	valgrind --leak-check=full ./unittest

check	: unittest
	valgrind ./unittest

unittest	: config.o milliseconds.o transit.o unittest.o convert_go_to_c.o Makefile
	${CC} -o unittest config.o milliseconds.o transit.o unittest.o convert_go_to_c.o ${JANSSON_LINK_FLAGS}

# Obsolete, still-in-development entry.
# Will be cleaned out when we also delete c.c, m.c, and t.c code and the support for it below.
#unittest	: c.o milliseconds.o transit.o unittest.o convert_go_to_c.o Makefile
#	${CC} -o unittest c.o milliseconds.o transit.o unittest.o convert_go_to_c.o ${JANSSON_LINK_FLAGS}

# Obsolete, still-in-development entry.
# Will be cleaned out when we also delete c.c, m.c, and t.c code and the support for it below.
#unittest	: c.o m.o transit.o unittest.o convert_go_to_c.o Makefile
#	${CC} -o unittest c.o m.o transit.o unittest.o convert_go_to_c.o ${JANSSON_LINK_FLAGS}

unittest.o	: unittest.c convert_go_to_c.h config.h milliseconds.h transit.h
	${CC} -c unittest.c -I${JANSSON_INCLUDE_DIRECTORY}

# c.c, m.c, and t.c were early manually-written templates for the conversion code, used
# as patterns and substitutes for the automatic conversions until those became available.

c.o	: c.c convert_go_to_c.h config.h
	${CC} -c c.c -I${JANSSON_INCLUDE_DIRECTORY}

m.o	: m.c convert_go_to_c.h milliseconds.h
	${CC} -c m.c -I${JANSSON_INCLUDE_DIRECTORY}

t.o	: t.c convert_go_to_c.h config.h milliseconds.h transit.h
	${CC} -c t.c -I${JANSSON_INCLUDE_DIRECTORY}

convert_go_to_c.o	: convert_go_to_c.h convert_go_to_c.c
	${CC} -c convert_go_to_c.c -I${JANSSON_INCLUDE_DIRECTORY}

config.h config.c	: gotocjson config.go
	./gotocjson config.go > ,c

milliseconds.h milliseconds.c	: gotocjson milliseconds.go
	./gotocjson milliseconds.go > ,m

transit.h transit.c	: gotocjson transit.go
	./gotocjson transit.go > ,t

config.o	: config.h config.c convert_go_to_c.h
	${CC} -c config.c -I${JANSSON_INCLUDE_DIRECTORY}

milliseconds.o	: milliseconds.h milliseconds.c convert_go_to_c.h
	${CC} -c milliseconds.c -I${JANSSON_INCLUDE_DIRECTORY}

transit.o	: config.h milliseconds.h transit.h transit.c convert_go_to_c.h
	${CC} -c transit.c -I${JANSSON_INCLUDE_DIRECTORY}

transit	: config.o milliseconds.o transit.o
	${CC} -o transit config.o milliseconds.o transit.o ${JANSSON_LINK_FLAGS}

clean	:
	rm -f gotocjson *.o config.h config.c milliseconds.h milliseconds.c transit.h transit.c

realclean	: clean
	rm -f ,c ,e ,g ,m ,t foo.o foo unittest
