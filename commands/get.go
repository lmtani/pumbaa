package commands

import (
	"fmt"

	"go.uber.org/zap"
)

type QueryResponse struct {
	results           []QueryResponseWorkflow
	totalResultsCount int
}

type QueryResponseWorkflow struct {
	id         string
	name       string
	status     string
	submission string
	start      string
	end        string
}

func QueryWorkflow(c Client, n string) error {
	err := c.Query("VsaCloud")
	if err != nil {
		return err
	}
	zap.S().Infow(fmt.Sprintf("Found %d workflows"))
	return err
}

// data := [][]string{
// 	{"1/1/2014", "Domain name", "2233", "$10.98"},
// 	{"1/1/2014", "January Hosting", "2233", "$54.95"},
// 	{"1/4/2014", "February Hosting", "2233", "$51.00"},
// 	{"1/4/2014", "February Extra Bandwidth", "2233", "$30.00"},
// }

// table := tablewriter.NewWriter(os.Stdout)
// table.SetHeader([]string{"Date", "Description", "CV2", "Amount"})
// table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
// table.SetCenterSeparator("|")
// table.AppendBulk(data) // Add Bulk Data
// table.Render()
