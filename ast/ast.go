package ast

import (
	"ks-volt/token"
)

type Node interface {
	TokenLiteral() string
}

type Statement interface {
	Node
	statementNode()
}

type Expression interface {
	Node
	expressionNode()
}

type Program struct {
	Statements []Statement
}

func (p *Program) TokenLiteral() string {
	if len(p.Statements) > 0 {
		return p.Statements[0].TokenLiteral()
	}
	return ""
}

type AssignmentStatement struct {
	Token token.Token
	Name  *Identifier
	Value Expression
}

func (as *AssignmentStatement) statementNode()       {}
func (as *AssignmentStatement) TokenLiteral() string { return as.Token.Literal }

type FunctionStatement struct {
	Token      token.Token // fn
	Name       *Identifier
	Parameters []*Identifier
	Body       *BlockStatement
}

func (fs *FunctionStatement) statementNode()       {}
func (fs *FunctionStatement) TokenLiteral() string { return fs.Token.Literal }

type ReturnStatement struct {
	Token       token.Token // return
	ReturnValue Expression
}

func (rs *ReturnStatement) statementNode()       {}
func (rs *ReturnStatement) TokenLiteral() string { return rs.Token.Literal }

type IfStatement struct {
	Token       token.Token // if
	Condition   Expression
	Consequence *BlockStatement
	Alternative *BlockStatement
}

func (is *IfStatement) statementNode()       {}
func (is *IfStatement) TokenLiteral() string { return is.Token.Literal }

type LoopStatement struct {
	Token    token.Token // loop
	Variable *Identifier
	Iterable Expression
	Body     *BlockStatement
}

func (ls *LoopStatement) statementNode()       {}
func (ls *LoopStatement) TokenLiteral() string { return ls.Token.Literal }

type TryCatchStatement struct {
	Token         token.Token // try
	TryBody       *BlockStatement
	CatchVariable *Identifier
	CatchBody     *BlockStatement
}

func (ts *TryCatchStatement) statementNode()       {}
func (ts *TryCatchStatement) TokenLiteral() string { return ts.Token.Literal }

type ExpressionStatement struct {
	Token      token.Token
	Expression Expression
}

func (es *ExpressionStatement) statementNode()       {}
func (es *ExpressionStatement) TokenLiteral() string { return es.Token.Literal }

type BlockStatement struct {
	Token      token.Token
	Statements []Statement
}

func (bs *BlockStatement) statementNode()       {}
func (bs *BlockStatement) TokenLiteral() string { return bs.Token.Literal }

type SpawnStatement struct {
	Token token.Token
	Name  *Identifier
	Args  []Expression
	Body  *BlockStatement
}

func (ss *SpawnStatement) statementNode()       {}
func (ss *SpawnStatement) TokenLiteral() string { return ss.Token.Literal }

type Identifier struct {
	Token token.Token
	Value string
}

func (i *Identifier) expressionNode()      {}
func (i *Identifier) TokenLiteral() string { return i.Token.Literal }

type IntegerLiteral struct {
	Token token.Token
	Value int64
}

func (il *IntegerLiteral) expressionNode()      {}
func (il *IntegerLiteral) TokenLiteral() string { return il.Token.Literal }

type StringLiteral struct {
	Token token.Token
	Value string
}

func (sl *StringLiteral) expressionNode()      {}
func (sl *StringLiteral) TokenLiteral() string { return sl.Token.Literal }

type Boolean struct {
	Token token.Token
	Value bool
}

func (b *Boolean) expressionNode()      {}
func (b *Boolean) TokenLiteral() string { return b.Token.Literal }

type ArrayLiteral struct {
	Token    token.Token // [
	Elements []Expression
}

func (al *ArrayLiteral) expressionNode()      {}
func (al *ArrayLiteral) TokenLiteral() string { return al.Token.Literal }

type InfixExpression struct {
	Token    token.Token
	Left     Expression
	Operator string
	Right    Expression
}

func (ie *InfixExpression) expressionNode()      {}
func (ie *InfixExpression) TokenLiteral() string { return ie.Token.Literal }

type CallExpression struct {
	Token     token.Token
	Function  Expression
	Arguments []Expression
}

func (ce *CallExpression) expressionNode()      {}
func (ce *CallExpression) TokenLiteral() string { return ce.Token.Literal }

type MethodCallExpression struct {
	Token     token.Token // .
	Object    Expression
	Method    *Identifier
	Arguments []Expression
}

func (mce *MethodCallExpression) expressionNode()      {}
func (mce *MethodCallExpression) TokenLiteral() string { return mce.Token.Literal }

type IndexExpression struct {
	Token token.Token // [
	Left  Expression
	Index Expression
}

func (ie *IndexExpression) expressionNode()      {}
func (ie *IndexExpression) TokenLiteral() string { return ie.Token.Literal }

type FSMacroStatement struct {
	Token token.Token
	Args  []Expression
}

func (fs *FSMacroStatement) statementNode()       {}
func (fs *FSMacroStatement) TokenLiteral() string { return fs.Token.Literal }

type PolyglotBlockStatement struct {
	Token token.Token
	Code  string
}

func (ps *PolyglotBlockStatement) statementNode()       {}
func (ps *PolyglotBlockStatement) TokenLiteral() string { return ps.Token.Literal }

type ImportComponentStatement struct {
	Token token.Token
	Path  string
	Alias *Identifier
}

func (is *ImportComponentStatement) statementNode()       {}
func (is *ImportComponentStatement) TokenLiteral() string { return is.Token.Literal }

type ComponentDefinition struct {
	Token      token.Token
	Name       *Identifier
	Parameters []*Identifier
	Body       *BlockStatement
}

func (cd *ComponentDefinition) statementNode()       {}
func (cd *ComponentDefinition) TokenLiteral() string { return cd.Token.Literal }

type WebBlockStatement struct {
	Token token.Token
	Name  string // Optional
	Body  *BlockStatement
}

func (ws *WebBlockStatement) statementNode()       {}
func (ws *WebBlockStatement) TokenLiteral() string { return ws.Token.Literal }

type PathStatement struct {
	Token token.Token
	Path  string
	Body  *BlockStatement
}

func (ps *PathStatement) statementNode()       {}
func (ps *PathStatement) TokenLiteral() string { return ps.Token.Literal }

type PathWsStatement struct {
	Token token.Token
	Path  string
	Body  *BlockStatement
}

func (ps *PathWsStatement) statementNode()       {}
func (ps *PathWsStatement) TokenLiteral() string { return ps.Token.Literal }

type BeforeEachStatement struct {
	Token token.Token
	Body  *BlockStatement
}

func (bs *BeforeEachStatement) statementNode()       {}
func (bs *BeforeEachStatement) TokenLiteral() string { return bs.Token.Literal }

type InterpolatedStringLiteral struct {
	Token    token.Token
	Segments []Expression // alternating StringLiteral and Expressions
}

func (is *InterpolatedStringLiteral) expressionNode()      {}
func (is *InterpolatedStringLiteral) TokenLiteral() string { return is.Token.Literal }
