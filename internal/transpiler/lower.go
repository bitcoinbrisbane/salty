package transpiler

import "github.com/bitcoinbrisbane/salty/internal/ast"

// typeAliases maps Salty type sugar to the canonical Solidity type. Adding a new
// alias is a single line here.
var typeAliases = map[string]string{
	"uint": "uint256",
	"int":  "int256",
}

// lowerFile rewrites a Salty AST into an equivalent AST that maps directly onto
// Solidity: switch statements become if/else-if chains and type aliases are
// expanded to their canonical Solidity names.
func lowerFile(f *ast.File) {
	for _, c := range f.Contracts {
		for _, m := range c.Members {
			switch member := m.(type) {
			case *ast.StateVar:
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
				lowerFunction(member)
			}
		}
	}
}

// lowerType expands type aliases, recursing into mapping key and value types so
// aliases inside mappings (e.g. mapping(address => uint)) are expanded too.
func lowerType(t ast.Type) ast.Type {
	if t.IsMapping() {
		key := lowerType(*t.Key)
		val := lowerType(*t.Value)
		return ast.Type{Key: &key, Value: &val}
	}
	if canonical, ok := typeAliases[t.Name]; ok {
		return ast.Type{Name: canonical}
	}
	return t
}

func lowerFunction(fn *ast.Function) {
	for _, p := range fn.Params {
		p.Type = lowerType(p.Type)
	}
	for _, r := range fn.Returns {
		r.Type = lowerType(r.Type)
	}
	if fn.Body != nil {
		lowerBlock(fn.Body)
	}
}

func lowerBlock(b *ast.Block) {
	for i, s := range b.Statements {
		b.Statements[i] = lowerStatement(s)
	}
}

func lowerStatement(s ast.Statement) ast.Statement {
	switch stmt := s.(type) {
	case *ast.VarDeclStmt:
		stmt.Type = lowerType(stmt.Type)
		return stmt
	case *ast.IfStmt:
		lowerBlock(stmt.Then)
		if stmt.Else != nil {
			stmt.Else = lowerStatement(stmt.Else)
		}
		return stmt
	case *ast.Block:
		lowerBlock(stmt)
		return stmt
	case *ast.SwitchStmt:
		return lowerSwitch(stmt)
	default:
		return s
	}
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
func lowerSwitch(sw *ast.SwitchStmt) ast.Statement {
	// Lower nested statements in each case body first.
	for _, c := range sw.Cases {
		lowerBlock(c.Body)
	}
	if sw.Default != nil {
		lowerBlock(sw.Default)
	}

	if len(sw.Cases) == 0 {
		if sw.Default != nil {
			return sw.Default
		}
		return &ast.Block{}
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

	return elseBranch
}
