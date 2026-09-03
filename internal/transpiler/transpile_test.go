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
