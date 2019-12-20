// gotocjson is a source-code converter.  It takes a Go source file containing a
// bunch of enumeration and structure declarations, and turns them into equivalent
// C code, including not just declarations but also JSON encoding and decoding
// routines that respect the field tags specified in the Go code.  This mechanism
// will both allow for rapid automated changes on the C side whenever we need to
// revise the Go interface, and ensure that the conversion rouines are up-to-date
// and accurate.
package main

// All operations in this program assume that the source code under
// inspection fits easily into memory all at once; there is no need
// for any type of streaming in the handling of the source code.

// FIX LATER:
//
// (*) Consider implementing a layer of routines that effectively provide what C++
//     calls "placement new" semantics, meaning the caller would be responsible for
//     allocating the principal C-structure memory block and the called routine
//     would then take a pointer to that block and fill it in.  Figure out whether
//     such an approach might eliminate some amount of allocation and deallocation
//     at the top and bottom of the hierarchy, enough to make a switch worthwhile.
//
// (*) Consider optionally using the Custom Memory Allocation routines of Jansson
//     to implement some sort of invocation-count and execution-time processing,
//     so we can measure the amount of effort involved at that level of the
//     implementation of JSON encoding and decoding.

import (
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"time"
	"unicode"
)

// Argument parsing in Go seems to be something of a mess.  The distributed Go language provides
// an {import "flag"} package (https://golang.org/pkg/flag/), but for no good reason it ignores
// longstanding conventions for how to construct long argument option names.  Alternatives for
// getopt-like behavior include:
//
//     import "github.com/pborman/getopt"     // version 1
//     import "github.com/pborman/getopt/v2"  // version 2, preferred over version 1
//     import "github.com/pborman/options"    // improvement over "github.com/pborman/getopt/v2"
//     import "github.com/mkideal/cli"
//     import "github.com/galdor/go-cmdline"
//     import "gopkg.in/alecthomas/kingpin.v2"
//     import "github.com/docopt/docopt-go"
//     import "github.com/jessevdk/go-flags"
//
// See:
//
//     https://godoc.org/github.com/pborman/getopt
//     https://godoc.org/github.com/pborman/getopt/v2
//     https://godoc.org/github.com/pborman/options
//     https://groups.google.com/forum/#!topic/golang-nuts/i8Qw9go6CnQ
//     https://github.com/mkideal/cli
//     https://github.com/galdor/go-cmdline
//     http://snowsyn.net/2016/08/11/parsing-command-line-options-in-go/
//     https://github.com/alecthomas/kingpin
//     http://docopt.org/
//     https://github.com/docopt/docopt.go
//     https://godoc.org/github.com/jessevdk/go-flags
//
// Not wanting to get into that complexity at the moment, I have recorded those resources above for
// future reference, but for now I am resricting this program to use only short-option command-line
// arguments, and very simple parsing.

// Globals.

var PROGRAM = "gotocjson"
var VERSION = "0.2.0"

var bad_args = false
var exit_early = false
var print_help = false
var print_version = false
var print_diagnostics = false
var print_errors = true
var input_filepath = ""
var output_directory = ""
var diag_file = os.Stdout

func show_help() {
	fmt.Fprintf(os.Stdout,
		`usage:  %[1]s [-d] [-o outdir] filename.go
	%[1]s -h
	%[1]s --help
	%[1]s --version
where:  -d           produces diagnostic output on the stdout stream
	-o outdir    specifies the directory where the generated .h and .c files
		     will be placed; default is the same directory in which the
		     filename.go file lives
	filename.go  path to the source-code file you wish to transform into C code
	-h           prints this usage message
	--help       prints this usage message
	--version    prints the version of this program
`, PROGRAM)
}

func show_version() {
	fmt.Fprintf(os.Stdout, "%s version %s\n", PROGRAM, VERSION)
}

// Because of the mess that is Go's handling of command-line arguments, we centralize the parsing of
// those options so just this one routine will need replacement if/when we switch the implementation
// to some other package.
//
func parse_args() {
	// Grab the full set of command-line arguments, so we can more readily manipulate them.
	cmd_args := os.Args
	// Skip the program name.
	cmd_args = cmd_args[1:]
	for {
		if len(cmd_args) == 0 || cmd_args[0] == "-h" || cmd_args[0] == "--help" {
			print_help = true
			exit_early = true
			break
		}
		if cmd_args[0] == "--version" {
			print_version = true
			exit_early = true
			break
		}
		if cmd_args[0] == "-d" {
			print_diagnostics = true
			cmd_args = cmd_args[1:]
			continue
		}
		if cmd_args[0] == "-o" {
			if len(cmd_args) > 1 {
				output_directory = cmd_args[1]
				if len(output_directory) == 0 {
					fmt.Fprintf(os.Stderr, "ERROR:  Output directory is specified as an empty string.\n")
					bad_args = true
					print_help = true
					exit_early = true
					break
				}
				cmd_args = cmd_args[2:]
				continue
			} else {
				bad_args = true
				print_help = true
				exit_early = true
				break
			}
		}
		if len(cmd_args) == 1 {
			input_filepath = cmd_args[0]
			if len(input_filepath) == 0 {
				fmt.Fprintf(os.Stderr, "ERROR:  Input filepath is specified as an empty string.\n")
				bad_args = true
				print_help = true
				exit_early = true
				break
			}
			if len(output_directory) == 0 {
				output_directory = filepath.Dir(input_filepath)
			}
			break
		} else {
			bad_args = true
			print_help = true
			exit_early = true
			break
		}
	}
	if print_diagnostics && diag_file == os.Stdout {
		print_errors = false
	}
}

func main() {
	parse_args()
	if print_help {
		show_help()
	}
	if print_version {
		show_version()
	}
	if exit_early {
		// Go ought to have a ternary operator, but doesn't.  Sigh.
		if bad_args {
			os.Exit(1)
		} else {
			os.Exit(0)
		}
	}

	// FIX MINOR:  add support for -V and --version options; automate updating the version string if that is somehow possible

	fset, f, err := parse_file(input_filepath)
	if err != nil {
		os.Exit(1)
	}
	package_name,
		simple_typedefs, enum_typedefs, const_groups, struct_typedefs, struct_field_typedefs,
		simple_typedef_nodes, enum_typedef_nodes, const_group_nodes, struct_typedef_nodes,
		other_headers,
		err := process_parse_nodes(fset, f)
	if err != nil {
		os.Exit(1)
	}
	final_type_order, err := topologically_sort_nodes(
		simple_typedefs, enum_typedefs, const_groups, struct_typedefs,
		simple_typedef_nodes, enum_typedef_nodes, const_group_nodes, struct_typedef_nodes,
	)
	if err != nil {
		os.Exit(1)
	}
	pointer_base_types, pointer_list_base_types, simple_list_base_types, list_base_types, key_value_pair_types,
		struct_fields, struct_field_Go_types, struct_field_C_types, struct_field_tags, generated_C_code,
		err := print_type_declarations(
		package_name,
		final_type_order,
		simple_typedefs, enum_typedefs, const_groups, struct_typedefs, struct_field_typedefs,
		simple_typedef_nodes, enum_typedef_nodes, const_group_nodes, struct_typedef_nodes,
	)
	if err != nil {
		os.Exit(1)
	}
	err = print_type_conversions(
		other_headers,
		generated_C_code,
		package_name,
		final_type_order, pointer_base_types, pointer_list_base_types, simple_list_base_types, list_base_types, key_value_pair_types,
		simple_typedefs, enum_typedefs, const_groups, struct_typedefs,
		simple_typedef_nodes, enum_typedef_nodes, const_group_nodes, struct_typedef_nodes,
		struct_fields, struct_field_Go_types, struct_field_C_types, struct_field_tags,
	)
	if err != nil {
		os.Exit(1)
	}

	os.Exit(0)
}

// A routine whose output is to be used in debug messages, to precisely
// identify the source-code origin of the debug message.
func file_line() string {
	var s string
	if _, file_path, line_number, ok := runtime.Caller(1); ok {
		// We get back the full absolute path for the file_path.
		// That's much more than we need, so we extract the file
		// basename and use that instead.
		path_components := strings.Split(file_path, "/")
		base_name := path_components[len(path_components)-1]
		s = fmt.Sprintf("%s:%d", base_name, line_number)
	} else {
		s = ""
	}
	return s
}

// Routine to parse the file.
func parse_file(filepath string) (*token.FileSet, *ast.File, error) {
	fset := token.NewFileSet() // positions are relative to fset
	// mode := parser.ParseComments | parser.Trace | parser.DeclarationErrors
	mode := parser.ParseComments | parser.DeclarationErrors

	// Parse the specified file.
	f, err := parser.ParseFile(fset, filepath, nil, mode)
	if err != nil {
		fmt.Fprintf(diag_file, "found Go-syntax parsing error in file %s: %s\n", filepath, err)
		return nil, nil, err
	}

	return fset, f, nil
}

// FIX LATER:  We could probably use a certain amount of refactoring, both to factor out similar
// code blocks and to allow for a certain degree of potential recursion in type declarations.
//
// FIX MAJOR:  Make sure we test the following types separately:
//
//     "foo"
//     "*foo"
//     "[]foo"
//     "**foo"
//     "*[]foo"
//     "[]*foo"
//     "[][]foo"
//     "*[]*foo"
//
// We only track file-level (i.e., package-level) typedefs, and consts.  We don't track signatures
// for generated functions because we expect that any topological sorting that would benefit from
// such tracking will be obviated by instead just putting all the necessary function declarations
// in a header file, where all the declarations will come ahead of the code that needs them.
//
// Here are the forms of the principal returned element-type maps:
//
//           simple_typedefs map[    typedef_name string] typedef_type string
//             enum_typedefs map[       enum_name string]    enum_type string
//              const_groups map[const_group_name string]constant_type string
//           struct_typedefs map[     struct_name string] []field_type string
//     struct_field_typedefs map[     struct_name string]map[field_name string]field_typedef string
//
// Since individual const groups are all anonymous in the Go syntax, we make up such names simply so
// we can coordinate access to multiple data structures that need to refer to the same const group.
// For simple uniqueness and to provide easy traceability, we will use the stringified content of the
// "const" keyword's parse-node "TokPos" field, such as "transit.go:21:1".  (We could do the Go thing
// and just use the TokPos field directly as the map key, even if it is a structure rather than some
// simple datatype, but that would make it harder to track in development/diagnostic output which const
// group is being referred to.)
//
// Similar maps, using the same keys, are used to find the top-level parse node for each respective object:
//
//     simple_typedef_nodes map[    typedef_name string]decl_node *ast.GenDecl
//       enum_typedef_nodes map[       enum_name string]decl_node *ast.GenDecl
//        const_group_nodes map[const_group_name string]decl_node *ast.GenDecl
//     struct_typedef_nodes map[     struct_name string]decl_node *ast.GenDecl
//
func process_parse_nodes(
	fset *token.FileSet,
	f *ast.File,
) (
	package_name string,
	simple_typedefs map[string]string,
	enum_typedefs map[string]string,
	const_groups map[string]string,
	struct_typedefs map[string][]string, // list of unique simplified types of the fields
	struct_field_typedefs map[string]map[string]string,
	simple_typedef_nodes map[string]*ast.GenDecl,
	enum_typedef_nodes map[string]*ast.GenDecl,
	const_group_nodes map[string]*ast.GenDecl,
	struct_typedef_nodes map[string]*ast.GenDecl,
	other_headers string,
	err error,
) {
	// FIX MINOR:
	// Having this function in play turns out to be somewhat less than completely desirable,
	// because the simple error message does not include all the failure-coordinate data that
	// would have been printed by allowing the panic to proceed without interception.
	defer func() {
		if false {
			if exception := recover(); exception != nil {
				err = fmt.Errorf("internal error: %v", exception)
				if print_diagnostics {
					fmt.Fprintln(diag_file, err)
				}
				if print_errors {
					fmt.Println(err)
				}
			}
		}
	}()
	// struct_field_typedefs[struct_name][field_name] = full_field_typedef
	// (Note that this data structure loses info about the ordering of the fields
	// in any given struct, but that is fine for the uses we will make of this map.)
	struct_field_typedefs = map[string]map[string]string{}

	// package name
	// f.Name *Ident
	//
	// package top-level declarations, or nil
	// f.Decls []Decl

	simple_typedefs = map[string]string{}
	enum_typedefs = map[string]string{}
	const_groups = map[string]string{}
	struct_typedefs = map[string][]string{}

	simple_typedef_nodes = map[string]*ast.GenDecl{}
	enum_typedef_nodes = map[string]*ast.GenDecl{}
	const_group_nodes = map[string]*ast.GenDecl{}
	struct_typedef_nodes = map[string]*ast.GenDecl{}

	// Print the package name.
	package_name = f.Name.Name // from the "package" declaration inside the file
	if print_diagnostics {
		fmt.Fprintln(diag_file, "=== Package:")
		fmt.Fprintln(diag_file, package_name)
	}

	// Print the file's imports.
	if print_diagnostics {
		fmt.Fprintln(diag_file, "=== Imports:")
	}
	// special_package_prefix := regexp.MustCompile(`^github.com/gwos/tng/([^/]+)$`)
	special_package_prefix := regexp.MustCompile(`^github.com/gwos/.*/([^/]+)$`)
	include_headers := []string{}
	for _, s := range f.Imports {
		if print_diagnostics {
			fmt.Fprintln(diag_file, s.Path.Value)
		}
		pkg := strings.ReplaceAll(s.Path.Value, "\"", "")
		special_package := special_package_prefix.FindStringSubmatch(pkg)
		if special_package != nil && special_package[1] != "log" {
			include_headers = append(include_headers, fmt.Sprintf(`#include "%s.h"`, special_package[1]))
		}
	}
	other_headers = strings.Join(include_headers, "\n")

	// Print the file's documentation.
	// It only prints the leading package doc, not function comments.
	// For that, one needs to dig deeper (see below).
	// FIX MAJOR:  This is not stripping the leading "//" from comment lines.
	if print_diagnostics {
		fmt.Fprintln(diag_file, "=== Package Documentation:")
		if f.Doc != nil {
			for _, doc := range f.Doc.List {
				fmt.Fprintln(diag_file, doc.Text)
			}
		}
	}

	if print_diagnostics {
		fmt.Fprintln(diag_file, "=== Declarations:")
	}
	// Print the file-level declarations.  This conveniently ignores declarations within functions,
	// which we don't care about for our purposes.
	panic_message := ""
node_loop:
	for _, file_decl := range f.Decls {
		if print_diagnostics {
			// fmt.Fprintln(diag_file, d)  // "&{<nil> <nil> parse_file 0xc000093660 0xc00007abd0}" and other forms
		}
		if func_decl, ok := file_decl.(*ast.FuncDecl); ok {
			if print_diagnostics {
				fmt.Fprintf(diag_file, "--- function name:  %v\n", func_decl.Name.Name)
				if func_decl.Doc != nil {
					fmt.Fprintln(diag_file, "--- function documentation:")
					// FIX MAJOR:  This is not stripping the leading "//" from comment lines.
					for _, doc := range func_decl.Doc.List {
						fmt.Fprintln(diag_file, doc.Text)
					}
				}
			}
		}
		if gen_decl, ok := file_decl.(*ast.GenDecl); ok {
			if gen_decl.Tok == token.TYPE {
				for _, spec := range gen_decl.Specs {
					// I'm just assuming that spec.(*ast.TypeSpec).Type is of type *ast.Ident here in all cases.
					// If that turns out not to be true, we'll have to fill in other cases.
					if type_ident, ok := spec.(*ast.TypeSpec).Type.(*ast.Ident); ok {
						if print_diagnostics {
							fmt.Fprintf(diag_file, "--- simple type declaration name and type:  %v %v\n", spec.(*ast.TypeSpec).Name.Name, type_ident.Name)
						}
						simple_typedefs[spec.(*ast.TypeSpec).Name.Name] = type_ident.Name
						simple_typedef_nodes[spec.(*ast.TypeSpec).Name.Name] = gen_decl
					} else if type_struct, ok := spec.(*ast.TypeSpec).Type.(*ast.StructType); ok {
						if print_diagnostics {
							// fmt.Fprintf(diag_file, "--- struct type:  %#v\n", type_struct)
							fmt.Fprintf(diag_file, "--- struct type declaration name:  %v\n", spec.(*ast.TypeSpec).Name.Name)
						}
						struct_typedefs[spec.(*ast.TypeSpec).Name.Name] = nil

						// FIX MINOR:  I'm not yet sure if this is correct (though it seems to be working).
						struct_field_typedefs[spec.(*ast.TypeSpec).Name.Name] = map[string]string{}

						// fiX QUICK:  drop the extra commented-out code here
						// struct_typedefs[spec.(*ast.TypeSpec).Name.Name] = []string{nil}
						// struct_typedefs[spec.(*ast.TypeSpec).Name.Name] = []string{}
						struct_typedef_nodes[spec.(*ast.TypeSpec).Name.Name] = gen_decl
						if type_struct.Incomplete {
							// I'm not sure when this condition might be true, so let's alarm on it if we encounter it
							// just to make sure we're not overlooking anything.
							if print_diagnostics {
								fmt.Fprintf(diag_file, "    --- The list of fields is incomplete.\n")
							}
							panic_message = "aborting due to previous errors"
							break node_loop
						}
						for _, field := range type_struct.Fields.List {
							// FIX MAJOR:  Add support for the .Doc and .Comment attributes as well.
							if print_diagnostics {
								// fmt.Fprintf(diag_file, "    --- field:  %#v\n", field)
							}
							// Field elements to process:
							// .Doc   *ast.CommentGroup    // may be nil
							// .Names []*ast.Ident
							if field.Names == nil {
								// Here, we have an anonymous field, such as occurs with Go's structure embedding.  Since
								// that won't do in C, we autovivify a field name from the field type, similar to how that
								// is done implicitly in Go itself but generally appending a small string to guarantee that
								// there will be no confusion in C between the field name and the type name.
								if type_ident, ok := field.Type.(*ast.Ident); ok {
									// Old construction:  just accept that we have a missing field name.
									if print_diagnostics {
										// fmt.Fprintf(diag_file, "    --- struct field name and type:  %#v %#v\n", "(none)", type_ident.Name)
									}
									// New construction:  autovivify a sensible field name.
									name_ident := new(ast.Ident)
									// Testing shows I was wrong; modern C can handle having a variable or struct field named
									// the same as a struct typedef.  So to keep things simple, we don't append an underscore
									// to type_ident.Name here.
									name_ident.Name = type_ident.Name
									field.Names = append(field.Names, name_ident)
								} else if type_starexpr, ok := field.Type.(*ast.StarExpr); ok {
									if print_diagnostics {
										// fmt.Fprintf(diag_file, "    --- struct field name and type:  %#v %#v\n", "(none)", type_starexpr)
									}
									if type_ident, ok := type_starexpr.X.(*ast.Ident); ok {
										if print_diagnostics {
											// fmt.Fprintf(diag_file, "    --- struct field name and StarExpr type:  %#v %#v\n", name.Name, type_ident.Name)
										}
										name_ident := new(ast.Ident)
										name_ident.Name = type_ident.Name + "_ptr_"
										field.Names = append(field.Names, name_ident)
									} else if type_selectorexpr, ok := type_starexpr.X.(*ast.SelectorExpr); ok {
										/*
											var x_type_ident *ast.Ident
											var ok bool
											if x_type_ident, ok = type_selectorexpr.X.(*ast.Ident); ok {
											    if print_diagnostics {
												// fmt.Fprintf(diag_file, "    --- struct field name and SelectorExpr X:  %#v %#v\n", name.Name, x_type_ident.Name)
												// fmt.Fprintf(diag_file, "    --- struct field SelectorExpr X:  %#v\n", x_type_ident.Name)
											    }
											} else {
											    fmt.Fprintf(diag_file, "ERROR:  when autovivifying at %s, found unexpected field.Type.X type:  %T\n",
												file_line(), type_selectorexpr.X)
											    fmt.Fprintf(diag_file, "ERROR:  struct field Type.X field is not of a recognized type\n")
											    panic_message = "aborting due to previous errors"
											    break node_loop
											}
										*/
										if type_selectorexpr.Sel == nil {
											fmt.Fprintf(diag_file, "ERROR:  when autovivifying at %s, struct field Type Sel field is unexpectedly nil\n", file_line())
											panic_message = "aborting due to previous errors"
											break node_loop
										}
										name_ident := new(ast.Ident)
										// We used to append an underscore in this construction of name_ident.Name, but we
										// are backing off from that until and unless we find it to actually be necessary.
										// (The backoff is not yet done, pending testing.)
										//
										// We logically ought to include the x_type_ident.Name as the first part of this constructed
										// field name to totally disambiguate it, but for now we are dropping that out.  This improves
										// our ability to identify whether a given field should be exported in JSON, by making the
										// selector name (which is what we believe Go will look for when deciding on the effective
										// name of the field, and thus for deciding whether the field is exported) be visible at
										// the start of the field name.  That way, we can just check the first rune for uppercase
										// just as Go would.  However, without having the x_type_ident.Name component as part of the
										// name, we risk generating a field-name conflict, which would happen if we had two identical
										// type_selectorexpr.Sel.Name names in the same structure originating from different packages.
										// If that happens and we therefore need to put the x_type_ident.Name back into the field name,
										// we could do so in some later part of the field name, even though that would look a bit ugly.
										//
										// name_ident.Name = x_type_ident.Name + "_" + type_selectorexpr.Sel.Name + "_ptr_"
										//
										name_ident.Name = type_selectorexpr.Sel.Name + "_ptr_"
										if print_diagnostics {
											// fmt.Fprintf(diag_file, "    ==> manufactured field name:  %s\n", name_ident.Name)
										}
										field.Names = append(field.Names, name_ident)
									} else {
										//
										//  .  .  .  .  .  .  .  List: []*ast.Field (len = 1) {
										//  .  .  .  .  .  .  .  .  0: *ast.Field {
										//  .  .  .  .  .  .  .  .  .  Type: *ast.StarExpr {
										//  .  .  .  .  .  .  .  .  .  .  Star: transit.go:404:2
										//  .  .  .  .  .  .  .  .  .  .  X: *ast.SelectorExpr {
										//  .  .  .  .  .  .  .  .  .  .  .  X: *ast.Ident {
										//  .  .  .  .  .  .  .  .  .  .  .  .  NamePos: transit.go:404:3
										//  .  .  .  .  .  .  .  .  .  .  .  .  Name: "config"
										//  .  .  .  .  .  .  .  .  .  .  .  }
										//  .  .  .  .  .  .  .  .  .  .  .  Sel: *ast.Ident {
										//  .  .  .  .  .  .  .  .  .  .  .  .  NamePos: transit.go:404:10
										//  .  .  .  .  .  .  .  .  .  .  .  .  Name: "Config"
										//  .  .  .  .  .  .  .  .  .  .  .  }
										//  .  .  .  .  .  .  .  .  .  .  }
										//  .  .  .  .  .  .  .  .  .  }
										//  .  .  .  .  .  .  .  .  }
										//  .  .  .  .  .  .  .  }
										//
										// The type of type_starexpr.X is a *ast.SelectorExpr, and that occurs within a field of type *ast.StarExpr .
										// So once we figure out the field name we will manufacture for type_starexpr.X, we will append "_ptr_" to that name.
										//
										if print_diagnostics {
											fmt.Fprintf(diag_file, "ERROR:  when autovivifying at %s, found unexpected field.Type.X type:  %T\n",
												file_line(), type_starexpr.X)
											fmt.Fprintf(diag_file, "ERROR:  struct field Type.X field is not of a recognized type\n")
										}
										panic_message = "aborting due to previous errors"
										break node_loop
									}
								} else if type_selectorexpr, ok := field.Type.(*ast.SelectorExpr); ok {
									/*
										    var x_type_ident *ast.Ident
										    var ok bool
										    if x_type_ident, ok = type_selectorexpr.X.(*ast.Ident); ok {
											if print_diagnostics {
											    // fmt.Fprintf(diag_file, "    --- struct field name and SelectorExpr X:  %#v %#v\n", name.Name, x_type_ident.Name)
											}
										    } else {
											if print_diagnostics {
											    fmt.Fprintf(diag_file, "ERROR:  when autovivifying at %s, found unexpected field.Type.X type:  %T\n",
												file_line(), type_selectorexpr.X)
											    fmt.Fprintf(diag_file, "ERROR:  struct field Type.X field is not of a recognized type\n")
											}
											panic_message = "aborting due to previous errors"
											break node_loop
										    }
									*/
									if type_selectorexpr.Sel == nil {
										if print_diagnostics {
											fmt.Fprintf(diag_file, "ERROR:  when autovivifying at %s, struct field Type Sel field is unexpectedly nil\n", file_line())
										}
										panic_message = "aborting due to previous errors"
										break node_loop
									}
									name_ident := new(ast.Ident)
									// We used to append an underscore in this construction of name_ident.Name, but we
									// are backing off from that until and unless we find it to actually be necessary.
									// (The backoff is not yet done, pending testing.)
									//
									// We logically ought to include the x_type_ident.Name as the first part of this constructed
									// field name to totally disambiguate it, but for now we are dropping that out.  This improves
									// our ability to identify whether a given field should be exported in JSON, by making the
									// selector name (which is what we believe Go will look for when deciding on the effective
									// name of the field, and thus for deciding whether the field is exported) be visible at
									// the start of the field name.  That way, we can just check the first rune for uppercase
									// just as Go would.  However, without having the x_type_ident.Name component as part of the
									// name, we risk generating a field-name conflict, which would happen if we had two identical
									// type_selectorexpr.Sel.Name names in the same structure originating from different packages.
									// If that happens and we therefore need to put the x_type_ident.Name back into the field name,
									// we could do so in some later part of the field name, even though that would look a bit ugly.
									//
									// name_ident.Name = x_type_ident.Name + "_" + type_selectorexpr.Sel.Name + "_"
									//
									name_ident.Name = type_selectorexpr.Sel.Name + "_"
									field.Names = append(field.Names, name_ident)
								} else {
									if print_diagnostics {
										fmt.Fprintf(diag_file, "ERROR:  when autovivifying at %s, found unexpected field.Type type:  %T\n", file_line(), field.Type)
										fmt.Fprintf(diag_file, "ERROR:  struct field Type field is not of a recognized type\n")
									}
									panic_message = "aborting due to previous errors"
									break node_loop
								}
							}
							for _, name := range field.Names {
								if print_diagnostics {
									// fmt.Fprintf(diag_file, "    --- field name:  %#v\n", name)
								}
								var field_type_name string
								if type_ident, ok := field.Type.(*ast.Ident); ok {
									field_type_name = type_ident.Name
									if print_diagnostics {
										fmt.Fprintf(diag_file, "    --- struct field name and type:  %#v %#v\n", name.Name, field_type_name)
									}
								} else if type_starexpr, ok := field.Type.(*ast.StarExpr); ok {
									if type_ident, ok := type_starexpr.X.(*ast.Ident); ok {
										field_type_name = "*" + type_ident.Name
										if print_diagnostics {
											fmt.Fprintf(diag_file, "    --- struct field name and StarExpr type:  %#v %#v\n", name.Name, field_type_name)
										}
									} else if type_array, ok := type_starexpr.X.(*ast.ArrayType); ok {
										var array_type_ident *ast.Ident
										// A nil type_array.Len means it's a slice type.
										if type_array.Len != nil {
											if print_diagnostics {
												fmt.Fprintf(diag_file, "ERROR:  at %s, a non-nil value for a StarExpr array-type Len is not yet handled (%#v)\n",
													file_line(), type_array.Len)
											}
											panic_message = "aborting due to previous errors"
											break node_loop
										}
										if array_type_ident, ok = type_array.Elt.(*ast.Ident); ok {
											if print_diagnostics {
												// fmt.Fprintf(diag_file, "    --- struct field Type X Elt array element ident %#v\n", array_type_ident)
											}
										} else {
											if print_diagnostics {
												fmt.Fprintf(diag_file, "ERROR:  at %s, found unexpected field.Type.X.Elt type:  %T\n", file_line(), type_array.Elt)
												fmt.Fprintf(diag_file, "ERROR:  struct field Type X Elt field is not of a recognized type\n")
											}
											panic_message = "aborting due to previous errors"
											break node_loop
										}
										field_type_name = "*[]" + array_type_ident.Name
										if print_diagnostics {
											fmt.Fprintf(diag_file, "    --- struct field name and type:  %#v %#v\n", name.Name, field_type_name)
										}
									} else if type_selectorexpr, ok := type_starexpr.X.(*ast.SelectorExpr); ok {
										var x_type_ident *ast.Ident
										var ok bool
										if x_type_ident, ok = type_selectorexpr.X.(*ast.Ident); ok {
											if print_diagnostics {
												// fmt.Fprintf(diag_file, "    --- struct field name and SelectorExpr X:  %#v %#v\n", name.Name, x_type_ident.Name)
											}
										} else {
											if print_diagnostics {
												fmt.Fprintf(diag_file, "ERROR:  at %s, found unexpected field.Type.X type:  %T\n", file_line(), type_selectorexpr.X)
												fmt.Fprintf(diag_file, "ERROR:  struct field Type.X field is not of a recognized type\n")
											}
											panic_message = "aborting due to previous errors"
											break node_loop
										}
										if type_selectorexpr.Sel == nil {
											if print_diagnostics {
												fmt.Fprintf(diag_file, "ERROR:  at %s, struct field Type Sel field is unexpectedly nil\n", file_line())
											}
											panic_message = "aborting due to previous errors"
											break node_loop
										}
										// FIX MINOR:  This may need work to fully and correctly reflect the complete selector.
										field_type_name = "*" + x_type_ident.Name + "." + type_selectorexpr.Sel.Name
										if print_diagnostics {
											fmt.Fprintf(diag_file, "    --- struct field name and type:  %#v *%v.%v\n", name.Name, x_type_ident.Name, field_type_name)
										}
									} else {
										if print_diagnostics {
											fmt.Fprintf(diag_file, "ERROR:  at %s, found unexpected field.Type.X type:  %T\n", file_line(), type_starexpr.X)
											fmt.Fprintf(diag_file, "ERROR:  struct field Type.X field is not of a recognized type\n")
										}
										panic_message = "aborting due to previous errors"
										break node_loop
									}
								} else if type_array, ok := field.Type.(*ast.ArrayType); ok {
									// A nil type_array.Len means it's a slice type.
									if type_array.Len != nil {
										if print_diagnostics {
											fmt.Fprintf(diag_file, "ERROR:  at %s, a non-nil value for an array-type Len is not yet handled (%#v)\n",
												file_line(), type_array.Len)
										}
										panic_message = "aborting due to previous errors"
										break node_loop
									}
									if type_ident, ok := type_array.Elt.(*ast.Ident); ok {
										if print_diagnostics {
											// fmt.Fprintf(diag_file, "    --- array element ident %#v\n", type_ident)
										}
										field_type_name = "[]" + type_ident.Name
										if print_diagnostics {
											fmt.Fprintf(diag_file, "    --- struct field name and type:  %#v %#v\n", name.Name, field_type_name)
										}
									} else if type_starexpr, ok := type_array.Elt.(*ast.StarExpr); ok {
										if print_diagnostics {
											// fmt.Fprintf(diag_file, "    --- array element starexpr %#v\n", type_starexpr)
										}
										if type_ident, ok := type_starexpr.X.(*ast.Ident); ok {
											field_type_name = "[]*" + type_ident.Name
											if print_diagnostics {
												fmt.Fprintf(diag_file, "    --- struct field name and interior StarExpr type:  %#v %#v\n", name.Name, field_type_name)
											}
										} else if type_array, ok := type_starexpr.X.(*ast.ArrayType); ok {
											if print_diagnostics {
												fmt.Fprintf(diag_file, "    --- UNEXPECTED interior field.Type.X Type *ast.ArrayType %#v\n", type_array)
											}
											// FIX MAJOR:  Handle this case.
											panic_message = "aborting due to previous errors"
											break node_loop
										} else {
											if print_diagnostics {
												fmt.Fprintf(diag_file, "ERROR:  at %s, found unexpected interior field.Type.X type:  %T\n", file_line(), type_starexpr.X)
												fmt.Fprintf(diag_file, "ERROR:  struct field interior Type.X field is not of a recognized type\n")
											}
											panic_message = "aborting due to previous errors"
											break node_loop
										}
									} else {
										if print_diagnostics {
											fmt.Fprintf(diag_file, "ERROR:  at %s, found unexpected field.Type.Elt type:  %T\n", file_line(), type_array.Elt)
											fmt.Fprintf(diag_file, "ERROR:  struct field Type Elt field is not of a recognized type\n")
										}
										panic_message = "aborting due to previous errors"
										break node_loop
									}
								} else if type_map, ok := field.Type.(*ast.MapType); ok {
									var key_type_ident *ast.Ident
									var value_type_ident *ast.Ident
									var ok bool
									if key_type_ident, ok = type_map.Key.(*ast.Ident); ok {
										if print_diagnostics {
											// fmt.Fprintf(diag_file, "    --- map Key Ident %#v\n", key_type_ident)
										}
									} else {
										if print_diagnostics {
											fmt.Fprintf(diag_file, "ERROR:  at %s, found unexpected field.Type.Key type:  %T\n", file_line(), type_map.Key)
											fmt.Fprintf(diag_file, "ERROR:  struct field Type Key field is not of a recognized type\n")
										}
										panic_message = "aborting due to previous errors"
										break node_loop
									}
									if value_type_ident, ok = type_map.Value.(*ast.Ident); ok {
										if print_diagnostics {
											// fmt.Fprintf(diag_file, "    --- map Value Ident %#v\n", value_type_ident)
										}
									} else {
										if print_diagnostics {
											fmt.Fprintf(diag_file, "ERROR:  at %s, found unexpected field.Type.Value type:  %T\n", file_line(), type_map.Value)
											fmt.Fprintf(diag_file, "ERROR:  struct field Type Value field is not of a recognized type\n")
										}
										panic_message = "aborting due to previous errors"
										break node_loop
									}
									// FIX QUICK:  This needs work to fully reflect the map structure; perhaps the new statements now do so.
									// field_type_name = value_type_ident.Name
									if print_diagnostics {
										// fmt.Fprintf(diag_file, "    --- struct field name and type:  %#v map[%#v]%#v\n", name.Name, key_type_ident.Name, field_type_name)
									}
									field_type_name = "map[" + key_type_ident.Name + "]" + value_type_ident.Name
									if print_diagnostics {
										fmt.Fprintf(diag_file, "    --- struct field name and type:  %#v %#v\n", name.Name, field_type_name)
									}
								} else if type_selectorexpr, ok := field.Type.(*ast.SelectorExpr); ok {
									var x_type_ident *ast.Ident
									var ok bool
									if x_type_ident, ok = type_selectorexpr.X.(*ast.Ident); ok {
										if print_diagnostics {
											// fmt.Fprintf(diag_file, "    --- struct field name and SelectorExpr X:  %#v %#v\n", name.Name, x_type_ident.Name)
										}
									} else {
										if print_diagnostics {
											fmt.Fprintf(diag_file, "ERROR:  at %s, found unexpected field.Type.X type:  %T\n", file_line(), type_selectorexpr.X)
											fmt.Fprintf(diag_file, "ERROR:  struct field Type.X field is not of a recognized type\n")
										}
										panic_message = "aborting due to previous errors"
										break node_loop
									}
									if type_selectorexpr.Sel == nil {
										if print_diagnostics {
											fmt.Fprintf(diag_file, "ERROR:  at %s, struct field Type Sel field is unexpectedly nil\n", file_line())
										}
										panic_message = "aborting due to previous errors"
										break node_loop
									}
									// FIX QUICK:  This may need work to fully and correctly reflect the complete selector.
									field_type_name = x_type_ident.Name + "." + type_selectorexpr.Sel.Name
									if print_diagnostics {
										fmt.Fprintf(diag_file, "    --- struct field name and type:  %#v %v.%v\n", name.Name, x_type_ident.Name, field_type_name)
									}
								} else {
									if print_diagnostics {
										fmt.Fprintf(diag_file, "ERROR:  at %s, found unexpected field.Type type:  %T\n", file_line(), field.Type)
										fmt.Fprintf(diag_file, "ERROR:  struct field Type field is not of a recognized type\n")
									}
									panic_message = "aborting due to previous errors"
									break node_loop
								}
								struct_typedefs[spec.(*ast.TypeSpec).Name.Name] = append(struct_typedefs[spec.(*ast.TypeSpec).Name.Name], field_type_name)
								struct_field_typedefs[spec.(*ast.TypeSpec).Name.Name][name.Name] = field_type_name
								if field.Tag != nil {
									if print_diagnostics {
										fmt.Fprintf(diag_file, "    --- struct field tag Value:  %#v\n", field.Tag.Value)
									}
								}
							}
							// .Type  *ast.Ident
							// .Tag   *ast.BasicLit        // may be nil
							// .Comment *ast.CommentGroup  // likely nil
						}
					} else if type_interface, ok := spec.(*ast.TypeSpec).Type.(*ast.InterfaceType); ok {
						if print_diagnostics {
							fmt.Fprintf(diag_file, "FIX MAJOR:  Handle this next case (where the type is *ast.InterfaceType)\n")
						}
						// This is an interface definition, which perhaps mostly declares methods, not simple types,
						// enumerations, constants, or structs.  Verify that assumption, and perhaps extend this case
						// to process whatever it might need to.  We might, for instance, at least need to emit function
						// signatures, even if we don't generate full function bodies.
						if print_diagnostics {
							fmt.Fprintf(diag_file, "--- interface type declaration name and type:  %v %#v\n", spec.(*ast.TypeSpec).Name.Name, type_interface)
						}
					} else if type_array, ok := spec.(*ast.TypeSpec).Type.(*ast.ArrayType); ok {
						// FIX MINOR:  There are many sub-cases here that are undeveloped because we have not yet seen them in source code.
						// Someday, we should extend our test cases to cover these aspects, and then use that to force filling this out.
						var field_type_name string
						// A nil type_array.Len means it's a slice type.
						if type_array.Len != nil {
							if print_diagnostics {
								fmt.Fprintf(diag_file, "ERROR:  at %s, a non-nil value for an array-type Len is not yet handled (%#v)\n",
									file_line(), type_array.Len)
							}
							panic_message = "aborting due to previous errors"
							break node_loop
						}
						if type_ident, ok := type_array.Elt.(*ast.Ident); ok {
							if print_diagnostics {
								// fmt.Fprintf(diag_file, "--- array element ident %#v\n", type_ident)
							}
							field_type_name = "[]" + type_ident.Name
							// FIX MAJOR:  Trigger this case and get it handled correctly.
							if print_diagnostics {
								fmt.Fprintf(diag_file, "--- UNEXPECTED simple type declaration and array type:  %#v %#v\n", type_ident.Name, field_type_name)
							}
							// simple_typedefs[spec.(*ast.TypeSpec).Name.Name] = field_type_name
							// simple_typedef_nodes[spec.(*ast.TypeSpec).Name.Name] = gen_decl
							panic_message = "aborting due to previous errors"
							break node_loop
						} else if type_starexpr, ok := type_array.Elt.(*ast.StarExpr); ok {
							if print_diagnostics {
								// fmt.Fprintf(diag_file, "--- array element starexpr %#v\n", type_starexpr)
							}
							if type_ident, ok := type_starexpr.X.(*ast.Ident); ok {
								field_type_name = "[]*" + type_ident.Name
								if print_diagnostics {
									fmt.Fprintf(diag_file, "--- simple type declaration and interior StarExpr type:  %#v %#v\n",
										spec.(*ast.TypeSpec).Name.Name, field_type_name)
								}
								simple_typedefs[spec.(*ast.TypeSpec).Name.Name] = package_name + "_" + type_ident.Name + "_Ptr_List"
								simple_typedef_nodes[spec.(*ast.TypeSpec).Name.Name] = gen_decl
							} else if type_array, ok := type_starexpr.X.(*ast.ArrayType); ok {
								if print_diagnostics {
									fmt.Fprintf(diag_file, "--- UNEXPECTED interior field.Type.X Type *ast.ArrayType %#v\n", type_array)
								}
								panic_message = "aborting due to previous errors"
								break node_loop
							} else {
								if print_diagnostics {
									fmt.Fprintf(diag_file, "ERROR:  at %s, found unexpected interior field.Type.X type:  %T\n", file_line(), type_starexpr.X)
									fmt.Fprintf(diag_file, "ERROR:  field interior Type.X field is not of a recognized type\n")
								}
								panic_message = "aborting due to previous errors"
								break node_loop
							}
						} else {
							if print_diagnostics {
								fmt.Fprintf(diag_file, "ERROR:  at %s, found unexpected field.Type.Elt type:  %T\n", file_line(), type_array.Elt)
								fmt.Fprintf(diag_file, "ERROR:  field Type Elt field is not of a recognized type\n")
							}
							panic_message = "aborting due to previous errors"
							break node_loop
						}
						// FIX MAJOR:  Clean this up, once all the individual subcases above have been handled
						// and we no longer need this for diagnosis in development.
						if print_diagnostics {
							// fmt.Fprintf(diag_file, "ERROR:  at %s, found unexpected spec.(*ast.TypeSpec).Type type:  %T\n", file_line(), spec.(*ast.TypeSpec).Type)
							// fmt.Fprintf(diag_file, "ERROR:  spec *ast.TypeSpec Type field is of type %T which is not yet handled\n", spec.(*ast.TypeSpec).Type)
						}
						// panic_message = "aborting due to previous errors"
						// break node_loop
					} else {
						if print_diagnostics {
							fmt.Fprintf(diag_file, "ERROR:  at %s, found unexpected spec.(*ast.TypeSpec).Type type:  %T\n", file_line(), spec.(*ast.TypeSpec).Type)
							fmt.Fprintf(diag_file, "ERROR:  spec *ast.TypeSpec Type field is not of a recognized type\n")
						}
						panic_message = "aborting due to previous errors"
						break node_loop
					}
				}
			} else if gen_decl.Tok == token.CONST {
				// FIX MAJOR:  This needs some testing to see when iota_value and value_is_from_iota need to be set or reset.
				var spec_type string
				var iota_value int = -1
				var value_is_from_iota bool = false
				for _, spec := range gen_decl.Specs {
					if spec.(*ast.ValueSpec).Type != nil {
						// I'm just assuming that spec.(*ast.TypeSpec).Type is of type *ast.Ident here in all cases.
						// If that turns out not to be true, we'll have to fill in other cases.
						if type_ident, ok := spec.(*ast.ValueSpec).Type.(*ast.Ident); ok {
							spec_type = type_ident.Name
						} else {
							if print_diagnostics {
								fmt.Fprintf(diag_file, "ERROR:  at %s, found unexpected spec.(*ast.ValueSpec).Type type:  %T\n", file_line(), spec.(*ast.ValueSpec).Type)
								fmt.Fprintf(diag_file, "ERROR:  spec *ast.ValueSpec Type field is not of a recognized type\n")
							}
							panic_message = "aborting due to previous errors"
							break node_loop
						}
						// value_type := spec.(*ast.ValueSpec).Type
						if print_diagnostics {
							// fmt.Fprintf(diag_file, "value_type = %T %[1]v %+[1]v %#[1]v %[1]s\n", value_type)
						}
					}
					var const_value string
					for i, name := range spec.(*ast.ValueSpec).Names {
						if i < len(spec.(*ast.ValueSpec).Values) {
							switch spec.(*ast.ValueSpec).Values[i].(type) {
							case *ast.Ident:
								// A const value of "iota" will show up this way, with a nil Obj.
								if spec.(*ast.ValueSpec).Values[i].(*ast.Ident).Name == "iota" {
									value_is_from_iota = true
									iota_value++
									const_value = fmt.Sprintf("%d", iota_value)
								} else {
									if print_diagnostics {
										fmt.Fprintf(diag_file, "ERROR:  at %s, value name is %#v\n", file_line(), spec.(*ast.ValueSpec).Values[i].(*ast.Ident).Name)
									}
									panic_message = "unexpected const value name"
									break node_loop
								}
								if spec.(*ast.ValueSpec).Values[i].(*ast.Ident).Obj != nil {
									if print_diagnostics {
										fmt.Fprintf(diag_file, "ERROR:  at %s, value object kind is %#v\n",
											file_line(), spec.(*ast.ValueSpec).Values[i].(*ast.Ident).Obj.Kind)
										fmt.Fprintf(diag_file, "ERROR:  at %s, value object name is %#v\n",
											file_line(), spec.(*ast.ValueSpec).Values[i].(*ast.Ident).Obj.Name)
									}
								}
							case *ast.BasicLit:
								switch spec.(*ast.ValueSpec).Values[i].(*ast.BasicLit).Kind {
								/*
									case token.INT:
									    const_value = spec.(*ast.ValueSpec).Values[i].(*ast.BasicLit).Value
									    if spec_type == "" {
										spec_type = "int"
									    }
								*/
								// case token.FLOAT:
								// case token.IMAG:
								// case token.CHAR:
								case token.STRING:
									const_value = spec.(*ast.ValueSpec).Values[i].(*ast.BasicLit).Value
									if spec_type == "" {
										spec_type = "string"
									}
								default:
									panic_message = "unexpected const value BasicLit type"
									break node_loop
								}
							case *ast.BinaryExpr:
								if print_diagnostics {
									fmt.Fprintf(diag_file, "ERROR:  at %s, value expression is %#v\n", file_line(), spec.(*ast.ValueSpec).Values[i].(*ast.BinaryExpr))
								}
								// FIX MAJOR:  This setting of spec_type is nowhere near a thorough analysis.
								if spec_type == "" {
									spec_type = "int"
								}
								r, err := eval_int_expr(spec.(*ast.ValueSpec).Values[i].(*ast.BinaryExpr), &iota_value)
								if err != nil {
									if print_diagnostics {
										fmt.Fprintln(diag_file, err)
									}
									if print_errors {
										fmt.Println(err)
									}
									panic_message = "cannot evaluate binary expression"
									break node_loop
								}
								const_value = fmt.Sprintf("%d", r)
							default:
								if print_diagnostics {
									fmt.Fprintf(diag_file, "ERROR:  at %s, found const value type %#v\n", file_line(), spec.(*ast.ValueSpec).Values[i])
								}
								panic_message = "unexpected const value type"
								break node_loop
							}
						} else if value_is_from_iota {
							iota_value++
							const_value = fmt.Sprintf("%d", iota_value)
						}
						// FIX MAJOR:  This is not yet showing the "int" spec_type for a "1 << iota" expression.
						if print_diagnostics {
							fmt.Fprintf(diag_file, "--- const element name, type, and value:  %v %v %v\n", name.Name, spec_type, const_value)
						}
						// It's not required by Go syntax that every assignment in a single const block has exactly
						// the same type, but we insist on that here to simplify our work.  If we encounter code that
						// violates this constraint, the code in this conversion tool will need to be extended.
						const_token_position := fset.Position(gen_decl.TokPos).String()
						if const_groups[const_token_position] == "" {
							const_groups[const_token_position] = spec_type
							const_group_nodes[const_token_position] = gen_decl
						} else if const_groups[const_token_position] != spec_type {
							if print_diagnostics {
								fmt.Fprintf(diag_file, "ERROR:  at %s, found conflicting const types in a single const block:  %s %s\n",
									file_line(), const_groups[const_token_position], spec_type)
							}
							panic_message = "found conflicting const types in a single const block"
							break node_loop
						}
					}
				}
			}
		}
	}

	if print_diagnostics {
		fmt.Fprintln(diag_file, "=== AST:")
		// Unexported struct fields are never printed.
		ast.Fprint(diag_file, fset, f, ast.NotNilFilter)
	}

	if panic_message != "" {
		if print_diagnostics {
			fmt.Fprintf(diag_file, "%s\n", panic_message)
		}
		panic(panic_message)
	}

	// Go doesn't distinguish an enumeration typedef from a simple typedef by syntax.
	// It's only by the presence of associated constants that we can conclude that an
	// enumeration is intended.  So, we implement that change of semantics here.
	for _, constant_type := range const_groups {
		if simple_typedefs[constant_type] != "" {
			enum_typedefs[constant_type] = simple_typedefs[constant_type]
			delete(simple_typedefs, constant_type)
			enum_typedef_nodes[constant_type] = simple_typedef_nodes[constant_type]
			delete(simple_typedef_nodes, constant_type)
		}
	}

	// A struct can have multiple fields with the same type.  Repeating those type names
	// is not useful in the downstream code that does a topological sort based on type
	// dependencies.  So we clean that up.  While we're doing so, we also remove any
	// punctuation we added earlier for precise type specification, since that level of
	// detail will interfere with our analysis of the simple-type dependencies.
	drop_punctuation := regexp.MustCompile(`^[\*\[\]]+`)
	is_a_map := regexp.MustCompile(`^map\[`)
	// This expression ought to be generalized to check for balanced [] characters within the map key.
	map_key_value_types := regexp.MustCompile(`map\[([^]]+)\](.+)`)

	for struct_name, field_types := range struct_typedefs {
		// Here we suppress duplicate values; too bad Go doesn't have Perl's "keys" function.
		unique_field_types := map[string]bool{}
		for _, field_type := range field_types {
			if is_a_map.MatchString(field_type) {
				// Break apart a map type into its separate key and value types.
				// FIX LATER:  Obviously, in the general case, both the key and value types might be more
				// complex than we are allowing for here (just individual base types, not involving slices,
				// pointers, further maps, or perhaps other exotic fauna).  If we run into trouble when
				// converting such code, the analysis here will need extension.
				key_value_types := map_key_value_types.FindStringSubmatch(field_type)
				if key_value_types == nil {
					panic(fmt.Sprintf("found incomprehensible map construction '%s'", field_type))
				}
				key_type := key_value_types[1]
				value_type := key_value_types[2]
				unique_field_types[key_type] = true
				unique_field_types[value_type] = true
			} else {
				unique_field_types[drop_punctuation.ReplaceAllLiteralString(field_type, "")] = true
			}
		}
		field_types = make([]string, len(unique_field_types))
		i := 0
		for field_type := range unique_field_types {
			field_types[i] = field_type
			i++
		}
		struct_typedefs[struct_name] = field_types
	}

	return package_name,
		simple_typedefs, enum_typedefs, const_groups, struct_typedefs, struct_field_typedefs,
		simple_typedef_nodes, enum_typedef_nodes, const_group_nodes, struct_typedef_nodes,
		other_headers,
		nil
}

// FIX MAJOR:  Add support for iota to the evaluation.
// FIX MAJOR:  This is not yet coordinated with the ast.Ident processing above.
func eval_int_expr(tree ast.Expr, iota *int) (int, error) {
	switch n := tree.(type) {
	case *ast.Ident:
		// FIX MAJOR:  Do the right thing here to prove we should really be accessing iota.
		*iota++
		return *iota, nil
	case *ast.BasicLit:
		if n.Kind != token.INT {
			return unsupported(n.Kind)
		}
		i, _ := strconv.Atoi(n.Value)
		return i, nil
	case *ast.BinaryExpr:
		x, err := eval_int_expr(n.X, iota)
		if err != nil {
			return 0, err
		}
		y, err := eval_int_expr(n.Y, iota)
		if err != nil {
			return 0, err
		}
		switch n.Op {
		case token.ADD:
			return x + y, nil
		case token.SUB:
			return x - y, nil
		case token.MUL:
			return x * y, nil
		case token.QUO:
			return x / y, nil
		case token.REM:
			return x % y, nil
		case token.AND:
			return x & y, nil
		case token.OR:
			return x | y, nil
		case token.XOR:
			return x ^ y, nil
		case token.SHL:
			return x << y, nil
		case token.SHR:
			return x >> y, nil
		case token.AND_NOT:
			return x &^ y, nil
		default:
			return unsupported(n.Op)
		}
	case *ast.ParenExpr:
		return eval_int_expr(n.X, iota)
	}
	return unsupported(reflect.TypeOf(tree))
}

func unsupported(i interface{}) (int, error) {
	return 0, errors.New(fmt.Sprintf("%v is unsupported", i))
}

func filter_parse_nodes() {
}

type declaration_kind struct {
	type_name string
	type_kind string
}

// We want this topological sorting to be stable with respect to ordering of declarations which
// are essentially in the same equivalence class.  Specifically, declarations in the same class
// should be ordered as much as possible in the same order as they appear in the input file.
// That's why we pass in the *_nodes maps, to have access to the declaration position info.
//
func topologically_sort_nodes(
	simple_typedefs map[string]string,
	enum_typedefs map[string]string,
	const_groups map[string]string,
	struct_typedefs map[string][]string,
	simple_typedef_nodes map[string]*ast.GenDecl,
	enum_typedef_nodes map[string]*ast.GenDecl,
	const_group_nodes map[string]*ast.GenDecl,
	struct_typedef_nodes map[string]*ast.GenDecl,
) (
	final_type_order []declaration_kind,
	err error,
) {
	type type_dependency struct {
		type_kind            string
		type_pos             token.Pos
		depends_on_type_name []string
	}

	// map[type_name]type_dependency
	dependency := map[string]type_dependency{}

	// Output at this stage is only for initial development, to ensure that we have the expected
	// kinds of data at this point.
	for typedef_name, typedef_type := range simple_typedefs {
		if print_diagnostics {
			fmt.Fprintf(diag_file, "simple typedef:  %s => %s\n", typedef_name, typedef_type)
		}
		dependency[typedef_name] = type_dependency{"simple", simple_typedef_nodes[typedef_name].TokPos, []string{typedef_type}}
	}
	for enum_name, enum_type := range enum_typedefs {
		if print_diagnostics {
			fmt.Fprintf(diag_file, "enum typedef:  %s => %s\n", enum_name, enum_type)
		}
		dependency[enum_name] = type_dependency{"enum", enum_typedef_nodes[enum_name].TokPos, []string{enum_type}}
	}
	for const_group_name, const_group_type := range const_groups {
		if print_diagnostics {
			fmt.Fprintf(diag_file, "const group:  %s => %s\n", const_group_name, const_group_type)
		}
		// Here, the TokPos value we provide is just a placeholder.  It does represent the position of the
		// original const group in the source code, but if this const block represents an set of enumeration
		// values, we will later replace that with the position of the enumeration type.  That will force
		// emission of the enumeration values immediately after emission of the enumeration declaration.
		dependency[const_group_name] = type_dependency{"const", const_group_nodes[const_group_name].TokPos, []string{const_group_type}}
	}
	for struct_name, struct_field_type_list := range struct_typedefs {
		if print_diagnostics {
			fmt.Fprintf(diag_file, "struct typedef:  %s => %v\n", struct_name, struct_field_type_list)
		}
		dependency[struct_name] = type_dependency{"struct", struct_typedef_nodes[struct_name].TokPos, struct_field_type_list}
	}

	tentative_type_order := make([]string, 0, len(dependency))
	// type_dep here (or at least its type_dep.depends_on_type_name component) apparently ends up
	// as a copy of the type_dependency object (or at least its []string component), not an alias.
	// So when we wish to alter the base data structure, we must refer to it directly.
	for type_name, type_dep := range dependency {
		if print_diagnostics {
			// fmt.Fprintf(diag_file, "=== dep types before filtering: %v\n", type_dep.depends_on_type_name)
		}
		// In this block, we effectively shrink the array of depends-on names.  In the next for loop, because of
		// either aliasing or a level of indirection imposed by the type_dep.depends_on_type_name[] slice which
		// points to the same underlying array as the original item we're iterating over, we directly change the
		// value in that underlying array.  However, we run into trouble later on when we then want to shrink the
		// current length (though not the capacity) of the slice as seen by the actual map element, to correspond
		// to the last element we copied in this loop.
		new_index := 0
		for old_index, dep_type := range type_dep.depends_on_type_name {
			if _, ok := dependency[dep_type]; ok {
				type_dep.depends_on_type_name[new_index] = type_dep.depends_on_type_name[old_index]
				new_index++
			}
		}

		// The next thing we want to do is to shorten the length of the dependency[type_name].depends_on_type_name
		// slice, without copying anything or doing anything to the underlying array.  But I don't know of any way
		// to adjust a slice length downward without some sort of array copy going on.
		//
		// This is the most direct way to express what we want:  just modify the array langth of interest, and leave
		// everything else intact.  But Go won't allow us to treat the slice length as an lvalue, as Perl does.  So
		// this attempt fails.  Is there some other way to express this same adjustment?
		// len(dependency[type_name].depends_on_type_name) = new_index
		//
		// Here is another way to attempt something close to that, by redefining the entire slice.  But Go won't let
		// us do this, even in a single-threaded program.
		// dependency[type_name].depends_on_type_name = dependency[type_name].depends_on_type_name[:new_index]

		// This next statement shortens the slice we operated on above (and that already affected individual values
		// in the underlying array which is shared by the dependency[type_name].depends_on_type_name slice).  But it
		// does not change the length of that original slice.
		type_dep.depends_on_type_name = type_dep.depends_on_type_name[:new_index]

		// This direct assignment of the slice itself doesn't work, not unlike the earlier attempt at redefining the
		// entire slice.
		// dependency[type_name].depends_on_type_name = type_dep.depends_on_type_name

		// This copying of the map element's entire struct-value does work, instead of trying to modify just one field
		// of the map element's struct.  Why Go imposes that restriction, I don't know.  As a general solution to this
		// problem, I certainly don't like having to copy extra data, when a much smaller adjustment ought to have been
		// possible.  But we'll run with this for now, until and unless we find some way to effectively use the slice
		// length as an lvalue and just shorten the map-element's own copy of the slice length.
		dependency[type_name] = type_dep

		// Next, we need to sort type_dep.depends_on_type_name by increasing values of their
		// respective dependency[type_dep.depends_on_type_name[_]].type_pos fields.
		sort.Slice(type_dep.depends_on_type_name, func(i int, j int) bool {
			return dependency[type_dep.depends_on_type_name[i]].type_pos < dependency[type_dep.depends_on_type_name[j]].type_pos
		})

		// Next, we replace the effective source-code position of "const" declarations by the
		// respective source-code positions of the types of their values.  This pulls up the
		// const definition to be coincident with the declarations of their respective value
		// types.  Topological sorting will disambiguate in a stable manner by using the
		// dependency of the const type on the value type to force the value-type declaration
		// to precede the const-type declaration.
		if type_dep.type_kind == "const" && len(type_dep.depends_on_type_name) == 1 {
			type_dep.type_pos = dependency[type_dep.depends_on_type_name[0]].type_pos
		}

		if print_diagnostics {
			// fmt.Fprintf(diag_file, "   base typedef:  %#v %#v\n", type_name, dependency[type_name])
			// fmt.Fprintf(diag_file, "generic typedef:  %#v %#v\n", type_name, type_dep)
		}

		tentative_type_order = append(tentative_type_order, type_name)
	}

	// Finally, we create a []string array derived from "dependency" which contains all of its keys
	// but has them all sorted by increasing values of the respective dependency[_].type_pos fields.
	// This is the order in which we will process the type names in our topological sort loop.
	sort.Slice(tentative_type_order, func(i int, j int) bool {
		return dependency[tentative_type_order[i]].type_pos < dependency[tentative_type_order[j]].type_pos
	})

	if print_diagnostics {
		for _, type_name := range tentative_type_order {
			fmt.Fprintf(diag_file, "sorted generic typedef:  %#v %#v\n", type_name, dependency[type_name])
		}
	}

	// Because C does not allow forward references (other than perhaps for pointers), we need to output
	// the declarations in a fixed order that will satisfy all dependencies between the types between
	// categories.  This order might not reflect the order of declarations in the original Go source
	// code, because Go seems to use multi-pass parsing in order to resolve references to types that
	// have not yet been declared when they are referenced in the code.

	// Sort the type names into a topological order, paying attention to type dependencies
	// while also moving const groups to immediately after the declarations of their value
	// types and otherwise preserving as much as possible the order of type declarations as
	// occurred in the source file.
	final_type_order = make([]declaration_kind, 0, len(tentative_type_order))
	has_been_seen := map[string]bool{}
	var process_type_names func(type_names []string)
	process_type_names = func(type_names []string) {
		for _, type_name := range type_names {
			if !has_been_seen[type_name] {
				has_been_seen[type_name] = true
				process_type_names(dependency[type_name].depends_on_type_name)
				final_type_order = append(final_type_order, declaration_kind{type_name, dependency[type_name].type_kind})
			}
		}
	}
	process_type_names(tentative_type_order)

	if print_diagnostics {
		for _, decl_kind := range final_type_order {
			fmt.Fprintf(diag_file, "final sorted generic typedef:  %#v %#v %#v\n", decl_kind.type_name, decl_kind.type_kind, dependency[decl_kind.type_name])
		}
	}

	return final_type_order, err
}

var C_header_header_boilerplate = `//
// {{.HeaderFilename}}
//
// DO NOT EDIT THIS FILE.  It is auto-generated from other code,
// and your edits will be lost.
//
// Copyright (c) {{.Year}} GroundWork Open Source, Inc. (www.gwos.com)
// Use of this software is subject to commercial license terms.
//

#ifndef {{.HeaderSymbol}}
#define {{.HeaderSymbol}}

#ifdef  __cplusplus
extern "C" {
#endif

#include <stdbool.h>    // as of C99, provides the "bool" datatype, along with "true" and "false" macros
#include <stdint.h>     // as or C99, provides "int32_t" and "int64_t" datatypes
#include <time.h>       // to supply "struct timespec", with time_t tv_sec (seconds) and long tv_nsec (nanoseconds) members

#ifndef NUL_TERM_LEN
// Size of a NUL-termination byte.  Generally useful for documenting the
// meaning of +1 and -1 length adjustments having to do with such bytes.
#define NUL_TERM_LEN 1  // sizeof('\0')
#endif  // NUL_TERM_LEN

// typedef int int;    // Go's "int" type is at least 32 bits in size; that might or might not be identical to C's "int" type
typedef int64_t int64;
typedef double  float64;
typedef int32_t int32;
typedef struct timespec struct_timespec;

#ifndef string
// Make a simple global substitution using the C preprocessor, so we don't
// have to deal with this ad infinitum in the language-conversion code.
#define string char *
#endif  // string

// --------------------------------------------------------------------------------
// FIX QUICK:  The content of this comment is obsolete.  It needs to be replaced
// with comments about the routines we actually expect the application code to use.
// --------------------------------------------------------------------------------
// Each encode_PackageName_StructTypeName_as_json() routine declared in this header file:
//
//     extern char *encode_PackageName_StructTypeName_as_json(const PackageName_StructTypeName *StructTypeName_ptr, size_t flags);
//
// returns the JSON representation of the structure as a string, or NULL on error.
// The returned string must ultimately be deallocated by the caller using a single
// call to free().  The flags are described here:
//
//     https://jansson.readthedocs.io/en/2.12/apiref.html#encoding
//
// The JSON_SORT_KEYS flag is used by default.  This is mostly for initial
// development purposes; we might not want the sorting overhead in production.
// --------------------------------------------------------------------------------
// Each decode_json_PackageName_StructTypeName() routine declared in this header file:
//
//     extern StructTypeName *decode_json_PackageName_StructTypeName(const char *json_str);
//
// returns a pointer to a new object, or NULL on error.  The returned object must
// ultimately be deallocated by the caller using a single call to this routine:
//
//     extern void free_PackageName_StructTypeName_ptr_tree(PackageName_StructTypeName *StructTypeName_ptr, json_t *json);
//
// That one call will at the same time free memory for all of the connected
// subsidary objects.
//
// Note that a similar routine:
//
//     extern void destroy_PackageName_StructTypeName_ptr_tree(PackageName_StructTypeName *PackageName_StructTypeName_ptr, json_t *json, bool free_pointers);
//
// is also available.  It has a very similar purpose, but it is intended for use
// with a tree of data structures which are manually allocated in application code,
// where the individual parts are likely not contiguous in memory.  In contrast, the
// free_StructTypeName_ptr_tree() implementation will be kept definitively matched to
// the decode_json_PackageName_StructTypeName() implementation.  So whether the decoding
// creates just a single large block of memory that contains not only the initial
// StructTypeName object but all of the subsidiary objects it recursively refers to,
// or whether it splays things out via independently floating allocations, a call to
// free_StructTypeName_ptr_tree() is guaranteed to match the internal requirements of
// releasing all of the memory allocated by decode_json_PackageName_StructTypeName().
// --------------------------------------------------------------------------------

`

var C_header_footer_boilerplate = `#ifdef  __cplusplus
}
#endif

#endif // {{.HeaderSymbol}}
`

var C_code_boilerplate = `//
// {{.CodeFilename}}
//
// DO NOT EDIT THIS FILE.  It is auto-generated from other code,
// and your edits will be lost.
//
// Copyright (c) {{.Year}} GroundWork Open Source, Inc. (www.gwos.com)
// Use of this software is subject to commercial license terms.
//

// Our JSON encoding and decoding of C structures depends on the Jansson library
// for a lot of low-level detail.  See:
//
//     http://www.digip.org/jansson/
//     https://github.com/akheron/jansson
//     https://jansson.readthedocs.io/
//
// FIX MINOR:  This inclusion might be better moved to the header file,
// if json_dumps() is expected to be used by application code.
#include "jansson.h"

#include "convert_go_to_c.h"

#include <stdlib.h>    // for the declaration of free(), at least
// #include <stdalign.h>  // Needed to supply alignof(), available starting with C11.
// #include <stddef.h>
// #include <string.h>

{{.OtherHeaders}}
#include "{{.HeaderFilename}}"

// FIX MAJOR:  Also include in here some initialization of the conversion library,
// so we can pass our logger to the package and have it use that for all error logging.

`

func print_type_declarations(
	package_name string,
	final_type_order []declaration_kind,
	simple_typedefs map[string]string,
	enum_typedefs map[string]string,
	const_groups map[string]string,
	struct_typedefs map[string][]string,
	struct_field_typedefs map[string]map[string]string,
	simple_typedef_nodes map[string]*ast.GenDecl,
	enum_typedef_nodes map[string]*ast.GenDecl,
	const_group_nodes map[string]*ast.GenDecl,
	struct_typedef_nodes map[string]*ast.GenDecl,
) (
	pointer_base_types map[string]string,
	pointer_list_base_types []string,
	simple_list_base_types []string,
	list_base_types []string,
	key_value_pair_types map[string][]string,
	struct_fields map[string][]string,
	struct_field_Go_types map[string]map[string]string,
	struct_field_C_types map[string]map[string]string,
	struct_field_tags map[string]map[string]string,
	generated_C_code string,
	err error,
) {
	package_defined_type := map[string]bool{}
	for key, _ := range simple_typedefs {
		if print_diagnostics {
			fmt.Fprintf(diag_file, "+++ simple typedef for %s\n", key)
		}
		package_defined_type[key] = true
	}
	for key, _ := range enum_typedefs {
		if print_diagnostics {
			fmt.Fprintf(diag_file, "+++   enum typedef for %s\n", key)
		}
		package_defined_type[key] = true
	}
	for key, _ := range struct_typedefs {
		if print_diagnostics {
			fmt.Fprintf(diag_file, "+++ struct typedef for %s\n", key)
		}
		package_defined_type[key] = true
	}

	// Hash of name of secondary pointer base types we have needed to generate a typedef for.
	have_ptr_type := map[string]bool{}
	// Hash of name of simple list base types we need to generate conversion code for.
	have_simple_list_type := map[string]bool{}
	// Hashes of names of secondary structs which we have needed to generate.
	have_list_struct := map[string]bool{}
	have_pair_structs := map[string]bool{}
	// Precompiled regular expressions to match against the package name and typenames.
	dot := regexp.MustCompile(`\.`)
	slash := regexp.MustCompile(`/`)
	leading_array := regexp.MustCompile(`^\[\]`)
	leading_star := regexp.MustCompile(`^\*`)
	// This expression ought to be generalized to check for balanced [] characters within the map key.
	map_key_value_types := regexp.MustCompile(`map\[([^]]+)\](.+)`)

	pointer_base_types = map[string]string{}
	// FIX QUICK:  Do we need to initialize these slices of strings?  If not, why not?
	// pointer_list_base_types = []string{}
	// simple_list_base_types  = []string{}
	// list_base_types         = []string{}
	key_value_pair_types = map[string][]string{}
	struct_fields = map[string][]string{}
	struct_field_Go_types = map[string]map[string]string{}
	struct_field_C_types = map[string]map[string]string{}
	struct_field_tags = map[string]map[string]string{}

	header_boilerplate := template.Must(template.New("header_header").Parse(C_header_header_boilerplate))
	footer_boilerplate := template.Must(template.New("header_footer").Parse(C_header_footer_boilerplate))

	type C_header_boilerplate_fields struct {
		Year           int
		HeaderFilename string
		HeaderSymbol   string
	}

	current_year := time.Now().Year()
	header_filename := package_name + ".h"
	header_filepath := filepath.Join(output_directory, header_filename)
	header_symbol := "_" + strings.ToUpper(slash.ReplaceAllLiteralString(package_name, "_")) + "_H"
	boilerplate_variables := C_header_boilerplate_fields{Year: current_year, HeaderFilename: header_filename, HeaderSymbol: header_symbol}

	header_file, err := os.Create(header_filepath)
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := header_file.Close(); err != nil {
			panic(err)
		}
	}()

	if err = header_boilerplate.Execute(header_file, boilerplate_variables); err != nil {
		panic("C header-file header processing failed")
	}

	trailing_Ptr_List := regexp.MustCompile(`(.+)_Ptr_List$`)

	for _, decl_kind := range final_type_order {
		if print_diagnostics {
			// fmt.Fprintf(diag_file, "processing type %s %s\n", decl_kind.type_name, decl_kind.type_kind)
		}
		switch decl_kind.type_kind {
		case "simple":
			type_name := simple_typedefs[decl_kind.type_name]
			// FIX LATER:  This code handles a _Ptr_List; if we ever need to process Go code that requires
			// a simple _List or _Ptr, we'll need to extend this code to cover those cases as well.
			if matches := trailing_Ptr_List.FindStringSubmatch(type_name); matches != nil {
				base_type := matches[1]
				array_base_type := base_type
				// FIX MINOR:  Perhaps this is not really the right place for this.
				if package_defined_type[array_base_type] {
					array_base_type = package_name + "_" + array_base_type
				}
				array_base_type_ptr := array_base_type + "_Ptr"
				if !have_ptr_type[array_base_type] {
					have_ptr_type[array_base_type] = true
					fmt.Fprintf(header_file, "typedef %s *%[1]s_Ptr;\n", array_base_type)
					fmt.Fprintf(header_file, "\n")
					pointer_list_base_types = append(pointer_list_base_types, array_base_type)
					pointer_base_types[array_base_type_ptr] = array_base_type
				}
				array_base_type = array_base_type_ptr
				list_type := array_base_type + "_List"
				if !have_list_struct[list_type] {
					have_list_struct[list_type] = true
					fmt.Fprintf(header_file, "typedef struct _%s_ {\n", list_type)
					fmt.Fprintf(header_file, "    size_t count;\n")
					fmt.Fprintf(header_file, "    %s *items;\n", array_base_type)
					fmt.Fprintf(header_file, "} %s;\n", list_type)
					fmt.Fprintf(header_file, "\n")
					fmt.Fprintf(header_file, "extern bool is_%[1]s_ptr_zero_value(const %[1]s *%[1]s_ptr);\n", list_type)
					fmt.Fprintf(header_file, "#define is_%s_%s_ptr_zero_value is_%s_ptr_zero_value\n", package_name, decl_kind.type_name, list_type)
					fmt.Fprintf(header_file, "#define %s_%s_ptr_as_JSON_ptr %s_ptr_as_JSON_ptr\n", package_name, decl_kind.type_name, list_type)
					fmt.Fprintf(header_file, "#define JSON_as_%s_%s_ptr JSON_as_%s_ptr\n", package_name, decl_kind.type_name, list_type)
					fmt.Fprintf(header_file, "\n")
					struct_fields[list_type] = append(struct_fields[list_type], "count")
					struct_fields[list_type] = append(struct_fields[list_type], "items")
					struct_field_C_types[list_type] = map[string]string{}
					struct_field_C_types[list_type]["count"] = "size_t"
					struct_field_C_types[list_type]["items"] = array_base_type + " *"
					list_base_types = append(list_base_types, array_base_type)
				}
			}
			fmt.Fprintf(header_file, "typedef %s %s_%s;\n", type_name, package_name, decl_kind.type_name)
			fmt.Fprintf(header_file, "\n")
		case "enum":
			//
			//  We want this Go code:
			//
			//      type UnitEnum string
			//
			//      const (
			//          NoUnits     UnitEnum = ""       // no units specified
			//          UnitCounter          = "1"      // unspecified-unit counter
			//          PercentCPU           = "%{cpu}" // percent CPU, as in load measurements
			//      )
			//
			//  to translate to the following C code.  Note that the string values for the integer
			//  enumeration constants might not just be quoted versions of those same names.
			//
			//      // ----------------------------------------------------------------
			//
			//      // this_package.h:
			//
			//      // generated solely by "enum" processing
			//      extern const string const UnitEnum_String[];  // index by UnitEnum numeric enumeration value, get associated string value back
			//      typedef enum UnitEnum UnitEnum;
			//
			//      // generated solely by "const" processing
			//      enum UnitEnum {
			//          NoUnits,
			//          UnitCounter,
			//          PercentCPU,
			//      };
			//
			//      // ----------------------------------------------------------------
			//
			//      // this_package.c:
			//
			//      // generated solely by "const" processing
			//      const string const UnitEnum_String[] = {
			//          "",            // no units specified
			//          "1",           // unspecified-unit counter
			//          "%{cpu}",      // percent CPU, as in load measurements
			//      };
			//
			//      // ----------------------------------------------------------------
			//
			//  That syntax will allow our enum-before-const processing to have all the data it needs at each step.
			//
			fmt.Fprintf(header_file, "typedef enum %s %s_%[1]s;\n", decl_kind.type_name, package_name)
			// FIX MINOR:  We should probably include the package name in the zero-value macro name.
			// The main concern there would then be to make sure we don't cause any issues with the means we
			// presently use to construct names for calling is_int_ptr_zero_value(), is_string_ptr_zero_value(),
			// and similar base-type functions.
			fmt.Fprintf(header_file, "#define is_%s_ptr_zero_value(enum_ptr) (enum_ptr == NULL || *enum_ptr == 0)\n", decl_kind.type_name)
			if enum_typedefs[decl_kind.type_name] == "string" {
				fmt.Fprintf(header_file, "extern const string const %s_%s_String[];\n", package_name, decl_kind.type_name)
			}
			fmt.Fprintf(header_file, "\n")
		case "const":
			decl_node := const_group_nodes[decl_kind.type_name]
			field_C_type := const_groups[decl_kind.type_name]
			fmt.Fprintf(header_file, "enum %s {\n", field_C_type)
			//
			// There's a question here of how to support enumerations that use iota expressions to define the
			// values of the particular enumeration identifiers.  With respect to indexing into some data structure
			// in the C code to find a value to specify in JSON code, it turns out that the only truly useful value
			// we can specify in that case is a bare "iota" expression itself, with it specified for the very first
			// member of the enumeration, so there is an exact identity mapping between the enumeration identifier
			// and data structure indexes.  Anything else would be very difficult to support, given that an expression
			// like "1 << iota" would translate into a huge array index into the structure for anything beyond the
			// first few items in the enumeration.
			//
			// Net of that, there's nothing wrong with supporting enumerations that do use iota in that very limited
			// way, but even in that case we're probably best off not using a "pkg_Enum_String[]" array at all.  We
			// don't want strings in this case anyway, we want numbers for an enumeration which is truly numeric-valued
			// even in its JSON representation.  And an identity map doesn't buy us anything.  So what we do instead
			// is to recognize that we're processing an enumeration with a numeric-valued JSON representation, suppress
			// generation of the "pkg_Enum_String[]" array in the header file (above) and C code, and recognize the
			// enumeration type in the code that deals with JSON tranfers.
			//
			if enum_typedefs[field_C_type] == "string" {
				generated_C_code += fmt.Sprintf("const string const %s_%s_String[] = {\n", package_name, field_C_type)
			}
			// Evidently there will typically be one spec per enumeration element,
			// so we assign the initial value of iota outside of this loop.
			iota := 0
			if enum_typedefs[field_C_type] == "string" {
				fmt.Fprintf(header_file, "    /* %d */ Unknown_%s_Value,\n", iota, field_C_type)
				generated_C_code += fmt.Sprintf("    /* %d */ \"Unknown_%s_Value\",\n", iota, field_C_type)
				iota++
			}
			for _, spec := range decl_node.Specs {
				// This processing could be more complex, if there are other name-node types we might encounter here.
				// Note that the Name array may have more elements than the Values array.  For convenience in programming,
				// we don't iterate separately over the values, but just assume that the Values array will never have
				// more elements than the Name array.  (That wouldn't make sense anyway.)
				for enumindex, name := range spec.(*ast.ValueSpec).Names {
					if enum_typedefs[field_C_type] == "string" {
						fmt.Fprintf(header_file, "    /* %d */ %s,\n", iota, name.Name)
						if enumindex < len(spec.(*ast.ValueSpec).Values) {
							value := spec.(*ast.ValueSpec).Values[enumindex]
							// This is a string literal that we will use as the JSON value of an enumeration constant.
							generated_C_code += fmt.Sprintf("    /* %d */ %s,\n", iota, value.(*ast.BasicLit).Value)
						}
					} else {
						// Logically we don't need to specify the value if the expression is just "iota", but we're
						// preparing here for eventual possible evaluation of more complex expressions like "1 << iota".
						// Also, it allows the generated header file to document whatever numeric values we might see
						// in non-JSON data structures we might need to debug.
						fmt.Fprintf(header_file, "    %s = %d,\n", name.Name, iota)
					}
					iota++
				}
			}
			fmt.Fprintf(header_file, "};\n")
			fmt.Fprintf(header_file, "\n")
			if enum_typedefs[field_C_type] == "string" {
				generated_C_code += fmt.Sprintf("};\n")
				generated_C_code += fmt.Sprintf("\n")
			}
		case "struct":
			decl_node := struct_typedef_nodes[decl_kind.type_name]
			struct_field_Go_types[decl_kind.type_name] = map[string]string{}
			struct_field_C_types[decl_kind.type_name] = map[string]string{}
			struct_field_tags[decl_kind.type_name] = map[string]string{}
			var struct_definition string
			for _, spec := range decl_node.Specs {
				//
				// We don't print the structure definition immediately here, as we step through its fields.
				// Instead, we queue the definition in case we need to generate some secondary structures to
				// represent variable-length arrays of other types.  The things we know we might encounter
				// as struct field types are:
				//
				//      []typename  variable-length array of objects
				//     []*typename  variable-length array of pointers to objects
				//     *[]typename  pointer to variable-length array of objects
				//       *typename  pointer an object
				//     map[keytypename]valuetypename  map from key to value
				//
				// We're not trying here yet to be totally general.  Those are the things we need to handle
				// in the short term, and we must panic if we encounter anything else so we at least don't
				// generate incorrect output.  This code can be extended as needed in the future.
				//
				// ----------------------------------------------------------------
				//
				// A Go "fieldname []typename" structure field will be turned into the following C code:
				//
				//     typename_List fieldname;  // go: fieldname []typename
				//
				// with a preceding structure generated, if it has not already been generated:
				//
				//     typedef struct {
				// 	       size_t count;
				// 	       typename *items;
				//     } typename_List;
				//
				// ----------------------------------------------------------------
				//
				// A Go "fieldname []*typename" structure field will be turned into the following C code:
				//
				//     typename_Ptr_List fieldname;  // go: fieldname []*typename
				//
				// with preceding declaration and structure generated, if they have not already been generated:
				//
				//     typedef typename *typename_Ptr;
				//
				//     typedef struct {
				// 	       size_t count;
				// 	       typename_Ptr *items;
				//     } typename_Ptr_List;
				//
				// ----------------------------------------------------------------
				//
				// A Go "fieldname *[]typename" structure field will be turned into the following C code:
				//
				//     typename_List_Ptr fieldname;  // go: fieldname *[]typename
				//
				// with preceding structure and declaration generated, if they have not already been generated:
				//
				//     typedef struct {
				// 	       size_t count;
				// 	       typename *items;
				//     } typename_List;
				//
				//     typedef typename_List *typename_List_Ptr;
				//
				// ----------------------------------------------------------------
				//
				// A Go "fieldname *typename" structure field will be turned into the following C code:
				//
				//     typename_Ptr fieldname;  // go: fieldname *typename
				//
				// with preceding declaration generated, if it has not already been generated:
				//
				//     typedef typename *typename_Ptr;
				//
				// ----------------------------------------------------------------
				//
				// A Go "fieldname map[keytypename]valuetypename" structure field will be turned into the following C code:
				//
				//     keytypename_valuetypename_Pair_List fieldname
				//
				// with preceding structures generated, if they have not already been generated:
				//
				//     typedef struct {
				//         keytypename key;
				//         valuetypename value;
				//     } keytypename_valuetypename_Pair;
				//
				//     typedef struct {
				//         size_t count;
				//         keytypename_valuetypename_Pair *items;
				//     } keytypename_valuetypename_Pair_List;
				//
				// This construction depends on having just simple alphanumeric type names for the key and value.
				// If we ever need to process code with more-complex key and value types, this code will need to
				// be extended.
				//
				// ----------------------------------------------------------------
				//
				// Logically, we don't really need to make the struct here, but it may help later on with compilation
				// error messages in application code.
				struct_definition = fmt.Sprintf("typedef struct _%s_%s_ {\n", package_name, decl_kind.type_name)
				for _, field := range spec.(*ast.TypeSpec).Type.(*ast.StructType).Fields.List {
					var field_tag string
					if field.Tag != nil {
						// The field.Tag.Value we see here includes whatever form of enclosing quoting was used in the
						// source code, and also includes whatever form of internal quote-escaping was used there as well.
						// That's all very inconvenient for later processing, so we eliminate all the extra layers of
						// quoting now in this central location, only saving away what we consider to be the raw string.
						field_tag, err = strconv.Unquote(field.Tag.Value)
						if err != nil {
							// This indicates a problem with the source code we're analyzing.  If we ever see this,
							// then it's time to fancify the error message we produce here to identify exactly where
							// in the source code the problem occurred.
							panic(err)
						}
					}
					if print_diagnostics {
						fmt.Fprintf(diag_file, "struct %s field tag:  %s\n", decl_kind.type_name, field_tag)
					}
					for _, name := range field.Names {
						switch field.Type.(type) {
						case *ast.Ident:
							type_name := field.Type.(*ast.Ident).Name
							if package_defined_type[type_name] {
								struct_definition += fmt.Sprintf("    %s_%s %s;\n", package_name, type_name, name.Name)
							} else {
								struct_definition += fmt.Sprintf("    %s %s;\n", type_name, name.Name)
							}
							struct_fields[decl_kind.type_name] = append(struct_fields[decl_kind.type_name], name.Name)
							if _, ok := struct_typedefs[type_name]; ok {
								struct_field_C_types[decl_kind.type_name][name.Name] = package_name + "_" + type_name
							} else if _, ok := simple_typedefs[type_name]; ok {
								struct_field_C_types[decl_kind.type_name][name.Name] = package_name + "_" + type_name
							} else {
								struct_field_C_types[decl_kind.type_name][name.Name] = type_name
							}
							struct_field_tags[decl_kind.type_name][name.Name] = field_tag
						case *ast.SelectorExpr:
							go_type := struct_field_typedefs[decl_kind.type_name][name.Name]
							type_selectorexpr := field.Type.(*ast.SelectorExpr)
							var x_type_ident *ast.Ident
							var ok bool
							if x_type_ident, ok = type_selectorexpr.X.(*ast.Ident); ok {
								if print_diagnostics {
									// fmt.Fprintf(diag_file, "    --- struct field name and SelectorExpr X:  %#v %#v\n", name.Name, x_type_ident.Name)
								}
							} else {
								if print_diagnostics {
									fmt.Fprintf(diag_file, "ERROR:  when autovivifying at %s, found unexpected field.Type.X type:  %T\n",
										file_line(), type_selectorexpr.X)
									fmt.Fprintf(diag_file, "ERROR:  struct field Type.X field is not of a recognized type\n")
								}
								// panic_message = "aborting due to previous errors"
								// break node_loop
							}
							if type_selectorexpr.Sel == nil {
								if print_diagnostics {
									fmt.Fprintf(diag_file, "ERROR:  when autovivifying at %s, struct field Type Sel field is unexpectedly nil\n", file_line())
								}
								// panic_message = "aborting due to previous errors"
								// break node_loop
							}
							name_ident := new(ast.Ident)
							name_ident.Name = x_type_ident.Name + "_" + type_selectorexpr.Sel.Name

							// special handling for "struct timespec"
							if name_ident.Name == "time_Time" {
								name_ident.Name = "struct_timespec"
							}

							struct_definition += fmt.Sprintf("    %s %s;  // go: %s\n", name_ident.Name, name.Name, go_type)
							// struct_definition += fmt.Sprintf("    %s %s;  // go: %[1]s\n", go_type, name)
							struct_fields[decl_kind.type_name] = append(struct_fields[decl_kind.type_name], name.Name)
							struct_field_C_types[decl_kind.type_name][name.Name] = name_ident.Name
							struct_field_tags[decl_kind.type_name][name.Name] = field_tag
						case *ast.StarExpr:
							go_type := struct_field_typedefs[decl_kind.type_name][name.Name]
							star_base_type := leading_star.ReplaceAllLiteralString(go_type, "")
							var base_type string
							if leading_array.MatchString(star_base_type) {
								star_base_type = leading_array.ReplaceAllLiteralString(star_base_type, "")
								if package_defined_type[star_base_type] {
									star_base_type = package_name + "_" + star_base_type
								}
								list_type := star_base_type + "_List"
								if !have_list_struct[list_type] {
									have_list_struct[list_type] = true
									fmt.Fprintf(header_file, "typedef struct _%s_ {\n", list_type)
									fmt.Fprintf(header_file, "    size_t count;\n")
									fmt.Fprintf(header_file, "    %s *items;\n", star_base_type)
									fmt.Fprintf(header_file, "} %s;\n", list_type)
									fmt.Fprintf(header_file, "\n")
									fmt.Fprintf(header_file, "extern bool is_%[1]s_List_ptr_zero_value(const %[1]s_List *%[1]s_List_ptr);\n",
										star_base_type)
									fmt.Fprintf(header_file, "\n")
									struct_fields[list_type] = append(struct_fields[list_type], "count")
									struct_fields[list_type] = append(struct_fields[list_type], "items")
									struct_field_C_types[list_type] = map[string]string{}
									struct_field_C_types[list_type]["count"] = "size_t"
									struct_field_C_types[list_type]["items"] = star_base_type + " *"
									if !have_simple_list_type[star_base_type] {
										have_simple_list_type[star_base_type] = true
										simple_list_base_types = append(simple_list_base_types, star_base_type)
									}
									list_base_types = append(list_base_types, star_base_type)
								}
								base_type = list_type
							} else {
								if package_defined_type[star_base_type] {
									star_base_type = package_name + "_" + star_base_type
								}
								base_type = star_base_type
							}
							base_type = dot.ReplaceAllLiteralString(base_type, "_")
							if package_defined_type[base_type] {
								base_type = package_name + "_" + base_type
							}
							base_type_ptr := base_type + "_Ptr"
							if !have_ptr_type[base_type] {
								have_ptr_type[base_type] = true
								fmt.Fprintf(header_file, "typedef %s *%[1]s_Ptr;\n", base_type)
								fmt.Fprintf(header_file, "\n")
								pointer_base_types[base_type_ptr] = base_type
							}
							field_type := base_type_ptr
							struct_definition += fmt.Sprintf("    %s %s;  // go: %s\n", field_type, name.Name, go_type)
							struct_fields[decl_kind.type_name] = append(struct_fields[decl_kind.type_name], name.Name)
							struct_field_C_types[decl_kind.type_name][name.Name] = field_type
							struct_field_Go_types[decl_kind.type_name][name.Name] = go_type
							struct_field_tags[decl_kind.type_name][name.Name] = field_tag
						case *ast.ArrayType:
							// The constructions here are limited to what we have encountered in our own code.
							// A more general conversion would handle additional array types, probably by some form of recursion.
							go_type := struct_field_typedefs[decl_kind.type_name][name.Name]
							array_base_type := leading_array.ReplaceAllLiteralString(go_type, "")
							if leading_star.MatchString(array_base_type) {
								array_base_type = leading_star.ReplaceAllLiteralString(array_base_type, "")
								// FIX MINOR:  Perhaps this is not really the right place for this.
								if package_defined_type[array_base_type] {
									array_base_type = package_name + "_" + array_base_type
								}
								array_base_type_ptr := array_base_type + "_Ptr"
								if !have_ptr_type[array_base_type] {
									have_ptr_type[array_base_type] = true
									fmt.Fprintf(header_file, "typedef %s *%[1]s_Ptr;\n", array_base_type)
									fmt.Fprintf(header_file, "\n")
									pointer_list_base_types = append(pointer_list_base_types, array_base_type)
									pointer_base_types[array_base_type_ptr] = array_base_type
								}
								array_base_type = array_base_type_ptr
							} else {
								if package_defined_type[array_base_type] {
									array_base_type = package_name + "_" + array_base_type
								}
								if !have_simple_list_type[array_base_type] {
									have_simple_list_type[array_base_type] = true
									simple_list_base_types = append(simple_list_base_types, array_base_type)
								}
							}
							list_type := array_base_type + "_List"
							if !have_list_struct[list_type] {
								have_list_struct[list_type] = true
								fmt.Fprintf(header_file, "typedef struct _%s_ {\n", list_type)
								fmt.Fprintf(header_file, "    size_t count;\n")
								fmt.Fprintf(header_file, "    %s *items;\n", array_base_type)
								fmt.Fprintf(header_file, "} %s;\n", list_type)
								fmt.Fprintf(header_file, "\n")
								fmt.Fprintf(header_file, "extern bool is_%[1]s_List_ptr_zero_value(const %[1]s_List *%[1]s_List_ptr);\n",
									array_base_type)
								fmt.Fprintf(header_file, "\n")
								struct_fields[list_type] = append(struct_fields[list_type], "count")
								struct_fields[list_type] = append(struct_fields[list_type], "items")
								struct_field_C_types[list_type] = map[string]string{}
								struct_field_C_types[list_type]["count"] = "size_t"
								struct_field_C_types[list_type]["items"] = array_base_type + " *"
								list_base_types = append(list_base_types, array_base_type)
							}
							struct_definition += fmt.Sprintf("    %s %s;  // go: %s\n", list_type, name.Name, go_type)
							struct_fields[decl_kind.type_name] = append(struct_fields[decl_kind.type_name], name.Name)
							struct_field_C_types[decl_kind.type_name][name.Name] = list_type
							struct_field_tags[decl_kind.type_name][name.Name] = field_tag
						case *ast.MapType:
							go_type := struct_field_typedefs[decl_kind.type_name][name.Name]
							field_type := go_type
							key_value_types := map_key_value_types.FindStringSubmatch(field_type)
							if key_value_types == nil {
								panic(fmt.Sprintf("found incomprehensible map construction '%s'", field_type))
							}
							key_type := key_value_types[1]
							value_type := key_value_types[2]
							if package_defined_type[key_type] {
								key_type = package_name + "_" + key_type
							}
							if package_defined_type[value_type] {
								value_type = package_name + "_" + value_type
							}
							type_pair_type := key_type + "_" + value_type + "_Pair"
							type_pair_list_type := type_pair_type + "_List"
							if !have_pair_structs[type_pair_type] {
								have_pair_structs[type_pair_type] = true
								fmt.Fprintf(header_file, "typedef struct _%s_ {\n", type_pair_type)
								fmt.Fprintf(header_file, "    %s key;\n", key_type)
								fmt.Fprintf(header_file, "    %s value;\n", value_type)
								fmt.Fprintf(header_file, "} %s;\n", type_pair_type)
								fmt.Fprintf(header_file, "\n")
								fmt.Fprintf(header_file, "#define make_empty_%s_array(n) (%[1]s *) calloc((n), sizeof (%[1]s))\n", type_pair_type)
								fmt.Fprintf(header_file, "#define make_empty_%s() make_empty_%[1]s_array(1)\n", type_pair_type)
								fmt.Fprintf(header_file, "extern bool     is_%[1]s_ptr_zero_value(const %[1]s *%[1]s_ptr);\n", type_pair_type)
								fmt.Fprintf(header_file, "\n")
								struct_fields[type_pair_type] = append(struct_fields[type_pair_type], "key")
								struct_fields[type_pair_type] = append(struct_fields[type_pair_type], "value")
								struct_field_C_types[type_pair_type] = map[string]string{}
								struct_field_C_types[type_pair_type]["key"] = key_type
								struct_field_C_types[type_pair_type]["value"] = value_type
								fmt.Fprintf(header_file, "typedef struct _%s_ {\n", type_pair_list_type)
								fmt.Fprintf(header_file, "    size_t count;\n")
								fmt.Fprintf(header_file, "    %s *items;\n", type_pair_type)
								fmt.Fprintf(header_file, "} %s;\n", type_pair_list_type)
								fmt.Fprintf(header_file, "\n")
								fmt.Fprintf(header_file, "extern bool is_%[1]s_ptr_zero_value(const %[1]s *%[1]s_ptr);\n", type_pair_list_type)
								fmt.Fprintf(header_file, "\n")
								struct_fields[type_pair_list_type] = append(struct_fields[type_pair_list_type], "count")
								struct_fields[type_pair_list_type] = append(struct_fields[type_pair_list_type], "items")
								struct_field_C_types[type_pair_list_type] = map[string]string{}
								struct_field_C_types[type_pair_list_type]["count"] = "size_t"
								struct_field_C_types[type_pair_list_type]["items"] = type_pair_type + " *"
								key_value_pair_types[key_type] = append(key_value_pair_types[key_type], value_type)
							}
							struct_definition += fmt.Sprintf("    %s %s;  // go: %s\n", type_pair_list_type, name.Name, field_type)
							struct_fields[decl_kind.type_name] = append(struct_fields[decl_kind.type_name], name.Name)
							struct_field_C_types[decl_kind.type_name][name.Name] = type_pair_list_type
							struct_field_tags[decl_kind.type_name][name.Name] = field_tag
						default:
							panic(fmt.Sprintf("found unexpected field type %T", field.Type))
						}
					}
				}
				struct_definition += fmt.Sprintf("} %s_%s;\n", package_name, decl_kind.type_name)
				struct_definition += fmt.Sprintf("\n")
			}
			struct_definition += fmt.Sprintf("#define  make_empty_%s_%s_array(n) (%[1]s_%[2]s *) calloc((n), sizeof (%[1]s_%[2]s))\n",
				package_name, decl_kind.type_name)
			struct_definition += fmt.Sprintf("#define  make_empty_%s_%s() make_empty_%[1]s_%[2]s_array(1)\n",
				package_name, decl_kind.type_name)
			struct_definition += fmt.Sprintf("extern bool      is_%s_%s_ptr_zero_value(const %[1]s_%[2]s *%[1]s_%[2]s_ptr);\n",
				package_name, decl_kind.type_name)
			struct_definition += fmt.Sprintf("extern void destroy_%s_%s_ptr_tree(%[1]s_%[2]s *%[1]s_%[2]s_ptr, json_t *json, bool free_pointers);\n",
				package_name, decl_kind.type_name)
			struct_definition += fmt.Sprintf("#define        free_%s_%s_ptr_tree(obj_ptr, json_ptr) destroy_%[1]s_%[2]s_ptr_tree(obj_ptr, json_ptr, true)\n",
				package_name, decl_kind.type_name)
			struct_definition += fmt.Sprintf("extern json_t *     %s_%s_ptr_as_JSON_ptr(const %[1]s_%[2]s *%[1]s_%[2]s_ptr);\n",
				package_name, decl_kind.type_name)

			struct_definition += fmt.Sprintf("#define             %s_%s_ptr_as_JSON_str(%[1]s_%[2]s_ptr) JSON_as_str(%[1]s_%[2]s_ptr_as_JSON_ptr(%[1]s_%[2]s_ptr), 0)\n",
				package_name, decl_kind.type_name)

			struct_definition += fmt.Sprintf("extern %s_%s *    JSON_as_%[1]s_%[2]s_ptr(json_t *json);\n",
				package_name, decl_kind.type_name)
			struct_definition += fmt.Sprintf("extern %s_%s *JSON_str_as_%[1]s_%[2]s_ptr(const char *json_str, json_t **json);\n",
				package_name, decl_kind.type_name)
			fmt.Fprintln(header_file, struct_definition)
		default:
			panic(fmt.Sprintf("found unknown type declaration kind '%s'", decl_kind.type_kind))
		}
	}

	if err = footer_boilerplate.Execute(header_file, boilerplate_variables); err != nil {
		panic("C header-file footer processing failed")
	}
	return pointer_base_types, pointer_list_base_types, simple_list_base_types, list_base_types, key_value_pair_types,
		struct_fields, struct_field_Go_types, struct_field_C_types, struct_field_tags, generated_C_code, nil
}

type json_field_tag struct {
	json_field_name string // `json:"name_in_json"`
	json_omitalways bool   // `json:"-"`
	json_omitempty  bool   // `json:",omitempty"`
	json_string     bool   // `json:",string"`
}

// FIX LATER:
// (*) The use of this function might not properly handle anonymous struct fields.
//     That remains to be investigated and tested.
// (*) The amendments to the Go visibility rules that apply during JSON marshalling
//     and unmarshalling as documented in https://golang.org/pkg/encoding/json/#Marshal
//     are badly written and thereby incomprehensible.  Until that situation clears up,
//     we are ignoring that level of finesse.
//
func strict_json_field_tag(field_name string, struct_field_tag string) json_field_tag {
	field_tag := json_field_tag{
		json_field_name: "",
		json_omitalways: false,
		json_omitempty:  false,
		json_string:     false,
	}

	// Pay attention to the lettercase of the first letter of the field_name, to check
	// whether this is not an exported field.  This allows us to match Go's rules for
	// determining which fields are marshalled in JSON.
	//
	// A "->" pointer dereference yields an unnamed object, which we might see at this
	// level as an empty string for the field_name.  Because it's a pointer dereference,
	// we don't try to pretend this is an unexported field.  Logically, though, I suppose
	// we perhaps ought to check the lettercase if the first letter of the pointed-to
	// object, and we're not doing that in the present code.  That remains a possible
	// future modification if we run into unexpected behavior with respect to marshalling.
	//
	if field_name != "" && !unicode.IsUpper([]rune(field_name)[0]) {
		field_tag.json_omitalways = true
	} else {
		tag := reflect.StructTag(struct_field_tag)
		if json_tag := tag.Get("json"); json_tag != "" {
			if json_tag == "-" {
				field_tag.json_omitalways = true
			} else {
				json_options := strings.Split(json_tag, ",")
				// FIX LATER:
				// According to https://golang.org/pkg/encoding/json/#Marshal we have:
				//     The key name will be used if it's a non-empty string
				//     consisting of only Unicode letters, digits, and ASCII
				//     punctuation except quotation marks, backslash, and comma.
				// So in fact we should be testing the JSON name for more than just
				// being non-empty.
				if json_options[0] != "" {
					field_tag.json_field_name = json_options[0]
				}
				for _, option := range json_options[1:] {
					if option == "omitempty" {
						field_tag.json_omitempty = true
					} else if option == "required" {
						// "required" is not a supported json-tag option in Go's JSON formatting,
						// but we could treat it as such here if we wanted
						// field_tag.json_omitempty = false
					} else if option == "string" {
						field_tag.json_string = true
					}
				}
			}
		}
	}

	return field_tag
}

func interpret_json_field_tag(field_name string, struct_field_tag string) json_field_tag {
	field_tag := strict_json_field_tag(field_name, struct_field_tag)
	if field_tag.json_field_name == "" {
		field_tag.json_field_name = field_name
	}
	return field_tag
}

// FIX MAJOR:  When generating these routines, apply the "json"-related content of struct field tags.

func generate_all_encode_tree_routines(
	package_name string,
	final_type_order []declaration_kind,

	// map[base_type_ptr]base_type
	pointer_base_types map[string]string,

	// []base_type
	list_base_types []string,

	// map[key_type][]value_type
	key_value_pair_types map[string][]string,

	// map[enum_name]enum_type
	enum_typedefs map[string]string,

	// map[struct_name][]field_name
	struct_fields map[string][]string,

	// map[struct_name]map[field_name] = field_Go_type
	struct_field_Go_types map[string]map[string]string,

	// map[struct_name]map[field_name] = field_type
	struct_field_C_types map[string]map[string]string,

	// map[struct_name]map[field_name] = field_tag
	struct_field_tags map[string]map[string]string,
) (
	all_encode_function_code string,
	err error,
) {
	all_encode_function_code = ""
	pointer_type_zero_value_code := ""

	// This is a shortcut, so we don't have to nearly-replicate some other code we will generate later on.
	for base_type_ptr, base_type := range pointer_base_types {
		pointer_type_zero_value_code += fmt.Sprintf(`
bool is_%[1]s_ptr_zero_value(const %[1]s *%[1]s_ptr) {
    return
	is_%[2]s_ptr_zero_value(*%[1]s_ptr)
    ;
}
`, base_type_ptr, base_type,
		)
		// FIX QUICK:  This ought to be moved to the header file.
		all_encode_function_code += fmt.Sprintf("#define %s_ptr_as_JSON_ptr(ptr) %s_ptr_as_JSON_ptr(*(ptr))\n", base_type_ptr, base_type)
	}
	// FIX QUICK:  There ought to be declarations for these routines in the header file, if there aren't already.
	all_encode_function_code += pointer_type_zero_value_code

	// FIX QUICK:  Perhaps this support for zero_value routines for _List types should be moved into the generate_encode_list_tree() routine.
	for _, base_type := range list_base_types {
		all_encode_function_code += fmt.Sprintf(`
bool is_%[1]s_List_ptr_zero_value(const %[1]s_List *%[1]s_List_ptr) {
    for (int index = %[1]s_List_ptr->count; --index >= 0; ) {
	if (!is_%[1]s_ptr_zero_value(&%[1]s_List_ptr->items[index])) {
	    return false;
	}
    }
    return true;
}
`, base_type,
		)
		function_code, err := generate_encode_list_tree(base_type)
		if err != nil {
			panic(err)
		}
		all_encode_function_code += function_code
	}

	for key_type, value_types := range key_value_pair_types {
		for _, value_type := range value_types {
			function_code, err := generate_encode_key_value_pair_tree(key_type, value_type)
			if err != nil {
				panic(err)
			}
			all_encode_function_code += function_code
		}
	}

	for _, final_type := range final_type_order {
		if final_type.type_kind == "struct" {
			function_code, err := generate_encode_PackageName_StructTypeName_ptr_tree(
				package_name, final_type.type_name,
				pointer_base_types, key_value_pair_types, enum_typedefs, struct_fields, struct_field_Go_types, struct_field_C_types, struct_field_tags,
			)
			if err != nil {
				panic(err)
			}
			all_encode_function_code += function_code
		}
	}
	return all_encode_function_code, err
}

func generate_decode_pointer_list_tree(
	base_type string,
) (
	function_code string,
	err error,
) {
	function_code += fmt.Sprintf(`
%[1]s_Ptr_List *JSON_as_%[1]s_Ptr_List_ptr(json_t *json) {
    %[1]s_Ptr_List *%[1]s_Ptr_List_ptr = (%[1]s_Ptr_List *) calloc(1, sizeof(%[1]s_Ptr_List));
    if (%[1]s_Ptr_List_ptr == NULL) {
	fprintf(stderr, FILE_LINE "ERROR:  in JSON_as_%[1]s_Ptr_List_ptr, %%s\n", "malloc failed");
    } else {
	int failed = 0;
	%[1]s_Ptr_List_ptr->count = json_array_size(json);
	%[1]s_Ptr_List_ptr->items = (%[1]s_Ptr *) malloc(%[1]s_Ptr_List_ptr->count * sizeof(%[1]s_Ptr));
	size_t index;
	json_t *value;
	json_array_foreach(json, index, value) {
	    %[1]s *%[1]s_ptr = JSON_as_%[1]s_ptr(value);
	    if (%[1]s_ptr == NULL) {
		fprintf(stderr, FILE_LINE "ERROR:  in JSON_as_%[1]s_Ptr_List_ptr, %%s\n", "transit_TypedValue_ptr is NULL");
		failed = 1;
		break;
	    } else {
		%[1]s_Ptr_List_ptr->items[index] = %[1]s_ptr;
	    }
	}
	if (failed) {
	    // FIX QUICK:  verify that this error handling is correct at all levels,
	    // including possible removal of any objects already copied into the array
	    // (which might not be the full array size)
	    free(%[1]s_Ptr_List_ptr);
	    %[1]s_Ptr_List_ptr = NULL;
	}
    }
    return %[1]s_Ptr_List_ptr;
}
`, base_type)

	return function_code, err
}

func generate_decode_simple_list_tree(
	base_type string,
) (
	function_code string,
	err error,
) {
	function_code += fmt.Sprintf(`
%[1]s_List *JSON_as_%[1]s_List_ptr(json_t *json) {
    %[1]s_List *%[1]s_List_ptr = (%[1]s_List *) calloc(1, sizeof(%[1]s_List));
    if (%[1]s_List_ptr == NULL) {
	fprintf(stderr, FILE_LINE "ERROR:  in JSON_as_%[1]s_List_ptr, %%s\n", "malloc failed");
    } else {
	int failed = 0;
	%[1]s_List_ptr->count = json_array_size(json);
	%[1]s_List_ptr->items = (%[1]s *) malloc(%[1]s_List_ptr->count * sizeof(%[1]s));
	size_t index;
	json_t *value;
	json_array_foreach(json, index, value) {
	    %[1]s *%[1]s_ptr = JSON_as_%[1]s_ptr(value);
	    if (%[1]s_ptr == NULL) {
		fprintf(stderr, FILE_LINE "ERROR:  in JSON_as_%[1]s_List_ptr, %%s\n", "transit_TypedValue_ptr is NULL");
		failed = 1;
		break;
	    } else {
		%[1]s_List_ptr->items[index] = *%[1]s_ptr;
		free(%[1]s_ptr);
	    }
	}
	if (failed) {
	    // FIX QUICK:  verify that this error handling is correct at all levels,
	    // including possible removal of any objects already copied into the array
	    // (which might not be the full array size)
	    free(%[1]s_List_ptr);
	    %[1]s_List_ptr = NULL;
	}
    }
    return %[1]s_List_ptr;
}
`, base_type)

	return function_code, err
}

func generate_all_decode_tree_routines(
	package_name string,
	final_type_order []declaration_kind,

	// map[base_type_ptr]base_type
	pointer_base_types map[string]string,

	// []list_base_type
	pointer_list_base_types []string,

	// []base_type
	simple_list_base_types []string,

	// map[key_type][]value_type
	key_value_pair_types map[string][]string,

	// map[enum_name]enum_type
	enum_typedefs map[string]string,

	// map[struct_name][]field_name
	struct_fields map[string][]string,

	// map[struct_name]map[field_name] = field_Go_type
	struct_field_Go_types map[string]map[string]string,

	// map[struct_name]map[field_name] = field_type
	struct_field_C_types map[string]map[string]string,

	// map[struct_name]map[field_name] = field_tag
	struct_field_tags map[string]map[string]string,
) (
	all_decode_function_code string,
	err error,
) {
	all_decode_function_code = ""

	// Prove that we really do have the struct_field_tags data structure populated as we expect it to be, in full detail.
	if print_diagnostics {
		for struct_name, field_tags := range struct_field_tags {
			for field_name, field_tag := range field_tags {
				fmt.Fprintf(diag_file, "struct_field_tags[%s][%s] = %s\n", struct_name, field_name, field_tag)
			}
		}
	}

	for _, base_type := range pointer_list_base_types {
		function_code, err := generate_decode_pointer_list_tree(base_type)
		if err != nil {
			panic(err)
		}
		all_decode_function_code += function_code
	}

	for _, base_type := range simple_list_base_types {
		function_code, err := generate_decode_simple_list_tree(base_type)
		if err != nil {
			panic(err)
		}
		all_decode_function_code += function_code
	}

	for key_type, value_types := range key_value_pair_types {
		for _, value_type := range value_types {
			function_code, err := generate_decode_key_value_pair_tree(key_type, value_type)
			if err != nil {
				panic(err)
			}
			all_decode_function_code += function_code
		}
	}

	for _, final_type := range final_type_order {
		if final_type.type_kind == "struct" {
			function_code, err := generate_decode_PackageName_StructTypeName_ptr_tree(
				package_name, final_type.type_name,
				pointer_base_types, key_value_pair_types, enum_typedefs, struct_fields, struct_field_Go_types, struct_field_C_types, struct_field_tags,
			)
			if err != nil {
				panic(err)
			}
			all_decode_function_code += function_code
		}
	}
	return all_decode_function_code, err
}

func generate_all_destroy_tree_routines(
	package_name string,
	final_type_order []declaration_kind,

	// map[key_type][]value_type
	key_value_pair_types map[string][]string,

	// map[typedef_name]typedef_type
	simple_typedefs map[string]string,

	// map[enum_name]enum_type
	enum_typedefs map[string]string,

	// map[struct_name][]field_name
	struct_fields map[string][]string,

	// map[struct_name]map[field_name] = field_type
	struct_field_C_types map[string]map[string]string,
) (
	all_destroy_function_code string,
	err error,
) {
	all_destroy_function_code = ""
	for _, final_type := range final_type_order {
		if final_type.type_kind == "struct" {
			function_code, err := generate_destroy_PackageName_StructTypeName_ptr_tree(
				package_name, final_type.type_name, key_value_pair_types, simple_typedefs, enum_typedefs, struct_fields, struct_field_C_types,
			)
			if err != nil {
				panic(err)
			}
			all_destroy_function_code += function_code
		} else if final_type.type_kind == "simple" {
			// This code is experimental.  It will get us a destroy routine built for the typedef name itself. as a
			// top-level object Whether or not we need that is as yet uncertain, but we leave this in just in case.
			function_code, err := generate_destroy_PackageName_StructTypeName_ptr_tree(
				package_name, final_type.type_name, key_value_pair_types, simple_typedefs, enum_typedefs, struct_fields, struct_field_C_types,
			)
			if err != nil {
				panic(err)
			}
			all_destroy_function_code += function_code
		} else {
			if print_diagnostics {
				fmt.Printf("skipping destroy of %s %s objects\n", final_type.type_kind, final_type.type_name)
			}
		}
	}
	return all_destroy_function_code, err
}

func generate_encode_list_tree(base_type string) (function_code string, err error) {

	// Here is a sample template for code to be generated for a _Pair_List.
	// This template is to be instantiated once for each _Pair type.  There
	// will be no separate conversion routine for the _Pair type itself.
	// Because we will generate declarations in the associated header file
	// for all conversion routines, there will be no concern about the order
	// in which we instantiate this template relative to the other routines
	// that will call it.
	/*
	   json_t *transit_MonitoredResource_List_ptr_as_JSON_ptr(const transit_MonitoredResource_List *transit_MonitoredResource_List) {
	       json_t *json;
	       if (transit_MonitoredResource_List == NULL) {
	   	json = NULL;
	       } else if (transit_MonitoredResource_List->count == 0) {
	   	json = NULL;
	       } else {
	   	json = json_array();
	   	if (json == NULL) {
	   	    printf(FILE_LINE "ERROR:  cannot create a JSON %s object\n", "transit_MonitoredResource_List");
	   	} else {
	   	    for (size_t i = 0; i < transit_MonitoredResource_List->count; ++i) {
	   		if (json_array_append_new(json,
	   		    transit_MonitoredResource_ptr_as_JSON_ptr( &transit_MonitoredResource_List->items[i] ) // transit_MonitoredResource*
	   		) != 0) {
	   		    //
	   		    // Report and handle the error condition.  Unfortunately, there is no json_error_t value to
	   		    // look at, to determine the exact cause.  Also, be aware that we might now have a memory leak.
	   		    // Since we don't know exactly what happened, we would rather suffer that leak than attempt to
	   		    // decrement the reference count on the subsidiary object that we just tried to add to the array
	   		    // (if in fact it was non-NULL).
	   		    //
	   		    // Since adding one element to the array didn't work, we abort the process of trying to add any
	   		    // additional elements to the array.  Instead, we just clear out the entire array, and we return
	   		    // a NULL value to indicate the error.
	   		    //
	   		    // A future version might print at least the failing key, if not also the failing value (which
	   		    // could be of some complicated type).
	   		    //
	   		    printf(FILE_LINE "ERROR:  cannot append an element to a JSON %s array\n", "transit_MonitoredResource_List");
	   		    json_array_clear(json);
	   		    json_decref(json);
	   		    json = NULL;
	   		    break;
	   		}
	   	    }
	   	}
	       }
	       return json;
	   }
	*/

	// Example substitutions:
	// {{.BaseType}}  // transit_MonitoredResource
	// {{.ListType}}  // transit_MonitoredResource_List
	var encode_routine_complete_template = `
json_t *{{.ListType}}_ptr_as_JSON_ptr(const {{.ListType}} *{{.ListType}}_ptr) {
    json_t *json;
    if ({{.ListType}}_ptr == NULL) {
	json = NULL;
    } else if ({{.ListType}}_ptr->count == 0) {
	json = NULL;
    } else {
	json = json_array();
	if (json == NULL) {
	    printf(FILE_LINE "ERROR:  cannot create a JSON %s object\n", "{{.ListType}}");
	} else {
	    for (size_t i = 0; i < {{.ListType}}_ptr->count; ++i) {
		if (json_array_append_new(json,
		    {{.BaseType}}_ptr_as_JSON_ptr( &{{.ListType}}_ptr->items[i] ) // {{.BaseType}}*
		) != 0) {
		    //
		    // Report and handle the error condition.  Unfortunately, there is no json_error_t value to
		    // look at, to determine the exact cause.  Also, be aware that we might now have a memory leak.
		    // Since we don't know exactly what happened, we would rather suffer that leak than attempt to
		    // decrement the reference count on the subsidiary object that we just tried to add to the array
		    // (if in fact it was non-NULL).
		    //
		    // Since adding one element to the array didn't work, we abort the process of trying to add any
		    // additional elements to the array.  Instead, we just clear out the entire array, and we return
		    // a NULL value to indicate the error.
		    //
		    // A future version might print at least the failing key, if not also the failing value (which
		    // could be of some complicated type).
		    //
		    printf(FILE_LINE "ERROR:  cannot append an element to a JSON %s array\n", "{{.ListType}}");
		    json_array_clear(json);
		    json_decref(json);
		    json = NULL;
		    break;
		}
	    }
	}
    }
    return json;
}
`

	complete_template := template.Must(template.New("encode_routine_complete").Parse(encode_routine_complete_template))

	type encode_routine_complete_fields struct {
		BaseType string
		ListType string
	}

	complete_variables := encode_routine_complete_fields{
		BaseType: base_type,
		ListType: base_type + "_List",
	}

	var complete_code bytes.Buffer
	if err := complete_template.Execute(&complete_code, complete_variables); err != nil {
		panic("encode routine complete processing failed")
	}
	function_code += complete_code.String()

	return function_code, err
}

func generate_encode_key_value_pair_tree(key_type string, value_type string) (function_code string, err error) {

	// Here is a sample template for code to be generated for a _Pair_List.
	// This template is to be instantiated once for each _Pair type.  There
	// will be no separate conversion routine for the _Pair type itself.
	// Because we will generate declarations in the associated header file
	// for all conversion routines, there will be no concern about the order
	// in which we instantiate this template relative to the other routines
	// that will call it.
	/*
	   json_t *string_transit_TypedValue_Pair_List_ptr_as_JSON_ptr(const string_transit_TypedValue_Pair_List *string_transit_TypedValue_Pair_List_ptr) {
	       json_t *json;
	       if (string_transit_TypedValue_Pair_List_ptr == NULL) {
	   	json = NULL;
	       } else if (string_transit_TypedValue_Pair_List_ptr->count == 0) {
	   	json = NULL;
	       } else {
	   	json = json_object();
	   	if (json == NULL) {
	   	    printf(FILE_LINE "ERROR:  cannot create a JSON %s object\n", "string_transit_TypedValue_Pair_List");
	   	} else {
	   	    for (size_t i = 0; i < string_transit_TypedValue_Pair_List_ptr->count; ++i) {
	   		if (json_object_set_new(json
	   		    , string_transit_TypedValue_Pair_List_ptr->items[i].key                                  // string
	   		    , transit_TypedValue_ptr_as_JSON_ptr( &string_transit_TypedValue_Pair_List_ptr->items[i].value ) // transit_TypedValue
	   		) != 0) {
	   		    //
	   		    // Report and handle the error condition.  Unfortunately, there is no json_error_t value to
	   		    // look at, to determine the exact cause.  Also, be aware that we might now have a memory leak.
	   		    // Since we don't know exactly what happened, we would rather suffer that leak than attempt to
	   		    // decrement the reference count on the subsidiary object that we just tried to add to the array
	   		    // (if in fact it was non-NULL).
	   		    //
	   		    // Since adding one key/value pair to the object didn't work, we abort the process of trying to
	   		    // add any additional key/value pairs to the object.  Instead, we just clear out the entire object,
	   		    // and we return a NULL value to indicate the error.
	   		    //
	   		    // A future version might print at least the failing key, if not also the failing value (which
	   		    // could be of some complicated type).
	   		    //
	   		    printf(FILE_LINE "ERROR:  cannot set a key/value pair in a JSON %s object\n", "string_transit_TypedValue_Pair_List");
	   		    json_object_clear(json);
	   		    json_decref(json);
	   		    json = NULL;
	   		    break;
	   		}
	   	    }
	   	}
	       }
	       return json;
	   }
	*/

	// Example substitutions:
	// {{.PairKeyType}}    // string
	// {{.PairValueType}}  // transit_TypedValue
	// {{.PairListType}}   // string_transit_TypedValue_Pair_List
	var encode_routine_complete_template = `
json_t *{{.PairListType}}_ptr_as_JSON_ptr(const {{.PairListType}} *{{.PairListType}}_ptr) {
    json_t *json;
    if ({{.PairListType}}_ptr == NULL) {
	json = NULL;
    } else if ({{.PairListType}}_ptr->count == 0) {
	json = NULL;
    } else {
	json = json_object();
	if (json == NULL) {
	    printf(FILE_LINE "ERROR:  cannot create a JSON %s object\n", "{{.PairListType}}");
	} else {
	    for (size_t i = 0; i < {{.PairListType}}_ptr->count; ++i) {
		if (json_object_set_new(json
		    , {{.PairListType}}_ptr->items[i].key // {{.PairKeyType}}
		    , {{.PairValueType}}_ptr_as_JSON_ptr( &{{.PairListType}}_ptr->items[i].value ) // {{.PairValueType}}
		) != 0) {
		    //
		    // Report and handle the error condition.  Unfortunately, there is no json_error_t value to
		    // look at, to determine the exact cause.  Also, be aware that we might now have a memory leak.
		    // Since we don't know exactly what happened, we would rather suffer that leak than attempt to
		    // decrement the reference count on the subsidiary object that we just tried to add to the array
		    // (if in fact it was non-NULL).
		    //
		    // Since adding one key/value pair to the object didn't work, we abort the process of trying to
		    // add any additional key/value pairs to the object.  Instead, we just clear out the entire object,
		    // and we return a NULL value to indicate the error.
		    //
		    // A future version might print at least the failing key, if not also the failing value (which
		    // could be of some complicated type).
		    //
		    printf(FILE_LINE "ERROR:  cannot set a key/value pair in a JSON %s object\n", "{{.PairListType}}");
		    json_object_clear(json);
		    json_decref(json);
		    json = NULL;
		    break;
		}
	    }
	}
    }
    return json;
}
`

	complete_template := template.Must(template.New("encode_routine_complete").Parse(encode_routine_complete_template))

	type encode_routine_complete_fields struct {
		PairKeyType   string
		PairValueType string
		PairListType  string
	}

	complete_variables := encode_routine_complete_fields{
		PairKeyType:   key_type,
		PairValueType: value_type,
		PairListType:  key_type + "_" + value_type + "_Pair_List",
	}

	var complete_code bytes.Buffer
	if err := complete_template.Execute(&complete_code, complete_variables); err != nil {
		panic("encode routine complete processing failed")
	}
	function_code += complete_code.String()

	return function_code, err
}

func generate_decode_key_value_pair_tree(key_type string, value_type string) (function_code string, err error) {

	// Here is a sample template for code to be generated for a _Pair_List.
	// This template is to be instantiated once for each _Pair type.  There
	// will be no separate conversion routine for the _Pair type itself.
	// Because we will generate declarations in the associated header file
	// for all conversion routines, there will be no concern about the order
	// in which we instantiate this template relative to the other routines
	// that will call it.
	/*
	   string_transit_TypedValue_Pair_List *JSON_as_string_transit_TypedValue_Pair_List_ptr(json_t *json) {
	       string_transit_TypedValue_Pair_List *Pair_List = (string_transit_TypedValue_Pair_List *) malloc(sizeof(string_transit_TypedValue_Pair_List));
	       if (Pair_List == NULL) {
	   	// FIX MAJOR:  Invoke proper logging for error conditions.
	   	fprintf(stderr, FILE_LINE "ERROR:  in JSON_as_string_transit_TypedValue_Pair_List_ptr, %s\n", "malloc failed");
	       } else {
	   	int failed = 0;
	   	Pair_List->count = json_object_size(json);
	   	Pair_List->items = (string_transit_TypedValue_Pair *) malloc(Pair_List->count * sizeof(string_transit_TypedValue_Pair));
	   	const char *key;
	   	json_t *value;
	   	size_t i = 0;
	   	json_object_foreach(json, key, value) {
	   	    // Here we throw away constness as far as the compiler is concerned, but by convention
	   	    // the calling code will never alter the key, so that won't matter.
	   	    Pair_List->items[i].key = (char *) key;
	   	    transit_TypedValue *TypedValue_ptr = JSON_as_transit_TypedValue_ptr(value);
	   	    if (TypedValue_ptr == NULL) {
	   		fprintf(stderr, FILE_LINE "ERROR:  in JSON_as_string_transit_TypedValue_Pair_List_ptr, %s\n", "TypedValue_ptr is NULL");
	   		failed = 1;
	   		break;
	   	    } else {
	   		Pair_List->items[i].value = *TypedValue_ptr;
	   		free(TypedValue_ptr);
	   	    }
	   	    ++i;
	   	}
	   	if (failed) {
	   	    // FIX QUICK:  verify that this error handling is correct at all levels,
	   	    // including possible removal of any objects already copied into the array
	   	    // (which might not be the full array size)
	   	    free(Pair_List);
	   	    Pair_List = NULL;
	   	}
	       }
	       return Pair_List;
	   }
	*/

	// Example substitutions:
	// {{.PairKeyType}}    // string
	// {{.PairValueType}}  // transit_TypedValue
	// {{.PairListType}}   // string_transit_TypedValue_Pair_List
	var decode_routine_complete_template = `
{{.PairListType}} *JSON_as_{{.PairListType}}_ptr(json_t *json) {
    {{.PairListType}} *Pair_List = ({{.PairListType}} *) malloc(sizeof({{.PairListType}}));
    if (Pair_List == NULL) {
	// FIX MAJOR:  Invoke proper logging for error conditions.
	fprintf(stderr, FILE_LINE "ERROR:  in JSON_as_{{.PairListType}}_ptr, %s\n", "malloc failed");
    } else {
	int failed = 0;
	Pair_List->count = json_object_size(json);
	Pair_List->items = ({{.PairKeyType}}_{{.PairValueType}}_Pair *) malloc(Pair_List->count * sizeof({{.PairKeyType}}_{{.PairValueType}}_Pair));
	const char *key;
	json_t *value;
	size_t i = 0;
	json_object_foreach(json, key, value) {
	    // Here we throw away constness as far as the compiler is concerned, but by convention
	    // the calling code will never alter the key, so that won't matter.
	    Pair_List->items[i].key = (char *) key;
	    {{.PairValueType}} *{{.PairValueType}}_ptr = JSON_as_{{.PairValueType}}_ptr(value);
	    if ({{.PairValueType}}_ptr == NULL) {
		fprintf(stderr, FILE_LINE "ERROR:  in JSON_as_{{.PairListType}}_ptr, %s\n", "{{.PairValueType}}_ptr is NULL");
		failed = 1;
		break;
	    } else {
		Pair_List->items[i].value = *{{.PairValueType}}_ptr;
		free({{.PairValueType}}_ptr);
	    }
	    ++i;
	}
	if (failed) {
	    // FIX QUICK:  verify that this error handling is correct at all levels,
	    // including possible removal of any objects already copied into the array
	    // (which might not be the full array size)
	    free(Pair_List);
	    Pair_List = NULL;
	}
    }
    return Pair_List;
}
`

	complete_template := template.Must(template.New("decode_routine_complete").Parse(decode_routine_complete_template))

	type decode_routine_complete_fields struct {
		PairKeyType   string
		PairValueType string
		PairListType  string
	}

	complete_variables := decode_routine_complete_fields{
		PairKeyType:   key_type,
		PairValueType: value_type,
		PairListType:  key_type + "_" + value_type + "_Pair_List",
	}

	var complete_code bytes.Buffer
	if err := complete_template.Execute(&complete_code, complete_variables); err != nil {
		panic("decode routine complete processing failed")
	}
	function_code += complete_code.String()

	/*
	   bool is_string_transit_TypedValue_Pair_ptr_zero_value(const string_transit_TypedValue_Pair *string_transit_TypedValue_Pair_ptr) {
	       return
	   	is_string_ptr_zero_value(&string_transit_TypedValue_Pair_ptr->key) &&
	   	is_transit_TypedValue_ptr_zero_value(&string_transit_TypedValue_Pair_ptr->value)
	       ;
	   }

	   bool is_string_transit_TypedValue_Pair_List_ptr_zero_value(const string_transit_TypedValue_Pair_List *string_transit_TypedValue_Pair_List_ptr) {
	       for (int index = string_transit_TypedValue_Pair_List_ptr->count; --index >= 0; ) {
	   	if (!is_string_transit_TypedValue_Pair_ptr_zero_value(&string_transit_TypedValue_Pair_List_ptr->items[index])) {
	   	    return false;
	   	}
	       }
	       return true;
	   }
	*/

	function_code += fmt.Sprintf(`
bool is_%[1]s_%s_Pair_ptr_zero_value(const %[1]s_%s_Pair *%[1]s_%s_Pair_ptr) {
    return
	is_%[1]s_ptr_zero_value(&%[1]s_%s_Pair_ptr->key) &&
	is_%[2]s_ptr_zero_value(&%[1]s_%s_Pair_ptr->value)
    ;
}

bool is_%[1]s_%s_Pair_List_ptr_zero_value(const %[1]s_%s_Pair_List *%[1]s_%s_Pair_List_ptr) {
    for (int index = %[1]s_%s_Pair_List_ptr->count; --index >= 0; ) {
	if (!is_%[1]s_%s_Pair_ptr_zero_value(&%[1]s_%s_Pair_List_ptr->items[index])) {
	    return false;
	}
    }
    return true;
}
`, key_type, value_type,
	)

	return function_code, err
}

func generate_encode_PackageName_StructTypeName_ptr_tree(
	package_name string,
	struct_name string,

	// map[base_type_ptr]base_type
	pointer_base_types map[string]string,

	// map[key_type][]value_type
	key_value_pair_types map[string][]string,

	// map[enum_name]enum_type
	enum_typedefs map[string]string,

	// map[struct_name][]field_name
	struct_fields map[string][]string,

	// map[struct_name]map[field_name] = field_Go_type
	struct_field_Go_types map[string]map[string]string,

	// map[struct_name]map[field_name] = field_type
	struct_field_C_types map[string]map[string]string,

	// map[struct_name]map[field_name] = field_tag
	struct_field_tags map[string]map[string]string,
) (
	function_code string,
	err error,
) {
	// Here's the template for the standard encoding function we need to generate.
	// There are also a few extra flavors, which we will get to in due course.

	var encode_routine_header_template = `
json_t *{{.StructName}}_ptr_as_JSON_ptr(const {{.StructName}} *{{.StructName}}_ptr) {
    char *failure = NULL;
    json_t *json;
    do  {
	if ({{.StructName}}_ptr == NULL) {
	    json = NULL;
	    failure = "received a NULL object to convert";
	    break;
	}`

	var encode_routine_normal_body_format = `
	json = json_object();
	if (json == NULL) {
	    failure = "failed to create an empty JSON object";
	    break;
	}`

	/*
	    var encode_routine_struct_timespec_body_format = `
		json_error_t error;
		size_t flags = 0;
		// We special-case the field packing in this routine, based on the "struct_timespec" field type.
		// The "I" conversion is used to handle a 64-bit number.
		json = json_pack_ex(&error, flags, "I"
		     // struct_timespec Time_;  // go: time.Time
		     , (json_int_t) (
			 (milliseconds_MillisecondTimestamp_ptr->Time_.tv_sec  * MILLISECONDS_PER_SECOND) +
			 (milliseconds_MillisecondTimestamp_ptr->Time_.tv_nsec / NANOSECONDS_PER_MILLISECOND)
		     )
		);
		if (json == NULL) {
		    // printf(FILE_LINE "ERROR:  text '%s', source '%s', line %d, column %d, position %d\n", error.text, error.source, error.line, error.column, error.position);
		    failure = error.text;
		    break;
		}`
	*/

	var encode_routine_struct_timespec_body_format = `
	// FIX QUICK:  Deal with the error scope issue (a text pointer from the error object may be
	// assigned to the failure pointer and then used after the error object has gone out of scope).
	json_error_t error;
	size_t flags = 0;
	// We special-case the field packing in this routine, based on the "struct_timespec" field type.
	// The "I" conversion is used to handle a 64-bit number.
	json = json_pack_ex(&error, flags, "I"
	     // %[4]s %[3]s;  // go: time.Time
	     , (json_int_t) (
		 (%[1]s_%s_ptr->%s.tv_sec  * MILLISECONDS_PER_SECOND) +
		 (%[1]s_%s_ptr->%s.tv_nsec / NANOSECONDS_PER_MILLISECOND)
	     )
	);
	if (json == NULL) {
	    // printf(FILE_LINE "ERROR:  text '%s', source '%s', line %d, column %d, position %d\n",
		// error.text, error.source, error.line, error.column, error.position);
	    failure = error.text;
	    break;
	}`

	var encode_routine_footer_template = `
    } while (0);
    if (failure) {
	// FIX MAJOR:  Re-jigger the function call API to either pass back the failure string up the call chain,
	// or call a logger with which the package has been previously initialilzed.
	printf(FILE_LINE "ERROR:  failed to create JSON for a %s structure:  %s\n", "{{.StructName}}", failure);
	if (json) {
	    json_decref(json);
	    json = NULL;
	}
    }
    return json;
}
`

	header_template := template.Must(template.New("encode_routine_header").Parse(encode_routine_header_template))
	footer_template := template.Must(template.New("encode_routine_footer").Parse(encode_routine_footer_template))

	type encode_routine_boilerplate_fields struct {
		StructName string
	}

	// Packing a JSON string is complicated by several factors:
	// (*) The desire to keep the packing order in a controlled sequence, mostly for easy
	//     comparison of resulting strings in unit testing.
	// (*) The fact that some fields might have the Go-language "omitempty" optino attached.
	// (*) The fact that the Jansson library we're using currently has no pack-syntax item
	//     modifiers that follow the Go-language rules, especially for handling the zero
	//     values of scalar types in the same manner that Go does, so we need to handle that
	//     manually in our own code.  (One might imagine the availability of a "~" suffix
	//     modifier, with that character chosen rather arbitrarily, to support the Go rules.)
	// The net result is that we cannot just make a single call to json_pack_ex() to cover
	// all fields in one step.  Instead, we must create an empty object, then handle each of
	// the fields sequentially.

	fields := struct_fields[struct_name] // []string
	max_field_name_len := 0
	for _, field_name := range fields {
		field_tag := interpret_json_field_tag(field_name, struct_field_tags[struct_name][field_name])
		if !field_tag.json_omitalways {
			field_name_len := len(field_name)
			if max_field_name_len < field_name_len {
				max_field_name_len = field_name_len
			}
		}
	}

	boilerplate_variables := encode_routine_boilerplate_fields{StructName: package_name + "_" + struct_name}

	var header_code bytes.Buffer
	if err := header_template.Execute(&header_code, boilerplate_variables); err != nil {
		panic("encode routine header processing failed")
	}
	function_code += header_code.String()

	type encode_routine_struct_fields struct {
		StructName string
		FieldName  string
		FieldType  string
	}

	is_zero_function_code := fmt.Sprintf(`
bool is_%[1]s_%s_ptr_zero_value(const %[1]s_%s *%[1]s_%s_ptr) {
    return
`,
		package_name, struct_name,
	)
	is_zero_field_separator := "        "

	last_go_type_component := regexp.MustCompile(`\.([^.]+)$`)
	have_encode_routine_normal_body_format := false
	for _, field_name := range fields {
		field_C_type := struct_field_C_types[struct_name][field_name]

		// FIX QUICK:  clean this up
		// field_tag := interpret_json_field_tag(field_name, struct_field_tags[struct_name][field_name])

		go_type := struct_field_Go_types[struct_name][field_name]
		field_tag := strict_json_field_tag(field_name, struct_field_tags[struct_name][field_name])
		var json_field_name string
		if field_tag.json_field_name != "" {
			json_field_name = field_tag.json_field_name
		} else if matches := last_go_type_component.FindStringSubmatch(go_type); matches != nil {
			// This possible adjustment of the field name is needed because we might have a complex
			// Go field with a Go type like "*config.Config" that is represented in the Go code as
			// an anonymous field (i.e., with no stated name).  In that case, Go's JSON package
			// will encode that field using a field name which is only the last component of the Go
			// typename, not the entire typename.  So we must replicate that behavior here.
			json_field_name = matches[1]
		} else {
			json_field_name = field_name
		}

		var include_condition string
		if !field_tag.json_omitalways {
			// FIX MAJOR:  This might need work to correctly handle all of int32, int64, and int;
			// check the JSON_INTEGER_IS_LONG_LONG macro, and handle stuff appropriately.

			switch field_C_type {
			case "bool":
				if field_tag.json_omitempty {
					include_condition = fmt.Sprintf("%s_%s_ptr->%s != false", package_name, struct_name, field_name)
				} else {
					include_condition = "1"
				}
				if !have_encode_routine_normal_body_format {
					function_code += encode_routine_normal_body_format
					have_encode_routine_normal_body_format = true
				}
				function_code += fmt.Sprintf(`
	if (%s) {
	    if (json_object_set_new(json, "%s", json_boolean(%s_%s_ptr->%s)) != 0) {
		failure = "cannot set %[3]s_%s_ptr->%s boolean value into object";
		break;
	    }
	}`,
					// include_condition, field_tag.json_field_name, package_name, struct_name, field_name,
					include_condition, json_field_name, package_name, struct_name, field_name,
				)
			case "int":
				fallthrough
			case "int32":
				fallthrough
			case "int64":
				if field_tag.json_omitempty {
					include_condition = fmt.Sprintf("%s_%s_ptr->%s != 0", package_name, struct_name, field_name)
				} else {
					include_condition = "1"
				}
				if !have_encode_routine_normal_body_format {
					function_code += encode_routine_normal_body_format
					have_encode_routine_normal_body_format = true
				}
				function_code += fmt.Sprintf(`
	if (%s) {
	    if (json_object_set_new(json, "%s", json_integer((json_int_t) %s_%s_ptr->%s)) != 0) {
		failure = "cannot set %[3]s_%s_ptr->%s %s value into object";
		break;
	    }
	}`,
					// include_condition, field_tag.json_field_name, package_name, struct_name, field_name, field_C_type,
					include_condition, json_field_name, package_name, struct_name, field_name, field_C_type,
				)
			case "float64":
				if field_tag.json_omitempty {
					include_condition = fmt.Sprintf("%s_%s_ptr->%s != 0.0", package_name, struct_name, field_name)
				} else {
					include_condition = "1"
				}
				if !have_encode_routine_normal_body_format {
					function_code += encode_routine_normal_body_format
					have_encode_routine_normal_body_format = true
				}
				function_code += fmt.Sprintf(`
	if (%s) {
	    if (json_object_set_new(json, "%s", json_real(%s_%s_ptr->%s)) != 0) {
		failure = "cannot set %[3]s_%s_ptr->%s double value into object";
		break;
	    }
	}`,
					// include_condition, field_tag.json_field_name, package_name, struct_name, field_name,
					include_condition, json_field_name, package_name, struct_name, field_name,
				)
			case "string":
				if field_tag.json_omitempty {
					include_condition = fmt.Sprintf("%s_%s_ptr->%s != NULL && %[1]s_%s_ptr->%s[0] != '\\0'", package_name, struct_name, field_name)
				} else {
					include_condition = "1"
				}
				if !have_encode_routine_normal_body_format {
					function_code += encode_routine_normal_body_format
					have_encode_routine_normal_body_format = true
				}
				function_code += fmt.Sprintf(`
	if (%s) {
	    if (json_object_set_new(json, "%s", json_string(%s_%s_ptr->%s)) != 0) {
		failure = "cannot set %[3]s_%s_ptr->%s string value into object";
		break;
	    }
	}`,
					// include_condition, field_tag.json_field_name, package_name, struct_name, field_name,
					include_condition, json_field_name, package_name, struct_name, field_name,
				)
			default:
				// FIX QUICK:  clean this up
				// if enum_C_type, ok := enum_typedefs[field_C_type]; ok { ... } // if the field type is an enumeration type
				// --------------------------------------------------------------------------------------------------------------------------------
				if _, ok := enum_typedefs[field_C_type]; ok { // if the field type is an enumeration type
					if field_tag.json_omitempty {
						if enum_typedefs[field_C_type] == "string" {
							include_condition = fmt.Sprintf("%s_%s_%s != NULL && %[1]s_%s_%s[0] != '\\0'", package_name, struct_name, field_name)
						} else {
							include_condition = fmt.Sprintf("%s_%s_ptr->%s != 0", package_name, struct_name, field_name)
						}
					} else {
						include_condition = "1"
					}
					if !have_encode_routine_normal_body_format {
						function_code += encode_routine_normal_body_format
						have_encode_routine_normal_body_format = true
					}
					if enum_typedefs[field_C_type] == "string" {
						function_code += fmt.Sprintf(`
	const string %[3]s_%s_%s = %[3]s_%[6]s_String[%[3]s_%s_ptr->%s];
	if (%[1]s) {
	    // %[3]s_%s.%s enumeration value, expressed as a string
	    if (json_object_set_new(json, "%[2]s", json_string(%s_%s_%s)) != 0) {
		failure = "cannot set %[3]s_%s_ptr->%s %s enumeration string value into object";
		break;
	    }
	}`,
							// include_condition, field_tag.json_field_name, package_name, struct_name, field_name, field_C_type,
							include_condition, json_field_name, package_name, struct_name, field_name, field_C_type,
						)
					} else {
						function_code += fmt.Sprintf(`
	if (%[1]s) {
	    // %[3]s_%s.%s enumeration value, expressed as a number
	    if (json_object_set_new(json, "%[2]s", json_integer(%[3]s_%s_ptr->%s)) != 0) {
		failure = "cannot set %[3]s_%s_ptr->%s %s enumeration numeric value into object";
		break;
	    }
	}`,
							// include_condition, field_tag.json_field_name, package_name, struct_name, field_name, field_C_type,
							include_condition, json_field_name, package_name, struct_name, field_name, field_C_type,
						)
					}
					// --------------------------------------------------------------------------------------------------------------------------------
				} else if strings.HasSuffix(field_C_type, "_Pair_List") {
					if !have_encode_routine_normal_body_format {
						function_code += encode_routine_normal_body_format
						have_encode_routine_normal_body_format = true
					}
					// FIX QUICK:  clean this up
					function_code += fmt.Sprintf("\n        // as-yet-undone (encoding _Pair_List)\n")
					function_code += fmt.Sprintf("        // package_name = %s\n", package_name)
					function_code += fmt.Sprintf("        //  struct_name = %s\n", struct_name)
					function_code += fmt.Sprintf("        //   field_name = %s\n", field_name)
					function_code += fmt.Sprintf("        // field_C_type = %s\n", field_C_type)
					function_code += fmt.Sprintf("        // FIX QUICK:  encoding placeholder for package %s struct %s field %s, of type %s\n",
						package_name, struct_name, field_name, field_C_type)
					if field_tag.json_omitempty {
						include_condition = fmt.Sprintf("%s_%s_ptr->%s.count != 0", package_name, struct_name, field_name)
					} else {
						include_condition = "1"
					}
					// FIX QUICK:  Verify that this block of code has been moved up intact, then delete it here.
					/*
						go_type := struct_field_Go_types[struct_name][field_name]
						field_tag := strict_json_field_tag(field_name, struct_field_tags[struct_name][field_name])
						var json_field_name string
						if  field_tag.json_field_name != "" {
						    json_field_name = field_tag.json_field_name
						} else if matches := last_go_type_component.FindStringSubmatch(go_type); matches != nil {
						    // This possible adjustment of the field name is needed because we might have a complex
						    // Go field with a Go type like "*config.Config" that is represented in the Go code as
						    // an anonymous field (i.e., with no stated name).  In that case, Go's JSON package
						    // will encode that field using a field name which is only the last component of the Go
						    // typename, not the entire typename.  So we must replicate that behavior here.
						    json_field_name = matches[1]
						} else {
						    json_field_name = field_name
						}
					*/
					function_code += fmt.Sprintf(`
	json_t *%[3]s_%s_%s = %[6]s_ptr_as_JSON_ptr(&%[3]s_%s_ptr->%s);
	if (%[1]s) {
	    // %[3]s_%s_ptr->%s object value
	    if (json_object_set_new(json, "%[2]s", %[3]s_%s_%s) != 0) {
		failure = "cannot set %[3]s_%s_ptr->%s %s subobject value into object";
		break;
	    }
	}`,
						include_condition, json_field_name, package_name, struct_name, field_name, field_C_type,
					)
					// --------------------------------------------------------------------------------------------------------------------------------
				} else if strings.HasSuffix(field_C_type, "_Ptr_List") {
					if !have_encode_routine_normal_body_format {
						function_code += encode_routine_normal_body_format
						have_encode_routine_normal_body_format = true
					}
					// FIX QUICK:  clean this up
					function_code += fmt.Sprintf("\n        // as-yet-undone (encoding _Ptr_List)\n")
					function_code += fmt.Sprintf("        // package_name = %s\n", package_name)
					function_code += fmt.Sprintf("        //  struct_name = %s\n", struct_name)
					function_code += fmt.Sprintf("        //   field_name = %s\n", field_name)
					function_code += fmt.Sprintf("        // field_C_type = %s\n", field_C_type)
					function_code += fmt.Sprintf("        // FIX QUICK:  encoding placeholder for package %s struct %s field %s, of type %s\n",
						package_name, struct_name, field_name, field_C_type)

					// FIX QUICK:  generalize this to include support for omitempty
					/*
						// package_name = transit
						//  struct_name = TimeSeries
						//   field_name = MetricSamples
						// field_C_type = transit_MetricSample_Ptr_List

						json_t *transit_TimeSeries_MetricSamples = transit_MetricSample_Ptr_List_ptr_as_JSON_ptr(&transit_TimeSeries->MetricSamples);
						if (transit_TimeSeries->MetricSamples.count != 0) {
						    if (json_object_set_new(json, "metricSamples", transit_TimeSeries_MetricSamples) != 0) {
							failure = "cannot set transit_MetricSample_Ptr_List subobject value into object";
							break;
						    }
						}

					*/

					function_code += fmt.Sprintf(`
	json_t *%[2]s_%s_%s = %[5]s_ptr_as_JSON_ptr(&%[2]s_%s_ptr->%s);
	if (%[2]s_%s_ptr->%s.count != 0) {
	    if (json_object_set_new(json, "%[1]s", %s_%s_%s) != 0) {
		failure = "cannot set %[2]s_%s_ptr->%s %s subobject value into object";
		break;
	    }
	}`,
						json_field_name, package_name, struct_name, field_name, field_C_type,
					)
					// --------------------------------------------------------------------------------------------------------------------------------
				} else if strings.HasSuffix(field_C_type, "_List") {
					if !have_encode_routine_normal_body_format {
						function_code += encode_routine_normal_body_format
						have_encode_routine_normal_body_format = true
					}
					// FIX QUICK:  clean this up
					function_code += fmt.Sprintf("\n        // as-yet-undone (encoding _List)\n")
					function_code += fmt.Sprintf("        // package_name = %s\n", package_name)
					function_code += fmt.Sprintf("        //  struct_name = %s\n", struct_name)
					function_code += fmt.Sprintf("        //   field_name = %s\n", field_name)
					function_code += fmt.Sprintf("        // field_C_type = %s\n", field_C_type)
					function_code += fmt.Sprintf("        // FIX QUICK:  encoding placeholder for package %s struct %s field %s, of type %s\n",
						package_name, struct_name, field_name, field_C_type)

					/*
						// as-yet-undone (encoding _List)
						// package_name = transit
						//  struct_name = ResourceWithMetrics
						//   field_name = Metrics
						// field_C_type = transit_TimeSeries_List
						// FIX QUICK:  encoding placeholder for package transit struct ResourceWithMetrics field Metrics, of type transit_TimeSeries_List

						json_t *json_Metrics = transit_TimeSeries_List_ptr_as_JSON_ptr(&transit_ResourceWithMetrics->Metrics);
						if (json_Metrics != NULL) {
						    if (json_object_set_new(json, "metrics", json_Metrics) != 0) {
							failure = "cannot set transit_TimeSeries_List subobject value into object";
							break;
						    }
						}
					*/
					function_code += fmt.Sprintf(`
	// FIX MAJOR:  deal correctly with the include_condition (for omitempty support)
	json_t *json_%[5]s = %[6]s_ptr_as_JSON_ptr(&%[3]s_%s_ptr->%s);
	if (json_%[5]s != NULL) {
	    if (json_object_set_new(json, "%[2]s", json_%[5]s) != 0) {
		failure = "cannot set %[3]s_%s_ptr->%s %s subobject value into object";
		break;
	    }
	}`,
						include_condition, json_field_name, package_name, struct_name, field_name, field_C_type,
					)
					// --------------------------------------------------------------------------------------------------------------------------------
				} else if strings.HasSuffix(field_C_type, "_List_Ptr") {
					if !have_encode_routine_normal_body_format {
						function_code += encode_routine_normal_body_format
						have_encode_routine_normal_body_format = true
					}
					// FIX QUICK:  clean this up
					function_code += fmt.Sprintf("\n        // as-yet-undone (encoding _List_Ptr)\n")
					function_code += fmt.Sprintf("        // package_name = %s\n", package_name)
					function_code += fmt.Sprintf("        //  struct_name = %s\n", struct_name)
					function_code += fmt.Sprintf("        //   field_name = %s\n", field_name)
					function_code += fmt.Sprintf("        // field_C_type = %s\n", field_C_type)
					function_code += fmt.Sprintf("        // FIX QUICK:  encoding placeholder for package %s struct %s field %s, of type %s\n",
						package_name, struct_name, field_name, field_C_type)

					/*
						// package_name = transit
						//  struct_name = OperationResults
						//   field_name = Results
						// field_C_type = transit_OperationResult_List_Ptr

						// FIX MAJOR:  deal correctly with the include_condition (for omitempty support)
						if (1) {
						    // json_t *json_Results = transit_OperationResult_List_ptr_as_JSON_ptr(transit_OperationResults->Results);
						    json_t *json_Results = transit_OperationResult_List_Ptr_ptr_as_JSON_ptr(&transit_OperationResults->Results);
						    if (json_Results != NULL) {
							if (json_object_set_new(json, "results", json_Results) != 0) {
							    failure = "cannot set transit_OperationResult_List_Ptr into object";
							    break;
							}
						    }
						}
					*/
					function_code += fmt.Sprintf(`
	// FIX MAJOR:  deal correctly with the include_condition (for omitempty support)
	if (1) {
	    // use pointer_base_types[field_C_type] if we need to use this next line
	    // json_t *json_%[5]s = transit_XXX_List_ptr_as_JSON_ptr(%[3]s_%s_ptr->%[5]s);
	    json_t *json_%[5]s = %[6]s_ptr_as_JSON_ptr(&%[3]s_%s_ptr->%s);
	    if (json_%[5]s != NULL) {
		if (json_object_set_new(json, "%[2]s", json_%[5]s) != 0) {
		    failure = "cannot set %[3]s_%s_ptr->%s %s into object";
		    break;
		}
	    }
	}`,
						include_condition, json_field_name, package_name, struct_name, field_name, field_C_type,
					)
					// --------------------------------------------------------------------------------------------------------------------------------
				} else if strings.HasSuffix(field_C_type, "_Ptr") {
					if field_tag.json_omitempty {
						include_condition = fmt.Sprintf("%s_%s_%s != NULL", package_name, struct_name, field_name)
					} else {
						include_condition = "1"
					}
					// FIX QUICK:  Verify that this block of code has been moved up intact, then delete it here.
					/*
						go_type := struct_field_Go_types[struct_name][field_name]
						field_tag := strict_json_field_tag(field_name, struct_field_tags[struct_name][field_name])
						var json_field_name string
						if  field_tag.json_field_name != "" {
						    json_field_name = field_tag.json_field_name
						} else if matches := last_go_type_component.FindStringSubmatch(go_type); matches != nil {
						    // This possible adjustment of the field name is needed because we might have a complex
						    // Go field with a Go type like "*config.Config" that is represented in the Go code as
						    // an anonymous field (i.e., with no stated name).  In that case, Go's JSON package
						    // will encode that field using a field name which is only the last component of the Go
						    // typename, not the entire typename.  So we must replicate that behavior here.
						    json_field_name = matches[1]
						} else {
						    json_field_name = field_name
						}
					*/
					if !have_encode_routine_normal_body_format {
						function_code += encode_routine_normal_body_format
						have_encode_routine_normal_body_format = true
					}
					function_code += fmt.Sprintf(`
	json_t *%[3]s_%s_%s = %[6]s_ptr_as_JSON_ptr(%[3]s_%s_ptr->%s);
	if (%[1]s) {
	    // %[3]s_%s_ptr->%s object value
	    if (json_object_set_new(json, "%[2]s", %[3]s_%s_%s) != 0) {
		failure = "cannot set %[3]s_%s_ptr->%s %s subobject value into object";
		break;
	    }
	}`,
						include_condition, json_field_name, package_name, struct_name, field_name, pointer_base_types[field_C_type],
					)
					// --------------------------------------------------------------------------------------------------------------------------------
				} else {

					/*
					   // Generate routines similar to this, stepping through all of the individual fields of a structure
					   // and &&ing the tests for all of those fields.  The is_struct_timespec_ptr_zero_value() call will
					   // be special, and custom-built outside of the code generation.  We could have collapsed out that
					   // level and just accesse the internal fields of a milliseconds_MillisecondTimestamp->Time_
					   // variable directly, to eliminate one level of function call.  But that would special-case the
					   // code generation for that type, and we would rather avoid that sort of adjustment.

					   bool is_milliseconds_MillisecondTimestamp_ptr_zero_value(const milliseconds_MillisecondTimestamp *milliseconds_MillisecondTimestamp_ptr) {
					       return is_struct_timespec_ptr_zero_value(&milliseconds_MillisecondTimestamp_ptr->Time_);
					   }
					*/

					// FIX QUICK:  deal properly with the omitempty handling in this branch;
					// what we have right now is a poor substitute for actually checking for a zero value
					// of the affected structure, and really just constitutes basic error checking that
					// should be unconditional instead of reflecting the omitempty option
					if field_tag.json_omitempty {
						// FIX QUICK:  Add an extra include condition for a milliseconds_MillisecondTimestamp value not being zero (or equivalent).
						// include_condition = "1"
						include_condition = fmt.Sprintf("!is_%[4]s_ptr_zero_value(&%[1]s_%s_ptr->%s)", package_name, struct_name, field_name, field_C_type)
					} else {
						include_condition = "1"
					}
					if field_C_type == "struct_timespec" {
						function_code += fmt.Sprintf(encode_routine_struct_timespec_body_format, package_name, struct_name, field_name, field_C_type)
					} else {
						if !have_encode_routine_normal_body_format {
							function_code += encode_routine_normal_body_format
							have_encode_routine_normal_body_format = true
						}
						// FIX QUICK:  Check the json_field_name, to make sure it is handled correctly.
						// FIX QUICK:  clean this up
						function_code += fmt.Sprintf("\n        // as-yet-undone (encoding default)\n")
						function_code += fmt.Sprintf("        //    package_name = %s\n", package_name)
						function_code += fmt.Sprintf("        //     struct_name = %s\n", struct_name)
						function_code += fmt.Sprintf("        //      field_name = %s\n", field_name)
						function_code += fmt.Sprintf("        // json_field_name = %s\n", json_field_name)
						function_code += fmt.Sprintf("        //    field_C_type = %s\n", field_C_type)
						function_code += fmt.Sprintf("        // FIX QUICK:  encoding placeholder for package %s struct %s field %s, of type %s\n",
							package_name, struct_name, field_name, field_C_type)

						/*
							//    package_name = transit
							//     struct_name = TypedValue
							//      field_name = TimeValue
							// json_field_name = timeValue
							//    field_C_type = milliseconds_MillisecondTimestamp

							json_t *json_TimeValue = milliseconds_MillisecondTimestamp_ptr_as_JSON_ptr(&transit_TypedValue->TimeValue);
							if (json_TimeValue != NULL) {
							    if (json_object_set_new(json, "timeValue", json_TimeValue) != 0) {
								failure = "cannot set milliseconds_MillisecondTimestamp subobject value into object";
								break;
							    }
							}
						*/
						// FIX MINOR:  Perhaps improve the failure messages to specify the particular structure/field in question,
						// to better pinpoint the specific problem that is occurring should it happen in a deployed system.
						function_code += fmt.Sprintf(`
	// FIX QUICK:  Deal properly with the omitempty handling here,
	// in particular for a milliseconds_MillisecondTimestamp object.
	if (%s) {
	    json_t *json_%[5]s = %[6]s_ptr_as_JSON_ptr(&%[3]s_%s_ptr->%s);
	    if (json_%[5]s != NULL) {
		if (json_object_set_new(json, "%[2]s", json_%[5]s) != 0) {
		    failure = "cannot set %[3]s_%s_ptr->%s %s subobject value into JSON object";
		    // Hopefully, that failure has failed in a manner which does not capture a copy of the json_%[5]s pointer.
		    // Assuming that is the case, we ought to free the block of memory here.  However, in order to allow the
		    // application to continue running for awhile in spite of a possible memory leak here, for the time being
		    // we won't invoke this call to free().
		    // free(json_%[5]s);
		    break;
		}
	    } else {
		failure = "cannot convert %[6]s value into a JSON object";
		break;
	    }
	}`,
							include_condition, json_field_name, package_name, struct_name, field_name, field_C_type,
						)
					}
				}
				// --------------------------------------------------------------------------------------------------------------------------------
			}

			// FIX QUICK:  clean this up
			// package_name = milliseconds
			//  struct_name = MillisecondTimestamp
			//   field_name = Time_
			// field_C_type = struct_timespec
			// is_struct_timespec_ptr_zero_value(&milliseconds_MillisecondTimestamp->Time_)

			is_zero_function_code += is_zero_field_separator
			is_zero_function_code += fmt.Sprintf("is_%[4]s_ptr_zero_value(&%[1]s_%s_ptr->%s)",
				package_name, struct_name, field_name, field_C_type,
			)
			is_zero_field_separator = " &&\n        "
		}
	}

	var footer_code bytes.Buffer
	if err := footer_template.Execute(&footer_code, boilerplate_variables); err != nil {
		panic("encode routine footer processing failed")
	}
	function_code += footer_code.String()

	is_zero_function_code += fmt.Sprintf(`
    ;
}
`)
	function_code += is_zero_function_code

	return function_code, err
}

/*
// This is a sample decode routine, splayed out all in one piece so I can see what kinds
// of code constructions need to be generated.
{{.StructName}} *decode{{.StructName}}(const char *json_str) {
  {{.StructName}} *{{.StructName}}_ptr = NULL;
  size_t size = 0;
  json_error_t error;
  json_t *json = NULL;

  json = json_loads(json_str, 0, &error);
  if (json) {
    fprintf(stderr, "decode{{.StructName}} error on line %d: %s\n", error.line, error.text);
  } else {
    json_t *jsonCfg         = json_object_get(json, "config");
    json_t *jsonCfgHostName = json_object_get(jsonCfg, "hostName");
    json_t *jsonCfgAccount  = json_object_get(jsonCfg, "account");
    json_t *jsonCfgToken    = json_object_get(jsonCfg, "token");
    json_t *jsonCfgSSL      = json_object_get(jsonCfg, "ssl");

    size_t jsonCfgHostName_len = json_string_length(jsonCfgHostName);
    size_t jsonCfgAccount_len  = json_string_length(jsonCfgAccount);
    size_t jsonCfgToken_len    = json_string_length(jsonCfgToken);

    // incrementally compute a total size for allocation of the
    // target struct, including all the strings it refers to
    size = sizeof({{.StructName}});
    size += jsonCfgHostName_len + NUL_TERM_LEN;
    size += jsonCfgAccount_len  + NUL_TERM_LEN;
    size += jsonCfgToken_len    + NUL_TERM_LEN;

    // allocate and fill the target struct by pointer
    {{.StructName}}_ptr = ({{.StructName}} *) malloc(size);
    if ({{.StructName}}_ptr == NULL) {
      fprintf(stderr, "decode{{.StructName}} error: %s\n", "malloc failed");
    } else {
      char *ptr = (char *){{.StructName}}_ptr;
      ptr += sizeof({{.StructName}});
      {{.StructName}}_ptr->config.hostName = strcpy(ptr, json_string_value(jsonCfgHostName));
      ptr += jsonCfgHostName_len + NUL_TERM_LEN;
      {{.StructName}}_ptr->config.account = strcpy(ptr, json_string_value(jsonCfgAccount));
      ptr += jsonCfgAccount_len + NUL_TERM_LEN;
      {{.StructName}}_ptr->config.token = strcpy(ptr, json_string_value(jsonCfgToken));
      {{.StructName}}_ptr->config.ssl = json_boolean_value(jsonCfgSSL);
    }

    json_decref(json);
  }

  return {{.StructName}}_ptr;
}
*/

func generate_decode_PackageName_StructTypeName_ptr_tree(
	package_name string,
	struct_name string,

	// map[base_type_ptr]base_type
	pointer_base_types map[string]string,

	// map[key_type][]value_type
	key_value_pair_types map[string][]string,

	// map[enum_name]enum_type
	enum_typedefs map[string]string,

	// map[struct_name][]field_name
	struct_fields map[string][]string,

	// map[struct_name]map[field_name] = field_Go_type
	struct_field_Go_types map[string]map[string]string,

	// map[struct_name]map[field_name] = field_type
	struct_field_C_types map[string]map[string]string,

	// map[struct_name]map[field_name] = field_tag
	struct_field_tags map[string]map[string]string,
) (
	function_code string,
	err error,
) {
	function_code = ""

	var JSON_as_object_template_ptr_part1 = `{{.Package_StructName}} *JSON_as_{{.Package_StructName}}_ptr(json_t *json) {
    {{.Package_StructName}} *{{.StructName}}_ptr = ({{.Package_StructName}} *) calloc(1, sizeof({{.Package_StructName}}));
    if ({{.StructName}}_ptr == NULL) {
	// FIX MAJOR:  Invoke proper logging for error conditions.
	fprintf(stderr, FILE_LINE "ERROR:  in JSON_as_{{.Package_StructName}}_ptr, %s\n", "malloc failed");
    } else {
	int failed = 0;
`

	var JSON_as_object_template_ptr_part3 = `        if (json_unpack(json, "{{.UnpackFormat}}"
`

	var JSON_as_object_template_ptr_part5 = `        ) != 0) {
	    // FIX MAJOR:  Invoke proper logging for error conditions.
	    fprintf(stderr, FILE_LINE "ERROR:  in JSON_as_{{.Package_StructName}}_ptr, %s\n", "JSON unpacking failed");
	    failed = 1;
	} else {
`

	var JSON_as_object_template_ptr_part7 = `        }
	if (failed) {
	    free({{.StructName}}_ptr);
	    {{.StructName}}_ptr = NULL;
	}
    }
    return {{.StructName}}_ptr;
}
`

	object_template_part1 := template.Must(template.New("JSON_as_object_ptr_part1").Parse(JSON_as_object_template_ptr_part1))
	object_template_part3 := template.Must(template.New("JSON_as_object_ptr_part3").Parse(JSON_as_object_template_ptr_part3))
	object_template_part5 := template.Must(template.New("JSON_as_object_ptr_part5").Parse(JSON_as_object_template_ptr_part5))
	object_template_part7 := template.Must(template.New("JSON_as_object_ptr_part7").Parse(JSON_as_object_template_ptr_part7))

	field_objects := ""

	// FIX QUICK:  Once we have full omitempty support for all decoding and encoding, add the JSON "!" format specifier
	// at the object level during unpacking, to guarantee that all fields in an object have been unpacked.
	fields := struct_fields[struct_name] // []string
	max_json_field_name_len := 0
	unpack_separator := "{"
	unpack_terminator := "}"
	unpack_format := ""
	for _, field_name := range fields {
		field_tag := interpret_json_field_tag(field_name, struct_field_tags[struct_name][field_name])
		if !field_tag.json_omitalways {
			json_field_name_len := len(field_tag.json_field_name)
			if max_json_field_name_len < json_field_name_len {
				max_json_field_name_len = json_field_name_len
			}

			// FIX MAJOR;  There are certain field types that devolve to a string or to a 32-bit integer,
			// that we should be handling differently here.

			// We specify "s?:X" instead of "s:X" as the unpack format for an optional field
			// (i.e., when "omitempty" is present in the struct field tag).
			var optional string
			if field_tag.json_omitempty {
				optional = "?"
			} else {
				optional = ""
			}
			var json_field_format string
			field_C_type := struct_field_C_types[struct_name][field_name]
			switch field_C_type {
			// FIX MAJOR:  Here we are making the assumption that Go's "int" type has the same size as C's "int" type,
			// without doing any sort of checking to verify that assumption.
			case "bool":
				json_field_format = "b"
			case "int":
				json_field_format = "i"
			case "int32":
				json_field_format = "i"
			case "int64":
				json_field_format = "I"
			case "float64":
				json_field_format = "f"
			case "string":
				json_field_format = "s"
			default:
				if enum_C_type, ok := enum_typedefs[field_C_type]; ok {
					if enum_C_type == "string" {
						json_field_format = "s"
					} else if enum_C_type == "int" {
						// An enumeration type is apparently treated by the C compiler as
						// an unsigned int (32 bits), so we use "i" here instead of "I".
						json_field_format = "i"
					} else {
						json_field_format = "o"
					}
				} else {
					json_field_format = "o"
				}
			}
			// The special-case handling for a "struct_timespec" C type currently only works if
			// that is the only field in the struct.  That's because we don't have an explicit
			// name for the field, and we're performing the direct extraction of an individual
			// scalar from the JSON element, not the value of some format-determined type based
			// on a key-string retrieval from a JSON hash object.
			if field_C_type == "struct_timespec" {
				unpack_format += "I"
				unpack_terminator = ""
			} else {
				unpack_format += unpack_separator + "s" + optional + ":" + json_field_format
				unpack_separator = " "
			}
		}
	}
	unpack_format += unpack_terminator

	last_go_type_component := regexp.MustCompile(`\.([^.]+)$`)
	// field_objects := ""
	field_unpacks := ""
	field_values := ""
	for _, field_name := range fields {
		// field_tag := interpret_json_field_tag(field_name, struct_field_tags[struct_name][field_name])
		go_type := struct_field_Go_types[struct_name][field_name]
		field_tag := strict_json_field_tag(field_name, struct_field_tags[struct_name][field_name])
		var json_field_name string
		if field_tag.json_field_name != "" {
			json_field_name = field_tag.json_field_name
		} else if matches := last_go_type_component.FindStringSubmatch(go_type); matches != nil {
			// This possible adjustment of the field name is needed because we might have a complex
			// Go field with a Go type like "*config.Config" that is represented in the Go code as
			// an anonymous field (i.e., with no stated name).  In that case, Go's JSON package
			// will encode that field using a field name which is only the last component of the Go
			// typename, not the entire typename.  So we must replicate that behavior here.
			json_field_name = matches[1]
		} else {
			json_field_name = field_name
		}
		json_field_name_len := len(json_field_name)

		if !field_tag.json_omitalways {
			field_C_type := struct_field_C_types[struct_name][field_name]
			switch field_C_type {
			case "bool":
				fallthrough
			case "int":
				fallthrough
			case "int32":
				fallthrough
			case "int64":
				fallthrough
			case "float64":
				fallthrough
			case "string":
				// FIX QUICK:  Does this handle field_tag.json_omitempty correctly for all relevant types?
				field_unpacks += fmt.Sprintf("            , \"%[2]s\",%*s&%[1]s_ptr->%[5]s\n",
					struct_name, json_field_name, max_json_field_name_len-json_field_name_len+1, " ", field_name)
			default:
				// FIX MAJOR:  When decoding an optional field (as specified by the presence of "omitempty" in the struct field tag),
				// pay attention to whether we really got back the object we thought we did, before dereferencing it.
				// --------------------------------------------------------------------------------------------------------------------------------
				if enum_C_type, ok := enum_typedefs[field_C_type]; ok {
					// FIX QUICK:  Does this handle field_tag.json_omitempty correctly for all relevant types?
					if enum_C_type == "string" {
						field_objects += fmt.Sprintf("        char *%s_as_string;\n", field_name)
						field_unpacks += fmt.Sprintf("            , \"%s\",%*s&%s_as_string\n",
							json_field_name, max_json_field_name_len-json_field_name_len+1, " ", field_name)
						field_values += fmt.Sprintf(`
	    // FIX MAJOR:  An enumeration type is apparently treated by the C compiler as an unsigned int.  So if
	    // we want to test an enumeration variable for a negative value, we need to cast it as a plain int.
	    // Alternatively, we now define our enumeration values so 0 is never used, and reserve the string at
	    // offset 0 in the corresponding _String array for that purpose, so we can test instead for equality
	    // to 0.  That is probably a better design overall, as it more readily allows for testing for the
	    // type's zero value in a structure where we might have an "omitempty" field that has been cleared
	    // when the structure was allocated but never modified thereafter.  That also means we have the
	    // implementation of enumeration_value() return a 0 instead of -1 if the string in hand cannot be
	    // found in the _String array.
	    if ((int) (%[2]s_ptr->%s = enumeration_value(%[1]s_%[4]s_String, arraysize(%[1]s_%[4]s_String), %[3]s_as_string)) == 0) {
		fprintf(stderr, FILE_LINE "ERROR:  cannot find the %[4]s enumeration value for %[3]s '%%s'\n", %[3]s_as_string);
		failed = 1;
	    }
`, package_name, struct_name, field_name, field_C_type)
					} else if enum_C_type == "int" {
						field_unpacks += fmt.Sprintf("            , \"%s\",%*s&%s_ptr->%s\n",
							json_field_name, max_json_field_name_len-json_field_name_len+1, " ", struct_name, field_name)
					} else {
						// This is a placeholder to show what would have been converted.
						// If this ever shows up, we have an uncovered case and we'll need to extend this code.
						field_unpacks += fmt.Sprintf("            // , \"%s\",%*s&json_%s\n",
							json_field_name, max_json_field_name_len-json_field_name_len+1, " ", field_name)
					}
					// --------------------------------------------------------------------------------------------------------------------------------
				} else if strings.HasSuffix(field_C_type, "_Pair_List") {
					// FIX QUICK:  Does this handle field_tag.json_omitempty correctly for all relevant types?
					field_values += fmt.Sprintf("            // package_name = %s\n", package_name)
					field_values += fmt.Sprintf("            //  struct_name = %s\n", struct_name)
					field_values += fmt.Sprintf("            //   field_name = %s\n", field_name)
					field_values += fmt.Sprintf("            // field_C_type = %s\n", field_C_type)
					field_objects += fmt.Sprintf("        json_t *json_%s;\n", field_name)
					field_unpacks += fmt.Sprintf("            , \"%s\",%*s&json_%s\n",
						json_field_name, max_json_field_name_len-json_field_name_len+1, " ", field_name)

					// FIX QUICK:  Create the missing JSON_as_XXX_YYY_Pair_List_ptr() routines that will make the rest of this work as intended.
					field_values += fmt.Sprintf(
						`	    %[4]s *%[4]s_ptr = JSON_as_%[4]s_ptr(json_%[3]s);
	    if (%[4]s_ptr == NULL) {
		fprintf(stderr, FILE_LINE "ERROR:  in JSON_as_%[1]s_%s_ptr, %%s\n", "%[4]s_ptr is NULL");
		failed = 1;
	    } else {
		%[2]s_ptr->%s = *%[4]s_ptr;
		free(%[4]s_ptr);
	    }
`,
						package_name, struct_name, field_name, field_C_type)
					// --------------------------------------------------------------------------------------------------------------------------------
				} else if strings.HasSuffix(field_C_type, "_Ptr_List") {
					// FIX QUICK:  Does this handle field_tag.json_omitempty correctly for all relevant types?
					var omitempty_condition string
					if field_tag.json_omitempty {
						omitempty_condition = "1"
					} else {
						omitempty_condition = "0"
					}
					field_values += fmt.Sprintf("\n            // as-yet-undone (decoding _Ptr_List)\n")
					field_values += fmt.Sprintf("            // package_name = %s\n", package_name)
					field_values += fmt.Sprintf("            //  struct_name = %s\n", struct_name)
					field_values += fmt.Sprintf("            //   field_name = %s\n", field_name)
					field_values += fmt.Sprintf("            // field_C_type = %s\n", field_C_type)
					field_values += fmt.Sprintf("            // FIX QUICK:  decoding placeholder for package %s struct %s field %s, of type %s\n",
						package_name, struct_name, field_name, field_C_type)

					// This part seems to be okay.
					field_objects += fmt.Sprintf("        json_t *json_%s = NULL;\n", field_name)
					field_unpacks += fmt.Sprintf("            , \"%s\",%*s&json_%s\n",
						json_field_name, max_json_field_name_len-json_field_name_len+1, " ", field_name)

					/*
						    // package_name = transit
						    //  struct_name = TimeSeries
						    //   field_name = MetricSamples
						    // field_C_type = transit_MetricSample_Ptr_List

						    if (json_MetricSamples == NULL) {
							// FIX QUICK:  When processing the failed flag below, be sure that we also
							// take care to recursively free up any memory that we have allocated to this
							// or any other sub-object data structure before freeing the top-level pointer.
							TimeSeries->MetricSamples.count = 0;
							TimeSeries->MetricSamples.items = NULL;
							if (!omitempty) {  // This is only a reportable error if the field is not declared as "omitempty".
							    fprintf(stderr, FILE_LINE "ERROR:  in JSON_as_transit_TimeSeries_ptr, %s\n", "json_MetricSamples is NULL");
							    failed = 1;
							}
						    } else {
							transit_MetricSample_Ptr_List *transit_MetricSample_Ptr_List_ptr = JSON_as_transit_MetricSample_Ptr_List_ptr(json_MetricSamples);
							if (transit_MetricSample_Ptr_List_ptr == NULL) {
							    TimeSeries->MetricSamples.count = 0;
							    TimeSeries->MetricSamples.items = NULL;
							    fprintf(stderr, FILE_LINE "ERROR:  in JSON_as_transit_TimeSeries_ptr, %s\n", "transit_MetricSample_Ptr_List_ptr is NULL");
							    failed = 1;
							} else {
							    TimeSeries->MetricSamples = *transit_MetricSample_Ptr_List_ptr;
							    free(transit_MetricSample_Ptr_List_ptr);
							}
						    }
					*/
					field_values += fmt.Sprintf(
						`            if (json_%[3]s == NULL) {
		// FIX QUICK:  When processing the failed flag below, be sure that we also
		// take care to recursively free up any memory that we have allocated to this
		// or any other sub-object data structure before freeing the top-level pointer.
		%[2]s_ptr->%s.count = 0;
		%[2]s_ptr->%s.items = NULL;
		if (!%[5]s) {  // This is only a reportable error if the field is not declared as "omitempty".
		    fprintf(stderr, FILE_LINE "ERROR:  in JSON_as_%[1]s_%s_ptr, %%s\n", "json_%[3]s is NULL");
		    failed = 1;
		}
	    } else {
		%[4]s *%[4]s_ptr = JSON_as_%[4]s_ptr(json_%[3]s);
		if (%[4]s_ptr == NULL) {
		    %[2]s_ptr->%s.count = 0;
		    %[2]s_ptr->%s.items = NULL;
		    fprintf(stderr, FILE_LINE "ERROR:  in JSON_as_%[1]s_%s_ptr, %%s\n", "%[4]s_ptr is NULL");
		    failed = 1;
		} else {
		    %[2]s_ptr->%s = *%[4]s_ptr;
		    free(%[4]s_ptr);
		}
	    }
`, package_name, struct_name, field_name, field_C_type, omitempty_condition)
					// --------------------------------------------------------------------------------------------------------------------------------
				} else if strings.HasSuffix(field_C_type, "_List") {
					// FIX QUICK:  Does this handle field_tag.json_omitempty correctly for all relevant types?
					field_values += fmt.Sprintf("\n            // as-yet-undone (decoding _List)\n")
					field_values += fmt.Sprintf("            // package_name = %s\n", package_name)
					field_values += fmt.Sprintf("            //  struct_name = %s\n", struct_name)
					field_values += fmt.Sprintf("            //   field_name = %s\n", field_name)
					field_values += fmt.Sprintf("            // field_C_type = %s\n", field_C_type)
					field_values += fmt.Sprintf("            // FIX QUICK:  decoding placeholder for package %s struct %s field %s, of type %s\n",
						package_name, struct_name, field_name, field_C_type)
					field_objects += fmt.Sprintf("        json_t *json_%s;\n", field_name)
					field_unpacks += fmt.Sprintf("            , \"%s\",%*s&json_%s\n",
						json_field_name, max_json_field_name_len-json_field_name_len+1, " ", field_name)
					/*
						    // package_name = transit
						    //  struct_name = ResourceWithMetrics
						    //   field_name = Metrics
						    // field_C_type = transit_TimeSeries_List

						    if (1) {
							transit_TimeSeries_List *Metrics_ptr = JSON_as_transit_TimeSeries_List_ptr(json_Metrics);
							if (Metrics_ptr == NULL) {
							    fprintf(stderr, FILE_LINE "ERROR:  cannot find the transit_TimeSeries_List value for transit_ResourceWithMetrics_ptr->Metrics\n");
							    failed = 1;
							} else {
							    ResourceWithMetrics->Metrics = *Metrics_ptr;
							    free(Metrics_ptr);
							}
						    }
					*/
					field_values += fmt.Sprintf(`
	    // FIX MAJOR:  Deal correctly with the omitempty_condition.
	    if (1) {
		%[4]s *%[3]s_ptr = JSON_as_%[4]s_ptr(json_%[3]s);
		if (%[3]s_ptr == NULL) {
		    fprintf(stderr, FILE_LINE "ERROR:  cannot find the %[4]s value for %[2]s_ptr->%s\n");
		    failed = 1;
		} else {
		    %[2]s_ptr->%s = *%[3]s_ptr;
		    free(%[3]s_ptr);
		}
	    }
`, package_name, struct_name, field_name, field_C_type)
					// `, package_name, struct_name, field_name, field_C_type, omitempty_condition)
					// --------------------------------------------------------------------------------------------------------------------------------
				} else if strings.HasSuffix(field_C_type, "_List_Ptr") {
					// FIX QUICK:  Does this handle field_tag.json_omitempty correctly for all relevant types?
					field_values += fmt.Sprintf("\n            // as-yet-undone (decoding _List_Ptr)\n")
					field_values += fmt.Sprintf("            // package_name = %s\n", package_name)
					field_values += fmt.Sprintf("            //  struct_name = %s\n", struct_name)
					field_values += fmt.Sprintf("            //   field_name = %s\n", field_name)
					field_values += fmt.Sprintf("            // field_C_type = %s\n", field_C_type)
					field_values += fmt.Sprintf("            // FIX QUICK:  decoding placeholder for package %s struct %s field %s, of type %s\n",
						package_name, struct_name, field_name, field_C_type)
					field_objects += fmt.Sprintf("        json_t *json_%s = NULL;\n", field_name)
					field_unpacks += fmt.Sprintf("            , \"%s\",%*s&json_%s\n",
						json_field_name, max_json_field_name_len-json_field_name_len+1, " ", field_name)

					/*
						    // package_name = transit
						    //  struct_name = OperationResults
						    //   field_name = Results
						    // field_C_type = transit_OperationResult_List_Ptr
						    // FIX QUICK:  decoding placeholder for package transit struct OperationResults field Results, of type transit_OperationResult_List_Ptr
						    if (1) {
							transit_OperationResult_List *Results_ptr = JSON_as_transit_OperationResult_List_ptr(json_Results);
							if (Results_ptr == NULL) {
							    fprintf(stderr, FILE_LINE "ERROR:  cannot find the transit_OperationResult_List_Ptr value for OperationResults->Results\n");
							    failed = 1;
							} else {
							    OperationResults->Results = Results_ptr;
							}
						    }
					*/
					field_values += fmt.Sprintf(`
	    // FIX MAJOR:  Deal correctly with the omitempty_condition (json_%[3]s != NULL).
	    if (1) {
		%[4]s *%[3]s_ptr = JSON_as_%[4]s_ptr(json_%[3]s);
		if (%[3]s_ptr == NULL) {
		    fprintf(stderr, FILE_LINE "ERROR:  cannot find the %[4]s value for %[2]s_ptr->%s\n");
		    failed = 1;
		} else {
		    %[2]s_ptr->%s = %[3]s_ptr;
		}
	    }
`, package_name, struct_name, field_name, pointer_base_types[field_C_type])
					// --------------------------------------------------------------------------------------------------------------------------------
				} else if strings.HasSuffix(field_C_type, "_Ptr") {
					// FIX QUICK:  Does this handle field_tag.json_omitempty correctly for all relevant types?
					// FIX QUICK:  check that development of this branch is done
					field_values += fmt.Sprintf("            // package_name = %s\n", package_name)
					field_values += fmt.Sprintf("            //  struct_name = %s\n", struct_name)
					field_values += fmt.Sprintf("            //   field_name = %s\n", field_name)
					field_values += fmt.Sprintf("            // field_C_type = %s\n", field_C_type)
					field_objects += fmt.Sprintf("        json_t *json_%s;\n", field_name)
					// FIX MAJOR:  Possibly change the JSON field name in this next item, based on
					// an analysis of the Go type (for "*config.Config", reduce to just "Config")
					// and then on a possible override from the struct field tag.
					field_tags := struct_field_tags[struct_name][field_name]
					// FIX QUICK:  Verify that this block of code has been moved up intact, then delete it here.
					/*
						go_type := struct_field_Go_types[struct_name][field_name]
						field_tag := strict_json_field_tag(field_name, struct_field_tags[struct_name][field_name])
						var json_field_name string
						var json_field_name_len int
						if  field_tag.json_field_name != "" {
						    json_field_name = field_tag.json_field_name
						    json_field_name_len = len(json_field_name)
						} else if matches := last_go_type_component.FindStringSubmatch(go_type); matches != nil {
						    // This possible adjustment of the field name is needed because we might have a complex
						    // Go field with a Go type like "*config.Config" that is represented in the Go code as
						    // an anonymous field (i.e., with no stated name).  In that case, Go's JSON package
						    // will encode that field using a field name which is only the last component of the Go
						    // typename, not the entire typename.  So we must replicate that behavior here.
						    json_field_name = matches[1]
						    json_field_name_len = len(json_field_name)
						} else {
						    json_field_name = field_name
						    json_field_name_len = field_name_len
						}
					*/
					field_unpacks += fmt.Sprintf("            // Go type: %s\n", go_type)
					field_unpacks += fmt.Sprintf("            // Go field tags: %s\n", field_tags)
					field_unpacks += fmt.Sprintf("            // field_tag.json_field_name: %s\n", field_tag.json_field_name)
					field_unpacks += fmt.Sprintf("            , \"%s\",%*s&json_%s\n",
						json_field_name, max_json_field_name_len-json_field_name_len+1, " ", field_name)
					field_values += fmt.Sprintf("            %[2]s_ptr->%s = JSON_as_%[4]s_ptr(json_%[3]s);\n",
						package_name, struct_name, field_name, pointer_base_types[field_C_type])
					// --------------------------------------------------------------------------------------------------------------------------------
				} else {
					// FIX QUICK:  There are lots of subtle distinctions and adjustments that need to
					// be made in this branch, that we were (at the time of this writing) not yet making.
					// Also handle a field_C_type of:
					//     "milliseconds_MillisecondTimestamp"
					//     "ResourceStatus"  (should that be "transit_ResourceStatus" instead?)
					//     "TracerContext"   (should that be "transit_TracerContext"  instead?)
					// FIX QUICK:  That is all close to being done, if it's not already done; check the json_field_name,
					// though, to make sure it is handled correctly.
					var decode_condition string
					if field_tag.json_omitempty {
						if field_C_type == "struct_timespec" {
							decode_condition = fmt.Sprintf("pure_milliseconds != 0")
						} else {
							decode_condition = fmt.Sprintf("json_%s != NULL", field_name)
						}
					} else {
						decode_condition = "1"
					}
					if field_C_type == "struct_timespec" {
						field_objects += fmt.Sprintf("        json_int_t pure_milliseconds = 0;\n")
						field_unpacks += fmt.Sprintf("            , &pure_milliseconds\n")
					} else {
						// A NULL assignment is needed here as a fundamental means of telling us whether an optional JSON
						// field representing a subobject actually was actually present in the input and got unpacked.
						// Otherwise, we get some random number as the initial value of the pointer, that random value is
						// retained if no unpacking occurs for this field, and we have no way to test whether unpacking
						// occurred for this field.
						field_objects += fmt.Sprintf("        json_t *json_%s = NULL;\n", field_name)
						field_unpacks += fmt.Sprintf("            , \"%s\",%*s&json_%s\n",
							json_field_name, max_json_field_name_len-json_field_name_len+1, " ", field_name)
					}
					// FIX QUICK:  this seems to be working; clean up the extra development output here
					field_values += fmt.Sprintf("\n            // decoding as-yet-undone (decoding default)\n")
					field_values += fmt.Sprintf("            // package_name = %s\n", package_name)
					field_values += fmt.Sprintf("            //  struct_name = %s\n", struct_name)
					field_values += fmt.Sprintf("            //   field_name = %s\n", field_name)
					field_values += fmt.Sprintf("            // field_C_type = %s\n", field_C_type)
					field_values += fmt.Sprintf("            // %[2]s->%s = JSON_as_%[4]s_ptr(json_%[3]s);\n",
						package_name, struct_name, field_name, field_C_type)
					field_values += fmt.Sprintf("            // FIX QUICK:  decoding placeholder for package %s struct %s field %s, of type %s\n",
						package_name, struct_name, field_name, field_C_type)

					/*
						// package_name = transit
						//  struct_name = TypedValue
						//   field_name = TimeValue
						// field_C_type = milliseconds_MillisecondTimestamp
						// TypedValue->TimeValue = JSON_as_milliseconds_MillisecondTimestamp_ptr(json_TimeValue);

						milliseconds_MillisecondTimestamp *TimeValue_ptr = JSON_as_milliseconds_MillisecondTimestamp_ptr(json_TimeValue);
						if (TimeValue_ptr == NULL) {
						    fprintf(stderr, FILE_LINE "ERROR:  cannot find the milliseconds_MillisecondTimestamp value for TimeValue_ptr\n");
						    failed = 1;
						} else {
						    TypedValue->TimeValue = *TimeValue_ptr;
						}
					*/
					// FIX MAJOR:  Only emit a warning mesage if this was not an omitalways or omitempty field?

					field_values += fmt.Sprintf("            if (%s) {\n", decode_condition)

					if field_C_type == "struct_timespec" {
						// FIX MAJOR:  Decide if we want to fail this (and return a NULL pointer) if the pure_milliseconds value is zero.
						field_values += fmt.Sprintf(
							`		%s_ptr->%s.tv_sec  = (time_t) (pure_milliseconds / MILLISECONDS_PER_SECOND);
		%[1]s_ptr->%s.tv_nsec = (long) (pure_milliseconds %% MILLISECONDS_PER_SECOND) * NANOSECONDS_PER_MILLISECOND;
`, struct_name, field_name)
					} else {
						field_values += fmt.Sprintf(
							`		%[4]s *%[3]s_ptr = JSON_as_%[4]s_ptr(json_%[3]s);
		if (%[3]s_ptr == NULL) {
		    fprintf(stderr, FILE_LINE "ERROR:  cannot find the %[4]s value for %[2]s_ptr->%s\n");
		    failed = 1;
		} else {
		    %[2]s_ptr->%s = *%[3]s_ptr;
		    free(%[3]s_ptr);
		}
`, package_name, struct_name, field_name, field_C_type)
					}

					field_values += fmt.Sprintf("            }\n")

				}
				// --------------------------------------------------------------------------------------------------------------------------------
			}
		}
	}

	type object_template_variables struct {
		Package_StructName string
		StructName         string
		UnpackFormat       string
	}

	object_template_values := object_template_variables{
		Package_StructName: package_name + "_" + struct_name,
		StructName:         struct_name,
		UnpackFormat:       unpack_format,
	}

	var object_template_code1 bytes.Buffer
	var object_template_code3 bytes.Buffer
	var object_template_code5 bytes.Buffer
	var object_template_code7 bytes.Buffer

	if err := object_template_part1.Execute(&object_template_code1, object_template_values); err != nil {
		panic("object template processing failed")
	}

	if err := object_template_part3.Execute(&object_template_code3, object_template_values); err != nil {
		panic("object template processing failed")
	}

	if err := object_template_part5.Execute(&object_template_code5, object_template_values); err != nil {
		panic("object template processing failed")
	}

	if err := object_template_part7.Execute(&object_template_code7, object_template_values); err != nil {
		panic("object template processing failed")
	}

	function_code += object_template_code1.String()
	function_code += field_objects
	function_code += object_template_code3.String()
	function_code += field_unpacks
	function_code += object_template_code5.String()
	function_code += field_values
	function_code += object_template_code7.String()

	var decode_routine_header_template = `
{{.StructName}} *JSON_str_as_{{.StructName}}_ptr(const char *json_str, json_t **json) {
    json_error_t error;
    *json = json_loads(json_str, 0, &error);
    if (*json == NULL) {
	// FIX MAJOR:  Once we have our official support for logging in place,
	// produce a log message based on the content of the "error" object.
	printf(FILE_LINE "json for {{.StructName}} is NULL\n");
	return NULL;
    }
    {{.StructName}} *{{.StructName}}_ptr = JSON_as_{{.StructName}}_ptr(*json);

    // Logically, we want to make this json_decref() call to release our hold on the JSON
    // object we obtained, because we are now supposedly done with that JSON object.  But
    // if we do so now, that will destroy all the strings we obtained from json_unpack()
    // and stuffed into the returned C object tree.  So instead, we just allow that pointer
    // to be returned to the caller, to be passed thereafter to free_JSON() once the caller
    // is completely done with the returned C object tree.
    //
    // json_decref(*json);

    return {{.StructName}}_ptr;
}
`

	header_template := template.Must(template.New("decode_routine_header").Parse(decode_routine_header_template))

	type decode_routine_boilerplate_fields struct {
		StructName string
	}

	boilerplate_variables := decode_routine_boilerplate_fields{StructName: package_name + "_" + struct_name}

	var header_code bytes.Buffer
	if err := header_template.Execute(&header_code, boilerplate_variables); err != nil {
		panic("decode routine header processing failed")
	}
	function_code += header_code.String()

	return function_code, err
}

// Let's define a function that will generate the destroy_StructTypeName_ptr_tree() code, given the StructTypeName
// and a list of all the available structs and their individual fields and field types.
func generate_destroy_PackageName_StructTypeName_ptr_tree(
	package_name string,
	struct_name string,

	// map[key_type][]value_type
	key_value_pair_types map[string][]string,

	// map[typedef_name]typedef_type
	simple_typedefs map[string]string,

	// map[enum_name]enum_type
	enum_typedefs map[string]string,

	// map[struct_name][]field_name
	struct_fields map[string][]string,

	// map[struct_name]map[field_name] = field_type
	struct_field_C_types map[string]map[string]string,
) (
	function_code string,
	err error,
) {
	trailing_List := regexp.MustCompile(`(.+)_List$`)
	// FIX QUICK:  Check out the details of this pattern, once everything else is working; should we only recognize _Ptr (capital) again?
	trailing_Ptr := regexp.MustCompile(`(.+)_[Pp]tr$`)
	leading_package := regexp.MustCompile(package_name + "_(.+)")
	function_code = ""

	var destroy_routine_header_template = `void destroy_{{.StructName}}_ptr_tree({{.StructName}} *{{.StructName}}_ptr, json_t *json, bool free_pointers) {
`
	var destroy_routine_footer_template = `        free_JSON(json);
    }
}

`

	header_template := template.Must(template.New("destroy_routine_header").Parse(destroy_routine_header_template))
	footer_template := template.Must(template.New("destroy_routine_footer").Parse(destroy_routine_footer_template))

	type destroy_routine_boilerplate_fields struct {
		StructName string
	}

	boilerplate_variables := destroy_routine_boilerplate_fields{StructName: package_name + "_" + struct_name}

	var header_code bytes.Buffer
	if err := header_template.Execute(&header_code, boilerplate_variables); err != nil {
		panic("destroy routine header processing failed")
	}
	function_code += header_code.String()

	// Recursively destroy all other structures pointed to by the given pointer.
	// To do so, we use our conventions of naming structures to identify which of them are lists
	// and pointers.  Possibilities for field types that need special handling are the following:
	//
	//     string     is a "char *", which needs to be directly free()d
	//
	//     xxx_List   (special structure) check for a NULL pointer in the items subfield; otherwise,
	//                use the count subfield and walk through the array of items, each of which will be an
	//                "xxx *" element, handling each one in turn, recursively, before deleting the array itself
	//
	//     xxx        (any other structure) walk through the fields of the "xxx" structure, and deal with
	//                them individually, as otherwise described here for the types of those subfields
	//
	//     xxx_Ptr    is a pointer to a single object of type "xxx" (though that type might be "yyy_List");
	//                check for a NULL pointer, then delete all necessary subsidiary-object elements first
	//                before deleting the pointed-to object itself

	indent := "    "
	var process_item func(line_prefix string, item_type string, item_prefix string, member_op string, item_name string)
	process_item = func(line_prefix string, item_type string, item_prefix string, member_op string, item_name string) {

		// FIX LATER:  This output is here to help figure out why we're not free()ing deep enough, when that happens.
		// function_code += fmt.Sprintf("// process_item(line_prefix, \"%s\", \"%s\", \"%s\", \"%s\")\n",
		// item_type, item_prefix, member_op, item_name)

		// A xxx_List structure is just like any other structure we have manufactured, in that the details of its
		// fields have been recorded for our later use.  However, the .items field in this structure is special, in
		// that it refers not just to a single instance of the referred-to object, but to potentially many more.
		// So we must test for this structure before testing for other types of structures, so we guarantee that
		// the necessary special handling is applied.
		if matches := trailing_List.FindStringSubmatch(item_type); matches != nil {
			// We have a List of items; we just need to process the list, recursively destroying its individual elements.
			// We have an embedded xxx_List structure; we can presume its own internal construction,
			// and use that to destroy the complete set of individual elements in the list.
			base_type := matches[1]
			var field_member_op string
			if member_op == "->" {
				if item_name == "" {
					member_op = ""
				}
				field_member_op = "->"
			} else {
				field_member_op = "."
			}
			count_field := package_name + "_" + item_prefix + member_op + item_name + field_member_op + "count"
			items_field := package_name + "_" + item_prefix + member_op + item_name + field_member_op + "items"
			function_code += fmt.Sprintf("%s// list structure:  %s %s%s%s\n", line_prefix, item_type, item_prefix, member_op, item_name)
			// function_code += fmt.Sprintf("%s// list field:  %s\n", line_prefix, count_field)
			// function_code += fmt.Sprintf("%s// list field:  %s\n", line_prefix, items_field)
			function_code += fmt.Sprintf("%sif (%s != NULL) {\n", line_prefix, items_field)
			// FIX LATER:  If it turns out that there are no free() operations that should take place inside the
			// for loop, the for loop itself has no practical effect and should just be omitted from our generated
			// code.  That optimization awaits some future version of this program.
			//
			// We can't use a fixed loop-index name like "index" to iterate over the list items, because this code
			// might be invoked recursively and it would be a mistake for an inner generated loop to re-use the same
			// loop-index name as an outer generated loop.  So we must manufacture an index-veriable name that we
			// know won't be re-used during recursion.  For that purpose, we will use the line-indentation length as
			// a proxy for the process_item() nesting level.
			index_name := "index_" + strconv.Itoa(len(line_prefix)/len(indent))
			function_code += fmt.Sprintf("%sfor (int %s = %s; --%[2]s >= 0; ) {\n", line_prefix+indent, index_name, count_field)
			function_code += fmt.Sprintf("%s// delete one %s item's fields\n", line_prefix+indent+indent, base_type)
			list_item_name := fmt.Sprintf("items[%s]", index_name)
			process_item(line_prefix+indent+indent, base_type, item_prefix+member_op+item_name, field_member_op, list_item_name)
			function_code += fmt.Sprintf("%s}\n", line_prefix+indent)
			function_code += fmt.Sprintf("%sfree(%s);\n", line_prefix+indent, items_field)
			function_code += fmt.Sprintf("%s}\n", line_prefix)
			// FIX QUICK:  What would this line have added to the discussion, if anything?
			// process_item(line_prefix, base_type, item_prefix + member_op + item_name, field_member_op, item_name)
		} else if matches := trailing_Ptr.FindStringSubmatch(item_type); matches != nil {
			// We have a pointer to some other item.
			base_type := matches[1]
			// function_code += fmt.Sprintf("%s// process pointer:  %s %s%s%s\n", line_prefix, item_type, item_prefix, member_op, item_name)
			function_code += fmt.Sprintf("%s// pointer field:  %s %s%s%s\n", line_prefix, item_type, item_prefix, member_op, item_name)
			function_code += fmt.Sprintf("%sif (1) {\n", line_prefix)
			process_item(line_prefix+indent, base_type, item_prefix+member_op+item_name, "->", "")
			function_code += fmt.Sprintf("%sfree(%s_%s%s%s);\n", line_prefix+indent, package_name, item_prefix, member_op, item_name)
			function_code += fmt.Sprintf("%s}\n", line_prefix)
		} else if field_list, ok := struct_fields[item_type]; ok {
			// We have a known structure in the same package; we just need to recursively destroy its individual fields.
			var field_member_op string
			if member_op == "->" {
				if item_name == "" {
					field_member_op = ""
				} else {
					field_member_op = "."
				}
			} else {
				field_member_op = "."
			}
			function_code += fmt.Sprintf("%s// embedded structure:  %s %s%s%s\n", line_prefix, item_type, item_prefix, member_op, item_name)
			function_code += fmt.Sprintf("%sif (1) {\n", line_prefix)
			for _, field_name := range field_list {
				field_C_type := struct_field_C_types[item_type][field_name]
				// process the field as an item (just make a recursive call here)
				// function_code += fmt.Sprintf("%s// struct field item_type=%s item_prefix=%s member_op=%s item_name=%s field_C_type=%s field_name=%s\n",
				// line_prefix, item_type, item_prefix, member_op, item_name, field_C_type, field_name)
				// The field_C_type here may be of any number of types, including a simple scalar (bool, int64, string,
				// etc.), an embedded list, a pointer to an item of some other type, or the C type of some other
				// structure.  In the latter case, the field_C_type will generally include a package_name component,
				// not just a struct_name component.  So any recursive invocations of process_item() must understand
				// that a lookup within struct_fields[] typically won't be possible, since the struct_name used as
				// the key in that map does not include a package_name prefix.  Furthermore, it's possible that the
				// field_C_type might refer to a struct from some other package, not the one we are converting here.
				process_item(line_prefix+indent, field_C_type, item_prefix+member_op+item_name, field_member_op, field_name)
			}
			function_code += fmt.Sprintf("%s}\n", line_prefix)
		} else if _, ok := enum_typedefs[item_type]; ok {
			// There's nothing to do in this case for de-allocation of an embedded scalar field,
			// except to list the field in the output as having been processed.
			function_code += fmt.Sprintf("%s// enumeration field:  %s %s_%s%s%s\n", line_prefix, item_type, package_name, item_prefix, member_op, item_name)
		} else if typedef_type, ok := simple_typedefs[item_type]; ok {
			// This is probably some sort of typedef beyond the usual simplistic kind that just renames a basic type.
			// For instance, it might result from "type Structs []*Struct" in the Go code,
			// which translates to "typedef pkg_Struct_Ptr_List pkg_Structs;" in the C code.
			function_code += fmt.Sprintf("%s// typedef'd field:  %s %s_%s%s%s (typedef type %s)\n",
				line_prefix, item_type, package_name, item_prefix, member_op, item_name, typedef_type)
			if matches := leading_package.FindStringSubmatch(typedef_type); matches != nil {
				base_type := matches[1]
				function_code += fmt.Sprintf("%s// BASE TYPE IS %s\n", line_prefix, base_type)
				process_item(line_prefix, base_type, item_prefix, member_op, item_name)
			} else {
				var address_op string
				if item_name == "" {
					member_op = "" // we expect (member_op == "->") in this case, and override it
					address_op = ""
				} else {
					address_op = "&"
				}
				// We have a structure from some other package.  We must call its own destroy...() routine,
				// and deal correctly with both its json argument (which we must pass) and deleting the
				// pointers we pass (which we must cause it to skip).
				function_code += fmt.Sprintf("%sdestroy_%s_ptr_tree(%s%s_%s%s%s, json, false);\n",
					line_prefix, typedef_type, address_op, package_name, item_prefix, member_op, item_name)
			}
		} else if item_type == "bool" {
			// There's nothing to do in this case for de-allocation of an embedded scalar field,
			// except to list the field in the output as having been processed.
			function_code += fmt.Sprintf("%s// scalar field:  %s %s_%s%s%s\n", line_prefix, item_type, package_name, item_prefix, member_op, item_name)
		} else if item_type == "int" {
			// There's nothing to do in this case for de-allocation of an embedded scalar field,
			// except to list the field in the output as having been processed.
			function_code += fmt.Sprintf("%s// scalar field:  %s %s_%s%s%s\n", line_prefix, item_type, package_name, item_prefix, member_op, item_name)
		} else if item_type == "int32" {
			// There's nothing to do in this case for de-allocation of an embedded scalar field,
			// except to list the field in the output as having been processed.
			function_code += fmt.Sprintf("%s// scalar field:  %s %s_%s%s%s\n", line_prefix, item_type, package_name, item_prefix, member_op, item_name)
		} else if item_type == "int64" {
			// There's nothing to do in this case for de-allocation of an embedded scalar field,
			// except to list the field in the output as having been processed.
			function_code += fmt.Sprintf("%s// scalar field:  %s %s_%s%s%s\n", line_prefix, item_type, package_name, item_prefix, member_op, item_name)
		} else if item_type == "float64" {
			// There's nothing to do in this case for de-allocation of an embedded scalar field,
			// except to list the field in the output as having been processed.
			function_code += fmt.Sprintf("%s// scalar field:  %s %s_%s%s%s\n", line_prefix, item_type, package_name, item_prefix, member_op, item_name)
		} else if item_type == "struct_timespec" {
			// There's nothing to do in this case for de-allocation of an embedded structure that itself contains
			// no deallocatable fields, except to list the field in the output as having been processed.
			function_code += fmt.Sprintf("%s// embedded structure field:  %s %s_%s%s%s\n",
				line_prefix, item_type, package_name, item_prefix, member_op, item_name)
		} else if item_type == "string" {
			// We don't bother checking for a NULL pointer, because modern free() will tolerate that
			// (i.e., do its own check for that, which we need not duplicate).
			// function_code += fmt.Sprintf("%s// string item_type=%s item_prefix=%s member_op=%s item_name=%s\n",
			// line_prefix, item_type, item_prefix, member_op, item_name)
			function_code += fmt.Sprintf("%s// string field:  %s %s_%s%s%s\n", line_prefix, item_type, package_name, item_prefix, member_op, item_name)
			function_code += fmt.Sprintf("%sif (!json) free(%s%s%s);\n", line_prefix, package_name+"_"+item_prefix, member_op, item_name)
		} else {
			// Most likely, this is a subsidiary embedded object, which may or may not be from the same package.
			// function_code += fmt.Sprintf("%s// object field:  item_type=%s item_prefix=%s member_op=%s item_name=%s\n",
			// line_prefix, item_type, item_prefix, member_op, item_name)
			function_code += fmt.Sprintf("%s// object field:  %s %s_%s%s%s\n", line_prefix, item_type, package_name, item_prefix, member_op, item_name)
			if matches := leading_package.FindStringSubmatch(item_type); matches != nil {
				base_type := matches[1]
				function_code += fmt.Sprintf("%s// BASE TYPE IS %s\n", line_prefix, base_type)
				process_item(line_prefix, base_type, item_prefix, member_op, item_name)
			} else {
				var address_op string
				if item_name == "" {
					member_op = "" // we expect (member_op == "->") in this case, and override it
					address_op = ""
				} else {
					address_op = "&"
				}
				// We have a structure from some other package.  We must call its own destroy...() routine,
				// and deal correctly with both its json argument (which we must pass) and deleting the
				// pointers we pass (which we must cause it to skip).
				function_code += fmt.Sprintf("%sdestroy_%s_ptr_tree(%s%s_%s%s%s, json, false);\n",
					line_prefix, item_type, address_op, package_name, item_prefix, member_op, item_name)
			}
		}
	}
	process_item(indent, struct_name, struct_name+"_ptr", "->", "")
	function_code += fmt.Sprintf("    if (free_pointers) {\n        free(%s_ptr);\n", package_name+"_"+struct_name)

	var footer_code bytes.Buffer
	if err := footer_template.Execute(&footer_code, boilerplate_variables); err != nil {
		panic("destroy routine footer processing failed")
	}
	function_code += footer_code.String()

	return function_code, err
}

// FIX MINOR:  I foresee application code wanting to have top-level recursive-destroy routines for each
// of the supported structures, so if the programmer has been careful to not have any cross-sharing of
// allocated objects, an entire tree of prior allocations can be deallocated in one call.  That is
// distinct from the kind of deallocation used when freeing memory returned by some varient of the
// decode_json_StructTypeName() routine (now replaced by JSON_str_as_StructTypeName()_ptr routines),
// which will just free a single block of memory that we know has embedded within it in contiguous
// memory, the top-level data structure and all of its possibly multi-generational offspring.  The
// recursive-destroy routines are what I am calling above:
//
//     extern void destroy_PackageName_StructTypeName_ptr_tree(PackageName_StructTypeName *PackageName_StructTypeName_ptr, json_t *json, bool free_pointers);
//
func print_type_conversions(
	other_headers string,
	generated_C_code string,
	package_name string,
	final_type_order []declaration_kind,
	pointer_base_types map[string]string,
	pointer_list_base_types []string,
	simple_list_base_types []string,
	list_base_types []string,
	key_value_pair_types map[string][]string,
	simple_typedefs map[string]string,
	enum_typedefs map[string]string,
	const_groups map[string]string,
	struct_typedefs map[string][]string,
	simple_typedef_nodes map[string]*ast.GenDecl,
	enum_typedef_nodes map[string]*ast.GenDecl,
	const_group_nodes map[string]*ast.GenDecl,
	struct_typedef_nodes map[string]*ast.GenDecl,
	struct_fields map[string][]string,
	struct_field_Go_types map[string]map[string]string,
	struct_field_C_types map[string]map[string]string,
	struct_field_tags map[string]map[string]string,
) error {

	header_boilerplate := template.Must(template.New("code_file_header").Parse(C_code_boilerplate))

	type C_code_boilerplate_fields struct {
		Year           int
		CodeFilename   string
		OtherHeaders   string
		HeaderFilename string
	}

	current_year := time.Now().Year()
	code_filename := package_name + ".c"
	code_filepath := filepath.Join(output_directory, code_filename)
	header_filename := package_name + ".h"
	boilerplate_variables := C_code_boilerplate_fields{
		Year:           current_year,
		CodeFilename:   code_filename,
		OtherHeaders:   other_headers,
		HeaderFilename: header_filename,
	}

	code_file, err := os.Create(code_filepath)
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := code_file.Close(); err != nil {
			panic(err)
		}
	}()

	if err := header_boilerplate.Execute(code_file, boilerplate_variables); err != nil {
		panic("C code-file header processing failed")
	}

	fmt.Fprintf(code_file, "%s", generated_C_code)

	all_encode_function_code, err := generate_all_encode_tree_routines(
		package_name, final_type_order, pointer_base_types, list_base_types, key_value_pair_types,
		enum_typedefs, struct_fields, struct_field_Go_types, struct_field_C_types, struct_field_tags,
	)
	if err != nil {
		panic(err)
	}
	fmt.Fprintf(code_file, "%s", all_encode_function_code)

	all_decode_function_code, err := generate_all_decode_tree_routines(
		package_name, final_type_order,
		pointer_base_types, pointer_list_base_types, simple_list_base_types, key_value_pair_types,
		enum_typedefs, struct_fields, struct_field_Go_types, struct_field_C_types, struct_field_tags,
	)
	if err != nil {
		panic(err)
	}
	fmt.Fprintf(code_file, "%s", all_decode_function_code)

	all_destroy_function_code, err := generate_all_destroy_tree_routines(
		package_name, final_type_order, key_value_pair_types, simple_typedefs, enum_typedefs, struct_fields, struct_field_C_types,
	)
	if err != nil {
		panic(err)
	}
	fmt.Fprintf(code_file, "%s", all_destroy_function_code)

	return nil
}
