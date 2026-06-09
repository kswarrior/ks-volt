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
	PLUS   = "+"

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
	FETCH_API  = "FETCH_API"
	DB_SAVE    = "DB_SAVE"
	DB_GET      = "DB_GET"
	CONNECT_BOT = "CONNECT_BOT"
	INTERVAL    = "INTERVAL"
	FILE_WRITE  = "FILE_WRITE"
	ON          = "ON"
	EMIT        = "EMIT"
)

var keywords = map[string]TokenType{
	"spawn":       SPAWN,
	"print":       PRINT,
	"serve_html":  SERVE_HTML,
	"fetch_api":   FETCH_API,
	"db_save":     DB_SAVE,
	"db_get":      DB_GET,
	"connect_bot": CONNECT_BOT,
	"interval":    INTERVAL,
	"file_write":  FILE_WRITE,
	"on":          ON,
	"emit":        EMIT,
}

func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}
