package token

type TokenType string

type Token struct {
	Type    TokenType
	Literal string
}

const (
	ILLEGAL = "ILLEGAL"
	EOF     = "EOF"

	// Identifiers + literals
	IDENT  = "IDENT"  // PORT, start_web_ui
	INT    = "INT"    // 8080
	STRING = "STRING" // "⚡ KS-Volt Web UI server is spinning up..."

	// Operators
	ASSIGN = "="

	// Delimiters
	COMMA  = ","
	LPAREN = "("
	RPAREN = ")"
	LBRACE = "{"
	RBRACE = "}"

	// Keywords
	SPAWN      = "SPAWN"
	PRINT      = "PRINT"
	SERVE_HTML = "SERVE_HTML"
)

var keywords = map[string]TokenType{
	"spawn":      SPAWN,
	"print":      PRINT,
	"serve_html": SERVE_HTML,
}

func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}
