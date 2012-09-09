/*

This is the TranScript SDK. TranScript is a dynamic programming language where
everything is an object.

Basic Syntax

Comments are as in Go. They are ignored by the language. Block comments don't
nest.

*/
//	// line comment
//	/* block comment */
/*

Numbers are represented as series of digits, optionally separated with "."
and/or preceded with "-".

	0
	150
	-13
	14.72
	-2.8

The basic arithmetic and comparative operators work as expected.

	1 + 3           // 4
	3 / 2           // 1.5
	12 * 4 - 6      // 42
	5 >= 4          // true
	5 == 4 + 2      // false

Supported operations are "+", "-", "*", "/", "==", "!=", "<", ">", "<=", ">=".

Names follow the C syntax convention: Letters or "_" to begin, letters, digits
or "_" after that.

	foo
	Object
	foo_bar

Names refer to things. They might refer to a definition, or they might refer to
a member on an object.

	a               // definition
	a.prop          // member

There are two boolean values, "true" and "false". 

Booleans support one operation, negation.

	!true           // false
	!false          // true

This is actually supported by every other object in TranScript as well. For
every value other than "false", "false" is returned. For "false", "true" is
returned.

The logical operators "&&" (and) and "||" (or) are provided.

	true && true    // true
	false || true   // true
	false && true   // false
	false || false  // false

These operators have short-circuiting. This means that if the value of the 
expression can be determined after evaluating the left operand, the right 
operand is not evaluated.

There is "nil", which represents no value. This is primarily used as the return 
value of functions and methods that are only called for their side effects, as
well as for uninitialised fields and variables.

Strings have the usual C-like syntax: enclosed in speech marks with "\" for
escape sequences.

	"Hello, world!\n"

Two strings may be concatenated using "+".

	"Hello, " + "world!\n" // gives the same value as before

Arrays are series of objects enclosed in brackets and separated with ",".

	[1, 2, 3]

The familiar subscript access is available.

	array[0]      // get the zeroth member
	array[1] = 2  // set the first member to 2

Arrays may be concatenated as strings are.

	[1, 2] + [3, 4] // [1, 2, 3, 4]

Hashes are key-value pairs enclosed in curly brackets and separated with ",".
In each pair, the key is separated from the value by ":".

	{"key1": "value1", "key2": "value2"}

Any object may be a key or a value. Only strings and numbers are likely to be
useful keys most of the time, though.

Conditional Evaluation

An if statement evaluates an expression. If it evaluates to "true" then "if" 
evalutates its "then" block. If the expression evaluates to "false" then "if" 
evaluates its else block. Every value other than "false" is counted as "true" to
an "if" statement.

	if <expression> then <block> else <block> end

A block is a series of statements, each terminated with ";".

	a = 1;
	b = 2;
	print(a + b);

e.g.

	def a = 1;
	if a < 5 then
		print("a is less than five");
	end;
	if a > 5 then
		print("a is greater than five");
	end;

This prints "a is less than five".

Variables And Scope

Variables allow you to store state and refer to the results of expresssions.

Defining a variable looks like

	def <name>;                  // uninitialised
	def <name> = <expr>;         // initialised to the value of <expr>
	def <name>, <name>;          // comma allows multiple definitions
	def <name> = <expr>, <name>; // different forms may be mixed freely

Variables may refer to previously defined variables in their initialisation
section. 

Locally defined variables shadow previous definitions.

e.g.

	def a = 1;
	def f()
		def a = 2;
		return a;
	end;
	print(f(), a);

This prints "2 1".

Once a variable has been defined, it may be updated so that the variable refers
to a new value.

	<name> = <expr>;

So if we alter the previous example, removing the internal definition, "a" is
altered in place rather than shadowed.

	def a = 1;
	def f()
		a = 2; // <--- no longer has "def" in front of it
		return a;
	end;
	print(f(), a);

This prints "2 2".

Functions

Functions are an important part of TranScript. They are first-class and are 
properly tail-recursive.

To call a function, take the expression that evaluates to the function and
append expressions enclosed in "(" and ")" and separated by ",". These 
expressions are evaluated before they are passed to the function. They are then 
bound to the corresponding arguments in the function's environment, and the 
function's body is evaluated in this new environment.

	<function>(<expr>, ...)

Here <function> means an expression that evaluates to a function.

e.g.

	print(5);

Prints "5".

Functions can be defined in two ways.

Named functions:

	def <name>(<args>) <body>

Anonymous functions:

	fn (<args>) <body>

<args> is a series of names separated by "," and represents the arguments to 
the function.

An argument may be designated as optional by appending "?" to the
name. All the optional arguments must appear after all the normal arguments. If
the argument isn't provided when the function is called its value is set to 
"false".

e.g.

	def fprint(x, f?)
		if f then
			f.writeString(x.toString() + "\n");
		else
			print(x);
		end;
	end;

If the final argument is followed by "*" then it is a "rest argument"
and represents any parameters passed to the function that are not caught by
previous argument names (optional or otherwise). This is encoded as an array.

e.g.

	def compose(f, g, more?, rest*)
		if more then
			g = compose.apply([g, more] + rest);
		end;
		return fn(x) = f(g(x));
	end;

<body> is either "=" followed by an expression, or a block terminated with 
"end".

i.e. for anonymous functions

	fn (<args>) = <expr>
	fn (<args>) <block> end

In the former case, it is as if

	fn (<args>) return <expr>; end

had been written instead.

Functions may return values using the "return" statement.

	return <expr>

This causes the function to return whatever the provided expression evaluates 
to. It returns straightaway, so no further statements are evaluated for that 
call. If the expression representing the return value is a function call, the 
caller's environment is cleared up before the call is made. This means that 
recursion in this case does not grow the stack.

You can exploit this property to make iterative processes using the syntax of
recursion.

e.g.

	def for(i, t, f)
		if i < t then
			f(i);
			return for(i+1, t, f);
		end;
	end;
	for(0, 10, fn(i) = print(i));

Prints "0" to "9" on consecutive lines. When "for" calls itself, the stack 
remains where it is. This is exactly equivalent to a "for loop" in many other
languages, but using function calls rather than special syntax.

A function has access to its enclosing scope. This includes definitions that
follow the function's definition. In the case of internal functions, this is
true even when the enclosing function has returned.

e.g.

	def accumulate(x)
		return fn(y)
			x = x + y;
			return x;
		end;
	end;

This implements a function that, when called, returns another function. This 
inner function refers to "x" after "accumulate" has returned. This is what some 
people call "closures," others call "lexical scope."

	def a = accumulate(1);
	a(1);                   // 2
	a(4);                   // 6

Objects and Classes

Objects are collections of slots. They support three basic operations: property
get; property set; method call.

	a.prop       // get prop
	a.prop = x   // set prop to x
	a.method()   // call method

Objects are instances of classes, which describe them. To create an object, call
its class like a function:

	def o = Object();

Here, "Object" is the class that will be used to instantiate the object. This
may be any expression that evaluates to a class.

To define a class, use the "class" keyword

	class <class name>(<ancestor>)
		<class body>
	end;

The <ancestor> is an expression that evaluates to a class, and is used for 
inheritance. Any members defined on the ancestor are also defined on this class.
It may be omitted, in which case "Object" is used.

The <class body> is a series of "def" statements punctuated with ";".
Definitions that look like variables correspond to properties and definitions
that look like functions correspond to methods.

e.g.

	class Vector()
		def x, y;
		def create(x, y)
			this.x = x;
			this.y = y;
		end;
		def length()
			return sqrt(this.x*this.x + this.y*this.y);
		end;
	end;

The "create" method is the constructor. This is called immediately after the
"Vector" object has been created. More precisely, "__new__" is called, which
calls "create" and then returns the new object.

Properties

Properties provide access to storage on objects. They are typically defined and
accessed as variables are.

Properties can also be defined with pairs of methods, a getter and a setter.
When the property is read from, the getter is called with zero parameters. Its 
return value is taken to be the value of the property. When the property is 
written to, the setter is called with one parameter, the value being assigned to 
that property.

To define a property, use "get" and "set" in a definition:

	def <name> get() <body> set(<val>) <body>;

Note that only one of "get" and "set" need to be present. If "set" is absent,
the property is read-only. If "get" is absent, the property is write-only.

Methods

Methods are functions that are attached to classes. When in the body of a method 
an extra variable is available, "this", which represents the object the method
is being called on. There is also "super", which represents "this" without the
current class' overrides.

e.g.

	1.toString()     // this set to 1 in call to toString()
	a.b.c(d);        // this set to b in call to c()

Methods will "remember" the "this" value when accessed as properties.

e.g.

	def f = 1.toString;
	print(f());

Prints "1".

Packages and Programs

Evaluation of programs is controlled by the client of the library. In the
default interpreter (https://github.com/bobappleyard/tsi) programs are a series
of top-level statements that are evaluated in the order they appear in the 
source file. A different approach may be preferred if TranScript is being used 
to extend a program.

Packages provide a mechanism for collecting code together into a namespace so
that it may be re-used by programs. To use a package, use the "import"
statement.

	import <package names>

Here <package names> consists of a list of <package name> separated with commas,
where <package name> is a list of <name> separated with dots.

e.g.

	import system;
	import my.big.long.package.name;
	import a, b, system; 

This creates a variable in the scope that the "import" appears, and is an object
containing all the exported definitions of the package as attributes.

To define a package, use the "package" statemtent.

	package <package name>
		<package body>
	end;

Here <package body> is a <body> where "export" statements may appear.

	export <names>

Where <names> is a list of <name> separated with commas. These should refer to
variables and functions defined within the package. Anything that has not been
exported is private to the package.

Packages need to be placed in the package path to be visible to the system. This
defaults to

	$(GOROOT)/src/pkg/github.com/bobappleyard/ts/pkg

but may be changed. Note that multiple paths separated by ":" may be used.

The directory containing the source file issuing the import request will also
be searched.

Packages are only loaded and evaluated once during the lifetime of the
interpreter.

*/
package ts
