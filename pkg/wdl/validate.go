package wdl

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lmtani/pumbaa/pkg/wdl/ast"
)

// MissingInputsError is returned when required workflow inputs are missing from the inputs JSON.
type MissingInputsError struct {
	Missing []string
}

func (e *MissingInputsError) Error() string {
	return fmt.Sprintf("missing required inputs: %s", strings.Join(e.Missing, ", "))
}

// ValidateInputs checks that all required workflow inputs are present in the inputs JSON.
// A required input is one that is non-optional and has no default value.
// Returns nil if all required inputs are present or if the WDL cannot be parsed (lets Cromwell handle it).
func ValidateInputs(wdlSource []byte, inputsJSON []byte) error {
	doc, err := ParseBytes(wdlSource)
	if err != nil {
		return nil
	}

	if doc.Workflow == nil {
		return nil
	}

	required := requiredInputNames(doc.Workflow)
	if len(required) == 0 {
		return nil
	}

	provided, err := parseInputKeys(inputsJSON)
	if err != nil {
		return fmt.Errorf("failed to parse inputs JSON: %w", err)
	}

	prefix := doc.Workflow.Name + "."
	var missing []string
	for _, name := range required {
		qualified := prefix + name
		if _, ok := provided[qualified]; !ok {
			missing = append(missing, qualified)
		}
	}

	if len(missing) > 0 {
		return &MissingInputsError{Missing: missing}
	}

	return nil
}

// requiredInputNames returns the names of inputs that are required
// (non-optional type and no default expression).
func requiredInputNames(wf *ast.Workflow) []string {
	var names []string
	for _, input := range wf.Inputs {
		if input.Type != nil && !input.Type.Optional && input.Expression == nil {
			names = append(names, input.Name)
		}
	}
	return names
}

// parseInputKeys extracts all keys from the inputs JSON.
func parseInputKeys(data []byte) (map[string]struct{}, error) {
	if len(data) == 0 {
		return make(map[string]struct{}), nil
	}

	var inputs map[string]json.RawMessage
	if err := json.Unmarshal(data, &inputs); err != nil {
		return nil, err
	}

	keys := make(map[string]struct{}, len(inputs))
	for k := range inputs {
		keys[k] = struct{}{}
	}
	return keys, nil
}
