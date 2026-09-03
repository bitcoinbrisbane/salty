package parser

import (
	"testing"

	"github.com/bitcoinbrisbane/salty/internal/ast"
)

func TestParseCounter(t *testing.T) {
	src := `contract Counter {
    uint count;
    function increment() public {
        count = count + 1;
    }
    function get() public view returns (uint256) {
        return count;
    }
}`

	file, err := ParseFile(src)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	if len(file.Contracts) != 1 {
		t.Fatalf("contracts = %d, want 1", len(file.Contracts))
	}
	c := file.Contracts[0]
	if c.Name != "Counter" {
		t.Fatalf("contract name = %q, want Counter", c.Name)
	}
	if len(c.Members) != 3 {
		t.Fatalf("members = %d, want 3", len(c.Members))
	}
	if _, ok := c.Members[0].(*ast.StateVar); !ok {
		t.Fatalf("member 0 = %T, want *ast.StateVar", c.Members[0])
	}
	get, ok := c.Members[2].(*ast.Function)
	if !ok {
		t.Fatalf("member 2 = %T, want *ast.Function", c.Members[2])
	}
	if get.Mutability != "view" || len(get.Returns) != 1 {
		t.Fatalf("get modifiers wrong: mut=%q returns=%d", get.Mutability, len(get.Returns))
	}
}

func TestParseSwitch(t *testing.T) {
	src := `contract S {
    function f(uint x) public {
        switch x {
            case 1: { y = 10; }
            case 2: { y = 20; }
            default: { y = 0; }
        }
    }
}`

	file, err := ParseFile(src)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	fn := file.Contracts[0].Members[0].(*ast.Function)
	sw, ok := fn.Body.Statements[0].(*ast.SwitchStmt)
	if !ok {
		t.Fatalf("stmt 0 = %T, want *ast.SwitchStmt", fn.Body.Statements[0])
	}
	if len(sw.Cases) != 2 {
		t.Fatalf("cases = %d, want 2", len(sw.Cases))
	}
	if sw.Default == nil {
		t.Fatalf("default is nil, want a block")
	}
}

func TestParseStructAndMapping(t *testing.T) {
	src := `contract Token {
    struct Account {
        uint balance;
        bool frozen;
    }
    mapping(address => Account) accounts;
    mapping(address => mapping(address => uint)) allowance;
    function balanceOf(address owner) public view returns (uint) {
        return accounts[owner].balance;
    }
}`

	file, err := ParseFile(src)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	c := file.Contracts[0]

	st, ok := c.Members[0].(*ast.Struct)
	if !ok {
		t.Fatalf("member 0 = %T, want *ast.Struct", c.Members[0])
	}
	if st.Name != "Account" || len(st.Fields) != 2 {
		t.Fatalf("struct = %q with %d fields, want Account with 2", st.Name, len(st.Fields))
	}

	sv, ok := c.Members[1].(*ast.StateVar)
	if !ok {
		t.Fatalf("member 1 = %T, want *ast.StateVar", c.Members[1])
	}
	if !sv.Type.IsMapping() {
		t.Fatalf("accounts type is not a mapping: %+v", sv.Type)
	}
	if sv.Type.Key.Name != "address" || sv.Type.Value.Name != "Account" {
		t.Fatalf("mapping types wrong: key=%q value=%q", sv.Type.Key.Name, sv.Type.Value.Name)
	}

	// Nested mapping.
	nested := c.Members[2].(*ast.StateVar)
	if !nested.Type.Value.IsMapping() {
		t.Fatalf("allowance value is not a nested mapping: %+v", nested.Type)
	}

	// Index + member access in the return expression.
	fn := c.Members[3].(*ast.Function)
	ret := fn.Body.Statements[0].(*ast.ReturnStmt)
	mem, ok := ret.Value.(*ast.MemberExpr)
	if !ok {
		t.Fatalf("return value = %T, want *ast.MemberExpr", ret.Value)
	}
	if mem.Member != "balance" {
		t.Fatalf("member = %q, want balance", mem.Member)
	}
	if _, ok := mem.Target.(*ast.IndexExpr); !ok {
		t.Fatalf("member target = %T, want *ast.IndexExpr", mem.Target)
	}
}

func TestParseGreeter(t *testing.T) {
	src := `contract Greeter {
    string greeting;
    constructor(string memory initial) public {
        greeting = initial;
    }
    function greet() public view returns (string memory) {
        return greeting;
    }
}`

	file, err := ParseFile(src)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	c := file.Contracts[0]

	ctor, ok := c.Members[1].(*ast.Function)
	if !ok || !ctor.IsConstructor {
		t.Fatalf("member 1 = %T (constructor=%v), want a constructor", c.Members[1], ok && ctor.IsConstructor)
	}
	if len(ctor.Params) != 1 {
		t.Fatalf("constructor params = %d, want 1", len(ctor.Params))
	}
	if ctor.Params[0].DataLoc != "memory" || ctor.Params[0].Type.Name != "string" {
		t.Fatalf("constructor param wrong: type=%q loc=%q", ctor.Params[0].Type.Name, ctor.Params[0].DataLoc)
	}

	greet := c.Members[2].(*ast.Function)
	if len(greet.Returns) != 1 || greet.Returns[0].DataLoc != "memory" {
		t.Fatalf("greet return wrong: %+v", greet.Returns)
	}
}

func TestParseStringLiteral(t *testing.T) {
	src := `contract T {
    string s;
    constructor() public { s = "Hello, World!"; }
}`

	file, err := ParseFile(src)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	ctor := file.Contracts[0].Members[1].(*ast.Function)
	assign := ctor.Body.Statements[0].(*ast.AssignStmt)
	lit, ok := assign.Value.(*ast.StringLiteral)
	if !ok {
		t.Fatalf("assign value = %T, want *ast.StringLiteral", assign.Value)
	}
	if lit.Value != "Hello, World!" {
		t.Fatalf("string value = %q, want %q", lit.Value, "Hello, World!")
	}
}

func TestParseEventAndEmit(t *testing.T) {
	src := `contract W {
    event Deposit(address indexed from, uint amount);
    function deposit(uint amount) public {
        emit Deposit(msg.sender, amount);
    }
}`

	file, err := ParseFile(src)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	c := file.Contracts[0]

	ev, ok := c.Members[0].(*ast.Event)
	if !ok {
		t.Fatalf("member 0 = %T, want *ast.Event", c.Members[0])
	}
	if ev.Name != "Deposit" || len(ev.Params) != 2 {
		t.Fatalf("event = %q with %d params, want Deposit with 2", ev.Name, len(ev.Params))
	}
	if !ev.Params[0].Indexed || ev.Params[0].Name != "from" {
		t.Fatalf("param 0 wrong: indexed=%v name=%q", ev.Params[0].Indexed, ev.Params[0].Name)
	}
	if ev.Params[1].Indexed {
		t.Fatalf("param 1 should not be indexed")
	}

	fn := c.Members[1].(*ast.Function)
	em, ok := fn.Body.Statements[0].(*ast.EmitStmt)
	if !ok {
		t.Fatalf("stmt 0 = %T, want *ast.EmitStmt", fn.Body.Statements[0])
	}
	callee, ok := em.Call.Callee.(*ast.Identifier)
	if !ok || callee.Name != "Deposit" {
		t.Fatalf("emit callee = %+v, want identifier Deposit", em.Call.Callee)
	}
	if len(em.Call.Args) != 2 {
		t.Fatalf("emit args = %d, want 2", len(em.Call.Args))
	}
}
