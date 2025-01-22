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

	"gitlab.com/iglou.eu/goulc/duration"
)

// Config represents a sample configuration structure that uses
// duration types for duration fields
type Config struct {
	Timeout         duration.Duration `json:"timeout"`
	RefreshInterval duration.Duration `json:"refreshInterval"`
}

func main() {
	// Marshal/Unmarshal JSON with Duration
	jsonData := `{
		"timeout": "30s",
		"refreshInterval": "5m"
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
		Timeout:         duration.Duration{Duration: 45 * 1e9},      // 45 seconds
		RefreshInterval: duration.Duration{Duration: 10 * 60 * 1e9}, // 10 minutes
	}

	altJson, err := json.MarshalIndent(alternativeConfig, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal alternative config: %v", err)
	}

	fmt.Printf("\nAlternative configuration:\n%s\n", string(altJson))
}
