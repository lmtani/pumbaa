package wdl

import (
	"sort"
	"strings"

	"github.com/lmtani/pumbaa/pkg/wdl/ast"
)

// CallNode is one call in a workflow, with the information needed to reason
// about whether it can be served from the call cache.
type CallNode struct {
	// Name is how the call is addressed in the workflow: its alias when it has
	// one, otherwise the task name.
	Name string
	// Task is the name of the task or subworkflow being called, stripped of
	// any import namespace.
	Task string
	// DependsOn lists the calls whose outputs this call consumes, sorted.
	DependsOn []string
	// Scattered marks a call inside a scatter block. Its shard count is a
	// runtime property, so a prediction for it covers the call as a whole and
	// cannot speak to individual shards.
	Scattered bool
	// Subworkflow marks a call whose target is not a task in this document —
	// typically an imported workflow, whose internals this graph does not see.
	Subworkflow bool
	// Bindings records where each of the call's inputs gets its value, which
	// is what lets a caller decide whether that value changed since a previous
	// run without evaluating the workflow.
	Bindings map[string]CallBinding
}

// BindingKind is the origin of a call input's value.
type BindingKind int

const (
	// BindingUnknown covers expressions this package will not evaluate —
	// function calls, interpolations, arithmetic. A call with one of these is
	// not statically comparable.
	BindingUnknown BindingKind = iota
	// BindingWorkflowInput means the value comes from a workflow-level input,
	// so it can be read from the inputs JSON.
	BindingWorkflowInput
	// BindingLiteral means the value is fixed in the WDL text.
	BindingLiteral
	// BindingCall means the value is another call's output, which is what
	// propagates a cache miss downstream.
	BindingCall
)

// CallBinding describes where one call input's value comes from.
type CallBinding struct {
	Kind BindingKind
	// Source names the workflow input or the producing call, depending on Kind.
	Source string
	// Literal carries the value when Kind is BindingLiteral.
	Literal string
}

// CallGraph is a workflow's calls indexed by name.
type CallGraph struct {
	Workflow string
	Nodes    map[string]*CallNode
}

// Dependencies returns the graph as a plain call → upstream calls map, the
// form the domain layer consumes for cache prediction.
func (g *CallGraph) Dependencies() map[string][]string {
	out := make(map[string][]string, len(g.Nodes))
	for name, n := range g.Nodes {
		out[name] = n.DependsOn
	}
	return out
}

// Names returns every call name, sorted.
func (g *CallGraph) Names() []string {
	out := make([]string, 0, len(g.Nodes))
	for name := range g.Nodes {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// BuildCallGraph extracts the dependency graph between a workflow's calls from
// WDL source. A call depends on another when it references that call's outputs
// (`Other.out`), which is exactly what makes the cache cascade: if the producer
// reruns, the consumer's inputs change.
//
// Calls nested in scatter and conditional blocks are included. Imported
// subworkflows are represented as single opaque nodes — their internal calls
// are not visible without resolving the import.
func BuildCallGraph(source []byte) (*CallGraph, error) {
	doc, err := ParseBytes(source)
	if err != nil {
		return nil, err
	}
	return CallGraphFromDocument(doc), nil
}

// CallGraphFromDocument builds the graph from an already-parsed document,
// for callers that have one in hand.
func CallGraphFromDocument(doc *ast.Document) *CallGraph {
	g := &CallGraph{Nodes: make(map[string]*CallNode)}
	if doc == nil || doc.Workflow == nil {
		return g
	}
	g.Workflow = doc.Workflow.Name

	localTasks := make(map[string]bool, len(doc.Tasks))
	for _, t := range doc.Tasks {
		localTasks[t.Name] = true
	}

	// Collect every call first: a call may reference one declared later in the
	// source, so dependencies can only be resolved once all names are known.
	collectCalls(doc.Workflow.Calls, false, localTasks, g)
	for _, s := range doc.Workflow.Scatters {
		collectFromBody(s.Body, true, localTasks, g)
	}
	for _, c := range doc.Workflow.Conditionals {
		collectFromBody(c.Body, false, localTasks, g)
	}

	resolveDependencies(doc.Workflow, g)
	return g
}

func collectCalls(calls []*ast.Call, scattered bool, localTasks map[string]bool, g *CallGraph) {
	for _, c := range calls {
		if c == nil {
			continue
		}
		task := targetName(c.Target)
		name := c.Alias
		if name == "" {
			name = task
		}
		// A call already collected at top level may reappear when walking a
		// scatter body; keep the scattered flag rather than overwriting it.
		if existing, ok := g.Nodes[name]; ok {
			existing.Scattered = existing.Scattered || scattered
			continue
		}
		g.Nodes[name] = &CallNode{
			Name:        name,
			Task:        task,
			Scattered:   scattered,
			Subworkflow: !localTasks[task],
		}
	}
}

func collectFromBody(body []ast.WorkflowElement, scattered bool, localTasks map[string]bool, g *CallGraph) {
	for _, el := range body {
		switch e := el.(type) {
		case *ast.Call:
			collectCalls([]*ast.Call{e}, scattered, localTasks, g)
		case *ast.Scatter:
			collectFromBody(e.Body, true, localTasks, g)
		case *ast.Conditional:
			collectFromBody(e.Body, scattered, localTasks, g)
		}
	}
}

// classifyBinding decides where a call input's value comes from.
func classifyBinding(expr ast.Expression, callName string, g *CallGraph, workflowInputs map[string]bool) CallBinding {
	if v, ok := StaticValue(expr); ok {
		return CallBinding{Kind: BindingLiteral, Literal: v}
	}
	switch e := expr.(type) {
	case *ast.Identifier:
		if workflowInputs[e.Name] {
			return CallBinding{Kind: BindingWorkflowInput, Source: e.Name}
		}
		// An identifier that is not a workflow input is a private declaration
		// or a scatter variable, neither of which we evaluate.
		return CallBinding{Kind: BindingUnknown, Source: e.Name}
	case *ast.MemberAccess:
		if id, ok := e.Expression.(*ast.Identifier); ok && id.Name != callName && g.Nodes[id.Name] != nil {
			return CallBinding{Kind: BindingCall, Source: id.Name}
		}
	}
	return CallBinding{Kind: BindingUnknown}
}

// resolveDependencies walks every call's input expressions a second time,
// recording references to other calls in the graph.
func resolveDependencies(wf *ast.Workflow, g *CallGraph) {
	workflowInputs := make(map[string]bool, len(wf.Inputs))
	for _, in := range wf.Inputs {
		if in != nil {
			workflowInputs[in.Name] = true
		}
	}

	walk := func(calls []*ast.Call) {
		for _, c := range calls {
			if c == nil {
				continue
			}
			name := c.Alias
			if name == "" {
				name = targetName(c.Target)
			}
			node, ok := g.Nodes[name]
			if !ok {
				continue
			}
			deps := make(map[string]bool)
			node.Bindings = make(map[string]CallBinding, len(c.Inputs))
			for inputName, expr := range c.Inputs {
				node.Bindings[inputName] = classifyBinding(expr, name, g, workflowInputs)
				for _, ref := range referencedCalls(expr) {
					// A call never depends on itself, and only calls in the
					// graph count — other identifiers are workflow inputs or
					// declarations.
					if ref != name && g.Nodes[ref] != nil {
						deps[ref] = true
					}
				}
			}
			// `call X after Y` is an explicit ordering dependency.
			for _, after := range c.After {
				if after != name && g.Nodes[after] != nil {
					deps[after] = true
				}
			}
			for d := range deps {
				node.DependsOn = append(node.DependsOn, d)
			}
			sort.Strings(node.DependsOn)
		}
	}

	walk(wf.Calls)
	for _, s := range wf.Scatters {
		walkBody(s.Body, g, walk)
	}
	for _, c := range wf.Conditionals {
		walkBody(c.Body, g, walk)
	}
}

func walkBody(body []ast.WorkflowElement, g *CallGraph, walk func([]*ast.Call)) {
	for _, el := range body {
		switch e := el.(type) {
		case *ast.Call:
			walk([]*ast.Call{e})
		case *ast.Scatter:
			walkBody(e.Body, g, walk)
		case *ast.Conditional:
			walkBody(e.Body, g, walk)
		}
	}
}

// referencedCalls returns the identifiers an expression accesses members of,
// which is how a WDL call references another call's outputs (`Other.out`).
func referencedCalls(expr ast.Expression) []string {
	var out []string
	var visit func(ast.Expression)
	visit = func(e ast.Expression) {
		switch v := e.(type) {
		case *ast.MemberAccess:
			// `A.out` — the base identifier names the producing call. A deeper
			// chain (`A.out.field`) still roots at that identifier.
			if id, ok := v.Expression.(*ast.Identifier); ok {
				out = append(out, id.Name)
			} else {
				visit(v.Expression)
			}
		case *ast.IndexAccess:
			visit(v.Expression)
			visit(v.Index)
		case *ast.BinaryOp:
			visit(v.Left)
			visit(v.Right)
		case *ast.UnaryOp:
			visit(v.Expression)
		case *ast.FunctionCall:
			for _, a := range v.Arguments {
				visit(a)
			}
		case *ast.TernaryOp:
			visit(v.Condition)
			visit(v.IfTrue)
			visit(v.IfFalse)
		case *ast.ArrayLiteral:
			for _, e := range v.Elements {
				visit(e)
			}
		case *ast.MapLiteral:
			for k, val := range v.Entries {
				visit(k)
				visit(val)
			}
		case *ast.PairLiteral:
			visit(v.Left)
			visit(v.Right)
		case *ast.ObjectLiteral:
			for _, m := range v.Members {
				visit(m)
			}
		case *ast.StringInterpolation:
			// `"~{Other.out}"` references a call from inside a string.
			for _, part := range v.Parts {
				if ph, ok := part.(*ast.StringPlaceholder); ok {
					visit(ph.Expression)
				}
			}
		}
	}
	visit(expr)
	return out
}

// targetName strips an import namespace from a call target ("lib.Task" → "Task").
func targetName(target string) string {
	if i := strings.LastIndex(target, "."); i >= 0 {
		return target[i+1:]
	}
	return target
}
