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

## Install

Install the CLI so `salty` is available from anywhere:

```bash
go install ./cmd/salty
```

This installs the binary to `$(go env GOPATH)/bin` (usually `~/go/bin`). If that
directory isn't on your `PATH`, add it — for zsh:

```bash
echo 'export PATH="$PATH:$(go env GOPATH)/bin"' >> ~/.zshrc
source ~/.zshrc   # or open a new terminal
```

Then `salty` works from any directory. Re-run `go install ./cmd/salty` after
changing the source to refresh the installed binary.

## Build (without installing)

To build a local binary instead of installing, run it by path:

```bash
go build -o bin/salty ./cmd/salty
./bin/salty transpile examples/counter.salty
```

## Usage

```bash
salty transpile examples/counter.salty        # prints Solidity to stdout
salty transpile examples/counter.salty -o Counter.sol
salty transpile examples/traffic.salty        # shows switch -> if/else if/else
salty transpile examples/token.salty          # shows mappings + structs
salty transpile examples/events.salty         # shows events + emit
salty transpile examples/greeter.salty        # the classic Greeter
```

## Features

- Contracts, state variables, functions (visibility + `view`/`pure`, returns).
- Constructors, string literals, and data-location keywords (`memory`/`storage`/`calldata`).
- Structs, and mappings (including nested `mapping(K => mapping(K => V))`).
- Events (with `indexed` params) and the `emit` statement.
- Index (`m[key]`) and member (`s.field`) access, chainable (`accounts[owner].balance`).
- `switch` statement that lowers to `if / else if / else` in Solidity.
- Type aliases: `uint` → `uint256`, `int` → `int256` (expanded everywhere,
  including inside mappings and struct fields).
- Auto-emitted SPDX header and `pragma solidity ^0.8.0;`.
