package wdl

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/lmtani/pumbaa/pkg/wdl/ast"
)

// Severity classifies a finding. Errors describe things Cromwell will
// reject; warnings describe things that are suspicious but may well be
// valid, since Cromwell coerces some types and this parser does not model
// every WDL construct.
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
)

// Finding is a single problem found while checking an inputs JSON against
// the workflow that consumes it.
type Finding struct {
	Severity Severity
	Input    string // Qualified input name, empty when not input-specific
	Message  string
}

// FileRef is a File-typed value found in the inputs, for callers that can
// verify whether the path actually exists.
type FileRef struct {
	Input string
	Path  string
}

// InputsReport is the result of checking an inputs JSON against a WDL.
type InputsReport struct {
	WorkflowName string
	// Parsed reports whether the WDL could be parsed. When false, only the
	// parse warning is present: Cromwell remains the authority on WDL that
	// this parser cannot read.
	Parsed   bool
	Findings []Finding
	Files    []FileRef
}

// HasErrors reports whether any finding would make Cromwell reject the run.
func (r *InputsReport) HasErrors() bool {
	for _, f := range r.Findings {
		if f.Severity == SeverityError {
			return true
		}
	}
	return false
}

// CheckInputs checks an inputs JSON against the workflow declared in the WDL
// source: required inputs present, no placeholders left from scaffolding,
// values plausible for their declared types, and no undeclared keys. It also
// returns every File-typed path, so callers can verify their existence.
//
// It performs no IO and never fails on WDL it cannot parse — that case is
// reported as a warning and left to Cromwell.
func CheckInputs(source, inputsJSON []byte) *InputsReport {
	report := &InputsReport{}

	doc, err := ParseBytes(source)
	if err != nil || doc.Workflow == nil {
		report.Findings = append(report.Findings, Finding{
			Severity: SeverityWarning,
			Message:  "could not parse the WDL locally, so input checks were skipped (Cromwell will validate it)",
		})
		return report
	}
	report.Parsed = true
	report.WorkflowName = doc.Workflow.Name

	provided, err := parseInputValues(inputsJSON)
	if err != nil {
		report.Findings = append(report.Findings, Finding{
			Severity: SeverityError,
			Message:  fmt.Sprintf("inputs file is not valid JSON: %v", err),
		})
		return report
	}

	declared := make(map[string]*ast.Declaration, len(doc.Workflow.Inputs))
	prefix := doc.Workflow.Name + "."
	for _, in := range doc.Workflow.Inputs {
		if in != nil && in.Type != nil {
			declared[prefix+in.Name] = in
		}
	}

	for _, spec := range workflowInputSpecs(doc.Workflow) {
		value, ok := provided[spec.Name]
		if !ok {
			if spec.Required() {
				report.Findings = append(report.Findings, Finding{
					Severity: SeverityError,
					Input:    spec.Name,
					Message:  fmt.Sprintf("required input is missing (type %s)", spec.Type),
				})
			}
			continue
		}
		if IsPlaceholder(value) {
			report.Findings = append(report.Findings, Finding{
				Severity: SeverityError,
				Input:    spec.Name,
				Message:  fmt.Sprintf("still holds the scaffold placeholder — replace it with a value of type %s", spec.Type),
			})
			continue
		}
		decl := declared[spec.Name]
		report.Findings = append(report.Findings, checkValue(spec.Name, decl.Type, value)...)
		report.Files = append(report.Files, collectFiles(spec.Name, decl.Type, value)...)
	}

	for name := range provided {
		if _, ok := declared[name]; ok {
			continue
		}
		report.Findings = append(report.Findings, Finding{
			Severity: SeverityWarning,
			Input:    name,
			Message:  "not declared by this workflow — check for a typo (subworkflow-qualified inputs are not modelled here)",
		})
	}

	sortFindings(report.Findings)
	return report
}

// checkValue compares a JSON value against a declared WDL type. It only
// reports an error when no reasonable coercion exists: being wrong here
// would block a valid submission, which is worse than staying quiet.
func checkValue(name string, t *ast.Type, value any) []Finding {
	if t == nil {
		return nil
	}
	if value == nil {
		if t.Optional {
			return nil
		}
		return []Finding{{
			Severity: SeverityError,
			Input:    name,
			Message:  fmt.Sprintf("is null, but %s is required", t.String()),
		}}
	}

	switch t.Base {
	case "Int":
		return checkNumber(name, t, value, true)
	case "Float":
		return checkNumber(name, t, value, false)
	case "Boolean":
		switch v := value.(type) {
		case bool:
			return nil
		case string:
			if v == "true" || v == "false" {
				return warn(name, "is the string %q where Boolean is expected", v)
			}
		}
		return mismatch(name, t, value)
	case "String":
		switch value.(type) {
		case string:
			return nil
		case float64, bool:
			return warn(name, "is a %s where String is expected; Cromwell may coerce it", jsonKind(value))
		}
		return mismatch(name, t, value)
	case "File", "Directory":
		if s, ok := value.(string); ok {
			if strings.TrimSpace(s) == "" {
				return []Finding{{Severity: SeverityError, Input: name, Message: "is an empty path"}}
			}
			return nil
		}
		return mismatch(name, t, value)
	case "Array":
		items, ok := value.([]any)
		if !ok {
			return mismatch(name, t, value)
		}
		if t.NonEmpty && len(items) == 0 {
			return []Finding{{
				Severity: SeverityError,
				Input:    name,
				Message:  fmt.Sprintf("is empty, but %s requires at least one element", t.String()),
			}}
		}
		var findings []Finding
		for i, item := range items {
			findings = append(findings, checkValue(fmt.Sprintf("%s[%d]", name, i), t.ArrayType, item)...)
		}
		return findings
	case "Map", "Object":
		if _, ok := value.(map[string]any); !ok {
			return mismatch(name, t, value)
		}
		return nil
	case "Pair":
		switch value.(type) {
		case map[string]any, []any:
			return nil
		}
		return mismatch(name, t, value)
	}

	// Unknown base (custom struct types): nothing reliable to check.
	return nil
}

// checkNumber validates Int/Float values, tolerating the coercions Cromwell
// itself performs.
func checkNumber(name string, t *ast.Type, value any, integral bool) []Finding {
	switch v := value.(type) {
	case float64:
		if integral && v != math.Trunc(v) {
			return []Finding{{
				Severity: SeverityError,
				Input:    name,
				Message:  fmt.Sprintf("is %v, but Int expects a whole number", v),
			}}
		}
		return nil
	case string:
		if _, err := strconv.ParseFloat(v, 64); err == nil {
			return warn(name, "is the quoted number %q where %s is expected", v, t.String())
		}
	}
	return mismatch(name, t, value)
}

func mismatch(name string, t *ast.Type, value any) []Finding {
	return []Finding{{
		Severity: SeverityError,
		Input:    name,
		Message:  fmt.Sprintf("is a %s, but %s is expected", jsonKind(value), t.String()),
	}}
}

func warn(name, format string, args ...any) []Finding {
	return []Finding{{Severity: SeverityWarning, Input: name, Message: fmt.Sprintf(format, args...)}}
}

// jsonKind names a decoded JSON value the way a user would describe it.
func jsonKind(v any) string {
	switch v.(type) {
	case nil:
		return "null"
	case bool:
		return "boolean"
	case float64:
		return "number"
	case string:
		return "string"
	case []any:
		return "list"
	case map[string]any:
		return "object"
	}
	return "value"
}

// collectFiles returns every File-typed path in a value, recursing into
// arrays so scattered inputs are covered too.
func collectFiles(name string, t *ast.Type, value any) []FileRef {
	if t == nil || value == nil {
		return nil
	}
	switch t.Base {
	case "File", "Directory":
		if s, ok := value.(string); ok && strings.TrimSpace(s) != "" {
			return []FileRef{{Input: name, Path: s}}
		}
	case "Array":
		items, ok := value.([]any)
		if !ok {
			return nil
		}
		var refs []FileRef
		for i, item := range items {
			refs = append(refs, collectFiles(fmt.Sprintf("%s[%d]", name, i), t.ArrayType, item)...)
		}
		return refs
	}
	return nil
}

// parseInputValues decodes the inputs JSON into plain Go values.
func parseInputValues(data []byte) (map[string]any, error) {
	if len(data) == 0 {
		return map[string]any{}, nil
	}
	var values map[string]any
	if err := json.Unmarshal(data, &values); err != nil {
		return nil, err
	}
	return values, nil
}

// sortFindings puts errors before warnings, keeping each group in the order
// it was produced (declaration order for inputs).
func sortFindings(findings []Finding) {
	stable := make([]Finding, 0, len(findings))
	for _, f := range findings {
		if f.Severity == SeverityError {
			stable = append(stable, f)
		}
	}
	for _, f := range findings {
		if f.Severity != SeverityError {
			stable = append(stable, f)
		}
	}
	copy(findings, stable)
}
