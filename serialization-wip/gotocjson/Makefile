#
# Makefile for gotocjson
#
# This is a tool that analyzes a Go source file, identifies all the struct typedefs,
# enumeration typdefs, and enumeration constants, and writes out C code that supports
# equivalent enumerations and structs along with associated JSON marshalling and
# unmarshalling that matches the JSON processing (as tweaked by field tags) in the
# original Go code.  This tool may find use when exchanging data between Go and C
# contexts.

JANSSON_LIB_DIRECTORY = ../local/lib

CFLAGS = -std=c11
CC = cc $(CFLAGS)

default	: all

all	: gotocjson

test	: unittest
	unittest

gotocjson	: gotocjson.go
	go build -o gotocjson gotocjson.go

unittest	: config.o milliseconds.o transit.o unittest.o convert_go_to_c.o
	${CC} -o unittest config.o milliseconds.o transit.o unittest.o convert_go_to_c.o -Wl,-L${JANSSON_LIB_DIRECTORY} -ljansson -Wl,-R${JANSSON_LIB_DIRECTORY}

#unittest	: c.o milliseconds.o transit.o unittest.o convert_go_to_c.o
#	${CC} -o unittest c.o milliseconds.o transit.o unittest.o convert_go_to_c.o -Wl,-L${JANSSON_LIB_DIRECTORY} -ljansson -Wl,-R${JANSSON_LIB_DIRECTORY}

#unittest	: c.o m.o transit.o unittest.o convert_go_to_c.o
#	${CC} -o unittest c.o m.o transit.o unittest.o convert_go_to_c.o -Wl,-L${JANSSON_LIB_DIRECTORY} -ljansson -Wl,-R${JANSSON_LIB_DIRECTORY}

unittest.o	: unittest.c convert_go_to_c.h config.h milliseconds.h transit.h
	${CC} -c unittest.c -I../local/include

c.o	: c.c convert_go_to_c.h config.h
	${CC} -c c.c -I../local/include

m.o	: m.c convert_go_to_c.h milliseconds.h
	${CC} -c m.c -I../local/include

t.o	: t.c convert_go_to_c.h config.h milliseconds.h transit.h
	${CC} -c t.c -I../local/include

convert_go_to_c.o	: convert_go_to_c.h convert_go_to_c.c
	${CC} -c convert_go_to_c.c -I../local/include

config.h config.c	: gotocjson config.go
	gotocjson config.go > ,c

milliseconds.h milliseconds.c	: gotocjson milliseconds.go
	gotocjson milliseconds.go > ,m

transit.h transit.c	: gotocjson transit.go
	gotocjson transit.go > ,t

config.o	: config.h config.c
	${CC} -c config.c -I../local/include

milliseconds.o	: milliseconds.h milliseconds.c
	${CC} -c milliseconds.c -I../local/include

transit.o	: config.h milliseconds.h transit.h transit.c
	${CC} -c transit.c -I../local/include

transit	: config.o milliseconds.o transit.o
	${CC} -o transit config.o milliseconds.o transit.o -Wl,-L${JANSSON_LIB_DIRECTORY} -ljansson -Wl,-R${JANSSON_LIB_DIRECTORY}

clean	:
	rm -f gotocjson *.o config.h config.c milliseconds.h milliseconds.c transit.h transit.c

realclean	: clean
	rm -f ,c ,e ,g ,m ,t foo.o foo unittest
