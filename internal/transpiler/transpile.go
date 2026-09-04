// Package transpiler turns Salty source into Solidity. It parses the source,
// lowers Salty-specific constructs (switch, type aliases) into their Solidity
// equivalents, then emits formatted Solidity.
package transpiler

import "github.com/bitcoinbrisbane/salty/internal/parser"

// Transpile compiles Salty source into Solidity source.
func Transpile(src string) (string, error) {
	file, err := parser.ParseFile(src)
	if err != nil {
		return "", err
	}
	res, err := lowerFile(file)
	if err != nil {
		return "", err
	}
	return Emit(file, res), nil
}
