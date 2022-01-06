package prompt

import (
	"github.com/manifoldco/promptui"
)

type Prompt struct{}

func New() *Prompt {
	return &Prompt{}
}

type Searcher func(input string, index int) bool

func (p Prompt) SelectByKey(taskOptions []string) (string, error) {
	prompt := promptui.Select{
		Label: "Select a task",
		Items: taskOptions,
	}
	_, taskName, err := prompt.Run()
	return taskName, err
}

func (p Prompt) SelectByIndex(t TemplateOptions, sfn func(input string, index int) bool, items interface{}) (int, error) {
	templates := &promptui.SelectTemplates{
		Label:    t.Label,
		Active:   t.Active,
		Inactive: t.Inactive,
		Selected: t.Selected,
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

type TemplateOptions struct {
	Label    string
	Active   string
	Inactive string
	Selected string
}
