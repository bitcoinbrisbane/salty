// Package parser turns a Salty token stream into an AST. It is a hand-written
// recursive-descent parser with Pratt-style expression parsing.
package parser

import (
	"fmt"

	"github.com/bitcoinbrisbane/salty/internal/ast"
	"github.com/bitcoinbrisbane/salty/internal/lexer"
)

// Parser holds the parsing state.
type Parser struct {
	lex  *lexer.Lexer
	cur  lexer.Token
	next lexer.Token
}

// New returns a Parser over src.
func New(src string) *Parser {
	p := &Parser{lex: lexer.New(src)}
	// Prime cur and next.
	p.advance()
	p.advance()
	return p
}

// ParseFile parses src into a *ast.File.
func ParseFile(src string) (*ast.File, error) {
	return New(src).ParseFile()
}

func (p *Parser) advance() {
	p.cur = p.next
	p.next = p.lex.NextToken()
}

func (p *Parser) expect(t lexer.TokenType) (lexer.Token, error) {
	if p.cur.Type != t {
		return p.cur, p.errf("expected %v, got %q", t, p.cur.Literal)
	}
	tok := p.cur
	p.advance()
	return tok, nil
}

func (p *Parser) errf(format string, args ...any) error {
	return fmt.Errorf("parse error at line %d, col %d: %s",
		p.cur.Line, p.cur.Col, fmt.Sprintf(format, args...))
}

// ParseFile is the top-level entry: zero or more contracts until EOF.
func (p *Parser) ParseFile() (*ast.File, error) {
	file := &ast.File{}
	for p.cur.Type != lexer.EOF {
		if p.cur.Type != lexer.CONTRACT {
			return nil, p.errf("expected 'contract', got %q", p.cur.Literal)
		}
		c, err := p.parseContract()
		if err != nil {
			return nil, err
		}
		file.Contracts = append(file.Contracts, c)
	}
	return file, nil
}

func (p *Parser) parseContract() (*ast.Contract, error) {
	if _, err := p.expect(lexer.CONTRACT); err != nil {
		return nil, err
	}
	name, err := p.expect(lexer.IDENT)
	if err != nil {
		return nil, err
	}
	c := &ast.Contract{Name: name.Literal}
	if _, err := p.expect(lexer.LBRACE); err != nil {
		return nil, err
	}
	for p.cur.Type != lexer.RBRACE && p.cur.Type != lexer.EOF {
		m, err := p.parseMember()
		if err != nil {
			return nil, err
		}
		c.Members = append(c.Members, m)
	}
	if _, err := p.expect(lexer.RBRACE); err != nil {
		return nil, err
	}
	return c, nil
}

// parseMember parses a function, struct, or state variable. Functions start
// with `function`, structs with `struct`, and state variables with a type
// (an elementary type, `mapping`, or a struct name used as a type).
func (p *Parser) parseMember() (ast.Node, error) {
	switch p.cur.Type {
	case lexer.FUNCTION:
		return p.parseFunction()
	case lexer.STRUCT:
		return p.parseStruct()
	case lexer.EVENT:
		return p.parseEvent()
	case lexer.TYPE, lexer.MAPPING, lexer.IDENT:
		return p.parseStateVar()
	}
	return nil, p.errf("expected 'function', 'struct', 'event', or a type, got %q", p.cur.Literal)
}

// parseEvent parses:
//
//	event Name(<type> [indexed] [name], ...);
func (p *Parser) parseEvent() (*ast.Event, error) {
	if _, err := p.expect(lexer.EVENT); err != nil {
		return nil, err
	}
	name, err := p.expect(lexer.IDENT)
	if err != nil {
		return nil, err
	}
	ev := &ast.Event{Name: name.Literal}
	if _, err := p.expect(lexer.LPAREN); err != nil {
		return nil, err
	}
	for p.cur.Type != lexer.RPAREN {
		ptyp, err := p.parseType()
		if err != nil {
			return nil, err
		}
		param := &ast.EventParam{Type: ptyp}
		if p.cur.Type == lexer.INDEXED {
			param.Indexed = true
			p.advance()
		}
		// The parameter name is optional.
		if p.cur.Type == lexer.IDENT {
			param.Name = p.cur.Literal
			p.advance()
		}
		ev.Params = append(ev.Params, param)
		if p.cur.Type == lexer.COMMA {
			p.advance()
		}
	}
	if _, err := p.expect(lexer.RPAREN); err != nil {
		return nil, err
	}
	if _, err := p.expect(lexer.SEMI); err != nil {
		return nil, err
	}
	return ev, nil
}

// parseType parses a type reference: an elementary type (TYPE), a struct name
// (IDENT), or a mapping(K => V).
func (p *Parser) parseType() (ast.Type, error) {
	switch p.cur.Type {
	case lexer.TYPE, lexer.IDENT:
		name := p.cur.Literal
		p.advance()
		return ast.Type{Name: name}, nil
	case lexer.MAPPING:
		return p.parseMappingType()
	}
	return ast.Type{}, p.errf("expected a type, got %q", p.cur.Literal)
}

// parseMappingType parses mapping(<type> => <type>). The value type may itself
// be a mapping, allowing nested mappings.
func (p *Parser) parseMappingType() (ast.Type, error) {
	if _, err := p.expect(lexer.MAPPING); err != nil {
		return ast.Type{}, err
	}
	if _, err := p.expect(lexer.LPAREN); err != nil {
		return ast.Type{}, err
	}
	key, err := p.parseType()
	if err != nil {
		return ast.Type{}, err
	}
	if _, err := p.expect(lexer.ARROW); err != nil {
		return ast.Type{}, err
	}
	val, err := p.parseType()
	if err != nil {
		return ast.Type{}, err
	}
	if _, err := p.expect(lexer.RPAREN); err != nil {
		return ast.Type{}, err
	}
	return ast.Type{Key: &key, Value: &val}, nil
}

func (p *Parser) parseStruct() (*ast.Struct, error) {
	if _, err := p.expect(lexer.STRUCT); err != nil {
		return nil, err
	}
	name, err := p.expect(lexer.IDENT)
	if err != nil {
		return nil, err
	}
	s := &ast.Struct{Name: name.Literal}
	if _, err := p.expect(lexer.LBRACE); err != nil {
		return nil, err
	}
	for p.cur.Type != lexer.RBRACE && p.cur.Type != lexer.EOF {
		ftyp, err := p.parseType()
		if err != nil {
			return nil, err
		}
		fname, err := p.expect(lexer.IDENT)
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(lexer.SEMI); err != nil {
			return nil, err
		}
		s.Fields = append(s.Fields, &ast.Field{Type: ftyp, Name: fname.Literal})
	}
	if _, err := p.expect(lexer.RBRACE); err != nil {
		return nil, err
	}
	return s, nil
}

func (p *Parser) parseStateVar() (*ast.StateVar, error) {
	typ, err := p.parseType()
	if err != nil {
		return nil, err
	}
	name, err := p.expect(lexer.IDENT)
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(lexer.SEMI); err != nil {
		return nil, err
	}
	return &ast.StateVar{Type: typ, Name: name.Literal}, nil
}

func (p *Parser) parseFunction() (*ast.Function, error) {
	if _, err := p.expect(lexer.FUNCTION); err != nil {
		return nil, err
	}
	name, err := p.expect(lexer.IDENT)
	if err != nil {
		return nil, err
	}
	fn := &ast.Function{Name: name.Literal}

	// Params.
	if _, err := p.expect(lexer.LPAREN); err != nil {
		return nil, err
	}
	for p.cur.Type != lexer.RPAREN {
		ptyp, err := p.parseType()
		if err != nil {
			return nil, err
		}
		pname, err := p.expect(lexer.IDENT)
		if err != nil {
			return nil, err
		}
		fn.Params = append(fn.Params, &ast.Param{Type: ptyp, Name: pname.Literal})
		if p.cur.Type == lexer.COMMA {
			p.advance()
		}
	}
	if _, err := p.expect(lexer.RPAREN); err != nil {
		return nil, err
	}

	// Modifiers: visibility and mutability, in any order.
	for {
		switch p.cur.Type {
		case lexer.PUBLIC, lexer.PRIVATE, lexer.INTERNAL, lexer.EXTERNAL:
			fn.Visibility = p.cur.Literal
			p.advance()
		case lexer.VIEW, lexer.PURE:
			fn.Mutability = p.cur.Literal
			p.advance()
		default:
			goto modifiersDone
		}
	}
modifiersDone:

	// Optional returns.
	if p.cur.Type == lexer.RETURNS {
		p.advance()
		if _, err := p.expect(lexer.LPAREN); err != nil {
			return nil, err
		}
		for p.cur.Type != lexer.RPAREN {
			rtyp, err := p.parseType()
			if err != nil {
				return nil, err
			}
			fn.Returns = append(fn.Returns, rtyp)
			if p.cur.Type == lexer.COMMA {
				p.advance()
			}
		}
		if _, err := p.expect(lexer.RPAREN); err != nil {
			return nil, err
		}
	}

	body, err := p.parseBlock()
	if err != nil {
		return nil, err
	}
	fn.Body = body
	return fn, nil
}

func (p *Parser) parseBlock() (*ast.Block, error) {
	if _, err := p.expect(lexer.LBRACE); err != nil {
		return nil, err
	}
	blk := &ast.Block{}
	for p.cur.Type != lexer.RBRACE && p.cur.Type != lexer.EOF {
		s, err := p.parseStatement()
		if err != nil {
			return nil, err
		}
		blk.Statements = append(blk.Statements, s)
	}
	if _, err := p.expect(lexer.RBRACE); err != nil {
		return nil, err
	}
	return blk, nil
}

func (p *Parser) parseStatement() (ast.Statement, error) {
	switch p.cur.Type {
	case lexer.RETURN:
		return p.parseReturn()
	case lexer.IF:
		return p.parseIf()
	case lexer.SWITCH:
		return p.parseSwitch()
	case lexer.EMIT:
		return p.parseEmit()
	case lexer.TYPE, lexer.MAPPING:
		return p.parseVarDecl()
	case lexer.IDENT:
		// A struct-typed local declaration looks like `Account a` — an
		// identifier (the type) immediately followed by another identifier
		// (the name). Anything else starting with an identifier is an
		// expression or assignment.
		if p.next.Type == lexer.IDENT {
			return p.parseVarDecl()
		}
		return p.parseExprOrAssign()
	default:
		return p.parseExprOrAssign()
	}
}

func (p *Parser) parseReturn() (ast.Statement, error) {
	p.advance() // return
	if p.cur.Type == lexer.SEMI {
		p.advance()
		return &ast.ReturnStmt{}, nil
	}
	val, err := p.parseExpression(lowest)
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(lexer.SEMI); err != nil {
		return nil, err
	}
	return &ast.ReturnStmt{Value: val}, nil
}

// parseEmit parses `emit Name(args...);`. The name and argument list are parsed
// as an ordinary call expression, which must resolve to a CallExpr.
func (p *Parser) parseEmit() (ast.Statement, error) {
	p.advance() // emit
	expr, err := p.parseExpression(lowest)
	if err != nil {
		return nil, err
	}
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return nil, p.errf("expected an event call after 'emit'")
	}
	if _, err := p.expect(lexer.SEMI); err != nil {
		return nil, err
	}
	return &ast.EmitStmt{Call: call}, nil
}

func (p *Parser) parseVarDecl() (ast.Statement, error) {
	typ, err := p.parseType()
	if err != nil {
		return nil, err
	}
	name, err := p.expect(lexer.IDENT)
	if err != nil {
		return nil, err
	}
	decl := &ast.VarDeclStmt{Type: typ, Name: name.Literal}
	if p.cur.Type == lexer.ASSIGN {
		p.advance()
		val, err := p.parseExpression(lowest)
		if err != nil {
			return nil, err
		}
		decl.Value = val
	}
	if _, err := p.expect(lexer.SEMI); err != nil {
		return nil, err
	}
	return decl, nil
}

func (p *Parser) parseIf() (ast.Statement, error) {
	p.advance() // if
	if _, err := p.expect(lexer.LPAREN); err != nil {
		return nil, err
	}
	cond, err := p.parseExpression(lowest)
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(lexer.RPAREN); err != nil {
		return nil, err
	}
	then, err := p.parseBlock()
	if err != nil {
		return nil, err
	}
	stmt := &ast.IfStmt{Cond: cond, Then: then}
	if p.cur.Type == lexer.ELSE {
		p.advance()
		if p.cur.Type == lexer.IF {
			elseIf, err := p.parseIf()
			if err != nil {
				return nil, err
			}
			stmt.Else = elseIf
		} else {
			elseBlk, err := p.parseBlock()
			if err != nil {
				return nil, err
			}
			stmt.Else = elseBlk
		}
	}
	return stmt, nil
}

// parseSwitch parses:
//
//	switch <expr> {
//	    case <expr>: { ... }
//	    default: { ... }
//	}
func (p *Parser) parseSwitch() (ast.Statement, error) {
	p.advance() // switch
	subject, err := p.parseExpression(lowest)
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(lexer.LBRACE); err != nil {
		return nil, err
	}
	sw := &ast.SwitchStmt{Subject: subject}
	for p.cur.Type != lexer.RBRACE && p.cur.Type != lexer.EOF {
		switch p.cur.Type {
		case lexer.CASE:
			p.advance()
			val, err := p.parseExpression(lowest)
			if err != nil {
				return nil, err
			}
			if _, err := p.expect(lexer.COLON); err != nil {
				return nil, err
			}
			body, err := p.parseBlock()
			if err != nil {
				return nil, err
			}
			sw.Cases = append(sw.Cases, &ast.SwitchCase{Value: val, Body: body})
		case lexer.DEFAULT:
			p.advance()
			if _, err := p.expect(lexer.COLON); err != nil {
				return nil, err
			}
			body, err := p.parseBlock()
			if err != nil {
				return nil, err
			}
			sw.Default = body
		default:
			return nil, p.errf("expected 'case' or 'default', got %q", p.cur.Literal)
		}
	}
	if _, err := p.expect(lexer.RBRACE); err != nil {
		return nil, err
	}
	return sw, nil
}

// parseExprOrAssign parses an expression statement, upgrading to an assignment
// if an '=' follows.
func (p *Parser) parseExprOrAssign() (ast.Statement, error) {
	target, err := p.parseExpression(lowest)
	if err != nil {
		return nil, err
	}
	if p.cur.Type == lexer.ASSIGN {
		p.advance()
		val, err := p.parseExpression(lowest)
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(lexer.SEMI); err != nil {
			return nil, err
		}
		return &ast.AssignStmt{Target: target, Value: val}, nil
	}
	if _, err := p.expect(lexer.SEMI); err != nil {
		return nil, err
	}
	return &ast.ExprStmt{X: target}, nil
}

// --- Expression parsing (Pratt) ----------------------------------------------

// Precedence levels.
const (
	lowest = iota
	equality
	comparison
	sum
	product
	call
)

func precedenceOf(t lexer.TokenType) int {
	switch t {
	case lexer.EQ, lexer.NEQ:
		return equality
	case lexer.LT, lexer.GT, lexer.LTE, lexer.GTE:
		return comparison
	case lexer.PLUS, lexer.MINUS:
		return sum
	case lexer.STAR, lexer.SLASH:
		return product
	case lexer.LPAREN, lexer.LBRACKET, lexer.DOT:
		return call
	}
	return lowest
}

func (p *Parser) parseExpression(minPrec int) (ast.Expression, error) {
	left, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}
	for {
		prec := precedenceOf(p.cur.Type)
		if prec <= minPrec {
			break
		}
		switch p.cur.Type {
		case lexer.LPAREN:
			left, err = p.parseCall(left)
			if err != nil {
				return nil, err
			}
			continue
		case lexer.LBRACKET:
			left, err = p.parseIndex(left)
			if err != nil {
				return nil, err
			}
			continue
		case lexer.DOT:
			left, err = p.parseMemberAccess(left)
			if err != nil {
				return nil, err
			}
			continue
		}
		op := p.cur.Literal
		p.advance()
		right, err := p.parseExpression(prec)
		if err != nil {
			return nil, err
		}
		left = &ast.BinaryExpr{Left: left, Op: op, Right: right}
	}
	return left, nil
}

func (p *Parser) parseCall(callee ast.Expression) (ast.Expression, error) {
	p.advance() // (
	c := &ast.CallExpr{Callee: callee}
	for p.cur.Type != lexer.RPAREN && p.cur.Type != lexer.EOF {
		arg, err := p.parseExpression(lowest)
		if err != nil {
			return nil, err
		}
		c.Args = append(c.Args, arg)
		if p.cur.Type == lexer.COMMA {
			p.advance()
		}
	}
	if _, err := p.expect(lexer.RPAREN); err != nil {
		return nil, err
	}
	return c, nil
}

// parseIndex parses a subscript: target[index].
func (p *Parser) parseIndex(target ast.Expression) (ast.Expression, error) {
	p.advance() // [
	idx, err := p.parseExpression(lowest)
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(lexer.RBRACKET); err != nil {
		return nil, err
	}
	return &ast.IndexExpr{Target: target, Index: idx}, nil
}

// parseMemberAccess parses a member access: target.member.
func (p *Parser) parseMemberAccess(target ast.Expression) (ast.Expression, error) {
	p.advance() // .
	member, err := p.expect(lexer.IDENT)
	if err != nil {
		return nil, err
	}
	return &ast.MemberExpr{Target: target, Member: member.Literal}, nil
}

func (p *Parser) parsePrimary() (ast.Expression, error) {
	switch p.cur.Type {
	case lexer.IDENT:
		id := &ast.Identifier{Name: p.cur.Literal}
		p.advance()
		return id, nil
	case lexer.INT:
		lit := &ast.IntLiteral{Value: p.cur.Literal}
		p.advance()
		return lit, nil
	case lexer.TRUE:
		p.advance()
		return &ast.BoolLiteral{Value: true}, nil
	case lexer.FALSE:
		p.advance()
		return &ast.BoolLiteral{Value: false}, nil
	case lexer.LPAREN:
		p.advance()
		e, err := p.parseExpression(lowest)
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(lexer.RPAREN); err != nil {
			return nil, err
		}
		return e, nil
	}
	return nil, p.errf("unexpected token %q in expression", p.cur.Literal)
}
