// Package ast defines the Abstract Syntax Tree structures for WDL documents.
package ast

// Document represents a complete WDL document
type Document struct {
	Version  string
	Imports  []*Import
	Structs  []*Struct
	Tasks    []*Task
	Workflow *Workflow
	Source   string // Source file path
}

// Import represents a WDL import statement
type Import struct {
	URI     string   // The import URI (can be relative path, http URL, etc.)
	As      string   // Optional alias for the import (import "x.wdl" as y)
	Aliases []*Alias // Optional member aliases (alias A as B)
}

// Alias represents an alias for an imported member
type Alias struct {
	Original string
	Alias    string
}

// Struct represents a WDL struct definition
type Struct struct {
	Name    string
	Members []*Declaration
}

// Task represents a WDL task
type Task struct {
	Name          string
	Inputs        []*Declaration
	Outputs       []*Declaration
	Command       string
	Runtime       map[string]Expression
	Meta          map[string]interface{}
	ParameterMeta map[string]interface{}
	Declarations  []*Declaration // private declarations
}

// Workflow represents a WDL workflow
type Workflow struct {
	Name          string
	Inputs        []*Declaration
	Outputs       []*Declaration
	Calls         []*Call
	Scatters      []*Scatter
	Conditionals  []*Conditional
	Declarations  []*Declaration
	Meta          map[string]interface{}
	ParameterMeta map[string]interface{}
}

// Declaration represents a WDL variable declaration
type Declaration struct {
	Type       *Type
	Name       string
	Expression Expression // nil if unbound
}

// Type represents a WDL type
type Type struct {
	Base     string // Int, String, File, Array, Map, Pair, etc.
	Optional bool
	// For compound types:
	ArrayType *Type // element type for Array
	MapKey    *Type // key type for Map
	MapValue  *Type // value type for Map
	PairLeft  *Type // left type for Pair
	PairRight *Type // right type for Pair
	NonEmpty  bool  // for Array+ (non-empty array)
}

// String returns a string representation of the type
func (t *Type) String() string {
	result := ""
	switch t.Base {
	case "Array":
		suffix := ""
		if t.NonEmpty {
			suffix = "+"
		}
		result = "Array[" + t.ArrayType.String() + "]" + suffix
	case "Map":
		result = "Map[" + t.MapKey.String() + ", " + t.MapValue.String() + "]"
	case "Pair":
		result = "Pair[" + t.PairLeft.String() + ", " + t.PairRight.String() + "]"
	default:
		result = t.Base
	}
	if t.Optional {
		result += "?"
	}
	return result
}

// Expression is an interface for WDL expressions
type Expression interface {
	ExpressionNode()
}

// Literal represents a literal value
type Literal struct {
	Value interface{}
}

func (l *Literal) ExpressionNode() {}

// Identifier represents an identifier reference
type Identifier struct {
	Name string
}

func (i *Identifier) ExpressionNode() {}

// MemberAccess represents accessing a member of an expression (e.g., foo.bar)
type MemberAccess struct {
	Expression Expression
	Member     string
}

func (m *MemberAccess) ExpressionNode() {}

// IndexAccess represents array/map access (e.g., foo[0])
type IndexAccess struct {
	Expression Expression
	Index      Expression
}

func (i *IndexAccess) ExpressionNode() {}

// FunctionCall represents a function call (e.g., read_string(file))
type FunctionCall struct {
	Name      string
	Arguments []Expression
}

func (f *FunctionCall) ExpressionNode() {}

// BinaryOp represents a binary operation
type BinaryOp struct {
	Left     Expression
	Operator string
	Right    Expression
}

func (b *BinaryOp) ExpressionNode() {}

// UnaryOp represents a unary operation
type UnaryOp struct {
	Operator   string
	Expression Expression
}

func (u *UnaryOp) ExpressionNode() {}

// TernaryOp represents a ternary/conditional expression (if-then-else)
type TernaryOp struct {
	Condition Expression
	IfTrue    Expression
	IfFalse   Expression
}

func (t *TernaryOp) ExpressionNode() {}

// ArrayLiteral represents an array literal [a, b, c]
type ArrayLiteral struct {
	Elements []Expression
}

func (a *ArrayLiteral) ExpressionNode() {}

// MapLiteral represents a map literal {a: b, c: d}
type MapLiteral struct {
	Entries map[Expression]Expression
}

func (m *MapLiteral) ExpressionNode() {}

// PairLiteral represents a pair literal (a, b)
type PairLiteral struct {
	Left  Expression
	Right Expression
}

func (p *PairLiteral) ExpressionNode() {}

// ObjectLiteral represents an object literal
type ObjectLiteral struct {
	Members map[string]Expression
}

func (o *ObjectLiteral) ExpressionNode() {}

// StringInterpolation represents a string with interpolated expressions
type StringInterpolation struct {
	Parts []StringPart
}

func (s *StringInterpolation) ExpressionNode() {}

// StringPart is a part of an interpolated string
type StringPart interface {
	StringPartNode()
}

// StringLiteral is a literal part of a string
type StringLiteral struct {
	Value string
}

func (s *StringLiteral) StringPartNode() {}
func (s *StringLiteral) ExpressionNode() {}

// StringPlaceholder is an expression placeholder in a string
type StringPlaceholder struct {
	Expression Expression
	Options    *PlaceholderOptions
}

func (s *StringPlaceholder) StringPartNode() {}

// PlaceholderOptions represents options in a placeholder like ${sep=", " arr}
type PlaceholderOptions struct {
	Sep     *string
	Default Expression
	True    *string
	False   *string
}

// Call represents a call to a task or workflow
type Call struct {
	Target string // The fully qualified name (e.g., "module.TaskName")
	Alias  string // Optional alias for the call
	Inputs map[string]Expression
	After  []string // tasks this call should run after
}

// Scatter represents a scatter block
type Scatter struct {
	Variable   string
	Expression Expression
	Body       []WorkflowElement
}

// Conditional represents a conditional (if) block
type Conditional struct {
	Condition Expression
	Body      []WorkflowElement
}

// WorkflowElement is an interface for elements that can appear in a workflow body
type WorkflowElement interface {
	WorkflowElementNode()
}

func (d *Declaration) WorkflowElementNode() {}
func (c *Call) WorkflowElementNode()        {}
func (s *Scatter) WorkflowElementNode()     {}
func (c *Conditional) WorkflowElementNode() {}
