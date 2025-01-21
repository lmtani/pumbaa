package writer

import (
	"fmt"

	"github.com/lmtani/pumbaa/internal/entities"
)

type ResourceTableResponse struct {
	Total entities.TotalResources
}

func (ResourceTableResponse) Header() []string {
	return []string{"Resource", "Normalized to", "Preemptive", "Normal"}
}

func (rtr ResourceTableResponse) Rows() [][]string {
	rows := [][]string{
		{
			"CPUs",
			"1 hour",
			dashIfZero(rtr.Total.PreemptCPU),
			dashIfZero(rtr.Total.CPU),
		},
		{
			"Memory (GB)",
			"1 hour",
			dashIfZero(rtr.Total.PreemptMemory),
			dashIfZero(rtr.Total.Memory),
		},
		{
			"HDD disk (GB)",
			"1 month",
			dashIfZero(rtr.Total.PreemptHdd),
			dashIfZero(rtr.Total.Hdd),
		},
		{
			"SSD disk (GB)",
			"1 month",
			dashIfZero(rtr.Total.PreemptSsd),
			dashIfZero(rtr.Total.Ssd),
		},
	}
	return rows
}

func dashIfZero(v float64) string {
	s := fmt.Sprintf("%.2f", v)
	if v == 0.0 {
		s = "-"
	}
	return s
}
