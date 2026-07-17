package wdl

import (
	"encoding/json"
	"strings"
	"testing"
)

// sampleWDL exercises the input shapes that matter: required and optional,
// defaults, compound types, and parameter_meta in both spellings.
const sampleWDL = `version 1.0

workflow AlignReads {
    input {
        File reads_fastq
        Array[File]+ reference_files
        String sample_name
        Int threads = 4
        Float? min_quality
        Boolean skip_qc = false
        File? adapters
    }

    parameter_meta {
        reads_fastq: "Sequencing reads in FASTQ format"
        reference_files: {description: "Reference genome and its index files"}
    }

    call align { input: reads = reads_fastq }
}

task align {
    input { File reads }
    command <<< echo ~{reads} >>>
    runtime { docker: "ubuntu:22.04" }
}
`

func specByName(t *testing.T, specs []InputSpec, name string) InputSpec {
	t.Helper()
	for _, s := range specs {
		if s.Name == name {
			return s
		}
	}
	t.Fatalf("input %q not found in %+v", name, specs)
	return InputSpec{}
}

func TestWorkflowInputs(t *testing.T) {
	specs, err := WorkflowInputs([]byte(sampleWDL))
	if err != nil {
		t.Fatalf("WorkflowInputs() error = %v", err)
	}
	if len(specs) != 7 {
		t.Fatalf("got %d inputs, want 7: %+v", len(specs), specs)
	}

	// Declaration order is preserved and names are qualified for Cromwell.
	if specs[0].Name != "AlignReads.reads_fastq" {
		t.Errorf("first input = %q, want AlignReads.reads_fastq", specs[0].Name)
	}

	reads := specByName(t, specs, "AlignReads.reads_fastq")
	if reads.Type != "File" || !reads.Required() {
		t.Errorf("reads_fastq = %+v, want required File", reads)
	}
	if reads.Description != "Sequencing reads in FASTQ format" {
		t.Errorf("parameter_meta string form not read: %q", reads.Description)
	}

	refs := specByName(t, specs, "AlignReads.reference_files")
	if refs.Type != "Array[File]+" {
		t.Errorf("reference_files type = %q, want Array[File]+", refs.Type)
	}
	if refs.Description != "Reference genome and its index files" {
		t.Errorf("parameter_meta object form not read: %q", refs.Description)
	}

	// A default makes an input optional to provide, even without "?".
	threads := specByName(t, specs, "AlignReads.threads")
	if threads.Required() || threads.Default != "4" {
		t.Errorf("threads = %+v, want not required with default 4", threads)
	}

	skip := specByName(t, specs, "AlignReads.skip_qc")
	if skip.Default != "false" {
		t.Errorf("skip_qc default = %q, want false", skip.Default)
	}

	// "?" without a default: not required, no default value.
	quality := specByName(t, specs, "AlignReads.min_quality")
	if quality.Required() || !quality.Optional || quality.Default != "" {
		t.Errorf("min_quality = %+v, want optional with no default", quality)
	}
}

func TestWorkflowInputsWithoutWorkflow(t *testing.T) {
	specs, err := WorkflowInputs([]byte("version 1.0\n\ntask lonely {\n  command <<< echo hi >>>\n}\n"))
	if err != nil {
		t.Fatalf("WorkflowInputs() error = %v", err)
	}
	if specs != nil {
		t.Errorf("a document with no workflow should yield no inputs, got %+v", specs)
	}
}

func TestScaffoldInputsRequiredOnly(t *testing.T) {
	sc, err := ScaffoldInputs([]byte(sampleWDL), ScaffoldOptions{})
	if err != nil {
		t.Fatalf("ScaffoldInputs() error = %v", err)
	}
	tpl := sc.Template
	if sc.WorkflowName != "AlignReads" {
		t.Errorf("WorkflowName = %q, want AlignReads", sc.WorkflowName)
	}
	if len(sc.Inputs) != 7 {
		t.Errorf("Inputs should describe every input, got %d", len(sc.Inputs))
	}

	// Angle brackets must survive: HTML-escaped placeholders are unreadable.
	if strings.Contains(string(tpl), "\\u003c") {
		t.Errorf("placeholder was HTML-escaped:\n%s", tpl)
	}

	var got map[string]any
	if err := json.Unmarshal(tpl, &got); err != nil {
		t.Fatalf("template is not valid JSON: %v\n%s", err, tpl)
	}
	if len(got) != 3 {
		t.Errorf("template should hold only the 3 required inputs, got %v", got)
	}
	for _, name := range []string{"AlignReads.reads_fastq", "AlignReads.reference_files", "AlignReads.sample_name"} {
		v, ok := got[name]
		if !ok {
			t.Fatalf("required input %q missing from template", name)
		}
		if !IsPlaceholder(v) {
			t.Errorf("%s = %v, want a placeholder", name, v)
		}
	}

	// The placeholder teaches: type and, when documented, description.
	reads := got["AlignReads.reads_fastq"].(string)
	if !strings.Contains(reads, "File") || !strings.Contains(reads, "Sequencing reads") {
		t.Errorf("placeholder should carry type and description, got %q", reads)
	}

	// Declaration order survives (a Go map would have shuffled it).
	first := strings.Index(string(tpl), "reads_fastq")
	second := strings.Index(string(tpl), "reference_files")
	third := strings.Index(string(tpl), "sample_name")
	if first > second || second > third {
		t.Errorf("template lost declaration order:\n%s", tpl)
	}
}

func TestScaffoldInputsIncludeOptional(t *testing.T) {
	sc, err := ScaffoldInputs([]byte(sampleWDL), ScaffoldOptions{IncludeOptional: true})
	if err != nil {
		t.Fatalf("ScaffoldInputs() error = %v", err)
	}
	tpl := sc.Template

	var got map[string]any
	if err := json.Unmarshal(tpl, &got); err != nil {
		t.Fatalf("template is not valid JSON: %v", err)
	}
	if len(got) != 7 {
		t.Fatalf("got %d inputs, want all 7: %v", len(got), got)
	}
	// Literal defaults are rendered as values...
	if got["AlignReads.threads"] != float64(4) {
		t.Errorf("threads = %v, want its default 4", got["AlignReads.threads"])
	}
	if got["AlignReads.skip_qc"] != false {
		t.Errorf("skip_qc = %v, want its default false", got["AlignReads.skip_qc"])
	}
	// ...and optionals without a default mean "not provided".
	if v, ok := got["AlignReads.adapters"]; !ok || v != nil {
		t.Errorf("adapters = %v, want null", v)
	}
}

func TestScaffoldInputsNoInputs(t *testing.T) {
	sc, err := ScaffoldInputs([]byte("version 1.0\n\nworkflow Empty {\n}\n"), ScaffoldOptions{})
	if err != nil {
		t.Fatalf("ScaffoldInputs() error = %v", err)
	}
	if strings.TrimSpace(string(sc.Template)) != "{}" {
		t.Errorf("template = %q, want {}", sc.Template)
	}
}

func TestCheckInputsAcceptsValidInputs(t *testing.T) {
	inputs := `{
	  "AlignReads.reads_fastq": "gs://bucket/reads.fastq",
	  "AlignReads.reference_files": ["gs://bucket/ref.fa", "gs://bucket/ref.fai"],
	  "AlignReads.sample_name": "NA12878",
	  "AlignReads.threads": 8,
	  "AlignReads.min_quality": 30.5,
	  "AlignReads.adapters": null
	}`

	report := CheckInputs([]byte(sampleWDL), []byte(inputs))

	if !report.Parsed || report.WorkflowName != "AlignReads" {
		t.Fatalf("report = %+v, want parsed AlignReads", report)
	}
	if len(report.Findings) != 0 {
		t.Errorf("valid inputs produced findings: %+v", report.Findings)
	}
	// Every File-typed value is offered for existence checking, including
	// each element of an Array[File].
	if len(report.Files) != 3 {
		t.Errorf("got %d file refs, want 3: %+v", len(report.Files), report.Files)
	}
	if report.Files[1].Input != "AlignReads.reference_files[0]" {
		t.Errorf("array element refs should be indexed, got %q", report.Files[1].Input)
	}
}

func TestCheckInputsFindsBlockingProblems(t *testing.T) {
	inputs := `{
	  "AlignReads.reads_fastq": "<FILL: File — Sequencing reads in FASTQ format>",
	  "AlignReads.reference_files": "gs://bucket/ref.fa",
	  "AlignReads.skip_qc": "yes"
	}`

	report := CheckInputs([]byte(sampleWDL), []byte(inputs))

	if !report.HasErrors() {
		t.Fatalf("expected errors, got %+v", report.Findings)
	}
	byInput := map[string]Finding{}
	for _, f := range report.Findings {
		byInput[f.Input] = f
	}

	if f := byInput["AlignReads.reads_fastq"]; f.Severity != SeverityError || !strings.Contains(f.Message, "placeholder") {
		t.Errorf("unfilled placeholder should be an error, got %+v", f)
	}
	if f := byInput["AlignReads.reference_files"]; f.Severity != SeverityError || !strings.Contains(f.Message, "Array[File]+") {
		t.Errorf("scalar for an array should be an error, got %+v", f)
	}
	if f := byInput["AlignReads.sample_name"]; f.Severity != SeverityError || !strings.Contains(f.Message, "missing") {
		t.Errorf("missing required input should be an error, got %+v", f)
	}
	if f := byInput["AlignReads.skip_qc"]; f.Severity != SeverityError {
		t.Errorf(`"yes" for a Boolean should be an error, got %+v`, f)
	}
	// Errors are listed before warnings.
	if report.Findings[0].Severity != SeverityError {
		t.Errorf("errors should come first, got %+v", report.Findings)
	}
	// A placeholder is not offered as a path to check.
	if len(report.Files) != 0 {
		t.Errorf("placeholders must not become file refs: %+v", report.Files)
	}
}

func TestCheckInputsWarnsWithoutBlocking(t *testing.T) {
	// Each of these is suspicious but plausibly coerced by Cromwell, or
	// simply beyond what this parser models — none may block a submission.
	inputs := `{
	  "AlignReads.reads_fastq": "gs://bucket/reads.fastq",
	  "AlignReads.reference_files": ["gs://bucket/ref.fa"],
	  "AlignReads.sample_name": "NA12878",
	  "AlignReads.threads": "8",
	  "AlignReads.sampleName": "typo",
	  "AlignReads.sub.inner.x": 1
	}`

	report := CheckInputs([]byte(sampleWDL), []byte(inputs))

	if report.HasErrors() {
		t.Fatalf("coercible values and unknown keys must not block: %+v", report.Findings)
	}
	if len(report.Findings) != 3 {
		t.Errorf("expected 3 warnings, got %+v", report.Findings)
	}
}

func TestCheckInputsTypeMatrix(t *testing.T) {
	const wdlSrc = `version 1.0

workflow T {
    input {
        Int i
        Float f
        Boolean b
        String s
        Array[Int] nums
        Map[String, String] m
    }
}
`
	cases := []struct {
		name     string
		inputs   string
		severity Severity // "" means no finding for that input
		input    string
	}{
		{"int accepts whole number", `{"T.i": 5}`, "", "T.i"},
		{"int rejects decimal", `{"T.i": 5.5}`, SeverityError, "T.i"},
		{"int warns on quoted number", `{"T.i": "5"}`, SeverityWarning, "T.i"},
		{"int rejects text", `{"T.i": "many"}`, SeverityError, "T.i"},
		{"float accepts whole number", `{"T.f": 3}`, "", "T.f"},
		{"float accepts decimal", `{"T.f": 3.5}`, "", "T.f"},
		{"boolean rejects number", `{"T.b": 1}`, SeverityError, "T.b"},
		{"boolean warns on quoted bool", `{"T.b": "true"}`, SeverityWarning, "T.b"},
		{"string warns on number", `{"T.s": 5}`, SeverityWarning, "T.s"},
		{"string rejects list", `{"T.s": [1]}`, SeverityError, "T.s"},
		{"array accepts list", `{"T.nums": [1, 2]}`, "", "T.nums"},
		{"array rejects scalar", `{"T.nums": 1}`, SeverityError, "T.nums"},
		{"array element type is checked", `{"T.nums": [1, "x"]}`, SeverityError, "T.nums[1]"},
		{"map accepts object", `{"T.m": {"a": "b"}}`, "", "T.m"},
		{"map rejects list", `{"T.m": [1]}`, SeverityError, "T.m"},
		{"null rejected for required", `{"T.i": null}`, SeverityError, "T.i"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			report := CheckInputs([]byte(wdlSrc), []byte(tc.inputs))

			var got Finding
			for _, f := range report.Findings {
				// Ignore the "missing required input" noise from the inputs
				// this case does not provide.
				if f.Input == tc.input && !strings.Contains(f.Message, "missing") {
					got = f
					break
				}
			}
			if got.Severity != tc.severity {
				t.Errorf("severity for %s = %q (%s), want %q", tc.input, got.Severity, got.Message, tc.severity)
			}
		})
	}
}

func TestCheckInputsNonEmptyArray(t *testing.T) {
	report := CheckInputs([]byte(sampleWDL), []byte(`{
	  "AlignReads.reads_fastq": "gs://b/r.fastq",
	  "AlignReads.sample_name": "NA12878",
	  "AlignReads.reference_files": []
	}`))

	found := false
	for _, f := range report.Findings {
		if f.Input == "AlignReads.reference_files" && f.Severity == SeverityError && strings.Contains(f.Message, "at least one") {
			found = true
		}
	}
	if !found {
		t.Errorf("Array[File]+ must reject an empty list, got %+v", report.Findings)
	}
}

func TestCheckInputsUnparseableWDLIsAWarning(t *testing.T) {
	report := CheckInputs([]byte("this is not WDL at all {{{"), []byte(`{"x": 1}`))

	if report.Parsed {
		t.Error("report should record that the WDL was not parsed")
	}
	if report.HasErrors() {
		t.Errorf("unparseable WDL must not block submission (Cromwell decides): %+v", report.Findings)
	}
	if len(report.Findings) != 1 || report.Findings[0].Severity != SeverityWarning {
		t.Errorf("expected a single warning, got %+v", report.Findings)
	}
}

func TestCheckInputsMalformedJSON(t *testing.T) {
	report := CheckInputs([]byte(sampleWDL), []byte(`{"AlignReads.sample_name": }`))

	if !report.HasErrors() {
		t.Fatalf("malformed inputs JSON should be an error, got %+v", report.Findings)
	}
	if !strings.Contains(report.Findings[0].Message, "not valid JSON") {
		t.Errorf("error should name the problem, got %q", report.Findings[0].Message)
	}
}

func TestCheckInputsEmptyInputsFile(t *testing.T) {
	report := CheckInputs([]byte(sampleWDL), nil)

	// All three required inputs are reported, not just the first.
	errs := 0
	for _, f := range report.Findings {
		if f.Severity == SeverityError {
			errs++
		}
	}
	if errs != 3 {
		t.Errorf("got %d errors, want one per required input: %+v", errs, report.Findings)
	}
}

func TestScaffoldInputsWithoutWorkflowFails(t *testing.T) {
	_, err := ScaffoldInputs([]byte("version 1.0\n\ntask lonely {\n  command <<< echo hi >>>\n}\n"), ScaffoldOptions{})
	if err == nil {
		t.Error("scaffolding a WDL with no workflow should fail with a clear message")
	}
}
