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

	var missing []string
	for _, name := range required {
		if _, ok := provided[name]; !ok {
			missing = append(missing, name)
		}
	}

	if len(missing) > 0 {
		return &MissingInputsError{Missing: missing}
	}

	return nil
}

// requiredInputNames returns the qualified names of the inputs that must be
// provided, using the same notion of "required" as InputSpec.
func requiredInputNames(wf *ast.Workflow) []string {
	var names []string
	for _, spec := range workflowInputSpecs(wf) {
		if spec.Required() {
			names = append(names, spec.Name)
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
