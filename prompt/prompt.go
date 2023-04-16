package prompt

import (
	"github.com/manifoldco/promptui"
)

type TemplateOptions struct {
	Label    string
	Active   string
	Inactive string
	Selected string
}

func SelectByKey(taskOptions []string) (string, error) {
	prompt := promptui.Select{
		Label: "Select a task",
		Items: taskOptions,
	}
	_, taskName, err := prompt.Run()
	return taskName, err
}

func SelectByIndex(sfn func(input string, index int) bool, items interface{}) (int, error) {
	templates := &promptui.SelectTemplates{
		Label:    "{{ . }}?",
		Active:   "✔ {{ .ShardIndex  | green }} ({{ .ExecutionStatus | green }}) Attempt: {{ .Attempt | green }} CallCaching: {{ .CallCaching.Hit | green}}",
		Inactive: "  {{ .ShardIndex | faint }} ({{ .ExecutionStatus | red }}) Attempt: {{ .Attempt | faint }} CallCaching: {{ .CallCaching.Hit | faint}}",
		Selected: "✔ {{ .ShardIndex | green }} ({{ .ExecutionStatus | green }}) Attempt: {{ .Attempt | green }} CallCaching: {{ .CallCaching.Hit | green}}",
	}

	prompt := promptui.Select{
		Label:     "Which shard?",
		Items:     items,
		Templates: templates,
		Size:      6,
		Searcher:  sfn,
	}

	i, _, err := prompt.Run()
	return i, err
}
