// Code generated from WdlV1_1Parser.g4 by ANTLR 4.13.1. DO NOT EDIT.

package parser // WdlV1_1Parser
import "github.com/antlr4-go/antlr/v4"

// WdlV1_1ParserListener is a complete listener for a parse tree produced by WdlV1_1Parser.
type WdlV1_1ParserListener interface {
	antlr.ParseTreeListener

	// EnterMap_type is called when entering the map_type production.
	EnterMap_type(c *Map_typeContext)

	// EnterArray_type is called when entering the array_type production.
	EnterArray_type(c *Array_typeContext)

	// EnterPair_type is called when entering the pair_type production.
	EnterPair_type(c *Pair_typeContext)

	// EnterType_base is called when entering the type_base production.
	EnterType_base(c *Type_baseContext)

	// EnterWdl_type is called when entering the wdl_type production.
	EnterWdl_type(c *Wdl_typeContext)

	// EnterUnbound_decls is called when entering the unbound_decls production.
	EnterUnbound_decls(c *Unbound_declsContext)

	// EnterBound_decls is called when entering the bound_decls production.
	EnterBound_decls(c *Bound_declsContext)

	// EnterAny_decls is called when entering the any_decls production.
	EnterAny_decls(c *Any_declsContext)

	// EnterNumber is called when entering the number production.
	EnterNumber(c *NumberContext)

	// EnterExpression_placeholder_option is called when entering the expression_placeholder_option production.
	EnterExpression_placeholder_option(c *Expression_placeholder_optionContext)

	// EnterString_part is called when entering the string_part production.
	EnterString_part(c *String_partContext)

	// EnterString_expr_part is called when entering the string_expr_part production.
	EnterString_expr_part(c *String_expr_partContext)

	// EnterString_expr_with_string_part is called when entering the string_expr_with_string_part production.
	EnterString_expr_with_string_part(c *String_expr_with_string_partContext)

	// EnterString is called when entering the string production.
	EnterString(c *StringContext)

	// EnterPrimitive_literal is called when entering the primitive_literal production.
	EnterPrimitive_literal(c *Primitive_literalContext)

	// EnterExpr is called when entering the expr production.
	EnterExpr(c *ExprContext)

	// EnterInfix0 is called when entering the infix0 production.
	EnterInfix0(c *Infix0Context)

	// EnterInfix1 is called when entering the infix1 production.
	EnterInfix1(c *Infix1Context)

	// EnterLor is called when entering the lor production.
	EnterLor(c *LorContext)

	// EnterInfix2 is called when entering the infix2 production.
	EnterInfix2(c *Infix2Context)

	// EnterLand is called when entering the land production.
	EnterLand(c *LandContext)

	// EnterEqeq is called when entering the eqeq production.
	EnterEqeq(c *EqeqContext)

	// EnterLt is called when entering the lt production.
	EnterLt(c *LtContext)

	// EnterInfix3 is called when entering the infix3 production.
	EnterInfix3(c *Infix3Context)

	// EnterGte is called when entering the gte production.
	EnterGte(c *GteContext)

	// EnterNeq is called when entering the neq production.
	EnterNeq(c *NeqContext)

	// EnterLte is called when entering the lte production.
	EnterLte(c *LteContext)

	// EnterGt is called when entering the gt production.
	EnterGt(c *GtContext)

	// EnterAdd is called when entering the add production.
	EnterAdd(c *AddContext)

	// EnterSub is called when entering the sub production.
	EnterSub(c *SubContext)

	// EnterInfix4 is called when entering the infix4 production.
	EnterInfix4(c *Infix4Context)

	// EnterMod is called when entering the mod production.
	EnterMod(c *ModContext)

	// EnterMul is called when entering the mul production.
	EnterMul(c *MulContext)

	// EnterDivide is called when entering the divide production.
	EnterDivide(c *DivideContext)

	// EnterInfix5 is called when entering the infix5 production.
	EnterInfix5(c *Infix5Context)

	// EnterExpr_infix5 is called when entering the expr_infix5 production.
	EnterExpr_infix5(c *Expr_infix5Context)

	// EnterMember is called when entering the member production.
	EnterMember(c *MemberContext)

	// EnterPair_literal is called when entering the pair_literal production.
	EnterPair_literal(c *Pair_literalContext)

	// EnterUnarysigned is called when entering the unarysigned production.
	EnterUnarysigned(c *UnarysignedContext)

	// EnterApply is called when entering the apply production.
	EnterApply(c *ApplyContext)

	// EnterExpression_group is called when entering the expression_group production.
	EnterExpression_group(c *Expression_groupContext)

	// EnterPrimitives is called when entering the primitives production.
	EnterPrimitives(c *PrimitivesContext)

	// EnterLeft_name is called when entering the left_name production.
	EnterLeft_name(c *Left_nameContext)

	// EnterAt is called when entering the at production.
	EnterAt(c *AtContext)

	// EnterNegate is called when entering the negate production.
	EnterNegate(c *NegateContext)

	// EnterMap_literal is called when entering the map_literal production.
	EnterMap_literal(c *Map_literalContext)

	// EnterIfthenelse is called when entering the ifthenelse production.
	EnterIfthenelse(c *IfthenelseContext)

	// EnterGet_name is called when entering the get_name production.
	EnterGet_name(c *Get_nameContext)

	// EnterObject_literal is called when entering the object_literal production.
	EnterObject_literal(c *Object_literalContext)

	// EnterArray_literal is called when entering the array_literal production.
	EnterArray_literal(c *Array_literalContext)

	// EnterStruct_literal is called when entering the struct_literal production.
	EnterStruct_literal(c *Struct_literalContext)

	// EnterVersion is called when entering the version production.
	EnterVersion(c *VersionContext)

	// EnterImport_alias is called when entering the import_alias production.
	EnterImport_alias(c *Import_aliasContext)

	// EnterImport_as is called when entering the import_as production.
	EnterImport_as(c *Import_asContext)

	// EnterImport_doc is called when entering the import_doc production.
	EnterImport_doc(c *Import_docContext)

	// EnterStruct is called when entering the struct production.
	EnterStruct(c *StructContext)

	// EnterMeta_value is called when entering the meta_value production.
	EnterMeta_value(c *Meta_valueContext)

	// EnterMeta_string_part is called when entering the meta_string_part production.
	EnterMeta_string_part(c *Meta_string_partContext)

	// EnterMeta_string is called when entering the meta_string production.
	EnterMeta_string(c *Meta_stringContext)

	// EnterMeta_array is called when entering the meta_array production.
	EnterMeta_array(c *Meta_arrayContext)

	// EnterMeta_object is called when entering the meta_object production.
	EnterMeta_object(c *Meta_objectContext)

	// EnterMeta_object_kv is called when entering the meta_object_kv production.
	EnterMeta_object_kv(c *Meta_object_kvContext)

	// EnterMeta_kv is called when entering the meta_kv production.
	EnterMeta_kv(c *Meta_kvContext)

	// EnterParameter_meta is called when entering the parameter_meta production.
	EnterParameter_meta(c *Parameter_metaContext)

	// EnterMeta is called when entering the meta production.
	EnterMeta(c *MetaContext)

	// EnterTask_runtime_kv is called when entering the task_runtime_kv production.
	EnterTask_runtime_kv(c *Task_runtime_kvContext)

	// EnterTask_runtime is called when entering the task_runtime production.
	EnterTask_runtime(c *Task_runtimeContext)

	// EnterTask_input is called when entering the task_input production.
	EnterTask_input(c *Task_inputContext)

	// EnterTask_output is called when entering the task_output production.
	EnterTask_output(c *Task_outputContext)

	// EnterTask_command_string_part is called when entering the task_command_string_part production.
	EnterTask_command_string_part(c *Task_command_string_partContext)

	// EnterTask_command_expr_part is called when entering the task_command_expr_part production.
	EnterTask_command_expr_part(c *Task_command_expr_partContext)

	// EnterTask_command_expr_with_string is called when entering the task_command_expr_with_string production.
	EnterTask_command_expr_with_string(c *Task_command_expr_with_stringContext)

	// EnterTask_command is called when entering the task_command production.
	EnterTask_command(c *Task_commandContext)

	// EnterTask_element is called when entering the task_element production.
	EnterTask_element(c *Task_elementContext)

	// EnterTask is called when entering the task production.
	EnterTask(c *TaskContext)

	// EnterInner_workflow_element is called when entering the inner_workflow_element production.
	EnterInner_workflow_element(c *Inner_workflow_elementContext)

	// EnterCall_alias is called when entering the call_alias production.
	EnterCall_alias(c *Call_aliasContext)

	// EnterCall_input is called when entering the call_input production.
	EnterCall_input(c *Call_inputContext)

	// EnterCall_inputs is called when entering the call_inputs production.
	EnterCall_inputs(c *Call_inputsContext)

	// EnterCall_body is called when entering the call_body production.
	EnterCall_body(c *Call_bodyContext)

	// EnterCall_after is called when entering the call_after production.
	EnterCall_after(c *Call_afterContext)

	// EnterCall_name is called when entering the call_name production.
	EnterCall_name(c *Call_nameContext)

	// EnterCall is called when entering the call production.
	EnterCall(c *CallContext)

	// EnterScatter is called when entering the scatter production.
	EnterScatter(c *ScatterContext)

	// EnterConditional is called when entering the conditional production.
	EnterConditional(c *ConditionalContext)

	// EnterWorkflow_input is called when entering the workflow_input production.
	EnterWorkflow_input(c *Workflow_inputContext)

	// EnterWorkflow_output is called when entering the workflow_output production.
	EnterWorkflow_output(c *Workflow_outputContext)

	// EnterInput is called when entering the input production.
	EnterInput(c *InputContext)

	// EnterOutput is called when entering the output production.
	EnterOutput(c *OutputContext)

	// EnterInner_element is called when entering the inner_element production.
	EnterInner_element(c *Inner_elementContext)

	// EnterParameter_meta_element is called when entering the parameter_meta_element production.
	EnterParameter_meta_element(c *Parameter_meta_elementContext)

	// EnterMeta_element is called when entering the meta_element production.
	EnterMeta_element(c *Meta_elementContext)

	// EnterWorkflow is called when entering the workflow production.
	EnterWorkflow(c *WorkflowContext)

	// EnterDocument_element is called when entering the document_element production.
	EnterDocument_element(c *Document_elementContext)

	// EnterDocument is called when entering the document production.
	EnterDocument(c *DocumentContext)

	// ExitMap_type is called when exiting the map_type production.
	ExitMap_type(c *Map_typeContext)

	// ExitArray_type is called when exiting the array_type production.
	ExitArray_type(c *Array_typeContext)

	// ExitPair_type is called when exiting the pair_type production.
	ExitPair_type(c *Pair_typeContext)

	// ExitType_base is called when exiting the type_base production.
	ExitType_base(c *Type_baseContext)

	// ExitWdl_type is called when exiting the wdl_type production.
	ExitWdl_type(c *Wdl_typeContext)

	// ExitUnbound_decls is called when exiting the unbound_decls production.
	ExitUnbound_decls(c *Unbound_declsContext)

	// ExitBound_decls is called when exiting the bound_decls production.
	ExitBound_decls(c *Bound_declsContext)

	// ExitAny_decls is called when exiting the any_decls production.
	ExitAny_decls(c *Any_declsContext)

	// ExitNumber is called when exiting the number production.
	ExitNumber(c *NumberContext)

	// ExitExpression_placeholder_option is called when exiting the expression_placeholder_option production.
	ExitExpression_placeholder_option(c *Expression_placeholder_optionContext)

	// ExitString_part is called when exiting the string_part production.
	ExitString_part(c *String_partContext)

	// ExitString_expr_part is called when exiting the string_expr_part production.
	ExitString_expr_part(c *String_expr_partContext)

	// ExitString_expr_with_string_part is called when exiting the string_expr_with_string_part production.
	ExitString_expr_with_string_part(c *String_expr_with_string_partContext)

	// ExitString is called when exiting the string production.
	ExitString(c *StringContext)

	// ExitPrimitive_literal is called when exiting the primitive_literal production.
	ExitPrimitive_literal(c *Primitive_literalContext)

	// ExitExpr is called when exiting the expr production.
	ExitExpr(c *ExprContext)

	// ExitInfix0 is called when exiting the infix0 production.
	ExitInfix0(c *Infix0Context)

	// ExitInfix1 is called when exiting the infix1 production.
	ExitInfix1(c *Infix1Context)

	// ExitLor is called when exiting the lor production.
	ExitLor(c *LorContext)

	// ExitInfix2 is called when exiting the infix2 production.
	ExitInfix2(c *Infix2Context)

	// ExitLand is called when exiting the land production.
	ExitLand(c *LandContext)

	// ExitEqeq is called when exiting the eqeq production.
	ExitEqeq(c *EqeqContext)

	// ExitLt is called when exiting the lt production.
	ExitLt(c *LtContext)

	// ExitInfix3 is called when exiting the infix3 production.
	ExitInfix3(c *Infix3Context)

	// ExitGte is called when exiting the gte production.
	ExitGte(c *GteContext)

	// ExitNeq is called when exiting the neq production.
	ExitNeq(c *NeqContext)

	// ExitLte is called when exiting the lte production.
	ExitLte(c *LteContext)

	// ExitGt is called when exiting the gt production.
	ExitGt(c *GtContext)

	// ExitAdd is called when exiting the add production.
	ExitAdd(c *AddContext)

	// ExitSub is called when exiting the sub production.
	ExitSub(c *SubContext)

	// ExitInfix4 is called when exiting the infix4 production.
	ExitInfix4(c *Infix4Context)

	// ExitMod is called when exiting the mod production.
	ExitMod(c *ModContext)

	// ExitMul is called when exiting the mul production.
	ExitMul(c *MulContext)

	// ExitDivide is called when exiting the divide production.
	ExitDivide(c *DivideContext)

	// ExitInfix5 is called when exiting the infix5 production.
	ExitInfix5(c *Infix5Context)

	// ExitExpr_infix5 is called when exiting the expr_infix5 production.
	ExitExpr_infix5(c *Expr_infix5Context)

	// ExitMember is called when exiting the member production.
	ExitMember(c *MemberContext)

	// ExitPair_literal is called when exiting the pair_literal production.
	ExitPair_literal(c *Pair_literalContext)

	// ExitUnarysigned is called when exiting the unarysigned production.
	ExitUnarysigned(c *UnarysignedContext)

	// ExitApply is called when exiting the apply production.
	ExitApply(c *ApplyContext)

	// ExitExpression_group is called when exiting the expression_group production.
	ExitExpression_group(c *Expression_groupContext)

	// ExitPrimitives is called when exiting the primitives production.
	ExitPrimitives(c *PrimitivesContext)

	// ExitLeft_name is called when exiting the left_name production.
	ExitLeft_name(c *Left_nameContext)

	// ExitAt is called when exiting the at production.
	ExitAt(c *AtContext)

	// ExitNegate is called when exiting the negate production.
	ExitNegate(c *NegateContext)

	// ExitMap_literal is called when exiting the map_literal production.
	ExitMap_literal(c *Map_literalContext)

	// ExitIfthenelse is called when exiting the ifthenelse production.
	ExitIfthenelse(c *IfthenelseContext)

	// ExitGet_name is called when exiting the get_name production.
	ExitGet_name(c *Get_nameContext)

	// ExitObject_literal is called when exiting the object_literal production.
	ExitObject_literal(c *Object_literalContext)

	// ExitArray_literal is called when exiting the array_literal production.
	ExitArray_literal(c *Array_literalContext)

	// ExitStruct_literal is called when exiting the struct_literal production.
	ExitStruct_literal(c *Struct_literalContext)

	// ExitVersion is called when exiting the version production.
	ExitVersion(c *VersionContext)

	// ExitImport_alias is called when exiting the import_alias production.
	ExitImport_alias(c *Import_aliasContext)

	// ExitImport_as is called when exiting the import_as production.
	ExitImport_as(c *Import_asContext)

	// ExitImport_doc is called when exiting the import_doc production.
	ExitImport_doc(c *Import_docContext)

	// ExitStruct is called when exiting the struct production.
	ExitStruct(c *StructContext)

	// ExitMeta_value is called when exiting the meta_value production.
	ExitMeta_value(c *Meta_valueContext)

	// ExitMeta_string_part is called when exiting the meta_string_part production.
	ExitMeta_string_part(c *Meta_string_partContext)

	// ExitMeta_string is called when exiting the meta_string production.
	ExitMeta_string(c *Meta_stringContext)

	// ExitMeta_array is called when exiting the meta_array production.
	ExitMeta_array(c *Meta_arrayContext)

	// ExitMeta_object is called when exiting the meta_object production.
	ExitMeta_object(c *Meta_objectContext)

	// ExitMeta_object_kv is called when exiting the meta_object_kv production.
	ExitMeta_object_kv(c *Meta_object_kvContext)

	// ExitMeta_kv is called when exiting the meta_kv production.
	ExitMeta_kv(c *Meta_kvContext)

	// ExitParameter_meta is called when exiting the parameter_meta production.
	ExitParameter_meta(c *Parameter_metaContext)

	// ExitMeta is called when exiting the meta production.
	ExitMeta(c *MetaContext)

	// ExitTask_runtime_kv is called when exiting the task_runtime_kv production.
	ExitTask_runtime_kv(c *Task_runtime_kvContext)

	// ExitTask_runtime is called when exiting the task_runtime production.
	ExitTask_runtime(c *Task_runtimeContext)

	// ExitTask_input is called when exiting the task_input production.
	ExitTask_input(c *Task_inputContext)

	// ExitTask_output is called when exiting the task_output production.
	ExitTask_output(c *Task_outputContext)

	// ExitTask_command_string_part is called when exiting the task_command_string_part production.
	ExitTask_command_string_part(c *Task_command_string_partContext)

	// ExitTask_command_expr_part is called when exiting the task_command_expr_part production.
	ExitTask_command_expr_part(c *Task_command_expr_partContext)

	// ExitTask_command_expr_with_string is called when exiting the task_command_expr_with_string production.
	ExitTask_command_expr_with_string(c *Task_command_expr_with_stringContext)

	// ExitTask_command is called when exiting the task_command production.
	ExitTask_command(c *Task_commandContext)

	// ExitTask_element is called when exiting the task_element production.
	ExitTask_element(c *Task_elementContext)

	// ExitTask is called when exiting the task production.
	ExitTask(c *TaskContext)

	// ExitInner_workflow_element is called when exiting the inner_workflow_element production.
	ExitInner_workflow_element(c *Inner_workflow_elementContext)

	// ExitCall_alias is called when exiting the call_alias production.
	ExitCall_alias(c *Call_aliasContext)

	// ExitCall_input is called when exiting the call_input production.
	ExitCall_input(c *Call_inputContext)

	// ExitCall_inputs is called when exiting the call_inputs production.
	ExitCall_inputs(c *Call_inputsContext)

	// ExitCall_body is called when exiting the call_body production.
	ExitCall_body(c *Call_bodyContext)

	// ExitCall_after is called when exiting the call_after production.
	ExitCall_after(c *Call_afterContext)

	// ExitCall_name is called when exiting the call_name production.
	ExitCall_name(c *Call_nameContext)

	// ExitCall is called when exiting the call production.
	ExitCall(c *CallContext)

	// ExitScatter is called when exiting the scatter production.
	ExitScatter(c *ScatterContext)

	// ExitConditional is called when exiting the conditional production.
	ExitConditional(c *ConditionalContext)

	// ExitWorkflow_input is called when exiting the workflow_input production.
	ExitWorkflow_input(c *Workflow_inputContext)

	// ExitWorkflow_output is called when exiting the workflow_output production.
	ExitWorkflow_output(c *Workflow_outputContext)

	// ExitInput is called when exiting the input production.
	ExitInput(c *InputContext)

	// ExitOutput is called when exiting the output production.
	ExitOutput(c *OutputContext)

	// ExitInner_element is called when exiting the inner_element production.
	ExitInner_element(c *Inner_elementContext)

	// ExitParameter_meta_element is called when exiting the parameter_meta_element production.
	ExitParameter_meta_element(c *Parameter_meta_elementContext)

	// ExitMeta_element is called when exiting the meta_element production.
	ExitMeta_element(c *Meta_elementContext)

	// ExitWorkflow is called when exiting the workflow production.
	ExitWorkflow(c *WorkflowContext)

	// ExitDocument_element is called when exiting the document_element production.
	ExitDocument_element(c *Document_elementContext)

	// ExitDocument is called when exiting the document production.
	ExitDocument(c *DocumentContext)
}
