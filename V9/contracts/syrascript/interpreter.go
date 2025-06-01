package syrascript

import (
	"fmt"
	"math"
)

// ObjectType representa o tipo de um objeto
type ObjectType string

const (
	INTEGER_OBJ      = "INTEGER"
	FLOAT_OBJ        = "FLOAT"
	BOOLEAN_OBJ      = "BOOLEAN"
	STRING_OBJ       = "STRING"
	NULL_OBJ         = "NULL"
	RETURN_VALUE_OBJ = "RETURN_VALUE"
	ERROR_OBJ        = "ERROR"
	FUNCTION_OBJ     = "FUNCTION"
	BUILTIN_OBJ      = "BUILTIN"
)

// Object representa um objeto em SyraScript
type Object interface {
	Type() ObjectType
	Inspect() string
}

// Integer representa um inteiro
type Integer struct {
	Value int64
}

func (i *Integer) Type() ObjectType { return INTEGER_OBJ }
func (i *Integer) Inspect() string  { return fmt.Sprintf("%d", i.Value) }

// Float representa um decimal
type Float struct {
	Value float64
}

func (f *Float) Type() ObjectType { return FLOAT_OBJ }
func (f *Float) Inspect() string  { return fmt.Sprintf("%f", f.Value) }

// Boolean representa um booleano
type Boolean struct {
	Value bool
}

func (b *Boolean) Type() ObjectType { return BOOLEAN_OBJ }
func (b *Boolean) Inspect() string  { return fmt.Sprintf("%t", b.Value) }

// String representa uma string
type String struct {
	Value string
}

func (s *String) Type() ObjectType { return STRING_OBJ }
func (s *String) Inspect() string  { return s.Value }

// Null representa um valor nulo
type Null struct{}

func (n *Null) Type() ObjectType { return NULL_OBJ }
func (n *Null) Inspect() string  { return "null" }

// ReturnValue representa um valor de retorno de função
type ReturnValue struct {
	Value Object
}

func (rv *ReturnValue) Type() ObjectType { return RETURN_VALUE_OBJ }
func (rv *ReturnValue) Inspect() string  { return rv.Value.Inspect() }

// Error representa um erro durante a execução
type Error struct {
	Message string
	Line    int
	Column  int
}

func (e *Error) Type() ObjectType { return ERROR_OBJ }
func (e *Error) Inspect() string {
	return fmt.Sprintf("Erro linha %d:%d: %s", e.Line, e.Column, e.Message)
}

// Function representa uma função definida pelo usuário
type Function struct {
	Parameters []*Identifier
	Body       *BlockStatement
	Env        *Environment
}

func (f *Function) Type() ObjectType { return FUNCTION_OBJ }
func (f *Function) Inspect() string {
	return "<função>"
}

// BuiltinFunction representa uma função nativa fornecida pelo ambiente
type BuiltinFunction func(args ...Object) Object

// Builtin representa uma função embutida
type Builtin struct {
	Fn BuiltinFunction
}

func (b *Builtin) Type() ObjectType { return BUILTIN_OBJ }
func (b *Builtin) Inspect() string  { return "builtin function" }

// Environment gerencia o escopo de variáveis
type Environment struct {
	store map[string]Object
	outer *Environment
}

// NewEnvironment cria um novo ambiente
func NewEnvironment() *Environment {
	s := make(map[string]Object)
	return &Environment{store: s, outer: nil}
}

// NewEnclosedEnvironment cria um ambiente com referência ao ambiente pai
func NewEnclosedEnvironment(outer *Environment) *Environment {
	env := NewEnvironment()
	env.outer = outer
	return env
}

// Get obtém um objeto do ambiente
func (e *Environment) Get(name string) (Object, bool) {
	obj, ok := e.store[name]
	if !ok && e.outer != nil {
		obj, ok = e.outer.Get(name)
	}
	return obj, ok
}

// Set define um objeto no ambiente
func (e *Environment) Set(name string, val Object) Object {
	e.store[name] = val
	return val
}

// Evaluator é responsável pela execução do código
type Evaluator struct {
	env      *Environment
	gasUsed  int
	gasLimit int
	line     int
	column   int
}

// Constantes e variáveis globais
var (
	NULL = &Null{}
)

// Boolean constants for interpreter use
var (
	INTERPRETER_TRUE  = &Boolean{Value: true}
	INTERPRETER_FALSE = &Boolean{Value: false}
)

// NewEvaluator cria um novo avaliador com funções nativas
func NewEvaluator(gasLimit int) *Evaluator {
	env := NewEnvironment()

	// Registra funções nativas do blockchain
	builtins := map[string]BuiltinFunction{
		"transfer": func(args ...Object) Object {
			if len(args) != 3 {
				return &Error{Message: "transfer requer 3 argumentos: from, to, amount", Line: 0, Column: 0}
			}

			from, ok := args[0].(*String)
			if !ok {
				return &Error{Message: "primeiro argumento de transfer deve ser string (from)", Line: 0, Column: 0}
			}

			to, ok := args[1].(*String)
			if !ok {
				return &Error{Message: "segundo argumento de transfer deve ser string (to)", Line: 0, Column: 0}
			}

			var amount int64
			switch a := args[2].(type) {
			case *Integer:
				amount = a.Value
			case *Float:
				amount = int64(a.Value)
			default:
				return &Error{Message: "terceiro argumento de transfer deve ser número (amount)", Line: 0, Column: 0}
			}

			return &String{Value: fmt.Sprintf("TRANSFER:%s:%s:%d", from.Value, to.Value, amount)}
		},

		"balance": func(args ...Object) Object {
			if len(args) != 1 {
				return &Error{Message: "balance requer 1 argumento: userID", Line: 0, Column: 0}
			}

			userID, ok := args[0].(*String)
			if !ok {
				return &Error{Message: "argumento de balance deve ser string (userID)", Line: 0, Column: 0}
			}

			return &String{Value: fmt.Sprintf("BALANCE:%s", userID.Value)}
		},

		"blockHeight": func(args ...Object) Object {
			if len(args) != 0 {
				return &Error{Message: "blockHeight não aceita argumentos", Line: 0, Column: 0}
			}

			return &String{Value: "BLOCK_HEIGHT"}
		},

		"blockTime": func(args ...Object) Object {
			if len(args) != 0 {
				return &Error{Message: "blockTime não aceita argumentos", Line: 0, Column: 0}
			}

			return &String{Value: "BLOCK_TIMESTAMP"}
		},

		"log": func(args ...Object) Object {
			if len(args) == 0 {
				return &Error{Message: "log requer pelo menos 1 argumento", Line: 0, Column: 0}
			}

			return &String{Value: fmt.Sprintf("LOG:%s", args[0].Inspect())}
		},
	}

	for name, fn := range builtins {
		env.Set(name, &Builtin{Fn: fn})
	}

	return &Evaluator{
		env:      env,
		gasUsed:  0,
		gasLimit: gasLimit,
	}
}

// GetGasUsed retorna o gás utilizado
func (e *Evaluator) GetGasUsed() int {
	return e.gasUsed
}

// GetEnvironment retorna o ambiente atual (para compatibilidade)
func (e *Evaluator) GetEnvironment() *Environment {
	return e.env
}

// Evaluate avalia um nó AST
func (e *Evaluator) Evaluate(node Node) Object {
	switch node := node.(type) {
	// Statements
	case *Program:
		return e.evaluateProgram(node)
	case *ExpressionStatement:
		return e.Evaluate(node.Expression)
	case *BlockStatement:
		return e.evaluateBlockStatement(node)
	case *IfStatement:
		return e.evaluateIfStatement(node)
	case *WhileStatement:
		return e.evaluateWhileStatement(node)
	case *LetStatement:
		val := e.Evaluate(node.Value)
		if isError(val) {
			return val
		}
		e.env.Set(node.Name.Value, val)
		return val

	case *ReturnStatement:
		val := e.Evaluate(node.ReturnValue)
		if isError(val) {
			return val
		}
		return &ReturnValue{Value: val}

	// Expressions
	case *IntegerLiteral:
		return &Integer{Value: node.Value}
	case *FloatLiteral:
		return &Float{Value: node.Value}
	case *StringLiteral:
		return &String{Value: node.Value}
	case *BooleanLiteral:
		return nativeBoolToBooleanObject(node.Value)
	case *IfExpression:
		return e.evaluateIfExpression(node)
	case *WhileExpression:
		return e.evaluateWhileExpression(node)
	case *PrefixExpression:
		right := e.Evaluate(node.Right)
		if isError(right) {
			return right
		}
		return e.evaluatePrefixExpression(node.Operator, right, node.Token.Line, node.Token.Column)
	case *InfixExpression:
		left := e.Evaluate(node.Left)
		if isError(left) {
			return left
		}
		right := e.Evaluate(node.Right)
		if isError(right) {
			return right
		}
		return e.evaluateInfixExpression(node.Operator, left, right, node.Token.Line, node.Token.Column)
	case *Identifier:
		return e.evaluateIdentifier(node)
	case *FunctionLiteral:
		params := node.Parameters
		body := node.Body
		return &Function{Parameters: params, Body: body, Env: e.env}
	case *CallExpression:
		function := e.Evaluate(node.Function)
		if isError(function) {
			return function
		}

		args := e.evaluateExpressions(node.Arguments)
		if len(args) == 1 && isError(args[0]) {
			return args[0]
		}

		return e.applyFunction(function, args, node.Token.Line, node.Token.Column)
	}

	return NULL
}

// Métodos auxiliares para avaliação
func (e *Evaluator) evaluateProgram(program *Program) Object {
	var result Object

	for _, statement := range program.Statements {
		if e.gasUsed > e.gasLimit {
			return &Error{
				Message: fmt.Sprintf("limite de gás excedido (%d > %d)", e.gasUsed, e.gasLimit),
				Line:    e.line,
				Column:  e.column,
			}
		}

		e.gasUsed++
		result = e.Evaluate(statement)

		if returnValue, ok := result.(*ReturnValue); ok {
			return returnValue.Value
		} else if errObj, ok := result.(*Error); ok {
			return errObj
		}
	}

	return result
}

func (e *Evaluator) evaluateBlockStatement(block *BlockStatement) Object {
	var result Object

	for _, statement := range block.Statements {
		if e.gasUsed > e.gasLimit {
			return &Error{
				Message: fmt.Sprintf("limite de gás excedido (%d > %d)", e.gasUsed, e.gasLimit),
				Line:    e.line,
				Column:  e.column,
			}
		}

		e.gasUsed++
		result = e.Evaluate(statement)

		if result != nil {
			rt := result.Type()
			if rt == RETURN_VALUE_OBJ || rt == ERROR_OBJ {
				return result
			}
		}
	}

	return result
}

func (e *Evaluator) evaluateIfStatement(is *IfStatement) Object {
	condition := e.Evaluate(is.Condition)
	if isError(condition) {
		return condition
	}

	if isTruthy(condition) {
		return e.Evaluate(is.Consequence)
	} else if is.Alternative != nil {
		return e.Evaluate(is.Alternative)
	} else {
		return NULL
	}
}

func (e *Evaluator) evaluateIfExpression(ie *IfExpression) Object {
	condition := e.Evaluate(ie.Condition)
	if isError(condition) {
		return condition
	}

	if isTruthy(condition) {
		return e.Evaluate(ie.Consequence)
	} else if ie.Alternative != nil {
		return e.Evaluate(ie.Alternative)
	} else {
		return NULL
	}
}

func (e *Evaluator) evaluateWhileStatement(ws *WhileStatement) Object {
	var result Object = NULL
	loopCount := 0
	maxLoops := 1000

	for {
		if loopCount >= maxLoops {
			return &Error{
				Message: fmt.Sprintf("limite máximo de loops atingido (%d)", maxLoops),
				Line:    ws.Token.Line,
				Column:  ws.Token.Column,
			}
		}

		loopCount++

		condition := e.Evaluate(ws.Condition)
		if isError(condition) {
			return condition
		}

		if !isTruthy(condition) {
			break
		}

		result = e.Evaluate(ws.Body)

		if result != nil {
			if _, ok := result.(*ReturnValue); ok {
				return result
			}
			if isError(result) {
				return result
			}
		}
	}

	return result
}

func (e *Evaluator) evaluateWhileExpression(we *WhileExpression) Object {
	var result Object = NULL
	loopCount := 0
	maxLoops := 1000

	for {
		if loopCount >= maxLoops {
			return &Error{
				Message: fmt.Sprintf("limite máximo de loops atingido (%d)", maxLoops),
				Line:    we.Token.Line,
				Column:  we.Token.Column,
			}
		}

		loopCount++

		condition := e.Evaluate(we.Condition)
		if isError(condition) {
			return condition
		}

		if !isTruthy(condition) {
			break
		}

		result = e.Evaluate(we.Body)

		if result != nil {
			if _, ok := result.(*ReturnValue); ok {
				return result
			}
			if isError(result) {
				return result
			}
		}
	}

	return result
}

func (e *Evaluator) evaluatePrefixExpression(operator string, right Object, line, column int) Object {
	switch operator {
	case "-":
		return e.evaluateMinusPrefixOperatorExpression(right, line, column)
	case "!":
		return e.evaluateBangOperatorExpression(right)
	default:
		return &Error{
			Message: fmt.Sprintf("operador prefixo desconhecido: %s%s", operator, right.Type()),
			Line:    line,
			Column:  column,
		}
	}
}

func (e *Evaluator) evaluateMinusPrefixOperatorExpression(right Object, line, column int) Object {
	switch right.Type() {
	case INTEGER_OBJ:
		value := right.(*Integer).Value
		return &Integer{Value: -value}
	case FLOAT_OBJ:
		value := right.(*Float).Value
		return &Float{Value: -value}
	default:
		return &Error{
			Message: fmt.Sprintf("operador não suportado: -%s", right.Type()),
			Line:    line,
			Column:  column,
		}
	}
}

func (e *Evaluator) evaluateBangOperatorExpression(right Object) Object {
	switch right {
	case INTERPRETER_TRUE:
		return INTERPRETER_FALSE
	case INTERPRETER_FALSE:
		return INTERPRETER_TRUE
	case NULL:
		return INTERPRETER_TRUE
	default:
		return INTERPRETER_FALSE
	}
}

func (e *Evaluator) evaluateInfixExpression(operator string, left, right Object, line, column int) Object {
	switch {
	case left.Type() == INTEGER_OBJ && right.Type() == INTEGER_OBJ:
		return e.evaluateIntegerInfixExpression(operator, left, right, line, column)
	case left.Type() == FLOAT_OBJ || right.Type() == FLOAT_OBJ:
		return e.evaluateFloatInfixExpression(operator, left, right, line, column)
	case left.Type() == STRING_OBJ && right.Type() == STRING_OBJ:
		return e.evaluateStringInfixExpression(operator, left, right, line, column)
	case operator == "==":
		return nativeBoolToBooleanObject(left == right)
	case operator == "!=":
		return nativeBoolToBooleanObject(left != right)
	case left.Type() != right.Type():
		return &Error{
			Message: fmt.Sprintf("tipos incompatíveis: %s %s %s", left.Type(), operator, right.Type()),
			Line:    line,
			Column:  column,
		}
	default:
		return &Error{
			Message: fmt.Sprintf("operador não suportado: %s %s %s", left.Type(), operator, right.Type()),
			Line:    line,
			Column:  column,
		}
	}
}

func (e *Evaluator) evaluateIntegerInfixExpression(operator string, left, right Object, line, column int) Object {
	leftVal := left.(*Integer).Value
	rightVal := right.(*Integer).Value

	switch operator {
	case "+":
		return &Integer{Value: leftVal + rightVal}
	case "-":
		return &Integer{Value: leftVal - rightVal}
	case "*":
		return &Integer{Value: leftVal * rightVal}
	case "/":
		if rightVal == 0 {
			return &Error{Message: "divisão por zero", Line: line, Column: column}
		}
		return &Integer{Value: leftVal / rightVal}
	case "<":
		return nativeBoolToBooleanObject(leftVal < rightVal)
	case ">":
		return nativeBoolToBooleanObject(leftVal > rightVal)
	case "<=":
		return nativeBoolToBooleanObject(leftVal <= rightVal)
	case ">=":
		return nativeBoolToBooleanObject(leftVal >= rightVal)
	case "==":
		return nativeBoolToBooleanObject(leftVal == rightVal)
	case "!=":
		return nativeBoolToBooleanObject(leftVal != rightVal)
	default:
		return &Error{
			Message: fmt.Sprintf("operador desconhecido: %s %s %s", left.Type(), operator, right.Type()),
			Line:    line,
			Column:  column,
		}
	}
}

func (e *Evaluator) evaluateFloatInfixExpression(operator string, left, right Object, line, column int) Object {
	var leftVal, rightVal float64

	if left.Type() == FLOAT_OBJ {
		leftVal = left.(*Float).Value
	} else if left.Type() == INTEGER_OBJ {
		leftVal = float64(left.(*Integer).Value)
	}

	if right.Type() == FLOAT_OBJ {
		rightVal = right.(*Float).Value
	} else if right.Type() == INTEGER_OBJ {
		rightVal = float64(right.(*Integer).Value)
	}

	switch operator {
	case "+":
		return &Float{Value: leftVal + rightVal}
	case "-":
		return &Float{Value: leftVal - rightVal}
	case "*":
		return &Float{Value: leftVal * rightVal}
	case "/":
		if rightVal == 0 {
			return &Error{Message: "divisão por zero", Line: line, Column: column}
		}
		return &Float{Value: leftVal / rightVal}
	case "<":
		return nativeBoolToBooleanObject(leftVal < rightVal)
	case ">":
		return nativeBoolToBooleanObject(leftVal > rightVal)
	case "<=":
		return nativeBoolToBooleanObject(leftVal <= rightVal)
	case ">=":
		return nativeBoolToBooleanObject(leftVal >= rightVal)
	case "==":
		epsilon := 0.0000001
		return nativeBoolToBooleanObject(math.Abs(leftVal-rightVal) < epsilon)
	case "!=":
		epsilon := 0.0000001
		return nativeBoolToBooleanObject(math.Abs(leftVal-rightVal) >= epsilon)
	default:
		return &Error{
			Message: fmt.Sprintf("operador desconhecido: %s %s %s", left.Type(), operator, right.Type()),
			Line:    line,
			Column:  column,
		}
	}
}

func (e *Evaluator) evaluateStringInfixExpression(operator string, left, right Object, line, column int) Object {
	leftVal := left.(*String).Value
	rightVal := right.(*String).Value

	switch operator {
	case "+":
		return &String{Value: leftVal + rightVal}
	case "==":
		return nativeBoolToBooleanObject(leftVal == rightVal)
	case "!=":
		return nativeBoolToBooleanObject(leftVal != rightVal)
	default:
		return &Error{
			Message: fmt.Sprintf("operador não suportado: %s %s %s", left.Type(), operator, right.Type()),
			Line:    line,
			Column:  column,
		}
	}
}

func (e *Evaluator) evaluateIdentifier(node *Identifier) Object {
	if val, ok := e.env.Get(node.Value); ok {
		return val
	}

	return &Error{
		Message: fmt.Sprintf("identificador não encontrado: %s", node.Value),
		Line:    node.Token.Line,
		Column:  node.Token.Column,
	}
}

func (e *Evaluator) evaluateExpressions(exps []Expression) []Object {
	var result []Object

	for _, exp := range exps {
		evaluated := e.Evaluate(exp)
		if isError(evaluated) {
			return []Object{evaluated}
		}
		result = append(result, evaluated)
	}

	return result
}

// Funções auxiliares para avaliação
func isTruthy(obj Object) bool {
	switch obj := obj.(type) {
	case *Boolean:
		return obj.Value
	case *Null:
		return false
	case *Integer:
		return obj.Value != 0
	case *Float:
		return obj.Value != 0
	case *String:
		return obj.Value != ""
	default:
		return true
	}
}

func isError(obj Object) bool {
	if obj != nil {
		return obj.Type() == ERROR_OBJ
	}
	return false
}

func nativeBoolToBooleanObject(input bool) *Boolean {
	if input {
		return INTERPRETER_TRUE
	}
	return INTERPRETER_FALSE
}

func extendFunctionEnv(fn *Function, args []Object) *Environment {
	env := NewEnclosedEnvironment(fn.Env)

	for paramIdx, param := range fn.Parameters {
		if paramIdx < len(args) {
			env.Set(param.Value, args[paramIdx])
		}
	}

	return env
}

func (e *Evaluator) evalInEnvironment(node Node, env *Environment) Object {
	oldEnv := e.env
	e.env = env
	result := e.Evaluate(node)
	e.env = oldEnv
	return result
}

func unwrapReturnValue(obj Object) Object {
	if returnValue, ok := obj.(*ReturnValue); ok {
		return returnValue.Value
	}
	return obj
}

func (e *Evaluator) applyFunction(fn Object, args []Object, line, column int) Object {
	switch fn := fn.(type) {
	case *Function:
		extendedEnv := extendFunctionEnv(fn, args)
		evaluated := e.evalInEnvironment(fn.Body, extendedEnv)
		return unwrapReturnValue(evaluated)

	case *Builtin:
		e.line = line
		e.column = column
		return fn.Fn(args...)

	default:
		return &Error{
			Message: fmt.Sprintf("not a function: %s", fn.Type()),
			Line:    line,
			Column:  column,
		}
	}
}
