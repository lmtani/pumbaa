// Package visitor implements a visitor pattern to build AST from parse tree.
package visitor

import (
	"strconv"
	"strings"

	"github.com/antlr4-go/antlr/v4"

	"github.com/lmtani/pumbaa/pkg/wdl/ast"
	"github.com/lmtani/pumbaa/pkg/wdl/parser"
)

// WDLVisitor implements the visitor pattern to build AST from parse tree
type WDLVisitor struct {
	parser.BaseWdlV1_1ParserVisitor
}

// NewWDLVisitor creates a new WDL visitor
func NewWDLVisitor() *WDLVisitor {
	return &WDLVisitor{}
}

// Visit dispatches to the appropriate visit method
func (v *WDLVisitor) Visit(tree antlr.ParseTree) interface{} {
	return tree.Accept(v)
}

// VisitDocument visits a document node and builds the Document AST
func (v *WDLVisitor) VisitDocument(ctx *parser.DocumentContext) interface{} {
	doc := &ast.Document{
		Imports: make([]*ast.Import, 0),
		Structs: make([]*ast.Struct, 0),
		Tasks:   make([]*ast.Task, 0),
	}

	// Visit version
	if ctx.Version() != nil {
		doc.Version = v.VisitVersion(ctx.Version().(*parser.VersionContext)).(string)
	}

	// Visit document elements (imports, structs, tasks)
	for _, elem := range ctx.AllDocument_element() {
		result := v.VisitDocument_element(elem.(*parser.Document_elementContext))
		switch r := result.(type) {
		case *ast.Import:
			doc.Imports = append(doc.Imports, r)
		case *ast.Struct:
			doc.Structs = append(doc.Structs, r)
		case *ast.Task:
			doc.Tasks = append(doc.Tasks, r)
		}
	}

	// Visit workflow if present
	if ctx.Workflow() != nil {
		doc.Workflow = v.VisitWorkflow(ctx.Workflow().(*parser.WorkflowContext)).(*ast.Workflow)
	}

	return doc
}

// VisitVersion extracts the WDL version string
func (v *WDLVisitor) VisitVersion(ctx *parser.VersionContext) interface{} {
	if ctx.ReleaseVersion() != nil {
		return ctx.ReleaseVersion().GetText()
	}
	return ""
}

// VisitDocument_element visits document elements (import, struct, task)
func (v *WDLVisitor) VisitDocument_element(ctx *parser.Document_elementContext) interface{} {
	if ctx.Import_doc() != nil {
		return v.VisitImport_doc(ctx.Import_doc().(*parser.Import_docContext))
	}
	if ctx.Struct_() != nil {
		return v.VisitStruct(ctx.Struct_().(*parser.StructContext))
	}
	if ctx.Task() != nil {
		return v.VisitTask(ctx.Task().(*parser.TaskContext))
	}
	return nil
}

// VisitImport_doc visits an import statement
func (v *WDLVisitor) VisitImport_doc(ctx *parser.Import_docContext) interface{} {
	imp := &ast.Import{}

	// Get the import URI (removing quotes)
	if ctx.String_() != nil {
		uri := v.visitStringValue(ctx.String_().(*parser.StringContext))
		imp.URI = uri
	}

	// Get optional alias (import "x.wdl" as y)
	if ctx.Import_as() != nil {
		imp.As = ctx.Import_as().(*parser.Import_asContext).Identifier().GetText()
	}

	// Get optional member aliases
	imp.Aliases = make([]*ast.Alias, 0)
	for _, aliasCtx := range ctx.AllImport_alias() {
		alias := v.VisitImport_alias(aliasCtx.(*parser.Import_aliasContext)).(*ast.Alias)
		imp.Aliases = append(imp.Aliases, alias)
	}

	return imp
}

// VisitImport_alias visits an import alias (alias A as B)
func (v *WDLVisitor) VisitImport_alias(ctx *parser.Import_aliasContext) interface{} {
	identifiers := ctx.AllIdentifier()
	if len(identifiers) >= 2 {
		return &ast.Alias{
			Original: identifiers[0].GetText(),
			Alias:    identifiers[1].GetText(),
		}
	}
	return &ast.Alias{}
}

// visitStringValue extracts the string content without quotes
func (v *WDLVisitor) visitStringValue(ctx *parser.StringContext) string {
	text := ctx.GetText()
	// Remove surrounding quotes
	if len(text) >= 2 {
		if (text[0] == '"' && text[len(text)-1] == '"') ||
			(text[0] == '\'' && text[len(text)-1] == '\'') {
			return text[1 : len(text)-1]
		}
	}
	return text
}

// VisitStruct visits a struct definition
func (v *WDLVisitor) VisitStruct(ctx *parser.StructContext) interface{} {
	s := &ast.Struct{
		Name:    ctx.Identifier().GetText(),
		Members: make([]*ast.Declaration, 0),
	}

	for _, declCtx := range ctx.AllUnbound_decls() {
		decl := v.VisitUnbound_decls(declCtx.(*parser.Unbound_declsContext)).(*ast.Declaration)
		s.Members = append(s.Members, decl)
	}

	return s
}

// VisitTask visits a task definition
func (v *WDLVisitor) VisitTask(ctx *parser.TaskContext) interface{} {
	task := &ast.Task{
		Name:          ctx.Identifier().GetText(),
		Inputs:        make([]*ast.Declaration, 0),
		Outputs:       make([]*ast.Declaration, 0),
		Runtime:       make(map[string]ast.Expression),
		Meta:          make(map[string]interface{}),
		ParameterMeta: make(map[string]interface{}),
		Declarations:  make([]*ast.Declaration, 0),
	}

	for _, elemCtx := range ctx.AllTask_element() {
		elem := elemCtx.(*parser.Task_elementContext)

		if elem.Task_input() != nil {
			inputs := v.VisitTask_input(elem.Task_input().(*parser.Task_inputContext)).([]*ast.Declaration)
			task.Inputs = append(task.Inputs, inputs...)
		}

		if elem.Task_output() != nil {
			outputs := v.VisitTask_output(elem.Task_output().(*parser.Task_outputContext)).([]*ast.Declaration)
			task.Outputs = append(task.Outputs, outputs...)
		}

		if elem.Task_command() != nil {
			task.Command = v.VisitTask_command(elem.Task_command().(*parser.Task_commandContext)).(string)
		}

		if elem.Task_runtime() != nil {
			runtime := v.VisitTask_runtime(elem.Task_runtime().(*parser.Task_runtimeContext)).(map[string]ast.Expression)
			for k, val := range runtime {
				task.Runtime[k] = val
			}
		}

		if elem.Bound_decls() != nil {
			decl := v.VisitBound_decls(elem.Bound_decls().(*parser.Bound_declsContext)).(*ast.Declaration)
			task.Declarations = append(task.Declarations, decl)
		}

		if elem.Meta() != nil {
			meta := v.VisitMeta(elem.Meta().(*parser.MetaContext)).(map[string]interface{})
			for k, val := range meta {
				task.Meta[k] = val
			}
		}

		if elem.Parameter_meta() != nil {
			paramMeta := v.VisitParameter_meta(elem.Parameter_meta().(*parser.Parameter_metaContext)).(map[string]interface{})
			for k, val := range paramMeta {
				task.ParameterMeta[k] = val
			}
		}
	}

	return task
}

// VisitTask_input visits task input section
func (v *WDLVisitor) VisitTask_input(ctx *parser.Task_inputContext) interface{} {
	inputs := make([]*ast.Declaration, 0)
	for _, declCtx := range ctx.AllAny_decls() {
		decl := v.VisitAny_decls(declCtx.(*parser.Any_declsContext)).(*ast.Declaration)
		inputs = append(inputs, decl)
	}
	return inputs
}

// VisitTask_output visits task output section
func (v *WDLVisitor) VisitTask_output(ctx *parser.Task_outputContext) interface{} {
	outputs := make([]*ast.Declaration, 0)
	for _, declCtx := range ctx.AllBound_decls() {
		decl := v.VisitBound_decls(declCtx.(*parser.Bound_declsContext)).(*ast.Declaration)
		outputs = append(outputs, decl)
	}
	return outputs
}

// VisitTask_command visits task command section
func (v *WDLVisitor) VisitTask_command(ctx *parser.Task_commandContext) interface{} {
	var sb strings.Builder

	// Get the string parts
	if ctx.Task_command_string_part() != nil {
		sb.WriteString(ctx.Task_command_string_part().GetText())
	}

	// Get the expression parts interleaved with string parts
	for _, exprStr := range ctx.AllTask_command_expr_with_string() {
		sb.WriteString(exprStr.GetText())
	}

	return sb.String()
}

// VisitTask_runtime visits task runtime section
func (v *WDLVisitor) VisitTask_runtime(ctx *parser.Task_runtimeContext) interface{} {
	runtime := make(map[string]ast.Expression)
	for _, kvCtx := range ctx.AllTask_runtime_kv() {
		kv := kvCtx.(*parser.Task_runtime_kvContext)
		key := kv.Identifier().GetText()
		expr := v.VisitExpr(kv.Expr().(*parser.ExprContext)).(ast.Expression)
		runtime[key] = expr
	}
	return runtime
}

// VisitMeta visits meta section
func (v *WDLVisitor) VisitMeta(ctx *parser.MetaContext) interface{} {
	meta := make(map[string]interface{})
	for _, kvCtx := range ctx.AllMeta_kv() {
		kv := kvCtx.(*parser.Meta_kvContext)
		key := kv.MetaIdentifier().GetText()
		value := v.VisitMeta_value(kv.Meta_value().(*parser.Meta_valueContext))
		meta[key] = value
	}
	return meta
}

// VisitParameter_meta visits parameter_meta section
func (v *WDLVisitor) VisitParameter_meta(ctx *parser.Parameter_metaContext) interface{} {
	meta := make(map[string]interface{})
	for _, kvCtx := range ctx.AllMeta_kv() {
		kv := kvCtx.(*parser.Meta_kvContext)
		key := kv.MetaIdentifier().GetText()
		value := v.VisitMeta_value(kv.Meta_value().(*parser.Meta_valueContext))
		meta[key] = value
	}
	return meta
}

// VisitMeta_value visits meta values
func (v *WDLVisitor) VisitMeta_value(ctx *parser.Meta_valueContext) interface{} {
	if ctx.MetaNull() != nil {
		return nil
	}
	if ctx.MetaBool() != nil {
		return ctx.MetaBool().GetText() == "true"
	}
	if ctx.MetaInt() != nil {
		val, _ := strconv.ParseInt(ctx.MetaInt().GetText(), 10, 64)
		return val
	}
	if ctx.MetaFloat() != nil {
		val, _ := strconv.ParseFloat(ctx.MetaFloat().GetText(), 64)
		return val
	}
	if ctx.Meta_string() != nil {
		return v.VisitMeta_string(ctx.Meta_string().(*parser.Meta_stringContext))
	}
	if ctx.Meta_array() != nil {
		return v.VisitMeta_array(ctx.Meta_array().(*parser.Meta_arrayContext))
	}
	if ctx.Meta_object() != nil {
		return v.VisitMeta_object(ctx.Meta_object().(*parser.Meta_objectContext))
	}
	return nil
}

// VisitMeta_string visits meta string values
func (v *WDLVisitor) VisitMeta_string(ctx *parser.Meta_stringContext) interface{} {
	if ctx.Meta_string_part() != nil {
		return ctx.Meta_string_part().GetText()
	}
	return ""
}

// VisitMeta_array visits meta array values
func (v *WDLVisitor) VisitMeta_array(ctx *parser.Meta_arrayContext) interface{} {
	if ctx.MetaEmptyArray() != nil {
		return []interface{}{}
	}
	arr := make([]interface{}, 0)
	for _, valCtx := range ctx.AllMeta_value() {
		arr = append(arr, v.VisitMeta_value(valCtx.(*parser.Meta_valueContext)))
	}
	return arr
}

// VisitMeta_object visits meta object values
func (v *WDLVisitor) VisitMeta_object(ctx *parser.Meta_objectContext) interface{} {
	if ctx.MetaEmptyObject() != nil {
		return map[string]interface{}{}
	}
	obj := make(map[string]interface{})
	for _, kvCtx := range ctx.AllMeta_object_kv() {
		kv := kvCtx.(*parser.Meta_object_kvContext)
		key := kv.MetaObjectIdentifier().GetText()
		value := v.VisitMeta_value(kv.Meta_value().(*parser.Meta_valueContext))
		obj[key] = value
	}
	return obj
}

// VisitWorkflow visits a workflow definition
func (v *WDLVisitor) VisitWorkflow(ctx *parser.WorkflowContext) interface{} {
	wf := &ast.Workflow{
		Name:          ctx.Identifier().GetText(),
		Inputs:        make([]*ast.Declaration, 0),
		Outputs:       make([]*ast.Declaration, 0),
		Calls:         make([]*ast.Call, 0),
		Scatters:      make([]*ast.Scatter, 0),
		Conditionals:  make([]*ast.Conditional, 0),
		Declarations:  make([]*ast.Declaration, 0),
		Meta:          make(map[string]interface{}),
		ParameterMeta: make(map[string]interface{}),
	}

	for _, elemCtx := range ctx.AllWorkflow_element() {
		v.visitWorkflowElement(elemCtx, wf)
	}

	return wf
}

// visitWorkflowElement visits a workflow element and adds it to the workflow
func (v *WDLVisitor) visitWorkflowElement(ctx parser.IWorkflow_elementContext, wf *ast.Workflow) {
	switch elem := ctx.(type) {
	case *parser.InputContext:
		inputs := v.visitWorkflowInput(elem.Workflow_input().(*parser.Workflow_inputContext))
		wf.Inputs = append(wf.Inputs, inputs...)
	case *parser.OutputContext:
		outputs := v.visitWorkflowOutput(elem.Workflow_output().(*parser.Workflow_outputContext))
		wf.Outputs = append(wf.Outputs, outputs...)
	case *parser.Inner_elementContext:
		v.visitInnerWorkflowElement(elem.Inner_workflow_element(), wf)
	case *parser.Parameter_meta_elementContext:
		paramMeta := v.VisitParameter_meta(elem.Parameter_meta().(*parser.Parameter_metaContext)).(map[string]interface{})
		for k, val := range paramMeta {
			wf.ParameterMeta[k] = val
		}
	case *parser.Meta_elementContext:
		meta := v.VisitMeta(elem.Meta().(*parser.MetaContext)).(map[string]interface{})
		for k, val := range meta {
			wf.Meta[k] = val
		}
	}
}

// visitWorkflowInput visits workflow input section
func (v *WDLVisitor) visitWorkflowInput(ctx *parser.Workflow_inputContext) []*ast.Declaration {
	inputs := make([]*ast.Declaration, 0)
	for _, declCtx := range ctx.AllAny_decls() {
		decl := v.VisitAny_decls(declCtx.(*parser.Any_declsContext)).(*ast.Declaration)
		inputs = append(inputs, decl)
	}
	return inputs
}

// visitWorkflowOutput visits workflow output section
func (v *WDLVisitor) visitWorkflowOutput(ctx *parser.Workflow_outputContext) []*ast.Declaration {
	outputs := make([]*ast.Declaration, 0)
	for _, declCtx := range ctx.AllBound_decls() {
		decl := v.VisitBound_decls(declCtx.(*parser.Bound_declsContext)).(*ast.Declaration)
		outputs = append(outputs, decl)
	}
	return outputs
}

// visitInnerWorkflowElement visits inner workflow elements
func (v *WDLVisitor) visitInnerWorkflowElement(ctx parser.IInner_workflow_elementContext, wf *ast.Workflow) {
	innerCtx := ctx.(*parser.Inner_workflow_elementContext)

	if innerCtx.Bound_decls() != nil {
		decl := v.VisitBound_decls(innerCtx.Bound_decls().(*parser.Bound_declsContext)).(*ast.Declaration)
		wf.Declarations = append(wf.Declarations, decl)
	}

	if innerCtx.Call() != nil {
		call := v.VisitCall(innerCtx.Call().(*parser.CallContext)).(*ast.Call)
		wf.Calls = append(wf.Calls, call)
	}

	if innerCtx.Scatter() != nil {
		scatter := v.VisitScatter(innerCtx.Scatter().(*parser.ScatterContext)).(*ast.Scatter)
		wf.Scatters = append(wf.Scatters, scatter)
	}

	if innerCtx.Conditional() != nil {
		cond := v.VisitConditional(innerCtx.Conditional().(*parser.ConditionalContext)).(*ast.Conditional)
		wf.Conditionals = append(wf.Conditionals, cond)
	}
}

// VisitCall visits a call statement
func (v *WDLVisitor) VisitCall(ctx *parser.CallContext) interface{} {
	call := &ast.Call{
		Inputs: make(map[string]ast.Expression),
		After:  make([]string, 0),
	}

	// Get the call target (e.g., "module.TaskName")
	if ctx.Call_name() != nil {
		call.Target = ctx.Call_name().GetText()
	}

	// Get optional alias
	if ctx.Call_alias() != nil {
		call.Alias = ctx.Call_alias().(*parser.Call_aliasContext).Identifier().GetText()
	}

	// Get after dependencies
	for _, afterCtx := range ctx.AllCall_after() {
		call.After = append(call.After, afterCtx.(*parser.Call_afterContext).Identifier().GetText())
	}

	// Get call inputs
	if ctx.Call_body() != nil {
		body := ctx.Call_body().(*parser.Call_bodyContext)
		if body.Call_inputs() != nil {
			inputs := v.VisitCall_inputs(body.Call_inputs().(*parser.Call_inputsContext)).(map[string]ast.Expression)
			call.Inputs = inputs
		}
	}

	return call
}

// VisitCall_inputs visits call inputs
func (v *WDLVisitor) VisitCall_inputs(ctx *parser.Call_inputsContext) interface{} {
	inputs := make(map[string]ast.Expression)
	for _, inputCtx := range ctx.AllCall_input() {
		input := inputCtx.(*parser.Call_inputContext)
		name := input.Identifier().GetText()
		if input.Expr() != nil {
			expr := v.VisitExpr(input.Expr().(*parser.ExprContext)).(ast.Expression)
			inputs[name] = expr
		} else {
			// Shorthand: input name without value means the variable with same name
			inputs[name] = &ast.Identifier{Name: name}
		}
	}
	return inputs
}

// VisitScatter visits a scatter block
func (v *WDLVisitor) VisitScatter(ctx *parser.ScatterContext) interface{} {
	scatter := &ast.Scatter{
		Variable: ctx.Identifier().GetText(),
		Body:     make([]ast.WorkflowElement, 0),
	}

	if ctx.Expr() != nil {
		scatter.Expression = v.VisitExpr(ctx.Expr().(*parser.ExprContext)).(ast.Expression)
	}

	for _, innerCtx := range ctx.AllInner_workflow_element() {
		elem := v.visitInnerElementAsWorkflowElement(innerCtx.(*parser.Inner_workflow_elementContext))
		if elem != nil {
			scatter.Body = append(scatter.Body, elem)
		}
	}

	return scatter
}

// VisitConditional visits a conditional block
func (v *WDLVisitor) VisitConditional(ctx *parser.ConditionalContext) interface{} {
	cond := &ast.Conditional{
		Body: make([]ast.WorkflowElement, 0),
	}

	if ctx.Expr() != nil {
		cond.Condition = v.VisitExpr(ctx.Expr().(*parser.ExprContext)).(ast.Expression)
	}

	for _, innerCtx := range ctx.AllInner_workflow_element() {
		elem := v.visitInnerElementAsWorkflowElement(innerCtx.(*parser.Inner_workflow_elementContext))
		if elem != nil {
			cond.Body = append(cond.Body, elem)
		}
	}

	return cond
}

// visitInnerElementAsWorkflowElement converts inner element to WorkflowElement
func (v *WDLVisitor) visitInnerElementAsWorkflowElement(ctx *parser.Inner_workflow_elementContext) ast.WorkflowElement {
	if ctx.Bound_decls() != nil {
		return v.VisitBound_decls(ctx.Bound_decls().(*parser.Bound_declsContext)).(*ast.Declaration)
	}
	if ctx.Call() != nil {
		return v.VisitCall(ctx.Call().(*parser.CallContext)).(*ast.Call)
	}
	if ctx.Scatter() != nil {
		return v.VisitScatter(ctx.Scatter().(*parser.ScatterContext)).(*ast.Scatter)
	}
	if ctx.Conditional() != nil {
		return v.VisitConditional(ctx.Conditional().(*parser.ConditionalContext)).(*ast.Conditional)
	}
	return nil
}

// VisitAny_decls visits any declaration (bound or unbound)
func (v *WDLVisitor) VisitAny_decls(ctx *parser.Any_declsContext) interface{} {
	if ctx.Unbound_decls() != nil {
		return v.VisitUnbound_decls(ctx.Unbound_decls().(*parser.Unbound_declsContext))
	}
	if ctx.Bound_decls() != nil {
		return v.VisitBound_decls(ctx.Bound_decls().(*parser.Bound_declsContext))
	}
	return nil
}

// VisitUnbound_decls visits an unbound declaration
func (v *WDLVisitor) VisitUnbound_decls(ctx *parser.Unbound_declsContext) interface{} {
	decl := &ast.Declaration{
		Name: ctx.Identifier().GetText(),
	}
	if ctx.Wdl_type() != nil {
		decl.Type = v.VisitWdl_type(ctx.Wdl_type().(*parser.Wdl_typeContext)).(*ast.Type)
	}
	return decl
}

// VisitBound_decls visits a bound declaration
func (v *WDLVisitor) VisitBound_decls(ctx *parser.Bound_declsContext) interface{} {
	decl := &ast.Declaration{
		Name: ctx.Identifier().GetText(),
	}
	if ctx.Wdl_type() != nil {
		decl.Type = v.VisitWdl_type(ctx.Wdl_type().(*parser.Wdl_typeContext)).(*ast.Type)
	}
	if ctx.Expr() != nil {
		decl.Expression = v.VisitExpr(ctx.Expr().(*parser.ExprContext)).(ast.Expression)
	}
	return decl
}

// VisitWdl_type visits a WDL type
func (v *WDLVisitor) VisitWdl_type(ctx *parser.Wdl_typeContext) interface{} {
	t := &ast.Type{}

	if ctx.Type_base() != nil {
		baseType := v.VisitType_base(ctx.Type_base().(*parser.Type_baseContext)).(*ast.Type)
		*t = *baseType
	}

	// Check for optional marker
	if ctx.OPTIONAL() != nil {
		t.Optional = true
	}

	return t
}

// VisitType_base visits a base type
func (v *WDLVisitor) VisitType_base(ctx *parser.Type_baseContext) interface{} {
	t := &ast.Type{}

	if ctx.Array_type() != nil {
		arrayType := v.VisitArray_type(ctx.Array_type().(*parser.Array_typeContext)).(*ast.Type)
		return arrayType
	}
	if ctx.Map_type() != nil {
		mapType := v.VisitMap_type(ctx.Map_type().(*parser.Map_typeContext)).(*ast.Type)
		return mapType
	}
	if ctx.Pair_type() != nil {
		pairType := v.VisitPair_type(ctx.Pair_type().(*parser.Pair_typeContext)).(*ast.Type)
		return pairType
	}

	// Primitive types
	if ctx.STRING() != nil {
		t.Base = "String"
	} else if ctx.FILE() != nil {
		t.Base = "File"
	} else if ctx.BOOLEAN() != nil {
		t.Base = "Boolean"
	} else if ctx.INT() != nil {
		t.Base = "Int"
	} else if ctx.FLOAT() != nil {
		t.Base = "Float"
	} else if ctx.OBJECT() != nil {
		t.Base = "Object"
	} else if ctx.Identifier() != nil {
		// Custom type (struct)
		t.Base = ctx.Identifier().GetText()
	}

	return t
}

// VisitArray_type visits an array type
func (v *WDLVisitor) VisitArray_type(ctx *parser.Array_typeContext) interface{} {
	t := &ast.Type{
		Base: "Array",
	}
	if ctx.Wdl_type() != nil {
		t.ArrayType = v.VisitWdl_type(ctx.Wdl_type().(*parser.Wdl_typeContext)).(*ast.Type)
	}
	if ctx.PLUS() != nil {
		t.NonEmpty = true
	}
	return t
}

// VisitMap_type visits a map type
func (v *WDLVisitor) VisitMap_type(ctx *parser.Map_typeContext) interface{} {
	t := &ast.Type{
		Base: "Map",
	}
	types := ctx.AllWdl_type()
	if len(types) >= 2 {
		t.MapKey = v.VisitWdl_type(types[0].(*parser.Wdl_typeContext)).(*ast.Type)
		t.MapValue = v.VisitWdl_type(types[1].(*parser.Wdl_typeContext)).(*ast.Type)
	}
	return t
}

// VisitPair_type visits a pair type
func (v *WDLVisitor) VisitPair_type(ctx *parser.Pair_typeContext) interface{} {
	t := &ast.Type{
		Base: "Pair",
	}
	types := ctx.AllWdl_type()
	if len(types) >= 2 {
		t.PairLeft = v.VisitWdl_type(types[0].(*parser.Wdl_typeContext)).(*ast.Type)
		t.PairRight = v.VisitWdl_type(types[1].(*parser.Wdl_typeContext)).(*ast.Type)
	}
	return t
}

// VisitExpr visits an expression
func (v *WDLVisitor) VisitExpr(ctx *parser.ExprContext) interface{} {
	if ctx.Expr_infix() != nil {
		return v.visitExprInfix(ctx.Expr_infix())
	}
	return &ast.Literal{Value: nil}
}

// visitExprInfix visits an infix expression
func (v *WDLVisitor) visitExprInfix(ctx parser.IExpr_infixContext) ast.Expression {
	switch expr := ctx.(type) {
	case *parser.Infix0Context:
		return v.visitExprInfix0(expr.Expr_infix0())
	}
	return &ast.Literal{Value: nil}
}

// visitExprInfix0 handles OR expressions
func (v *WDLVisitor) visitExprInfix0(ctx parser.IExpr_infix0Context) ast.Expression {
	switch expr := ctx.(type) {
	case *parser.LorContext:
		return &ast.BinaryOp{
			Left:     v.visitExprInfix0(expr.Expr_infix0()),
			Operator: "||",
			Right:    v.visitExprInfix1(expr.Expr_infix1()),
		}
	case *parser.Infix1Context:
		return v.visitExprInfix1(expr.Expr_infix1())
	}
	return &ast.Literal{Value: nil}
}

// visitExprInfix1 handles AND expressions
func (v *WDLVisitor) visitExprInfix1(ctx parser.IExpr_infix1Context) ast.Expression {
	switch expr := ctx.(type) {
	case *parser.LandContext:
		return &ast.BinaryOp{
			Left:     v.visitExprInfix1(expr.Expr_infix1()),
			Operator: "&&",
			Right:    v.visitExprInfix2(expr.Expr_infix2()),
		}
	case *parser.Infix2Context:
		return v.visitExprInfix2(expr.Expr_infix2())
	}
	return &ast.Literal{Value: nil}
}

// visitExprInfix2 handles comparison expressions
func (v *WDLVisitor) visitExprInfix2(ctx parser.IExpr_infix2Context) ast.Expression {
	switch expr := ctx.(type) {
	case *parser.EqeqContext:
		return &ast.BinaryOp{
			Left:     v.visitExprInfix2(expr.Expr_infix2()),
			Operator: "==",
			Right:    v.visitExprInfix3(expr.Expr_infix3()),
		}
	case *parser.NeqContext:
		return &ast.BinaryOp{
			Left:     v.visitExprInfix2(expr.Expr_infix2()),
			Operator: "!=",
			Right:    v.visitExprInfix3(expr.Expr_infix3()),
		}
	case *parser.LtContext:
		return &ast.BinaryOp{
			Left:     v.visitExprInfix2(expr.Expr_infix2()),
			Operator: "<",
			Right:    v.visitExprInfix3(expr.Expr_infix3()),
		}
	case *parser.GtContext:
		return &ast.BinaryOp{
			Left:     v.visitExprInfix2(expr.Expr_infix2()),
			Operator: ">",
			Right:    v.visitExprInfix3(expr.Expr_infix3()),
		}
	case *parser.LteContext:
		return &ast.BinaryOp{
			Left:     v.visitExprInfix2(expr.Expr_infix2()),
			Operator: "<=",
			Right:    v.visitExprInfix3(expr.Expr_infix3()),
		}
	case *parser.GteContext:
		return &ast.BinaryOp{
			Left:     v.visitExprInfix2(expr.Expr_infix2()),
			Operator: ">=",
			Right:    v.visitExprInfix3(expr.Expr_infix3()),
		}
	case *parser.Infix3Context:
		return v.visitExprInfix3(expr.Expr_infix3())
	}
	return &ast.Literal{Value: nil}
}

// visitExprInfix3 handles addition/subtraction
func (v *WDLVisitor) visitExprInfix3(ctx parser.IExpr_infix3Context) ast.Expression {
	switch expr := ctx.(type) {
	case *parser.AddContext:
		return &ast.BinaryOp{
			Left:     v.visitExprInfix3(expr.Expr_infix3()),
			Operator: "+",
			Right:    v.visitExprInfix4(expr.Expr_infix4()),
		}
	case *parser.SubContext:
		return &ast.BinaryOp{
			Left:     v.visitExprInfix3(expr.Expr_infix3()),
			Operator: "-",
			Right:    v.visitExprInfix4(expr.Expr_infix4()),
		}
	case *parser.Infix4Context:
		return v.visitExprInfix4(expr.Expr_infix4())
	}
	return &ast.Literal{Value: nil}
}

// visitExprInfix4 handles multiplication/division/modulo
func (v *WDLVisitor) visitExprInfix4(ctx parser.IExpr_infix4Context) ast.Expression {
	switch expr := ctx.(type) {
	case *parser.MulContext:
		return &ast.BinaryOp{
			Left:     v.visitExprInfix4(expr.Expr_infix4()),
			Operator: "*",
			Right:    v.visitExprInfix5(expr.Expr_infix5()),
		}
	case *parser.DivideContext:
		return &ast.BinaryOp{
			Left:     v.visitExprInfix4(expr.Expr_infix4()),
			Operator: "/",
			Right:    v.visitExprInfix5(expr.Expr_infix5()),
		}
	case *parser.ModContext:
		return &ast.BinaryOp{
			Left:     v.visitExprInfix4(expr.Expr_infix4()),
			Operator: "%",
			Right:    v.visitExprInfix5(expr.Expr_infix5()),
		}
	case *parser.Infix5Context:
		return v.visitExprInfix5(expr.Expr_infix5())
	}
	return &ast.Literal{Value: nil}
}

// visitExprInfix5 handles core expressions
func (v *WDLVisitor) visitExprInfix5(ctx parser.IExpr_infix5Context) ast.Expression {
	expr5 := ctx.(*parser.Expr_infix5Context)
	if expr5.Expr_core() != nil {
		return v.visitExprCore(expr5.Expr_core())
	}
	return &ast.Literal{Value: nil}
}

// visitExprCore handles core expression types
func (v *WDLVisitor) visitExprCore(ctx parser.IExpr_coreContext) ast.Expression {
	switch expr := ctx.(type) {
	case *parser.ApplyContext:
		return v.visitApply(expr)
	case *parser.Array_literalContext:
		return v.visitArrayLiteral(expr)
	case *parser.Pair_literalContext:
		return v.visitPairLiteral(expr)
	case *parser.Map_literalContext:
		return v.visitMapLiteral(expr)
	case *parser.Object_literalContext:
		return v.visitObjectLiteral(expr)
	case *parser.Struct_literalContext:
		return v.visitStructLiteral(expr)
	case *parser.IfthenelseContext:
		return v.visitIfThenElse(expr)
	case *parser.Expression_groupContext:
		return v.VisitExpr(expr.Expr().(*parser.ExprContext)).(ast.Expression)
	case *parser.AtContext:
		return v.visitAt(expr)
	case *parser.Get_nameContext:
		return v.visitGetName(expr)
	case *parser.NegateContext:
		return &ast.UnaryOp{
			Operator:   "!",
			Expression: v.VisitExpr(expr.Expr().(*parser.ExprContext)).(ast.Expression),
		}
	case *parser.UnarysignedContext:
		op := "+"
		if expr.MINUS() != nil {
			op = "-"
		}
		return &ast.UnaryOp{
			Operator:   op,
			Expression: v.VisitExpr(expr.Expr().(*parser.ExprContext)).(ast.Expression),
		}
	case *parser.PrimitivesContext:
		return v.visitPrimitives(expr)
	case *parser.Left_nameContext:
		return &ast.Identifier{Name: expr.Identifier().GetText()}
	}
	return &ast.Literal{Value: nil}
}

// visitApply handles function calls
func (v *WDLVisitor) visitApply(ctx *parser.ApplyContext) ast.Expression {
	fc := &ast.FunctionCall{
		Name:      ctx.Identifier().GetText(),
		Arguments: make([]ast.Expression, 0),
	}
	for _, exprCtx := range ctx.AllExpr() {
		arg := v.VisitExpr(exprCtx.(*parser.ExprContext)).(ast.Expression)
		fc.Arguments = append(fc.Arguments, arg)
	}
	return fc
}

// visitArrayLiteral handles array literals
func (v *WDLVisitor) visitArrayLiteral(ctx *parser.Array_literalContext) ast.Expression {
	arr := &ast.ArrayLiteral{
		Elements: make([]ast.Expression, 0),
	}
	for _, exprCtx := range ctx.AllExpr() {
		elem := v.VisitExpr(exprCtx.(*parser.ExprContext)).(ast.Expression)
		arr.Elements = append(arr.Elements, elem)
	}
	return arr
}

// visitPairLiteral handles pair literals
func (v *WDLVisitor) visitPairLiteral(ctx *parser.Pair_literalContext) ast.Expression {
	exprs := ctx.AllExpr()
	if len(exprs) >= 2 {
		return &ast.PairLiteral{
			Left:  v.VisitExpr(exprs[0].(*parser.ExprContext)).(ast.Expression),
			Right: v.VisitExpr(exprs[1].(*parser.ExprContext)).(ast.Expression),
		}
	}
	return &ast.PairLiteral{}
}

// visitMapLiteral handles map literals
func (v *WDLVisitor) visitMapLiteral(ctx *parser.Map_literalContext) ast.Expression {
	m := &ast.MapLiteral{
		Entries: make(map[ast.Expression]ast.Expression),
	}
	exprs := ctx.AllExpr()
	for i := 0; i < len(exprs)-1; i += 2 {
		key := v.VisitExpr(exprs[i].(*parser.ExprContext)).(ast.Expression)
		value := v.VisitExpr(exprs[i+1].(*parser.ExprContext)).(ast.Expression)
		m.Entries[key] = value
	}
	return m
}

// visitObjectLiteral handles object literals
func (v *WDLVisitor) visitObjectLiteral(ctx *parser.Object_literalContext) ast.Expression {
	obj := &ast.ObjectLiteral{
		Members: make(map[string]ast.Expression),
	}
	members := ctx.AllMember()
	exprs := ctx.AllExpr()
	for i := 0; i < len(members) && i < len(exprs); i++ {
		key := members[i].(*parser.MemberContext).Identifier().GetText()
		value := v.VisitExpr(exprs[i].(*parser.ExprContext)).(ast.Expression)
		obj.Members[key] = value
	}
	return obj
}

// visitStructLiteral handles struct literals
func (v *WDLVisitor) visitStructLiteral(ctx *parser.Struct_literalContext) ast.Expression {
	obj := &ast.ObjectLiteral{
		Members: make(map[string]ast.Expression),
	}
	members := ctx.AllMember()
	exprs := ctx.AllExpr()
	for i := 0; i < len(members) && i < len(exprs); i++ {
		key := members[i].(*parser.MemberContext).Identifier().GetText()
		value := v.VisitExpr(exprs[i].(*parser.ExprContext)).(ast.Expression)
		obj.Members[key] = value
	}
	return obj
}

// visitIfThenElse handles ternary expressions
func (v *WDLVisitor) visitIfThenElse(ctx *parser.IfthenelseContext) ast.Expression {
	exprs := ctx.AllExpr()
	if len(exprs) >= 3 {
		return &ast.TernaryOp{
			Condition: v.VisitExpr(exprs[0].(*parser.ExprContext)).(ast.Expression),
			IfTrue:    v.VisitExpr(exprs[1].(*parser.ExprContext)).(ast.Expression),
			IfFalse:   v.VisitExpr(exprs[2].(*parser.ExprContext)).(ast.Expression),
		}
	}
	return &ast.TernaryOp{}
}

// visitAt handles index access
func (v *WDLVisitor) visitAt(ctx *parser.AtContext) ast.Expression {
	return &ast.IndexAccess{
		Expression: v.visitExprCore(ctx.Expr_core()),
		Index:      v.VisitExpr(ctx.Expr().(*parser.ExprContext)).(ast.Expression),
	}
}

// visitGetName handles member access
func (v *WDLVisitor) visitGetName(ctx *parser.Get_nameContext) ast.Expression {
	return &ast.MemberAccess{
		Expression: v.visitExprCore(ctx.Expr_core()),
		Member:     ctx.Identifier().GetText(),
	}
}

// visitPrimitives handles primitive literals
func (v *WDLVisitor) visitPrimitives(ctx *parser.PrimitivesContext) ast.Expression {
	primCtx := ctx.Primitive_literal().(*parser.Primitive_literalContext)

	if primCtx.BoolLiteral() != nil {
		return &ast.Literal{Value: primCtx.BoolLiteral().GetText() == "true"}
	}
	if primCtx.Number() != nil {
		numCtx := primCtx.Number().(*parser.NumberContext)
		if numCtx.IntLiteral() != nil {
			val, _ := strconv.ParseInt(numCtx.IntLiteral().GetText(), 10, 64)
			return &ast.Literal{Value: val}
		}
		if numCtx.FloatLiteral() != nil {
			val, _ := strconv.ParseFloat(numCtx.FloatLiteral().GetText(), 64)
			return &ast.Literal{Value: val}
		}
	}
	if primCtx.String_() != nil {
		return &ast.StringLiteral{Value: v.visitStringValue(primCtx.String_().(*parser.StringContext))}
	}
	if primCtx.NONELITERAL() != nil {
		return &ast.Literal{Value: nil}
	}
	if primCtx.Identifier() != nil {
		return &ast.Identifier{Name: primCtx.Identifier().GetText()}
	}

	return &ast.Literal{Value: nil}
}
