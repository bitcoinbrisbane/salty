// Package ast defines the Salty abstract syntax tree. Nodes are produced by the
// parser and consumed by the transpiler (lowering + Solidity emission).
package ast

// Node is any AST node.
type Node interface{ node() }

// Statement is a node that appears in a statement position.
type Statement interface {
	Node
	stmt()
}

// Expression is a node that produces a value.
type Expression interface {
	Node
	expr()
}

// Type is a Salty type reference (e.g. "uint", "uint256"). It is a thin wrapper
// so lowering passes can rewrite aliases in place.
type Type struct {
	Name string
}

func (Type) node() {}

// --- Top level ---------------------------------------------------------------

// File is a whole Salty source unit.
type File struct {
	Contracts []*Contract
}

func (*File) node() {}

// Contract is a single contract declaration and its members.
type Contract struct {
	Name    string
	Members []Node // *StateVar or *Function
}

func (*Contract) node() {}

// StateVar is a contract-level state variable declaration.
type StateVar struct {
	Type Type
	Name string
}

func (*StateVar) node() {}

// Param is a single function parameter.
type Param struct {
	Type Type
	Name string
}

func (*Param) node() {}

// Function is a contract function.
type Function struct {
	Name       string
	Params     []*Param
	Visibility string // "public", "private", "internal", "external", or ""
	Mutability string // "view", "pure", or ""
	Returns    []Type // return types, in order
	Body       *Block
}

func (*Function) node() {}

// --- Statements --------------------------------------------------------------

// Block is a brace-delimited sequence of statements.
type Block struct {
	Statements []Statement
}

func (*Block) node() {}
func (*Block) stmt() {}

// VarDeclStmt is a local variable declaration, optionally with an initializer.
type VarDeclStmt struct {
	Type  Type
	Name  string
	Value Expression // may be nil
}

func (*VarDeclStmt) node() {}
func (*VarDeclStmt) stmt() {}

// AssignStmt assigns Value to Target.
type AssignStmt struct {
	Target Expression
	Value  Expression
}

func (*AssignStmt) node() {}
func (*AssignStmt) stmt() {}

// ReturnStmt returns an optional value.
type ReturnStmt struct {
	Value Expression // may be nil
}

func (*ReturnStmt) node() {}
func (*ReturnStmt) stmt() {}

// IfStmt is an if / else statement. Else may be a *Block (else) or another
// *IfStmt (else if), or nil.
type IfStmt struct {
	Cond Expression
	Then *Block
	Else Statement
}

func (*IfStmt) node() {}
func (*IfStmt) stmt() {}

// SwitchCase is one case of a switch statement.
type SwitchCase struct {
	Value Expression
	Body  *Block
}

// SwitchStmt is a Salty switch. It is lowered to an if/else-if chain when
// transpiling to Solidity.
type SwitchStmt struct {
	Subject Expression
	Cases   []*SwitchCase
	Default *Block // may be nil
}

func (*SwitchStmt) node() {}
func (*SwitchStmt) stmt() {}

// ExprStmt is an expression used as a statement (e.g. a call).
type ExprStmt struct {
	X Expression
}

func (*ExprStmt) node() {}
func (*ExprStmt) stmt() {}

// --- Expressions -------------------------------------------------------------

// Identifier references a name.
type Identifier struct {
	Name string
}

func (*Identifier) node() {}
func (*Identifier) expr() {}

// IntLiteral is an integer literal, kept as its source text.
type IntLiteral struct {
	Value string
}

func (*IntLiteral) node() {}
func (*IntLiteral) expr() {}

// BoolLiteral is true or false.
type BoolLiteral struct {
	Value bool
}

func (*BoolLiteral) node() {}
func (*BoolLiteral) expr() {}

// BinaryExpr is a binary operation such as a + b or x == y.
type BinaryExpr struct {
	Left  Expression
	Op    string
	Right Expression
}

func (*BinaryExpr) node() {}
func (*BinaryExpr) expr() {}

// CallExpr is a function call.
type CallExpr struct {
	Callee Expression
	Args   []Expression
}

func (*CallExpr) node() {}
func (*CallExpr) expr() {}
