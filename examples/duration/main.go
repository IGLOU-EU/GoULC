/*
 * Copyright 2025 Adrien Kara
 *
 * Licensed under the GNU General Public License v3.0
 */

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"gitlab.com/iglou.eu/goulc/duration"
)

// Config represents a sample configuration structure that uses
// duration types for duration fields
type Config struct {
	Timeout         duration.Duration `json:"timeout"`
	RefreshInterval duration.Duration `json:"refreshInterval"`
}

func main() {
	// Marshal/Unmarshal JSON with Duration. A bare JSON number is a count
	// of nanoseconds, the unit of time.Duration itself (here 5 minutes).
	jsonData := `{
		"timeout": "30s",
		"refreshInterval": 300000000000
	}`

	var config Config
	if err := json.Unmarshal([]byte(jsonData), &config); err != nil {
		log.Fatalf("Failed to unmarshal config: %v", err)
	}

	// Print the parsed values
	fmt.Printf("Parsed configuration:\n")
	fmt.Printf("Timeout: %v\n", config.Timeout.ToTimeDuration())
	fmt.Printf("Refresh Interval: %v\n", config.RefreshInterval.ToTimeDuration())

	// Marshal back to JSON
	newJson, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal config: %v", err)
	}

	fmt.Printf("\nMarshaled back to JSON:\n%s\n", string(newJson))

	// Different ways to specify values
	alternativeConfig := Config{
		Timeout:         duration.New(45 * time.Second),
		RefreshInterval: duration.New(10 * time.Minute),
	}

	altJson, err := json.MarshalIndent(alternativeConfig, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal alternative config: %v", err)
	}

	fmt.Printf("\nAlternative configuration:\n%s\n", string(altJson))
}
