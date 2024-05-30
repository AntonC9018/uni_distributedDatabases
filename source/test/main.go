package main

import (
	"fmt"
	"reflect"
	"unsafe"
)

func main() {
    var x = 0xFFFFFFFF
    var px = &x
    var iValue interface{} = x
    var iPointer interface{} = px
    var iValueFromPointer interface{} = *(iPointer.(*int))
    fmt.Printf("Address of x: %x\n", &x)
    fmt.Printf("Address of px: %x\n", &px)
    fmt.Printf("Value of x: %x\n", x)
    printInterfaceValueAsBytes(reflect.TypeOf(iValue), "TypeOf(iValue)")
    printInterfaceValueAsBytes(iValue, "iValue")
    printInterfaceValueAsBytes(reflect.TypeOf(iPointer), "TypeOf(iPointer)")
    printInterfaceValueAsBytes(iPointer, "iPointer")
    printInterfaceValueAsBytes(iValueFromPointer, "iValueFromPointer")
}

func printInterfaceValueAsBytes(i interface{}, varName string) {
	rawPtr := unsafe.Pointer(&i)

	byteSlice := (*[unsafe.Sizeof(i)]byte)(rawPtr)[:]
    integerSlice := BytesToInts(byteSlice)

	fmt.Printf("Value stored in %s as ints: ", varName)
    for _, i := range integerSlice {
        fmt.Printf("%8x ", i)
    }
    fmt.Println()
}

func BytesToInts(b []byte) []int {
    intLen := int(unsafe.Sizeof(int(0)))
	if len(b) % intLen != 0 {
		panic("[]byte length must be a multiple of the size of int")
	}

    startPtrAsInt := (*int)(unsafe.Pointer(&b[0]))
    return unsafe.Slice(startPtrAsInt, len(b) / intLen)
}
