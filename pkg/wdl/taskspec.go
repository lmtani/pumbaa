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
// template under `callCaching.hashes["command template"]`, and reports whether
// that reproduction can be trusted for this particular command.
//
// The normalisation was derived by matching real Cromwell metadata (see
// docs/design/cache-explainer.md): the command is dedented by its common
// leading whitespace — relative indentation is preserved — and `~{expr}`
// placeholders are rewritten to `${expr}`.
//
// The catch is that Cromwell does not hash the placeholder text verbatim: it
// re-serialises the parsed expression. For a bare reference that round-trip is
// the identity, so the text transform is exact. For anything richer — a
// conditional, arithmetic, a function call — the canonical form is unknown to
// us and a mismatch would say more about our rendering than about the command.
// ok is false in exactly those cases, and callers must then skip the comparison
// rather than report a change.
//
// Measured against a real 15-task production pipeline, this predicate never
// admitted a command whose hash it then got wrong; it only declines a couple it
// would in fact have reproduced.
//
// Computing this locally is what makes a command change detectable without the
// reference run's WDL source, which matters because a run's metadata carries
// only the top-level workflow, never its imported files.
func (t TaskSpec) CommandHash() (hash string, ok bool) {
	return CommandTemplateHash(t.Command)
}

// CommandTemplateHash hashes a raw WDL command template the way Cromwell does,
// reporting whether every placeholder in it is one we can render canonically.
func CommandTemplateHash(command string) (hash string, ok bool) {
	normalized := placeholderPattern.ReplaceAllString(dedentCommand(command), "${$1}")
	sum := md5.Sum([]byte(normalized)) //nolint:gosec // reproducing Cromwell's hash, not a security primitive
	return strings.ToUpper(hex.EncodeToString(sum[:])), canonicalPlaceholders(command)
}

// dedentCommand removes the whitespace prefix common to every non-blank line,
// preserving relative indentation.
func dedentCommand(command string) string {
	lines := strings.Split(strings.Trim(command, "\n"), "\n")
	indent := -1
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		n := len(line) - len(strings.TrimLeft(line, " \t"))
		if indent < 0 || n < indent {
			indent = n
		}
	}
	if indent > 0 {
		for i, line := range lines {
			if len(line) >= indent {
				lines[i] = line[indent:]
			} else {
				lines[i] = strings.TrimLeft(line, " \t")
			}
		}
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

// canonicalPlaceholders reports whether every interpolation in the command is a
// bare reference, optionally behind a placeholder option (`sep=`, `default=`,
// `true=`/`false=`). Those are the forms whose textual rewrite is exactly what
// Cromwell hashes.
func canonicalPlaceholders(command string) bool {
	for _, form := range []*regexp.Regexp{placeholderPattern, dollarPlaceholderPattern} {
		for _, m := range form.FindAllStringSubmatch(command, -1) {
			body := strings.TrimSpace(m[1])
			body = strings.TrimSpace(placeholderOptionPattern.ReplaceAllString(body, ""))
			if !bareReferencePattern.MatchString(body) {
				return false
			}
		}
	}
	return true
}

var (
	// placeholderPattern matches WDL 1.0 `~{...}` interpolation, which Cromwell
	// normalises to the `${...}` form before hashing.
	placeholderPattern = regexp.MustCompile(`~\{([^}]*)\}`)
	// dollarPlaceholderPattern matches the older `${...}` interpolation. In a
	// heredoc command it may equally be a shell variable, which is harmless:
	// a plain `${VAR}` passes the bare-reference test either way, and anything
	// more elaborate only makes the check more conservative.
	dollarPlaceholderPattern = regexp.MustCompile(`\$\{([^}]*)\}`)
	// placeholderOptionPattern strips a leading placeholder option so the
	// reference behind it can be examined.
	placeholderOptionPattern = regexp.MustCompile(`^(sep|default|true|false)\s*=\s*("[^"]*"|'[^']*'|\S+)\s*`)
	bareReferencePattern     = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_.]*$`)
)

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
