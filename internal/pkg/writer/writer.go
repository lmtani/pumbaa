package writer

import (
	"encoding/json"
	"fmt"
	"github.com/lmtani/pumbaa/internal/interfaces"
	"io"
	"os"
	"sort"

	"github.com/lmtani/pumbaa/internal/entities"

	"github.com/mattn/go-isatty"
	"github.com/olekukonko/tablewriter"
)

var (
	NoticeColor = "\033[1;36m%s\033[0m"
	ErrorColor  = "\033[1;31m%s\033[0m"
	NoColor     = os.Getenv("TERM") == "dumb" ||
		(!isatty.IsTerminal(os.Stdout.Fd()) && !isatty.IsCygwinTerminal(os.Stdout.Fd()))
)

type ColoredWriter struct {
	table *tablewriter.Table
	w     io.Writer
}

func NewColoredWriter(writer io.Writer) *ColoredWriter {
	return &ColoredWriter{
		table: tablewriter.NewWriter(writer),
		w:     writer,
	}
}

func (ColoredWriter) Primary(s string) {
	fmt.Println(s)
}

func (w ColoredWriter) Accent(s string) {
	w.colorPrint(NoticeColor, s)
}

func (w ColoredWriter) Message(s string) {
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

func (w ColoredWriter) Table(table interfaces.Table) {
	w.table.SetHeader(table.Header())
	w.table.SetBorders(tablewriter.Border{Left: true, Top: true, Right: true, Bottom: true})
	w.table.SetAlignment(tablewriter.ALIGN_LEFT)
	w.table.AppendBulk(table.Rows())
	w.table.Render()
}

func (w ColoredWriter) Json(i interface{}) error {
	b, err := json.MarshalIndent(i, "", "   ")
	if err != nil {
		return err
	}
	data := string(b)
	_, err = w.w.Write([]byte(data))
	return err
}

func (w ColoredWriter) QueryTable(d entities.QueryResponse) {
	var qtr = QueryTableResponse(d)

	w.Table(qtr)
	w.Accent(fmt.Sprintf("- Found %d workflows", d.TotalResultsCount))
}

func (w ColoredWriter) ResourceTable(d entities.TotalResources) {
	var rtr = ResourceTableResponse{Total: d}
	w.Table(rtr)
	w.Accent(fmt.Sprintf("- Tasks with cache hit: %d", d.CachedCalls))
	w.Accent(fmt.Sprintf("- Total time with running VMs: %.0fh", d.TotalTime.Hours()))
}

func (w ColoredWriter) MetadataTable(d entities.MetadataResponse) error {

	var mtr = MetadataTableResponse{Metadata: d}
	w.Table(mtr)
	if len(d.Failures) > 0 {
		w.Error(hasFailureMsg(d.Failures))
		w.recursiveFailureParse(d.Failures)
	}

	items, err := showCustomOptions(d.SubmittedFiles)
	if err != nil {
		return err
	}

	if len(items) > 0 {
		w.Accent("🔧 Custom options")
	}
	// iterate over items strings
	for _, v := range items {
		w.Primary(v)
	}
	return nil
}

func showCustomOptions(s entities.SubmittedFiles) ([]string, error) {
	items := make([]string, 0)

	var options map[string]interface{}
	err := json.Unmarshal([]byte(s.Options), &options)
	if err != nil {
		return items, err
	}

	keys := sortOptionsKeys(options)

	if len(keys) > 0 {
		items = writeOptions(keys, options)
	}

	return items, nil
}

func writeOptions(keys []string, o map[string]interface{}) []string {
	items := make([]string, 0)
	for _, v := range keys {
		if o[v] != "" {
			items = append(items, fmt.Sprintf("- %s: %v", v, o[v]))
		}
	}
	return items
}

func sortOptionsKeys(f map[string]interface{}) []string {
	keys := make([]string, 0)
	for k := range f {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func (w ColoredWriter) recursiveFailureParse(f []entities.Failure) {
	for idx := range f {
		w.Primary(" - " + f[idx].Message)
		w.recursiveFailureParse(f[idx].CausedBy)
	}
}

func hasFailureMsg(fails []entities.Failure) string {
	m := "issue"
	if len(fails) > 1 {
		m = "issues"
	}
	msg := fmt.Sprintf("❗You have %d %s:\n", len(fails), m)
	return msg
}
