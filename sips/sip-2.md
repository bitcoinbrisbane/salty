---
sip: 2
title: Decimal fixed-point type
status: Final
type: Standards Track
author: Salty maintainers
created: 2026-09-03
discussion: https://github.com/bitcoinbrisbane/salty/issues/1
---

## Abstract

This SIP introduces a `decimal` type to Salty. Solidity has no usable decimal or
floating-point type, so contracts conventionally represent fractional values as
integers scaled by a fixed number of decimal places (as ERC-20 tokens do with
18-place "wei" units). Salty makes this convention a first-class type: `decimal`
and `decimal(N)` are written naturally, and the transpiler lowers them to
`uint256` while scaling decimal literals to their integer representation.

## Motivation

Fractional quantities — prices, interest rates, token amounts — are everywhere in
smart contracts, but Solidity forces developers to track scaling by hand:

```solidity
// Solidity today: "1.5 tokens at 18 decimals" written as a raw integer.
uint256 amount = 1500000000000000000;
```

This is error-prone (miscounted zeros) and obscures intent. Salty should let the
author write the value and the precision directly, and generate the correct
integer.

## Specification

### Types

- `decimal` is shorthand for `decimal(18)`.
- `decimal(N)` is a fixed-point type with `N` fractional decimal places, where
  `N` is an integer literal in the range `0 <= N <= 77`.
- `decimal(N)` **lowers to `uint256`** in the emitted Solidity. The scale `N` is
  a compile-time property of the type; it does not appear in the output.

```
// Salty
decimal price;         //  decimal(18)
decimal(8) feeRate;    //  8 fractional places

// Solidity
uint256 price;
uint256 feeRate;
```

### Decimal literals

- A decimal literal has the form `<digits>.<digits>` (e.g. `1.5`, `0.001`,
  `100.0`).
- A decimal literal is scaled to an integer by the scale `N` of the context in
  which it appears, i.e. it is multiplied by `10^N`, and the emitted Solidity is
  that integer.
- The literal must not have more fractional digits than the target scale `N`
  (that would lose precision); doing so is a compile-time error.

```
// Salty (scale 18)
decimal price = 1.5;

// Solidity
uint256 price = 1500000000000000000;
```

For an assignment or initializer, the scale is taken from the declared type of
the target. A bare decimal literal with no fractional part still requires a `.`
to be recognised as `decimal` (otherwise it is an ordinary integer).

### Arithmetic (this SIP)

Arithmetic on `decimal` values lowers to ordinary `uint256` arithmetic. Addition
and subtraction of two values **at the same scale** are correct. Multiplication
and division are **not** rescaled by this SIP and will produce values at the
wrong scale; authors must rescale manually for now. A future SIP may introduce
scale-aware arithmetic. This limitation is documented rather than prevented.

## Rationale

- **`uint256`, not `fixed`/`ufixed`.** Solidity reserves `fixed`/`ufixed` but
  `solc` does not implement them, so emitting them would not compile. The scaled
  integer convention is what real contracts use.
- **Parameterized `decimal(N)`.** Different domains need different precision
  (8 for many price feeds, 18 for ERC-20). A single fixed scale would be too
  rigid; `decimal` defaults to 18 for the common case.
- **Compile-time scaling of literals.** Scaling `1.5` to `1500000000000000000`
  at transpile time keeps the emitted Solidity constant-foldable and free of
  runtime cost.
- **Reject excess precision.** Silently truncating `1.234` at scale 2 would hide
  bugs; erroring is safer.

## Backwards compatibility

Fully backwards compatible. `decimal` was not previously a keyword and decimal
literals were not previously valid Salty, so no existing program changes meaning.

## Reference Implementation

Implemented in the Salty transpiler:

- Lexer: `decimal` keyword and decimal-literal scanning — `internal/lexer/`.
- AST: `Type` carries an optional decimal scale; `DecimalLiteral` expression —
  `internal/ast/ast.go`.
- Parser: `decimal` / `decimal(N)` type parsing — `internal/parser/parser.go`.
- Lowering + emission: `decimal(N)` → `uint256`, literal scaling by `10^N` —
  `internal/transpiler/`.

Example: [`examples/price.salty`](../examples/price.salty).
