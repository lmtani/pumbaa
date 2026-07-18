package workflow

import (
	"reflect"
	"testing"
)

// realHashTree mirrors the shape Cromwell 91 returns under callCaching.hashes,
// captured from the showcase server (see docs/design/cache-explainer.md).
func realHashTree(docker, inputVcf string) map[string]any {
	return map[string]any{
		"backend name":     "509820290D57F333403F490DDE7316F4",
		"command template": "94A353F002A219A742B02214A920582A",
		"input count":      "ECCBC87E4B5CE2FE28308FD9F2A7BAF3",
		"output count":     "C81E728D9D4C2F636F067F89CC14862C",
		"runtime attribute": map[string]any{
			"docker":               docker,
			"continueOnReturnCode": "CFCD208495D565EF66E7DFF9F98764DA",
			"failOnStderr":         "68934A3E9455FA72420237EB05902327",
		},
		"input": map[string]any{
			"String output_basename": "7A6BE2B0057A7F4DE29BC1AC77497B5F",
			"String docker":          docker,
			"File input_vcf":         inputVcf,
		},
		"output expression": map[string]any{
			"File out_vcf": "3BE037CC96CE1D29EC506AF62BBDDDAF",
		},
	}
}

func TestFlattenHashesFlattensNestedGroups(t *testing.T) {
	got := FlattenHashes(realHashTree("D0", "F0"))

	want := CallFingerprint{
		"backend name":                            "509820290D57F333403F490DDE7316F4",
		"command template":                        "94A353F002A219A742B02214A920582A",
		"input count":                             "ECCBC87E4B5CE2FE28308FD9F2A7BAF3",
		"output count":                            "C81E728D9D4C2F636F067F89CC14862C",
		"runtime attribute: docker":               "D0",
		"runtime attribute: continueOnReturnCode": "CFCD208495D565EF66E7DFF9F98764DA",
		"runtime attribute: failOnStderr":         "68934A3E9455FA72420237EB05902327",
		"input: String output_basename":           "7A6BE2B0057A7F4DE29BC1AC77497B5F",
		"input: String docker":                    "D0",
		"input: File input_vcf":                   "F0",
		"output expression: File out_vcf":         "3BE037CC96CE1D29EC506AF62BBDDDAF",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("FlattenHashes() mismatch\ngot:  %v\nwant: %v", got, want)
	}
}

func TestFlattenHashesSkipsNonStringLeaves(t *testing.T) {
	got := FlattenHashes(map[string]any{"ok": "ABC", "weird": 42, "null": nil})
	if len(got) != 1 || got["ok"] != "ABC" {
		t.Errorf("expected only the string leaf to survive, got %v", got)
	}
}

// A docker change surfaces under two keys when the WDL passes docker as a task
// input. Both must collapse to one category so the report says "docker" once.
func TestCompareFingerprintsCollapsesDockerToOneCategory(t *testing.T) {
	ref := FlattenHashes(realHashTree("D_OLD", "F0"))
	cur := FlattenHashes(realHashTree("D_NEW", "F0"))

	changes := CompareFingerprints(ref, cur)
	if len(changes) != 2 {
		t.Fatalf("expected 2 differing keys, got %d: %+v", len(changes), changes)
	}
	for _, c := range changes {
		if c.Category != CategoryDocker {
			t.Errorf("key %q: got category %v, want CategoryDocker", c.Key, c.Category)
		}
	}
	if cats := Categories(changes); len(cats) != 1 || cats[0] != CategoryDocker {
		t.Errorf("Categories() = %v, want exactly [docker image]", cats)
	}
}

// The cascade signal: a downstream call differs only in an input file.
func TestCompareFingerprintsIdentifiesInputFileChange(t *testing.T) {
	ref := FlattenHashes(realHashTree("D0", "F_OLD"))
	cur := FlattenHashes(realHashTree("D0", "F_NEW"))

	changes := CompareFingerprints(ref, cur)
	if len(changes) != 1 {
		t.Fatalf("expected 1 differing key, got %d: %+v", len(changes), changes)
	}
	if changes[0].Key != "input: File input_vcf" {
		t.Errorf("got key %q, want input: File input_vcf", changes[0].Key)
	}
	if changes[0].Category != CategoryInputFile {
		t.Errorf("got category %v, want CategoryInputFile", changes[0].Category)
	}
	if changes[0].Reference != "F_OLD" || changes[0].Current != "F_NEW" {
		t.Errorf("got %q → %q, want F_OLD → F_NEW", changes[0].Reference, changes[0].Current)
	}
}

// Identical fingerprints are the load-bearing case for the "cached copy was
// unusable" verdict: a miss with no differing key cannot be blamed on a change.
func TestCompareFingerprintsReturnsNothingWhenIdentical(t *testing.T) {
	ref := FlattenHashes(realHashTree("D0", "F0"))
	cur := FlattenHashes(realHashTree("D0", "F0"))

	if changes := CompareFingerprints(ref, cur); len(changes) != 0 {
		t.Errorf("expected no changes for identical fingerprints, got %+v", changes)
	}
}

func TestCompareFingerprintsReportsKeysMissingFromOneSide(t *testing.T) {
	ref := CallFingerprint{"command template": "A", "input: File gone": "B"}
	cur := CallFingerprint{"command template": "A", "input: File added": "C"}

	changes := CompareFingerprints(ref, cur)
	if len(changes) != 2 {
		t.Fatalf("expected 2 changes, got %+v", changes)
	}
	// Sorted by key: "input: File added" precedes "input: File gone".
	if changes[0].Key != "input: File added" || changes[0].Reference != "" {
		t.Errorf("expected added key with empty reference, got %+v", changes[0])
	}
	if changes[1].Key != "input: File gone" || changes[1].Current != "" {
		t.Errorf("expected removed key with empty current, got %+v", changes[1])
	}
}

func TestParseInputHashKey(t *testing.T) {
	tests := []struct {
		key      string
		wantType string
		wantName string
	}{
		{"input: File input_vcf", "File", "input_vcf"},
		{"input: String sample", "String", "sample"},
		{"input: Array[File] bams", "Array[File]", "bams"},
		{"input: Map[String, Int] m", "Map[String, Int]", "m"},
		{"runtime attribute: docker", "", ""},
		{"command template", "", ""},
		{"input: malformed", "", ""},
	}
	for _, tt := range tests {
		gotType, gotName := ParseInputHashKey(tt.key)
		if gotType != tt.wantType || gotName != tt.wantName {
			t.Errorf("ParseInputHashKey(%q) = (%q, %q), want (%q, %q)",
				tt.key, gotType, gotName, tt.wantType, tt.wantName)
		}
	}
}

func TestCategorize(t *testing.T) {
	tests := []struct {
		key  string
		want ChangeCategory
	}{
		{"command template", CategoryCommand},
		{"backend name", CategoryBackend},
		{"input count", CategoryCount},
		{"output count", CategoryCount},
		{"runtime attribute: docker", CategoryDocker},
		{"input: String docker", CategoryDocker},
		{"runtime attribute: failOnStderr", CategoryRuntime},
		{"input: File ref", CategoryInputFile},
		{"input: String sample", CategoryInputValue},
		{"output expression: File out", CategoryOther},
		{"something new in cromwell 92", CategoryOther},
	}
	for _, tt := range tests {
		if got := categorize(tt.key); got != tt.want {
			t.Errorf("categorize(%q) = %v, want %v", tt.key, got, tt.want)
		}
	}
}
