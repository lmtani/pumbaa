package commands

import (
	"os"

	"github.com/olekukonko/tablewriter"
)

func CreateTable(header []string, rows [][]string) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader(header)
	table.SetBorders(tablewriter.Border{Left: true, Top: true, Right: true, Bottom: true})
	table.AppendBulk(rows)
	table.Render()
}
