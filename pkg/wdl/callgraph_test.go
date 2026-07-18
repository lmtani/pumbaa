package wdl

import (
	"reflect"
	gosort "sort"
	"strings"
	"testing"
)

// The showcase workflow used to validate the cache design end to end: StatsVcf
// consumes IndexVcf's outputs, which is what makes the cache cascade.
const chainedWDL = `version 1.0

workflow VcfIndexAndStats {
  input {
    File input_vcf
    String output_basename
  }

  call IndexVcf {
    input:
      input_vcf = input_vcf,
      output_basename = output_basename
  }

  call StatsVcf {
    input:
      input_vcf = IndexVcf.out_vcf,
      input_vcf_index = IndexVcf.out_vcf_index,
      output_basename = output_basename
  }

  output {
    File stats_report = StatsVcf.report
  }
}

task IndexVcf {
  input {
    File input_vcf
    String output_basename
  }
  command <<< bcftools index ~{input_vcf} >>>
  runtime { docker: "bcftools:1.11" }
  output {
    File out_vcf = "~{output_basename}.vcf.gz"
    File out_vcf_index = "~{output_basename}.vcf.gz.tbi"
  }
}

task StatsVcf {
  input {
    File input_vcf
    File input_vcf_index
    String output_basename
  }
  command <<< bcftools stats ~{input_vcf} >>>
  runtime { docker: "bcftools:1.11" }
  output {
    File report = "~{output_basename}.stats.txt"
  }
}
`

// only returns the single source of a binding, failing when the binding is not
// a simple one-leaf expression.
func only(t *testing.T, b ResolvedBinding, what string) ValueSource {
	t.Helper()
	if !b.Complete {
		t.Fatalf("%s: binding is incomplete (%s)", what, b.Incomplete)
	}
	if len(b.Sources) != 1 {
		t.Fatalf("%s: expected one source, got %+v", what, b.Sources)
	}
	return b.Sources[0]
}

func TestBuildCallGraphResolvesChainedDependency(t *testing.T) {
	g, err := BuildCallGraph([]byte(chainedWDL))
	if err != nil {
		t.Fatalf("BuildCallGraph() error: %v", err)
	}

	if g.Workflow != "VcfIndexAndStats" {
		t.Errorf("workflow name = %q, want VcfIndexAndStats", g.Workflow)
	}
	if got := g.Names(); !reflect.DeepEqual(got, []string{"IndexVcf", "StatsVcf"}) {
		t.Fatalf("Names() = %v, want [IndexVcf StatsVcf]", got)
	}

	index := g.Nodes["IndexVcf"]
	if len(index.DependsOn) != 0 {
		t.Errorf("IndexVcf.DependsOn = %v, want none (its inputs are workflow-level)", index.DependsOn)
	}

	stats := g.Nodes["StatsVcf"]
	// Two inputs come from IndexVcf; the dependency must be recorded once.
	if !reflect.DeepEqual(stats.DependsOn, []string{"IndexVcf"}) {
		t.Errorf("StatsVcf.DependsOn = %v, want [IndexVcf]", stats.DependsOn)
	}
	if stats.Subworkflow || stats.Scattered {
		t.Errorf("StatsVcf: got subworkflow=%v scattered=%v, want both false", stats.Subworkflow, stats.Scattered)
	}
}

func TestBuildCallGraphDependenciesMapFeedsDomain(t *testing.T) {
	g, err := BuildCallGraph([]byte(chainedWDL))
	if err != nil {
		t.Fatalf("BuildCallGraph() error: %v", err)
	}
	deps := g.Dependencies()
	if len(deps) != 2 {
		t.Fatalf("Dependencies() has %d entries, want 2", len(deps))
	}
	if !reflect.DeepEqual(deps["StatsVcf"], []string{"IndexVcf"}) {
		t.Errorf("deps[StatsVcf] = %v, want [IndexVcf]", deps["StatsVcf"])
	}
}

const scatteredWDL = `version 1.0

workflow Fanout {
  input {
    Array[File] bams
  }

  scatter (bam in bams) {
    call Align {
      input: bam = bam
    }
  }

  call Merge {
    input: aligned = Align.out
  }
}

task Align {
  input { File bam }
  command <<< echo ~{bam} >>>
  output { File out = "a.bam" }
}

task Merge {
  input { Array[File] aligned }
  command <<< echo ~{sep=" " aligned} >>>
  output { File out = "m.bam" }
}
`

func TestBuildCallGraphIncludesScatteredCalls(t *testing.T) {
	g, err := BuildCallGraph([]byte(scatteredWDL))
	if err != nil {
		t.Fatalf("BuildCallGraph() error: %v", err)
	}

	align, ok := g.Nodes["Align"]
	if !ok {
		t.Fatal("Align missing: calls inside a scatter must appear in the graph")
	}
	if !align.Scattered {
		t.Error("Align.Scattered = false, want true (it is inside a scatter block)")
	}

	merge, ok := g.Nodes["Merge"]
	if !ok {
		t.Fatal("Merge missing from graph")
	}
	if !reflect.DeepEqual(merge.DependsOn, []string{"Align"}) {
		t.Errorf("Merge.DependsOn = %v, want [Align]", merge.DependsOn)
	}
	if merge.Scattered {
		t.Error("Merge.Scattered = true, want false (it is outside the scatter)")
	}
}

const aliasedWDL = `version 1.0

workflow Aliased {
  input { File x }

  call Process as First {
    input: f = x
  }

  call Process as Second {
    input: f = First.out
  }
}

task Process {
  input { File f }
  command <<< echo ~{f} >>>
  output { File out = "o.txt" }
}
`

func TestBuildCallGraphUsesAliasAsCallName(t *testing.T) {
	g, err := BuildCallGraph([]byte(aliasedWDL))
	if err != nil {
		t.Fatalf("BuildCallGraph() error: %v", err)
	}

	if got := g.Names(); !reflect.DeepEqual(got, []string{"First", "Second"}) {
		t.Fatalf("Names() = %v, want the aliases [First Second]", got)
	}
	if g.Nodes["First"].Task != "Process" {
		t.Errorf("First.Task = %q, want Process", g.Nodes["First"].Task)
	}
	if !reflect.DeepEqual(g.Nodes["Second"].DependsOn, []string{"First"}) {
		t.Errorf("Second.DependsOn = %v, want [First]", g.Nodes["Second"].DependsOn)
	}
}

func TestBuildCallGraphEmptyWorkflowIsNotAnError(t *testing.T) {
	g, err := BuildCallGraph([]byte("version 1.0\n\nworkflow Empty {\n}\n"))
	if err != nil {
		t.Fatalf("BuildCallGraph() error: %v", err)
	}
	if len(g.Nodes) != 0 {
		t.Errorf("expected no calls, got %v", g.Names())
	}
}

func TestBuildCallGraphClassifiesInputBindings(t *testing.T) {
	g, err := BuildCallGraph([]byte(chainedWDL))
	if err != nil {
		t.Fatalf("BuildCallGraph() error: %v", err)
	}

	index := g.Nodes["IndexVcf"].Bindings
	if s := only(t, index["input_vcf"], "IndexVcf.input_vcf"); s.Kind != SourceInput || s.Name != "input_vcf" {
		t.Errorf("IndexVcf.input_vcf = %+v, want workflow input input_vcf", s)
	}

	stats := g.Nodes["StatsVcf"].Bindings
	if s := only(t, stats["input_vcf"], "StatsVcf.input_vcf"); s.Kind != SourceCall || s.Name != "IndexVcf" {
		t.Errorf("StatsVcf.input_vcf = %+v, want call IndexVcf", s)
	}
	if s := only(t, stats["output_basename"], "StatsVcf.output_basename"); s.Kind != SourceInput {
		t.Errorf("StatsVcf.output_basename = %+v, want workflow input", s)
	}
}

func TestTaskSpecsExtractsCommandAndDocker(t *testing.T) {
	specs, err := TaskSpecs([]byte(chainedWDL))
	if err != nil {
		t.Fatalf("TaskSpecs() error: %v", err)
	}

	index, ok := specs["IndexVcf"]
	if !ok {
		t.Fatal("IndexVcf missing from task specs")
	}
	if !strings.Contains(index.Command, "bcftools index") {
		t.Errorf("IndexVcf.Command = %q, want the raw command template", index.Command)
	}
	docker, ok := index.DockerValue()
	if !ok || docker != "bcftools:1.11" {
		t.Errorf("IndexVcf docker = (%q, %v), want bcftools:1.11", docker, ok)
	}
}

// The showcase pattern: docker is a task input with a default, referenced from
// the runtime section. Resolving it needs the input default, not the runtime.
func TestTaskSpecsResolvesDockerViaInputDefault(t *testing.T) {
	const src = `version 1.0

workflow W {
  call T
}

task T {
  input {
    String docker = "bcftools:1.12"
  }
  command <<< echo hi >>>
  runtime { docker: docker }
  output { File out = "o" }
}
`
	specs, err := TaskSpecs([]byte(src))
	if err != nil {
		t.Fatalf("TaskSpecs() error: %v", err)
	}
	got, ok := specs["T"].DockerValue()
	if !ok || got != "bcftools:1.12" {
		t.Errorf("DockerValue() = (%q, %v), want bcftools:1.12", got, ok)
	}
}

func TestStaticValueRejectsInputDependentExpressions(t *testing.T) {
	const src = `version 1.0

workflow W {
  input { String tag }
  call T { input: image = "repo:" + tag }
}

task T {
  input { String image }
  command <<< echo ~{image} >>>
  output { File out = "o" }
}
`
	g, err := BuildCallGraph([]byte(src))
	if err != nil {
		t.Fatalf("BuildCallGraph() error: %v", err)
	}
	// A concatenation of an input and a literal is fully resolvable: both
	// leaves are classified and the operator is deterministic.
	b := g.Nodes["T"].Bindings["image"]
	if !b.Complete {
		t.Fatalf("concatenation should resolve: %s", b.Incomplete)
	}
	if len(b.Sources) != 2 {
		t.Errorf("concatenation sources = %+v, want the literal and the input", b.Sources)
	}
}

// A subworkflow: the parent passes its own inputs down, and the leaf tasks are
// what Cromwell actually caches.
const subWorkflowSource = `version 1.0

workflow AlignSample {
  input {
    File reads
    String sample
  }

  call Align {
    input: reads = reads, sample = sample
  }

  call Sort {
    input: bam = Align.out, sample = sample
  }

  output {
    File sorted = Sort.out
    File raw = Align.out
  }
}

task Align {
  input { File reads String sample }
  command <<< bwa mem ~{reads} > ~{sample}.bam >>>
  runtime { docker: "bwa:0.7.17" }
  output { File out = "~{sample}.bam" }
}

task Sort {
  input { File bam String sample }
  command <<< samtools sort ~{bam} >>>
  runtime { docker: "samtools:1.21" }
  output { File out = "~{sample}.sorted.bam" }
}
`

const parentWithSubSource = `version 1.0

import "align_sample.wdl" as align

workflow Cohort {
  input {
    File reads
    String sample
  }

  call align.AlignSample {
    input: reads = reads, sample = sample
  }

  call Report {
    input: bam = AlignSample.sorted, sample = sample
  }
}

task Report {
  input { File bam String sample }
  command <<< echo ~{bam} >>>
  runtime { docker: "report:1.0" }
  output { File out = "~{sample}.txt" }
}
`

func subSources() SourceSet {
	s := SourceSet{}
	s.Add("align_sample.wdl", []byte(subWorkflowSource))
	return s
}

// The core of subworkflow support: leaves are flattened into the graph under
// their call path, because a subworkflow call is not itself a cacheable unit.
func TestBuildCallGraphFlattensSubworkflowIntoLeaves(t *testing.T) {
	g, err := BuildCallGraphWithSources([]byte(parentWithSubSource), subSources())
	if err != nil {
		t.Fatalf("BuildCallGraphWithSources() error: %v", err)
	}

	want := []string{"AlignSample.Align", "AlignSample.Sort", "Report"}
	if got := g.Names(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Names() = %v, want %v", got, want)
	}
	// The subworkflow call itself must not survive as a node.
	if _, ok := g.Nodes["AlignSample"]; ok {
		t.Error("the subworkflow call should be flattened away, not kept as a node")
	}
	for _, n := range g.Nodes {
		if n.Unresolved || n.Subworkflow {
			t.Errorf("%s: got unresolved=%v subworkflow=%v, want both false",
				n.Name, n.Unresolved, n.Subworkflow)
		}
	}
}

// Values passed into a subworkflow must be followed through to the leaf, or the
// leaf's inputs cannot be compared against a reference run.
func TestBuildCallGraphTranslatesBindingsThroughSubworkflow(t *testing.T) {
	g, err := BuildCallGraphWithSources([]byte(parentWithSubSource), subSources())
	if err != nil {
		t.Fatalf("BuildCallGraphWithSources() error: %v", err)
	}

	align := g.Nodes["AlignSample.Align"]
	// `reads` is a subworkflow input, bound by the parent to its own input.
	if s := only(t, align.Bindings["reads"], "Align.reads"); s.Kind != SourceInput || s.Name != "reads" {
		t.Errorf("Align.reads = %+v, want the top-level workflow input reads", s)
	}
	if s := only(t, align.Bindings["sample"], "Align.sample"); s.Kind != SourceInput || s.Name != "sample" {
		t.Errorf("Align.sample = %+v, want the top-level workflow input sample", s)
	}

	// Inside the subworkflow, Sort consumes Align — the edge must be qualified.
	sort := g.Nodes["AlignSample.Sort"]
	if s := only(t, sort.Bindings["bam"], "Sort.bam"); s.Kind != SourceCall || s.Name != "AlignSample.Align" {
		t.Errorf("Sort.bam = %+v, want call AlignSample.Align", s)
	}
	if !reflect.DeepEqual(sort.DependsOn, []string{"AlignSample.Align"}) {
		t.Errorf("Sort.DependsOn = %v, want [AlignSample.Align]", sort.DependsOn)
	}
}

// A consumer of a subworkflow output must depend on the leaf that produces it,
// not on every leaf — otherwise an unrelated rerun inside the subworkflow would
// cascade to it.
func TestBuildCallGraphResolvesSubworkflowOutputToProducingLeaf(t *testing.T) {
	g, err := BuildCallGraphWithSources([]byte(parentWithSubSource), subSources())
	if err != nil {
		t.Fatalf("BuildCallGraphWithSources() error: %v", err)
	}

	report := g.Nodes["Report"]
	// Cohort reads AlignSample.sorted, which the subworkflow declares as Sort.out.
	if s := only(t, report.Bindings["bam"], "Report.bam"); s.Kind != SourceCall || s.Name != "AlignSample.Sort" {
		t.Errorf("Report.bam = %+v, want call AlignSample.Sort", s)
	}
	if !reflect.DeepEqual(report.DependsOn, []string{"AlignSample.Sort"}) {
		t.Errorf("Report.DependsOn = %v, want only [AlignSample.Sort]", report.DependsOn)
	}
}

// Without the imported source the subworkflow stays opaque and is flagged, so
// callers withhold a verdict instead of assuming it is unchanged.
func TestBuildCallGraphMarksUnresolvedSubworkflow(t *testing.T) {
	g, err := BuildCallGraph([]byte(parentWithSubSource))
	if err != nil {
		t.Fatalf("BuildCallGraph() error: %v", err)
	}

	node, ok := g.Nodes["AlignSample"]
	if !ok {
		t.Fatal("an unresolvable subworkflow must remain in the graph as one node")
	}
	if !node.Unresolved || !node.Subworkflow {
		t.Errorf("AlignSample: got unresolved=%v subworkflow=%v, want both true",
			node.Unresolved, node.Subworkflow)
	}
}

// An imported *task* is not a subworkflow. Treating it as one was a real bug:
// it silently withheld a verdict for the most common import pattern.
func TestBuildCallGraphResolvesImportedTask(t *testing.T) {
	const lib = `version 1.0

task AlignReads {
  input { File bam }
  command <<< bwa ~{bam} >>>
  runtime { docker: "bwa:0.7.17" }
  output { File out = "o.bam" }
}
`
	const main = `version 1.0

import "lib.wdl" as lib

workflow W {
  input { File bam }
  call lib.AlignReads { input: bam = bam }
}
`
	sources := SourceSet{}
	sources.Add("lib.wdl", []byte(lib))

	g, err := BuildCallGraphWithSources([]byte(main), sources)
	if err != nil {
		t.Fatalf("BuildCallGraphWithSources() error: %v", err)
	}

	node, ok := g.Nodes["AlignReads"]
	if !ok {
		t.Fatalf("AlignReads missing; graph has %v", g.Names())
	}
	if node.Subworkflow {
		t.Error("an imported task must not be classified as a subworkflow")
	}
	if node.Unresolved {
		t.Error("an imported task whose source is available must be resolved")
	}
}

// Without the library source the task body is invisible, so a command change
// would be undetectable — that has to be flagged, not silently ignored.
func TestBuildCallGraphMarksImportedTaskUnresolvedWithoutSources(t *testing.T) {
	const main = `version 1.0

import "lib.wdl" as lib

workflow W {
  input { File bam }
  call lib.AlignReads { input: bam = bam }
}
`
	g, err := BuildCallGraph([]byte(main))
	if err != nil {
		t.Fatalf("BuildCallGraph() error: %v", err)
	}
	node, ok := g.Nodes["AlignReads"]
	if !ok {
		t.Fatalf("AlignReads missing; graph has %v", g.Names())
	}
	if !node.Unresolved {
		t.Error("AlignReads: want unresolved when the import source is absent")
	}
}

// A subworkflow input the parent does not pass falls back to the subworkflow's
// own default.
func TestBuildCallGraphUsesSubworkflowDefaultWhenParentDoesNotBind(t *testing.T) {
	const sub = `version 1.0

workflow Sub {
  input {
    File data
    String mode = "fast"
  }
  call Run { input: data = data, mode = mode }
}

task Run {
  input { File data String mode }
  command <<< echo ~{mode} ~{data} >>>
  output { File out = "o" }
}
`
	const main = `version 1.0

import "sub.wdl" as s

workflow Top {
  input { File data }
  call s.Sub { input: data = data }
}
`
	sources := SourceSet{}
	sources.Add("sub.wdl", []byte(sub))

	g, err := BuildCallGraphWithSources([]byte(main), sources)
	if err != nil {
		t.Fatalf("BuildCallGraphWithSources() error: %v", err)
	}
	run := g.Nodes["Sub.Run"]
	if run == nil {
		t.Fatalf("Sub.Run missing; graph has %v", g.Names())
	}
	if s := only(t, run.Bindings["mode"], "Run.mode"); s.Kind != SourceLiteral || s.Literal != "fast" {
		t.Errorf("Run.mode = %+v, want the subworkflow default literal \"fast\"", s)
	}
	if s := only(t, run.Bindings["data"], "Run.data"); s.Kind != SourceInput || s.Name != "data" {
		t.Errorf("Run.data = %+v, want the top-level input data", s)
	}
}

// An input neither passed by the parent nor defaulted must be looked up under
// the call path, the way Cromwell expects it in the inputs JSON.
func TestBuildCallGraphScopesUnboundSubworkflowInput(t *testing.T) {
	const sub = `version 1.0

workflow Sub {
  input {
    File data
    String mode
  }
  call Run { input: data = data, mode = mode }
}

task Run {
  input { File data String mode }
  command <<< echo ~{mode} >>>
  output { File out = "o" }
}
`
	const main = `version 1.0

import "sub.wdl" as s

workflow Top {
  input { File data }
  call s.Sub { input: data = data }
}
`
	sources := SourceSet{}
	sources.Add("sub.wdl", []byte(sub))

	g, err := BuildCallGraphWithSources([]byte(main), sources)
	if err != nil {
		t.Fatalf("BuildCallGraphWithSources() error: %v", err)
	}
	s := only(t, g.Nodes["Sub.Run"].Bindings["mode"], "Run.mode")
	if s.Kind != SourceInput || s.Name != "mode" || s.Scope != "Sub" {
		t.Errorf("Run.mode = %+v, want workflow input mode scoped to Sub", s)
	}
}

func TestTaskSpecsWithSourcesIncludesImportedTasks(t *testing.T) {
	specs, err := TaskSpecsWithSources([]byte(parentWithSubSource), subSources())
	if err != nil {
		t.Fatalf("TaskSpecsWithSources() error: %v", err)
	}
	for _, name := range []string{"Report", "Align", "Sort"} {
		if _, ok := specs[name]; !ok {
			t.Errorf("task %q missing; got %v", name, specKeys(specs))
		}
	}
	if docker, ok := specs["Align"].DockerValue(); !ok || docker != "bwa:0.7.17" {
		t.Errorf("Align docker = (%q, %v), want bwa:0.7.17", docker, ok)
	}
}

func specKeys(m map[string]TaskSpec) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	gosort.Strings(out)
	return out
}

// The parser hands back an interpolated string as a single literal with the
// placeholder text intact, so `"L~{idx}"` arrives verbatim. Treating that as a
// fixed value compares a template against whatever a run produced from it, and
// reports a change for every such input — a confident, wrong answer.
func TestStaticValueRejectsInterpolatedStrings(t *testing.T) {
	const src = `version 1.0

workflow W {
  input { Array[File] items String tag }

  scatter (idx in range(length(items))) {
    String label = "L~{idx}"
    call T { input: label = label, fixed = "plain", tagged = tag }
  }
}

task T {
  input { String label String fixed String tagged }
  command <<< echo ~{label} >>>
  output { File out = "o" }
}
`
	g, err := BuildCallGraph([]byte(src))
	if err != nil {
		t.Fatalf("BuildCallGraph() error: %v", err)
	}
	bindings := g.Nodes["T"].Bindings

	label := bindings["label"]
	if label.Complete {
		t.Errorf("label resolved to %+v, but it interpolates a scatter variable "+
			"and has no value fixed in the text", label.Sources)
	}

	// A string with no placeholder is still a literal, or the guard would have
	// cost every genuine constant.
	if s := only(t, bindings["fixed"], "T.fixed"); s.Kind != SourceLiteral || s.Literal != "plain" {
		t.Errorf("fixed = %+v, want the literal \"plain\"", s)
	}
	if s := only(t, bindings["tagged"], "T.tagged"); s.Kind != SourceInput || s.Name != "tag" {
		t.Errorf("tagged = %+v, want the workflow input tag", s)
	}
}
