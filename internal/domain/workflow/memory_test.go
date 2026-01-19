package workflow

import "testing"

func TestMemory_ToBytes(t *testing.T) {
	tests := []struct {
		input    Memory
		expected int64
	}{
		{"1 GB", 1 * 1024 * 1024 * 1024},
		{"8 GB", 8 * 1024 * 1024 * 1024},
		{"14 GB", 14 * 1024 * 1024 * 1024},
		{"512 MB", 512 * 1024 * 1024},
		{"1024 MB", 1024 * 1024 * 1024},
		{"1 TB", 1 * 1024 * 1024 * 1024 * 1024},
		{"2GB", 2 * 1024 * 1024 * 1024},   // No space
		{"4gb", 4 * 1024 * 1024 * 1024},   // Lowercase
		{"1 GiB", 1 * 1024 * 1024 * 1024}, // GiB variant
		{"", 0},                           // Empty string
		{"invalid", 0},                    // Invalid format
		{"GB", 0},                         // No number
	}

	for _, tt := range tests {
		t.Run(string(tt.input), func(t *testing.T) {
			result := tt.input.ToBytes()
			if result != tt.expected {
				t.Errorf("Memory(%q).ToBytes() = %d, expected %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestMemory_ToMB(t *testing.T) {
	m := Memory("1 GB")
	expected := 1024.0
	if m.ToMB() != expected {
		t.Errorf("Memory(%q).ToMB() = %f, expected %f", m, m.ToMB(), expected)
	}
}

func TestMemory_ToGB(t *testing.T) {
	m := Memory("8 GB")
	expected := 8.0
	if m.ToGB() != expected {
		t.Errorf("Memory(%q).ToGB() = %f, expected %f", m, m.ToGB(), expected)
	}
}

func TestMemory_IsValid(t *testing.T) {
	tests := []struct {
		input    Memory
		expected bool
	}{
		{"8 GB", true},
		{"512 MB", true},
		{"1TB", true},
		{"", false},
		{"invalid", false},
		{"GB", false},
	}

	for _, tt := range tests {
		t.Run(string(tt.input), func(t *testing.T) {
			if tt.input.IsValid() != tt.expected {
				t.Errorf("Memory(%q).IsValid() = %v, expected %v", tt.input, tt.input.IsValid(), tt.expected)
			}
		})
	}
}

func TestMemory_String(t *testing.T) {
	m := Memory("8 GB")
	if m.String() != "8 GB" {
		t.Errorf("Memory.String() = %q, expected %q", m.String(), "8 GB")
	}
}
