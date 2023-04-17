package output

import (
	"fmt"
	"io"
	"os"

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
	w *tablewriter.Table
}

func NewColoredWriter(writer io.Writer) *ColoredWriter {
	return &ColoredWriter{
		w: tablewriter.NewWriter(writer),
	}
}

func (w ColoredWriter) Primary(s string) {
	fmt.Println(s)
}

func (w ColoredWriter) Accent(s string) {
	w.colorPrint(NoticeColor, s)
}

func (w ColoredWriter) Error(s string) {
	w.colorPrint(ErrorColor, s)
}

func (w ColoredWriter) colorPrint(c string, s string) {
	if NoColor {
		fmt.Println(s)
	} else {
		fmt.Printf(c, s+"\n")
	}
}

func (w ColoredWriter) Table(tab Table) {
	w.w.SetHeader(tab.Header())
	w.w.SetBorders(tablewriter.Border{Left: true, Top: true, Right: true, Bottom: true})
	w.w.SetAlignment(tablewriter.ALIGN_LEFT)
	w.w.AppendBulk(tab.Rows())
	w.w.Render()
}
