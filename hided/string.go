package hided

import (
	"crypto/md5"
	"encoding/hex"
)

// String holds sensitive data and implements obfuscation
type String string

// String implements fmt.Stringer to return an obfuscated string
func (s String) String() string {
	return "***"
}

// HashMD5 returns an MD5 hash of the string for obfuscation comparison
// Note: MD5 is used solely for obfuscation, not for security
func (s String) HashMD5() string {
	hash := md5.Sum([]byte(s))
	return hex.EncodeToString(hash[:])
}

// Value returns the underlying string value
func (s String) Value() any {
	return string(s)
}
