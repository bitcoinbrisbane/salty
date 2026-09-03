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

// parseMember parses either a function or a state variable. Both start with a
// keyword we can distinguish: functions start with `function`, state vars with
// a type name.
func (p *Parser) parseMember() (ast.Node, error) {
	if p.cur.Type == lexer.FUNCTION {
		return p.parseFunction()
	}
	if p.cur.Type == lexer.TYPE {
		return p.parseStateVar()
	}
	return nil, p.errf("expected 'function' or a type, got %q", p.cur.Literal)
}

func (p *Parser) parseStateVar() (*ast.StateVar, error) {
	typ := ast.Type{Name: p.cur.Literal}
	p.advance()
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
		if p.cur.Type != lexer.TYPE {
			return nil, p.errf("expected parameter type, got %q", p.cur.Literal)
		}
		ptyp := ast.Type{Name: p.cur.Literal}
		p.advance()
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
			if p.cur.Type != lexer.TYPE {
				return nil, p.errf("expected return type, got %q", p.cur.Literal)
			}
			fn.Returns = append(fn.Returns, ast.Type{Name: p.cur.Literal})
			p.advance()
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
	case lexer.TYPE:
		return p.parseVarDecl()
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

func (p *Parser) parseVarDecl() (ast.Statement, error) {
	typ := ast.Type{Name: p.cur.Literal}
	p.advance()
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
	case lexer.LPAREN:
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
		if p.cur.Type == lexer.LPAREN {
			left, err = p.parseCall(left)
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
