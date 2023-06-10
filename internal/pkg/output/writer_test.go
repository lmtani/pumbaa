package output

import (
	"bytes"
	"os"
	"testing"
)

func Example_writer_msgs() {
	w := NewColoredWriter(os.Stdout)

	w.Accent("Accent")
	w.Primary("Primary")
	w.Error("Error")

	// Output:
	// Accent
	// Primary
	// Error
}

type testRow struct {
	col1 string
	col2 string
}

type testTable struct {
	rows []testRow
}

func (tb testTable) Header() []string {
	return []string{"col1", "col2"}
}

func (tb testTable) Rows() [][]string {
	rows := make([][]string, len(tb.rows))
	for _, r := range tb.rows {
		rows = append(rows, []string{
			r.col1,
			r.col2,
		})
	}
	return rows
}

func TestOutputTable(t *testing.T) {
	a := testRow{col1: "value11", col2: "value12"}
	b := testRow{col1: "value21", col2: "value22"}
	tb := testTable{rows: []testRow{a, b}}

	var buffer bytes.Buffer

	w := NewColoredWriter(&buffer)
	w.Table(tb)

	got := buffer.String()
	want := "+---------+---------+\n|  COL1   |  COL2   |\n+---------+---------+\n| value11 | value12 |\n| value21 | value22 |\n+---------+---------+\n"
	if got != want {
		t.Errorf("got = %q, want %q", got, want)
	}
}
