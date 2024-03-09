package types

type Table interface {
	Header() []string
	Rows() [][]string
}

type ResourceTableResponse struct {
	Total TotalResources
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
