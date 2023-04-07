package prompt

import (
	"fmt"
	"github.com/lmtani/cromwell-cli/pkg/cromwell"
	"github.com/lmtani/cromwell-cli/pkg/output"
	"os"
	"strconv"
	"strings"
)

type TermUi struct {
	CromwellClient cromwell.Client
	Writer         output.Writer
	Prompt         Prompt
}

var (
	defaultClient = cromwell.Default()
	defaultWriter = output.NewColoredWriter(os.Stdout)
	defaultPrompt = &Ui{}
)

func NewTermUi() *TermUi {
	return &TermUi{
		CromwellClient: defaultClient,
		Writer:         defaultWriter,
		Prompt:         defaultPrompt,
	}
}

func (p TermUi) Navigate(operation string) error {
	params := cromwell.ParamsMetadataGet{
		ExcludeKey: []string{"executionEvents", "submittedFiles", "jes", "inputs"},
	}
	resp, err := p.CromwellClient.Metadata(operation, &params)
	if err != nil {
		return err
	}
	var item cromwell.CallItem
	for {
		task, err := p.selectDesiredTask(&resp)
		if err != nil {
			return err
		}
		item, err = p.selectDesiredShard(task)
		if err != nil {
			return err
		}
		if item.SubWorkflowID == "" {
			break
		}
		resp, err = p.CromwellClient.Metadata(item.SubWorkflowID, &params)
		if err != nil {
			return err
		}
	}

	fmt.Printf("Command status: %s\n", item.ExecutionStatus)
	if item.ExecutionStatus == "QueuedInCromwell" {
		return nil
	}
	if item.CallCaching.Hit {
		p.Writer.Accent(item.CallCaching.Result)
	} else {
		p.Writer.Accent(item.CommandLine)
	}

	fmt.Printf("Logs:\n")
	p.Writer.Accent(fmt.Sprintf("%s\n%s\n", item.Stderr, item.Stdout))
	if item.MonitoringLog != "" {
		p.Writer.Accent(fmt.Sprintf("%s\n", item.MonitoringLog))
	}
	if item.BackendLogs.Log != "" {
		p.Writer.Accent(fmt.Sprintf("%s\n", item.BackendLogs.Log))
	}

	fmt.Printf("🐋 Docker image:\n")
	p.Writer.Accent(fmt.Sprintf("%s\n", item.RuntimeAttributes.Docker))
	return nil
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func (p TermUi) selectDesiredTask(m *cromwell.MetadataResponse) ([]cromwell.CallItem, error) {
	var taskOptions []string
	calls := make(map[string][]cromwell.CallItem)
	for key, value := range m.Calls {
		sliceName := strings.Split(key, ".")
		taskName := sliceName[len(sliceName)-1]
		calls[taskName] = value
		if !contains(taskOptions, taskName) {
			taskOptions = append(taskOptions, taskName)
		}
	}
	cat := "Workflow"
	if m.RootWorkflowID != "" {
		cat = "SubWorkflow"
	}
	p.Writer.Accent(fmt.Sprintf("%s: %s\n", cat, m.WorkflowName))

	taskName, err := p.Prompt.SelectByKey(taskOptions)
	if err != nil {
		fmt.Printf("Ui failed %v\n", err)
		return []cromwell.CallItem{}, err
	}
	return calls[taskName], nil
}

func (p TermUi) selectDesiredShard(shards []cromwell.CallItem) (cromwell.CallItem, error) {
	if len(shards) == 1 {
		return shards[0], nil
	}

	template := TemplateOptions{
		Label:    "{{ . }}?",
		Active:   "✔ {{ .ShardIndex  | green }} ({{ .ExecutionStatus | green }}) Attempt: {{ .Attempt | green }} CallCaching: {{ .CallCaching.Hit | green}}",
		Inactive: "  {{ .ShardIndex | faint }} ({{ .ExecutionStatus | red }}) Attempt: {{ .Attempt | faint }} CallCaching: {{ .CallCaching.Hit | faint}}",
		Selected: "✔ {{ .ShardIndex | green }} ({{ .ExecutionStatus | green }}) Attempt: {{ .Attempt | green }} CallCaching: {{ .CallCaching.Hit | green}}",
	}

	searcher := func(input string, index int) bool {
		shard := shards[index]
		name := strconv.Itoa(shard.ShardIndex)
		return name == input
	}

	i, err := p.Prompt.SelectByIndex(template, searcher, shards)
	if err != nil {
		return cromwell.CallItem{}, err
	}

	return shards[i], err
}
