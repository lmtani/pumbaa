package informativeMessage

import (
	"fmt"

	"github.com/fatih/color"
)

func InformativeMessage(c color.Attribute, message string) {
	_, err := color.New(c).Println(message)
	if err != nil {
		fmt.Println(message)
	}
}
