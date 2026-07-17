package wdl

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lmtani/pumbaa/pkg/wdl/ast"
)

// PlaceholderPrefix marks a scaffolded value the user still has to fill in.
// It is deliberately detectable so preflight can tell "template not filled"
// apart from a genuine value.
const PlaceholderPrefix = "<FILL:"

// InputSpec describes one workflow input as declared in the WDL.
type InputSpec struct {
	Name        string // Qualified as Cromwell expects it: "workflow.input"
	Type        string // Rendered WDL type: "File", "Array[File]+", "Int?"
	Optional    bool   // Declared with a "?" suffix
	Default     string // Rendered default expression, empty when unbound
	Description string // From the workflow's parameter_meta, when documented
}

// Required reports whether the input must be provided: no "?" and no default.
func (s InputSpec) Required() bool {
	return !s.Optional && s.Default == ""
}

// WorkflowInputs returns the declared inputs of the workflow in the document,
// in declaration order. Returns nil when the source has no workflow.
func WorkflowInputs(source []byte) ([]InputSpec, error) {
	doc, err := ParseBytes(source)
	if err != nil {
		return nil, err
	}
	if doc.Workflow == nil {
		return nil, nil
	}
	return workflowInputSpecs(doc.Workflow), nil
}

func workflowInputSpecs(wf *ast.Workflow) []InputSpec {
	specs := make([]InputSpec, 0, len(wf.Inputs))
	for _, in := range wf.Inputs {
		if in == nil || in.Type == nil {
			continue
		}
		specs = append(specs, InputSpec{
			Name:        wf.Name + "." + in.Name,
			Type:        in.Type.String(),
			Optional:    in.Type.Optional,
			Default:     renderExpression(in.Expression),
			Description: parameterMetaDescription(wf.ParameterMeta, in.Name),
		})
	}
	return specs
}

// parameterMetaDescription extracts an input's documentation from
// parameter_meta, which WDL allows as either a bare string or an object
// carrying a "description" (or "help") field.
func parameterMetaDescription(meta map[string]any, name string) string {
	raw, ok := meta[name]
	if !ok {
		return ""
	}
	switch v := raw.(type) {
	case string:
		return v
	case map[string]any:
		for _, key := range []string{"description", "help"} {
			if s, ok := v[key].(string); ok {
				return s
			}
		}
	}
	return ""
}

// renderExpression renders a default expression for display. Only literal
// forms are rendered; anything computed (function calls, references) renders
// as empty, which keeps "has a default" honest for scaffolding purposes.
func renderExpression(e ast.Expression) string {
	v, ok := literalValue(e)
	if !ok {
		return ""
	}
	data, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(data)
}

// literalValue extracts the Go value of a literal expression, reporting
// whether the expression was a literal at all.
func literalValue(e ast.Expression) (any, bool) {
	switch v := e.(type) {
	case nil:
		return nil, false
	case *ast.StringLiteral:
		return v.Value, true
	case *ast.Literal:
		if v.Value == nil {
			// Both "None" and unparsed expressions land here; treat as
			// no usable literal.
			return nil, false
		}
		return v.Value, true
	case *ast.ArrayLiteral:
		items := make([]any, 0, len(v.Elements))
		for _, el := range v.Elements {
			item, ok := literalValue(el)
			if !ok {
				return nil, false
			}
			items = append(items, item)
		}
		return items, true
	}
	return nil, false
}

// Scaffold is a generated inputs template together with the declarations it
// was generated from.
type Scaffold struct {
	WorkflowName string
	// Template is the inputs JSON to fill in.
	Template []byte
	// Inputs describes every declared input, including the optional ones
	// left out of the template, so callers can explain them to the user.
	Inputs []InputSpec
}

// ScaffoldOptions controls how the inputs template is rendered.
type ScaffoldOptions struct {
	// IncludeOptional adds optional inputs, rendered with their default
	// value (or null when the default is not a literal).
	IncludeOptional bool
}

// ScaffoldInputs renders an inputs JSON template for the workflow, with
// required inputs first, in declaration order. Values of required inputs are
// placeholders (see PlaceholderPrefix) carrying the type and, when the WDL
// documents it, the input's description.
func ScaffoldInputs(source []byte, opts ScaffoldOptions) (*Scaffold, error) {
	doc, err := ParseBytes(source)
	if err != nil {
		return nil, err
	}
	if doc.Workflow == nil {
		return nil, fmt.Errorf("no workflow found in the WDL (only tasks?); nothing to scaffold")
	}

	scaffold := &Scaffold{WorkflowName: doc.Workflow.Name}
	specs := workflowInputSpecs(doc.Workflow)
	scaffold.Inputs = specs
	if len(specs) == 0 {
		scaffold.Template = []byte("{}\n")
		return scaffold, nil
	}

	type entry struct {
		name  string
		value any
	}
	var entries []entry

	for _, s := range specs {
		if s.Required() {
			entries = append(entries, entry{name: s.Name, value: placeholderFor(s)})
		}
	}
	if opts.IncludeOptional {
		for _, s := range specs {
			if s.Required() {
				continue
			}
			entries = append(entries, entry{name: s.Name, value: defaultOrNull(s)})
		}
	}

	// Rendered by hand: a Go map would lose the declaration order that makes
	// the template readable.
	var buf bytes.Buffer
	buf.WriteString("{\n")
	for i, e := range entries {
		key, err := encodeJSON(e.name)
		if err != nil {
			return nil, err
		}
		value, err := encodeJSON(e.value)
		if err != nil {
			return nil, err
		}
		fmt.Fprintf(&buf, "  %s: %s", key, value)
		if i < len(entries)-1 {
			buf.WriteString(",")
		}
		buf.WriteString("\n")
	}
	buf.WriteString("}\n")

	scaffold.Template = buf.Bytes()
	return scaffold, nil
}

// encodeJSON marshals a value without HTML escaping, which would turn the
// placeholder's angle brackets into unreadable < sequences.
func encodeJSON(v any) (string, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return "", err
	}
	return strings.TrimRight(buf.String(), "\n"), nil
}

// placeholderFor builds the sentinel value for a required input.
func placeholderFor(s InputSpec) string {
	p := PlaceholderPrefix + " " + s.Type
	if s.Description != "" {
		p += " — " + s.Description
	}
	return p + ">"
}

// defaultOrNull renders an optional input's value: its literal default when
// it has one, null otherwise (which means "not provided" to Cromwell).
func defaultOrNull(s InputSpec) any {
	if s.Default == "" {
		return nil
	}
	var v any
	if err := json.Unmarshal([]byte(s.Default), &v); err != nil {
		return nil
	}
	return v
}

// IsPlaceholder reports whether a JSON value is a scaffolded placeholder the
// user has not replaced yet.
func IsPlaceholder(v any) bool {
	s, ok := v.(string)
	return ok && strings.HasPrefix(s, PlaceholderPrefix)
}
