// This is a Go package that defines generic complex datatypes that use only
// combinations of facilities like "string" and perhaps "time.Time" that are
// not specific to application classes.  The danger in defining such datatypes
// in an application class instead is that several such classes can define
// the same datatype.  If code is then generated independently for those
// classes, there will be header-file and object-file collisions when an
// encompassing application tries to use those classes together.  To solve
// that problem, this generic_datatypes package provides a place to house
// any such datatypes, factoring them out so there will be only one copy
// of them in a generated header file and only one copy of them in the
// compiled object files.

package generic_datatypes

// FIX LATER:  We ought to be able to support type declarations like this as well.
// type map_string_string map[string]string

// We don't expect this structure itself to be used, so no code will be generated
// for it in either the header file or the C code.  It is only a placeholder so we
// can invoke the parts of the gotocjson conversion tool under a special mode (which
// is invoked with the -g option) that will generate code for each of the datatypes
// of the fields in this structure whose datatype has no particular association
// with some non-generic application class.  That is, those datatypes consist
// only of combinations of strings, struct timespecs (time.Time, as it would be
// represented in the Go code), and the like, that are generic and might be (in such
// combinations) used across multiple application .go files.
//
// Essentially, this provides a means of factoring out such shared items into a
// separate header and object file, to avoid conflicts.  In the current iteration of
// the gotocjson tool, generation of declarations and functions to handle conversion
// of the GenericDataTypes structure itself is suppressed, to get rid of extra noise.
// If we find we want to support generating declarations and code for shared data
// structures consisting only of fields of generic datatypes, we can revisit that
// implementation decision.  But that would only make sense if the field names in
// such structures were named in some very generic ways.
//
// We may add new fields to this structure as the application code that uses these
// generic datatypes evolves.
type GenericDataTypes struct {
    MapStringString map[string]string
}
