package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/bitcoinbrisbane/salty/internal/transpiler"
)

const version = "salty 0.0.1"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "version", "-v", "--version":
		fmt.Println(version)
	case "transpile":
		os.Exit(runTranspile(os.Args[2:]))
	case "help", "-h", "--help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", os.Args[1])
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: salty <command> [args]")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "commands:")
	fmt.Fprintln(os.Stderr, "  transpile <file.salty> [-o out.sol]   transpile Salty to Solidity")
	fmt.Fprintln(os.Stderr, "  version                                print version")
}

func runTranspile(args []string) int {
	fs := flag.NewFlagSet("transpile", flag.ContinueOnError)
	out := fs.String("o", "", "write Solidity output to this file instead of stdout")
	// Reorder so flags may appear before or after the input path.
	flags, positional := splitArgs(args)
	if err := fs.Parse(flags); err != nil {
		return 2
	}
	if len(positional) != 1 {
		fmt.Fprintln(os.Stderr, "usage: salty transpile <file.salty> [-o out.sol]")
		return 2
	}

	inPath := positional[0]
	src, err := os.ReadFile(inPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "salty: %v\n", err)
		return 1
	}

	solidity, err := transpiler.Transpile(string(src))
	if err != nil {
		fmt.Fprintf(os.Stderr, "salty: %v\n", err)
		return 1
	}

	if *out == "" {
		fmt.Print(solidity)
		return 0
	}
	if err := os.WriteFile(*out, []byte(solidity), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "salty: %v\n", err)
		return 1
	}
	fmt.Fprintf(os.Stderr, "wrote %s\n", *out)
	return 0
}

// splitArgs separates flag arguments (and their values) from positional
// arguments so flags may be given before or after the input path. A flag whose
// name is followed by a separate value token (e.g. "-o out.sol") consumes that
// token too.
func splitArgs(args []string) (flags, positional []string) {
	for i := 0; i < len(args); i++ {
		a := args[i]
		if len(a) > 0 && a[0] == '-' {
			flags = append(flags, a)
			// If this is "-o" (no "=") take the next token as its value.
			if a == "-o" && i+1 < len(args) {
				i++
				flags = append(flags, args[i])
			}
			continue
		}
		positional = append(positional, a)
	}
	return flags, positional
}
