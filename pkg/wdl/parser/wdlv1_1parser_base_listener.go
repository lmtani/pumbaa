// Code generated from WdlV1_1Parser.g4 by ANTLR 4.13.1. DO NOT EDIT.

package parser // WdlV1_1Parser
import "github.com/antlr4-go/antlr/v4"

// BaseWdlV1_1ParserListener is a complete listener for a parse tree produced by WdlV1_1Parser.
type BaseWdlV1_1ParserListener struct{}

var _ WdlV1_1ParserListener = &BaseWdlV1_1ParserListener{}

// VisitTerminal is called when a terminal node is visited.
func (s *BaseWdlV1_1ParserListener) VisitTerminal(node antlr.TerminalNode) {}

// VisitErrorNode is called when an error node is visited.
func (s *BaseWdlV1_1ParserListener) VisitErrorNode(node antlr.ErrorNode) {}

// EnterEveryRule is called when any rule is entered.
func (s *BaseWdlV1_1ParserListener) EnterEveryRule(ctx antlr.ParserRuleContext) {}

// ExitEveryRule is called when any rule is exited.
func (s *BaseWdlV1_1ParserListener) ExitEveryRule(ctx antlr.ParserRuleContext) {}

// EnterMap_type is called when production map_type is entered.
func (s *BaseWdlV1_1ParserListener) EnterMap_type(ctx *Map_typeContext) {}

// ExitMap_type is called when production map_type is exited.
func (s *BaseWdlV1_1ParserListener) ExitMap_type(ctx *Map_typeContext) {}

// EnterArray_type is called when production array_type is entered.
func (s *BaseWdlV1_1ParserListener) EnterArray_type(ctx *Array_typeContext) {}

// ExitArray_type is called when production array_type is exited.
func (s *BaseWdlV1_1ParserListener) ExitArray_type(ctx *Array_typeContext) {}

// EnterPair_type is called when production pair_type is entered.
func (s *BaseWdlV1_1ParserListener) EnterPair_type(ctx *Pair_typeContext) {}

// ExitPair_type is called when production pair_type is exited.
func (s *BaseWdlV1_1ParserListener) ExitPair_type(ctx *Pair_typeContext) {}

// EnterType_base is called when production type_base is entered.
func (s *BaseWdlV1_1ParserListener) EnterType_base(ctx *Type_baseContext) {}

// ExitType_base is called when production type_base is exited.
func (s *BaseWdlV1_1ParserListener) ExitType_base(ctx *Type_baseContext) {}

// EnterWdl_type is called when production wdl_type is entered.
func (s *BaseWdlV1_1ParserListener) EnterWdl_type(ctx *Wdl_typeContext) {}

// ExitWdl_type is called when production wdl_type is exited.
func (s *BaseWdlV1_1ParserListener) ExitWdl_type(ctx *Wdl_typeContext) {}

// EnterUnbound_decls is called when production unbound_decls is entered.
func (s *BaseWdlV1_1ParserListener) EnterUnbound_decls(ctx *Unbound_declsContext) {}

// ExitUnbound_decls is called when production unbound_decls is exited.
func (s *BaseWdlV1_1ParserListener) ExitUnbound_decls(ctx *Unbound_declsContext) {}

// EnterBound_decls is called when production bound_decls is entered.
func (s *BaseWdlV1_1ParserListener) EnterBound_decls(ctx *Bound_declsContext) {}

// ExitBound_decls is called when production bound_decls is exited.
func (s *BaseWdlV1_1ParserListener) ExitBound_decls(ctx *Bound_declsContext) {}

// EnterAny_decls is called when production any_decls is entered.
func (s *BaseWdlV1_1ParserListener) EnterAny_decls(ctx *Any_declsContext) {}

// ExitAny_decls is called when production any_decls is exited.
func (s *BaseWdlV1_1ParserListener) ExitAny_decls(ctx *Any_declsContext) {}

// EnterNumber is called when production number is entered.
func (s *BaseWdlV1_1ParserListener) EnterNumber(ctx *NumberContext) {}

// ExitNumber is called when production number is exited.
func (s *BaseWdlV1_1ParserListener) ExitNumber(ctx *NumberContext) {}

// EnterExpression_placeholder_option is called when production expression_placeholder_option is entered.
func (s *BaseWdlV1_1ParserListener) EnterExpression_placeholder_option(ctx *Expression_placeholder_optionContext) {
}

// ExitExpression_placeholder_option is called when production expression_placeholder_option is exited.
func (s *BaseWdlV1_1ParserListener) ExitExpression_placeholder_option(ctx *Expression_placeholder_optionContext) {
}

// EnterString_part is called when production string_part is entered.
func (s *BaseWdlV1_1ParserListener) EnterString_part(ctx *String_partContext) {}

// ExitString_part is called when production string_part is exited.
func (s *BaseWdlV1_1ParserListener) ExitString_part(ctx *String_partContext) {}

// EnterString_expr_part is called when production string_expr_part is entered.
func (s *BaseWdlV1_1ParserListener) EnterString_expr_part(ctx *String_expr_partContext) {}

// ExitString_expr_part is called when production string_expr_part is exited.
func (s *BaseWdlV1_1ParserListener) ExitString_expr_part(ctx *String_expr_partContext) {}

// EnterString_expr_with_string_part is called when production string_expr_with_string_part is entered.
func (s *BaseWdlV1_1ParserListener) EnterString_expr_with_string_part(ctx *String_expr_with_string_partContext) {
}

// ExitString_expr_with_string_part is called when production string_expr_with_string_part is exited.
func (s *BaseWdlV1_1ParserListener) ExitString_expr_with_string_part(ctx *String_expr_with_string_partContext) {
}

// EnterString is called when production string is entered.
func (s *BaseWdlV1_1ParserListener) EnterString(ctx *StringContext) {}

// ExitString is called when production string is exited.
func (s *BaseWdlV1_1ParserListener) ExitString(ctx *StringContext) {}

// EnterPrimitive_literal is called when production primitive_literal is entered.
func (s *BaseWdlV1_1ParserListener) EnterPrimitive_literal(ctx *Primitive_literalContext) {}

// ExitPrimitive_literal is called when production primitive_literal is exited.
func (s *BaseWdlV1_1ParserListener) ExitPrimitive_literal(ctx *Primitive_literalContext) {}

// EnterExpr is called when production expr is entered.
func (s *BaseWdlV1_1ParserListener) EnterExpr(ctx *ExprContext) {}

// ExitExpr is called when production expr is exited.
func (s *BaseWdlV1_1ParserListener) ExitExpr(ctx *ExprContext) {}

// EnterInfix0 is called when production infix0 is entered.
func (s *BaseWdlV1_1ParserListener) EnterInfix0(ctx *Infix0Context) {}

// ExitInfix0 is called when production infix0 is exited.
func (s *BaseWdlV1_1ParserListener) ExitInfix0(ctx *Infix0Context) {}

// EnterInfix1 is called when production infix1 is entered.
func (s *BaseWdlV1_1ParserListener) EnterInfix1(ctx *Infix1Context) {}

// ExitInfix1 is called when production infix1 is exited.
func (s *BaseWdlV1_1ParserListener) ExitInfix1(ctx *Infix1Context) {}

// EnterLor is called when production lor is entered.
func (s *BaseWdlV1_1ParserListener) EnterLor(ctx *LorContext) {}

// ExitLor is called when production lor is exited.
func (s *BaseWdlV1_1ParserListener) ExitLor(ctx *LorContext) {}

// EnterInfix2 is called when production infix2 is entered.
func (s *BaseWdlV1_1ParserListener) EnterInfix2(ctx *Infix2Context) {}

// ExitInfix2 is called when production infix2 is exited.
func (s *BaseWdlV1_1ParserListener) ExitInfix2(ctx *Infix2Context) {}

// EnterLand is called when production land is entered.
func (s *BaseWdlV1_1ParserListener) EnterLand(ctx *LandContext) {}

// ExitLand is called when production land is exited.
func (s *BaseWdlV1_1ParserListener) ExitLand(ctx *LandContext) {}

// EnterEqeq is called when production eqeq is entered.
func (s *BaseWdlV1_1ParserListener) EnterEqeq(ctx *EqeqContext) {}

// ExitEqeq is called when production eqeq is exited.
func (s *BaseWdlV1_1ParserListener) ExitEqeq(ctx *EqeqContext) {}

// EnterLt is called when production lt is entered.
func (s *BaseWdlV1_1ParserListener) EnterLt(ctx *LtContext) {}

// ExitLt is called when production lt is exited.
func (s *BaseWdlV1_1ParserListener) ExitLt(ctx *LtContext) {}

// EnterInfix3 is called when production infix3 is entered.
func (s *BaseWdlV1_1ParserListener) EnterInfix3(ctx *Infix3Context) {}

// ExitInfix3 is called when production infix3 is exited.
func (s *BaseWdlV1_1ParserListener) ExitInfix3(ctx *Infix3Context) {}

// EnterGte is called when production gte is entered.
func (s *BaseWdlV1_1ParserListener) EnterGte(ctx *GteContext) {}

// ExitGte is called when production gte is exited.
func (s *BaseWdlV1_1ParserListener) ExitGte(ctx *GteContext) {}

// EnterNeq is called when production neq is entered.
func (s *BaseWdlV1_1ParserListener) EnterNeq(ctx *NeqContext) {}

// ExitNeq is called when production neq is exited.
func (s *BaseWdlV1_1ParserListener) ExitNeq(ctx *NeqContext) {}

// EnterLte is called when production lte is entered.
func (s *BaseWdlV1_1ParserListener) EnterLte(ctx *LteContext) {}

// ExitLte is called when production lte is exited.
func (s *BaseWdlV1_1ParserListener) ExitLte(ctx *LteContext) {}

// EnterGt is called when production gt is entered.
func (s *BaseWdlV1_1ParserListener) EnterGt(ctx *GtContext) {}

// ExitGt is called when production gt is exited.
func (s *BaseWdlV1_1ParserListener) ExitGt(ctx *GtContext) {}

// EnterAdd is called when production add is entered.
func (s *BaseWdlV1_1ParserListener) EnterAdd(ctx *AddContext) {}

// ExitAdd is called when production add is exited.
func (s *BaseWdlV1_1ParserListener) ExitAdd(ctx *AddContext) {}

// EnterSub is called when production sub is entered.
func (s *BaseWdlV1_1ParserListener) EnterSub(ctx *SubContext) {}

// ExitSub is called when production sub is exited.
func (s *BaseWdlV1_1ParserListener) ExitSub(ctx *SubContext) {}

// EnterInfix4 is called when production infix4 is entered.
func (s *BaseWdlV1_1ParserListener) EnterInfix4(ctx *Infix4Context) {}

// ExitInfix4 is called when production infix4 is exited.
func (s *BaseWdlV1_1ParserListener) ExitInfix4(ctx *Infix4Context) {}

// EnterMod is called when production mod is entered.
func (s *BaseWdlV1_1ParserListener) EnterMod(ctx *ModContext) {}

// ExitMod is called when production mod is exited.
func (s *BaseWdlV1_1ParserListener) ExitMod(ctx *ModContext) {}

// EnterMul is called when production mul is entered.
func (s *BaseWdlV1_1ParserListener) EnterMul(ctx *MulContext) {}

// ExitMul is called when production mul is exited.
func (s *BaseWdlV1_1ParserListener) ExitMul(ctx *MulContext) {}

// EnterDivide is called when production divide is entered.
func (s *BaseWdlV1_1ParserListener) EnterDivide(ctx *DivideContext) {}

// ExitDivide is called when production divide is exited.
func (s *BaseWdlV1_1ParserListener) ExitDivide(ctx *DivideContext) {}

// EnterInfix5 is called when production infix5 is entered.
func (s *BaseWdlV1_1ParserListener) EnterInfix5(ctx *Infix5Context) {}

// ExitInfix5 is called when production infix5 is exited.
func (s *BaseWdlV1_1ParserListener) ExitInfix5(ctx *Infix5Context) {}

// EnterExpr_infix5 is called when production expr_infix5 is entered.
func (s *BaseWdlV1_1ParserListener) EnterExpr_infix5(ctx *Expr_infix5Context) {}

// ExitExpr_infix5 is called when production expr_infix5 is exited.
func (s *BaseWdlV1_1ParserListener) ExitExpr_infix5(ctx *Expr_infix5Context) {}

// EnterMember is called when production member is entered.
func (s *BaseWdlV1_1ParserListener) EnterMember(ctx *MemberContext) {}

// ExitMember is called when production member is exited.
func (s *BaseWdlV1_1ParserListener) ExitMember(ctx *MemberContext) {}

// EnterPair_literal is called when production pair_literal is entered.
func (s *BaseWdlV1_1ParserListener) EnterPair_literal(ctx *Pair_literalContext) {}

// ExitPair_literal is called when production pair_literal is exited.
func (s *BaseWdlV1_1ParserListener) ExitPair_literal(ctx *Pair_literalContext) {}

// EnterUnarysigned is called when production unarysigned is entered.
func (s *BaseWdlV1_1ParserListener) EnterUnarysigned(ctx *UnarysignedContext) {}

// ExitUnarysigned is called when production unarysigned is exited.
func (s *BaseWdlV1_1ParserListener) ExitUnarysigned(ctx *UnarysignedContext) {}

// EnterApply is called when production apply is entered.
func (s *BaseWdlV1_1ParserListener) EnterApply(ctx *ApplyContext) {}

// ExitApply is called when production apply is exited.
func (s *BaseWdlV1_1ParserListener) ExitApply(ctx *ApplyContext) {}

// EnterExpression_group is called when production expression_group is entered.
func (s *BaseWdlV1_1ParserListener) EnterExpression_group(ctx *Expression_groupContext) {}

// ExitExpression_group is called when production expression_group is exited.
func (s *BaseWdlV1_1ParserListener) ExitExpression_group(ctx *Expression_groupContext) {}

// EnterPrimitives is called when production primitives is entered.
func (s *BaseWdlV1_1ParserListener) EnterPrimitives(ctx *PrimitivesContext) {}

// ExitPrimitives is called when production primitives is exited.
func (s *BaseWdlV1_1ParserListener) ExitPrimitives(ctx *PrimitivesContext) {}

// EnterLeft_name is called when production left_name is entered.
func (s *BaseWdlV1_1ParserListener) EnterLeft_name(ctx *Left_nameContext) {}

// ExitLeft_name is called when production left_name is exited.
func (s *BaseWdlV1_1ParserListener) ExitLeft_name(ctx *Left_nameContext) {}

// EnterAt is called when production at is entered.
func (s *BaseWdlV1_1ParserListener) EnterAt(ctx *AtContext) {}

// ExitAt is called when production at is exited.
func (s *BaseWdlV1_1ParserListener) ExitAt(ctx *AtContext) {}

// EnterNegate is called when production negate is entered.
func (s *BaseWdlV1_1ParserListener) EnterNegate(ctx *NegateContext) {}

// ExitNegate is called when production negate is exited.
func (s *BaseWdlV1_1ParserListener) ExitNegate(ctx *NegateContext) {}

// EnterMap_literal is called when production map_literal is entered.
func (s *BaseWdlV1_1ParserListener) EnterMap_literal(ctx *Map_literalContext) {}

// ExitMap_literal is called when production map_literal is exited.
func (s *BaseWdlV1_1ParserListener) ExitMap_literal(ctx *Map_literalContext) {}

// EnterIfthenelse is called when production ifthenelse is entered.
func (s *BaseWdlV1_1ParserListener) EnterIfthenelse(ctx *IfthenelseContext) {}

// ExitIfthenelse is called when production ifthenelse is exited.
func (s *BaseWdlV1_1ParserListener) ExitIfthenelse(ctx *IfthenelseContext) {}

// EnterGet_name is called when production get_name is entered.
func (s *BaseWdlV1_1ParserListener) EnterGet_name(ctx *Get_nameContext) {}

// ExitGet_name is called when production get_name is exited.
func (s *BaseWdlV1_1ParserListener) ExitGet_name(ctx *Get_nameContext) {}

// EnterObject_literal is called when production object_literal is entered.
func (s *BaseWdlV1_1ParserListener) EnterObject_literal(ctx *Object_literalContext) {}

// ExitObject_literal is called when production object_literal is exited.
func (s *BaseWdlV1_1ParserListener) ExitObject_literal(ctx *Object_literalContext) {}

// EnterArray_literal is called when production array_literal is entered.
func (s *BaseWdlV1_1ParserListener) EnterArray_literal(ctx *Array_literalContext) {}

// ExitArray_literal is called when production array_literal is exited.
func (s *BaseWdlV1_1ParserListener) ExitArray_literal(ctx *Array_literalContext) {}

// EnterStruct_literal is called when production struct_literal is entered.
func (s *BaseWdlV1_1ParserListener) EnterStruct_literal(ctx *Struct_literalContext) {}

// ExitStruct_literal is called when production struct_literal is exited.
func (s *BaseWdlV1_1ParserListener) ExitStruct_literal(ctx *Struct_literalContext) {}

// EnterVersion is called when production version is entered.
func (s *BaseWdlV1_1ParserListener) EnterVersion(ctx *VersionContext) {}

// ExitVersion is called when production version is exited.
func (s *BaseWdlV1_1ParserListener) ExitVersion(ctx *VersionContext) {}

// EnterImport_alias is called when production import_alias is entered.
func (s *BaseWdlV1_1ParserListener) EnterImport_alias(ctx *Import_aliasContext) {}

// ExitImport_alias is called when production import_alias is exited.
func (s *BaseWdlV1_1ParserListener) ExitImport_alias(ctx *Import_aliasContext) {}

// EnterImport_as is called when production import_as is entered.
func (s *BaseWdlV1_1ParserListener) EnterImport_as(ctx *Import_asContext) {}

// ExitImport_as is called when production import_as is exited.
func (s *BaseWdlV1_1ParserListener) ExitImport_as(ctx *Import_asContext) {}

// EnterImport_doc is called when production import_doc is entered.
func (s *BaseWdlV1_1ParserListener) EnterImport_doc(ctx *Import_docContext) {}

// ExitImport_doc is called when production import_doc is exited.
func (s *BaseWdlV1_1ParserListener) ExitImport_doc(ctx *Import_docContext) {}

// EnterStruct is called when production struct is entered.
func (s *BaseWdlV1_1ParserListener) EnterStruct(ctx *StructContext) {}

// ExitStruct is called when production struct is exited.
func (s *BaseWdlV1_1ParserListener) ExitStruct(ctx *StructContext) {}

// EnterMeta_value is called when production meta_value is entered.
func (s *BaseWdlV1_1ParserListener) EnterMeta_value(ctx *Meta_valueContext) {}

// ExitMeta_value is called when production meta_value is exited.
func (s *BaseWdlV1_1ParserListener) ExitMeta_value(ctx *Meta_valueContext) {}

// EnterMeta_string_part is called when production meta_string_part is entered.
func (s *BaseWdlV1_1ParserListener) EnterMeta_string_part(ctx *Meta_string_partContext) {}

// ExitMeta_string_part is called when production meta_string_part is exited.
func (s *BaseWdlV1_1ParserListener) ExitMeta_string_part(ctx *Meta_string_partContext) {}

// EnterMeta_string is called when production meta_string is entered.
func (s *BaseWdlV1_1ParserListener) EnterMeta_string(ctx *Meta_stringContext) {}

// ExitMeta_string is called when production meta_string is exited.
func (s *BaseWdlV1_1ParserListener) ExitMeta_string(ctx *Meta_stringContext) {}

// EnterMeta_array is called when production meta_array is entered.
func (s *BaseWdlV1_1ParserListener) EnterMeta_array(ctx *Meta_arrayContext) {}

// ExitMeta_array is called when production meta_array is exited.
func (s *BaseWdlV1_1ParserListener) ExitMeta_array(ctx *Meta_arrayContext) {}

// EnterMeta_object is called when production meta_object is entered.
func (s *BaseWdlV1_1ParserListener) EnterMeta_object(ctx *Meta_objectContext) {}

// ExitMeta_object is called when production meta_object is exited.
func (s *BaseWdlV1_1ParserListener) ExitMeta_object(ctx *Meta_objectContext) {}

// EnterMeta_object_kv is called when production meta_object_kv is entered.
func (s *BaseWdlV1_1ParserListener) EnterMeta_object_kv(ctx *Meta_object_kvContext) {}

// ExitMeta_object_kv is called when production meta_object_kv is exited.
func (s *BaseWdlV1_1ParserListener) ExitMeta_object_kv(ctx *Meta_object_kvContext) {}

// EnterMeta_kv is called when production meta_kv is entered.
func (s *BaseWdlV1_1ParserListener) EnterMeta_kv(ctx *Meta_kvContext) {}

// ExitMeta_kv is called when production meta_kv is exited.
func (s *BaseWdlV1_1ParserListener) ExitMeta_kv(ctx *Meta_kvContext) {}

// EnterParameter_meta is called when production parameter_meta is entered.
func (s *BaseWdlV1_1ParserListener) EnterParameter_meta(ctx *Parameter_metaContext) {}

// ExitParameter_meta is called when production parameter_meta is exited.
func (s *BaseWdlV1_1ParserListener) ExitParameter_meta(ctx *Parameter_metaContext) {}

// EnterMeta is called when production meta is entered.
func (s *BaseWdlV1_1ParserListener) EnterMeta(ctx *MetaContext) {}

// ExitMeta is called when production meta is exited.
func (s *BaseWdlV1_1ParserListener) ExitMeta(ctx *MetaContext) {}

// EnterTask_runtime_kv is called when production task_runtime_kv is entered.
func (s *BaseWdlV1_1ParserListener) EnterTask_runtime_kv(ctx *Task_runtime_kvContext) {}

// ExitTask_runtime_kv is called when production task_runtime_kv is exited.
func (s *BaseWdlV1_1ParserListener) ExitTask_runtime_kv(ctx *Task_runtime_kvContext) {}

// EnterTask_runtime is called when production task_runtime is entered.
func (s *BaseWdlV1_1ParserListener) EnterTask_runtime(ctx *Task_runtimeContext) {}

// ExitTask_runtime is called when production task_runtime is exited.
func (s *BaseWdlV1_1ParserListener) ExitTask_runtime(ctx *Task_runtimeContext) {}

// EnterTask_input is called when production task_input is entered.
func (s *BaseWdlV1_1ParserListener) EnterTask_input(ctx *Task_inputContext) {}

// ExitTask_input is called when production task_input is exited.
func (s *BaseWdlV1_1ParserListener) ExitTask_input(ctx *Task_inputContext) {}

// EnterTask_output is called when production task_output is entered.
func (s *BaseWdlV1_1ParserListener) EnterTask_output(ctx *Task_outputContext) {}

// ExitTask_output is called when production task_output is exited.
func (s *BaseWdlV1_1ParserListener) ExitTask_output(ctx *Task_outputContext) {}

// EnterTask_command_string_part is called when production task_command_string_part is entered.
func (s *BaseWdlV1_1ParserListener) EnterTask_command_string_part(ctx *Task_command_string_partContext) {
}

// ExitTask_command_string_part is called when production task_command_string_part is exited.
func (s *BaseWdlV1_1ParserListener) ExitTask_command_string_part(ctx *Task_command_string_partContext) {
}

// EnterTask_command_expr_part is called when production task_command_expr_part is entered.
func (s *BaseWdlV1_1ParserListener) EnterTask_command_expr_part(ctx *Task_command_expr_partContext) {}

// ExitTask_command_expr_part is called when production task_command_expr_part is exited.
func (s *BaseWdlV1_1ParserListener) ExitTask_command_expr_part(ctx *Task_command_expr_partContext) {}

// EnterTask_command_expr_with_string is called when production task_command_expr_with_string is entered.
func (s *BaseWdlV1_1ParserListener) EnterTask_command_expr_with_string(ctx *Task_command_expr_with_stringContext) {
}

// ExitTask_command_expr_with_string is called when production task_command_expr_with_string is exited.
func (s *BaseWdlV1_1ParserListener) ExitTask_command_expr_with_string(ctx *Task_command_expr_with_stringContext) {
}

// EnterTask_command is called when production task_command is entered.
func (s *BaseWdlV1_1ParserListener) EnterTask_command(ctx *Task_commandContext) {}

// ExitTask_command is called when production task_command is exited.
func (s *BaseWdlV1_1ParserListener) ExitTask_command(ctx *Task_commandContext) {}

// EnterTask_element is called when production task_element is entered.
func (s *BaseWdlV1_1ParserListener) EnterTask_element(ctx *Task_elementContext) {}

// ExitTask_element is called when production task_element is exited.
func (s *BaseWdlV1_1ParserListener) ExitTask_element(ctx *Task_elementContext) {}

// EnterTask is called when production task is entered.
func (s *BaseWdlV1_1ParserListener) EnterTask(ctx *TaskContext) {}

// ExitTask is called when production task is exited.
func (s *BaseWdlV1_1ParserListener) ExitTask(ctx *TaskContext) {}

// EnterInner_workflow_element is called when production inner_workflow_element is entered.
func (s *BaseWdlV1_1ParserListener) EnterInner_workflow_element(ctx *Inner_workflow_elementContext) {}

// ExitInner_workflow_element is called when production inner_workflow_element is exited.
func (s *BaseWdlV1_1ParserListener) ExitInner_workflow_element(ctx *Inner_workflow_elementContext) {}

// EnterCall_alias is called when production call_alias is entered.
func (s *BaseWdlV1_1ParserListener) EnterCall_alias(ctx *Call_aliasContext) {}

// ExitCall_alias is called when production call_alias is exited.
func (s *BaseWdlV1_1ParserListener) ExitCall_alias(ctx *Call_aliasContext) {}

// EnterCall_input is called when production call_input is entered.
func (s *BaseWdlV1_1ParserListener) EnterCall_input(ctx *Call_inputContext) {}

// ExitCall_input is called when production call_input is exited.
func (s *BaseWdlV1_1ParserListener) ExitCall_input(ctx *Call_inputContext) {}

// EnterCall_inputs is called when production call_inputs is entered.
func (s *BaseWdlV1_1ParserListener) EnterCall_inputs(ctx *Call_inputsContext) {}

// ExitCall_inputs is called when production call_inputs is exited.
func (s *BaseWdlV1_1ParserListener) ExitCall_inputs(ctx *Call_inputsContext) {}

// EnterCall_body is called when production call_body is entered.
func (s *BaseWdlV1_1ParserListener) EnterCall_body(ctx *Call_bodyContext) {}

// ExitCall_body is called when production call_body is exited.
func (s *BaseWdlV1_1ParserListener) ExitCall_body(ctx *Call_bodyContext) {}

// EnterCall_after is called when production call_after is entered.
func (s *BaseWdlV1_1ParserListener) EnterCall_after(ctx *Call_afterContext) {}

// ExitCall_after is called when production call_after is exited.
func (s *BaseWdlV1_1ParserListener) ExitCall_after(ctx *Call_afterContext) {}

// EnterCall_name is called when production call_name is entered.
func (s *BaseWdlV1_1ParserListener) EnterCall_name(ctx *Call_nameContext) {}

// ExitCall_name is called when production call_name is exited.
func (s *BaseWdlV1_1ParserListener) ExitCall_name(ctx *Call_nameContext) {}

// EnterCall is called when production call is entered.
func (s *BaseWdlV1_1ParserListener) EnterCall(ctx *CallContext) {}

// ExitCall is called when production call is exited.
func (s *BaseWdlV1_1ParserListener) ExitCall(ctx *CallContext) {}

// EnterScatter is called when production scatter is entered.
func (s *BaseWdlV1_1ParserListener) EnterScatter(ctx *ScatterContext) {}

// ExitScatter is called when production scatter is exited.
func (s *BaseWdlV1_1ParserListener) ExitScatter(ctx *ScatterContext) {}

// EnterConditional is called when production conditional is entered.
func (s *BaseWdlV1_1ParserListener) EnterConditional(ctx *ConditionalContext) {}

// ExitConditional is called when production conditional is exited.
func (s *BaseWdlV1_1ParserListener) ExitConditional(ctx *ConditionalContext) {}

// EnterWorkflow_input is called when production workflow_input is entered.
func (s *BaseWdlV1_1ParserListener) EnterWorkflow_input(ctx *Workflow_inputContext) {}

// ExitWorkflow_input is called when production workflow_input is exited.
func (s *BaseWdlV1_1ParserListener) ExitWorkflow_input(ctx *Workflow_inputContext) {}

// EnterWorkflow_output is called when production workflow_output is entered.
func (s *BaseWdlV1_1ParserListener) EnterWorkflow_output(ctx *Workflow_outputContext) {}

// ExitWorkflow_output is called when production workflow_output is exited.
func (s *BaseWdlV1_1ParserListener) ExitWorkflow_output(ctx *Workflow_outputContext) {}

// EnterInput is called when production input is entered.
func (s *BaseWdlV1_1ParserListener) EnterInput(ctx *InputContext) {}

// ExitInput is called when production input is exited.
func (s *BaseWdlV1_1ParserListener) ExitInput(ctx *InputContext) {}

// EnterOutput is called when production output is entered.
func (s *BaseWdlV1_1ParserListener) EnterOutput(ctx *OutputContext) {}

// ExitOutput is called when production output is exited.
func (s *BaseWdlV1_1ParserListener) ExitOutput(ctx *OutputContext) {}

// EnterInner_element is called when production inner_element is entered.
func (s *BaseWdlV1_1ParserListener) EnterInner_element(ctx *Inner_elementContext) {}

// ExitInner_element is called when production inner_element is exited.
func (s *BaseWdlV1_1ParserListener) ExitInner_element(ctx *Inner_elementContext) {}

// EnterParameter_meta_element is called when production parameter_meta_element is entered.
func (s *BaseWdlV1_1ParserListener) EnterParameter_meta_element(ctx *Parameter_meta_elementContext) {}

// ExitParameter_meta_element is called when production parameter_meta_element is exited.
func (s *BaseWdlV1_1ParserListener) ExitParameter_meta_element(ctx *Parameter_meta_elementContext) {}

// EnterMeta_element is called when production meta_element is entered.
func (s *BaseWdlV1_1ParserListener) EnterMeta_element(ctx *Meta_elementContext) {}

// ExitMeta_element is called when production meta_element is exited.
func (s *BaseWdlV1_1ParserListener) ExitMeta_element(ctx *Meta_elementContext) {}

// EnterWorkflow is called when production workflow is entered.
func (s *BaseWdlV1_1ParserListener) EnterWorkflow(ctx *WorkflowContext) {}

// ExitWorkflow is called when production workflow is exited.
func (s *BaseWdlV1_1ParserListener) ExitWorkflow(ctx *WorkflowContext) {}

// EnterDocument_element is called when production document_element is entered.
func (s *BaseWdlV1_1ParserListener) EnterDocument_element(ctx *Document_elementContext) {}

// ExitDocument_element is called when production document_element is exited.
func (s *BaseWdlV1_1ParserListener) ExitDocument_element(ctx *Document_elementContext) {}

// EnterDocument is called when production document is entered.
func (s *BaseWdlV1_1ParserListener) EnterDocument(ctx *DocumentContext) {}

// ExitDocument is called when production document is exited.
func (s *BaseWdlV1_1ParserListener) ExitDocument(ctx *DocumentContext) {}
