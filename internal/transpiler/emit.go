package transpiler

import (
	"fmt"
	"strings"

	"github.com/bitcoinbrisbane/salty/internal/ast"
)

const (
	solidityPragma = "pragma solidity ^0.8.0;"
	spdxHeader     = "// SPDX-License-Identifier: MIT"
	indentUnit     = "    "
)

// emitter builds Solidity source with indentation tracking.
type emitter struct {
	sb     strings.Builder
	indent int
}

// Emit renders a (lowered) Salty AST as Solidity source.
func Emit(f *ast.File) string {
	e := &emitter{}
	e.line(spdxHeader)
	e.line(solidityPragma)
	for _, c := range f.Contracts {
		e.raw("\n")
		e.emitContract(c)
	}
	return e.sb.String()
}

func (e *emitter) line(s string) {
	e.sb.WriteString(strings.Repeat(indentUnit, e.indent))
	e.sb.WriteString(s)
	e.sb.WriteString("\n")
}

func (e *emitter) raw(s string) { e.sb.WriteString(s) }

func (e *emitter) emitContract(c *ast.Contract) {
	e.line(fmt.Sprintf("contract %s {", c.Name))
	e.indent++
	for i, m := range c.Members {
		// Separate members with a blank line, but keep consecutive single-line
		// declarations (state variables, events) grouped together without gaps.
		if i > 0 && !(isSingleLineMember(m) && isSingleLineMember(c.Members[i-1])) {
			e.raw("\n")
		}
		switch member := m.(type) {
		case *ast.StateVar:
			e.line(fmt.Sprintf("%s %s;", member.Type.String(), member.Name))
		case *ast.Event:
			e.emitEvent(member)
		case *ast.Struct:
			e.emitStruct(member)
		case *ast.Function:
			e.emitFunction(member)
		}
	}
	e.indent--
	e.line("}")
}

// isSingleLineMember reports whether n renders as a single-line declaration,
// used to decide member spacing.
func isSingleLineMember(n ast.Node) bool {
	switch n.(type) {
	case *ast.StateVar, *ast.Event:
		return true
	}
	return false
}

func (e *emitter) emitEvent(ev *ast.Event) {
	var sb strings.Builder
	sb.WriteString("event ")
	sb.WriteString(ev.Name)
	sb.WriteString("(")
	for i, p := range ev.Params {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(p.Type.String())
		if p.Indexed {
			sb.WriteString(" indexed")
		}
		if p.Name != "" {
			sb.WriteString(" ")
			sb.WriteString(p.Name)
		}
	}
	sb.WriteString(");")
	e.line(sb.String())
}

func (e *emitter) emitStruct(s *ast.Struct) {
	e.line(fmt.Sprintf("struct %s {", s.Name))
	e.indent++
	for _, f := range s.Fields {
		e.line(fmt.Sprintf("%s %s;", f.Type.String(), f.Name))
	}
	e.indent--
	e.line("}")
}

func (e *emitter) emitFunction(fn *ast.Function) {
	var sig strings.Builder
	if fn.IsConstructor {
		sig.WriteString("constructor")
	} else {
		sig.WriteString("function ")
		sig.WriteString(fn.Name)
	}
	sig.WriteString("(")
	sig.WriteString(formatParams(fn.Params))
	sig.WriteString(")")
	if fn.Visibility != "" {
		sig.WriteString(" ")
		sig.WriteString(fn.Visibility)
	}
	if fn.Mutability != "" {
		sig.WriteString(" ")
		sig.WriteString(fn.Mutability)
	}
	if len(fn.Returns) > 0 {
		sig.WriteString(" returns (")
		sig.WriteString(formatParams(fn.Returns))
		sig.WriteString(")")
	}
	sig.WriteString(" {")
	e.line(sig.String())

	e.indent++
	for _, s := range fn.Body.Statements {
		e.emitStatement(s)
	}
	e.indent--
	e.line("}")
}

// formatParams renders a comma-separated parameter/return list, including data
// locations and names when present.
func formatParams(params []*ast.Param) string {
	parts := make([]string, len(params))
	for i, p := range params {
		s := p.Type.String()
		if p.DataLoc != "" {
			s += " " + p.DataLoc
		}
		if p.Name != "" {
			s += " " + p.Name
		}
		parts[i] = s
	}
	return strings.Join(parts, ", ")
}

func (e *emitter) emitStatement(s ast.Statement) {
	switch stmt := s.(type) {
	case *ast.VarDeclStmt:
		decl := stmt.Type.String()
		if stmt.DataLoc != "" {
			decl += " " + stmt.DataLoc
		}
		decl += " " + stmt.Name
		if stmt.Value != nil {
			e.line(fmt.Sprintf("%s = %s;", decl, emitExpr(stmt.Value)))
		} else {
			e.line(decl + ";")
		}
	case *ast.AssignStmt:
		e.line(fmt.Sprintf("%s = %s;", emitExpr(stmt.Target), emitExpr(stmt.Value)))
	case *ast.ReturnStmt:
		if stmt.Value != nil {
			e.line(fmt.Sprintf("return %s;", emitExpr(stmt.Value)))
		} else {
			e.line("return;")
		}
	case *ast.ExprStmt:
		e.line(emitExpr(stmt.X) + ";")
	case *ast.EmitStmt:
		e.line("emit " + emitExpr(stmt.Call) + ";")
	case *ast.Block:
		for _, inner := range stmt.Statements {
			e.emitStatement(inner)
		}
	case *ast.IfStmt:
		e.emitIf(stmt)
	}
}

func (e *emitter) emitIf(stmt *ast.IfStmt) {
	e.line(fmt.Sprintf("if (%s) {", emitExpr(stmt.Cond)))
	e.indent++
	for _, s := range stmt.Then.Statements {
		e.emitStatement(s)
	}
	e.indent--

	switch els := stmt.Else.(type) {
	case nil:
		e.line("}")
	case *ast.IfStmt:
		e.line(fmt.Sprintf("} else if (%s) {", emitExpr(els.Cond)))
		e.indent++
		for _, s := range els.Then.Statements {
			e.emitStatement(s)
		}
		e.indent--
		// Continue the chain by delegating the remaining else.
		e.emitElseTail(els.Else)
	case *ast.Block:
		e.line("} else {")
		e.indent++
		for _, s := range els.Statements {
			e.emitStatement(s)
		}
		e.indent--
		e.line("}")
	}
}

// emitElseTail closes an else-if chain: either another else-if, a final else,
// or nothing.
func (e *emitter) emitElseTail(els ast.Statement) {
	switch tail := els.(type) {
	case nil:
		e.line("}")
	case *ast.IfStmt:
		e.line(fmt.Sprintf("} else if (%s) {", emitExpr(tail.Cond)))
		e.indent++
		for _, s := range tail.Then.Statements {
			e.emitStatement(s)
		}
		e.indent--
		e.emitElseTail(tail.Else)
	case *ast.Block:
		e.line("} else {")
		e.indent++
		for _, s := range tail.Statements {
			e.emitStatement(s)
		}
		e.indent--
		e.line("}")
	}
}

// emitExpr renders an expression to Solidity source. Parentheses are added
// around binary sub-expressions to preserve grouping without tracking
// precedence on output.
func emitExpr(x ast.Expression) string {
	switch e := x.(type) {
	case *ast.Identifier:
		return e.Name
	case *ast.IntLiteral:
		return e.Value
	case *ast.DecimalLiteral:
		// SIP-2: decimal literals are scaled to integers during lowering when
		// their target scale is known. Reaching here means the literal appeared
		// in a position without a known decimal scale; emit the raw text so it
		// is visible rather than silently wrong.
		return e.Text
	case *ast.StringLiteral:
		return "\"" + e.Value + "\""
	case *ast.BoolLiteral:
		if e.Value {
			return "true"
		}
		return "false"
	case *ast.BinaryExpr:
		return fmt.Sprintf("%s %s %s", emitOperand(e.Left), e.Op, emitOperand(e.Right))
	case *ast.CallExpr:
		args := make([]string, len(e.Args))
		for i, a := range e.Args {
			args[i] = emitExpr(a)
		}
		return fmt.Sprintf("%s(%s)", emitExpr(e.Callee), strings.Join(args, ", "))
	case *ast.IndexExpr:
		return fmt.Sprintf("%s[%s]", emitExpr(e.Target), emitExpr(e.Index))
	case *ast.MemberExpr:
		return fmt.Sprintf("%s.%s", emitExpr(e.Target), e.Member)
	}
	return ""
}

// emitOperand wraps nested binary expressions in parentheses to keep grouping
// explicit and unambiguous.
func emitOperand(x ast.Expression) string {
	if _, ok := x.(*ast.BinaryExpr); ok {
		return "(" + emitExpr(x) + ")"
	}
	return emitExpr(x)
}
