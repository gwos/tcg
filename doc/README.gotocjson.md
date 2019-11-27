## The `gotocjson` code-conversion tool

This tool is designed to absorb Go source code and emit corresponding C code that can
serialize and deserialize equivalent data structures to/from JSON strings.  It finds use
where complex data must be exchanged between Go code and C code in a portable fashion,
whether or not the Go code and C code are running in the same process.

Generally speaking, Go enumeration values are expressed in the JSON strings as strings
rather than integer enumeration values.  The generated C code automatically handles
translation of those string values to/from C-code integer enumeration values.

In general, a Go-code enumeration must be expressed by a `const` block, wherein the first
element of the block is declared with the specific named Go type of the enumeration.
This allows the conversion tool to identify which `const` block contains the values
for each enumeration type.

At least the following kinds of Go datatypes are supported:

```
foo
*foo
[]foo
**foo
*[]foo
[]*foo
[][]foo
*[]*foo
```

If additional complexity is encountered as the tool is used in practice, it will need
to be extended.

### Building the `gotocjson` tool

Certain OS packages are needed.  In particular, you need the following before beginning
(these are not the actual package names):

```
make
tar
autoconf
libtool
Go
GCC (a recent version that supports C99 or C11 by default; I forget which)
binutils (for things like "ar")
```

From a standing start, as checked out from Git, follow the yellow brick road:

```
# Drop into the top-level directory for the TNG repository,
# whereever you have checked it out.  This is just an example.
cd github/gwos/tng

# Make the Jansson library.  Header files and compiled library files
# for use with the gotocjson code will be installed under the local/
# subdirectory within the checked-out TNG repository.  The Makefile
# for the C test code we are generating knows to look there.
make

# Drop into the bottom-level tool directory.  This is where the
# conversion program, its test programs, and associated test JSON
# files are developed and maintained.
cd gotocjson

# Compile the Go conversion tool (gotocjson).
make

# Compile the unit-test program.
make unittest

# Build everything necessary, and run simple hardcoded unit tests.
make test

# Build the unit-test program, and run it under valgrind to check
# for memory leaks.
make check

# Build the unit-test program, and run it under valgrind with extra
# options to spill out code-coordinate details of memory leaks.
make fullcheck

# Build everything necessary, and run a variety of tests using
# a set of standalone JSON files in the build directory.
make tests

# Cleanup targets.
make clean
make realclean
```

### Tool usage

To see usage, run the `gotocjson` program with no command-line arguments.

The tool is designed to convert just one Go source file at a time.  It creates a pair of
`.h` and `.c` files which are named after the package which is declared in the file,
not after the Go source-code filename.

In typical production use, you will run the tool with the `-o directory` option, to
place the generated `.h` and `.c` files in some target-code build directory which is
separate from the place where the original Go code is located.

### Diagnostic output

The tool can generate a lot of extra diagnostic output which has been useful during
development to expose the innards of what is happening during Go-code parsing,
C-code generation, and testing.  It's all very rough, and there are no current plans
to improve the formatting or content of that output.  It is normally suppressed, but
may be enabled by using the `-d` option on the command line.  Doing so may help in the
process of ensuring that all necessary Go constructs are covered in all cases (e.g.,
support for JSON field renaming and the `omitenpty` option in Go struct field tags,
for all Go datatypes).

If code is thrown at the tool that it cannot yet handle, it should panic instead
of silently generating bad output code -- although by then there may already be some
amount of generated code in the `.h` and `.c` files.  There has not yet been care taken
to ensure that some other notion of failure will appear in all cases when diagnostic
output is suppressed.

The exit status from the `gotocjson` tool should reflect the success or failure of
the conversion.

### Tool files

There is currently a small bit of debris left around from the development and test
process.  The most important files are:

  * **`doc/README.md`**

    The file you are now reading.

  * **`Makefile`**

    A makefile used to compile the C-language Jansson library used for generating and
    parsing JSON strings.

  * **`jansson-2.12.tar.gz`**

    The upstream Jansson library, which is used for parsing and generating JSON strings.
    It doesn't really belong in our checked-in code.  If this file is locally missing,
    the sibling `Makefile` will download the original file and rename it to this
    sensible name (the Jansson folks didn't follow the usual standard conventions,
    which makes for confusion).

  * **`gotocjson/Makefile`**

    A makefile used to compile the conversion program and the unit-test program.
    As part of that build, the conversion program is run on some of the `.go` files
    in this directory, to generate corresponding C header and code files.

  * **`gotocjson/gotocjson.go`**

    The Go-to-C conversion tool.  For development/test purposes, this is built and
    invoked via the `Makefile` in that same directory.

    The program works by running the standard Go compiler's own parser, then walking
    the Abstract Syntax Tree that results to find the parts we are interested in.
    Those parts are picked through to extract package imports, type definitions,
    const blocks, and structure definitions with their field name, field types, and
    field tags.  That information is saved in other data structures which are more
    convenient to use during C-code generation.

    Because Go allows forward references to as-yet-undefined types and objects, while
    C does not, the conversion program runs a topological sort on the relationships
    between all the data types, so they can be output in a form which closely mirrors
    that of the original Go code but also satisfies the constraints of the C compiler.

    Output is split between a `.h` file, containing only declarations and preprocessor
    definitions, and a `.c` file, containing both enumeration values represented as
    strings and the generated conversion functions.

  * **`gotocjson/convert_go_to_c.h`**

    **`gotocjson/convert_go_to_c.c`**

    These files contain code that is needed for conversion of C data structures to
    and from JSON, but that is not specific to the particular Go-package code that is
    being targeted.  The common parts have been factored out into these files so it
    can be represented only once in a compiled application.

  * **`gotocjson/setup.go`**

    This is a mostly-frozen copy of an early version of the TNG `config.go` file,
    with possible later local edits.  This test file is renamed to avoid confusion
    with the production file, and placed here to insulate against random changes in
    the production file.  It is part of a testbed to show that we have support for
    all features we need to handle.

  * **`gotocjson/subseconds.go`**

    This is a mostly-frozen copy of an early version of the TNG `milliseconds.go` file,
    with possible later local edits.  This test file is renamed to avoid confusion
    with the production file, and placed here to insulate against random changes in
    the production file.  It is part of a testbed to show that we have support for
    all features we need to handle.

  * **`gotocjson/transport.go`**

    This is a mostly-frozen copy of an early version of the TNG `transit.go` file,
    with possible later local edits.  This test file is renamed to avoid confusion
    with the production file, and placed here to insulate against random changes in
    the production file.  It is part of a testbed to show that we have support for
    all features we need to handle.

  * **`gotocjson/unittest.c`**

    This is the basic unit-test program.  It contains a bunch of hardcoded JSON
    strings that we decode into C structures, then encode those back into JSON and
    compare against the original JSON strings to verify that we get back exactly what
    we started with.

  * **`gotocjson/testjson.c`**

    This is a kind of template source code that gets customized at build time to handle
    some particular package and data structure, for test purposes.  The `Makefile`
    holds the instructions for that customization, based on the presence of standalone
    JSON test files in the `gotocjson` directory.

  * **`gotocjson/enum.go`**

    This is a test file for the `gotocjson` tool's Go-language parsing and handling
    of enumerations.  In particular for instance, this would be used for proving that
    we have adequate support for `iota` in `const` blocks.

Some of the other files represent older development code that has not yet been
cleaned up.

### Current limitations

`iota` is not yet fully supported in the processing of `const` blocks.  It is likely
that we will only be able to support very simple usage patterns of `iota`.

The current iteration of the conversion tool collects the struct field tags as it
walks the parse tree and saves them away for later use, but the tags are not yet fully
used to modify the generation or parsing of JSON strings in the generated C code.
Use cases covered by the test Go code are covered, but support for other datatypes
must still be added.  That will take a later iteration of development.

One issue with the current playpen code is that when parsing JSON into C structures,
we encounter strings that end up being referenced by the C structures.  However,
those strings will disappear when we destroy the JSON (Jansson) objects that contain
them after parsing the incoming JSON string.  For that reason, at the moment we have
some `json_decref(json);` calls commented out.  We will either need to `strdup()`
such strings when assigning them to C structures to sidestep such aliasing, or modify
the decoding API (I believe this has been done now) to also return the relevant JSON
objects so their lifetime can be controlled by the calling application rather than
just the conversion routines.

At the time of this writing, there is a probable bug in the C-code error handling of
some JSON string decoding, wherein some error text is referenced after it has likely
been deleted.

Specific details on other issues and further code improvements are documented in the
TASKS file.

At the time of this writing, the `doc/*` files have not yet been cleaned up to reflect
the current status of this tool.

The internal processing of Go datatypes is hardcoded on a case-by-case basis, taking into
account the particulars of what has been seen in actual Go parse trees.  If additional
use cases are encountered, the tool will need to be extended.  Partly because this is
the first cut of the program, and partly because there is a lot of uniqueness in the
handling of each of the use cases encountered so far, we have not yet put any effort
into refactoring the datatype handling to make it more general and automatically adapt
to further complications (e.g., more levels of maps, slices, and pointers).

### Usage of generated C-code routines

C code which is generated from Go data structures will declare a set of associated
routines.  Their intended usage is documented in this section.  In the following
discussion, assume that you have a `pkg.go` Go source file containing the Go package
named "`pkg`", and a structure within that package named "`Struct`".  These routines
will be generated or `#define`d, and their respective declarations and definitions
will be found in the generated `pkg.h` file.

A C application will use two different code patterns when copying data between itself
and a Go partner.

#### Sending data from C code

  * To send data, a pkg.Struct will be dynamically allocated by
    `make_empty_pkg_Struct()`.  Its fields will be filled in by the application, either
    directly (including fields for any embedded structures) or by dynamically creating
    subsidiary objects with equivalent allocation routines for those structures
    and saving the pointers to those subobjects into the growing data structure.
    It is assumed that all strings in the ptr.Struct object (and recursively, in its
    subobjects) will be dynamically allocated.  The C code is expected to understand
    the various "List" forms and populate them appropriately, including calling
    `make_empty_pkg_Struct_array()` to dynamically allocate the necessary arrays of
    list elements.

  * Once the pkg.Struct is fully populated, the application will call
    `pkg_Struct_ptr_as_JSON_str()` to convert the assembly into a dynamically-allocated
    string containing the JSON that will be passed to the Go code via some mechanism
    which is outside the scope of the conversion tool itself.

  * Once the data transfer is complete, the application must `free()` the JSON string
    used to send data to the Go partner.

  * Once the data transfer is complete, the application must also call
    `free_pkg_Struct_ptr_tree(pkg_Struct_ptr, NULL)` to de-allocate the entire assembly.
    In this case, since the pkg.Struct was manually assembled, there are no associated
    Jansson objects, so that second parameter must be `NULL`.

#### Receiving data into C code

  * To receive data, the application must interact with its Go partner to fetch a
    dynamically-allocated JSON string.

  * The application will call `JSON_str_as_pkg_Struct_ptr()` to convert the JSON
    string to a dynamically-allocated pkg.Struct structure (and its subobjects).
    The application can then walk the structure as needed to extract the data it
    cares about.

  * [This step is not completely clear yet.  We will clarify this step once we have
    implemented the receipt of some data from a Go partner.]  Once the data transfer
    is complete, the application must `free()` the JSON string used to receive data
    from the Go partner.

  * Once the data transfer is complete, the application must also call
    `free_pkg_Struct_ptr_tree(pkg_Struct_ptr, json)` to de-allocate the entire assembly.
    In this case, since the pkg.Struct was automatically assembled from intermediate
    Jansson objects, that second parameter must be non-`NULL`, pointing to those objects.

#### Routines for application code

  * **`pkg_Struct *make_empty_pkg_Struct();`**

    **`pkg_Struct *make_empty_pkg_Struct_array(n);`**

    These two routines provide for the dynamic allocation of memory for a single
    pkg.Struct object, and for the creation of an array of such objects.  The created
    objects are intentionally all zeroed out, to provide the best general approximation
    of having each member of the object default to its own native zero value.

  * **`char *pkg_Struct_ptr_as_JSON_str(const pkg_Struct *pkg_Struct_ptr);`**

    This routine converts the data in a pkg.Struct object into a JSON string.  It is
    the principal routine intended to be called from application code for that task.

  * **`pkg_Struct *JSON_str_as_pkg_Struct_ptr(const char *json_str, json_t **json);`**

    This routine converts a JSON string to a pkg.Struct object and its subsidiary
    objects.  The address of a "`json_t *`" pointer must be passed to this routine for
    use in creating intermediate Jansson objects and tracking the memory used during
    the conversion.  Memory for strings in the generated pkg.Struct object is shared
    between that object (and its sub-objects) and the Jansson objects.

    Exposing the Jansson objects at this level is something of an intrusion of the
    implementation into the definition of the API.  This is considered acceptable
    because otherwise lots of addiitional memory allocation/deallocation and copying
    would be required, for data that is likely to be of only transient value.

  * **`void free_pkg_Struct_ptr_tree(pkg_Struct *pkg_Struct_ptr, json_t *json);`**

    This routine is intended for use by application code, to free a single pkg.Struct
    object and all of its subsidiary objects in one step.  If the pkg.Struct object was
    obtained from a call to `JSON_str_as_pkg_Struct_ptr()`, the Jansson objects that
    were created in that process must be passed back here, so they can be deallocated
    at the same time.  Conversely, if the pkg.Struct object was manually constructed,
    a `NULL` should be passed for the "`json`" argument.

    This routine assumes that ALL strings, subsidiary objects, and the pkg.Struct
    object itself have been dynamically allocated.  All of them will be recursively
    deallocated in the one call.

#### Internal routines, not intended to be called by application code

  * **`bool is_pkg_Struct_ptr_zero_value(const pkg_Struct *pkg_Struct_ptr);`**

    This routine checks to see whether the content of the specified pkg.Struct consists
    fully of zero values for all of its fields.  This is important internally within
    the code that converts objects to JSON, in support of the `omitempty` property of
    Go struct field tags.

  * **`void destroy_pkg_Struct_ptr_tree(pkg_Struct *pkg_Struct_ptr, json_t *json, bool free_pointers);`**

    This is an internal routine used by the JSON conversion code to recursively walk
    a given pkg.Struct object and deallocate all the associated memory.  It should
    not be called directly from application code.

  * **`json_t *pkg_Struct_ptr_as_JSON_ptr(const pkg_Struct *pkg_Struct_ptr);`**

    This is an internal routine that converts a pkg.Struct object into equivalent Jansson
    structures, which can then be serialized to an actual JSON string for data exchange
    with external code.  This routine is not intended to be called by application code.

  * **`pkg_Struct *JSON_as_pkg_Struct_ptr(json_t *json);`**

    This is an internal routine that converts Jansson structures into an equivalent
    pkg.Struct object, which can then be examined by application code.  This routine
    is not intended to be called by application code.

