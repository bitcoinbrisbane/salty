# Salty

**Salty** is a Solidity-like smart-contract language that transpiles **to** Solidity.
It adds ergonomic features on top of Solidity (e.g. `switch`, which lowers to
`if / else if / else`), while emitting clean, compilable Solidity.

Written in Go, paired with a pure HTML website (no framework, no build step).

```
Salty source (.salty)  ──►  Salty transpiler (Go)  ──►  Solidity (.sol)
```

## Status

Early scaffolding. See the plan below.

## Layout

```
cmd/salty/           # CLI entrypoint
internal/lexer/      # tokenizer
internal/parser/     # tokens -> AST
internal/ast/        # AST node definitions
internal/transpiler/ # AST -> Solidity emitter
web/                 # pure HTML site (no framework)
examples/            # sample .salty inputs
```

## Build

```bash
go build -o bin/salty ./cmd/salty
```

## Usage

```bash
salty transpile examples/counter.salty        # prints Solidity to stdout
salty transpile examples/counter.salty -o Counter.sol
salty transpile examples/traffic.salty        # shows switch -> if/else if/else
salty transpile examples/token.salty          # shows mappings + structs
```

## Features

- Contracts, state variables, functions (visibility + `view`/`pure`, returns).
- Structs, and mappings (including nested `mapping(K => mapping(K => V))`).
- Index (`m[key]`) and member (`s.field`) access, chainable (`accounts[owner].balance`).
- `switch` statement that lowers to `if / else if / else` in Solidity.
- Type aliases: `uint` → `uint256`, `int` → `int256` (expanded everywhere,
  including inside mappings and struct fields).
- Auto-emitted SPDX header and `pragma solidity ^0.8.0;`.
