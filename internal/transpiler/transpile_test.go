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
		"uint256 count;", // alias expanded
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
		"uint256 balance;",                      // alias inside struct
		"mapping(address => Account) accounts;", // mapping to struct
		"mapping(address => mapping(address => uint256)) allowance;", // nested mapping + alias
		"return accounts[owner].balance;",                            // index + member access
		"allowance[msg.sender][spender] = amount;",                   // chained index + member
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
		"constructor(string memory initial) public {",            // constructor + data location
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

func TestTranspileArrays(t *testing.T) {
	// SIP-3: T[] lowers to Solidity arrays; element types lower by the usual
	// rules (uint->uint256, decimal(N)->uint256).
	src := `contract R {
    uint[] ids;
    decimal(18)[] prices;
    uint[][] grid;
    function add(uint id) public {
        ids.push(id);
    }
    function count() public view returns (uint) {
        return ids.length;
    }
    function f(uint[] memory xs) public pure returns (uint) {
        return xs[0];
    }
}`

	out, err := Transpile(src)
	if err != nil {
		t.Fatalf("Transpile: %v", err)
	}

	for _, want := range []string{
		"uint256[] ids;",
		"uint256[] prices;", // decimal(18)[] element lowered (SIP-2)
		"uint256[][] grid;", // nested array
		"ids.push(id);",
		"return ids.length;",
		"function f(uint256[] memory xs) public pure returns (uint256) {",
		"return xs[0];",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n---\n%s", want, out)
		}
	}
}

func TestTranspileSort(t *testing.T) {
	// SIP-4: a.sort() -> LibSort.sort(a), with the LibSort library injected once.
	src := `contract L {
    uint[] a;
    uint[] b;
    function f() public {
        a.sort();
        b.sort();
    }
}`

	out, err := Transpile(src)
	if err != nil {
		t.Fatalf("Transpile: %v", err)
	}

	if !strings.Contains(out, "LibSort.sort(a);") || !strings.Contains(out, "LibSort.sort(b);") {
		t.Errorf("sort calls not rewritten:\n%s", out)
	}
	if n := strings.Count(out, "library LibSort {"); n != 1 {
		t.Errorf("LibSort library defined %d times, want exactly 1:\n%s", n, out)
	}
	if strings.Contains(out, ".sort()") {
		t.Errorf("unrewritten .sort() left in output:\n%s", out)
	}
	// The library must precede the contract that uses it.
	if strings.Index(out, "library LibSort {") > strings.Index(out, "contract L {") {
		t.Errorf("LibSort library emitted after the contract:\n%s", out)
	}
}

func TestTranspileNoSortNoLib(t *testing.T) {
	// SIP-4: LibSort is omitted entirely when sort is not used.
	src := `contract A {
    uint[] xs;
    function f() public {
        xs.push(1);
    }
}`

	out, err := Transpile(src)
	if err != nil {
		t.Fatalf("Transpile: %v", err)
	}
	if strings.Contains(out, "LibSort") {
		t.Errorf("LibSort injected when sort was not used:\n%s", out)
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

func TestTranspileDateTime(t *testing.T) {
	// SIP-5: datetime -> uint256, now() -> block.timestamp, and duration methods
	// -> LibDateTime.<method>(t, n), with the LibDateTime library injected once.
	src := `contract Sub {
    datetime start;
    datetime expiry;
    function subscribe() public {
        datetime t = now();
        start = t;
        expiry = t.addDays(30);
    }
    function extend() public {
        expiry = expiry.addDays(7).addHours(12);
    }
}`

	out, err := Transpile(src)
	if err != nil {
		t.Fatalf("Transpile: %v", err)
	}

	for _, want := range []string{
		"uint256 start;",               // datetime state var -> uint256
		"uint256 t = block.timestamp;", // datetime local + now() lowering
		"expiry = LibDateTime.addDays(t, 30);",
		// Chained calls lower to nested calls, applied inside-out.
		"expiry = LibDateTime.addHours(LibDateTime.addDays(expiry, 7), 12);",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n---\n%s", want, out)
		}
	}
	if n := strings.Count(out, "library LibDateTime {"); n != 1 {
		t.Errorf("LibDateTime library defined %d times, want exactly 1:\n%s", n, out)
	}
	if strings.Contains(out, ".addDays(") && strings.Contains(out, "expiry.addDays") {
		t.Errorf("unrewritten datetime method left in output:\n%s", out)
	}
	// The library must precede the contract that uses it.
	if strings.Index(out, "library LibDateTime {") > strings.Index(out, "contract Sub {") {
		t.Errorf("LibDateTime library emitted after the contract:\n%s", out)
	}
}

func TestTranspileNowWithoutLib(t *testing.T) {
	// SIP-5: now() alone lowers to block.timestamp and does NOT inject
	// LibDateTime — only duration methods require the library.
	src := `contract Clock {
    function current() public view returns (datetime) {
        return now();
    }
}`

	out, err := Transpile(src)
	if err != nil {
		t.Fatalf("Transpile: %v", err)
	}
	if !strings.Contains(out, "return block.timestamp;") {
		t.Errorf("now() not lowered to block.timestamp:\n%s", out)
	}
	if !strings.Contains(out, "returns (uint256)") {
		t.Errorf("datetime return type not lowered to uint256:\n%s", out)
	}
	if strings.Contains(out, "LibDateTime") {
		t.Errorf("LibDateTime injected when only now() was used:\n%s", out)
	}
}
