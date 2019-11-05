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

default	: all

all	: gotocjson

test	: tjunit
	tjunit

gotocjson	: gotocjson.go
	go build -o gotocjson gotocjson.go

tjunit	: tj.o tjunit.o
	cc -o tjunit tj.o tjunit.o -Wl,-L${JANSSON_LIB_DIRECTORY} -ljansson -Wl,-R${JANSSON_LIB_DIRECTORY}

tjunit.o	: tjunit.c config.h milliseconds.h transit.h
	cc -c tjunit.c -I../local/include

tj.o	: tj.c config.h milliseconds.h transit.h
	cc -c tj.c -I../local/include

config.h config.c	: gotocjson config.go
	gotocjson config.go > ,c

milliseconds.h milliseconds.c	: gotocjson milliseconds.go
	gotocjson milliseconds.go > ,m

transit.h transit.c	: gotocjson transit.go
	gotocjson transit.go > ,t

config.o	: config.c
	cc -c config.c -I../local/include

milliseconds.o	: milliseconds.c
	cc -c milliseconds.c -I../local/include

transit.o	: transit.c config.h milliseconds.h transit.h
	cc -c transit.c -I../local/include

transit	: config.o milliseconds.o transit.o
	cc -o transit config.o milliseconds.o transit.o -Wl,-L${JANSSON_LIB_DIRECTORY} -ljansson -Wl,-R${JANSSON_LIB_DIRECTORY}

clean	:
	rm -f gotocjson *.o config.h config.c milliseconds.h milliseconds.c transit.h transit.c

realclean	: clean
	rm -f ,c ,e ,g ,m ,t foo.o foo tjunit
