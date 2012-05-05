TranScript Manual
=================

TranScript is a dynamic object oriented language. It is designed to avoid some
of the major shortcomings of existing dynamic languages, particularly with
respect to performance.

Contents
========

* [Getting Started](#getting-started) 
* [Using The Interpreter](#using-the-interpreter)
* [The TranScript Language](#the-transcript-language)
* [The TranScript Library](#the-transcript-library)

Getting Started
===============

In a terminal as root, type

	go get -u github.com/bobappleyard/tsi

This will install the interpreter and associated packages.

Using The Interpreter
=====================

To run the interpreter, in a terminal type

	tsi

The interpreter comes in two basic modes:

* Script -- Executes a TranScript program. This program may be in source form 
or in binary form.

* Prompt -- The interpreter compiles an evaluates expressions as they are
entered into a terminal. Useful for finding out about the system.

Script Files
------------

Script files are text files whose names end in ".bs". They contain a series of
toplevel statements that are to be executed by the runtime, each terminated
with `;`.

Running the interpreter with a file as its first argument treats that file as
as TranScript program and runs it.

	tsi <script>

By default, the interpreter will quit when the script has finished running. 
Putting `-p` in at the start forces the prompt to appear when the program has 
finished.

	tsi -p <script>

The interpreter provides a variable named "args" containing the name of the 
script followed by the arguments to the script. Those arguments come after the
name of the script on the command line. 

Running the interpreter with no parameters gives a prompt. 

	tsi

Using The Compiler
------------------

Using the `-c` command line switch invokes the TranScript compiler. Pass in the 
source files that you want compiled.

	tsi -c <input files> ...

By default, the target file is the name of the first source file with `c`
appended. For instance `./bsi -c test.bs` will use `test.bsc` as the target
file. Override this using `-o`.

	tsi -c -o <output file> <input files> ...

Environment Variables
---------------------

The interpreter assumes that the go installation is at /usr/local/go, and that
the interpreter is installed there. However, if GOROOT is set then the 
interpreter will use that. This behaviour can be overrode by setting TSROOT,
specifying exactly where to find the TranScript installation.

The Transcript Language
=======================

TranScript is like many object-oriented languages in that everything is an
object. Functions are objects too, and have full lexical scope and tail
recursion. Objects are described by classes, which are themselves objects.

Because of this, the types described in this section have extra capabilities
that are described in the library section.

Basic Syntax
------------

Comments are like C++

	// line comment
	/* block comment */


Numbers are represented as series of digits, optionally separated with `.`
and/or preceded with `-`.

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

Supported operations are `+`, `-`, `*`, `/`, `==`, `!=`, `<`, `>`, `<=`, `>=`.

Names follow the C syntax convention: Letters or `_` to begin, letters, digits
or `_` after that.

	foo
	Object
	foo_bar

Names refer to things. They might refer to a definition, or they might refer to
a member on an object.

	a               // definition
	a.prop          // member

There are two boolean values, `true` and `false`. 

Booleans support one operation, negation.

	!true           // false
	!false          // true

This is actually supported by every other object in TranScript as well. For
every value other than `false`, `false` is returned. For `false`, `true` is
returned.

The logical operators `&&` (and) and `||` (or) are provided.

	true && true    // true
	false || true   // true
	false && true   // false
	false || false  // false

These operator have short-circuiting. This means that if the value of the 
expression can be determined after evaluating the left operand, the right 
operand is not evaluated.

There is `nil`, which represents no value. This is primarily used as the return 
value of functions and methods that are only called for their side effects.

Strings have the usual C-like syntax: enclosed in `"`, with `\` for escape 
sequences.

	"Hello, world!\n"

Two strings may be concatenated using `+`.

	"Hello, " + "world!\n" // gives the same value as before

Arrays are series of objects enclosed in brackets and separated with `,`.

	[1, 2, 3]

The familiar subscript access is available.

	array[0]      // get the 0th member
	array[1] = 2  // set the first member to 2

Arrays may be concatenated as strings are.

	[1, 2] + [3, 4] // [1, 2, 3, 4]

Hashes are key-value pairs enclosed in curly brackets and separated with `,`.
In each pair, the key is separated from the value by `:`.

	{"key1": "value1", "key2": "value2"}

Any object may be a key or a value. Only strings and numbers are likely to be
useful keys most of the time, though.

Control Structures
------------------

A block is a series of statements, each punctuated with a `;`.

	a = 1;
	b = 2;
	print(a + b);

Blocks have their own scope. This is described in more detail below.

An if statement evaluates an expression. If it evaluates to `true` then `if` 
evalutates its `then` block. If the expression evaluates to `false` then `if` 
evaluates its else block. Every value other than `false` is counted as `true` to
an `if` statement.

	if <expression> then <block> else <block> end

e.g.

	def a = 1;
	if a < 5 then
		print("a is less than five");
	end;
	if a > 5 then
		print("a is greater than five");
	end;

This prints `a is less than five`.

Variables And Scope
-------------------

Variables allow you to store state and refer to the results of expresssions.

Defining a variable looks like

	def <name>;           // uninitialised
	def <name> = <expr>;  // initialised to the value of <expr>
	def <name>, <name>;   // comma allows multiple definitions
	def <name> = <expr>, <name>; // different forms may be mixed freely

Variables may refer to previously defined variables in their initialisation
section. Local variables may be defined at the start of a block.

Locally defined variables shadow previous definitions.

e.g.

	def a = 1;
	def f()
		def a = 2;
		return a;
	end;
	print(f(), a);

This prints `2 1`.

Once a variable has been defined, it may be updated so that the variable refers
to a new value.

	<name> = <expr>;

So if we alter the previous example, removing the internal definition, `a` is
altered in place rather than shadowed.

	def a = 1;
	def f()
		a = 2; // <--- no longer has "def" in front of it
		return a;
	end;
	print(f(), a);

This now prints `2 2`.

Functions
---------

Functions are an important part of TranScript. They are first-class, have
lexical scope, and are properly tail-recursive.

Functions can be defined in two ways.

Named functions:

	def <name>(<args>) <body>

Anonymous functions:

	fn (<args>) <body>

<args> is a series of names separated by `,` and represents the arguments to 
the function.

<body> is either `=` followed by an expression, or a block terminated with 
`end`.

i.e. for anonymous functions

	fn (<args>) = <expr>
	fn (<args>) <block> end

In the former case, it is as if

	fn (<args>) return <expr>; end

had been written instead. See below for a description of what `return` does.

To call a function, append expressions enclosed in `(` and `)` and separated by
`,`. These expressions are evaluated in the caller's environment and passed to 
the function. They are then bound to the corresponding arguments in the callee's
environment, and the callee's body is evaluated in this new environment.

	<function>(<expr>, ...)

Here <function> means an expression that evaluates to a function.

e.g.

	def f(x)
		print(x);
	end;
	f(5);

Prints `5`.

Functions may return values using the `return` statement.

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
	for(0, 10, fn(i)
		print(i);
	end);

Prints `0` to `9` on consecutive lines. When `for` calls itself, the stack 
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
inner function refers to `x` after `accumulate` has returned. This is what some 
people call "closures," others call "lexical scope."

	def a = accumulate(1);
	a(1);                   // 2
	a(4);                   // 6

Objects
-------

Objects are collections of slots. There are three kinds of slot:

* Fields -- These provide storage on the object.

* Methods -- These are functions that are associated with the object.

* Properties -- These look like fields but call methods underneath.

Objects primarily support one operation, `.`.

	a.field      // reading a field/property
	a.field = x  // writing a field/property
	a.method()   // calling a method

Objects are instances of classes. The method `Object.is()` describes this
relationship.

	class X() end;
	print(new X().is(X));

Prints `true`.

Classes
-------

Classes describe objects. They support one operation, which is to construct an 
instance.

	def obj = new Object();

Here, `Object` is the class that will be used to instantiate the object. This
may be any expression, but if it involves function calls it should be
parenthesised, as they are otherwise caught by the syntax. This is so arguments
may be passed to the object's constructor, `create`.

To define a class, use the `class` keyword

	class <class name>(<ancestor>)
		<class body>
	end;

The <ancestor> is an expression that evaluates to a class, and is used for 
inheritance. Any members defined on the ancestor are also defined on this class.
It may be omitted, in which case `Object` is used.

The <class body> is a series of `def` statements punctuated with `;`.
Definitions that look like variables correspond to fields and definitions that
look like functions correspond to methods.

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

The `create` method is the constructor. This is called immediately after the
`Vector` object has been created. More precisely, `__new__` is called, which
calls `create` and then returns the new object.

Classes have access specifiers that control which members are visible where.
There are three kinds of access:

* Private -- The member is visible only to that class.

* Public -- The member is visible everywhere.

This is done using the `private` and `public` keywords.

Methods
-------

Methods are functions that are attached to classes. When in the body of a method 
an extra variable is available, `this`, which represents the object the method
is being called on.

e.g.

	1.toString()     // this set to 1 in call to toString()
	a.b.c(d);        // this set to b in call to c()

Methods will "remember" the object they were associated with. This means that
you can access methods as if they were fields, then later call them and `this`
will still be the object the method was on, no matter what happened in between.

e.g.

	def f = 1.toString;
	print(f());

Prints `1`.

Properties
----------

Properties are pairs of methods, a getter and a setter. The property is
accessed like a field. When the property is read from, the getter is called with 
zero parameters. Its return value is taken to be the value of the property. When
the property is written to, the setter is called with one parameter, the value
being assigned to that property.

To define a property, use `get` and `set` in a definition:

	def <name> get() <body> set(<val>) <body>;

Note that only one of `get` and `set` need to be present. If `set` is absent,
the property is read-only. If `get` is absent, the property is write-only.

The Transcript Library
======================

This is extremely small and underpowered. Most of the work so far has gone into
the basic behaviour of the language.

Before discussing the built-in classes, there are some toplevel functions
defined by the runtime. Many of these will disappear with time.

* **print(x...)

Print 0 or more objects to standard output.

* **read()

Read a line from standard input.

* **exit(code)	

Exit the process with the given code.

* **throw(x)	

Throw an error.

* **catch(thk)	

Catch an error: Call `thk`. If an error is thrown while `thk` is executing, 
return it. Otherwise return `false`.

Classes
-------

This is a list of built-in classes.

* [Object](#object)
* [Function](#function)
* [Number](#number)
* [String](#string)
* [Array](#array)
* [Hash](#hash)

Object
-------

The root class. All objects instantiate Object.

* **copy()**

Creates a copy of the object.

* **toString()**

The default method for string conversion/printing.

* **is(c)**

Test whether the object instantiates class `c`.

* **type()**

The object's class.

Function
---------

*final, primitive*

* **apply(args)**

Call the function, passing `args`, which should be an array containing the
intended arguments to the function.

CLASS

*final, primitive*

* **name()**

Returns the name of the class in question.

* **names()**

Returns the names of members this class defines.

* **allNames()**

Returns all the defined members. That is, it includes members defined in all
  of the class' ancestors.

Number
-------

*final, primitive*

There are two classes which descend from `Number`, `Integer` and `Float`. These
have the same interface as `Number` but actually implement it. `Number` is,
itself, absolutely useless.

* **toInt()**

Convert the number to an integer.

* **toFloat()**

Convert the number to a float.

String
-------

*final, primitive*

* **length()**

Returns the number of characters in the string.

* **split(sep?)**

Returns an array of substrings. If called with no parameters or the parameter
is an empty string, each substring represents a character from the string.
Otherwise the parameter is a delimiter, and each substring is made of
contiguous regions of the string that are not the delimiter.
  
e.g. 

	text.split("\n").map(fn(x) = x.split(",")).zip()

Parses a csv file.

* **subst(args*)**

Returns a string where occurrences of `%` are replaced with the corresponding
argument.

* **toInt()**

Convert the string to an integer.

* **toFloat()**

Convert the string to a float.

* **toNumber()**

Convert the string to a number.

* **startsWith(s)**

Does the string start with `s`?

* **endsWith(s)**

Does the string end with `s`?

* **contains(s)**

Does the string contain `s`?

* **match(e)**

Does the string match the regular expression `e`?

* **trim(s?)**

Remove any characters in `s` from the beginning and end of the string. If no 
parameter is provided, default to whitespace characters.

* **trimLeft(s?)**

Remove any characters in `s` from the beginning of the string. If no parameter 
is provided, default to whitespace characters.

* **trimRight(s?)**

Remove any characters in `s` from the end of the string. If no parameter is 
provided, default to whitespace characters.

* **quote()**

Returns the quoted representation of the string.

* **unquote()**

Given the string is a quoted representation, return the string that it 
represents.

Array
------

*final*

* **length()**

Returns the number of items in the array.

* **add(x*)**

Adds an item to the array.

* **remove(x)**

Removes all instances of an item from the array.

* **insert(i, x)**

Inserts an item to the array at the given index.

* **delete(i)**

Deletes an item to the array with the given index.

* **slice(from?, to?)**

Returns a section of the array starting at `from` and ending on the element
before `to`. This is an array that shares structure with the original array,
so changes to one are reflected in the other.

* **each(f)**

Calls `f` on every item in the array, passing in the index followed by the
item.

* **filter(f)**

Returns an array containing all the elements for which a call to `f` returns
a true value (anything but `false`).

* **map(f)**

Returns an array containing all the return values of calling `f` on each item
in the current array, in order.  
  
  i.e.
  
	def r = [1, 2, 3].map(fn(x) = x+5);
  
  is equivalent to
  
	def r = [];
	[1, 2, 3].each(fn(i, x)
		r.add(x+5);
	end);

	reduce(x, f)	

Apply `f` to every item in the array, passing the return value of the previous 
call into the next call. 

* **join(sep?)**

Given an array of strings, return a single string consisting of each item in
the array concatenated together, separated with `sep` (or "" if the parameter
is not present).

Hash
----

*final*

* **keys()**

Return all the keys on this hash.


s
