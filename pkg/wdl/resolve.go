package wdl

import (
	"sort"
	"strings"

	"github.com/lmtani/pumbaa/pkg/wdl/ast"
)

// SourceKind classifies where one leaf of an input's expression gets its value.
type SourceKind int

const (
	// SourceLiteral is a value fixed in the workflow text.
	SourceLiteral SourceKind = iota
	// SourceInput is a workflow-level input, readable from the inputs JSON.
	SourceInput
	// SourceCall is another call's output, which is what makes a cache miss
	// cascade.
	SourceCall
)

// ValueSource is one leaf of the expression behind a call input.
type ValueSource struct {
	Kind SourceKind
	// Name is the workflow input's name, or the producing call's path.
	Name string
	// Member is the output read from a producing call, for SourceCall.
	Member string
	// Literal carries the value, for SourceLiteral.
	Literal string
	// Scope qualifies an input that must be supplied under a call path rather
	// than at the top level, e.g. "Sub" for "Top.Sub.name".
	Scope string
}

// ResolvedBinding is a call input's expression reduced to the leaves its value
// can be built from.
//
// The central invariant: **a value is unchanged exactly when every leaf of its
// expression is unchanged** — provided every leaf was identified and only
// deterministic operators connect them. That covers conditionals too, because a
// predicate's own leaves are leaves of the expression: if they are unchanged the
// branch taken is the same one the reference took, and if the selected source is
// also unchanged the value is what was recorded.
//
// Complete reports whether that proviso holds. When it is false the sources are
// still real dependencies and may be used to propagate a rerun, but they cannot
// establish that a value is unchanged: something the walk could not see may also
// feed the value.
type ResolvedBinding struct {
	Sources  []ValueSource
	Complete bool
	// Incomplete explains the gap when Complete is false.
	Incomplete string
}

// Calls returns the paths of the producing calls among the sources.
func (b ResolvedBinding) Calls() []string {
	var out []string
	seen := make(map[string]bool)
	for _, s := range b.Sources {
		if s.Kind == SourceCall && !seen[s.Name] {
			seen[s.Name] = true
			out = append(out, s.Name)
		}
	}
	sort.Strings(out)
	return out
}

// pureFunctions is the allowlist of operators whose result is a deterministic
// function of their arguments, so that "every argument unchanged" implies
// "result unchanged".
//
// It is an allowlist rather than a blocklist on purpose: an unrecognised
// function makes the binding incomplete, which costs coverage, while an
// unrecognised *impure* function silently breaks the invariant. Each entry is a
// claim about the engine's semantics and should be added only with a test.
var pureFunctions = map[string]bool{
	"select_first": true,
	"select_all":   true,
	"defined":      true,
	"basename":     true,
	"length":       true,
	"sub":          true,
	"flatten":      true,
	"prefix":       true,
	"range":        true,
}

// resolver reduces expressions to their leaves within one workflow's scope.
type resolver struct {
	// declarations are the workflow's intermediate declarations, which an
	// expression may reference by name and which have to be followed.
	declarations map[string]ast.Expression
	inputs       map[string]bool
	// callNames are the calls visible in this workflow, used to tell a
	// producing call from any other identifier.
	callNames map[string]bool
	// prefix qualifies call paths when resolving inside a subworkflow.
	prefix string
}

// resolve reduces an expression to its leaves, following intermediate
// declarations to a bounded depth.
func (r *resolver) resolve(expr ast.Expression, depth int) ResolvedBinding {
	if depth > maxImportDepth {
		return incomplete("expression nests too deeply to follow")
	}
	if expr == nil {
		return incomplete("no expression")
	}

	switch e := expr.(type) {
	case *ast.Literal, *ast.StringLiteral:
		if v, ok := StaticValue(expr); ok {
			return complete(ValueSource{Kind: SourceLiteral, Literal: v})
		}
		return incomplete("literal of an unsupported type")

	case *ast.Identifier:
		return r.resolveIdentifier(e.Name, depth)

	case *ast.MemberAccess:
		// `Producer.out` is the shape that makes a dependency.
		if id, ok := e.Expression.(*ast.Identifier); ok && r.callNames[id.Name] {
			return complete(ValueSource{Kind: SourceCall, Name: r.prefix + id.Name, Member: e.Member})
		}
		// A member of anything else — a struct field, say — is not something
		// this walk models.
		return incomplete("member access on a value that is not a call output")

	case *ast.TernaryOp:
		// The condition's leaves matter as much as the branches': a predicate
		// that flips selects a different source even when both sources are
		// themselves unchanged.
		return r.merge(depth, e.Condition, e.IfTrue, e.IfFalse)

	case *ast.BinaryOp:
		return r.merge(depth, e.Left, e.Right)

	case *ast.UnaryOp:
		return r.merge(depth, e.Expression)

	case *ast.IndexAccess:
		// An element of a collection. The index participates: selecting a
		// different element yields a different value.
		return r.merge(depth, e.Expression, e.Index)

	case *ast.FunctionCall:
		if !pureFunctions[e.Name] {
			return incomplete("calls " + e.Name + "(), which is not known to be deterministic")
		}
		return r.merge(depth, e.Arguments...)

	case *ast.ArrayLiteral:
		return r.merge(depth, e.Elements...)

	case *ast.PairLiteral:
		return r.merge(depth, e.Left, e.Right)

	case *ast.ObjectLiteral:
		return r.mergeMap(depth, e.Members)

	case *ast.MapLiteral:
		var parts []ast.Expression
		for k, v := range e.Entries {
			parts = append(parts, k, v)
		}
		return r.merge(depth, parts...)

	case *ast.StringInterpolation:
		var parts []ast.Expression
		for _, p := range e.Parts {
			switch sp := p.(type) {
			case *ast.StringPlaceholder:
				parts = append(parts, sp.Expression)
			case *ast.StringLiteral:
				parts = append(parts, sp)
			}
		}
		return r.merge(depth, parts...)

	default:
		return incomplete("expression form not modelled")
	}
}

// resolveIdentifier classifies a bare name: a workflow input, an intermediate
// declaration to be followed, or something this walk does not model.
func (r *resolver) resolveIdentifier(name string, depth int) ResolvedBinding {
	if r.inputs[name] {
		return complete(ValueSource{Kind: SourceInput, Name: name})
	}
	if expr, ok := r.declarations[name]; ok {
		return r.resolve(expr, depth+1)
	}
	// A scatter variable, a name from an enclosing scope, or a declaration we
	// did not collect. Either way its value is not derivable here.
	return incomplete("reads " + name + ", which is not a workflow input")
}

func (r *resolver) merge(depth int, parts ...ast.Expression) ResolvedBinding {
	out := ResolvedBinding{Complete: true}
	for _, part := range parts {
		sub := r.resolve(part, depth+1)
		out.Sources = append(out.Sources, sub.Sources...)
		if !sub.Complete && out.Complete {
			out.Complete = false
			out.Incomplete = sub.Incomplete
		}
	}
	return dedupeSources(out)
}

func (r *resolver) mergeMap(depth int, members map[string]ast.Expression) ResolvedBinding {
	names := make([]string, 0, len(members))
	for name := range members {
		names = append(names, name)
	}
	sort.Strings(names)
	parts := make([]ast.Expression, 0, len(names))
	for _, name := range names {
		parts = append(parts, members[name])
	}
	return r.merge(depth, parts...)
}

func complete(sources ...ValueSource) ResolvedBinding {
	return ResolvedBinding{Sources: sources, Complete: true}
}

func incomplete(why string) ResolvedBinding {
	return ResolvedBinding{Incomplete: why}
}

// dedupeSources collapses repeated leaves — a name read twice in one expression
// is one source — and orders them so output is stable.
func dedupeSources(b ResolvedBinding) ResolvedBinding {
	if len(b.Sources) < 2 {
		return b
	}
	seen := make(map[ValueSource]bool, len(b.Sources))
	out := b.Sources[:0]
	for _, s := range b.Sources {
		if seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	b.Sources = out
	sort.Slice(b.Sources, func(i, j int) bool {
		if b.Sources[i].Kind != b.Sources[j].Kind {
			return b.Sources[i].Kind < b.Sources[j].Kind
		}
		if b.Sources[i].Name != b.Sources[j].Name {
			return b.Sources[i].Name < b.Sources[j].Name
		}
		return b.Sources[i].Member < b.Sources[j].Member
	})
	return b
}

// translate re-expresses a binding resolved inside a subworkflow in the terms of
// the top-level workflow: an input the enclosing call supplied adopts that
// call's own sources, one it did not falls back to the subworkflow's default and
// then to a call-scoped lookup.
func translate(b ResolvedBinding, prefix string, outer map[string]ResolvedBinding, defaults map[string]string) ResolvedBinding {
	if outer == nil && prefix == "" {
		return b
	}
	out := ResolvedBinding{Complete: b.Complete, Incomplete: b.Incomplete}
	for _, s := range b.Sources {
		switch s.Kind {
		case SourceInput:
			if outerBinding, ok := outer[s.Name]; ok {
				out.Sources = append(out.Sources, outerBinding.Sources...)
				if !outerBinding.Complete && out.Complete {
					out.Complete = false
					out.Incomplete = outerBinding.Incomplete
				}
				continue
			}
			if def, ok := defaults[s.Name]; ok {
				out.Sources = append(out.Sources, ValueSource{Kind: SourceLiteral, Literal: def})
				continue
			}
			s.Scope = strings.TrimSuffix(prefix, ".")
			out.Sources = append(out.Sources, s)
		default:
			out.Sources = append(out.Sources, s)
		}
	}
	return dedupeSources(out)
}
