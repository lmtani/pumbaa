package adapters

import (
	"fmt"
	"io"
	"os"

	"github.com/lmtani/pumbaa/internal/types"

	"github.com/mattn/go-isatty"
	"github.com/olekukonko/tablewriter"
)

var (
	NoticeColor = "\033[1;36m%s\033[0m"
	ErrorColor  = "\033[1;31m%s\033[0m"
	NoColor     = os.Getenv("TERM") == "dumb" ||
		(!isatty.IsTerminal(os.Stdout.Fd()) && !isatty.IsCygwinTerminal(os.Stdout.Fd()))
)

type Table interface {
	Header() []string
	Rows() [][]string
}

type ColoredWriter struct {
	table *tablewriter.Table
}

func NewColoredWriter(writer io.Writer) *ColoredWriter {
	return &ColoredWriter{
		table: tablewriter.NewWriter(writer),
	}
}

func (ColoredWriter) Primary(s string) {
	fmt.Println(s)
}

func (w ColoredWriter) Accent(s string) {
	w.colorPrint(NoticeColor, s)
}

func (w ColoredWriter) Error(s string) {
	w.colorPrint(ErrorColor, s)
}

func (w ColoredWriter) colorPrint(c, s string) {
	if NoColor {
		fmt.Println(s)
	} else {
		fmt.Printf(c, s+"\n")
	}
}

func (w ColoredWriter) Table(table types.Table) {
	w.table.SetHeader(table.Header())
	w.table.SetBorders(tablewriter.Border{Left: true, Top: true, Right: true, Bottom: true})
	w.table.SetAlignment(tablewriter.ALIGN_LEFT)
	w.table.AppendBulk(table.Rows())
	w.table.Render()
}
