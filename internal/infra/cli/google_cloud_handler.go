package cli

import (
	"github.com/lmtani/pumbaa/internal/entities"
	"github.com/lmtani/pumbaa/internal/interfaces"
	"github.com/lmtani/pumbaa/internal/usecase"
	urfaveCli "github.com/urfave/cli/v2"
)

// GoogleCloudHandler is a handler for Google Cloud
type GoogleCloudHandler struct {
	CromwellServer interfaces.CromwellServer
	Writer         interfaces.Writer
}

// NewGoogleCloudHandler creates a new GoogleCloudHandler
func NewGoogleCloudHandler(c interfaces.CromwellServer, w interfaces.Writer) *GoogleCloudHandler {
	return &GoogleCloudHandler{
		CromwellServer: c, Writer: w,
	}
}

// GetComputeUsageForPricing gets the compute usage for pricing
func (g *GoogleCloudHandler) GetComputeUsageForPricing(c *urfaveCli.Context) error {
	gceUsage := usecase.NewWorkflowGCEUsage(g.CromwellServer)
	input := usecase.WorkflowGCEUsageInputDTO{
		WorkflowID: c.String("operation"),
	}
	output, err := gceUsage.Execute(&input)
	if err != nil {
		return err
	}

	wdto := entities.TotalResources{
		PreemptHdd:    output.PreemptHdd,
		PreemptSsd:    output.PreemptSsd,
		PreemptCPU:    output.PreemptCPU,
		PreemptMemory: output.PreemptMemory,
		Hdd:           output.Hdd,
		Ssd:           output.Ssd,
		CPU:           output.CPU,
		Memory:        output.Memory,
		CachedCalls:   output.CachedCalls,
		TotalTime:     output.TotalTime,
	}
	g.Writer.ResourceTable(wdto)
	return nil
}
