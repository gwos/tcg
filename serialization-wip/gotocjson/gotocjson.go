// gotocjson is a source-code converter.  It takes a Go source file containing a
// bunch of enumeration and structure declarations, and turns them into equivalent
// C code, including not just declarations but also JSON encoding and decoding
// routines that respect the field tags specified in the Go code.  This mechanism
// will both allow for rapid automated changes on the C side whenever we need to
// revise the Go interface, and ensure that the conversion rouines are up-to-date
// and accurate.
package main

// FIX MAJOR:
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
    "reflect"
    "regexp"
    "runtime"
    "sort"
    "strconv"
    "strings"
    "text/template"
    "time"
    // "go/printer"
    // "go/scanner"
    // "io/ioutil"
)

var debug = true

func file_line() string {
    var s string
    if _, file_path, line_number, ok := runtime.Caller(1); ok {
	// We get back the full absolute path for the file_path.
	// That's much more than we need, so we extract the file
	// basename and use that instead.
	path_components := strings.Split(file_path, "/")
	base_name := path_components[len(path_components) - 1]
        s = fmt.Sprintf("%s:%d", base_name, line_number)
    } else {
        s = ""
    }
    return s
}

// All operations in this program assume that the source code under
// inspection fits easily into memory all at once; there is no need
// for any type of streaming in the handling of the source code.

func main() {
    if len(os.Args) != 2 || os.Args[1] == "-h" {
	print_help()
	// Go ought to have a ternary operator, but doesn't.  Sigh.
	if len(os.Args) <= 1 || os.Args[1] == "-h" { os.Exit(0) } else { os.Exit(1) }
    }
    // scan_file(os.Args[1])
    fset, f, err := parse_file(os.Args[1])
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
    struct_fields, struct_field_C_types, struct_field_tags, generated_C_code, err := print_type_declarations(
	final_type_order,
	package_name,
	simple_typedefs, enum_typedefs, const_groups, struct_typedefs, struct_field_typedefs,
	simple_typedef_nodes, enum_typedef_nodes, const_group_nodes, struct_typedef_nodes,
    )
    if err != nil {
	os.Exit(1)
    }
    err = print_type_conversions(
	other_headers,
	generated_C_code,
	final_type_order,
	package_name,
	simple_typedefs, enum_typedefs, const_groups, struct_typedefs, struct_field_typedefs,
	simple_typedef_nodes, enum_typedef_nodes, const_group_nodes, struct_typedef_nodes,
	struct_fields, struct_field_C_types, struct_field_tags,
    )
    if err != nil {
	os.Exit(1)
    }

    os.Exit(0)
}

func print_help() {
    fmt.Fprintf(os.Stderr,
`usage:  gotocjson filename.go
        gotocjson -h
where:  filename.go  is the source-code file you wish to transform into C code
        -h           prints this usage message
`)
}

/*
func scan_file(filepath string) ([]byte, error) {
    code, err := ioutil.ReadFile(filepath)
    if err != nil {
        return nil, err
    }
    var s scanner.Scanner
    fset := token.NewFileSet()                       // positions are relative to fset
    file := fset.AddFile("", fset.Base(), len(code)) // register input "file"
    // "nil" as the third argument means we are not providing an error handler
    s.Init(file, code, nil, scanner.ScanComments)

    // Repeated calls to Scan yield the token sequence found in the input.
    // var tokens []token.Token
    for {
	pos, tok, lit := s.Scan()
	if tok == token.EOF {
	    break
	}
	fmt.Printf("%s\t%s\t%q\n", fset.Position(pos), tok, lit)
    }
    return nil, nil
}
*/

// Routine to parse the file.
func parse_file(filepath string) (*token.FileSet, *ast.File, error) {
    fset := token.NewFileSet() // positions are relative to fset
    // mode := parser.ParseComments | parser.Trace | parser.DeclarationErrors
    mode := parser.ParseComments | parser.DeclarationErrors

    // Parse the specified file.
    f, err := parser.ParseFile(fset, filepath, nil, mode)
    if err != nil {
	fmt.Printf("found Go-syntax parsing error in file %s: %s\n", filepath, err)
	return nil, nil, err
    }

    return fset, f, nil
}

// FIX MAJOR:  We could probably use a certain amount of refactoring, both to factor out similar
// code blocks and to allow for a certain degree of potential recursion in type declarations.
// FIX MAJOR:  make sure we test the following types separately:
//     "foo"
//     "*foo"
//     "[]foo"
//     "**foo"
//     "*[]foo"
//     "[]*foo"
//     "[][]foo"
//     "*[]*foo"
// FIX MAJOR:  Have this routine return separate slices for parse nodes representing simple_types,
// enum_types, const_groups, struct_types, and function_signatures, along with info on cross-references
// between those objects, so we can take some effort to run a topological sort on them and output the
// respective series of nodes in a clean order so the C compiler sees no forward references.
//
// We only track file-level (i.e., package-level) typedefs, consts, and (possibly ) function signatures.
// Anything which is at function level instead of at package level, we don't want to include in our
// output maps or ultimately in the generated C code.
//
// Here are the forms of the returned element-type maps:
//
//     simple_typedefs map[    typedef_name string] typedef_type string
//       enum_typedefs map[       enum_name string]    enum_type string
//        const_groups map[const_group_name string]constant_type string
//     struct_typedefs map[     struct_name string] []field_type string
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
	package_name          string,
	simple_typedefs       map[string]string,
	enum_typedefs         map[string]string,
	const_groups          map[string]string,
	struct_typedefs       map[string][]string,		// list of unique simplified types of the fields
	struct_field_typedefs map[string]map[string]string,	// indexed by struct name and field name, yields full field typedef
	simple_typedef_nodes  map[string]*ast.GenDecl,
	enum_typedef_nodes    map[string]*ast.GenDecl,
	const_group_nodes     map[string]*ast.GenDecl,
	struct_typedef_nodes  map[string]*ast.GenDecl,
	other_headers         string,
	err error,
    ) {
    // FIX MINOR
    // Having this function in play turns out to be somewhat less than completely desirable,
    // because the simple error message does not include all the failure-coordinate data that
    // would have been printed by allowing the panic to proceed without interception.
    defer func() {
	if false {
	    if exception := recover(); exception != nil {
		err = fmt.Errorf("internal error: %v", exception)
		fmt.Println(err)
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
    enum_typedefs   = map[string]string{}
    const_groups    = map[string]string{}
    struct_typedefs = map[string][]string{}

    simple_typedef_nodes = map[string]*ast.GenDecl{}
    enum_typedef_nodes   = map[string]*ast.GenDecl{}
    const_group_nodes    = map[string]*ast.GenDecl{}
    struct_typedef_nodes = map[string]*ast.GenDecl{}

    // Print the package name.
    package_name = f.Name.Name  // from the "package" declaration inside the file
    fmt.Println("=== Package:")
    fmt.Println(package_name)

    // Print the file's imports.
    fmt.Println("=== Imports:")
    special_package_prefix := regexp.MustCompile(`^github.com/gwos/tng/([^/]+)$`)
    include_headers := []string{}
    for _, s := range f.Imports {
	fmt.Println(s.Path.Value)
	pkg := strings.ReplaceAll(s.Path.Value, "\"", "")
	special_package := special_package_prefix.FindStringSubmatch(pkg)
	if special_package != nil {
	    include_headers = append(include_headers, fmt.Sprintf(`#include "%s.h"`, special_package[1]))
	}
    }
    other_headers = strings.Join(include_headers, "\n")

    // Print the file's documentation.
    // It only prints the leading package doc, not function comments.
    // For that, one needs to dig deeper (see below).
    // FIX MAJOR:  this is not stripping the leading "//" from comment lines
    fmt.Println("=== Package Documentation:")
    if f.Doc != nil {
	for _, doc := range f.Doc.List {
	    fmt.Println(doc.Text)
	}
    }

    fmt.Println("=== Declarations:")
    // Print the file-level declarations.  This conveniently ignores declarations within functions,
    // which we don't care about for our purposes.
    panic_message := ""
node_loop:
    for _, file_decl := range f.Decls {
	// fmt.Println(d)  // "&{<nil> <nil> parse_file 0xc000093660 0xc00007abd0}" and other forms
	if func_decl, ok := file_decl.(*ast.FuncDecl); ok {
	    fmt.Printf("--- function name:  %v\n", func_decl.Name.Name)
	    if func_decl.Doc != nil {
		fmt.Println("--- function documentation:")
		// FIX MAJOR:  this is not stripping the leading "//" from comment lines
		for _, doc := range func_decl.Doc.List {
		    fmt.Println(doc.Text)
		}
	    }
	}
	if gen_decl, ok := file_decl.(*ast.GenDecl); ok {
	    if gen_decl.Tok == token.TYPE {
		for _, spec := range gen_decl.Specs {
		    // I'm just assuming that spec.(*ast.TypeSpec).Type is of type *ast.Ident here in all cases.
		    // If that turns out not to be true, we'll have to fill in other cases.
		    if type_ident, ok := spec.(*ast.TypeSpec).Type.(*ast.Ident); ok {
			fmt.Printf("--- simple type declaration name and type:  %v %v\n", spec.(*ast.TypeSpec).Name.Name, type_ident.Name)
			simple_typedefs[spec.(*ast.TypeSpec).Name.Name] = type_ident.Name
			simple_typedef_nodes[spec.(*ast.TypeSpec).Name.Name] = gen_decl
		    } else if type_struct, ok := spec.(*ast.TypeSpec).Type.(*ast.StructType); ok {
			// fmt.Printf("--- struct type:  %#v\n", type_struct)
			fmt.Printf("--- struct type declaration name:  %v\n", spec.(*ast.TypeSpec).Name.Name)
			struct_typedefs[spec.(*ast.TypeSpec).Name.Name] = nil

			// FIX QUICK:  not yet sure if this is correct
			struct_field_typedefs[spec.(*ast.TypeSpec).Name.Name] = map[string]string{}

			// fiX QUICK:  drop the extra commented-out code here
			// struct_typedefs[spec.(*ast.TypeSpec).Name.Name] = []string{nil}
			// struct_typedefs[spec.(*ast.TypeSpec).Name.Name] = []string{}
			struct_typedef_nodes[spec.(*ast.TypeSpec).Name.Name] = gen_decl
			if type_struct.Incomplete {
			    // I'm not sure when this condition might be true, so let's alarm on it if we encounter it
			    // just to make sure we're not overlooking anything.
			    fmt.Printf("    --- The list of fields is incomplete.\n")
			    panic_message = "aborting due to previous errors"
			    break node_loop
			}
			for _, field := range type_struct.Fields.List {
			    // FIX MAJOR:  add support for the .Doc and .Comment attributes as well
			    // fmt.Printf("    --- field:  %#v\n", field)
			    // Field elements to process:
			    // .Doc   *ast.CommentGroup    // may be nil
			    // .Names []*ast.Ident
			    if field.Names == nil {
				// Here, we have an anonymous field, such as occurs with Go's structure embedding.
				// Since that won't do in C, we autovivify a field name from the field type, similar
				// to how that is done implicitly in Go itself but appending a small string to guarantee
				// that there will be no confusion in C between the field name and the type name.
				if type_ident, ok := field.Type.(*ast.Ident); ok {
				    // Old construction:  just accept that we have a missing field name.
				    // fmt.Printf("    --- struct field name and type:  %#v %#v\n", "(none)", type_ident.Name)
				    // New construction:  autovivify a sensible field name.
				    name_ident := new(ast.Ident)
				    // Testing shows I was wrong; modern C can handle having a variable or struct field named
				    // the same as a struct typedef.  So to keep things simple, we don't append an underscore
				    // to type_ident.Name here.
				    name_ident.Name = type_ident.Name
				    field.Names = append(field.Names, name_ident)
				} else if type_starexpr, ok := field.Type.(*ast.StarExpr); ok {
				    // fmt.Printf("    --- struct field name and type:  %#v %#v\n", "(none)", type_starexpr)
				    if type_ident, ok := type_starexpr.X.(*ast.Ident); ok {
					// fmt.Printf("    --- struct field name and StarExpr type:  %#v %#v\n", name.Name, type_ident.Name)
					name_ident := new(ast.Ident)
					name_ident.Name = type_ident.Name + "_ptr_"
					field.Names = append(field.Names, name_ident)
				    } else if type_selectorexpr, ok := type_starexpr.X.(*ast.SelectorExpr); ok {
					var x_type_ident *ast.Ident
					var ok bool
					if x_type_ident, ok = type_selectorexpr.X.(*ast.Ident); ok {
					    // fmt.Printf("    --- struct field name and SelectorExpr X:  %#v %#v\n", name.Name, x_type_ident.Name)
					    // fmt.Printf("    --- struct field SelectorExpr X:  %#v\n", x_type_ident.Name)
					} else {
					    fmt.Printf("ERROR:  when autovivifying at %s, found unexpected field.Type.X type:  %T\n",
						file_line, type_selectorexpr.X)
					    fmt.Printf("ERROR:  struct field Type.X field is not of a recognized type\n")
					    panic_message = "aborting due to previous errors"
					    break node_loop
					}
					if type_selectorexpr.Sel == nil {
					    fmt.Printf("ERROR:  when autovivifying at %s, struct field Type Sel field is unexpectedly nil\n", file_line())
					    panic_message = "aborting due to previous errors"
					    break node_loop
					}
					name_ident := new(ast.Ident)
					// We used to append an underscore in this construction of name_ident.Name, but we
					// are backing off from that until and unless we find it to actually be necessary.
					// (The backoff is not yet done, pending testing.)
					name_ident.Name = x_type_ident.Name + "_" + type_selectorexpr.Sel.Name + "_ptr_"
					// fmt.Printf("    ==> manufactured field name:  %s\n", name_ident.Name)
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
					// So once we figure out the field name we will manufature for type_starexpr.X, we will append "_ptr_" to that name.
					//
					fmt.Printf("ERROR:  when autovivifying at %s, found unexpected field.Type.X type:  %T\n",
					    file_line(), type_starexpr.X)
					fmt.Printf("ERROR:  struct field Type.X field is not of a recognized type\n")
					panic_message = "aborting due to previous errors"
					break node_loop
				    }
				} else if type_selectorexpr, ok := field.Type.(*ast.SelectorExpr); ok {
				    var x_type_ident *ast.Ident
				    var ok bool
				    if x_type_ident, ok = type_selectorexpr.X.(*ast.Ident); ok {
					// fmt.Printf("    --- struct field name and SelectorExpr X:  %#v %#v\n", name.Name, x_type_ident.Name)
				    } else {
					fmt.Printf("ERROR:  when autovivifying at %s, found unexpected field.Type.X type:  %T\n",
					    file_line, type_selectorexpr.X)
					fmt.Printf("ERROR:  struct field Type.X field is not of a recognized type\n")
					panic_message = "aborting due to previous errors"
					break node_loop
				    }
				    if type_selectorexpr.Sel == nil {
					fmt.Printf("ERROR:  when autovivifying at %s, struct field Type Sel field is unexpectedly nil\n", file_line())
					panic_message = "aborting due to previous errors"
					break node_loop
				    }
				    name_ident := new(ast.Ident)
				    // We used to append an underscore in this construction of name_ident.Name, but we
				    // are backing off from that until and unless we find it to actually be necessary.
				    // (The backoff is not yet done, pending testing.)
				    name_ident.Name = x_type_ident.Name + "_" + type_selectorexpr.Sel.Name + "_"
				    field.Names = append(field.Names, name_ident)
				} else {
				    fmt.Printf("ERROR:  when autovivifying at %s, found unexpected field.Type type:  %T\n", file_line(), field.Type)
				    fmt.Printf("ERROR:  struct field Type field is not of a recognized type\n")
				    panic_message = "aborting due to previous errors"
				    break node_loop
				}
			    }
			    for _, name := range field.Names {
				// fmt.Printf("    --- field name:  %#v\n", name)
				var field_type_name string
				if type_ident, ok := field.Type.(*ast.Ident); ok {
				    field_type_name = type_ident.Name
				    fmt.Printf("    --- struct field name and type:  %#v %#v\n", name.Name, field_type_name)
				} else if type_starexpr, ok := field.Type.(*ast.StarExpr); ok {
				    if type_ident, ok := type_starexpr.X.(*ast.Ident); ok {
					field_type_name = "*" + type_ident.Name
					fmt.Printf("    --- struct field name and StarExpr type:  %#v %#v\n", name.Name, field_type_name)
				    } else if type_array, ok := type_starexpr.X.(*ast.ArrayType); ok {
					var array_type_ident *ast.Ident
					// A nil type_array.Len means it's a slice type.
					if type_array.Len != nil {
					    fmt.Printf("ERROR:  at %s, a non-nil value for a StarExpr array-type Len is not yet handled (%#v)\n",
						file_line(), type_array.Len)
					    panic_message = "aborting due to previous errors"
					    break node_loop
					}
					if array_type_ident, ok = type_array.Elt.(*ast.Ident); ok {
					    // fmt.Printf("    --- struct field Type X Elt array element ident %#v\n", array_type_ident)
					} else {
					    fmt.Printf("ERROR:  at %s, found unexpected field.Type.X.Elt type:  %T\n", file_line(), type_array.Elt)
					    fmt.Printf("ERROR:  struct field Type X Elt field is not of a recognized type\n")
					    panic_message = "aborting due to previous errors"
					    break node_loop
					}
					field_type_name = "*[]" + array_type_ident.Name
					fmt.Printf("    --- struct field name and type:  %#v %#v\n", name.Name, field_type_name)
				    } else if type_selectorexpr, ok := type_starexpr.X.(*ast.SelectorExpr); ok {
					var x_type_ident *ast.Ident
					var ok bool
					if x_type_ident, ok = type_selectorexpr.X.(*ast.Ident); ok {
					    // fmt.Printf("    --- struct field name and SelectorExpr X:  %#v %#v\n", name.Name, x_type_ident.Name)
					} else {
					    fmt.Printf("ERROR:  at %s, found unexpected field.Type.X type:  %T\n", file_line(), type_selectorexpr.X)
					    fmt.Printf("ERROR:  struct field Type.X field is not of a recognized type\n")
					    panic_message = "aborting due to previous errors"
					    break node_loop
					}
					if type_selectorexpr.Sel == nil {
					    fmt.Printf("ERROR:  at %s, struct field Type Sel field is unexpectedly nil\n", file_line())
					    panic_message = "aborting due to previous errors"
					    break node_loop
					}
					// FIX QUICK:  this may need work to fully and correctly reflect the complete selector
					field_type_name = "*" + x_type_ident.Name + "." + type_selectorexpr.Sel.Name
					fmt.Printf("    --- struct field name and type:  %#v *%v.%v\n", name.Name, x_type_ident.Name, field_type_name)
				    } else {
					fmt.Printf("ERROR:  at %s, found unexpected field.Type.X type:  %T\n", file_line(), type_starexpr.X)
					fmt.Printf("ERROR:  struct field Type.X field is not of a recognized type\n")
					panic_message = "aborting due to previous errors"
					break node_loop
				    }
				} else if type_array, ok := field.Type.(*ast.ArrayType); ok {
				    // A nil type_array.Len means it's a slice type.
				    if type_array.Len != nil {
					fmt.Printf("ERROR:  at %s, a non-nil value for an array-type Len is not yet handled (%#v)\n",
					    file_line(), type_array.Len)
					panic_message = "aborting due to previous errors"
					break node_loop
				    }
				    if type_ident, ok := type_array.Elt.(*ast.Ident); ok {
				        // fmt.Printf("    --- array element ident %#v\n", type_ident)
					field_type_name = "[]" + type_ident.Name
					fmt.Printf("    --- struct field name and type:  %#v %#v\n", name.Name, field_type_name)
				    } else if type_starexpr, ok := type_array.Elt.(*ast.StarExpr); ok {
				        // fmt.Printf("    --- array element starexpr %#v\n", type_starexpr)
					if type_ident, ok := type_starexpr.X.(*ast.Ident); ok {
					    field_type_name = "[]*" + type_ident.Name
					    fmt.Printf("    --- struct field name and interior StarExpr type:  %#v %#v\n", name.Name, field_type_name)
					} else if type_array, ok := type_starexpr.X.(*ast.ArrayType); ok {
					    fmt.Printf("    --- UNEXPECTED interior field.Type.X Type *ast.ArrayType %#v\n", type_array)
					    // FIX MAJOR:  handle this case
					} else {
					    fmt.Printf("ERROR:  at %s, found unexpected interior field.Type.X type:  %T\n", file_line(), type_starexpr.X)
					    fmt.Printf("ERROR:  struct field interior Type.X field is not of a recognized type\n")
					    panic_message = "aborting due to previous errors"
					    break node_loop
					}
				    } else {
					fmt.Printf("ERROR:  at %s, found unexpected field.Type.Elt type:  %T\n", file_line(), type_array.Elt)
					fmt.Printf("ERROR:  struct field Type Elt field is not of a recognized type\n")
					panic_message = "aborting due to previous errors"
					break node_loop
				    }
				} else if type_map, ok := field.Type.(*ast.MapType); ok {
				    var key_type_ident *ast.Ident
				    var value_type_ident *ast.Ident
				    var ok bool
				    if key_type_ident, ok = type_map.Key.(*ast.Ident); ok {
				        // fmt.Printf("    --- map Key Ident %#v\n", key_type_ident)
				    } else {
					fmt.Printf("ERROR:  at %s, found unexpected field.Type.Key type:  %T\n", file_line(), type_map.Key)
					fmt.Printf("ERROR:  struct field Type Key field is not of a recognized type\n")
					panic_message = "aborting due to previous errors"
					break node_loop
				    }
				    if value_type_ident, ok = type_map.Value.(*ast.Ident); ok {
				        // fmt.Printf("    --- map Value Ident %#v\n", value_type_ident)
				    } else {
					fmt.Printf("ERROR:  at %s, found unexpected field.Type.Value type:  %T\n", file_line(), type_map.Value)
					fmt.Printf("ERROR:  struct field Type Value field is not of a recognized type\n")
					panic_message = "aborting due to previous errors"
					break node_loop
				    }
				    // FIX QUICK:  this needs work to fully reflect the map structure; perhaps the new statements now do so
				    // field_type_name = value_type_ident.Name
				    // fmt.Printf("    --- struct field name and type:  %#v map[%#v]%#v\n", name.Name, key_type_ident.Name, field_type_name)
				    field_type_name = "map[" + key_type_ident.Name + "]" + value_type_ident.Name
				    fmt.Printf("    --- struct field name and type:  %#v %#v\n", name.Name, field_type_name)
				} else if type_selectorexpr, ok := field.Type.(*ast.SelectorExpr); ok {
				    var x_type_ident *ast.Ident
				    var ok bool
				    if x_type_ident, ok = type_selectorexpr.X.(*ast.Ident); ok {
					// fmt.Printf("    --- struct field name and SelectorExpr X:  %#v %#v\n", name.Name, x_type_ident.Name)
				    } else {
					fmt.Printf("ERROR:  at %s, found unexpected field.Type.X type:  %T\n", file_line(), type_selectorexpr.X)
					fmt.Printf("ERROR:  struct field Type.X field is not of a recognized type\n")
					panic_message = "aborting due to previous errors"
					break node_loop
				    }
				    if type_selectorexpr.Sel == nil {
					fmt.Printf("ERROR:  at %s, struct field Type Sel field is unexpectedly nil\n", file_line())
					panic_message = "aborting due to previous errors"
					break node_loop
				    }
				    // FIX QUICK:  this may need work to fully and correctly reflect the complete selector
				    field_type_name = x_type_ident.Name + "." + type_selectorexpr.Sel.Name
				    fmt.Printf("    --- struct field name and type:  %#v %v.%v\n", name.Name, x_type_ident.Name, field_type_name)
				} else {
				    fmt.Printf("ERROR:  at %s, found unexpected field.Type type:  %T\n", file_line(), field.Type)
				    fmt.Printf("ERROR:  struct field Type field is not of a recognized type\n")
				    panic_message = "aborting due to previous errors"
				    break node_loop
				}
				struct_typedefs[spec.(*ast.TypeSpec).Name.Name] = append(struct_typedefs[spec.(*ast.TypeSpec).Name.Name], field_type_name)
				struct_field_typedefs[spec.(*ast.TypeSpec).Name.Name][name.Name] = field_type_name
				if field.Tag != nil {
				    fmt.Printf("    --- struct field tag Value:  %#v\n", field.Tag.Value)
				}
			    }
			    // .Type  *ast.Ident
			    // .Tag   *ast.BasicLit        // may be nil
			    // .Comment *ast.CommentGroup  // likely nil
			}
		    } else if type_interface, ok := spec.(*ast.TypeSpec).Type.(*ast.InterfaceType); ok {
			fmt.Printf("FIX MAJOR:  handle this next case (where the type is *ast.InterfaceType)\n")
			// This is an interface definition, which perhaps mostly declares methods, not simple types,
			// enumerations, constants, or structs.  Verify that assumption, and perhaps extend this case
			// to process whatever it might need to.  We might, for instance, at least need to emit function
			// signatures, even if we don't generate full function bodies.
			fmt.Printf("--- interface type declaration name and type:  %v %#v\n", spec.(*ast.TypeSpec).Name.Name, type_interface)
		    } else {
			fmt.Printf("ERROR:  at %s, found unexpected spec.(*ast.TypeSpec).Type type:  %T\n", file_line(), spec.(*ast.TypeSpec).Type)
		        fmt.Printf("ERROR:  spec *ast.TypeSpec Type field is not of a recognized type\n")
			panic_message = "aborting due to previous errors"
			break node_loop
		    }
		}
	    }
	    if gen_decl.Tok == token.CONST {
		// FIX MAJOR:  this needs some testing to see when iota_value and value_is_from_iota need to be set or reset
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
			    fmt.Printf("ERROR:  at %s, found unexpected spec.(*ast.ValueSpec).Type type:  %T\n", file_line(), spec.(*ast.ValueSpec).Type)
			    fmt.Printf("ERROR:  spec *ast.ValueSpec Type field is not of a recognized type\n")
			    panic_message = "aborting due to previous errors"
			    break node_loop
			}
			// value_type := spec.(*ast.ValueSpec).Type
			// fmt.Printf("value_type = %T %[1]v %+[1]v %#[1]v %[1]s\n", value_type)
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
					fmt.Printf("ERROR:  at %s, value name is %#v\n", file_line(), spec.(*ast.ValueSpec).Values[i].(*ast.Ident).Name)
					panic_message = "unexpected const value name"
					break node_loop
				    }
				    if spec.(*ast.ValueSpec).Values[i].(*ast.Ident).Obj != nil {
					fmt.Printf("ERROR:  at %s, value object kind is %#v\n",
					    file_line(), spec.(*ast.ValueSpec).Values[i].(*ast.Ident).Obj.Kind)
					fmt.Printf("ERROR:  at %s, value object name is %#v\n",
					    file_line(), spec.(*ast.ValueSpec).Values[i].(*ast.Ident).Obj.Name)
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
				    fmt.Printf("ERROR:  at %s, value expression is %#v\n", file_line(), spec.(*ast.ValueSpec).Values[i].(*ast.BinaryExpr))
				    // FIX MAJOR:  this setting of spec_type is nowhere near a thorough analysis
				    if spec_type == "" {
					spec_type = "int"
				    }
				    r, err := eval_int_expr(spec.(*ast.ValueSpec).Values[i].(*ast.BinaryExpr), &iota_value)
				    if err != nil {
					fmt.Println(err)
					panic_message = "cannot evaluate binary expression"
					break node_loop
				    }
				    const_value = fmt.Sprintf("%d", r)
				default:
				    fmt.Printf("ERROR:  at %s, found const value type %#v\n", file_line(), spec.(*ast.ValueSpec).Values[i])
				    panic_message = "unexpected const value type"
				    break node_loop
			    }
			} else if value_is_from_iota {
			    iota_value++
			    const_value = fmt.Sprintf("%d", iota_value)
			}
			// FIX MAJOR:  this is not yet showing the "int" spec_type for a "1 << iota" expression
			fmt.Printf("--- const element name, type, and value:  %v %v %v\n", name.Name, spec_type, const_value)
			// It's not required by Go syntax that every assignment in a single const block has exactly
			// the same type, but we insist on that here to simplify our work.  If we encounter code that
			// violates this constraint, the code in this conversion tool will need to be extended.
			const_token_position := fset.Position(gen_decl.TokPos).String()
			if const_groups[const_token_position] == "" {
			    const_groups[const_token_position] = spec_type
			    const_group_nodes[const_token_position] = gen_decl
			} else if const_groups[const_token_position] != spec_type {
			    fmt.Printf("ERROR:  at %s, found conflicting const types in a single const block:  %s %s\n",
				file_line(), const_groups[const_token_position], spec_type)
			    panic_message = "found conflicting const types in a single const block"
			    break node_loop
			}
		    }
		}
	    }
	}
    }

    fmt.Println("=== AST:")
    ast.Print(fset, f)

    if panic_message != "" {
	fmt.Printf("%s\n", panic_message)
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
		// FIX QUICK
		// break apart a map type into its separate key and value types
		// FIX LATER:  Obviously, in the general case, both the key and value types might be more
		// complex than we are allowing for here (just individual base types, not involving slices,
		// pointers, further maps, or perhaps other exotic fauna).  If we run into trouble when
		// converting such code, the analysis here will need extension.
		key_value_types := map_key_value_types.FindStringSubmatch(field_type)
		if key_value_types == nil {
		    panic(fmt.Sprintf("found incomprehensible map construction '%s'", field_type))
		}
		key_type   := key_value_types[1]
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

// FIX MAJOR:  add support for iota to the evaluation
// FIX MAJOR:  this is not yet coordinated with the ast.Ident processing above
func eval_int_expr(tree ast.Expr, iota *int) (int, error) {
    switch n := tree.(type) {
	case *ast.Ident:
	    // FIX MAJOR:  do the right thing here to prove we should really be accessing iota
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
		case token.ADD:     return x + y, nil
		case token.SUB:     return x - y, nil
		case token.MUL:     return x * y, nil
		case token.QUO:     return x / y, nil
		case token.REM:     return x % y, nil
		case token.AND:     return x & y, nil
		case token.OR:      return x | y, nil
		case token.XOR:     return x ^ y, nil
		case token.SHL:     return x << y, nil
		case token.SHR:     return x >> y, nil
		case token.AND_NOT: return x &^ y, nil
		default:            return unsupported(n.Op)
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
	simple_typedefs      map[string]string,
	enum_typedefs        map[string]string,
	const_groups         map[string]string,
	struct_typedefs      map[string][]string,
	simple_typedef_nodes map[string]*ast.GenDecl,
	enum_typedef_nodes   map[string]*ast.GenDecl,
	const_group_nodes    map[string]*ast.GenDecl,
	struct_typedef_nodes map[string]*ast.GenDecl,
    ) (
	final_type_order []declaration_kind,
	err error,
    ) {

    type type_dependency struct {
	type_kind string
	type_pos token.Pos
	depends_on_type_name []string
    }

    // map[type_name]type_dependency
    dependency := map[string]type_dependency{}

    // Output at this stage is only for initial development, to ensure that we have the expected
    // kinds of data at this point.
    for typedef_name, typedef_type := range simple_typedefs {
	if debug {
	    fmt.Printf("simple typedef:  %s => %s\n", typedef_name, typedef_type)
	}
	dependency[typedef_name] = type_dependency{"simple", simple_typedef_nodes[typedef_name].TokPos, []string{typedef_type}}
    }
    for enum_name, enum_type := range enum_typedefs {
	if debug {
	    fmt.Printf("enum typedef:  %s => %s\n", enum_name, enum_type)
	}
	dependency[enum_name] = type_dependency{"enum", enum_typedef_nodes[enum_name].TokPos, []string{enum_type}}
    }
    for const_group_name, const_group_type := range const_groups {
	if debug {
	    fmt.Printf("const group:  %s => %s\n", const_group_name, const_group_type)
	}
	// Here, the TokPos value we provide is just a placeholder.  It does represent the position of the
	// original const group in the source code, but if this const block represents an set of enumeration
	// values, we will later replace that with the position of the enumeration type.  That will force
	// emission of the enumeration values immediately after emission of the enumeration declaration.
	dependency[const_group_name] = type_dependency{"const", const_group_nodes[const_group_name].TokPos, []string{const_group_type}}
    }
    for struct_name, struct_field_type_list := range struct_typedefs {
	if debug {
	    fmt.Printf("struct typedef:  %s => %v\n", struct_name, struct_field_type_list)
	}
	dependency[struct_name] = type_dependency{"struct", struct_typedef_nodes[struct_name].TokPos, struct_field_type_list}
    }

    tentative_type_order := make([]string, 0, len(dependency))
    // type_dep here (or at least its type_dep.depends_on_type_name component) apparently ends up
    // as a copy of the type_dependency object (or at least its []string component), not an alias.
    // So when we wish to alter the base data structure, we must refer to it directly.
    for type_name, type_dep := range dependency{
	// fmt.Printf("=== dep types before filtering: %v\n", type_dep.depends_on_type_name)
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
	sort.Slice(type_dep.depends_on_type_name, func (i int, j int) bool {
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

	// fmt.Printf("   base typedef:  %#v %#v\n", type_name, dependency[type_name])
	// fmt.Printf("generic typedef:  %#v %#v\n", type_name, type_dep)

	tentative_type_order = append(tentative_type_order, type_name)
    }

    // Finally, we create a []string array derived from "dependency" which contains all of its keys
    // but has them all sorted by increasing values of the respective dependency[_].type_pos fields.
    // This is the order in which we will process the type names in our topological sort loop.
    sort.Slice(tentative_type_order, func (i int, j int) bool {
	return dependency[tentative_type_order[i]].type_pos < dependency[tentative_type_order[j]].type_pos
    })

    if debug {
	for _, type_name := range tentative_type_order {
	    fmt.Printf("sorted generic typedef:  %#v %#v\n", type_name, dependency[type_name])
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

    if debug {
	for _, decl_kind := range final_type_order {
	    fmt.Printf("final sorted generic typedef:  %#v %#v %#v\n", decl_kind.type_name, decl_kind.type_kind, dependency[decl_kind.type_name])
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
#include <time.h>	// to supply "struct timespec", with time_t tv_sec (seconds) and long tv_nsec (nanoseconds) members

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

// FIX MAJOR:  this is just for initial development, to allow the generated code
// to compile until we are able to handle *ast.SelectorExpr struct fields
#define FIX_MAJOR_dummy_typename string

#ifndef string
// Make a simple global substitution using the C preprocessor, so we don't
// have to deal with this ad infinitum in the language-conversion code.
#define string char *
#endif  // string

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
//     extern bool free_PackageName_StructTypeName_tree(PackageName_StructTypeName *StructTypeName_ptr);
//
// That one call will at the same time free memory for all of the connected
// subsidary objects.
//
// Note that a similar routine:
//
//     extern bool destroy_PackageName_StructTypeName_tree(PackageName_StructTypeName *PackageName_StructTypeName_ptr);
//
// is also available.  It has a very similar purpose, but it is intended for use
// with a tree of data structures which are manually allocated in application code,
// where the individual parts are likely not contiguous in memory.  In contrast, the
// free_StructTypeName_tree() implementation will be kept definitively matched to the
// decode_json_PackageName_StructTypeName() implementation.  So whether the decoding
// creates just a single large block of memory that contains not only the initial
// StructTypeName object but all of the subsidiary objects it recursively refers to,
// or whether it splays things out via independently floating allocations, a call to
// free_StructTypeName_tree() is guaranteed to match the internal requirements of
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
// FIX MAJOR:  This inclusion might be better moved to the header file,
// if json_dumps() is expected to be used by application code.
#include <jansson.h>

#include <stdlib.h>    // for the declaration of free(), at least
// #include <stdalign.h>  // Needed to supply alignof(), available starting with C11.
// #include <stddef.h>
// #include <string.h>

{{.OtherHeaders}}
#include "{{.HeaderFilename}}"

#define stringify(x)                    #x
#define expand_and_stringify(x)         stringify(x)

// FILE_LINE is defined so you can just say:
// log_message (APP_FATAL, FILE_LINE "Insufficient memory for %s; exiting!", foobar);
// (Notice the lack of a comma after the FILE_LINE invocation.)

#define FILE_LINE       __FILE__ "[" expand_and_stringify(__LINE__) "] "

#define arraysize(array) (sizeof(array) / sizeof(array[0]))

// If this routine fails to find any matching string within the array, it returns a negative result. 
// It doesn't log anything in that situation, both because the caller is going to need to check the
// result anyway for such an out-of-bound result, and because the calling code has a much better
// idea of the full context of what ought to be included in a log message.
int enumeration_value(const string const enum_string[], int enum_string_count, const char *enum_value_as_string) {
    int enum_value;
    for (enum_value = enum_string_count; --enum_value >= 0; ) {
        if (!strcmp(enum_value_as_string, enum_string[enum_value])) {
            break;
        }
    }
    return enum_value;
}

// FIX MAJOR:  Also include in here some initialization of the conversion library,
// so we can pass our logger to the package and have it use that for all error logging.

`

func print_type_declarations(
	final_type_order      []declaration_kind,
	package_name          string,
	simple_typedefs       map[string]string,
	enum_typedefs         map[string]string,
	const_groups          map[string]string,
	struct_typedefs       map[string][]string,
	struct_field_typedefs map[string]map[string]string,
	simple_typedef_nodes  map[string]*ast.GenDecl,
	enum_typedef_nodes    map[string]*ast.GenDecl,
	const_group_nodes     map[string]*ast.GenDecl,
	struct_typedef_nodes  map[string]*ast.GenDecl,
    ) (
	struct_fields        map[string][]string,
	struct_field_C_types map[string]map[string]string,
	struct_field_tags    map[string]map[string]string,
	generated_C_code     string,
	err error,
    ) {
    package_defined_type := map[string]bool{};
    for key, _ := range simple_typedefs {
        fmt.Printf("+++ simple typedef for %s\n", key)
	package_defined_type[key] = true
    }
    for key, _ := range enum_typedefs {
        fmt.Printf("+++   enum typedef for %s\n", key)
	package_defined_type[key] = true
    }
    for key, _ := range struct_typedefs {
        fmt.Printf("+++ struct typedef for %s\n", key)
	package_defined_type[key] = true
    }

    // Hash of name of secondary pointer base types we have needed to generate a typedef for.
    have_ptr_type := map[string]bool{}
    // Hashe of names of secondary structs which we have needed to generate.
    have_list_struct  := map[string]bool{}
    have_pair_structs := map[string]bool{}
    // Precompiled regular expressions to match against the package name and typenames.
    dot           := regexp.MustCompile(`\.`)
    slash         := regexp.MustCompile(`/`)
    leading_array := regexp.MustCompile(`^\[\]`)
    leading_star  := regexp.MustCompile(`^\*`)
    // This expression ought to be generalized to check for balanced [] characters within the map key.
    map_key_value_types := regexp.MustCompile(`map\[([^]]+)\](.+)`)

    struct_fields        = map[string][]string{}
    struct_field_C_types = map[string]map[string]string{}
    struct_field_tags    = map[string]map[string]string{}

    header_boilerplate := template.Must(template.New("header_header").Parse(C_header_header_boilerplate))
    footer_boilerplate := template.Must(template.New("header_footer").Parse(C_header_footer_boilerplate))

    type C_header_boilerplate_fields struct {
        Year int
        HeaderFilename string
        HeaderSymbol string
    }

    current_year := time.Now().Year()
    header_filename := package_name + ".h"
    header_symbol := "_" + strings.ToUpper( slash.ReplaceAllLiteralString(package_name, "_") ) + "_H"
    boilerplate_variables := C_header_boilerplate_fields{Year: current_year, HeaderFilename: header_filename, HeaderSymbol: header_symbol}

    header_file, err := os.Create(header_filename);
    if err != nil {
        panic(err)
    }
    defer func() {
        if err := header_file.Close(); err != nil {
	    panic(err)
	}
    }()

    if err := header_boilerplate.Execute(header_file, boilerplate_variables); err != nil {
        panic("C header-file header processing failed");
    }

    for _, decl_kind := range final_type_order {
        // fmt.Printf("processing type %s %s\n", decl_kind.type_name, decl_kind.type_kind)
	switch decl_kind.type_kind {
	    case "simple":
		type_name := simple_typedefs[decl_kind.type_name]
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
		fmt.Fprintf(header_file, "extern const string const %s_String[];\n", decl_kind.type_name)
		fmt.Fprintf(header_file, "typedef enum %s %s_%[1]s;\n", decl_kind.type_name, package_name)
		fmt.Fprintf(header_file, "\n")
	    case "const":
		decl_node := const_group_nodes[decl_kind.type_name]
		fmt.Fprintf(header_file, "enum %s {\n", const_groups[decl_kind.type_name])
		for _, spec := range decl_node.Specs {
		    // This processing could be more complex, if there are other name-node types we might encounter here.
		    for _, name := range spec.(*ast.ValueSpec).Names {
			fmt.Fprintf(header_file, "    %s,\n", name.Name)
		    }
		}
		fmt.Fprintf(header_file, "};\n")
		fmt.Fprintf(header_file, "\n")
		generated_C_code += fmt.Sprintf("const string const %s_String[] = {\n", const_groups[decl_kind.type_name])
		for _, spec := range decl_node.Specs {
		    for _, value := range spec.(*ast.ValueSpec).Values {
			// This processing could be more complex, if there are other value-node types we might encounter here.
			generated_C_code += fmt.Sprintf("    %s,\n", value.(*ast.BasicLit).Value)
		    }
		}
		generated_C_code += fmt.Sprintf("};\n")
		generated_C_code += fmt.Sprintf("\n")
	    case "struct":
		decl_node := struct_typedef_nodes[decl_kind.type_name]
		struct_field_C_types[decl_kind.type_name] = map[string]string{}
		struct_field_tags   [decl_kind.type_name] = map[string]string{}
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
			    field_tag = field.Tag.Value
			}
			fmt.Printf("struct %s field tag:  %s\n", decl_kind.type_name, field_tag)
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
				    struct_field_C_types[decl_kind.type_name][name.Name] = type_name
				    struct_field_tags   [decl_kind.type_name][name.Name] = field_tag
				case *ast.SelectorExpr:
				    type_selectorexpr := field.Type.(*ast.SelectorExpr)
				    var x_type_ident *ast.Ident
				    var ok bool
				    if x_type_ident, ok = type_selectorexpr.X.(*ast.Ident); ok {
					// fmt.Printf("    --- struct field name and SelectorExpr X:  %#v %#v\n", name.Name, x_type_ident.Name)
				    } else {
					fmt.Printf("ERROR:  when autovivifying at %s, found unexpected field.Type.X type:  %T\n",
					    file_line, type_selectorexpr.X)
					fmt.Printf("ERROR:  struct field Type.X field is not of a recognized type\n")
					// panic_message = "aborting due to previous errors"
					// break node_loop
				    }
				    if type_selectorexpr.Sel == nil {
					fmt.Printf("ERROR:  when autovivifying at %s, struct field Type Sel field is unexpectedly nil\n", file_line())
					// panic_message = "aborting due to previous errors"
					// break node_loop
				    }
				    name_ident := new(ast.Ident)
				    // We used to append an underscore in this construction of name_ident.Name, but we
				    // are backing off from that until and unless we find it to actually be necessary.
				    // (The backoff is not yet done, pending testing.)
				    name_ident.Name = x_type_ident.Name + "_" + type_selectorexpr.Sel.Name

				    // special handling for "struct timespec"
				    if name_ident.Name == "time_Time" {
					name_ident.Name = "struct_timespec"
				    }

				    // FIX MAJOR:  clean this up
				    struct_definition += fmt.Sprintf("    %s %s;  // go:  %s\n",
					// "*ast.SelectorExpr.typename",
					// "FIX_MAJOR_dummy_typename",
					name_ident.Name,
					name.Name, struct_field_typedefs[decl_kind.type_name][name.Name])
				    // struct_definition += fmt.Sprintf("    %s %s;  // go: %[1]s\n", struct_field_typedefs[decl_kind.type_name][name.Name], name)
				    struct_fields[decl_kind.type_name] = append(struct_fields[decl_kind.type_name], name.Name)
				    // struct_field_C_types[decl_kind.type_name][name.Name] = "FIX_MAJOR_dummy_typename"
				    struct_field_C_types[decl_kind.type_name][name.Name] = name_ident.Name
				    struct_field_tags   [decl_kind.type_name][name.Name] = field_tag
				case *ast.StarExpr:
				    star_base_type := leading_star.ReplaceAllLiteralString(struct_field_typedefs[decl_kind.type_name][name.Name], "")
				    var base_type string
				    var field_type string
				    if leading_array.MatchString(star_base_type) {
					star_base_type = leading_array.ReplaceAllLiteralString(star_base_type, "")
					if package_defined_type[star_base_type] {
					    star_base_type = package_name + "_" + star_base_type
					}
					list_type := star_base_type + "_List"
					if !have_list_struct[list_type] {
					    have_list_struct[list_type] = true
					    fmt.Fprintf(header_file, "typedef struct _%s_ {\n", list_type);
					    fmt.Fprintf(header_file, "    size_t count;\n");
					    fmt.Fprintf(header_file, "    %s *items; // FIX MAJOR:  AAA\n", star_base_type);
					    fmt.Fprintf(header_file, "} %s;\n", list_type);
					    fmt.Fprintf(header_file, "\n");
					    struct_fields[list_type] = append(struct_fields[list_type], "count")
					    struct_fields[list_type] = append(struct_fields[list_type], "items")
					    struct_field_C_types[list_type] = map[string]string{}
					    struct_field_C_types[list_type]["count"] = "size_t"
					    struct_field_C_types[list_type]["items"] = star_base_type + " *"
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
				    if !have_ptr_type[base_type] {
					have_ptr_type[base_type] = true
					fmt.Fprintf(header_file, "typedef %s *%[1]s_Ptr;\n", base_type);
					fmt.Fprintf(header_file, "\n");
				    }
				    field_type = base_type + "_Ptr"
				    struct_definition += fmt.Sprintf("    %s %s;  // go: %s\n",
					field_type, name.Name, struct_field_typedefs[decl_kind.type_name][name.Name])
				    struct_fields[decl_kind.type_name] = append(struct_fields[decl_kind.type_name], name.Name)
				    struct_field_C_types[decl_kind.type_name][name.Name] = field_type
				    struct_field_tags   [decl_kind.type_name][name.Name] = field_tag
				case *ast.ArrayType:
				    // The constructions here are limited to what we have encountered in our own code.
				    // A more general conversion would handle additional array types, probably by some form of recursion.
				    array_base_type := leading_array.ReplaceAllLiteralString(struct_field_typedefs[decl_kind.type_name][name.Name], "")
				    if leading_star.MatchString(array_base_type) {
					array_base_type = leading_star.ReplaceAllLiteralString(array_base_type, "")
					// FIX QUICK:  this is not really the right place for this
					if package_defined_type[array_base_type] {
					    array_base_type = package_name + "_" + array_base_type
					}
					if !have_ptr_type[array_base_type] {
					    have_ptr_type[array_base_type] = true
					    fmt.Fprintf(header_file, "typedef %s *%[1]s_Ptr;\n", array_base_type);
					    fmt.Fprintf(header_file, "\n");
					}
					array_base_type += "_Ptr"
				    } else {
					if package_defined_type[array_base_type] {
					    array_base_type = package_name + "_" + array_base_type
					}
				    }
				    list_type := array_base_type + "_List"
				    if !have_list_struct[list_type] {
					have_list_struct[list_type] = true
					fmt.Fprintf(header_file, "typedef struct _%s_ {\n", list_type);
					fmt.Fprintf(header_file, "    size_t count;\n");
					fmt.Fprintf(header_file, "    %s *items;\n", array_base_type);
					fmt.Fprintf(header_file, "} %s;\n", list_type);
					fmt.Fprintf(header_file, "\n");
					struct_fields[list_type] = append(struct_fields[list_type], "count")
					struct_fields[list_type] = append(struct_fields[list_type], "items")
					struct_field_C_types[list_type] = map[string]string{}
					struct_field_C_types[list_type]["count"] = "size_t"
					struct_field_C_types[list_type]["items"] = array_base_type + " *"
				    }
				    struct_definition += fmt.Sprintf("    %s %s;  // go: %s\n",
					list_type, name.Name, struct_field_typedefs[decl_kind.type_name][name.Name])
				    struct_fields[decl_kind.type_name] = append(struct_fields[decl_kind.type_name], name.Name)
				    struct_field_C_types[decl_kind.type_name][name.Name] = list_type
				    struct_field_tags   [decl_kind.type_name][name.Name] = field_tag
				case *ast.MapType:
				    field_type := struct_field_typedefs[decl_kind.type_name][name.Name]
				    key_value_types := map_key_value_types.FindStringSubmatch(field_type)
				    if key_value_types == nil {
					panic(fmt.Sprintf("found incomprehensible map construction '%s'", field_type))
				    }
				    key_type   := key_value_types[1]
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
					fmt.Fprintf(header_file, "typedef struct _%s_ {\n", type_pair_type);
					fmt.Fprintf(header_file, "    %s key;\n", key_type);
					fmt.Fprintf(header_file, "    %s value;\n", value_type);
					fmt.Fprintf(header_file, "} %s;\n", type_pair_type);
					fmt.Fprintf(header_file, "\n");
					struct_fields[type_pair_type] = append(struct_fields[type_pair_type], "key")
					struct_fields[type_pair_type] = append(struct_fields[type_pair_type], "value")
					struct_field_C_types[type_pair_type] = map[string]string{}
					struct_field_C_types[type_pair_type]["key"] = key_type
					struct_field_C_types[type_pair_type]["value"] = value_type
					fmt.Fprintf(header_file, "typedef struct _%s_ {\n", type_pair_list_type);
					fmt.Fprintf(header_file, "    size_t count;\n");
					fmt.Fprintf(header_file, "    %s *items;\n", type_pair_type);
					fmt.Fprintf(header_file, "} %s;\n", type_pair_list_type);
					fmt.Fprintf(header_file, "\n");
					struct_fields[type_pair_list_type] = append(struct_fields[type_pair_list_type], "count")
					struct_fields[type_pair_list_type] = append(struct_fields[type_pair_list_type], "items")
					struct_field_C_types[type_pair_list_type] = map[string]string{}
					struct_field_C_types[type_pair_list_type]["count"] = "size_t"
					struct_field_C_types[type_pair_list_type]["items"] = type_pair_type + " *"
				    }
				    struct_definition += fmt.Sprintf("    %s %s;  // go: %s\n", type_pair_list_type, name.Name, field_type)
				    struct_fields[decl_kind.type_name] = append(struct_fields[decl_kind.type_name], name.Name)
				    struct_field_C_types[decl_kind.type_name][name.Name] = type_pair_list_type
				    struct_field_tags   [decl_kind.type_name][name.Name] = field_tag
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
		struct_definition += fmt.Sprintf("extern bool destroy_%s_%s_tree(%[1]s_%[2]s *%[1]s_%[2]s_ptr);\n", package_name, decl_kind.type_name)
		struct_definition += fmt.Sprintf("extern char *encode_%s_%s_as_json(const %[1]s_%[2]s *%[1]s_%[2]s_ptr, size_t flags);\n",
		    package_name, decl_kind.type_name)
		struct_definition += fmt.Sprintf("extern %s_%s *decode_json_%[1]s_%[2]s(const char *json_str);\n", package_name, decl_kind.type_name)
		struct_definition += fmt.Sprintf("#define free_%s_%s_tree destroy_%[1]s_%[2]s_tree\n", package_name, decl_kind.type_name)
		struct_definition += fmt.Sprintf("extern json_t *%s_%s_as_JSON(const %[1]s_%[2]s *%[1]s_%[2]s);\n", package_name, decl_kind.type_name)
		struct_definition += fmt.Sprintf("extern char *%s_%s_as_JSON_str(const %[1]s_%[2]s *%[1]s_%[2]s);\n", package_name, decl_kind.type_name)
		struct_definition += fmt.Sprintf("extern %s_%s *JSON_as_%[1]s_%[2]s(json_t *json);\n", package_name, decl_kind.type_name)
		struct_definition += fmt.Sprintf("extern %s_%s *JSON_str_as_%[1]s_%[2]s(const char *json_str);\n", package_name, decl_kind.type_name)
		fmt.Fprintln(header_file, struct_definition)
	    default:
		panic(fmt.Sprintf("found unknown type declaration kind '%s'", decl_kind.type_kind))
	}
    }

    if err := footer_boilerplate.Execute(header_file, boilerplate_variables); err != nil {
        panic("C header-file footer processing failed");
    }
    return struct_fields, struct_field_C_types, struct_field_tags, generated_C_code, nil
}

type json_field_tag struct {
    json_field_name string // `json:"name_in_json"`
    json_omitalways bool   // `json:"-"`
    json_omitempty  bool   // `json:",omitempty"`
    json_string     bool   // `json:",string"`
}

// FIX LATER:
// (*) The use of this function might not properly handle anonymous struct fields.
//     That remains to be investigated.
// (*) We presently take no notice of Go visibility rules, as reflected in the
//     initial capitalization of the struct field name.  Perhaps this processing
//     should be extended to take that into account.
// (*) The amendments to the Go visibility rules that apply during JSON marshalling
//     and unmarshalling as documented in https://golang.org/pkg/encoding/json/#Marshal
//     are badly written and thereby incomprehensible.  Until that situation clears up,
//     we are ignoring that level of finesse.
func interpret_json_field_tag(field_name string, struct_field_tag string) json_field_tag {
    field_tag := json_field_tag{
	json_field_name: field_name,
	json_omitalways: false,
	json_omitempty:  false,
	json_string:     false,
    }

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
		}
		if option == "string" {
		    field_tag.json_string = true
		}
	    }
	}
    }

    return field_tag
}

// FIX MAJOR:  add some routines here to decipher the content of a field tag

func generate_all_encode_tree_routines(
	package_name string,
	final_type_order []declaration_kind,

	// map[struct_name][]field_name
	struct_fields map[string][]string,

	// map[struct_name]map[field_name] = field_type
	struct_field_C_types map[string]map[string]string,

	// map[struct_name]map[field_name] = field_tag
	struct_field_tags map[string]map[string]string,
    ) (
	all_encode_function_code string,
	err error,
    ) {
    all_encode_function_code = ""
    for _, final_type := range final_type_order {
	if final_type.type_kind == "struct" {
	    function_code, err := generate_encode_PackageName_StructTypeName_tree(
		package_name, final_type.type_name, struct_fields, struct_field_C_types, struct_field_tags,
	    )
	    if err != nil {
	        panic(err)
	    }
	    all_encode_function_code += function_code
	}
    }
    return all_encode_function_code, err
}

func generate_all_decode_tree_routines(
	package_name string,
	final_type_order []declaration_kind,

	// map[struct_name][]field_name
	struct_fields map[string][]string,

	// map[struct_name]map[field_name] = field_type
	struct_field_C_types map[string]map[string]string,

	// map[struct_name]map[field_name] = field_tag
	struct_field_tags map[string]map[string]string,
    ) (
	all_decode_function_code string,
	err error,
    ) {

    // Prove that we really do have the struct_field_tags data structure populated as we expect it to be, in full detail.
    for struct_name, field_tags := range struct_field_tags {
	for field_name, field_tag := range field_tags {
	    fmt.Printf("struct_field_tags[%s][%s] = %s\n", struct_name, field_name, field_tag)
	}
    }

    all_decode_function_code = ""
    for _, final_type := range final_type_order {
	if final_type.type_kind == "struct" {
	    function_code, err := generate_decode_PackageName_StructTypeName_tree(
		package_name, final_type.type_name, struct_fields, struct_field_C_types, struct_field_tags,
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
	    function_code, err := generate_destroy_PackageName_StructTypeName_tree(
		package_name, final_type.type_name, struct_fields, struct_field_C_types,
	    )
	    if err != nil {
	        panic(err)
	    }
	    all_destroy_function_code += function_code
	}
    }
    return all_destroy_function_code, err
}

func generate_encode_PackageName_StructTypeName_tree(
	package_name string,
	struct_name string,

	// map[struct_name][]field_name
	struct_fields map[string][]string,

	// map[struct_name]map[field_name] = field_type
	struct_field_C_types map[string]map[string]string,

	// map[struct_name]map[field_name] = field_tag
	struct_field_tags map[string]map[string]string,
    ) (
	function_code string,
	err error,
    ) {
    // FIX MAJOR:  walk the structure, recursively, and fill this in
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
    {{.StructName}}_ptr = ({{.StructName}} *)malloc(size);
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

func generate_decode_PackageName_StructTypeName_tree(
	package_name string,
	struct_name string,

	// map[struct_name][]field_name
	struct_fields map[string][]string,

	// map[struct_name]map[field_name] = field_type
	struct_field_C_types map[string]map[string]string,

	// map[struct_name]map[field_name] = field_tag
	struct_field_tags map[string]map[string]string,
    ) (
	function_code string,
	err error,
    ) {
    trailing_List := regexp.MustCompile(`(.+)_List$`)
    trailing_Ptr  := regexp.MustCompile(`(.+)_Ptr$`)
    function_code = ""

    var decode_routine_header_template = `
{{.StructName}} *decode_json_{{.StructName}}(const char *json_str) {
  {{.StructName}} *{{.StructName}}_ptr = NULL; 
  size_t size = 0;
  json_error_t error;
  json_t *json = NULL; 

  json = json_loads(json_str, 0, &error);
  if (json) {
    fprintf(stderr, "decode_json_{{.StructName}} error on line %d: %s\n", error.line, error.text);
  } else {
`
    var decode_routine_footer_template = `
    json_decref(json);
  }

  return {{.StructName}}_ptr;
}

`

    header_template := template.Must(template.New("decode_routine_header").Parse(decode_routine_header_template))
    footer_template := template.Must(template.New("decode_routine_footer").Parse(decode_routine_footer_template))

    type decode_routine_boilerplate_fields struct {
        StructName string
    }

    boilerplate_variables := decode_routine_boilerplate_fields{StructName: package_name + "_" + struct_name}

    var header_code bytes.Buffer
    if err := header_template.Execute(&header_code, boilerplate_variables); err != nil {
        panic("decode routine header processing failed");
    }
    function_code += header_code.String()

    // When adding code like this, we need to double the % characters in any fprintf() format specifications
    // that we don't want Go itself to attempt to interpret.
    function_code += fmt.Sprintf(`    // allocate and fill the target struct by pointer
    %[1]s_ptr = (%[1]s *)malloc(size);
    if (%[1]s_ptr == NULL) {
      fprintf(stderr, "decode_json_%[1]s error: %%s\n", "malloc failed");
    } else {
`, package_name + "_" + struct_name)

      // char *ptr = (char *){{.StructName}}_ptr;
      // ptr += sizeof({{.StructName}});
      // {{.StructName}}_ptr->config.hostName = strcpy(ptr, json_string_value(jsonCfgHostName));
      // ptr += jsonCfgHostName_len + NUL_TERM_LEN;
      // {{.StructName}}_ptr->config.account = strcpy(ptr, json_string_value(jsonCfgAccount));
      // ptr += jsonCfgAccount_len + NUL_TERM_LEN;
      // {{.StructName}}_ptr->config.token = strcpy(ptr, json_string_value(jsonCfgToken));
      // {{.StructName}}_ptr->config.ssl = json_boolean_value(jsonCfgSSL);

    // Recursively decode all other structures pointed to by the given pointer.
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

    // FIX MINOR:  Clean up the generated comments used during development, to leave only short forms of
    // comments that reflect strutures and fields being skipped because they need no subsidiary deallocation.

    indent := "  "
    var process_item func(line_prefix string, parent_struct_type string, item_type string, item_prefix string, item_name string)
    process_item = func(line_prefix string, parent_struct_type string, item_type string, item_prefix string, item_name string) {
	// A xxx_List structure is just like any other structure we have manufactured, in that the details of its
	// fields have been recorded for our later use.  However, the .items field in this structure is special, in
	// that it refers not just to a single instance of the referred-to object, but to potentially many more.
	// So we must test for this structure before testing for other types of structures, so we guarantee that
	// the necessary special handling is applied.
	if matches := trailing_List.FindStringSubmatch(item_type); matches != nil {
	    // we have List of items; we just need to process the list, recursively decoding its individual elements
	    // we have an embedded xxx_List structure; we can presume its own internal construction,
	    // and use that to decode the complete set of individual elements in the list

/*
type json_field_tag struct {
    json_field_name string // `json:"name_in_json"`
    json_omitalways bool   // `json:"-"`
    json_omitempty  bool   // `json:",omitempty"`
    json_string     bool   // `json:",string"`
}
*/

	    field_tag := interpret_json_field_tag(item_name, struct_field_tags[parent_struct_type][item_name])
	    if !field_tag.json_omitalways {
		/*
		if !field_tag.json_omitempty || !is_empty_value("FIX MAJOR") {
		    function_code += fmt.Sprintf(`%sjson_t *json%s = json_object_get(json, "status");`, line_prefix, item_name, field_tag.json_field_name)
		}
		*/

		base_type := matches[1]
		var member_op string
		if item_name == "" {  // A "->" pointer dereference yields an unnamed object which needs no additional structure member operator.
		    member_op = ""
		} else {
		    member_op = "."
		}
		count_field := package_name + "_" + item_prefix + item_name + member_op + "count"
		items_field := package_name + "_" + item_prefix + item_name + member_op + "items"
		function_code += fmt.Sprintf("%s// list structure:  %s %s%s\n", line_prefix, item_type, item_prefix, item_name)
		function_code += fmt.Sprintf("%sif (%s != NULL) {\n", line_prefix + indent, items_field)
		function_code += fmt.Sprintf("%sfor (int index = %s; --index >= 0; ) {\n", line_prefix + indent + indent, count_field)
		function_code += fmt.Sprintf("%s// decode one %s item's fields\n", line_prefix + indent + indent + indent, base_type)
		process_item(line_prefix + indent + indent + indent, base_type, base_type, item_prefix + item_name + member_op, "items[index]")
		function_code += fmt.Sprintf("%s}\n", line_prefix + indent + indent)
		function_code += fmt.Sprintf("%sfree(%s);\n", line_prefix + indent + indent, items_field)
		function_code += fmt.Sprintf("%s}\n", line_prefix + indent)
		// process_item(line_prefix + indent, base_type, base_type, item_prefix + item_name, item_name)
	    }
	} else if field_list, ok := struct_fields[item_type]; ok {
	    // we have a known structure; we just need to recursively decode its individual fields
	    function_code += fmt.Sprintf("%s// embedded structure:  %s %s%s\n", line_prefix, item_type, item_prefix, item_name)
	    for _, field_name := range field_list {
		field_type := struct_field_C_types[item_type][field_name]
		// process the field as an item (just make a recursive call here)
		var member_op string
		if item_name == "" {  // A "->" pointer dereference yields an unnamed object which needs no additional structure member operator.
		    member_op = ""
		} else {
		    member_op = "."
		}
		process_item(line_prefix + indent, parent_struct_type, field_type, item_prefix + item_name + member_op, field_name)
	    }
	} else if matches := trailing_Ptr.FindStringSubmatch(item_type); matches != nil {

	    base_type := matches[1]
	    function_code += fmt.Sprintf("%s// pointer field:  %s %s%s\n", line_prefix, item_type, item_prefix, item_name)
	    process_item(line_prefix + indent, parent_struct_type, base_type, item_prefix + item_name + "->", "")
	} else if item_type == "string" {
	    // We don't bother checking for a NULL pointer, because modern free() will tolerate that.
	    function_code += fmt.Sprintf("%s// string field: parent_struct_type=%s item_name=%s\n", line_prefix, parent_struct_type, item_name)
	    function_code += fmt.Sprintf("%s// string field:  %s %s%s %s\n", line_prefix, item_type, item_prefix, item_name,
		struct_field_tags[parent_struct_type][item_name])
	    function_code += fmt.Sprintf("%sfree(%s%s);\n", line_prefix, package_name + "_" + item_prefix, item_name)

	} else {
	    // most likely, this will just be a simple base type whose storage is allocated directly in the existing structure
	    function_code += fmt.Sprintf("%s//  other field:  parent_struct_type=%s item_name=%s\n", line_prefix, parent_struct_type, item_name)
	    function_code += fmt.Sprintf("%s//  other field:  %s %s%s %s\n", line_prefix, item_type, item_prefix, item_name,
		struct_field_tags[parent_struct_type][item_name])
	}
    }
    process_item("    " + indent, struct_name, struct_name, struct_name + "_ptr->", "")
    function_code += fmt.Sprintf("    free(%s_ptr);\n", package_name + "_" + struct_name)

    function_code += `    }
`

    var footer_code bytes.Buffer
    if err := footer_template.Execute(&footer_code, boilerplate_variables); err != nil {
        panic("decode routine footer processing failed");
    }
    function_code += footer_code.String()

    return function_code, err
}

/*
Let's define a function that will generate the destroy_StructTypeName_tree() code, given the StructTypeName
and a list of all the available structs and their individual fields and field types.
*/
func generate_destroy_PackageName_StructTypeName_tree(
	package_name string,
	struct_name string,

	// map[struct_name][]field_name
	struct_fields map[string][]string,

	// map[struct_name]map[field_name] = field_type
	struct_field_C_types map[string]map[string]string,
    ) (
	function_code string,
	err error,
    ) {
    trailing_List := regexp.MustCompile(`(.+)_List$`)
    trailing_Ptr  := regexp.MustCompile(`(.+)_Ptr$`)
    function_code = ""

var destroy_routine_header_template = `bool destroy_{{.StructName}}_tree({{.StructName}} *{{.StructName}}_ptr) {
`
var destroy_routine_footer_template = `}

`

    header_template := template.Must(template.New("destroy_routine_header").Parse(destroy_routine_header_template))
    footer_template := template.Must(template.New("destroy_routine_footer").Parse(destroy_routine_footer_template))

    type destroy_routine_boilerplate_fields struct {
        StructName string
    }

    boilerplate_variables := destroy_routine_boilerplate_fields{StructName: package_name + "_" + struct_name}

    var header_code bytes.Buffer
    if err := header_template.Execute(&header_code, boilerplate_variables); err != nil {
        panic("destroy routine header processing failed");
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

    // FIX MINOR:  Clean up the generated comments used during development, to leave only short forms of
    // comments that reflect strutures and fields being skipped because they need no subsidiary deallocation.

    indent := "  "
    var process_item func(line_prefix string, item_type string, item_prefix string, item_name string)
    process_item = func(line_prefix string, item_type string, item_prefix string, item_name string) {
	// A xxx_List structure is just like any other structure we have manufactured, in that the details of its
	// fields have been recorded for our later use.  However, the .items field in this structure is special, in
	// that it refers not just to a single instance of the referred-to object, but to potentially many more.
	// So we must test for this structure before testing for other types of structures, so we guarantee that
	// the necessary special handling is applied.
	if matches := trailing_List.FindStringSubmatch(item_type); matches != nil {
	    // we have List of items; we just need to process the list, recursively destroying its individual elements
	    // we have an embedded xxx_List structure; we can presume its own internal construction,
	    // and use that to destroy the complete set of individual elements in the list
	    base_type := matches[1]
	    var member_op string
	    if item_name == "" {  // A "->" pointer dereference yields an unnamed object which needs no additional structure member operator.
		member_op = ""
	    } else {
		member_op = "."
	    }
	    count_field := package_name + "_" + item_prefix + item_name + member_op + "count"
	    items_field := package_name + "_" + item_prefix + item_name + member_op + "items"
	    function_code += fmt.Sprintf("%s// list structure:  %s %s%s\n", line_prefix, item_type, item_prefix, item_name)
	    // function_code += fmt.Sprintf("%s// list field:  %s\n", line_prefix + indent, count_field)
	    // function_code += fmt.Sprintf("%s// list field:  %s\n", line_prefix + indent, items_field)
	    function_code += fmt.Sprintf("%sif (%s != NULL) {\n", line_prefix + indent, items_field)
	    // FIX LATER:  If it turns out that there are no free() operations that should take place inside the for loop,
	    // the for loop itself has no practical effect and should just be omitted from our generated code.  That
	    // optimization awaits some future version of this program.
	    function_code += fmt.Sprintf("%sfor (int index = %s; --index >= 0; ) {\n", line_prefix + indent + indent, count_field)
	    function_code += fmt.Sprintf("%s// delete one %s item's fields\n", line_prefix + indent + indent + indent, base_type)
	    process_item(line_prefix + indent + indent + indent, base_type, item_prefix + item_name + member_op, "items[index]")
	    function_code += fmt.Sprintf("%s}\n", line_prefix + indent + indent)
	    function_code += fmt.Sprintf("%sfree(%s);\n", line_prefix + indent + indent, items_field)
	    function_code += fmt.Sprintf("%s}\n", line_prefix + indent)
	    // process_item(line_prefix + indent, base_type, item_prefix + item_name, item_name)
	} else if field_list, ok := struct_fields[item_type]; ok {
	    // we have a known structure; we just need to recursively destroy its individual fields
	    function_code += fmt.Sprintf("%s// embedded structure:  %s %s%s\n", line_prefix, item_type, item_prefix, item_name)
	    for _, field_name := range field_list {
		field_type := struct_field_C_types[item_type][field_name]
		// process the field as an item (just make a recursive call here)
		// function_code += fmt.Sprintf("%s// struct field item_type=%s item_prefix=%s item_name=%s field_type=%s field_name=%s\n",
		    // line_prefix, item_type, item_prefix, item_name, field_type, field_name)
		var member_op string
		if item_name == "" {  // A "->" pointer dereference yields an unnamed object which needs no additional structure member operator.
		    member_op = ""
		} else {
		    member_op = "."
		}
		process_item(line_prefix + indent, field_type, item_prefix + item_name + member_op, field_name)
	    }
	} else if matches := trailing_Ptr.FindStringSubmatch(item_type); matches != nil {

	    base_type := matches[1]
	    // function_code += fmt.Sprintf("%s// process pointer:  %s %s%s\n", line_prefix, item_type, item_prefix, item_name)
	    function_code += fmt.Sprintf("%s// pointer field:  %s %s%s\n", line_prefix, item_type, item_prefix, item_name)
	    process_item(line_prefix + indent, base_type, item_prefix + item_name + "->", "")
	} else if item_type == "string" {
	    // We don't bother checking for a NULL pointer, because modern free() will tolerate that.
	    // function_code += fmt.Sprintf("%s// string item_type=%s item_prefix=%s item_name=%s\n", line_prefix, item_type, item_prefix, item_name)
	    function_code += fmt.Sprintf("%s// string field:  %s %s%s\n", line_prefix, item_type, item_prefix, item_name)
	    function_code += fmt.Sprintf("%sfree(%s%s);\n", line_prefix, package_name + "_" + item_prefix, item_name)

	} else {
	    // most likely, there is nothing to do for such a field, as it will just be a simple base type
	    // whose storage is allocated directly in the existing structure
	    // function_code += fmt.Sprintf("%s//  other field:  item_type=%s item_prefix=%s item_name=%s\n",
		// line_prefix, item_type, item_prefix, item_name)
	    function_code += fmt.Sprintf("%s//  other field:  %s %s%s\n", line_prefix, item_type, item_prefix, item_name)
	}
    }
    process_item(indent, struct_name, struct_name + "_ptr->", "")
    function_code += fmt.Sprintf("    free(%s_ptr);\n", package_name + "_" + struct_name)

    var footer_code bytes.Buffer
    if err := footer_template.Execute(&footer_code, boilerplate_variables); err != nil {
        panic("destroy routine footer processing failed");
    }
    function_code += footer_code.String()

    return function_code, err
}

// FIX MINOR:  I foresee application code wanting to have top-level recursive-destroy routines for each
// of the supported structures, so if the programmer has been careful to not have any cross-sharing of
// allocated objects, an entire tree of prior allocations can be deallocated in one call.  That is
// distinct from the kind of deallocation used when freeing memory returned by some varient of the
// decode_json_StructTypeName() routine, which will just free a single block of memory that we know
// has embedded within it in contiguous memory, the top-level data structure and all of its possibly
// multi-generational offspring.  The recursive-destroy routines are what I am calling above:
//
//     extern bool destroy_PackageName_StructTypeName_tree(PackageName_StructTypeName *PackageName_StructTypeName_ptr);
//
func print_type_conversions(
	other_headers         string,
	generated_C_code      string,
	final_type_order      []declaration_kind,
	package_name          string,
	simple_typedefs       map[string]string,
	enum_typedefs         map[string]string,
	const_groups          map[string]string,
	struct_typedefs       map[string][]string,
	struct_field_typedefs map[string]map[string]string,
	simple_typedef_nodes  map[string]*ast.GenDecl,
	enum_typedef_nodes    map[string]*ast.GenDecl,
	const_group_nodes     map[string]*ast.GenDecl,
	struct_typedef_nodes  map[string]*ast.GenDecl,
	struct_fields         map[string][]string,
	struct_field_C_types  map[string]map[string]string,
	struct_field_tags     map[string]map[string]string,
    ) error {

    header_boilerplate := template.Must(template.New("code_file_header").Parse(C_code_boilerplate))

    type C_code_boilerplate_fields struct {
        Year int
        CodeFilename string
	OtherHeaders string
        HeaderFilename string
    }

    current_year    := time.Now().Year()
    code_filename   := package_name + ".c"
    header_filename := package_name + ".h"
    boilerplate_variables := C_code_boilerplate_fields{
	Year: current_year,
	CodeFilename: code_filename,
	OtherHeaders: other_headers,
	HeaderFilename: header_filename,
    }

    code_file, err := os.Create(code_filename);
    if err != nil {
        panic(err)
    }
    defer func() {
        if err := code_file.Close(); err != nil {
	    panic(err)
	}
    }()

    if err := header_boilerplate.Execute(code_file, boilerplate_variables); err != nil {
        panic("C code-file header processing failed");
    }

    fmt.Fprintf(code_file, "%s", generated_C_code);

    all_encode_function_code, err := generate_all_encode_tree_routines(
	package_name, final_type_order, struct_fields, struct_field_C_types, struct_field_tags,
    )
    if err != nil {
        panic(err)
    }
    fmt.Fprintf(code_file, "%s", all_encode_function_code);

    all_decode_function_code, err := generate_all_decode_tree_routines(
	package_name, final_type_order, struct_fields, struct_field_C_types, struct_field_tags,
    )
    if err != nil {
        panic(err)
    }
    fmt.Fprintf(code_file, "%s", all_decode_function_code);

    all_destroy_function_code, err := generate_all_destroy_tree_routines(
	package_name, final_type_order, struct_fields, struct_field_C_types,
    )
    if err != nil {
        panic(err)
    }
    fmt.Fprintf(code_file, "%s", all_destroy_function_code);

    for _, decl_kind := range final_type_order {
        // fmt.Printf("processing type %s %s\n", decl_kind.type_name, decl_kind.type_kind)
	switch decl_kind.type_kind {
	    case "simple":
		/*
		type_name := simple_typedefs[decl_kind.type_name]
		fmt.Fprintf(code_file, "typedef %s %s;\n", type_name, decl_kind.type_name)
		fmt.Fprintf(code_file, "\n")
		*/
	    case "enum":
		/*
		fmt.Fprintf(code_file, "extern const string const %s_String[];\n", decl_kind.type_name)
		fmt.Fprintf(code_file, "typedef enum %s %[1]s;\n", decl_kind.type_name)
		fmt.Fprintf(code_file, "\n")
		*/
	    case "const":
		/*
		decl_node := const_group_nodes[decl_kind.type_name]
		fmt.Fprintf(code_file, "enum %s {\n", const_groups[decl_kind.type_name])
		for _, spec := range decl_node.Specs {
		    // This processing could be more complex, if there are other name-node types we might encounter here.
		    for _, name := range spec.(*ast.ValueSpec).Names {
			fmt.Fprintf(code_file, "    %s,\n", name.Name)
		    }
		}
		fmt.Fprintf(code_file, "};\n")
		fmt.Fprintf(code_file, "\n")
		fmt.Fprintf(code_file, "const string const %s_String[] = {\n", const_groups[decl_kind.type_name])
		for _, spec := range decl_node.Specs {
		    for _, value := range spec.(*ast.ValueSpec).Values {
			// This processing could be more complex, if there are other value-node types we might encounter here.
			fmt.Fprintf(code_file, "    %s,\n", value.(*ast.BasicLit).Value)
		    }
		}
		fmt.Fprintf(code_file, "};\n")
		fmt.Fprintf(code_file, "\n")
		*/
	    case "struct":
		//
		// Here, the goal is to generate source code for two functions:
		//
		//     extern char *encode_StructTypeName_as_json(const StructTypeName *StructTypeName_ptr, size_t flags);
		//     extern StructTypeName *decode_json_StructTypeName(const char *json_str);
		//
		// The mechanism for dat conversion involve calls to the Jansson library, which handles lots of the low-level detail.
		// That said, we must walk the data structures, handle some aspects of memory management and padding to meet structure
		// alignment requirements, create subsidiary objects and strings, and otherwise orchestrate the process.
		//
		// To make this work on a practical basis, we don't want to be walking the AST all over again, if we can help it.
		// Instead we want to have created equivalent data structures that we can walk more readily here.
		//
		// Also note that we need to take into account any field tags that are attached to struct fields in the Go code, to
		// at a minimum ensure that (1) we switch fields names when required, and (2) we handle omitted fields correctly,
		// during both encoding and decoding.
		//
		/*
		decl_node := struct_typedef_nodes[decl_kind.type_name]
		var struct_definition string
		for _, spec := range decl_node.Specs {
		    struct_definition = fmt.Sprintf("typedef struct {\n")
		    for _, field := range spec.(*ast.TypeSpec).Type.(*ast.StructType).Fields.List {
			for _, name := range field.Names {
			    switch field.Type.(type) {
				case *ast.Ident:
				    type_name := field.Type.(*ast.Ident).Name
				    struct_definition += fmt.Sprintf("    %s %s;\n", type_name, name)
				case *ast.SelectorExpr:
				    // FIX MAJOR:  clean this up
				    struct_definition += fmt.Sprintf("    %s %s;  // go:  %s\n",
					// "*ast.SelectorExpr.typename",
					"FIX_MAJOR_dummy_typename",
					name, struct_field_typedefs[decl_kind.type_name][name.Name])
				    // struct_definition += fmt.Sprintf("    %s %s;  // go: %[1]s\n", struct_field_typedefs[decl_kind.type_name][name.Name], name)
				case *ast.StarExpr:
				    star_base_type := leading_star.ReplaceAllLiteralString(struct_field_typedefs[decl_kind.type_name][name.Name], "")
				    var base_type string
				    var field_type string
				    if leading_array.MatchString(star_base_type) {
					star_base_type = leading_array.ReplaceAllLiteralString(star_base_type, "")
					list_type := star_base_type + "_List"
					if !have_list_struct[list_type] {
					    have_list_struct[list_type] = true
					    fmt.Fprintf(code_file, "typedef struct {\n");
					    fmt.Fprintf(code_file, "    size_t count;\n");
					    fmt.Fprintf(code_file, "    %s *items;\n", star_base_type);
					    fmt.Fprintf(code_file, "} %s;\n", list_type);
					    fmt.Fprintf(code_file, "\n");
					}
				        base_type = list_type
				    } else {
				        base_type = star_base_type
				    }
				    if !have_ptr_type[base_type] {
					have_ptr_type[base_type] = true
					fmt.Fprintf(code_file, "typedef %s *%[1]s_Ptr;\n", base_type);
					fmt.Fprintf(code_file, "\n");
				    }
				    field_type = base_type + "_Ptr"
				    struct_definition += fmt.Sprintf("    %s %s;  // go: %s\n",
					field_type, name, struct_field_typedefs[decl_kind.type_name][name.Name])
				case *ast.ArrayType:
				    // The constructions here are limited to what we have encountered in our own code.
				    // A more general conversion would handle additional array types, probably by some form of recursion.
				    array_base_type := leading_array.ReplaceAllLiteralString(struct_field_typedefs[decl_kind.type_name][name.Name], "")
				    if leading_star.MatchString(array_base_type) {
					array_base_type = leading_star.ReplaceAllLiteralString(array_base_type, "")
					if !have_ptr_type[array_base_type] {
					    have_ptr_type[array_base_type] = true
					    fmt.Fprintf(code_file, "typedef %s *%[1]s_Ptr;\n", array_base_type);
					    fmt.Fprintf(code_file, "\n");
					}
					array_base_type += "_Ptr"
				    }
				    list_type := array_base_type + "_List"
				    if !have_list_struct[list_type] {
					have_list_struct[list_type] = true
					fmt.Fprintf(code_file, "typedef struct {\n");
					fmt.Fprintf(code_file, "    size_t count;\n");
					fmt.Fprintf(code_file, "    %s *items;\n", array_base_type);
					fmt.Fprintf(code_file, "} %s;\n", list_type);
					fmt.Fprintf(code_file, "\n");
				    }
				    struct_definition += fmt.Sprintf("    %s %s;  // go: %s\n",
					list_type, name, struct_field_typedefs[decl_kind.type_name][name.Name])
				case *ast.MapType:
				    field_type := struct_field_typedefs[decl_kind.type_name][name.Name]
				    key_value_types := map_key_value_types.FindStringSubmatch(field_type)
				    if key_value_types == nil {
					panic(fmt.Sprintf("found incomprehensible map construction '%s'", field_type))
				    }
				    key_type   := key_value_types[1]
				    value_type := key_value_types[2]
				    type_pair_type := key_type + "_" + value_type + "_Pair"
				    type_pair_list_type := type_pair_type + "_List"
				    if !have_pair_structs[type_pair_type] {
					have_pair_structs[type_pair_type] = true
					fmt.Fprintf(code_file, "typedef struct {\n");
					fmt.Fprintf(code_file, "    %s key;\n", key_type);
					fmt.Fprintf(code_file, "    %s value;\n", value_type);
					fmt.Fprintf(code_file, "} %s;\n", type_pair_type);
					fmt.Fprintf(code_file, "\n");
					fmt.Fprintf(code_file, "typedef struct {\n");
					fmt.Fprintf(code_file, "    size_t count;\n");
					fmt.Fprintf(code_file, "    %s *items;\n", type_pair_type);
					fmt.Fprintf(code_file, "} %s;\n", type_pair_list_type);
					fmt.Fprintf(code_file, "\n");
				    }
				    struct_definition += fmt.Sprintf("    %s %s;  // go: %s\n", type_pair_list_type, name, field_type)
				default:
				    panic(fmt.Sprintf("found unexpected field type %T", field.Type))
			    }
			}
		    }
		    struct_definition += fmt.Sprintf("} %s;\n", decl_kind.type_name)
		    struct_definition += fmt.Sprintf("\n")
		}
		struct_definition += fmt.Sprintf("extern char *encode_%s_as_json(const %[1]s *%[1]s_ptr, size_t flags);\n", decl_kind.type_name)
		struct_definition += fmt.Sprintf("extern %s *decode_json_%[1]s(const char *json_str);\n", decl_kind.type_name)
		fmt.Fprintln(code_file, struct_definition)
		*/
	    default:
		panic(fmt.Sprintf("found unknown type declaration kind '%s'", decl_kind.type_kind))
	}
    }

    return nil
}
