package lexer

// TokenType enumerates the kinds of lexical tokens Salty recognizes.
type TokenType int

const (
	ILLEGAL TokenType = iota
	EOF

	// Identifiers and literals.
	IDENT // foo, bar, count
	INT   // 123
	// Keywords / built-in type names.
	CONTRACT
	FUNCTION
	PUBLIC
	PRIVATE
	INTERNAL
	EXTERNAL
	VIEW
	PURE
	RETURNS
	RETURN
	IF
	ELSE
	SWITCH
	CASE
	DEFAULT
	MAPPING
	STRUCT
	TYPE // a type name such as uint256, uint, int, bool, address
	TRUE
	FALSE

	// Punctuation.
	LBRACE   // {
	RBRACE   // }
	LPAREN   // (
	RPAREN   // )
	LBRACKET // [
	RBRACKET // ]
	SEMI     // ;
	COMMA    // ,
	COLON    // :
	DOT      // .
	ASSIGN   // =
	ARROW    // =>

	// Operators.
	PLUS  // +
	MINUS // -
	STAR  // *
	SLASH // /
	EQ    // ==
	NEQ   // !=
	LT    // <
	GT    // >
	LTE   // <=
	GTE   // >=
)

// Token is a single lexical unit with its source position.
type Token struct {
	Type    TokenType
	Literal string
	Line    int
	Col     int
}

// keywords maps reserved words to their token type. Built-in type names are
// tagged as TYPE so the parser can treat them uniformly.
var keywords = map[string]TokenType{
	"contract": CONTRACT,
	"function": FUNCTION,
	"public":   PUBLIC,
	"private":  PRIVATE,
	"internal": INTERNAL,
	"external": EXTERNAL,
	"view":     VIEW,
	"pure":     PURE,
	"returns":  RETURNS,
	"return":   RETURN,
	"if":       IF,
	"else":     ELSE,
	"switch":   SWITCH,
	"case":     CASE,
	"default":  DEFAULT,
	"mapping":  MAPPING,
	"struct":   STRUCT,
	"true":     TRUE,
	"false":    FALSE,

	// Built-in types.
	"uint256": TYPE,
	"uint":    TYPE,
	"int256":  TYPE,
	"int":     TYPE,
	"bool":    TYPE,
	"address": TYPE,
	"string":  TYPE,
}

// lookupIdent returns the keyword token type for ident, or IDENT if it is not
// a reserved word.
func lookupIdent(ident string) TokenType {
	if t, ok := keywords[ident]; ok {
		return t
	}
	return IDENT
}

// String returns a human-readable name for the token type, used in errors.
func (t TokenType) String() string {
	switch t {
	case ILLEGAL:
		return "ILLEGAL"
	case EOF:
		return "EOF"
	case IDENT:
		return "IDENT"
	case INT:
		return "INT"
	case TYPE:
		return "TYPE"
	}
	if name, ok := tokenNames[t]; ok {
		return name
	}
	return "TOKEN"
}

var tokenNames = map[TokenType]string{
	CONTRACT: "contract", FUNCTION: "function", PUBLIC: "public",
	PRIVATE: "private", INTERNAL: "internal", EXTERNAL: "external",
	VIEW: "view", PURE: "pure", RETURNS: "returns", RETURN: "return",
	IF: "if", ELSE: "else", SWITCH: "switch", CASE: "case", DEFAULT: "default",
	MAPPING: "mapping", STRUCT: "struct", TRUE: "true", FALSE: "false",
	LBRACE: "{", RBRACE: "}", LPAREN: "(", RPAREN: ")",
	LBRACKET: "[", RBRACKET: "]", SEMI: ";",
	COMMA: ",", COLON: ":", DOT: ".", ASSIGN: "=", ARROW: "=>",
	PLUS: "+", MINUS: "-", STAR: "*", SLASH: "/",
	EQ: "==", NEQ: "!=", LT: "<", GT: ">", LTE: "<=", GTE: ">=",
}
