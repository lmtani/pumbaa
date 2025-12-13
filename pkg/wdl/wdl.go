// Package wdl provides parsing, analysis, and bundling capabilities for WDL files.
//
// This package allows you to:
//   - Parse WDL files into an Abstract Syntax Tree (AST)
//   - Analyze dependencies between WDL files
//   - Create self-contained bundles with all dependencies
//
// Example usage:
//
//	doc, err := wdl.Parse("workflow.wdl")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	graph, err := wdl.AnalyzeDependencies(doc, "workflow.wdl")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	err = wdl.CreateBundle("workflow.wdl", "bundle.zip")
//	if err != nil {
//	    log.Fatal(err)
//	}
package wdl

import (
	"fmt"
	"os"

	"github.com/antlr4-go/antlr/v4"
	"github.com/lmtani/pumbaa/pkg/wdl/ast"
	"github.com/lmtani/pumbaa/pkg/wdl/parser"
	"github.com/lmtani/pumbaa/pkg/wdl/visitor"
)

// Parse parses a WDL file and returns the AST Document
func Parse(filename string) (*ast.Document, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filename, err)
	}

	doc, err := ParseBytes(data)
	if err != nil {
		return nil, err
	}

	doc.Source = filename
	return doc, nil
}

// ParseBytes parses WDL content from bytes and returns the AST Document
func ParseBytes(data []byte) (*ast.Document, error) {
	input := antlr.NewInputStream(string(data))
	lexer := parser.NewWdlV1_1Lexer(input)

	// Custom error listener for lexer
	lexerErrors := &errorListener{}
	lexer.RemoveErrorListeners()
	lexer.AddErrorListener(lexerErrors)

	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	p := parser.NewWdlV1_1Parser(stream)

	// Custom error listener for parser
	parserErrors := &errorListener{}
	p.RemoveErrorListeners()
	p.AddErrorListener(parserErrors)

	// Parse the document
	tree := p.Document()

	// Check for errors
	if len(lexerErrors.errors) > 0 {
		return nil, fmt.Errorf("lexer errors: %v", lexerErrors.errors)
	}
	if len(parserErrors.errors) > 0 {
		return nil, fmt.Errorf("parser errors: %v", parserErrors.errors)
	}

	// Build AST using visitor
	v := visitor.NewWDLVisitor()
	result := v.VisitDocument(tree.(*parser.DocumentContext))

	doc, ok := result.(*ast.Document)
	if !ok {
		return nil, fmt.Errorf("failed to build AST")
	}

	return doc, nil
}

// errorListener collects syntax errors during parsing
type errorListener struct {
	*antlr.DefaultErrorListener
	errors []string
}

func (l *errorListener) SyntaxError(recognizer antlr.Recognizer, offendingSymbol interface{},
	line, column int, msg string, e antlr.RecognitionException) {
	l.errors = append(l.errors, fmt.Sprintf("line %d:%d %s", line, column, msg))
}
