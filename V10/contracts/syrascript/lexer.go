package syrascript

// TokenType representa o tipo de um token
type TokenType int

const (
	ILLEGAL TokenType = iota
	EOF
	IDENT
	INT
	FLOAT
	STRING

	// Operadores
	ASSIGN
	PLUS
	MINUS
	BANG
	ASTERISK
	SLASH
	EQ
	NOT_EQ
	LT
	GT
	LTE
	GTE

	// Delimitadores
	COMMA
	SEMICOLON
	LPAREN
	RPAREN
	LBRACE
	RBRACE

	// Palavras-chave
	FUNCTION
	LET
	TRUE
	FALSE
	IF
	ELSE
	RETURN
	WHILE
)

// Token representa um token léxico
type Token struct {
	Type    TokenType
	Literal string
	Line    int
	Column  int
}

// Lexer realiza análise léxica
type Lexer struct {
	input        string
	position     int  // posição atual no input (aponta para char atual)
	readPosition int  // posição de leitura atual (após char atual)
	ch           byte // char sendo examinado
	line         int  // linha atual
	column       int  // coluna atual
}

// NewLexer cria um novo Lexer
func NewLexer(input string) *Lexer {
	l := &Lexer{input: input, line: 1, column: 0}
	l.readChar()
	return l
}

// readChar lê o próximo caractere
func (l *Lexer) readChar() {
	if l.readPosition >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = l.input[l.readPosition]
	}

	l.position = l.readPosition
	l.readPosition++
	l.column++

	// Atualiza contagem de linhas
	if l.ch == '\n' {
		l.line++
		l.column = 0
	}
}

// NextToken determina o próximo token
func (l *Lexer) NextToken() Token {
	var tok Token

	// Ignora espaços em branco
	l.skipWhitespace()

	// Define a posição atual para o token
	tok.Line = l.line
	tok.Column = l.column

	switch l.ch {
	case '=':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok.Type = EQ
			tok.Literal = string(ch) + string(l.ch)
		} else {
			tok.Type = ASSIGN
			tok.Literal = string(l.ch)
		}
	case '!':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok.Type = NOT_EQ
			tok.Literal = string(ch) + string(l.ch)
		} else {
			tok.Type = BANG
			tok.Literal = string(l.ch)
		}
	case '+':
		tok.Type = PLUS
		tok.Literal = string(l.ch)
	case '-':
		tok.Type = MINUS
		tok.Literal = string(l.ch)
	case '*':
		tok.Type = ASTERISK
		tok.Literal = string(l.ch)
	case '/':
		// Verifica se é comentário
		if l.peekChar() == '/' {
			l.skipComment()
			return l.NextToken()
		} else {
			tok.Type = SLASH
			tok.Literal = string(l.ch)
		}
	case '<':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok.Type = LTE
			tok.Literal = string(ch) + string(l.ch)
		} else {
			tok.Type = LT
			tok.Literal = string(l.ch)
		}
	case '>':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok.Type = GTE
			tok.Literal = string(ch) + string(l.ch)
		} else {
			tok.Type = GT
			tok.Literal = string(l.ch)
		}
	case ',':
		tok.Type = COMMA
		tok.Literal = string(l.ch)
	case ';':
		tok.Type = SEMICOLON
		tok.Literal = string(l.ch)
	case '(':
		tok.Type = LPAREN
		tok.Literal = string(l.ch)
	case ')':
		tok.Type = RPAREN
		tok.Literal = string(l.ch)
	case '{':
		tok.Type = LBRACE
		tok.Literal = string(l.ch)
	case '}':
		tok.Type = RBRACE
		tok.Literal = string(l.ch)
	case '"':
		tok.Type = STRING
		tok.Literal = l.readString()
	case 0:
		tok.Type = EOF
		tok.Literal = ""
	default:
		if isLetter(l.ch) {
			tok.Literal = l.readIdentifier()
			tok.Type = l.lookupIdent(tok.Literal)
			return tok
		} else if isDigit(l.ch) {
			return l.readNumber()
		} else {
			tok.Type = ILLEGAL
			tok.Literal = string(l.ch)
		}
	}

	l.readChar()
	return tok
}

// skipWhitespace pula espaços em branco
func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
		l.readChar()
	}
}

// skipComment pula comentários até o final da linha
func (l *Lexer) skipComment() {
	for l.ch != '\n' && l.ch != 0 {
		l.readChar()
	}
}

// readIdentifier lê um identificador
func (l *Lexer) readIdentifier() string {
	position := l.position
	for isLetter(l.ch) || isDigit(l.ch) {
		l.readChar()
	}
	return l.input[position:l.position]
}

// readNumber lê um número (inteiro ou decimal)
func (l *Lexer) readNumber() Token {
	position := l.position
	isFloat := false

	for isDigit(l.ch) {
		l.readChar()
	}

	if l.ch == '.' && isDigit(l.peekChar()) {
		isFloat = true
		l.readChar() // Consume o '.'
		for isDigit(l.ch) {
			l.readChar()
		}
	}

	if isFloat {
		return Token{Type: FLOAT, Literal: l.input[position:l.position], Line: l.line, Column: l.column - (l.position - position)}
	}
	return Token{Type: INT, Literal: l.input[position:l.position], Line: l.line, Column: l.column - (l.position - position)}
}

// readString lê uma string entre aspas
func (l *Lexer) readString() string {
	l.readChar() // Consume a primeira aspas
	position := l.position

	for l.ch != '"' && l.ch != 0 {
		l.readChar()
	}

	return l.input[position:l.position]
}

// peekChar olha o próximo caractere sem avançar
func (l *Lexer) peekChar() byte {
	if l.readPosition >= len(l.input) {
		return 0
	}
	return l.input[l.readPosition]
}

// lookupIdent verifica se o identificador é uma palavra-chave
func (l *Lexer) lookupIdent(ident string) TokenType {
	keywords := map[string]TokenType{
		"fn":     FUNCTION,
		"let":    LET,
		"true":   TRUE,
		"false":  FALSE,
		"if":     IF,
		"else":   ELSE,
		"return": RETURN,
		"while":  WHILE,
	}

	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}

// Funções auxiliares
func isLetter(ch byte) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '_'
}

func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}
