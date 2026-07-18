package presenter

import (
	"fmt"
	"io"
	"os"

	"golang.org/x/term"

	"github.com/lmtani/pumbaa/internal/application/ports"
)

// Progress writes a single, self-overwriting status line while a command works.
//
// It goes to standard error on purpose: a command whose output is piped or
// parsed must not have progress chatter mixed into it, and stderr is where a
// reader looks for "what is it doing" anyway.
//
// When the destination is not a terminal — a log file, a pipe, CI — the line is
// dropped entirely rather than emitted as a stream of half-lines that nobody
// will overwrite.
type Progress struct {
	out         io.Writer
	interactive bool
	width       int
}

// NewProgress builds a reporter over standard error.
func NewProgress() *Progress {
	return newProgressTo(os.Stderr, int(os.Stderr.Fd()))
}

func newProgressTo(out io.Writer, fd int) *Progress {
	p := &Progress{out: out, interactive: term.IsTerminal(fd)}
	if p.interactive {
		if width, _, err := term.GetSize(fd); err == nil && width > 0 {
			p.width = width
		}
	}
	return p
}

// Step replaces the current status line.
func (p *Progress) Step(format string, args ...any) {
	if !p.interactive {
		return
	}
	line := "  " + fmt.Sprintf(format, args...) + "…"
	if p.width > 0 && len(line) > p.width-1 {
		line = line[:p.width-1]
	}
	// A status line that cannot be written is not worth failing a command over.
	_, _ = fmt.Fprintf(p.out, "\r\033[K%s", line)
}

// Done erases the status line so the result is not printed beneath it.
func (p *Progress) Done() {
	if !p.interactive {
		return
	}
	_, _ = fmt.Fprint(p.out, "\r\033[K")
}

// Silent is a reporter that says nothing, for callers that want no output at
// all rather than a nil check at every site.
type Silent struct{}

func (Silent) Step(string, ...any) {}
func (Silent) Done()               {}

var (
	_ ports.ProgressReporter = (*Progress)(nil)
	_ ports.ProgressReporter = Silent{}
)
