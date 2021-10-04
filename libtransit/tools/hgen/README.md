
# H-files generator

Inspired by headscan
	https://cs.opensource.google/go/go/+/refs/tags/go1.17.1:src/go/doc/headscan.go

There is a simple tool for generating header files for C lang API.
Based on scanned sources it creates `<packagename>.h` files with definitions for exported constants.


## Usage

Just build and run with `--folders` option.
```
$ go build . && ./hgen --folders ../../../transit && head transit.h
7:24PM INF main.go:55 > starting cfg={"Folders":["../../../transit"],"NoPrefix":false,"Verbose":2}
7:24PM INF main.go:76 > create file fname=transit.h
#ifndef TRANSIT_H
#define TRANSIT_H

/* CloudHub Compute Types */
#define TRANSIT_QUERY "Query"
#define TRANSIT_REGEX "Regex"
#define TRANSIT_SYNTHETIC "Synthetic"
#define TRANSIT_INFORMATIONAL "Informational"
#define TRANSIT_PERFORMANCE "Performance"
#define TRANSIT_HEALTH "Health"
```

There are few options supported
```
$ go build . && ./hgen --help
Usage of ./hgen:
      --folders strings   Folder pathes
      --no-prefix         Omit prefixing constant name
      --verbose int8      Verbose level 0..3 (default 2)
pflag: help requested
```

Omit build step with `go run`
```
$ go run ./libtransit/tools/hgen/ --folders ./transit
12:14PM INF libtransit/tools/hgen/main.go:55 > starting cfg={"Folders":["./transit"],"NoPrefix":false,"Verbose":2}
12:14PM INF libtransit/tools/hgen/main.go:76 > create file fname=transit.h
```
