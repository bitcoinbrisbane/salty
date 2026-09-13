package transpiler

import (
	"fmt"
	"strings"

	"github.com/bitcoinbrisbane/salty/internal/ast"
)

// typeAliases maps Salty type sugar to the canonical Solidity type. Adding a new
// alias is a single line here.
var typeAliases = map[string]string{
	"uint": "uint256",
	"int":  "int256",
}

// lowerer rewrites a Salty AST into an equivalent AST that maps directly onto
// Solidity. It tracks the scale of decimal-typed variables (SIP-2) so decimal
// literals assigned to them can be scaled to integers, and whether an array
// sort was used (SIP-4) so the emitter can inject the LibSort library.
type lowerer struct {
	// decimalScales maps a variable name in scope to its decimal scale, used to
	// scale decimal literals at their assignment site (SIP-2).
	decimalScales map[string]int
	// needsLibSort records that an a.sort() was rewritten (SIP-4), so the
	// emitter must inject the LibSort library.
	needsLibSort bool
	// needsLibDateTime records that a datetime duration method was rewritten
	// (SIP-5), so the emitter must inject the LibDateTime library.
	needsLibDateTime bool
}

// lowerResult reports auxiliary facts the emitter needs after lowering.
type lowerResult struct {
	needsLibSort     bool // SIP-4: emit the LibSort library
	needsLibDateTime bool // SIP-5: emit the LibDateTime library
}

// dateTimeMethods is the set of datetime duration methods (SIP-5). Each
// t.method(n) lowers to LibDateTime.method(t, n).
var dateTimeMethods = map[string]bool{
	"addSeconds": true, "subSeconds": true,
	"addMinutes": true, "subMinutes": true,
	"addHours": true, "subHours": true,
	"addDays": true, "subDays": true,
}

// lowerFile lowers f in place, returning what the emitter needs and the first
// error encountered (e.g. a decimal literal with more precision than its target
// type allows).
func lowerFile(f *ast.File) (lowerResult, error) {
	l := &lowerer{decimalScales: map[string]int{}}
	for _, c := range f.Contracts {
		for _, m := range c.Members {
			switch member := m.(type) {
			case *ast.StateVar:
				if member.Type.IsDecimal() {
					l.decimalScales[member.Name] = *member.Type.DecimalScale
				}
				member.Type = lowerType(member.Type)
			case *ast.Struct:
				for _, field := range member.Fields {
					field.Type = lowerType(field.Type)
				}
			case *ast.Event:
				for _, param := range member.Params {
					param.Type = lowerType(param.Type)
				}
			case *ast.Function:
				if err := l.lowerFunction(member); err != nil {
					return lowerResult{}, err
				}
			}
		}
	}
	return lowerResult{needsLibSort: l.needsLibSort, needsLibDateTime: l.needsLibDateTime}, nil
}

// lowerType expands type aliases and lowers decimal(N) to uint256 (SIP-2),
// recursing into mapping key and value types.
func lowerType(t ast.Type) ast.Type {
	if t.IsArray() {
		// SIP-3: lower the element type and keep the array suffix.
		elem := lowerType(*t.Elem)
		return ast.Type{Elem: &elem}
	}
	if t.IsMapping() {
		key := lowerType(*t.Key)
		val := lowerType(*t.Value)
		return ast.Type{Key: &key, Value: &val}
	}
	if t.IsDecimal() {
		// SIP-2: decimal(N) is represented as uint256 fixed-point.
		return ast.Type{Name: "uint256"}
	}
	if t.Name == "datetime" {
		// SIP-5: datetime is represented as uint256 Unix seconds.
		return ast.Type{Name: "uint256"}
	}
	if canonical, ok := typeAliases[t.Name]; ok {
		return ast.Type{Name: canonical}
	}
	return t
}

func (l *lowerer) lowerFunction(fn *ast.Function) error {
	for _, p := range fn.Params {
		if p.Type.IsDecimal() {
			l.decimalScales[p.Name] = *p.Type.DecimalScale
		}
		p.Type = lowerType(p.Type)
	}
	for _, r := range fn.Returns {
		r.Type = lowerType(r.Type)
	}
	if fn.Body != nil {
		return l.lowerBlock(fn.Body)
	}
	return nil
}

func (l *lowerer) lowerBlock(b *ast.Block) error {
	for i, s := range b.Statements {
		lowered, err := l.lowerStatement(s)
		if err != nil {
			return err
		}
		b.Statements[i] = lowered
	}
	return nil
}

func (l *lowerer) lowerStatement(s ast.Statement) (ast.Statement, error) {
	switch stmt := s.(type) {
	case *ast.VarDeclStmt:
		// SIP-2: a decimal local establishes a scale for its initializer and
		// for later assignments to it.
		if stmt.Type.IsDecimal() {
			scale := *stmt.Type.DecimalScale
			l.decimalScales[stmt.Name] = scale
			if stmt.Value != nil {
				v, err := l.scaleExpr(stmt.Value, scale)
				if err != nil {
					return nil, err
				}
				stmt.Value = v
			}
		}
		stmt.Type = lowerType(stmt.Type)
		// SIP-5: lower now()/datetime methods in the initializer.
		if stmt.Value != nil {
			stmt.Value = l.lowerExpr(stmt.Value)
		}
		return stmt, nil
	case *ast.AssignStmt:
		// If the target is a known decimal variable, scale a decimal-literal RHS.
		if scale, ok := l.assignTargetScale(stmt.Target); ok {
			v, err := l.scaleExpr(stmt.Value, scale)
			if err != nil {
				return nil, err
			}
			stmt.Value = v
		}
		// SIP-5: lower now()/datetime methods on both sides.
		stmt.Target = l.lowerExpr(stmt.Target)
		stmt.Value = l.lowerExpr(stmt.Value)
		return stmt, nil
	case *ast.ReturnStmt:
		if stmt.Value != nil {
			stmt.Value = l.lowerExpr(stmt.Value)
		}
		return stmt, nil
	case *ast.IfStmt:
		stmt.Cond = l.lowerExpr(stmt.Cond)
		if err := l.lowerBlock(stmt.Then); err != nil {
			return nil, err
		}
		if stmt.Else != nil {
			e, err := l.lowerStatement(stmt.Else)
			if err != nil {
				return nil, err
			}
			stmt.Else = e
		}
		return stmt, nil
	case *ast.Block:
		if err := l.lowerBlock(stmt); err != nil {
			return nil, err
		}
		return stmt, nil
	case *ast.SwitchStmt:
		return l.lowerSwitch(stmt)
	case *ast.ExprStmt:
		// SIP-4: rewrite a.sort() -> LibSort.sort(a).
		stmt.X = l.lowerExpr(stmt.X)
		return stmt, nil
	default:
		return s, nil
	}
}

// lowerExpr rewrites expressions that require lowering, recursing into
// sub-expressions so nested and chained calls are handled. It covers:
//   - SIP-4 array sort: a.sort() -> LibSort.sort(a).
//   - SIP-5 datetime: now() -> block.timestamp, and t.addX(n)/t.subX(n) ->
//     LibDateTime.addX(t, n) / LibDateTime.subX(t, n).
func (l *lowerer) lowerExpr(x ast.Expression) ast.Expression {
	switch e := x.(type) {
	case *ast.CallExpr:
		// SIP-5: now() -> block.timestamp.
		if id, ok := e.Callee.(*ast.Identifier); ok && id.Name == "now" && len(e.Args) == 0 {
			return &ast.MemberExpr{
				Target: &ast.Identifier{Name: "block"},
				Member: "timestamp",
			}
		}
		if member, ok := e.Callee.(*ast.MemberExpr); ok {
			// SIP-4: a.sort() -> LibSort.sort(a).
			if member.Member == "sort" && len(e.Args) == 0 {
				l.needsLibSort = true
				return &ast.CallExpr{
					Callee: &ast.MemberExpr{
						Target: &ast.Identifier{Name: libSortName},
						Member: "sort",
					},
					Args: []ast.Expression{l.lowerExpr(member.Target)},
				}
			}
			// SIP-5: t.addDays(n) -> LibDateTime.addDays(t, n).
			if dateTimeMethods[member.Member] && len(e.Args) == 1 {
				l.needsLibDateTime = true
				return &ast.CallExpr{
					Callee: &ast.MemberExpr{
						Target: &ast.Identifier{Name: libDateTimeName},
						Member: member.Member,
					},
					Args: []ast.Expression{l.lowerExpr(member.Target), l.lowerExpr(e.Args[0])},
				}
			}
		}
		// Generic call: lower the callee and arguments.
		e.Callee = l.lowerExpr(e.Callee)
		for i, a := range e.Args {
			e.Args[i] = l.lowerExpr(a)
		}
		return e
	case *ast.BinaryExpr:
		e.Left = l.lowerExpr(e.Left)
		e.Right = l.lowerExpr(e.Right)
		return e
	case *ast.IndexExpr:
		e.Target = l.lowerExpr(e.Target)
		e.Index = l.lowerExpr(e.Index)
		return e
	case *ast.MemberExpr:
		e.Target = l.lowerExpr(e.Target)
		return e
	default:
		return x
	}
}

// assignTargetScale returns the decimal scale of an assignment target if it is a
// plain decimal-typed variable.
func (l *lowerer) assignTargetScale(target ast.Expression) (int, bool) {
	id, ok := target.(*ast.Identifier)
	if !ok {
		return 0, false
	}
	scale, ok := l.decimalScales[id.Name]
	return scale, ok
}

// scaleExpr rewrites a decimal literal (SIP-2) into the integer literal that
// represents it at the given scale. Non-decimal-literal expressions pass
// through unchanged. It errors if the literal has more fractional digits than
// the scale allows (which would lose precision).
func (l *lowerer) scaleExpr(x ast.Expression, scale int) (ast.Expression, error) {
	dec, ok := x.(*ast.DecimalLiteral)
	if !ok {
		return x, nil
	}
	if len(dec.Frac) > scale {
		return nil, fmt.Errorf(
			"decimal literal %s has %d fractional digits, more than the scale %d allows",
			dec.Text, len(dec.Frac), scale)
	}
	// Concatenate integer + fractional digits, then pad with zeros so the
	// value is scaled by exactly 10^scale.
	digits := dec.Int + dec.Frac
	digits += strings.Repeat("0", scale-len(dec.Frac))
	digits = strings.TrimLeft(digits, "0")
	if digits == "" {
		digits = "0"
	}
	return &ast.IntLiteral{Value: digits}, nil
}

// lowerSwitch turns a switch into a chain of if / else if / else. Each case
// value is compared against the subject for equality:
//
//	if (subject == v0) { ... }
//	else if (subject == v1) { ... }
//	else { default }
//
// It returns nil-safe: a switch with no cases and no default lowers to an empty
// block.
func (l *lowerer) lowerSwitch(sw *ast.SwitchStmt) (ast.Statement, error) {
	// Lower nested statements in each case body first.
	for _, c := range sw.Cases {
		if err := l.lowerBlock(c.Body); err != nil {
			return nil, err
		}
	}
	if sw.Default != nil {
		if err := l.lowerBlock(sw.Default); err != nil {
			return nil, err
		}
	}

	if len(sw.Cases) == 0 {
		if sw.Default != nil {
			return sw.Default, nil
		}
		return &ast.Block{}, nil
	}

	// Build the chain from the last case backwards so each if's Else points at
	// the next link.
	var elseBranch ast.Statement
	if sw.Default != nil {
		elseBranch = sw.Default
	}

	for i := len(sw.Cases) - 1; i >= 0; i-- {
		c := sw.Cases[i]
		cond := &ast.BinaryExpr{
			Left:  sw.Subject,
			Op:    "==",
			Right: c.Value,
		}
		ifStmt := &ast.IfStmt{Cond: cond, Then: c.Body, Else: elseBranch}
		elseBranch = ifStmt
	}

	return elseBranch, nil
}
