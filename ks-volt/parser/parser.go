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
	SUM   // +
	CALL  // . or (
	INDEX // [
)

var precedences = map[token.TokenType]int{
	token.PLUS:     SUM,
	token.LPAREN:   CALL,
	token.DOT:      CALL,
	token.LBRACKET: INDEX,
}

type Parser struct {
	l      *lexer.Lexer
	errors []string

	curToken  token.Token
	peekToken token.Token
}

func New(l *lexer.Lexer) *Parser {
	p := &Parser{l: l}
	p.nextToken()
	p.nextToken()
	return p
}

func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.l.NextToken()
}

func (p *Parser) ParseProgram() *ast.Program {
	program := &ast.Program{Statements: []ast.Statement{}}
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
	case token.LOOP:
		return p.parseLoopStatement()
	case token.TRY:
		return p.parseTryCatchStatement()
	case token.IF:
		return p.parseIfStatement()
	case token.FN:
		return p.parseFunctionStatement()
	case token.RETURN:
		return p.parseReturnStatement()
	case token.IDENT:
		if p.peekToken.Type == token.ASSIGN {
			return p.parseAssignmentStatement()
		}
		return p.parseExpressionStatement()
	default:
		return p.parseExpressionStatement()
	}
}

func (p *Parser) parseFunctionStatement() *ast.FunctionStatement {
	stmt := &ast.FunctionStatement{Token: p.curToken}
	if !p.expectPeek(token.IDENT) { return nil }
	stmt.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	if !p.expectPeek(token.LPAREN) { return nil }
	stmt.Parameters = p.parseFunctionParameters()
	if !p.expectPeek(token.LBRACE) { return nil }
	stmt.Body = p.parseBlockStatement()
	return stmt
}

func (p *Parser) parseFunctionParameters() []*ast.Identifier {
	identifiers := []*ast.Identifier{}
	if p.peekToken.Type == token.RPAREN {
		p.nextToken()
		return identifiers
	}
	p.nextToken()
	identifiers = append(identifiers, &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal})
	for p.peekToken.Type == token.COMMA {
		p.nextToken()
		p.nextToken()
		identifiers = append(identifiers, &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal})
	}
	if !p.expectPeek(token.RPAREN) { return nil }
	return identifiers
}

func (p *Parser) parseReturnStatement() *ast.ReturnStatement {
	stmt := &ast.ReturnStatement{Token: p.curToken}
	p.nextToken()
	stmt.ReturnValue = p.parseExpression(LOWEST)
	return stmt
}

func (p *Parser) parseIfStatement() *ast.IfStatement {
	stmt := &ast.IfStatement{Token: p.curToken}
	if !p.expectPeek(token.LPAREN) { return nil }
	p.nextToken()
	stmt.Condition = p.parseExpression(LOWEST)
	if !p.expectPeek(token.RPAREN) { return nil }
	if !p.expectPeek(token.LBRACE) { return nil }
	stmt.Consequence = p.parseBlockStatement()
	if p.peekToken.Type == token.ELSE {
		p.nextToken()
		if !p.expectPeek(token.LBRACE) { return nil }
		stmt.Alternative = p.parseBlockStatement()
	}
	return stmt
}

func (p *Parser) parseSpawnStatement() *ast.SpawnStatement {
	stmt := &ast.SpawnStatement{Token: p.curToken}
	if p.peekToken.Type == token.LPAREN {
		stmt.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	} else if p.peekToken.Type == token.IDENT || p.peekToken.Type == token.INTERVAL || p.peekToken.Type == token.CONNECT_BOT || p.peekToken.Type == token.ON {
		p.nextToken()
		stmt.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	} else {
		p.peekError(token.IDENT)
		return nil
	}
	if !p.expectPeek(token.LPAREN) { return nil }
	stmt.Args = p.parseCallArguments()
	if !p.expectPeek(token.LBRACE) { return nil }
	stmt.Body = p.parseBlockStatement()
	return stmt
}

func (p *Parser) parseLoopStatement() *ast.LoopStatement {
	stmt := &ast.LoopStatement{Token: p.curToken}
	if !p.expectPeek(token.IDENT) { return nil }
	stmt.Variable = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	if !p.expectPeek(token.IN) { return nil }
	p.nextToken()
	stmt.Iterable = p.parseExpression(LOWEST)
	if !p.expectPeek(token.LBRACE) { return nil }
	stmt.Body = p.parseBlockStatement()
	return stmt
}

func (p *Parser) parseTryCatchStatement() *ast.TryCatchStatement {
	stmt := &ast.TryCatchStatement{Token: p.curToken}
	if !p.expectPeek(token.LBRACE) { return nil }
	stmt.TryBody = p.parseBlockStatement()
	if !p.expectPeek(token.CATCH) { return nil }
	if !p.expectPeek(token.LPAREN) { return nil }
	if !p.expectPeek(token.IDENT) { return nil }
	stmt.CatchVariable = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	if !p.expectPeek(token.RPAREN) { return nil }
	if !p.expectPeek(token.LBRACE) { return nil }
	stmt.CatchBody = p.parseBlockStatement()
	return stmt
}

func (p *Parser) parseAssignmentStatement() *ast.AssignmentStatement {
	stmt := &ast.AssignmentStatement{Token: p.curToken}
	stmt.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	if !p.expectPeek(token.ASSIGN) { return nil }
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
	block := &ast.BlockStatement{Token: p.curToken, Statements: []ast.Statement{}}
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
	case token.TRUE, token.FALSE:
		leftExp = &ast.Boolean{Token: p.curToken, Value: p.curToken.Type == token.TRUE}
	case token.LBRACKET:
		leftExp = p.parseArrayLiteral()
	case token.PRINT, token.SERVE_HTML, token.FETCH_API, token.DB_SAVE, token.DB_GET, token.JSON_PARSE, token.FILE_WRITE, token.EMIT, token.GET_ADDR:
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
		case token.LBRACKET:
			p.nextToken()
			leftExp = p.parseIndexExpression(leftExp)
		case token.DOT:
			p.nextToken()
			leftExp = p.parseMethodCallExpression(leftExp)
		default:
			return leftExp
		}
	}
	return leftExp
}

func (p *Parser) parseMethodCallExpression(object ast.Expression) ast.Expression {
	exp := &ast.MethodCallExpression{Token: p.curToken, Object: object}
	if !p.expectPeek(token.IDENT) { return nil }
	exp.Method = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	if p.peekToken.Type == token.LPAREN {
		p.nextToken()
		exp.Arguments = p.parseCallArguments()
	}
	return exp
}

func (p *Parser) parseArrayLiteral() ast.Expression {
	array := &ast.ArrayLiteral{Token: p.curToken}
	array.Elements = p.parseExpressionList(token.RBRACKET)
	return array
}

func (p *Parser) parseExpressionList(end token.TokenType) []ast.Expression {
	list := []ast.Expression{}
	if p.peekToken.Type == end {
		p.nextToken()
		return list
	}
	p.nextToken()
	list = append(list, p.parseExpression(LOWEST))
	for p.peekToken.Type == token.COMMA {
		p.nextToken()
		p.nextToken()
		list = append(list, p.parseExpression(LOWEST))
	}
	if !p.expectPeek(end) { return nil }
	return list
}

func (p *Parser) parseIndexExpression(left ast.Expression) ast.Expression {
	exp := &ast.IndexExpression{Token: p.curToken, Left: left}
	p.nextToken()
	exp.Index = p.parseExpression(LOWEST)
	if !p.expectPeek(token.RBRACKET) { return nil }
	return exp
}

func (p *Parser) parseInfixExpression(left ast.Expression) ast.Expression {
	exp := &ast.InfixExpression{Token: p.curToken, Operator: p.curToken.Literal, Left: left}
	pre := p.curPrecedence()
	p.nextToken()
	exp.Right = p.parseExpression(pre)
	return exp
}

func (p *Parser) parseIntegerLiteral() ast.Expression {
	lit := &ast.IntegerLiteral{Token: p.curToken}
	val, _ := strconv.ParseInt(p.curToken.Literal, 0, 64)
	lit.Value = val
	return lit
}

func (p *Parser) parseCallExpression(fn ast.Expression) ast.Expression {
	exp := &ast.CallExpression{Token: p.curToken, Function: fn}
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
	if !p.expectPeek(token.RPAREN) { return nil }
	return args
}

func (p *Parser) peekPrecedence() int {
	if pre, ok := precedences[p.peekToken.Type]; ok { return pre }
	return LOWEST
}

func (p *Parser) curPrecedence() int {
	if pre, ok := precedences[p.curToken.Type]; ok { return pre }
	return LOWEST
}

func (p *Parser) expectPeek(t token.TokenType) bool {
	if p.peekToken.Type == t {
		p.nextToken()
		return true
	}
	p.peekError(t)
	return false
}

func (p *Parser) Errors() []string { return p.errors }

func (p *Parser) peekError(t token.TokenType) {
	p.errors = append(p.errors, fmt.Sprintf("expected %s, got %s", t, p.peekToken.Type))
}
