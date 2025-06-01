package syrascript

import (
	"fmt"
	"strconv"
)

// Precedências de operadores
const (
	_ int = iota
	LOWEST
	EQUALS      // ==
	LESSGREATER // > ou <
	SUM         // +
	PRODUCT     // *
	PREFIX      // -X ou !X
	CALL        // myFunction(X)
)

var precedences = map[TokenType]int{
	EQ:       EQUALS,
	NOT_EQ:   EQUALS,
	LT:       LESSGREATER,
	GT:       LESSGREATER,
	LTE:      LESSGREATER,
	GTE:      LESSGREATER,
	PLUS:     SUM,
	MINUS:    SUM,
	SLASH:    PRODUCT,
	ASTERISK: PRODUCT,
	LPAREN:   CALL,
}

// tokenTypeToString converte TokenType para string legível
func tokenTypeToString(t TokenType) string {
	switch t {
	case ILLEGAL:
		return "ILLEGAL"
	case EOF:
		return "EOF"
	case IDENT:
		return "IDENT"
	case INT:
		return "INT"
	case FLOAT:
		return "FLOAT"
	case STRING:
		return "STRING"
	case ASSIGN:
		return "="
	case PLUS:
		return "+"
	case MINUS:
		return "-"
	case BANG:
		return "!"
	case ASTERISK:
		return "*"
	case SLASH:
		return "/"
	case EQ:
		return "=="
	case NOT_EQ:
		return "!="
	case LT:
		return "<"
	case GT:
		return ">"
	case LTE:
		return "<="
	case GTE:
		return ">="
	case COMMA:
		return ","
	case SEMICOLON:
		return ";"
	case LPAREN:
		return "("
	case RPAREN:
		return ")"
	case LBRACE:
		return "{"
	case RBRACE:
		return "}"
	case FUNCTION:
		return "fn"
	case LET:
		return "let"
	case TRUE:
		return "true"
	case FALSE:
		return "false"
	case IF:
		return "if"
	case ELSE:
		return "else"
	case RETURN:
		return "return"
	case WHILE:
		return "while"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", int(t))
	}
}

type (
	prefixParseFn func() Expression
	infixParseFn  func(Expression) Expression
)

// Parser responsável por criar a AST
type Parser struct {
	l *Lexer

	curToken  Token
	peekToken Token
	errors    []string

	prefixParseFns map[TokenType]prefixParseFn
	infixParseFns  map[TokenType]infixParseFn
}

// NewParser cria um novo Parser
func NewParser(l *Lexer) *Parser {
	p := &Parser{
		l:      l,
		errors: []string{},
	}

	// Inicializa mapas de funções de parsing
	p.prefixParseFns = make(map[TokenType]prefixParseFn)
	p.registerPrefix(IDENT, p.parseIdentifier)
	p.registerPrefix(INT, p.parseIntegerLiteral)
	p.registerPrefix(FLOAT, p.parseFloatLiteral)
	p.registerPrefix(STRING, p.parseStringLiteral)
	p.registerPrefix(TRUE, p.parseBoolean)
	p.registerPrefix(FALSE, p.parseBoolean)
	p.registerPrefix(MINUS, p.parsePrefixExpression)
	p.registerPrefix(BANG, p.parsePrefixExpression)
	p.registerPrefix(LPAREN, p.parseGroupedExpression)
	p.registerPrefix(IF, p.parseIfExpression)
	p.registerPrefix(FUNCTION, p.parseFunctionLiteral)
	p.registerPrefix(WHILE, func() Expression { return p.parseWhileStatement() })

	p.infixParseFns = make(map[TokenType]infixParseFn)
	p.registerInfix(PLUS, p.parseInfixExpression)
	p.registerInfix(MINUS, p.parseInfixExpression)
	p.registerInfix(SLASH, p.parseInfixExpression)
	p.registerInfix(ASTERISK, p.parseInfixExpression)
	p.registerInfix(EQ, p.parseInfixExpression)
	p.registerInfix(NOT_EQ, p.parseInfixExpression)
	p.registerInfix(LT, p.parseInfixExpression)
	p.registerInfix(GT, p.parseInfixExpression)
	p.registerInfix(LTE, p.parseInfixExpression)
	p.registerInfix(GTE, p.parseInfixExpression)
	p.registerInfix(LPAREN, p.parseCallExpression)

	// Lê dois tokens para definir curToken e peekToken
	p.nextToken()
	p.nextToken()

	return p
}

func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.l.NextToken()
}

// Errors retorna erros de parsing
func (p *Parser) Errors() []string {
	return p.errors
}

// ParseProgram cria a AST para o programa
func (p *Parser) ParseProgram() *Program {
	program := &Program{
		Statements: []Statement{},
	}

	for !p.curTokenIs(EOF) {
		stmt := p.parseStatement()
		program.Statements = append(program.Statements, stmt)
		p.nextToken()
	}

	return program
}

// Métodos de ajuda para verificar tokens
func (p *Parser) curTokenIs(t TokenType) bool {
	return p.curToken.Type == t
}

func (p *Parser) peekTokenIs(t TokenType) bool {
	return p.peekToken.Type == t
}

func (p *Parser) expectPeek(t TokenType) bool {
	if p.peekTokenIs(t) {
		p.nextToken()
		return true
	}
	p.peekError(t)
	return false
}

func (p *Parser) peekError(t TokenType) {
	msg := fmt.Sprintf("Linha %d:%d: esperado token %s, encontrado %s",
		p.peekToken.Line, p.peekToken.Column, tokenTypeToString(t), tokenTypeToString(p.peekToken.Type))
	p.errors = append(p.errors, msg)
}

func (p *Parser) registerPrefix(tokenType TokenType, fn prefixParseFn) {
	p.prefixParseFns[tokenType] = fn
}

func (p *Parser) registerInfix(tokenType TokenType, fn infixParseFn) {
	p.infixParseFns[tokenType] = fn
}

func (p *Parser) noPrefixParseFnError(t TokenType) {
	msg := fmt.Sprintf("Linha %d:%d: não há função de parsing prefix para %s",
		p.curToken.Line, p.curToken.Column, tokenTypeToString(t))
	p.errors = append(p.errors, msg)
}

func (p *Parser) peekPrecedence() int {
	if p, ok := precedences[p.peekToken.Type]; ok {
		return p
	}
	return LOWEST
}

func (p *Parser) curPrecedence() int {
	if p, ok := precedences[p.curToken.Type]; ok {
		return p
	}
	return LOWEST
}

// Parsers de statements
func (p *Parser) parseStatement() Statement {
	switch p.curToken.Type {
	case LET:
		return p.parseLetStatement()
	case RETURN:
		return p.parseReturnStatement()
	default:
		return p.parseExpressionStatement()
	}
}

func (p *Parser) parseLetStatement() *LetStatement {
	stmt := &LetStatement{Token: p.curToken}

	if !p.expectPeek(IDENT) {
		return nil
	}

	stmt.Name = &Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if !p.expectPeek(ASSIGN) {
		return nil
	}

	p.nextToken()

	stmt.Value = p.parseExpression(LOWEST)

	if p.peekTokenIs(SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseReturnStatement() *ReturnStatement {
	stmt := &ReturnStatement{Token: p.curToken}

	p.nextToken()

	stmt.ReturnValue = p.parseExpression(LOWEST)

	if p.peekTokenIs(SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseExpressionStatement() *ExpressionStatement {
	stmt := &ExpressionStatement{Token: p.curToken}

	stmt.Expression = p.parseExpression(LOWEST)

	if p.peekTokenIs(SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseBlockStatement() *BlockStatement {
	block := &BlockStatement{Token: p.curToken}
	block.Statements = []Statement{}

	p.nextToken()

	for !p.curTokenIs(RBRACE) && !p.curTokenIs(EOF) {
		stmt := p.parseStatement()
		block.Statements = append(block.Statements, stmt)
		p.nextToken()
	}

	return block
}

func (p *Parser) parseWhileStatement() Expression {
	stmt := &WhileExpression{Token: p.curToken}

	if !p.expectPeek(LPAREN) {
		return nil
	}

	p.nextToken()
	stmt.Condition = p.parseExpression(LOWEST)

	if !p.expectPeek(RPAREN) {
		return nil
	}

	if !p.expectPeek(LBRACE) {
		return nil
	}

	stmt.Body = p.parseBlockStatement()

	return stmt
}

// Parsers de expressões
func (p *Parser) parseExpression(precedence int) Expression {
	prefix := p.prefixParseFns[p.curToken.Type]
	if prefix == nil {
		p.noPrefixParseFnError(p.curToken.Type)
		return nil
	}
	leftExp := prefix()

	for !p.peekTokenIs(SEMICOLON) && precedence < p.peekPrecedence() {
		infix := p.infixParseFns[p.peekToken.Type]
		if infix == nil {
			return leftExp
		}

		p.nextToken()

		leftExp = infix(leftExp)
	}

	return leftExp
}

func (p *Parser) parseIdentifier() Expression {
	return &Identifier{Token: p.curToken, Value: p.curToken.Literal}
}

func (p *Parser) parseIntegerLiteral() Expression {
	lit := &IntegerLiteral{Token: p.curToken}

	value, err := strconv.ParseInt(p.curToken.Literal, 0, 64)
	if err != nil {
		msg := fmt.Sprintf("Linha %d:%d: não foi possível interpretar %q como inteiro",
			p.curToken.Line, p.curToken.Column, p.curToken.Literal)
		p.errors = append(p.errors, msg)
		return nil
	}

	lit.Value = value
	return lit
}

func (p *Parser) parseFloatLiteral() Expression {
	lit := &FloatLiteral{Token: p.curToken}

	value, err := strconv.ParseFloat(p.curToken.Literal, 64)
	if err != nil {
		msg := fmt.Sprintf("Linha %d:%d: não foi possível interpretar %q como decimal",
			p.curToken.Line, p.curToken.Column, p.curToken.Literal)
		p.errors = append(p.errors, msg)
		return nil
	}

	lit.Value = value
	return lit
}

func (p *Parser) parseStringLiteral() Expression {
	return &StringLiteral{Token: p.curToken, Value: p.curToken.Literal}
}

func (p *Parser) parseBoolean() Expression {
	return &BooleanLiteral{Token: p.curToken, Value: p.curTokenIs(TRUE)}
}

func (p *Parser) parsePrefixExpression() Expression {
	expression := &PrefixExpression{
		Token:    p.curToken,
		Operator: p.curToken.Literal,
	}

	p.nextToken()

	expression.Right = p.parseExpression(PREFIX)

	return expression
}

func (p *Parser) parseInfixExpression(left Expression) Expression {
	expression := &InfixExpression{
		Token:    p.curToken,
		Operator: p.curToken.Literal,
		Left:     left,
	}

	precedence := p.curPrecedence()
	p.nextToken()
	expression.Right = p.parseExpression(precedence)

	return expression
}

func (p *Parser) parseGroupedExpression() Expression {
	p.nextToken()

	exp := p.parseExpression(LOWEST)

	if !p.expectPeek(RPAREN) {
		return nil
	}

	return exp
}

func (p *Parser) parseIfExpression() Expression {
	expression := &IfExpression{Token: p.curToken}

	if !p.expectPeek(LPAREN) {
		return nil
	}

	p.nextToken()
	expression.Condition = p.parseExpression(LOWEST)

	if !p.expectPeek(RPAREN) {
		return nil
	}

	if !p.expectPeek(LBRACE) {
		return nil
	}

	expression.Consequence = p.parseBlockStatement()

	if p.peekTokenIs(ELSE) {
		p.nextToken()

		if !p.expectPeek(LBRACE) {
			return nil
		}

		expression.Alternative = p.parseBlockStatement()
	}

	return expression
}

func (p *Parser) parseFunctionLiteral() Expression {
	lit := &FunctionLiteral{Token: p.curToken}

	if !p.expectPeek(LPAREN) {
		return nil
	}

	lit.Parameters = p.parseFunctionParameters()

	if !p.expectPeek(LBRACE) {
		return nil
	}

	lit.Body = p.parseBlockStatement()

	return lit
}

func (p *Parser) parseFunctionParameters() []*Identifier {
	identifiers := []*Identifier{}

	if p.peekTokenIs(RPAREN) {
		p.nextToken()
		return identifiers
	}

	p.nextToken()

	ident := &Identifier{Token: p.curToken, Value: p.curToken.Literal}
	identifiers = append(identifiers, ident)

	for p.peekTokenIs(COMMA) {
		p.nextToken()
		p.nextToken()
		ident := &Identifier{Token: p.curToken, Value: p.curToken.Literal}
		identifiers = append(identifiers, ident)
	}

	if !p.expectPeek(RPAREN) {
		return nil
	}

	return identifiers
}

func (p *Parser) parseCallExpression(function Expression) Expression {
	exp := &CallExpression{Token: p.curToken, Function: function}
	exp.Arguments = p.parseExpressionList(RPAREN)
	return exp
}

func (p *Parser) parseExpressionList(end TokenType) []Expression {
	list := []Expression{}

	if p.peekTokenIs(end) {
		p.nextToken()
		return list
	}

	p.nextToken()
	list = append(list, p.parseExpression(LOWEST))

	for p.peekTokenIs(COMMA) {
		p.nextToken()
		p.nextToken()
		list = append(list, p.parseExpression(LOWEST))
	}

	if !p.expectPeek(end) {
		return nil
	}

	return list
}
