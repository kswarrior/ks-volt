package parser

import (
	"fmt"
	"ks-volt/ast"
	"ks-volt/lexer"
	"ks-volt/token"
	"strconv"
)

const (
	_ int = iota
	LOWEST
	SUM  // +
	CALL // function(args)
)

var precedences = map[token.TokenType]int{
	token.PLUS:   SUM,
	token.LPAREN: CALL,
}

type Parser struct {
	l      *lexer.Lexer
	errors []string

	curToken  token.Token
	peekToken token.Token
}

func New(l *lexer.Lexer) *Parser {
	p := &Parser{
		l:      l,
		errors: []string{},
	}

	// Read two tokens, so curToken and peekToken are both set
	p.nextToken()
	p.nextToken()

	return p
}

func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.l.NextToken()
}

func (p *Parser) ParseProgram() *ast.Program {
	program := &ast.Program{}
	program.Statements = []ast.Statement{}

	for p.curToken.Type != token.EOF {
		stmt := p.parseStatement()
		if stmt != nil {
			program.Statements = append(program.Statements, stmt)
		}
		p.nextToken()
	}

	return program
}

func (p *Parser) parseStatement() ast.Statement {
	switch p.curToken.Type {
	case token.SPAWN, token.CONNECT_BOT, token.INTERVAL, token.ON:
		return p.parseSpawnStatement()
	case token.IDENT:
		if p.peekToken.Type == token.ASSIGN {
			return p.parseAssignmentStatement()
		}
		return p.parseExpressionStatement()
	default:
		return p.parseExpressionStatement()
	}
}

func (p *Parser) parseSpawnStatement() *ast.SpawnStatement {
	stmt := &ast.SpawnStatement{Token: p.curToken}

	if p.peekToken.Type == token.LPAREN {
		// handle anonymous spawn or builtin block like connect_bot(...) { ... }
		stmt.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	} else if p.peekToken.Type == token.IDENT || p.peekToken.Type == token.CONNECT_BOT {
		p.nextToken()
		stmt.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	} else {
		p.peekError(token.IDENT)
		return nil
	}

	if !p.expectPeek(token.LPAREN) {
		return nil
	}

	stmt.Args = p.parseCallArguments()

	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	stmt.Body = p.parseBlockStatement()

	return stmt
}

func (p *Parser) parseAssignmentStatement() *ast.AssignmentStatement {
	stmt := &ast.AssignmentStatement{Token: p.curToken}
	stmt.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if !p.expectPeek(token.ASSIGN) {
		return nil
	}

	p.nextToken()

	stmt.Value = p.parseExpression(LOWEST)

	return stmt
}

func (p *Parser) parseExpressionStatement() *ast.ExpressionStatement {
	stmt := &ast.ExpressionStatement{Token: p.curToken}

	stmt.Expression = p.parseExpression(LOWEST)

	return stmt
}

func (p *Parser) parseBlockStatement() *ast.BlockStatement {
	block := &ast.BlockStatement{Token: p.curToken}
	block.Statements = []ast.Statement{}

	p.nextToken()

	for p.curToken.Type != token.RBRACE && p.curToken.Type != token.EOF {
		stmt := p.parseStatement()
		if stmt != nil {
			block.Statements = append(block.Statements, stmt)
		}
		p.nextToken()
	}

	return block
}

func (p *Parser) parseExpression(precedence int) ast.Expression {
	var leftExp ast.Expression

	switch p.curToken.Type {
	case token.IDENT:
		leftExp = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	case token.INT:
		leftExp = p.parseIntegerLiteral()
	case token.STRING:
		leftExp = &ast.StringLiteral{Token: p.curToken, Value: p.curToken.Literal}
	case token.PRINT:
		leftExp = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	case token.SERVE_HTML:
		leftExp = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	case token.FETCH_API:
		leftExp = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	case token.DB_SAVE:
		leftExp = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	case token.DB_GET:
		leftExp = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	case token.CONNECT_BOT:
		leftExp = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	case token.FILE_WRITE:
		leftExp = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	case token.EMIT:
		leftExp = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	default:
		return nil
	}

	for p.peekToken.Type != token.EOF && precedence < p.peekPrecedence() {
		switch p.peekToken.Type {
		case token.PLUS:
			p.nextToken()
			leftExp = p.parseInfixExpression(leftExp)
		case token.LPAREN:
			p.nextToken()
			leftExp = p.parseCallExpression(leftExp)
		default:
			return leftExp
		}
	}

	return leftExp
}

func (p *Parser) parseInfixExpression(left ast.Expression) ast.Expression {
	expression := &ast.InfixExpression{
		Token:    p.curToken,
		Operator: p.curToken.Literal,
		Left:     left,
	}

	precedence := p.curPrecedence()
	p.nextToken()
	expression.Right = p.parseExpression(precedence)

	return expression
}

func (p *Parser) parseIntegerLiteral() ast.Expression {
	lit := &ast.IntegerLiteral{Token: p.curToken}

	value, err := strconv.ParseInt(p.curToken.Literal, 0, 64)
	if err != nil {
		p.errors = append(p.errors, fmt.Sprintf("could not parse %q as integer", p.curToken.Literal))
		return nil
	}

	lit.Value = value
	return lit
}

func (p *Parser) parseCallExpression(function ast.Expression) ast.Expression {
	exp := &ast.CallExpression{Token: p.curToken, Function: function}
	exp.Arguments = p.parseCallArguments()
	return exp
}

func (p *Parser) parseCallArguments() []ast.Expression {
	args := []ast.Expression{}

	if p.peekToken.Type == token.RPAREN {
		p.nextToken()
		return args
	}

	p.nextToken()
	args = append(args, p.parseExpression(LOWEST))

	for p.peekToken.Type == token.COMMA {
		p.nextToken()
		p.nextToken()
		args = append(args, p.parseExpression(LOWEST))
	}

	if !p.expectPeek(token.RPAREN) {
		return nil
	}

	return args
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

func (p *Parser) curTokenIs(t token.TokenType) bool {
	return p.curToken.Type == t
}

func (p *Parser) peekTokenIs(t token.TokenType) bool {
	return p.peekToken.Type == t
}

func (p *Parser) expectPeek(t token.TokenType) bool {
	if p.peekTokenIs(t) {
		p.nextToken()
		return true
	} else {
		p.peekError(t)
		return false
	}
}

func (p *Parser) Errors() []string {
	return p.errors
}

func (p *Parser) peekError(t token.TokenType) {
	msg := fmt.Sprintf("expected next token to be %s, got %s instead", t, p.peekToken.Type)
	p.errors = append(p.errors, msg)
}
