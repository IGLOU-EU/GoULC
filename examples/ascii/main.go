package main

import (
	"fmt"

	"gitlab.com/iglou.eu/goulc/ascii"
)

func main() {
	// Standard ASCII check
	fmt.Println(ascii.Is("Hello World")) // true
	fmt.Println(ascii.Is("Pokémon"))     // false

	// Printable ASCII check, control characters including DEL are rejected
	fmt.Println(ascii.IsPrintable("Hello!"))    // true
	fmt.Println(ascii.IsPrintable("Hello\n"))   // false
	fmt.Println(ascii.IsPrintable("Hello\x7f")) // false

	// Extended ASCII detection: true when any byte is in the 128-255 range
	fmt.Println(ascii.IsExtended("Hello"))   // false
	fmt.Println(ascii.IsExtended("Pokémon")) // true
	fmt.Println(ascii.IsExtended("caf\xe9")) // true

	// Null byte detection
	fmt.Println(ascii.HasNil("Hello\x00World")) // true
}
