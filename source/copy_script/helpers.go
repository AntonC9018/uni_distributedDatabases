package main

import (
	"log"
	"reflect"
	"unsafe"
)

func copyFieldPointers(ptr interface{}, output []interface{}) []interface{} {
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

type InterfaceMemory struct {
    mem [2]unsafe.Pointer
}

func Inspect(i interface{}) InterfaceMemory {
    return *(*InterfaceMemory)(unsafe.Pointer(&i))
}

func (i InterfaceMemory) ToInterface() interface{} {
    return *(*interface{})(unsafe.Pointer(&i.mem))
}

func (i *InterfaceMemory) Value() *unsafe.Pointer {
    return &i.mem[1]
}

func (i *InterfaceMemory) TypeInfo() *unsafe.Pointer {
    return (*unsafe.Pointer)(&i.mem[0])
}

// This one might break in the future and involves some gross hacks.
func valuesFromPointers(pointers []interface{}, output []interface{}) {
    if len(pointers) != len(output) {
        panic("Lengths of input and output must be equal")
    }

    for i, p := range pointers {
        elemType := reflect.TypeOf(p).Elem()
        elem := Inspect(elemType)
        pointer := Inspect(p)

        var v InterfaceMemory
        *v.Value() = *pointer.Value()
        *v.TypeInfo() = *elem.Value()

        output[i] = v.ToInterface();

        // output[i] = reflect.ValueOf(p).Elem().Interface()
    }
}

