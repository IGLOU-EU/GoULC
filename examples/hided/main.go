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

	// Convert a sensitive string into a "hided" string that masks its content
	myHidedSecretKey := hided.String("Haha, this time, i'm hided!")
	// Print a masked version of the secret key
	fmt.Println("Batcode for the Batcave:", myHidedSecretKey)
	// Retrieve and print the MD5 hash for debugging without revealing the actual key
	fmt.Println("DEBUG: I need to know if the code is correct, but without displaying it:", myHidedSecretKey.HashMD5())
	// Extract the original value safely when necessary
	fmt.Printf("I need to use it! batcavePass(%v) \n", myHidedSecretKey.Value())
}
