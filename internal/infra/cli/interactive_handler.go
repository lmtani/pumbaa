package cli

import (
	"github.com/lmtani/pumbaa/internal/ports"
	"github.com/lmtani/pumbaa/internal/usecase"
	urfaveCli "github.com/urfave/cli/v2"
)

type InteractiveHandler struct {
	c ports.CromwellServer
	w ports.Writer
	p ports.Prompt
}

func NewInteractiveHandler(c ports.CromwellServer, w ports.Writer, p ports.Prompt) *InteractiveHandler {
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
