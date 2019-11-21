//--------------------------------------------------------------//

#define stringify(x)			#x
#define expand_and_stringify(x)		stringify(x)

// Here's a statement we can use to fall through from one case to the next in a
// switch statement, documenting such an action without a comment.  Use it like
// you would "break;", except that it has the opposite effect.
#define	fallthrough

// Here are statements we can use to introduce a local block that is to be
// executed just once, or not at all, while keeping our indendation standards
// intact.  Think of "perform" as a do-while without the while.  Use it like this:
//
//     perform
//         {
//         do_stuff ();
//         }
//
// That can be useful when you need/want to declare a local block within open
// code in order to declare some variables with limited scope.
//
// Similarly, "disregard" can be used to preface a local block of code you want
// to be commented out for the time being:
//
//     disregard
//         {
//         do_stuff ();
//         }
//
// as an alternative to enclosing the block in #if 0 ... #endif .  You can then
// switch the block on and off by switching between perform and disregard.
//
// The only potential downside of these statements is that some compilers may
// warn about using a "constant in conditional context" (while still generating
// correct code).
//
#define	perform		if (1)
#define disregard	if (0)

//--------------------------------------------------------------//

// Memory-related definitions

// This coding pattern is so common, it's time we made it easier to invoke.
#define	arraysize(array)	(sizeof (array) / sizeof (array[0]))

// check_pointer() is typically used in C code after calling routines such as malloc() or strdup(),
// to validate the result.
//
// BEWARE:  Don't use check_pointer() in C++ code to check the result of a new operation;
// new will throw a bad_alloc exception, not return a NULL pointer.  And trying to get around
// this by using using new(nothrow) turns out to be a horrible idea for some very technical
// reasons.  You may still use this in C++ code that has some need to call malloc(), strdup(),
// or a similar function; otherwise, we would disable it entirely when compiling under C++.
#define check_pointer(ptr)		\
	do  {				\
	    if ((ptr) == NULL)		\
		{			\
		log_message (APP_FATAL, FILE_LINE "Insufficient memory for " #ptr "; exiting!");	\
		exit (EXIT_FAILURE);	\
		}			\
	    } while (0)

//--------------------------------------------------------------//

// Portability definitions

#if defined(_LP64)
#define	SIZE_T_FORMAT	"%lu"
#else
#define	SIZE_T_FORMAT	"%u"
#endif

//--------------------------------------------------------------//
