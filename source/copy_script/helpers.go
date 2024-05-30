package main

import (
	"log"
	"reflect"
	"unsafe"
)

func CopyFieldPointers(ptr interface{}, output []interface{}) []interface{} {
    valueInfo := reflect.ValueOf(ptr)
    numFields := valueInfo.Elem().Type().NumField()
    if numFields > len(output) {
        log.Fatalf("The output array should have at least %d elements, got %d", numFields, len(output))
    }
    for i := 0; i < numFields; i++ {
        ptr1 := valueInfo.Elem().Field(i).Addr().Interface()
        output[i] = ptr1
    }
    return output[0 : numFields]
}

type interfaceMemory struct {
    mem [2]unsafe.Pointer
}

func inspect(i interface{}) interfaceMemory {
    return *(*interfaceMemory)(unsafe.Pointer(&i))
}

func (i interfaceMemory) toInterface() interface{} {
    return *(*interface{})(unsafe.Pointer(&i.mem))
}

func (i *interfaceMemory) value() *unsafe.Pointer {
    return &i.mem[1]
}

func (i *interfaceMemory) typeInfo() *unsafe.Pointer {
    return (*unsafe.Pointer)(&i.mem[0])
}

// This one might break in the future and involves some gross hacks.
func ValuesFromPointers(pointers []interface{}, output []interface{}) {
    if len(pointers) != len(output) {
        panic("Lengths of input and output must be equal")
    }

    for i, p := range pointers {
        elem := reflect.TypeOf(p).Elem()
        elemType := inspect(elem)
        pointer := inspect(p)

        var v interfaceMemory
        *v.value() = *pointer.value()
        // The address stored here seems to be the exact same it stores for the type info.
        *v.typeInfo() = *elemType.value()

        output[i] = v.toInterface();
    }
}

