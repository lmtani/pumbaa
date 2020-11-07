package commands

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/guptarohit/asciigraph"
	"github.com/urfave/cli/v2"
)

func Monitoring(c *cli.Context) error {
	resource := c.String("resource")
	var r int
	switch resource {
	case "cpu":
		r = 1
	case "mem":
		r = 2
	case "disk":
		r = 3
	default:
		return errors.New("You need to choose between cpu, mem or disk")
	}
	_, err := os.Stdin.Stat()
	if err != nil {
		panic(err)
	}

	reader := bufio.NewReader(os.Stdin)
	var output []rune

	for {
		input, _, err := reader.ReadRune()
		if err != nil && err == io.EOF {
			break
		}
		output = append(output, input)
	}

	var linesBuf string
	for j := 0; j < len(output); j++ {
		linesBuf += fmt.Sprintf("%c", output[j])
	}
	lines := strings.Split(linesBuf, "\n")
	var data []float64
	for _, l := range lines {
		values := strings.Split(l, "\t")
		if len(values) != 4 {
			continue
		}

		if s, err := strconv.ParseFloat(values[r], 32); err == nil {
			data = append(data, s)
		}
	}
	color.Cyan(fmt.Sprintf("%s usage (%%)", strings.ToUpper(resource)))
	option := asciigraph.Height(10)
	graph := asciigraph.Plot(data, option)
	fmt.Println(graph)

	return nil
}
