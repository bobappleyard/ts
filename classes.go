package ts

// The root class. All other classes descend from this one.
var ObjectClass *Class

// Primitive data types. The normal operations on classes (extension, creation)
// do not work on these classes.
var ClassClass, AccessorClass, NilClass, BooleanClass, TrueClass, FalseClass,
	CollectionClass, SequenceClass, IteratorClass, sequenceIteratorClass,
    StringClass, NumberClass, IntClass, FltClass, FunctionClass, ArrayClass,
    ErrorClass, BufferClass, PairClass,
    frameClass, skeletonClass, boxClass, undefinedClass *Class

/*

	class Hash(Collection)

A Hash maps keys to values.

For keys, the following rules hold:

	* If the key is a string or number, its value is used directly.
	* Otherwise the pointer value of the key is used.
	* If the key defines a __key__() method then this is called, and its return
	  value is used for the key.
	* If the return value from a call to __key__() equals another call to
	  __key__() but the receivers for the method calls have different classes
	  then the hash table will consider those keys as different.
*/
var HashClass *Class


