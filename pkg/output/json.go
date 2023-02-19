package output

import (
	"encoding/json"
	"fmt"
)

type JsonWriter struct {
}

func (w JsonWriter) Primary(s string) {
	return
}

func (w JsonWriter) Accent(s string) {
	return
}

func (w JsonWriter) Error(s string) {
	return
}

func (w JsonWriter) Table(tab Table) {
	b, err := json.Marshal(tab)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(string(b))
}

func NewJsonWriter() *JsonWriter {
	return &JsonWriter{}
}
