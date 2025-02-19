//go:build gorm

package hided

import "fmt"

// GormHider defines an interface for secure string representations
type GormHider interface {
	// String returns a clear string representation
	fmt.Stringer

	// Hiding returns the obfuscated string (expected output: "***")
	Hiding() string
}
