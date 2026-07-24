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

	// Value() and Reveal() are the ONLY ways to access the real value
	fmt.Printf("I need to use it! batcavePass(%v) \n", myHidedSecretKey.Value())

	// Reveal() returns the plaintext as a typed string when you statically
	// hold a hided.String, no `any` detour, no assertion.
	fmt.Printf("Direct access: batcavePass(%s)\n", myHidedSecretKey.Reveal())

	// hided.Value[T] is a type-safe alternative to Value() + type assertion
	// for values only known as the Hider interface. It returns the zero
	// value of T on a type mismatch (or nil Hider) instead of panicking.
	var anyHider hided.Hider = myHidedSecretKey
	batcode := hided.Value[string](anyHider)
	fmt.Printf("Type-safe access: batcavePass(%s)\n", batcode)
}
