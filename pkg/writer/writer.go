package writer

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

func New() ColoredWriter {
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

type Uncolored struct{}

func NewUncolored() Uncolored {
	return Uncolored{}
}

func (w Uncolored) Primary(s string) {
	fmt.Println(s)
}

func (w Uncolored) Accent(s string) {
	fmt.Println(s)
}

func (w Uncolored) Error(s string) {
	fmt.Println(s)
}
