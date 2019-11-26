// We expect the calling Makefile to define Package and Structure symbols for
// use here, so we can dynamically customize this code to generate function calls
// to the rouintes that support a specific externally-defined package and structure.
//
// Alas, as far as I know we cannot generalize this code to be completely independent
// of the packages it is testing, because of the preprocessor #include lines we must
// specifspecify in this file (see below).

#include <stdlib.h>
#include <stdio.h>
#include <sys/types.h>
#include <sys/stat.h>
#include <unistd.h>
#include <errno.h>
#include <string.h>

#include "jansson.h"

#include "convert_go_to_c.h"

// These headers are required on a per-package basis, for using any of the
// declarations and definitions provided in the respective header files.
//
// FIX LATER:  Ideally, we ought to have each generated .h file automatically
// include #include lines for all of the other package headers it needs.
// Then we would only need to externally specify the single #include line
// we would need here (though generating that line via preprocessor macro
// substitution within this file is perhaps impossible).
//
#include "config.h"
#include "milliseconds.h"
#include "transit.h"

#define	FAILURE	0	// for use in routine return values 
#define	SUCCESS	1	// for use in routine return values

// We make this a const string to attempt to bypass some overly aggressive compiler security warnings.
const char separation_line[] = "--------------------------------------------------------------------------------\n";

void print_first_different_character(char *a, char *b) {
    int i;
    int line = 1;
    int character = 1;
    for (i = 0; *a && *b; ++i, ++character, ++a, ++b) {
	if (*a != *b) {
	    break;
	} else if (*a == '\n') {
	    ++line;
	    character = 0;
	}
    }
    if (*a != *b) {
	char a_string[] = "  ";
	char b_string[] = "  ";
	if (*a < ' ') {
	    a_string[0] = '\\';
	    a_string[1] = *a == '\r' ? 'r' : *a == '\n' ? 'n' : *a == '\t' ? 't' : *a + 0x40;
	    if (a_string[1] < ' ') {
		a_string[0] = '^';
		a_string[1] = *a + 0x40;
	    }
	} else {
	    a_string[0] = *a;
	    a_string[1] = '\0';
	}
	if (*b < ' ') {
	    b_string[0] = '\\';
	    b_string[1] = *b == '\r' ? 'r' : *b == '\n' ? 'n' : *b == '\t' ? 't' : *b;
	    if (b_string[1] < ' ') {
		b_string[0] = '^';
		b_string[1] = *b + 0x40;
	    }
	} else {
	    b_string[0] = *b;
	    b_string[1] = '\0';
	}
	printf("strings are different at position %d (line %d char %d ['%s' vs. '%s'])\n", i, line, character, a_string, b_string);
    }
}

// Because the values we will substitute into the first two of these macros are themselves #define'd strings
// (in our case externally defined), we apparently need a level of indirection in the expansion of those macros.
//
#define test_object(PACKAGE, STRUCTURE)				generate_test_object    (PACKAGE, STRUCTURE)
#define run_object_test(PACKAGE, STRUCTURE, JSON_FILEPATH)	generate_run_object_test(PACKAGE, STRUCTURE, JSON_FILEPATH)
//
#define generate_test_object(PACKAGE, STRUCTURE)			test_json_string(PACKAGE##_##STRUCTURE, stringify(PACKAGE##_##STRUCTURE))
#define generate_run_object_test(PACKAGE, STRUCTURE, JSON_FILEPATH)	test_##PACKAGE##_##STRUCTURE##_json_string(JSON_FILEPATH)

#define test_json_string(OBJECT, OBJSTR)										\
int test_##OBJECT##_json_string(char *json_filepath) {									\
    if (0) {														\
	printf(separation_line);											\
    }															\
    struct stat buf;													\
    int outcome = stat(json_filepath, &buf);										\
    if (outcome != 0) {													\
	printf("ERROR:  %s cannot be read (error %d)\n", json_filepath, errno);						\
	exit (1);													\
    }															\
    char *initial_##OBJECT##_as_json_string = (char *) calloc(1, buf.st_size + NUL_TERM_LEN);				\
    if (initial_##OBJECT##_as_json_string == NULL) {									\
	printf("ERROR:  Cannot allocate memory to read %s (error %d)\n", json_filepath, errno);				\
	exit (1);													\
    }															\
    FILE *json_file = fopen(json_filepath, "r");									\
    if (json_file == NULL) {												\
	printf("ERROR:  Cannot open file %s for reading (error %d)\n", json_filepath, errno);				\
	exit (1);													\
    }															\
    size_t objects_read = fread(initial_##OBJECT##_as_json_string, buf.st_size, 1, json_file);				\
    if (objects_read != 1) {												\
	printf("ERROR:  Cannot read file %s\n", json_filepath);								\
	exit (1);													\
    }															\
    if (initial_##OBJECT##_as_json_string[buf.st_size - 1] == '\n') {							\
	initial_##OBJECT##_as_json_string[buf.st_size - 1] = '\0';							\
    }															\
    outcome = fclose(json_file);											\
    if (outcome != 0) {													\
	printf("ERROR:  %s cannot be closed (error %d)\n", json_filepath, errno);					\
	exit (1);													\
    }															\
    json_t *json;													\
    printf("Decoding "OBJSTR" JSON string ...\n");									\
    OBJECT *OBJECT##_ptr = JSON_str_as_##OBJECT##_ptr(initial_##OBJECT##_as_json_string, &json);			\
    if (OBJECT##_ptr == NULL) {												\
	printf (FILE_LINE "ERROR:  JSON string cannot be decoded into a "OBJSTR" object\n");				\
	return FAILURE;													\
    }															\
    else {														\
	printf ("Encoding "OBJSTR" object tree ...\n");									\
	char *final_##OBJECT##_as_json_string = OBJECT##_ptr_as_JSON_str(OBJECT##_ptr);					\
	if (final_##OBJECT##_as_json_string == NULL) {									\
	    printf (FILE_LINE "ERROR:  "OBJSTR" object cannot be encoded as a JSON string\n");				\
	    return FAILURE;												\
	}														\
	else {														\
	    int matches = !strcmp(final_##OBJECT##_as_json_string, initial_##OBJECT##_as_json_string);			\
	    printf (													\
		"Final string for decode/encode of "OBJSTR" %s the original string.\n",					\
		(matches ? "matches" : "DOES NOT MATCH")								\
	    );														\
	    if (!matches) {												\
		printf("original string:\n%s\n", initial_##OBJECT##_as_json_string);					\
		printf("   final string:\n%s\n",   final_##OBJECT##_as_json_string);					\
		print_first_different_character(initial_##OBJECT##_as_json_string, final_##OBJECT##_as_json_string);	\
		return FAILURE;												\
	    }														\
	    free(final_##OBJECT##_as_json_string);									\
	}														\
	free_##OBJECT##_ptr_tree(OBJECT##_ptr, json);									\
    }															\
    /*															\
    // We use just the first of these two calls (done just above, for now), because it's our official cleanup routine.	\
    destroy_##OBJECT##_tree(OBJECT##_ptr, json);									\
    free_JSON(json);													\
    */															\
    return SUCCESS;													\
}

// Generate all the individual test functions for particular objects.
test_object(Package, Structure)

int main (int argc, char *argv[]) {
    json_t *json;
    char *json_filepath = argv[1];

    int success = run_object_test(Package, Structure, json_filepath);
    // printf(separation_line);
    printf("Test of package %s structure %s %s.\n", expand_and_stringify(Package), expand_and_stringify(Structure), success ? "PASSED" : "FAILED");

    return success ? EXIT_SUCCESS : EXIT_FAILURE;
}
