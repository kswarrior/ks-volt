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
	IDENT  = "IDENT"
	INT    = "INT"
	STRING = "STRING"

	// Operators
	ASSIGN = "="
	PLUS   = "+"
	DOT    = "."

	// Delimiters
	COMMA    = ","
	LPAREN   = "("
	RPAREN   = ")"
	LBRACE   = "{"
	RBRACE   = "}"
	LBRACKET = "["
	RBRACKET = "]"

	// Keywords
	SPAWN       = "SPAWN"
	PRINT       = "PRINT"
	SERVE_HTML  = "SERVE_HTML"
	FETCH_API   = "FETCH_API"
	DB_SAVE     = "DB_SAVE"
	DB_GET      = "DB_GET"
	CONNECT_BOT = "CONNECT_BOT"
	INTERVAL    = "INTERVAL"
	FILE_WRITE  = "FILE_WRITE"
	ON          = "ON"
	EMIT        = "EMIT"

	IF         = "IF"
	ELSE       = "ELSE"
	TRUE       = "TRUE"
	FALSE      = "FALSE"
	LOOP       = "LOOP"
	IN         = "IN"
	TRY        = "TRY"
	CATCH      = "CATCH"
	JSON_PARSE = "JSON_PARSE"
	FN         = "FN"
	RETURN     = "RETURN"
	GET_ADDR   = "GET_ADDR"
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
	"if":          IF,
	"else":        ELSE,
	"true":        TRUE,
	"false":       FALSE,
	"loop":        LOOP,
	"in":          IN,
	"try":         TRY,
	"catch":       CATCH,
	"json_parse":  JSON_PARSE,
	"fn":          FN,
	"return":      RETURN,
	"get_addr":    GET_ADDR,
}

func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}
