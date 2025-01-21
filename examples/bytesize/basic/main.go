package main

import (
	"fmt"
	"log"

	"gitlab.com/iglou.eu/goulc/bytesize"
)

func main() {
	// Parse a string representation of a byte size
	t, f, r, err := bytesize.Parse("42.42MiB")

	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Direct parsing:\n")
	fmt.Printf("Truncated: %d\n", t)
	fmt.Printf("Exact: %f\n", f)
	fmt.Printf("Representation: %s\n", r)

	// Create a new representation from an exact value
	r = bytesize.ToString(44480593.92)

	fmt.Printf("\nDirect from float:\n")
	fmt.Printf("Representation: %s\n", r)

	// Create a new Size from a string representation
	// We can also use NewInt to create a new Size from a raw byte count
	s, err := bytesize.New("42.42MiB")

	if err != nil {
		log.Fatal(err)
	}

	// Print some ressources
	fmt.Printf("\nNew representation:\n")
	fmt.Printf("Truncated: %d\n", s.Bytes())
	fmt.Printf("Exact: %f\n", s.Exact())
	fmt.Printf("Representation: %s\n", s.String())

	// Add some sizes
	s.Add("42.42MiB")
	s.AddInt(42)

	fmt.Printf("\nAdd some sizes:\n")
	fmt.Printf("Truncated: %d\n", s.Bytes())

	// Now some subtraction
	s.Add("-42.42MiB")
	s.AddInt(-42)

	fmt.Printf("\nSubtract some sizes:\n")
	fmt.Printf("Truncated: %d\n", s.Bytes())

}
