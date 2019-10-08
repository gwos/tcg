# Top-level Makefile for TNG code.

# Here, we do not assume that the jansson library is available from an
# OS-provided package.  That's because such a package is likely to be out
# of date.  Instead, we download and install an appropriate package here.

JANSSON_VERSION = 2.12

all	: local/lib/libjansson.so

# For no good reason, the upstream code does not follow the universal
# standard for naming the release tarball after the top-level directory
# that contains it.  So we must play a game here to clean that up.
jansson-${JANSSON_VERSION}.tar.gz	:
	wget https://github.com/akheron/jansson/archive/v${JANSSON_VERSION}.tar.gz
	mv v${JANSSON_VERSION}.tar.gz jansson-${JANSSON_VERSION}.tar.gz

jansson-${JANSSON_VERSION}/configure	: jansson-${JANSSON_VERSION}.tar.gz
	rm -rf jansson-${JANSSON_VERSION}
	tar xfz jansson-${JANSSON_VERSION}.tar.gz
	cd jansson-${JANSSON_VERSION}; autoreconf -i

local/lib/libjansson.so	: jansson-${JANSSON_VERSION}/configure
	cd jansson-${JANSSON_VERSION}; ./configure --prefix=${PWD}/local
	cd jansson-${JANSSON_VERSION}; make
	cd jansson-${JANSSON_VERSION}; make install

clean	:
	rm -rf jansson-${JANSSON_VERSION}

distclean	: clean
	rm -rf local v${JANSSON_VERSION}.tar.gz jansson-${JANSSON_VERSION}.tar.gz 
