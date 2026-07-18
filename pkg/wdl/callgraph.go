package wdl

import (
	"slices"
	"sort"

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
	// Bindings records, per input, the leaves its value can be built from,
	// already translated into the top-level workflow's namespace.
	Bindings map[string]ResolvedBinding
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
func (b *graphBuilder) addWorkflow(doc *ast.Document, prefix string, outer map[string]ResolvedBinding, depth int) {
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

	calls := collectCalls(wf)
	callNames := make(map[string]bool, len(calls))
	for _, c := range calls {
		callNames[callName(c.call)] = true
	}

	res := &resolver{
		declarations: collectDeclarations(wf),
		inputs:       declaredInputs(wf.Inputs),
		callNames:    callNames,
		prefix:       prefix,
	}
	defaults := staticDefaults(wf.Inputs)

	for _, c := range calls {
		b.addCall(c, ns, localTasks, res, defaults, prefix, outer, depth)
	}
}

// callName is how a call is addressed within its workflow: its alias when it
// has one, otherwise the task name without any import namespace.
func callName(c *ast.Call) string {
	if c.Alias != "" {
		return c.Alias
	}
	_, target := splitTarget(c.Target)
	return target
}

func (b *graphBuilder) addCall(
	c scopedCall,
	ns map[string]string,
	localTasks map[string]bool,
	res *resolver,
	defaults map[string]string,
	prefix string,
	outer map[string]ResolvedBinding,
	depth int,
) {
	namespace, target := splitTarget(c.call.Target)
	path := prefix + callName(c.call)

	// Each input is reduced to its leaves in this workflow's own terms, then
	// followed outward so every node speaks in top-level terms.
	translated := make(map[string]ResolvedBinding, len(c.call.Inputs))
	for inputName, expr := range c.call.Inputs {
		translated[inputName] = translate(res.resolve(expr, 0), prefix, outer, defaults)
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

// rewireSubworkflowOutputs replaces dependencies on a flattened subworkflow
// call with dependencies on the leaf that actually produces the value, so a
// consumer does not inherit a rerun from an unrelated part of the subworkflow.
func (b *graphBuilder) rewireSubworkflowOutputs() {
	for _, node := range b.graph.Nodes {
		for inputName, binding := range node.Bindings {
			rewired := binding
			for i, source := range rewired.Sources {
				if source.Kind != SourceCall {
					continue
				}
				outputs, ok := b.subOutputs[source.Name]
				if !ok {
					continue
				}
				producer, ok := outputs[source.Member]
				if !ok {
					// An output we could not trace to a leaf: the value's
					// origin is unknown, so the binding stops being complete.
					rewired.Complete = false
					rewired.Incomplete = "reads an output of " + source.Name +
						" that could not be traced to a producing call"
					continue
				}
				rewired.Sources[i].Name = producer
			}
			node.Bindings[inputName] = dedupeSources(rewired)
		}
	}
}

// deriveDependencies recomputes every edge from the final bindings, so the
// graph cannot disagree with the bindings it was built from.
func (b *graphBuilder) deriveDependencies() {
	for _, node := range b.graph.Nodes {
		deps := make(map[string]bool)
		for _, binding := range node.Bindings {
			for _, producer := range binding.Calls() {
				if producer != node.Name && b.graph.Nodes[producer] != nil {
					deps[producer] = true
				}
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

// collectDeclarations gathers a workflow's intermediate declarations by name,
// including those nested in scatter and conditional blocks, so an expression
// that reads one can be followed to its own leaves.
func collectDeclarations(wf *ast.Workflow) map[string]ast.Expression {
	out := make(map[string]ast.Expression)
	add := func(d *ast.Declaration) {
		if d != nil && d.Expression != nil {
			out[d.Name] = d.Expression
		}
	}
	for _, d := range wf.Declarations {
		add(d)
	}
	var walk func(body []ast.WorkflowElement)
	walk = func(body []ast.WorkflowElement) {
		for _, el := range body {
			switch e := el.(type) {
			case *ast.Declaration:
				add(e)
			case *ast.Scatter:
				walk(e.Body)
			case *ast.Conditional:
				walk(e.Body)
			}
		}
	}
	for _, s := range wf.Scatters {
		walk(s.Body)
	}
	for _, c := range wf.Conditionals {
		walk(c.Body)
	}
	return out
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
