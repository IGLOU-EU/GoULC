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

	// Extended ASCII check, rune oriented: accepts valid UTF-8 whose code
	// points fit in Latin-1, rejects raw Latin-1 bytes
	fmt.Println(ascii.IsExtended("Pokémon"))     // true
	fmt.Println(ascii.IsExtended("Hello 👋"))     // false
	fmt.Println(ascii.IsExtended("caf\xc3\xa9")) // true, UTF-8 encoded é
	fmt.Println(ascii.IsExtended("caf\xe9"))     // false, invalid UTF-8

	// Null byte detection
	fmt.Println(ascii.HasNil("Hello\x00World")) // true
}
