package main

import (
	"fmt"

	"gitlab.com/iglou.eu/goulc/hided"
)

func main() {
	// A secret key that is visible; example of a logging leakage issue
	mySecretKey := "ho no ! A logging leakage!"
	// Print the secret key directly... Ooops
	fmt.Printf("Try to connect to Batman with secret key: %s\n\n", mySecretKey)

	// Create a hided string using the constructor
	myHidedSecretKey := hided.NewString("Haha, this time, i'm hided!")

	// All fmt verbs produce "***", the value never leaks
	fmt.Println("Batcode for the Batcave:", myHidedSecretKey)
	fmt.Printf("  %%s  = %s\n", myHidedSecretKey)
	fmt.Printf("  %%v  = %v\n", myHidedSecretKey)
	fmt.Printf("  %%q  = %q\n", myHidedSecretKey)
	fmt.Printf("  %%+v = %+v\n", myHidedSecretKey)
	fmt.Printf("  %%#v = %#v\n\n", myHidedSecretKey)

	// Retrieve and print the MD5 hash for debugging without revealing the actual key
	fmt.Println("DEBUG: I need to know if the code is correct, but without displaying it:", myHidedSecretKey.HashMD5())

	// Value() is the ONLY way to access the real value
	fmt.Printf("I need to use it! batcavePass(%v) \n", myHidedSecretKey.Value())

	// hided.Value[T] is a type-safe alternative to Value() + type assertion.
	// It returns the zero value of T on a type mismatch instead of panicking.
	batcode := hided.Value[string](myHidedSecretKey)
	fmt.Printf("Type-safe access: batcavePass(%s)\n", batcode)
}
