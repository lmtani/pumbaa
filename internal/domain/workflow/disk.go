// Package workflow contains the domain entities and business logic for workflows.
package workflow

import (
	"regexp"
	"strconv"
	"strings"
)

// DiskConfig is a Value Object representing a disk configuration.
// It encapsulates the parsing logic for disk strings like "local-disk 100 SSD".
type DiskConfig struct {
	raw      string
	sizeGB   int64
	diskType string
}

// diskConfigRegex matches patterns like "local-disk 31 HDD", "local-disk 100 SSD"
var diskConfigRegex = regexp.MustCompile(`(?i)local-disk\s+(\d+)\s+(\w+)`)

// NewDiskConfig creates a DiskConfig from a configuration string.
func NewDiskConfig(config string) DiskConfig {
	d := DiskConfig{raw: config}

	if config == "" {
		return d
	}

	matches := diskConfigRegex.FindStringSubmatch(strings.TrimSpace(config))
	if matches == nil {
		return d
	}

	sizeGB, err := strconv.ParseInt(matches[1], 10, 64)
	if err != nil {
		return d
	}

	d.sizeGB = sizeGB
	d.diskType = strings.ToUpper(matches[2])
	return d
}

// SizeBytes returns the disk size in bytes.
func (d DiskConfig) SizeBytes() int64 {
	return d.sizeGB * 1024 * 1024 * 1024
}

// SizeGB returns the disk size in gigabytes.
func (d DiskConfig) SizeGB() int64 {
	return d.sizeGB
}

// Type returns the disk type (e.g., "SSD", "HDD").
func (d DiskConfig) Type() string {
	return d.diskType
}

// IsValid returns true if the disk configuration has a valid format.
func (d DiskConfig) IsValid() bool {
	return d.sizeGB > 0 && d.diskType != ""
}

// String returns the original disk configuration string.
func (d DiskConfig) String() string {
	return d.raw
}
