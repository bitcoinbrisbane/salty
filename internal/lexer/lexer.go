package lexer

// Lexer scans Salty source into a stream of tokens. It is hand-written and
// operates over the input as a slice of runes so column tracking is correct
// for multi-byte characters.
type Lexer struct {
	input []rune
	pos   int // index of the current rune
	line  int
	col   int
}

// New returns a Lexer ready to tokenize src.
func New(src string) *Lexer {
	return &Lexer{input: []rune(src), pos: 0, line: 1, col: 1}
}

func (l *Lexer) peek() rune {
	if l.pos >= len(l.input) {
		return 0
	}
	return l.input[l.pos]
}

func (l *Lexer) peekNext() rune {
	if l.pos+1 >= len(l.input) {
		return 0
	}
	return l.input[l.pos+1]
}

// advance consumes the current rune and updates line/col.
func (l *Lexer) advance() rune {
	r := l.input[l.pos]
	l.pos++
	if r == '\n' {
		l.line++
		l.col = 1
	} else {
		l.col++
	}
	return r
}

func (l *Lexer) skipWhitespaceAndComments() {
	for l.pos < len(l.input) {
		r := l.peek()
		switch {
		case r == ' ' || r == '\t' || r == '\r' || r == '\n':
			l.advance()
		case r == '/' && l.peekNext() == '/':
			for l.pos < len(l.input) && l.peek() != '\n' {
				l.advance()
			}
		case r == '/' && l.peekNext() == '*':
			l.advance() // /
			l.advance() // *
			for l.pos < len(l.input) && !(l.peek() == '*' && l.peekNext() == '/') {
				l.advance()
			}
			if l.pos < len(l.input) {
				l.advance() // *
				l.advance() // /
			}
		default:
			return
		}
	}
}

// NextToken returns the next token in the stream. It returns an EOF token
// repeatedly once the input is exhausted.
func (l *Lexer) NextToken() Token {
	l.skipWhitespaceAndComments()

	line, col := l.line, l.col

	if l.pos >= len(l.input) {
		return Token{Type: EOF, Literal: "", Line: line, Col: col}
	}

	r := l.peek()

	switch {
	case isLetter(r):
		lit := l.readIdentifier()
		return Token{Type: lookupIdent(lit), Literal: lit, Line: line, Col: col}
	case isDigit(r):
		lit, typ := l.readNumber()
		return Token{Type: typ, Literal: lit, Line: line, Col: col}
	case r == '"':
		lit := l.readString()
		return Token{Type: STRING, Literal: lit, Line: line, Col: col}
	}

	return l.readSymbol(line, col)
}

// readString reads a double-quoted string literal, returning the contents
// without the surrounding quotes. Supports \" and \\ escapes.
func (l *Lexer) readString() string {
	l.advance() // opening quote
	var sb []rune
	for l.pos < len(l.input) && l.peek() != '"' {
		r := l.advance()
		if r == '\\' && l.pos < len(l.input) {
			sb = append(sb, r, l.advance())
			continue
		}
		sb = append(sb, r)
	}
	if l.pos < len(l.input) {
		l.advance() // closing quote
	}
	return string(sb)
}

func (l *Lexer) readIdentifier() string {
	start := l.pos
	for l.pos < len(l.input) && (isLetter(l.peek()) || isDigit(l.peek())) {
		l.advance()
	}
	return string(l.input[start:l.pos])
}

// readNumber reads an integer or a decimal literal. A '.' immediately followed
// by a digit begins a fractional part and makes the token a DECIMAL (SIP-2);
// otherwise the token is an INT and a following '.' is left for the scanner
// (e.g. member access like msg.sender).
func (l *Lexer) readNumber() (string, TokenType) {
	start := l.pos
	for l.pos < len(l.input) && isDigit(l.peek()) {
		l.advance()
	}
	typ := INT
	if l.peek() == '.' && isDigit(l.peekNext()) {
		typ = DECIMAL
		l.advance() // '.'
		for l.pos < len(l.input) && isDigit(l.peek()) {
			l.advance()
		}
	}
	return string(l.input[start:l.pos]), typ
}

// readSymbol handles punctuation and operators, including two-character
// operators like == and <=.
func (l *Lexer) readSymbol(line, col int) Token {
	r := l.advance()
	tok := func(t TokenType, lit string) Token {
		return Token{Type: t, Literal: lit, Line: line, Col: col}
	}

	switch r {
	case '{':
		return tok(LBRACE, "{")
	case '}':
		return tok(RBRACE, "}")
	case '(':
		return tok(LPAREN, "(")
	case ')':
		return tok(RPAREN, ")")
	case '[':
		return tok(LBRACKET, "[")
	case ']':
		return tok(RBRACKET, "]")
	case ';':
		return tok(SEMI, ";")
	case ',':
		return tok(COMMA, ",")
	case ':':
		return tok(COLON, ":")
	case '.':
		return tok(DOT, ".")
	case '+':
		return tok(PLUS, "+")
	case '-':
		return tok(MINUS, "-")
	case '*':
		return tok(STAR, "*")
	case '/':
		return tok(SLASH, "/")
	case '=':
		if l.peek() == '=' {
			l.advance()
			return tok(EQ, "==")
		}
		if l.peek() == '>' {
			l.advance()
			return tok(ARROW, "=>")
		}
		return tok(ASSIGN, "=")
	case '!':
		if l.peek() == '=' {
			l.advance()
			return tok(NEQ, "!=")
		}
		return tok(ILLEGAL, "!")
	case '<':
		if l.peek() == '=' {
			l.advance()
			return tok(LTE, "<=")
		}
		return tok(LT, "<")
	case '>':
		if l.peek() == '=' {
			l.advance()
			return tok(GTE, ">=")
		}
		return tok(GT, ">")
	}

	return tok(ILLEGAL, string(r))
}

func isLetter(r rune) bool {
	return r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}
