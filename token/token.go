package token

type TokenType string

type Token struct {
	Type    TokenType
	Literal string
	Line    int
	Column  int
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
	ARROW  = "->"

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

	FS_RM    = "FS_RM"
	FS_MV    = "FS_MV"
	FS_CP    = "FS_CP"
	FS_TOUCH = "FS_TOUCH"
	FS_CAT   = "FS_CAT"

	GO_BLOCK   = "GO_BLOCK"
	RUST_BLOCK = "RUST_BLOCK"
	JS_BLOCK   = "JS_BLOCK"
	PY_BLOCK   = "PY_BLOCK"

	IMPORT_COMPONENT = "IMPORT_COMPONENT"
	AS               = "AS"
	COMPONENT        = "COMPONENT"
	WEB_BLOCK        = "WEB_BLOCK"
	PATH             = "PATH"
	PATH_WS          = "PATH_WS"
	BEFORE_EACH      = "BEFORE_EACH"
	BACKTICK         = "BACKTICK"

	IMPORT = "IMPORT"
	EXPORT = "EXPORT"
	FROM   = "FROM"
	RESULT = "RESULT"
	OK     = "OK"
	ERR    = "ERR"
)

var keywords = map[string]TokenType{
	"spawn":            SPAWN,
	"print":            PRINT,
	"serve_html":       SERVE_HTML,
	"fetch_api":        FETCH_API,
	"db_save":          DB_SAVE,
	"db_get":           DB_GET,
	"connect_bot":      CONNECT_BOT,
	"interval":         INTERVAL,
	"file_write":       FILE_WRITE,
	"on":               ON,
	"emit":             EMIT,
	"if":               IF,
	"else":             ELSE,
	"true":             TRUE,
	"false":            FALSE,
	"loop":             LOOP,
	"in":               IN,
	"try":              TRY,
	"catch":            CATCH,
	"json_parse":       JSON_PARSE,
	"fn":               FN,
	"return":           RETURN,
	"get_addr":         GET_ADDR,
	"fs_rm":            FS_RM,
	"fs_mv":            FS_MV,
	"fs_cp":            FS_CP,
	"fs_touch":         FS_TOUCH,
	"fs_cat":           FS_CAT,
	"go_block":         GO_BLOCK,
	"rust_block":       RUST_BLOCK,
	"js_block":         JS_BLOCK,
	"py_block":         PY_BLOCK,
	"import_component": IMPORT_COMPONENT,
	"as":               AS,
	"component":        COMPONENT,
	"web_block":        WEB_BLOCK,
	"path":             PATH,
	"path_ws":          PATH_WS,
	"before_each":      BEFORE_EACH,
	"import":           IMPORT,
	"export":           EXPORT,
	"from":             FROM,
	"result":           RESULT,
	"ok":               OK,
	"err":              ERR,
}

func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}
