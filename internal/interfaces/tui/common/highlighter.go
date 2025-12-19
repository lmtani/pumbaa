package common

import (
	"bytes"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/muesli/reflow/wordwrap"
)

// ContentProfile defines the type of content for highlighting
type ContentProfile string

const (
	ProfileStream ContentProfile = "stream" // stdout, stderr, *.out, *.err
	ProfileLog    ContentProfile = "log"    // *.log
	ProfileShell  ContentProfile = "shell"  // *.sh, *.bash, *.zsh, commands
	ProfileJSON   ContentProfile = "json"   // json content
	ProfileText   ContentProfile = "text"   // fallback
)

// Custom log lexer for bioinformatics/Java logs
var bioLogLexer = chroma.MustNewLexer(
	&chroma.Config{
		Name:      "BioLog",
		Aliases:   []string{"biolog", "cromwelllog"},
		Filenames: []string{"*.log", "stderr", "stdout"},
		MimeTypes: []string{"text/x-log"},
	},
	func() chroma.Rules {
		return chroma.Rules{
			"root": {
				// Bracketed timestamps: [Thu Dec 18 14:38:53 GMT 2025]
				{Pattern: `\[[A-Za-z]{3} [A-Za-z]{3} \d{1,2} \d{2}:\d{2}:\d{2} [A-Z]{3,4} \d{4}\]`, Type: chroma.Comment, Mutator: nil},
				// ISO-like timestamps: 2025-12-18 14:39:13 or 14:38:53.778
				{Pattern: `\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}`, Type: chroma.Comment, Mutator: nil},
				{Pattern: `\d{2}:\d{2}:\d{2}\.\d{3}`, Type: chroma.Comment, Mutator: nil},
				// Log levels
				{Pattern: `\bERROR\b`, Type: chroma.GenericError, Mutator: nil},
				{Pattern: `\bFATAL\b`, Type: chroma.GenericError, Mutator: nil},
				{Pattern: `\bFAILED\b`, Type: chroma.GenericError, Mutator: nil},
				{Pattern: `\bWARN(ING)?\b`, Type: chroma.GenericEmph, Mutator: nil},
				{Pattern: `\bINFO\b`, Type: chroma.GenericInserted, Mutator: nil},
				{Pattern: `\bDEBUG\b`, Type: chroma.GenericOutput, Mutator: nil},
				{Pattern: `\bSUCCESS\b`, Type: chroma.GenericInserted, Mutator: nil},
				{Pattern: `\bDONE\b`, Type: chroma.GenericInserted, Mutator: nil},
				// GCS paths
				{Pattern: `gs://[^\s]+`, Type: chroma.LiteralString, Mutator: nil},
				// Local paths
				{Pattern: `/[a-zA-Z0-9_\-./]+`, Type: chroma.LiteralString, Mutator: nil},
				// Java class names (CamelCase with dots)
				{Pattern: `[A-Z][a-zA-Z0-9]*(\.[A-Z][a-zA-Z0-9]*)+`, Type: chroma.NameClass, Mutator: nil},
				// Tool/class names (single CamelCase word followed by specific patterns)
				{Pattern: `\b[A-Z][a-zA-Z0-9]+(?=\s+(--|done|Reverted|Loading))`, Type: chroma.NameFunction, Mutator: nil},
				// Numbers with units or formatting
				{Pattern: `\d{1,3}(,\d{3})+`, Type: chroma.LiteralNumber, Mutator: nil},
				{Pattern: `\d+(\.\d+)?\s*(MB|GB|KB|ms|s|minutes?)`, Type: chroma.LiteralNumber, Mutator: nil},
				{Pattern: `\d+\.\d+`, Type: chroma.LiteralNumber, Mutator: nil},
				// Elapsed time patterns
				{Pattern: `\d{2}:\d{2}:\d{2}s`, Type: chroma.LiteralNumber, Mutator: nil},
				// Key=value pairs (common in Java tools)
				{Pattern: `--[A-Z_]+`, Type: chroma.NameAttribute, Mutator: nil},
				// Chromosome positions like chr2:203,729,685
				{Pattern: `chr[0-9XYM]+:\d{1,3}(,\d{3})*`, Type: chroma.LiteralStringSymbol, Mutator: nil},
				// Everything else
				{Pattern: `.`, Type: chroma.Text, Mutator: nil},
			},
		}
	},
)

// DetectProfile returns the content profile based on filename
func DetectProfile(filename string) ContentProfile {
	filename = strings.ToLower(filename)

	// Extract basename for path detection (handles gs://bucket/.../stderr)
	basename := filename
	if idx := strings.LastIndex(filename, "/"); idx >= 0 {
		basename = filename[idx+1:]
	}

	// Priority checks for specific profiles
	if basename == "command" || strings.HasSuffix(basename, ".sh") || strings.HasSuffix(basename, ".bash") || strings.HasSuffix(basename, ".zsh") {
		return ProfileShell
	}

	if basename == "stdout" || basename == "stderr" || strings.HasSuffix(basename, ".out") || strings.HasSuffix(basename, ".err") {
		return ProfileStream
	}

	if strings.HasSuffix(basename, ".log") || strings.Contains(filename, "workflow") || basename == "workflow log" {
		return ProfileLog
	}

	return ProfileText
}

// Highlight applies syntax highlighting to the content based on profile
func Highlight(content string, profile ContentProfile, width int) string {
	if profile == ProfileText || content == "" {
		return content
	}

	var lexer chroma.Lexer
	switch profile {
	case ProfileShell:
		lexer = lexers.Get("bash")
	case ProfileJSON:
		lexer = lexers.Get("json")
	case ProfileLog, ProfileStream:
		// Use our custom bioinformatics log lexer
		lexer = bioLogLexer
	default:
		lexer = lexers.Analyse(content)
	}

	if lexer == nil {
		lexer = lexers.Fallback
	}

	style := styles.Get("monokai")
	if style == nil {
		style = styles.Fallback
	}

	formatter := formatters.Get("terminal256")
	if formatter == nil {
		formatter = formatters.Fallback
	}

	iterator, err := lexer.Tokenise(nil, content)
	if err != nil {
		return content
	}

	var buf bytes.Buffer
	err = formatter.Format(&buf, style, iterator)
	if err != nil {
		return content
	}

	result := buf.String()
	if width > 0 {
		return wordwrap.String(result, width)
	}
	return result
}

// HighlightWithFilename detects profile and applies highlighting
func HighlightWithFilename(content, filename string, width int) string {
	profile := DetectProfile(filename)
	return Highlight(content, profile, width)
}
