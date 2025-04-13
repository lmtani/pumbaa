package cli

import (
	"github.com/lmtani/pumbaa/internal/interfaces"
	"github.com/lmtani/pumbaa/internal/usecase"
	urfaveCli "github.com/urfave/cli/v2"
)

type InteractiveHandler struct {
	c interfaces.CromwellServer
	w interfaces.Writer
	p interfaces.Prompt
}

func NewInteractiveHandler(c interfaces.CromwellServer, w interfaces.Writer, p interfaces.Prompt) *InteractiveHandler {
	return &InteractiveHandler{c: c, w: w, p: p}
}

func (i *InteractiveHandler) Navigate(c *urfaveCli.Context) error {
	navigateUseCase := usecase.NewWorkflowNavigate(i.c, i.w, i.p)
	input := &usecase.WorkflowNavigateInputDTO{
		WorkflowID: c.String("operation"),
	}
	err := navigateUseCase.Execute(input)
	if err != nil {
		return err
	}
	return nil
}
