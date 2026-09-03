package transpiler

import (
	"strings"
	"testing"
)

func TestTranspileCounter(t *testing.T) {
	src := `contract Counter {
    uint count;
    function increment() public {
        count = count + 1;
    }
    function get() public view returns (uint) {
        return count;
    }
}`

	out, err := Transpile(src)
	if err != nil {
		t.Fatalf("Transpile: %v", err)
	}

	for _, want := range []string{
		"// SPDX-License-Identifier: MIT",
		"pragma solidity ^0.8.0;",
		"contract Counter {",
		"uint256 count;",              // alias expanded
		"function increment() public {",
		"function get() public view returns (uint256) {", // alias in returns
		"return count;",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n---\n%s", want, out)
		}
	}

	if strings.Contains(out, "uint ") {
		t.Errorf("unexpanded 'uint' alias in output:\n%s", out)
	}
}

func TestTranspileSwitch(t *testing.T) {
	src := `contract S {
    function f(uint x) public {
        switch x {
            case 1: { y = 10; }
            case 2: { y = 20; }
            default: { y = 0; }
        }
    }
}`

	out, err := Transpile(src)
	if err != nil {
		t.Fatalf("Transpile: %v", err)
	}

	if strings.Contains(out, "switch") {
		t.Errorf("switch keyword leaked into output:\n%s", out)
	}
	if !strings.Contains(out, "if (x == 1) {") {
		t.Errorf("missing first if branch:\n%s", out)
	}
	if !strings.Contains(out, "} else if (x == 2) {") {
		t.Errorf("missing else-if branch:\n%s", out)
	}
	if !strings.Contains(out, "} else {") {
		t.Errorf("missing else (default) branch:\n%s", out)
	}
}

func TestTranspileMappingAndStruct(t *testing.T) {
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
    function approve(address spender, uint amount) public {
        allowance[msg.sender][spender] = amount;
    }
}`

	out, err := Transpile(src)
	if err != nil {
		t.Fatalf("Transpile: %v", err)
	}

	for _, want := range []string{
		"struct Account {",
		"uint256 balance;",                                        // alias inside struct
		"mapping(address => Account) accounts;",                   // mapping to struct
		"mapping(address => mapping(address => uint256)) allowance;", // nested mapping + alias
		"return accounts[owner].balance;",                         // index + member access
		"allowance[msg.sender][spender] = amount;",                // chained index + member
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n---\n%s", want, out)
		}
	}
}

func TestTranspileDecimal(t *testing.T) {
	// SIP-2: decimal(N) lowers to uint256; literals scale by 10^N.
	src := `contract P {
    decimal price;
    decimal(8) feeRate;
    function f() public {
        decimal p = 1.5;
        decimal(8) fee = 0.025;
    }
}`

	out, err := Transpile(src)
	if err != nil {
		t.Fatalf("Transpile: %v", err)
	}

	for _, want := range []string{
		"uint256 price;",
		"uint256 feeRate;",
		"uint256 p = 1500000000000000000;", // 1.5 at scale 18
		"uint256 fee = 2500000;",           // 0.025 at scale 8
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n---\n%s", want, out)
		}
	}
	if strings.Contains(out, "decimal") {
		t.Errorf("decimal keyword leaked into output:\n%s", out)
	}
}

func TestTranspileDecimalExcessPrecision(t *testing.T) {
	// SIP-2: more fractional digits than the scale is a compile-time error.
	src := `contract P {
    function f() public {
        decimal(2) x = 1.234;
    }
}`

	_, err := Transpile(src)
	if err == nil {
		t.Fatalf("expected an error for excess precision, got none")
	}
	if !strings.Contains(err.Error(), "fractional digits") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTranspileGreeter(t *testing.T) {
	src := `contract Greeter {
    string greeting;
    constructor(string memory initial) public {
        greeting = initial;
    }
    function greet() public view returns (string memory) {
        return greeting;
    }
    function setGreeting(string memory newGreeting) public {
        greeting = newGreeting;
    }
}`

	out, err := Transpile(src)
	if err != nil {
		t.Fatalf("Transpile: %v", err)
	}

	for _, want := range []string{
		"string greeting;",
		"constructor(string memory initial) public {",           // constructor + data location
		"function greet() public view returns (string memory) {", // return data location
		"function setGreeting(string memory newGreeting) public {",
		"return greeting;",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n---\n%s", want, out)
		}
	}
}

func TestTranspileStringLiteral(t *testing.T) {
	src := `contract T {
    string s;
    constructor() public { s = "Hello, World!"; }
}`

	out, err := Transpile(src)
	if err != nil {
		t.Fatalf("Transpile: %v", err)
	}
	if !strings.Contains(out, `s = "Hello, World!";`) {
		t.Errorf("string literal not emitted correctly:\n%s", out)
	}
}

func TestTranspileEventAndEmit(t *testing.T) {
	src := `contract Wallet {
    event Deposit(address indexed from, uint amount);
    function deposit(uint amount) public {
        emit Deposit(msg.sender, amount);
    }
}`

	out, err := Transpile(src)
	if err != nil {
		t.Fatalf("Transpile: %v", err)
	}

	for _, want := range []string{
		"event Deposit(address indexed from, uint256 amount);", // indexed + alias in params
		"emit Deposit(msg.sender, amount);",                    // emit statement
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n---\n%s", want, out)
		}
	}
}
