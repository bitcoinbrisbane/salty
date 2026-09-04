---
sip: 3
title: Dynamic array type
status: Accepted
type: Standards Track
author: Salty maintainers
created: 2026-09-04
discussion: https://github.com/bitcoinbrisbane/salty/issues/2
---

## Abstract

This SIP adds dynamic array types to Salty: `T[]` for any element type `T`,
along with element access, `.length`, and the `push`/`pop` operations. Arrays
lower directly to Solidity dynamic arrays. This is the prerequisite for
array operations such as sorting (SIP-4).

## Motivation

Salty can declare scalars, mappings, and structs, but has no way to hold an
ordered, growable collection. Arrays are fundamental — token holder lists,
queues, batches — and are the substrate for higher-level operations like `sort`.
Because Salty already tokenises `[` and `]` and parses index (`a[i]`) and member
(`a.length`) expressions for mappings, adding the array *type* is the main
missing piece.

## Specification

### Type syntax

- `T[]` is a dynamic array whose elements have type `T`.
- `T` may be any Salty type: an elementary type (`uint`, `address`, …), a struct
  name, a `decimal(N)` (SIP-2), or another array (`T[][]`).
- Arrays may appear anywhere a type may: state variables, function parameters
  and returns (with a data location, e.g. `uint[] memory`), and local
  declarations.

```
// Salty
uint[] ids;
address[] holders;
decimal(18)[] prices;

// Solidity
uint256[] ids;
address[] holders;
uint256[] prices;      // decimal(18) element lowered per SIP-2
```

The element type is lowered by the normal rules (SIP-2 decimals, `uint`→`uint256`
aliases), and the `[]` suffix is preserved.

### Operations

The following are lowered 1:1 to their Solidity equivalents (they already parse
as index/member/call expressions in Salty):

| Salty | Solidity | Meaning |
| ----- | -------- | ------- |
| `a[i]` | `a[i]` | element access |
| `a.length` | `a.length` | element count |
| `a.push(x)` | `a.push(x)` | append |
| `a.pop()` | `a.pop()` | remove last |

### Data locations

In Solidity, array parameters, returns, and some locals require a data location
(`memory`, `storage`, `calldata`). Salty already parses data-location keywords
on parameters, returns, and locals; they apply unchanged to array types:

```
// Salty
function ids(uint[] memory input) public returns (uint[] memory) { ... }

// Solidity
function ids(uint256[] memory input) public returns (uint256[] memory) { ... }
```

## Rationale

- **`T[]` suffix, matching Solidity.** Salty is a thin, familiar layer over
  Solidity; mirroring Solidity's array syntax keeps the mental model simple and
  the lowering trivial.
- **Reuse existing expression parsing.** Index, member, and call expressions
  already exist for mappings and general calls, so `a[i]`, `a.length`, and
  `a.push(x)` need no new expression nodes.
- **Fixed-size arrays deferred.** `T[N]` fixed-size arrays are a separate,
  smaller feature and can be a later SIP; dynamic arrays are the common case and
  what `sort` needs.
- **Struct-typed array *locals* deferred.** A local like `Account[] xs;` is
  ambiguous with index assignment (`xs[0] = ...`) under the parser's two-token
  lookahead. Struct arrays are fully supported as state variables, parameters,
  and returns; only local declarations of struct arrays are out of scope for the
  first implementation. Elementary-typed array locals (`uint[] xs`) work.

## Backwards compatibility

Backwards compatible. `T[]` was not previously valid in a type position, so no
existing program changes meaning.

## Reference Implementation

- AST: `Type` gains an array element type — `internal/ast/ast.go`.
- Parser: `parseType` recognises a trailing `[]` suffix — `internal/parser/parser.go`.
- Lowering + emission: array element types lower recursively; `T[]` is emitted —
  `internal/transpiler/`.

Example: [`examples/arrays.salty`](../examples/arrays.salty).
