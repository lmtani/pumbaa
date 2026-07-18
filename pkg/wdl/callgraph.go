package wdl

import (
	"slices"
	"sort"
	"strings"

	"github.com/lmtani/pumbaa/pkg/wdl/ast"
)

// CallNode is one call in a workflow, with the information needed to reason
// about whether it can be served from the call cache.
//
// Calls inside subworkflows appear here too, flattened and addressed by their
// path ("RunSample.AlignReads"), because Cromwell caches leaf tasks — a
// subworkflow call is not itself a cacheable unit.
type CallNode struct {
	// Name is the call's path from the top-level workflow: its alias when it
	// has one, otherwise the task name, prefixed by the subworkflow calls it
	// sits under.
	Name string
	// Task is the name of the task being called, without any import namespace.
	Task string
	// DependsOn lists the calls whose outputs this call consumes, sorted and
	// expressed as paths into the flattened graph.
	DependsOn []string
	// Scattered marks a call inside a scatter block. Its shard count is a
	// runtime property, so a verdict for it covers the call as a whole and
	// cannot speak to individual shards.
	Scattered bool
	// Unresolved marks a call whose definition could not be read — an import
	// missing from the sources, or a file that did not parse. Its task body is
	// invisible, so a change to its command cannot be detected and callers
	// must withhold judgement rather than assume nothing changed.
	Unresolved bool
	// Subworkflow marks an unresolved call that targets a workflow rather than
	// a task. Resolved subworkflows are flattened away and never appear.
	Subworkflow bool
	// Bindings records where each of the call's inputs gets its value, already
	// translated into the top-level workflow's namespace.
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
	// Member is the output name read from the producing call, for BindingCall.
	Member string
	// Literal carries the value when Kind is BindingLiteral.
	Literal string
	// Scope is the call path a BindingWorkflowInput belongs to when the value
	// was not supplied by the enclosing call — an input the user must provide
	// as "Top.Sub.name". Empty means a top-level workflow input.
	Scope string
}

// CallGraph is a workflow's calls, flattened across subworkflows and indexed
// by path.
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

// Names returns every call path, sorted.
func (g *CallGraph) Names() []string {
	out := make([]string, 0, len(g.Nodes))
	for name := range g.Nodes {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// BuildCallGraph extracts the dependency graph between a workflow's calls from
// WDL source alone. Imports cannot be resolved without their sources, so calls
// into them are marked Unresolved; use BuildCallGraphWithSources to see inside.
func BuildCallGraph(source []byte) (*CallGraph, error) {
	return BuildCallGraphWithSources(source, nil)
}

// BuildCallGraphWithSources extracts the dependency graph, resolving imports
// against the given sources.
//
// A call depends on another when it references that call's outputs
// (`Other.out`), which is exactly what makes the cache cascade: if the producer
// reruns, the consumer's inputs change. Calls nested in scatter and conditional
// blocks are included, and resolved subworkflows are flattened into their leaf
// calls so every node in the graph is something Cromwell can actually cache.
func BuildCallGraphWithSources(source []byte, deps SourceSet) (*CallGraph, error) {
	doc, err := ParseBytes(source)
	if err != nil {
		return nil, err
	}
	return CallGraphFromDocument(doc, deps), nil
}

// CallGraphFromDocument builds the graph from an already-parsed document.
func CallGraphFromDocument(doc *ast.Document, deps SourceSet) *CallGraph {
	g := &CallGraph{Nodes: make(map[string]*CallNode)}
	if doc == nil || doc.Workflow == nil {
		return g
	}
	g.Workflow = doc.Workflow.Name

	b := &graphBuilder{
		docs:       newDocumentSet(deps),
		graph:      g,
		subOutputs: make(map[string]map[string]string),
	}
	b.addWorkflow(doc, "", nil, 0)
	b.rewireSubworkflowOutputs()
	b.deriveDependencies()
	return g
}

// graphBuilder flattens a workflow, and the workflows it calls, into one graph.
type graphBuilder struct {
	docs  *documentSet
	graph *CallGraph
	// subOutputs maps a flattened subworkflow call path to its workflow
	// outputs and the leaf call that produces each, so a consumer of
	// `Sub.result` ends up depending on the leaf rather than on all of them.
	subOutputs map[string]map[string]string
	// pendingAfter holds explicit `after` edges until every node exists.
	pendingAfter []afterEdge
}

type afterEdge struct{ from, to string }

// addWorkflow adds nodes for every call in wf, under the given path prefix.
//
// outer maps this workflow's input names to bindings already expressed in the
// top-level namespace; it is how a value passed into a subworkflow is followed
// through to the leaf that consumes it.
func (b *graphBuilder) addWorkflow(doc *ast.Document, prefix string, outer map[string]CallBinding, depth int) {
	if depth > maxImportDepth || doc.Workflow == nil {
		return
	}
	wf := doc.Workflow
	ns := namespaces(doc)
	localTasks := make(map[string]bool, len(doc.Tasks))
	for _, t := range doc.Tasks {
		if t != nil {
			localTasks[t.Name] = true
		}
	}
	workflowInputs := declaredInputs(wf.Inputs)
	defaults := staticDefaults(wf.Inputs)

	for _, c := range collectCalls(wf) {
		b.addCall(c, ns, localTasks, workflowInputs, defaults, prefix, outer, depth)
	}
}

func (b *graphBuilder) addCall(
	c scopedCall,
	ns map[string]string,
	localTasks map[string]bool,
	workflowInputs map[string]bool,
	defaults map[string]string,
	prefix string,
	outer map[string]CallBinding,
	depth int,
) {
	namespace, target := splitTarget(c.call.Target)
	name := c.call.Alias
	if name == "" {
		name = target
	}
	path := prefix + name

	// Bindings are first read in this workflow's own namespace, then followed
	// outward so every node speaks in top-level terms.
	local := make(map[string]CallBinding, len(c.call.Inputs))
	for inputName, expr := range c.call.Inputs {
		local[inputName] = classifyBinding(expr, name, workflowInputs)
	}
	translated := make(map[string]CallBinding, len(local))
	for inputName, binding := range local {
		translated[inputName] = b.translate(binding, prefix, outer, defaults)
	}

	for _, after := range c.call.After {
		b.pendingAfter = append(b.pendingAfter, afterEdge{from: path, to: prefix + after})
	}

	sub, isSub := b.resolveSubworkflow(namespace, target, ns)
	if isSub {
		if sub == nil {
			// A workflow we cannot read: keep it whole and opaque rather than
			// pretending it has no calls.
			b.graph.Nodes[path] = &CallNode{
				Name: path, Task: target, Scattered: c.scattered,
				Unresolved: true, Subworkflow: true, Bindings: translated,
			}
			return
		}
		b.addWorkflow(sub, path+".", translated, depth+1)
		b.subOutputs[path] = subworkflowOutputs(sub.Workflow, path+".")
		return
	}

	b.graph.Nodes[path] = &CallNode{
		Name:       path,
		Task:       target,
		Scattered:  c.scattered,
		Unresolved: !b.taskIsVisible(namespace, target, ns, localTasks),
		Bindings:   translated,
	}
}

// resolveSubworkflow reports whether a call targets a workflow rather than a
// task, and returns its document when the source is available.
func (b *graphBuilder) resolveSubworkflow(namespace, target string, ns map[string]string) (*ast.Document, bool) {
	if namespace == "" {
		// Without a namespace the target is a task in this document; a
		// workflow cannot be called without importing it.
		return nil, false
	}
	uri, ok := ns[namespace]
	if !ok {
		// An unknown namespace: we cannot tell task from workflow, so treat it
		// as an unreadable subworkflow — the conservative reading, since it
		// withholds a verdict instead of inventing one.
		return nil, true
	}
	doc, ok := b.docs.document(uri)
	if !ok {
		return nil, true
	}
	for _, t := range doc.Tasks {
		if t != nil && t.Name == target {
			return nil, false
		}
	}
	if doc.Workflow != nil && doc.Workflow.Name == target {
		return doc, true
	}
	// Present but neither a task nor the workflow we expected.
	return nil, true
}

// taskIsVisible reports whether the called task's definition can actually be
// read, which decides whether a command or docker change is detectable.
func (b *graphBuilder) taskIsVisible(namespace, target string, ns map[string]string, localTasks map[string]bool) bool {
	if namespace == "" {
		return localTasks[target]
	}
	uri, ok := ns[namespace]
	if !ok {
		return false
	}
	doc, ok := b.docs.document(uri)
	if !ok {
		return false
	}
	for _, t := range doc.Tasks {
		if t != nil && t.Name == target {
			return true
		}
	}
	return false
}

// translate expresses a binding read inside a workflow in the terms of the
// top-level workflow.
func (b *graphBuilder) translate(binding CallBinding, prefix string, outer map[string]CallBinding, defaults map[string]string) CallBinding {
	switch binding.Kind {
	case BindingCall:
		// The producing call is a sibling, so it lives under the same prefix.
		binding.Source = prefix + binding.Source
		return binding
	case BindingWorkflowInput:
		if outer == nil {
			return binding
		}
		if outerBinding, ok := outer[binding.Source]; ok {
			// The enclosing call supplied this input: adopt its origin.
			return outerBinding
		}
		if def, ok := defaults[binding.Source]; ok {
			// Not supplied, but the subworkflow declares a default.
			return CallBinding{Kind: BindingLiteral, Literal: def}
		}
		// Not supplied and no default: the user must provide it qualified by
		// the call path, e.g. "Top.Sub.name".
		binding.Scope = strings.TrimSuffix(prefix, ".")
		return binding
	default:
		return binding
	}
}

// rewireSubworkflowOutputs replaces dependencies on a flattened subworkflow
// call with dependencies on the leaf that actually produces the value, so a
// consumer does not inherit a rerun from an unrelated part of the subworkflow.
func (b *graphBuilder) rewireSubworkflowOutputs() {
	for _, node := range b.graph.Nodes {
		for inputName, binding := range node.Bindings {
			if binding.Kind != BindingCall {
				continue
			}
			outputs, ok := b.subOutputs[binding.Source]
			if !ok {
				continue
			}
			producer, ok := outputs[binding.Member]
			if !ok {
				// The output is not one we could trace to a leaf; without a
				// producer the value's origin is unknown.
				node.Bindings[inputName] = CallBinding{Kind: BindingUnknown, Source: binding.Source}
				continue
			}
			binding.Source = producer
			node.Bindings[inputName] = binding
		}
	}
}

// deriveDependencies recomputes every edge from the final bindings, so the
// graph cannot disagree with the bindings it was built from.
func (b *graphBuilder) deriveDependencies() {
	for _, node := range b.graph.Nodes {
		deps := make(map[string]bool)
		for _, binding := range node.Bindings {
			if binding.Kind == BindingCall && binding.Source != node.Name && b.graph.Nodes[binding.Source] != nil {
				deps[binding.Source] = true
			}
		}
		node.DependsOn = node.DependsOn[:0]
		for d := range deps {
			node.DependsOn = append(node.DependsOn, d)
		}
		sort.Strings(node.DependsOn)
	}
	for _, edge := range b.pendingAfter {
		node, ok := b.graph.Nodes[edge.from]
		if !ok || b.graph.Nodes[edge.to] == nil {
			continue
		}
		if !slices.Contains(node.DependsOn, edge.to) {
			node.DependsOn = append(node.DependsOn, edge.to)
			sort.Strings(node.DependsOn)
		}
	}
}

// subworkflowOutputs maps a workflow's declared outputs to the calls producing
// them, addressed under the given prefix.
func subworkflowOutputs(wf *ast.Workflow, prefix string) map[string]string {
	out := make(map[string]string, len(wf.Outputs))
	for _, decl := range wf.Outputs {
		if decl == nil || decl.Expression == nil {
			continue
		}
		if ma, ok := decl.Expression.(*ast.MemberAccess); ok {
			if id, ok := ma.Expression.(*ast.Identifier); ok {
				out[decl.Name] = prefix + id.Name
			}
		}
	}
	return out
}

// scopedCall is a call plus whether it sits inside a scatter block.
type scopedCall struct {
	call      *ast.Call
	scattered bool
}

// collectCalls gathers every call in a workflow, including those nested in
// scatter and conditional blocks.
func collectCalls(wf *ast.Workflow) []scopedCall {
	seen := make(map[*ast.Call]bool)
	var out []scopedCall

	add := func(c *ast.Call, scattered bool) {
		if c == nil || seen[c] {
			return
		}
		seen[c] = true
		out = append(out, scopedCall{call: c, scattered: scattered})
	}

	var walk func(body []ast.WorkflowElement, scattered bool)
	walk = func(body []ast.WorkflowElement, scattered bool) {
		for _, el := range body {
			switch e := el.(type) {
			case *ast.Call:
				add(e, scattered)
			case *ast.Scatter:
				walk(e.Body, true)
			case *ast.Conditional:
				walk(e.Body, scattered)
			}
		}
	}

	for _, c := range wf.Calls {
		add(c, false)
	}
	for _, s := range wf.Scatters {
		walk(s.Body, true)
	}
	for _, c := range wf.Conditionals {
		walk(c.Body, false)
	}
	return out
}

// classifyBinding decides where a call input's value comes from, in the terms
// of the workflow the call is written in.
func classifyBinding(expr ast.Expression, callName string, workflowInputs map[string]bool) CallBinding {
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
		if id, ok := e.Expression.(*ast.Identifier); ok && id.Name != callName {
			return CallBinding{Kind: BindingCall, Source: id.Name, Member: e.Member}
		}
	}
	return CallBinding{Kind: BindingUnknown}
}

func declaredInputs(decls []*ast.Declaration) map[string]bool {
	out := make(map[string]bool, len(decls))
	for _, d := range decls {
		if d != nil {
			out[d.Name] = true
		}
	}
	return out
}

func staticDefaults(decls []*ast.Declaration) map[string]string {
	out := make(map[string]string, len(decls))
	for _, d := range decls {
		if d == nil || d.Expression == nil {
			continue
		}
		if v, ok := StaticValue(d.Expression); ok {
			out[d.Name] = v
		}
	}
	return out
}
