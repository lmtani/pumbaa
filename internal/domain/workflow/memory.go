// Package workflow contains the domain entities and business logic for workflows.
package workflow

import (
	"regexp"
	"strconv"
	"strings"
)

// Memory is a Value Object representing a memory configuration.
// It encapsulates the parsing logic for memory strings like "8 GB", "512 MB".
type Memory string

// memoryRegex matches patterns like "1 GB", "512 MB", "2GB", "4gb"
var memoryRegex = regexp.MustCompile(`(?i)^(\d+(?:\.\d+)?)\s*(GB|MB|KB|TB|GiB|MiB|KiB|TiB|G|M|K|T)?$`)

// ToBytes converts the memory configuration to bytes.
// Returns 0 if the format is invalid.
func (m Memory) ToBytes() int64 {
	if m == "" {
		return 0
	}

	matches := memoryRegex.FindStringSubmatch(strings.TrimSpace(string(m)))
	if matches == nil {
		return 0
	}

	value, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return 0
	}

	unit := strings.ToUpper(matches[2])
	var multiplier float64 = 1

	switch unit {
	case "TB", "TIB", "T":
		multiplier = 1024 * 1024 * 1024 * 1024
	case "GB", "GIB", "G", "":
		multiplier = 1024 * 1024 * 1024
	case "MB", "MIB", "M":
		multiplier = 1024 * 1024
	case "KB", "KIB", "K":
		multiplier = 1024
	}

	return int64(value * multiplier)
}

// ToMB converts the memory configuration to megabytes.
func (m Memory) ToMB() float64 {
	return float64(m.ToBytes()) / (1024 * 1024)
}

// ToGB converts the memory configuration to gigabytes.
func (m Memory) ToGB() float64 {
	return float64(m.ToBytes()) / (1024 * 1024 * 1024)
}

// IsValid returns true if the memory string has a valid format.
func (m Memory) IsValid() bool {
	if m == "" {
		return false
	}
	return memoryRegex.MatchString(strings.TrimSpace(string(m)))
}

// String returns the original memory string.
func (m Memory) String() string {
	return string(m)
}
