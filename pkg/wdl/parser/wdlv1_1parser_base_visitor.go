// Code generated from WdlV1_1Parser.g4 by ANTLR 4.13.1. DO NOT EDIT.

package parser // WdlV1_1Parser
import "github.com/antlr4-go/antlr/v4"

type BaseWdlV1_1ParserVisitor struct {
	*antlr.BaseParseTreeVisitor
}

func (v *BaseWdlV1_1ParserVisitor) VisitMap_type(ctx *Map_typeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitArray_type(ctx *Array_typeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitPair_type(ctx *Pair_typeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitType_base(ctx *Type_baseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitWdl_type(ctx *Wdl_typeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitUnbound_decls(ctx *Unbound_declsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitBound_decls(ctx *Bound_declsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitAny_decls(ctx *Any_declsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitNumber(ctx *NumberContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitExpression_placeholder_option(ctx *Expression_placeholder_optionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitString_part(ctx *String_partContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitString_expr_part(ctx *String_expr_partContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitString_expr_with_string_part(ctx *String_expr_with_string_partContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitString(ctx *StringContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitPrimitive_literal(ctx *Primitive_literalContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitExpr(ctx *ExprContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitInfix0(ctx *Infix0Context) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitInfix1(ctx *Infix1Context) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitLor(ctx *LorContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitInfix2(ctx *Infix2Context) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitLand(ctx *LandContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitEqeq(ctx *EqeqContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitLt(ctx *LtContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitInfix3(ctx *Infix3Context) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitGte(ctx *GteContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitNeq(ctx *NeqContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitLte(ctx *LteContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitGt(ctx *GtContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitAdd(ctx *AddContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitSub(ctx *SubContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitInfix4(ctx *Infix4Context) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitMod(ctx *ModContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitMul(ctx *MulContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitDivide(ctx *DivideContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitInfix5(ctx *Infix5Context) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitExpr_infix5(ctx *Expr_infix5Context) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitMember(ctx *MemberContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitPair_literal(ctx *Pair_literalContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitUnarysigned(ctx *UnarysignedContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitApply(ctx *ApplyContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitExpression_group(ctx *Expression_groupContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitPrimitives(ctx *PrimitivesContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitLeft_name(ctx *Left_nameContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitAt(ctx *AtContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitNegate(ctx *NegateContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitMap_literal(ctx *Map_literalContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitIfthenelse(ctx *IfthenelseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitGet_name(ctx *Get_nameContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitObject_literal(ctx *Object_literalContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitArray_literal(ctx *Array_literalContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitStruct_literal(ctx *Struct_literalContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitVersion(ctx *VersionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitImport_alias(ctx *Import_aliasContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitImport_as(ctx *Import_asContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitImport_doc(ctx *Import_docContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitStruct(ctx *StructContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitMeta_value(ctx *Meta_valueContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitMeta_string_part(ctx *Meta_string_partContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitMeta_string(ctx *Meta_stringContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitMeta_array(ctx *Meta_arrayContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitMeta_object(ctx *Meta_objectContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitMeta_object_kv(ctx *Meta_object_kvContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitMeta_kv(ctx *Meta_kvContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitParameter_meta(ctx *Parameter_metaContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitMeta(ctx *MetaContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitTask_runtime_kv(ctx *Task_runtime_kvContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitTask_runtime(ctx *Task_runtimeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitTask_input(ctx *Task_inputContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitTask_output(ctx *Task_outputContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitTask_command_string_part(ctx *Task_command_string_partContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitTask_command_expr_part(ctx *Task_command_expr_partContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitTask_command_expr_with_string(ctx *Task_command_expr_with_stringContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitTask_command(ctx *Task_commandContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitTask_element(ctx *Task_elementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitTask(ctx *TaskContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitInner_workflow_element(ctx *Inner_workflow_elementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitCall_alias(ctx *Call_aliasContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitCall_input(ctx *Call_inputContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitCall_inputs(ctx *Call_inputsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitCall_body(ctx *Call_bodyContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitCall_after(ctx *Call_afterContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitCall_name(ctx *Call_nameContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitCall(ctx *CallContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitScatter(ctx *ScatterContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitConditional(ctx *ConditionalContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitWorkflow_input(ctx *Workflow_inputContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitWorkflow_output(ctx *Workflow_outputContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitInput(ctx *InputContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitOutput(ctx *OutputContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitInner_element(ctx *Inner_elementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitParameter_meta_element(ctx *Parameter_meta_elementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitMeta_element(ctx *Meta_elementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitWorkflow(ctx *WorkflowContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitDocument_element(ctx *Document_elementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseWdlV1_1ParserVisitor) VisitDocument(ctx *DocumentContext) interface{} {
	return v.VisitChildren(ctx)
}
