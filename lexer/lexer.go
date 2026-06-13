package lexer

import (
	"ks-volt/token"
)

type Lexer struct {
	input        string
	position     int
	readPosition int
	ch           byte
	line         int
	column       int
}

func New(input string) *Lexer {
	l := &Lexer{input: input, line: 1, column: 0}
	l.readChar()
	return l
}

func (l *Lexer) readChar() {
	if l.readPosition >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = l.input[l.readPosition]
	}

	if l.ch == '\n' {
		l.line++
		l.column = 0
	} else {
		l.column++
	}

	l.position = l.readPosition
	l.readPosition++
}

func (l *Lexer) peekChar() byte {
	if l.readPosition >= len(l.input) {
		return 0
	}
	return l.input[l.readPosition]
}

func (l *Lexer) NextToken() token.Token {
	var tok token.Token
	l.skipWhitespace()

	line := l.line
	col := l.column

	if l.ch == '/' && l.peekChar() == '/' {
		l.skipComment()
		return l.NextToken()
	}

	if l.ch == '-' && l.peekChar() == '>' {
		l.readChar()
		l.readChar()
		return token.Token{Type: token.ARROW, Literal: "->", Line: line, Column: col}
	}

	switch l.ch {
	case '=':
		tok = newToken(token.ASSIGN, l.ch, line, col)
	case '+':
		tok = newToken(token.PLUS, l.ch, line, col)
	case '.':
		tok = newToken(token.DOT, l.ch, line, col)
	case ',':
		tok = newToken(token.COMMA, l.ch, line, col)
	case '(':
		tok = newToken(token.LPAREN, l.ch, line, col)
	case ')':
		tok = newToken(token.RPAREN, l.ch, line, col)
	case '{':
		tok = newToken(token.LBRACE, l.ch, line, col)
	case '}':
		tok = newToken(token.RBRACE, l.ch, line, col)
	case '[':
		tok = newToken(token.LBRACKET, l.ch, line, col)
	case ']':
		tok = newToken(token.RBRACKET, l.ch, line, col)
	case '&':
		tok = newToken(token.AMPERSAND, l.ch, line, col)
	case '`':
		tok.Type = token.BACKTICK
		tok.Literal = l.readBacktickString()
		tok.Line = line
		tok.Column = col
	case '"':
		tok.Type = token.STRING
		tok.Literal = l.readString()
		tok.Line = line
		tok.Column = col
	case 0:
		tok.Type = token.EOF
		tok.Literal = ""
		tok.Line = line
		tok.Column = col
	default:
		if isLetter(l.ch) {
			tok.Literal = l.readIdentifier()
			tok.Type = token.LookupIdent(tok.Literal)
			tok.Line = line
			tok.Column = col

			if isPolyglotBlock(tok.Type) {
				l.skipWhitespace()
				if l.ch == '{' {
					tok.Literal = l.readRawBlock()
				}
			}

			return tok
		} else if isDigit(l.ch) {
			tok.Type = token.INT
			tok.Literal = l.readNumber()
			tok.Line = line
			tok.Column = col
			return tok
		} else {
			tok = newToken(token.ILLEGAL, l.ch, line, col)
		}
	}
	l.readChar()
	return tok
}

func (l *Lexer) readString() string {
	position := l.position + 1
	for {
		l.readChar()
		if l.ch == '\\' && l.peekChar() == '"' {
			l.readChar() // skip \
			l.readChar() // skip "
			continue
		}
		if l.ch == '"' || l.ch == 0 {
			break
		}
	}
	return l.input[position:l.position]
}

func (l *Lexer) readBacktickString() string {
	position := l.position + 1
	for {
		l.readChar()
		if l.ch == '`' || l.ch == 0 {
			break
		}
	}
	return l.input[position:l.position]
}

func (l *Lexer) readIdentifier() string {
	pos := l.position
	for isLetter(l.ch) || isDigit(l.ch) {
		l.readChar()
	}
	return l.input[pos:l.position]
}

func (l *Lexer) readNumber() string {
	pos := l.position
	for isDigit(l.ch) {
		l.readChar()
	}
	return l.input[pos:l.position]
}

func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
		l.readChar()
	}
}

func (l *Lexer) skipComment() {
	for l.ch != '\n' && l.ch != 0 {
		l.readChar()
	}
	l.skipWhitespace()
}

func isLetter(ch byte) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '_'
}

func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}

func isPolyglotBlock(t token.TokenType) bool {
	return t == token.GO_BLOCK || t == token.RUST_BLOCK || t == token.JS_BLOCK || t == token.PY_BLOCK
}

func (l *Lexer) readRawBlock() string {
	l.readChar() // skip {
	pos := l.position
	count := 1
	for count > 0 && l.ch != 0 {
		if l.ch == '{' {
			count++
		} else if l.ch == '}' {
			count--
		}
		if count > 0 {
			l.readChar()
		}
	}
	content := l.input[pos:l.position]
	if l.ch == '}' {
		l.readChar()
	}
	return content
}

func newToken(t token.TokenType, ch byte, line, col int) token.Token {
	return token.Token{Type: t, Literal: string(ch), Line: line, Column: col}
}
