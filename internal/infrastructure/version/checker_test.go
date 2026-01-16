package version

import "testing"

func TestIsNewerVersion(t *testing.T) {
	tests := []struct {
		name      string
		latest    string
		current   string
		wantNewer bool
	}{
		{"patch update", "1.0.5", "1.0.4", true},
		{"minor update", "1.1.0", "1.0.4", true},
		{"major update", "2.0.0", "1.9.9", true},
		{"same version", "1.0.4", "1.0.4", false},
		{"current is newer", "1.0.4", "1.1.0", false},
		{"with v prefix", "v1.0.5", "1.0.4", true},
		{"both with v prefix", "v1.0.5", "v1.0.4", true},
		{"major downgrade", "1.0.0", "2.0.0", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isNewerVersion(tt.latest, tt.current)
			if got != tt.wantNewer {
				t.Errorf("isNewerVersion(%q, %q) = %v, want %v",
					tt.latest, tt.current, got, tt.wantNewer)
			}
		})
	}
}

func TestParseVersion(t *testing.T) {
	tests := []struct {
		input string
		want  [3]int
	}{
		{"1.0.4", [3]int{1, 0, 4}},
		{"2.10.0", [3]int{2, 10, 0}},
		{"0.0.1", [3]int{0, 0, 1}},
		{"1.0", [3]int{1, 0, 0}},
		{"1", [3]int{1, 0, 0}},
		{"", [3]int{0, 0, 0}},
		{"invalid", [3]int{0, 0, 0}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseVersion(tt.input)
			if got != tt.want {
				t.Errorf("parseVersion(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestGitHubChecker_DevVersion(t *testing.T) {
	checker := NewGitHubChecker("lmtani/pumbaa")
	ch := checker.Check("dev")

	// Should return nil (skipped check)
	result := <-ch
	if result != nil {
		t.Error("expected nil result for dev version")
	}
}
