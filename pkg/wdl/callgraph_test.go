package wdl

import (
	"reflect"
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
	if b := index["input_vcf"]; b.Kind != BindingWorkflowInput || b.Source != "input_vcf" {
		t.Errorf("IndexVcf.input_vcf binding = %+v, want workflow input input_vcf", b)
	}

	stats := g.Nodes["StatsVcf"].Bindings
	if b := stats["input_vcf"]; b.Kind != BindingCall || b.Source != "IndexVcf" {
		t.Errorf("StatsVcf.input_vcf binding = %+v, want call IndexVcf", b)
	}
	if b := stats["output_basename"]; b.Kind != BindingWorkflowInput {
		t.Errorf("StatsVcf.output_basename binding = %+v, want workflow input", b)
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
	if b := g.Nodes["T"].Bindings["image"]; b.Kind != BindingUnknown {
		t.Errorf("concatenated expression binding = %+v, want unknown", b)
	}
}
