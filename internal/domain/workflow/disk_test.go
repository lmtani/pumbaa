package workflow

import "testing"

func TestDiskConfig_SizeBytes(t *testing.T) {
	tests := []struct {
		input         string
		expectedBytes int64
		expectedType  string
	}{
		{"local-disk 100 HDD", 100 * 1024 * 1024 * 1024, "HDD"},
		{"local-disk 13 SSD", 13 * 1024 * 1024 * 1024, "SSD"},
		{"local-disk 31 HDD", 31 * 1024 * 1024 * 1024, "HDD"},
		{"local-disk 2 SSD", 2 * 1024 * 1024 * 1024, "SSD"},
		{"LOCAL-DISK 50 ssd", 50 * 1024 * 1024 * 1024, "SSD"}, // Case insensitive
		{"", 0, ""},               // Empty string
		{"100 GB", 0, ""},         // Invalid format (no local-disk prefix)
		{"local-disk HDD", 0, ""}, // Missing size
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			d := NewDiskConfig(tt.input)
			if d.SizeBytes() != tt.expectedBytes {
				t.Errorf("DiskConfig(%q).SizeBytes() = %d, expected %d", tt.input, d.SizeBytes(), tt.expectedBytes)
			}
			if d.Type() != tt.expectedType {
				t.Errorf("DiskConfig(%q).Type() = %q, expected %q", tt.input, d.Type(), tt.expectedType)
			}
		})
	}
}

func TestDiskConfig_SizeGB(t *testing.T) {
	d := NewDiskConfig("local-disk 100 SSD")
	if d.SizeGB() != 100 {
		t.Errorf("DiskConfig.SizeGB() = %d, expected %d", d.SizeGB(), 100)
	}
}

func TestDiskConfig_IsValid(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"local-disk 100 SSD", true},
		{"local-disk 50 HDD", true},
		{"", false},
		{"100 GB", false},
		{"local-disk HDD", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			d := NewDiskConfig(tt.input)
			if d.IsValid() != tt.expected {
				t.Errorf("DiskConfig(%q).IsValid() = %v, expected %v", tt.input, d.IsValid(), tt.expected)
			}
		})
	}
}

func TestDiskConfig_String(t *testing.T) {
	input := "local-disk 100 SSD"
	d := NewDiskConfig(input)
	if d.String() != input {
		t.Errorf("DiskConfig.String() = %q, expected %q", d.String(), input)
	}
}
