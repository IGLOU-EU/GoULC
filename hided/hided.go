// Package hided provides types and methods to obfuscate or mask sensitive data
// It can be used to ensure that sensitive information is not exposed in logs or any outputs
package hided

import "fmt"

// Hider defines types that can be obfuscated
type Hider interface {
	// String returns the obfuscated string (expected output: "***")
	fmt.Stringer

	// HashMD5 returns an MD5 hashed representation for obfuscation comparison
	// Note: MD5 is used only for obfuscation, not for cryptographic security
	HashMD5() string

	// Value returns the underlying value
	Value() any
}
