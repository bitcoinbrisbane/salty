// Package ast defines the Salty abstract syntax tree. Nodes are produced by the
// parser and consumed by the transpiler (lowering + Solidity emission).
package ast

import "fmt"

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

// Type is a Salty type reference. For simple types (e.g. "uint", "uint256",
// a struct name) only Name is set. For a mapping, Key and Value hold the
// mapping's key and value types and Name is left empty. For a decimal (SIP-2),
// DecimalScale holds the number of fractional places. For a dynamic array
// (SIP-3), Elem holds the element type.
type Type struct {
	Name         string // simple/elementary type or struct name
	Key          *Type  // mapping key type, nil for non-mappings
	Value        *Type  // mapping value type, nil for non-mappings
	DecimalScale *int   // SIP-2: fractional places for decimal types, nil otherwise
	Elem         *Type  // SIP-3: element type for dynamic arrays, nil otherwise
}

func (Type) node() {}

// IsMapping reports whether t is a mapping type.
func (t Type) IsMapping() bool { return t.Key != nil && t.Value != nil }

// IsDecimal reports whether t is a decimal type (SIP-2).
func (t Type) IsDecimal() bool { return t.DecimalScale != nil }

// IsArray reports whether t is a dynamic array type (SIP-3).
func (t Type) IsArray() bool { return t.Elem != nil }

// String renders the type as Salty source, recursively for nested mappings and
// arrays.
func (t Type) String() string {
	if t.IsArray() {
		return t.Elem.String() + "[]"
	}
	if t.IsMapping() {
		return "mapping(" + t.Key.String() + " => " + t.Value.String() + ")"
	}
	if t.IsDecimal() {
		return fmt.Sprintf("decimal(%d)", *t.DecimalScale)
	}
	return t.Name
}

// --- Top level ---------------------------------------------------------------

// File is a whole Salty source unit.
type File struct {
	Contracts []*Contract
}

func (*File) node() {}

// Contract is a single contract declaration and its members.
type Contract struct {
	Name    string
	Members []Node // *StateVar, *Function, *Struct, or *Event
}

func (*Contract) node() {}

// EventParam is a single parameter of an event declaration. Indexed marks the
// parameter with Solidity's `indexed` keyword.
type EventParam struct {
	Type    Type
	Name    string // may be empty; event params are often anonymous
	Indexed bool
}

func (*EventParam) node() {}

// Event is an event declaration inside a contract.
type Event struct {
	Name   string
	Params []*EventParam
}

func (*Event) node() {}

// Field is a single named field within a struct.
type Field struct {
	Type Type
	Name string
}

func (*Field) node() {}

// Struct is a struct type declaration inside a contract.
type Struct struct {
	Name   string
	Fields []*Field
}

func (*Struct) node() {}

// StateVar is a contract-level state variable declaration.
type StateVar struct {
	Type Type
	Name string
}

func (*StateVar) node() {}

// Param is a single function parameter or return value. Name may be empty
// (common for return values). DataLoc is a data-location keyword ("memory",
// "storage", "calldata") or "" when none is specified.
type Param struct {
	Type    Type
	Name    string
	DataLoc string
}

func (*Param) node() {}

// Function is a contract function or constructor.
type Function struct {
	Name          string
	IsConstructor bool
	Params        []*Param
	Visibility    string   // "public", "private", "internal", "external", or ""
	Mutability    string   // "view", "pure", or ""
	Returns       []*Param // return values, in order
	Body          *Block
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
// DataLoc is a data-location keyword ("memory", "storage", "calldata") or "".
type VarDeclStmt struct {
	Type    Type
	Name    string
	DataLoc string
	Value   Expression // may be nil
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

// EmitStmt emits an event: `emit Name(args...)`. Call is the underlying call
// expression whose callee is the event name.
type EmitStmt struct {
	Call *CallExpr
}

func (*EmitStmt) node() {}
func (*EmitStmt) stmt() {}

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

// StringLiteral is a double-quoted string. Value holds the contents without the
// surrounding quotes.
type StringLiteral struct {
	Value string
}

func (*StringLiteral) node() {}
func (*StringLiteral) expr() {}

// DecimalLiteral is a fixed-point literal such as 1.5 (SIP-2). Text is the
// original source text; Int and Frac are the integer and fractional digit
// strings split on the decimal point (Frac excludes the point).
type DecimalLiteral struct {
	Text string
	Int  string
	Frac string
}

func (*DecimalLiteral) node() {}
func (*DecimalLiteral) expr() {}

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

// IndexExpr is a subscript such as balances[owner].
type IndexExpr struct {
	Target Expression
	Index  Expression
}

func (*IndexExpr) node() {}
func (*IndexExpr) expr() {}

// MemberExpr is a member access such as account.balance.
type MemberExpr struct {
	Target Expression
	Member string
}

func (*MemberExpr) node() {}
func (*MemberExpr) expr() {}
