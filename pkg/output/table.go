package output

import (
	"io"

	"github.com/olekukonko/tablewriter"
)

type Table struct {
	w *tablewriter.Table
}

type Tablee interface {
	Header() []string
	Rows() [][]string
}

func NewTable(w io.Writer) Table {
	return Table{
		tablewriter.NewWriter(w),
	}
}

func (t Table) Render(tab Tablee) {
	t.w.SetHeader(tab.Header())
	t.w.SetBorders(tablewriter.Border{Left: true, Top: true, Right: true, Bottom: true})
	t.w.SetAlignment(tablewriter.ALIGN_LEFT)
	t.w.AppendBulk(tab.Rows())
	t.w.Render()
}
