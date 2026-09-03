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
