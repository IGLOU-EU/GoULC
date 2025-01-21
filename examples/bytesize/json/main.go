package main

import (
	"encoding/json"
	"fmt"
	"log"

	"gitlab.com/iglou.eu/goulc/bytesize"
)

type Config struct {
	MaxFileSize bytesize.Size `json:"maxFileSize"`
	BufferSize  bytesize.Size `json:"bufferSize"`
}

func main() {
	// Json data
	data := `{
		"maxFileSize": "2.5G",
		"bufferSize": "1MiB"
	}`

	// Unmarshal json data
	var config Config
	if err := json.Unmarshal([]byte(data), &config); err != nil {
		log.Fatalf("Failed to unmarshal config: %v", err)
	}

	// Print the parsed values
	fmt.Printf("Parsed configuration:\n")
	fmt.Printf("Max File Size: %v bytes; Exact: %v; Representation: %v\n", config.MaxFileSize.Bytes(), config.MaxFileSize.Exact(), config.MaxFileSize)
	fmt.Printf("Buffer Size: %v bytes; Exact: %v; Representation: %v\n", config.BufferSize.Bytes(), config.BufferSize.Exact(), config.BufferSize)

	// Marshal back to json
	newJson, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal config: %v", err)
	}

	fmt.Printf("\nMarshaled back to JSON:\n%s\n", string(newJson))

	// Different ways to specify values
	fmt.Printf("\nDifferent ways to specify values:\n")
	fmt.Printf("Max File Size: %v bytes; Exact: %v; Representation: %v\n", bytesize.NewInt(256*bytesize.Gibi).Bytes(), bytesize.NewInt(256*bytesize.Gibi).Exact(), bytesize.NewInt(256*bytesize.Gibi))
}
