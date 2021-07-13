package output

import (
	"fmt"

	"github.com/fatih/color"
)

type ColoredWriter struct{}

type IWriter interface {
	Primary(string)
	Accent(string)
	Error(string)
}

func NewColoredWriter() ColoredWriter {
	return ColoredWriter{}
}

func (w ColoredWriter) Primary(s string) {
	fmt.Println(s)
}

func (w ColoredWriter) Accent(s string) {
	color.Magenta(s)
}

func (w ColoredWriter) Error(s string) {
	color.Red(s)
}
