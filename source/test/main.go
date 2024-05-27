package main

import (
	"fmt"
	"unsafe"
)

func main() {
    // Primitive values are boxed
    {
        var x = 0xFFFFFFFF
        var i interface{} = x
        fmt.Printf("Address of x: %x\n", &x)
        fmt.Printf("Value of x: %x\n", x)
        printInterfaceValueAsBytes(i)
    }
    // Pointers don't seem to be boxed
    {
        var x = 0xFFFFFFFF
        var i interface{} = &x
        fmt.Printf("Address of x: %x\n", &x)
        fmt.Printf("Value of x: %x\n", x)
        printInterfaceValueAsBytes(i)
    }
}

func printInterfaceValueAsBytes(i interface{}) {
	// Get the raw pointer to the interface data
	rawPtr := unsafe.Pointer(&i)

	// Reinterpret the memory pointed to by rawPtr as a byte slice
	byteSlice := (*[unsafe.Sizeof(i)]byte)(rawPtr)[:]

	// Print the byte slice
	fmt.Printf("Value stored in i as []byte: %x\n", byteSlice)
}
