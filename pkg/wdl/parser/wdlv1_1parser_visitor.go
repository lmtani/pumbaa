// Code generated from WdlV1_1Parser.g4 by ANTLR 4.13.1. DO NOT EDIT.

package parser // WdlV1_1Parser

import "github.com/antlr4-go/antlr/v4"

// A complete Visitor for a parse tree produced by WdlV1_1Parser.
type WdlV1_1ParserVisitor interface {
	antlr.ParseTreeVisitor

	// Visit a parse tree produced by WdlV1_1Parser#map_type.
	VisitMap_type(ctx *Map_typeContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#array_type.
	VisitArray_type(ctx *Array_typeContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#pair_type.
	VisitPair_type(ctx *Pair_typeContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#type_base.
	VisitType_base(ctx *Type_baseContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#wdl_type.
	VisitWdl_type(ctx *Wdl_typeContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#unbound_decls.
	VisitUnbound_decls(ctx *Unbound_declsContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#bound_decls.
	VisitBound_decls(ctx *Bound_declsContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#any_decls.
	VisitAny_decls(ctx *Any_declsContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#number.
	VisitNumber(ctx *NumberContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#expression_placeholder_option.
	VisitExpression_placeholder_option(ctx *Expression_placeholder_optionContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#string_part.
	VisitString_part(ctx *String_partContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#string_expr_part.
	VisitString_expr_part(ctx *String_expr_partContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#string_expr_with_string_part.
	VisitString_expr_with_string_part(ctx *String_expr_with_string_partContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#string.
	VisitString(ctx *StringContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#primitive_literal.
	VisitPrimitive_literal(ctx *Primitive_literalContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#expr.
	VisitExpr(ctx *ExprContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#infix0.
	VisitInfix0(ctx *Infix0Context) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#infix1.
	VisitInfix1(ctx *Infix1Context) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#lor.
	VisitLor(ctx *LorContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#infix2.
	VisitInfix2(ctx *Infix2Context) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#land.
	VisitLand(ctx *LandContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#eqeq.
	VisitEqeq(ctx *EqeqContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#lt.
	VisitLt(ctx *LtContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#infix3.
	VisitInfix3(ctx *Infix3Context) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#gte.
	VisitGte(ctx *GteContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#neq.
	VisitNeq(ctx *NeqContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#lte.
	VisitLte(ctx *LteContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#gt.
	VisitGt(ctx *GtContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#add.
	VisitAdd(ctx *AddContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#sub.
	VisitSub(ctx *SubContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#infix4.
	VisitInfix4(ctx *Infix4Context) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#mod.
	VisitMod(ctx *ModContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#mul.
	VisitMul(ctx *MulContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#divide.
	VisitDivide(ctx *DivideContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#infix5.
	VisitInfix5(ctx *Infix5Context) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#expr_infix5.
	VisitExpr_infix5(ctx *Expr_infix5Context) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#member.
	VisitMember(ctx *MemberContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#pair_literal.
	VisitPair_literal(ctx *Pair_literalContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#unarysigned.
	VisitUnarysigned(ctx *UnarysignedContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#apply.
	VisitApply(ctx *ApplyContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#expression_group.
	VisitExpression_group(ctx *Expression_groupContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#primitives.
	VisitPrimitives(ctx *PrimitivesContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#left_name.
	VisitLeft_name(ctx *Left_nameContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#at.
	VisitAt(ctx *AtContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#negate.
	VisitNegate(ctx *NegateContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#map_literal.
	VisitMap_literal(ctx *Map_literalContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#ifthenelse.
	VisitIfthenelse(ctx *IfthenelseContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#get_name.
	VisitGet_name(ctx *Get_nameContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#object_literal.
	VisitObject_literal(ctx *Object_literalContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#array_literal.
	VisitArray_literal(ctx *Array_literalContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#struct_literal.
	VisitStruct_literal(ctx *Struct_literalContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#version.
	VisitVersion(ctx *VersionContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#import_alias.
	VisitImport_alias(ctx *Import_aliasContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#import_as.
	VisitImport_as(ctx *Import_asContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#import_doc.
	VisitImport_doc(ctx *Import_docContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#struct.
	VisitStruct(ctx *StructContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#meta_value.
	VisitMeta_value(ctx *Meta_valueContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#meta_string_part.
	VisitMeta_string_part(ctx *Meta_string_partContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#meta_string.
	VisitMeta_string(ctx *Meta_stringContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#meta_array.
	VisitMeta_array(ctx *Meta_arrayContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#meta_object.
	VisitMeta_object(ctx *Meta_objectContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#meta_object_kv.
	VisitMeta_object_kv(ctx *Meta_object_kvContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#meta_kv.
	VisitMeta_kv(ctx *Meta_kvContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#parameter_meta.
	VisitParameter_meta(ctx *Parameter_metaContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#meta.
	VisitMeta(ctx *MetaContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#task_runtime_kv.
	VisitTask_runtime_kv(ctx *Task_runtime_kvContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#task_runtime.
	VisitTask_runtime(ctx *Task_runtimeContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#task_input.
	VisitTask_input(ctx *Task_inputContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#task_output.
	VisitTask_output(ctx *Task_outputContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#task_command_string_part.
	VisitTask_command_string_part(ctx *Task_command_string_partContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#task_command_expr_part.
	VisitTask_command_expr_part(ctx *Task_command_expr_partContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#task_command_expr_with_string.
	VisitTask_command_expr_with_string(ctx *Task_command_expr_with_stringContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#task_command.
	VisitTask_command(ctx *Task_commandContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#task_element.
	VisitTask_element(ctx *Task_elementContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#task.
	VisitTask(ctx *TaskContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#inner_workflow_element.
	VisitInner_workflow_element(ctx *Inner_workflow_elementContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#call_alias.
	VisitCall_alias(ctx *Call_aliasContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#call_input.
	VisitCall_input(ctx *Call_inputContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#call_inputs.
	VisitCall_inputs(ctx *Call_inputsContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#call_body.
	VisitCall_body(ctx *Call_bodyContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#call_after.
	VisitCall_after(ctx *Call_afterContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#call_name.
	VisitCall_name(ctx *Call_nameContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#call.
	VisitCall(ctx *CallContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#scatter.
	VisitScatter(ctx *ScatterContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#conditional.
	VisitConditional(ctx *ConditionalContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#workflow_input.
	VisitWorkflow_input(ctx *Workflow_inputContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#workflow_output.
	VisitWorkflow_output(ctx *Workflow_outputContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#input.
	VisitInput(ctx *InputContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#output.
	VisitOutput(ctx *OutputContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#inner_element.
	VisitInner_element(ctx *Inner_elementContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#parameter_meta_element.
	VisitParameter_meta_element(ctx *Parameter_meta_elementContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#meta_element.
	VisitMeta_element(ctx *Meta_elementContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#workflow.
	VisitWorkflow(ctx *WorkflowContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#document_element.
	VisitDocument_element(ctx *Document_elementContext) interface{}

	// Visit a parse tree produced by WdlV1_1Parser#document.
	VisitDocument(ctx *DocumentContext) interface{}
}
