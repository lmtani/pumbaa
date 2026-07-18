package wdl

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"

	"github.com/lmtani/pumbaa/pkg/wdl/ast"
)

// TaskSpec is the part of a task definition that feeds Cromwell's call-cache
// fingerprint and can be read from the WDL text alone.
//
// Command is the raw, un-substituted command template. Cromwell hashes it
// before interpolating input values — verified against a real server, see
// docs/design/cache-explainer.md — so comparing this text between two WDLs is
// equivalent to comparing the hash, without reimplementing the hashing.
type TaskSpec struct {
	Name    string
	Command string
	// Runtime holds runtime attributes whose value is a literal in the WDL.
	Runtime map[string]string
	// DynamicRuntime lists attributes whose value depends on an input, so it
	// cannot be resolved from the WDL alone (the classic case is
	// `docker: docker`, where docker is a task input with a default).
	DynamicRuntime map[string]string
	// InputDefaults holds statically-resolvable defaults for task inputs.
	InputDefaults map[string]string
}

// CommandHash reproduces the hash Cromwell records for this task's command
// template under `callCaching.hashes["command template"]`.
//
// The normalisation was derived by matching against real Cromwell 91 metadata
// (see docs/design/cache-explainer.md): each line is trimmed and rejoined with
// newlines, and `~{expr}` placeholders are rewritten to `${expr}`. It matches
// every task in the captured fixtures, including a run where the docker image
// changed and the command did not.
//
// Being able to compute this locally means a command change is detectable
// without the reference run's WDL source — which matters because a run's
// metadata carries only the top-level workflow, never its imported files.
// Callers should still confirm the formula reproduces at least one known hash
// before trusting a mismatch, since the normalisation is not a documented
// contract.
func (t TaskSpec) CommandHash() string {
	return CommandTemplateHash(t.Command)
}

// CommandTemplateHash hashes a raw WDL command template the way Cromwell does.
func CommandTemplateHash(command string) string {
	lines := strings.Split(strings.TrimSpace(command), "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSpace(line)
	}
	normalized := placeholderPattern.ReplaceAllString(strings.Join(lines, "\n"), "${$1}")
	sum := md5.Sum([]byte(normalized)) //nolint:gosec // reproducing Cromwell's hash, not a security primitive
	return strings.ToUpper(hex.EncodeToString(sum[:]))
}

// placeholderPattern matches WDL 1.0 `~{...}` interpolation, which Cromwell
// normalises to the `${...}` form before hashing.
var placeholderPattern = regexp.MustCompile(`~\{([^}]*)\}`)

// DockerValue resolves the task's docker image, preferring a literal in the
// runtime section and falling back to the default of the input the runtime
// references. ok is false when the image cannot be determined from the WDL.
func (t TaskSpec) DockerValue() (string, bool) {
	if v, ok := t.Runtime["docker"]; ok {
		return v, true
	}
	if ref, ok := t.DynamicRuntime["docker"]; ok {
		if def, ok := t.InputDefaults[ref]; ok {
			return def, true
		}
	}
	return "", false
}

// TaskSpecs extracts every task definition from WDL source, keyed by task name.
// Imports are not resolved; use TaskSpecsWithSources to include imported tasks.
func TaskSpecs(source []byte) (map[string]TaskSpec, error) {
	return TaskSpecsWithSources(source, nil)
}

// TaskSpecsWithSources extracts task definitions from the source and, walking
// its imports transitively, from every document it can resolve.
//
// Tasks are keyed by bare name because that is how a call addresses them once
// the namespace is stripped. A name collision across imported files keeps the
// definition nearest the root, which is the one a reader would assume wins.
func TaskSpecsWithSources(source []byte, deps SourceSet) (map[string]TaskSpec, error) {
	doc, err := ParseBytes(source)
	if err != nil {
		return nil, err
	}

	out := TaskSpecsFromDocument(doc)
	docs := newDocumentSet(deps)

	// Breadth-first so nearer definitions are added first and shadow deeper ones.
	frontier := []*ast.Document{doc}
	visited := make(map[string]bool)
	for depth := 0; depth < maxImportDepth && len(frontier) > 0; depth++ {
		var next []*ast.Document
		for _, d := range frontier {
			for _, imp := range d.Imports {
				if imp == nil || visited[imp.URI] {
					continue
				}
				visited[imp.URI] = true
				imported, ok := docs.document(imp.URI)
				if !ok {
					continue
				}
				for name, spec := range TaskSpecsFromDocument(imported) {
					if _, exists := out[name]; !exists {
						out[name] = spec
					}
				}
				next = append(next, imported)
			}
		}
		frontier = next
	}
	return out, nil
}

// TaskSpecsFromDocument extracts task specs from an already-parsed document.
func TaskSpecsFromDocument(doc *ast.Document) map[string]TaskSpec {
	out := make(map[string]TaskSpec)
	if doc == nil {
		return out
	}
	for _, t := range doc.Tasks {
		if t == nil {
			continue
		}
		spec := TaskSpec{
			Name:           t.Name,
			Command:        t.Command,
			Runtime:        make(map[string]string),
			DynamicRuntime: make(map[string]string),
			InputDefaults:  make(map[string]string),
		}
		for attr, expr := range t.Runtime {
			if v, ok := StaticValue(expr); ok {
				spec.Runtime[attr] = v
				continue
			}
			if id, ok := expr.(*ast.Identifier); ok {
				spec.DynamicRuntime[attr] = id.Name
			} else {
				spec.DynamicRuntime[attr] = ""
			}
		}
		for _, in := range t.Inputs {
			if in == nil || in.Expression == nil {
				continue
			}
			if v, ok := StaticValue(in.Expression); ok {
				spec.InputDefaults[in.Name] = v
			}
		}
		out[t.Name] = spec
	}
	return out
}

// StaticValue renders an expression whose value is fixed in the WDL text.
// It returns ok=false for anything that depends on inputs or runtime state,
// so callers can degrade to "cannot determine" instead of comparing garbage.
func StaticValue(e ast.Expression) (string, bool) {
	switch v := e.(type) {
	case *ast.Literal:
		return literalString(v.Value)
	case *ast.StringLiteral:
		return v.Value, true
	case *ast.StringInterpolation:
		// Only fully-literal interpolations are static; a single placeholder
		// makes the whole string input-dependent.
		var b strings.Builder
		for _, part := range v.Parts {
			lit, ok := part.(*ast.StringLiteral)
			if !ok {
				return "", false
			}
			b.WriteString(lit.Value)
		}
		return b.String(), true
	default:
		return "", false
	}
}

func literalString(v any) (string, bool) {
	switch t := v.(type) {
	case string:
		return t, true
	case bool:
		return fmt.Sprintf("%t", t), true
	case int:
		return fmt.Sprintf("%d", t), true
	case int64:
		return fmt.Sprintf("%d", t), true
	case float64:
		return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%f", t), "0"), "."), true
	default:
		return "", false
	}
}
