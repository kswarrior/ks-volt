package parser

import (
	"fmt"
	"ks-volt/ast"
	"ks-volt/lexer"
	"ks-volt/token"
	"strconv"
	"strings"
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
	case token.FS_RM, token.FS_MV, token.FS_CP, token.FS_TOUCH, token.FS_CAT:
		return p.parseFSMacroStatement()
	case token.GO_BLOCK, token.RUST_BLOCK, token.JS_BLOCK, token.PY_BLOCK:
		return p.parsePolyglotBlockStatement()
	case token.IMPORT_COMPONENT:
		return p.parseImportComponentStatement()
	case token.COMPONENT:
		return p.parseComponentDefinition()
	case token.WEB_BLOCK:
		return p.parseWebBlockStatement()
	case token.PATH:
		return p.parsePathStatement()
	case token.PATH_WS:
		return p.parsePathWsStatement()
	case token.BEFORE_EACH:
		return p.parseBeforeEachStatement()
	case token.IMPORT:
		return p.parseImportStatement()
	case token.EXPORT:
		return p.parseExportStatement()
	case token.RESULT:
		return p.parseMatchResultStatement()
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
	if !p.expectPeek(token.IDENT) {
		return nil
	}
	stmt.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	if !p.expectPeek(token.LPAREN) {
		return nil
	}
	stmt.Parameters = p.parseFunctionParameters()
	if !p.expectPeek(token.LBRACE) {
		return nil
	}
	stmt.Body = p.parseBlockStatement()
	return stmt
}

func (p *Parser) parseImportStatement() *ast.ImportStatement {
	stmt := &ast.ImportStatement{Token: p.curToken}
	if !p.expectPeek(token.IDENT) {
		return nil
	}
	stmt.Alias = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	if !p.expectPeek(token.FROM) {
		return nil
	}
	if !p.expectPeek(token.STRING) {
		return nil
	}
	stmt.Path = p.curToken.Literal
	return stmt
}

func (p *Parser) parseExportStatement() *ast.ExportStatement {
	stmt := &ast.ExportStatement{Token: p.curToken}
	if !p.expectPeek(token.IDENT) {
		return nil
	}
	stmt.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	if !p.expectPeek(token.ASSIGN) {
		return nil
	}
	p.nextToken()
	stmt.Value = p.parseExpression(LOWEST)
	return stmt
}

func (p *Parser) parseResultLiteral() ast.Expression {
	lit := &ast.ResultLiteral{Token: p.curToken}
	if !p.expectPeek(token.LPAREN) {
		return nil
	}
	p.nextToken()
	lit.Value = p.parseExpression(LOWEST)
	if !p.expectPeek(token.RPAREN) {
		return nil
	}
	return lit
}

func (p *Parser) parseMatchResultStatement() *ast.MatchResultStatement {
	stmt := &ast.MatchResultStatement{Token: p.curToken}
	p.nextToken()
	stmt.ResultExpr = p.parseExpression(LOWEST)

	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	if !p.expectPeek(token.OK) {
		return nil
	}
	if !p.expectPeek(token.LPAREN) {
		return nil
	}
	if !p.expectPeek(token.IDENT) {
		return nil
	}
	stmt.OkVariable = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	if !p.expectPeek(token.RPAREN) {
		return nil
	}
	if !p.expectPeek(token.ARROW) {
		return nil
	}
	if !p.expectPeek(token.LBRACE) {
		return nil
	}
	stmt.OkBody = p.parseBlockStatement()

	if !p.expectPeek(token.ERR) {
		return nil
	}
	if !p.expectPeek(token.LPAREN) {
		return nil
	}
	if !p.expectPeek(token.IDENT) {
		return nil
	}
	stmt.ErrVariable = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	if !p.expectPeek(token.RPAREN) {
		return nil
	}
	if !p.expectPeek(token.ARROW) {
		return nil
	}
	if !p.expectPeek(token.LBRACE) {
		return nil
	}
	stmt.ErrBody = p.parseBlockStatement()

	if !p.expectPeek(token.RBRACE) {
		return nil
	}

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
	if !p.expectPeek(token.RPAREN) {
		return nil
	}
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
	if !p.expectPeek(token.LPAREN) {
		return nil
	}
	p.nextToken()
	stmt.Condition = p.parseExpression(LOWEST)
	if !p.expectPeek(token.RPAREN) {
		return nil
	}
	if !p.expectPeek(token.LBRACE) {
		return nil
	}
	stmt.Consequence = p.parseBlockStatement()
	if p.peekToken.Type == token.ELSE {
		p.nextToken()
		if !p.expectPeek(token.LBRACE) {
			return nil
		}
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

func (p *Parser) parseLoopStatement() *ast.LoopStatement {
	stmt := &ast.LoopStatement{Token: p.curToken}
	if !p.expectPeek(token.IDENT) {
		return nil
	}
	stmt.Variable = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	if !p.expectPeek(token.IN) {
		return nil
	}
	p.nextToken()
	stmt.Iterable = p.parseExpression(LOWEST)
	if !p.expectPeek(token.LBRACE) {
		return nil
	}
	stmt.Body = p.parseBlockStatement()
	return stmt
}

func (p *Parser) parseTryCatchStatement() *ast.TryCatchStatement {
	stmt := &ast.TryCatchStatement{Token: p.curToken}
	if !p.expectPeek(token.LBRACE) {
		return nil
	}
	stmt.TryBody = p.parseBlockStatement()
	if !p.expectPeek(token.CATCH) {
		return nil
	}
	if !p.expectPeek(token.LPAREN) {
		return nil
	}
	if !p.expectPeek(token.IDENT) {
		return nil
	}
	stmt.CatchVariable = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	if !p.expectPeek(token.RPAREN) {
		return nil
	}
	if !p.expectPeek(token.LBRACE) {
		return nil
	}
	stmt.CatchBody = p.parseBlockStatement()
	return stmt
}

func (p *Parser) parseFSMacroStatement() *ast.FSMacroStatement {
	stmt := &ast.FSMacroStatement{Token: p.curToken}
	if !p.expectPeek(token.LPAREN) {
		return nil
	}
	stmt.Args = p.parseCallArguments()
	return stmt
}

func (p *Parser) parsePolyglotBlockStatement() *ast.PolyglotBlockStatement {
	stmt := &ast.PolyglotBlockStatement{Token: p.curToken, Code: p.curToken.Literal}
	return stmt
}

func (p *Parser) parseImportComponentStatement() *ast.ImportComponentStatement {
	stmt := &ast.ImportComponentStatement{Token: p.curToken}
	if !p.expectPeek(token.STRING) {
		return nil
	}
	stmt.Path = p.curToken.Literal
	if !p.expectPeek(token.AS) {
		return nil
	}
	if !p.expectPeek(token.IDENT) {
		return nil
	}
	stmt.Alias = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	return stmt
}

func (p *Parser) parseComponentDefinition() *ast.ComponentDefinition {
	stmt := &ast.ComponentDefinition{Token: p.curToken}
	if !p.expectPeek(token.IDENT) {
		return nil
	}
	stmt.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	if p.peekToken.Type == token.LPAREN {
		p.nextToken()
		stmt.Parameters = p.parseFunctionParameters()
	}
	if !p.expectPeek(token.LBRACE) {
		return nil
	}
	stmt.Body = p.parseBlockStatement()
	return stmt
}

func (p *Parser) parseWebBlockStatement() *ast.WebBlockStatement {
	stmt := &ast.WebBlockStatement{Token: p.curToken}
	if p.peekToken.Type == token.STRING {
		p.nextToken()
		stmt.Name = p.curToken.Literal
	}
	if !p.expectPeek(token.LBRACE) {
		return nil
	}
	stmt.Body = p.parseBlockStatement()
	return stmt
}

func (p *Parser) parsePathStatement() *ast.PathStatement {
	stmt := &ast.PathStatement{Token: p.curToken}
	if !p.expectPeek(token.LPAREN) {
		return nil
	}
	if !p.expectPeek(token.STRING) {
		return nil
	}
	stmt.Path = p.curToken.Literal
	if !p.expectPeek(token.RPAREN) {
		return nil
	}
	if p.peekToken.Type == token.ARROW {
		p.nextToken()
	}
	if !p.expectPeek(token.LBRACE) {
		return nil
	}
	stmt.Body = p.parseBlockStatement()
	return stmt
}

func (p *Parser) parsePathWsStatement() *ast.PathWsStatement {
	stmt := &ast.PathWsStatement{Token: p.curToken}
	if !p.expectPeek(token.LPAREN) {
		return nil
	}
	if !p.expectPeek(token.STRING) {
		return nil
	}
	stmt.Path = p.curToken.Literal
	if !p.expectPeek(token.RPAREN) {
		return nil
	}
	if p.peekToken.Type == token.ARROW {
		p.nextToken()
	}
	if !p.expectPeek(token.LBRACE) {
		return nil
	}
	stmt.Body = p.parseBlockStatement()
	return stmt
}

func (p *Parser) parseBeforeEachStatement() *ast.BeforeEachStatement {
	stmt := &ast.BeforeEachStatement{Token: p.curToken}
	if p.peekToken.Type == token.LPAREN {
		p.nextToken()
		if !p.expectPeek(token.RPAREN) {
			return nil
		}
	}
	if p.peekToken.Type == token.ARROW {
		p.nextToken()
	}
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
	case token.BACKTICK:
		leftExp = p.parseInterpolatedStringLiteral()
	case token.OK, token.ERR:
		leftExp = p.parseResultLiteral()
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
	if !p.expectPeek(token.IDENT) {
		return nil
	}
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
	if !p.expectPeek(end) {
		return nil
	}
	return list
}

func (p *Parser) parseInterpolatedStringLiteral() ast.Expression {
	lit := &ast.InterpolatedStringLiteral{Token: p.curToken}
	raw := p.curToken.Literal
	segments := []ast.Expression{}

	start := 0
	for {
		idx := strings.Index(raw[start:], "${")
		if idx == -1 {
			segments = append(segments, &ast.StringLiteral{Token: lit.Token, Value: raw[start:]})
			break
		}
		actualIdx := start + idx
		segments = append(segments, &ast.StringLiteral{Token: lit.Token, Value: raw[start:actualIdx]})

		endIdx := strings.Index(raw[actualIdx:], "}")
		if endIdx == -1 {
			segments = append(segments, &ast.StringLiteral{Token: lit.Token, Value: raw[actualIdx:]})
			break
		}
		exprStr := raw[actualIdx+2 : actualIdx+endIdx]
		// Sub-parser for the interpolation expression
		subL := lexer.New(exprStr)
		subP := New(subL)
		expr := subP.parseExpression(LOWEST)
		segments = append(segments, expr)

		start = actualIdx + endIdx + 1
	}
	lit.Segments = segments
	return lit
}

func (p *Parser) parseIndexExpression(left ast.Expression) ast.Expression {
	exp := &ast.IndexExpression{Token: p.curToken, Left: left}
	p.nextToken()
	exp.Index = p.parseExpression(LOWEST)
	if !p.expectPeek(token.RBRACKET) {
		return nil
	}
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
	if !p.expectPeek(token.RPAREN) {
		return nil
	}
	return args
}

func (p *Parser) peekPrecedence() int {
	if pre, ok := precedences[p.peekToken.Type]; ok {
		return pre
	}
	return LOWEST
}

func (p *Parser) curPrecedence() int {
	if pre, ok := precedences[p.curToken.Type]; ok {
		return pre
	}
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
