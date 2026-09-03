package lexer

import "testing"

func TestNextToken(t *testing.T) {
	input := `contract Counter {
    uint count;
    // a comment
    function get() public view returns (uint256) {
        switch x {
            case 1: { return count; }
            default: { return 0; }
        }
        if (a <= b) { count = count + 1; }
    }
}`

	want := []struct {
		typ TokenType
		lit string
	}{
		{CONTRACT, "contract"}, {IDENT, "Counter"}, {LBRACE, "{"},
		{TYPE, "uint"}, {IDENT, "count"}, {SEMI, ";"},
		{FUNCTION, "function"}, {IDENT, "get"}, {LPAREN, "("}, {RPAREN, ")"},
		{PUBLIC, "public"}, {VIEW, "view"}, {RETURNS, "returns"},
		{LPAREN, "("}, {TYPE, "uint256"}, {RPAREN, ")"}, {LBRACE, "{"},
		{SWITCH, "switch"}, {IDENT, "x"}, {LBRACE, "{"},
		{CASE, "case"}, {INT, "1"}, {COLON, ":"}, {LBRACE, "{"},
		{RETURN, "return"}, {IDENT, "count"}, {SEMI, ";"}, {RBRACE, "}"},
		{DEFAULT, "default"}, {COLON, ":"}, {LBRACE, "{"},
		{RETURN, "return"}, {INT, "0"}, {SEMI, ";"}, {RBRACE, "}"},
		{RBRACE, "}"},
		{IF, "if"}, {LPAREN, "("}, {IDENT, "a"}, {LTE, "<="}, {IDENT, "b"}, {RPAREN, ")"},
		{LBRACE, "{"}, {IDENT, "count"}, {ASSIGN, "="},
		{IDENT, "count"}, {PLUS, "+"}, {INT, "1"}, {SEMI, ";"}, {RBRACE, "}"},
		{RBRACE, "}"},
		{RBRACE, "}"},
		{EOF, ""},
	}

	l := New(input)
	for i, w := range want {
		got := l.NextToken()
		if got.Type != w.typ {
			t.Fatalf("token %d: type = %v (%q), want %v (%q)", i, got.Type, got.Literal, w.typ, w.lit)
		}
		if got.Literal != w.lit {
			t.Fatalf("token %d: literal = %q, want %q", i, got.Literal, w.lit)
		}
	}
}

func TestMappingAndStructTokens(t *testing.T) {
	input := `struct S { uint x; }
mapping(address => uint) m;
m[k].field = 1;`

	want := []struct {
		typ TokenType
		lit string
	}{
		{STRUCT, "struct"}, {IDENT, "S"}, {LBRACE, "{"},
		{TYPE, "uint"}, {IDENT, "x"}, {SEMI, ";"}, {RBRACE, "}"},
		{MAPPING, "mapping"}, {LPAREN, "("}, {TYPE, "address"},
		{ARROW, "=>"}, {TYPE, "uint"}, {RPAREN, ")"}, {IDENT, "m"}, {SEMI, ";"},
		{IDENT, "m"}, {LBRACKET, "["}, {IDENT, "k"}, {RBRACKET, "]"},
		{DOT, "."}, {IDENT, "field"}, {ASSIGN, "="}, {INT, "1"}, {SEMI, ";"},
		{EOF, ""},
	}

	l := New(input)
	for i, w := range want {
		got := l.NextToken()
		if got.Type != w.typ || got.Literal != w.lit {
			t.Fatalf("token %d: got %v (%q), want %v (%q)", i, got.Type, got.Literal, w.typ, w.lit)
		}
	}
}

func TestEventAndEmitTokens(t *testing.T) {
	input := `event E(address indexed a, uint b);
emit E(x, y);`

	want := []struct {
		typ TokenType
		lit string
	}{
		{EVENT, "event"}, {IDENT, "E"}, {LPAREN, "("},
		{TYPE, "address"}, {INDEXED, "indexed"}, {IDENT, "a"}, {COMMA, ","},
		{TYPE, "uint"}, {IDENT, "b"}, {RPAREN, ")"}, {SEMI, ";"},
		{EMIT, "emit"}, {IDENT, "E"}, {LPAREN, "("},
		{IDENT, "x"}, {COMMA, ","}, {IDENT, "y"}, {RPAREN, ")"}, {SEMI, ";"},
		{EOF, ""},
	}

	l := New(input)
	for i, w := range want {
		got := l.NextToken()
		if got.Type != w.typ || got.Literal != w.lit {
			t.Fatalf("token %d: got %v (%q), want %v (%q)", i, got.Type, got.Literal, w.typ, w.lit)
		}
	}
}
