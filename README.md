# tng
The New Groundwork Transit connectors (feeders). TNG contains two sub-systems/packages:

1. Transit agent - connects and sends metrics to Groundwork Monitor 
2. Controller service - an http server for external control of agent
 
Building
--------
```
cd go/src/github/com/gwos/tng
go build .
```

Running 
--------
```
cd go/src/github/com/gwos/tng
go run .
```


## C-API

Requirements
------------
* The jansson library should be installed.
* The libjansson-dev library package should be installed for development.

Building and running tests
--------------------------
```
gcc transit-c/*.c -o ./transit-c.test -ljansson && ./transit-c.test
```

Building and running tests in Docker container
----------------------------------------------
```
docker build -t groundworkdevelopment/tng .
docker run --rm --name tng-test-c groundworkdevelopment/tng sh -c "gcc transit-c/*.c -o ./transit-c.test -ljansson && ./transit-c.test"
```

Development
-----------
The `clang-format` tool is used for formatting with the "Google" style option.
* https://clang.llvm.org/docs/ClangFormat.html
* https://clang.llvm.org/docs/ClangFormatStyleOptions.html

```
cd transit-c
clang-format -style=Google -i *.c *.h
```
