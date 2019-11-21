#include <stdio.h>

// #include "transit.h"

extern const char * const foo_UnitEnum_String[];  // index by UnitEnum value, get associated string value back
typedef enum foo_UnitEnum foo_UnitEnum;

typedef int *int_Ptr;

int_Ptr int_ptr;

enum foo_UnitEnum {
    foo_NoUnits,
    foo_UnitCounter,
    foo_PercentCPU,
    foo_bar,
};

#define string char *

foo_UnitEnum units;

const string const foo_UnitEnum_String[] = {
    "",            // no units specified
    "1",           // unspecified-unit counter 
    "%{cpu}",      // percent CPU, as in load measurements
    "foo is bar",
};

typedef struct {
    int abc;
} foo;

typedef struct {
    foo foo;
} bar;

int main () {
    foo foo;
    bar bar;
    foo.abc = 1;
    bar.foo.abc = 2;
    printf ("foo.abc = %d; bar.foo.abc = %d\n", foo.abc, bar.foo.abc);
    printf ("number of foo_UnitEnum enumeration elements: %ld\n", sizeof(foo_UnitEnum_String) / sizeof(string));
    enum foo_UnitEnum UnitEnum_variable;
    UnitEnum_variable = 3;
    printf ("foo_UnitEnum element %d: %s\n", UnitEnum_variable, foo_UnitEnum_String[UnitEnum_variable]);
    return 23;
}
