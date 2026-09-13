---
sip: 5
title: datetime type and duration math via an auto-injected library
status: Final
type: Standards Track
author: Salty maintainers
created: 2026-09-13
requires: 4
discussion: https://github.com/bitcoinbrisbane/salty/issues/4
---

## Abstract

This SIP adds a `datetime` type and a small, moment.js-flavoured API for working
with time. `datetime` lowers to `uint256` — Unix seconds since the epoch — so it
interoperates directly with `block.timestamp` and is gas-cheap. `now()` lowers
to `block.timestamp`, and fluent duration methods — `t.addDays(n)`,
`t.subDays(n)`, `t.addHours(n)`, `t.addMinutes(n)`, `t.addSeconds(n)` (and their
`sub` counterparts) — lower to calls into a generated `library LibDateTime`,
which Salty injects once, only when used. It reuses the SIP-4 pattern: *a
language feature that lowers to a call plus an auto-injected helper library*.

## Motivation

Time handling in Solidity is raw. Everything is `uint256` seconds and developers
hand-roll `+ 1 days`, `* 3600`, and similar arithmetic inline — error-prone and
hard to read. JavaScript developers reach for [moment.js](https://momentjs.com/)
for exactly this ergonomics gap. Salty can offer a readable, self-documenting
`datetime` with fluent duration math while emitting clean, dependency-free
Solidity.

The impractical-or-unsafe-on-chain parts of moment — timezones, locale-aware
formatting, and string parsing — are intentionally out of scope. Calendar-field
extraction (`year()`, `month()`, `day()`) and construction (`fromYMD`) are
deferred to a follow-up SIP (see [Rationale](#rationale)).

## Specification

### Syntax

`datetime` is a new elementary type. It may be used anywhere an elementary type
can appear: state variables, locals, parameters, and returns.

```
// Salty
datetime start;

function schedule() public {
    datetime t = now();
    start = t.addDays(1).addHours(6);
}
```

- `now()` is an expression that yields the current block time.
- The following methods are defined on a `datetime` value `t`, where `n` is an
  integer count. Each returns a new `datetime` (they do not mutate `t`):

  | Method            | Meaning              |
  | ----------------- | -------------------- |
  | `t.addSeconds(n)` | `t` plus `n` seconds |
  | `t.subSeconds(n)` | `t` minus `n` seconds |
  | `t.addMinutes(n)` | `t` plus `n` minutes |
  | `t.subMinutes(n)` | `t` minus `n` minutes |
  | `t.addHours(n)`   | `t` plus `n` hours   |
  | `t.subHours(n)`   | `t` minus `n` hours  |
  | `t.addDays(n)`    | `t` plus `n` days    |
  | `t.subDays(n)`    | `t` minus `n` days   |

Because each method returns a `datetime`, calls chain: `t.addDays(1).addHours(6)`.

### Lowering

- `datetime` lowers to `uint256` (identical shape to SIP-2 `decimal -> uint256`).
- `now()` lowers to `block.timestamp`.
- `t.addX(n)` lowers to `LibDateTime.addX(t, n)`; `t.subX(n)` to
  `LibDateTime.subX(t, n)`. Chained calls lower to nested calls.

```solidity
// Salty
datetime t = now();
start = t.addDays(1).addHours(6);

// Solidity
uint256 t = block.timestamp;
start = LibDateTime.addHours(LibDateTime.addDays(t, 1), 6);
```

### Injected library

When at least one `datetime` method is used in a compilation unit, Salty
prepends a single `library LibDateTime` after the pragma and before the first
contract. It is emitted at most once per output regardless of how many call
sites use it, and is omitted entirely when unused (matching SIP-4 `LibSort`).

`now()` alone does **not** require the library — it lowers to `block.timestamp`
directly. The library is injected only when a duration method is used.

```solidity
// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

// Auto-injected by Salty for SIP-5 (datetime).
// Modelled on BokkyPooBah's DateTimeLibrary (MIT):
// https://github.com/bokkypoobah/BokkyPooBahsDateTimeLibrary
library LibDateTime {
    uint256 constant SECONDS_PER_DAY = 24 * 60 * 60;
    uint256 constant SECONDS_PER_HOUR = 60 * 60;
    uint256 constant SECONDS_PER_MINUTE = 60;

    function addSeconds(uint256 t, uint256 n) internal pure returns (uint256) {
        return t + n;
    }
    function subSeconds(uint256 t, uint256 n) internal pure returns (uint256) {
        return t - n;
    }
    function addMinutes(uint256 t, uint256 n) internal pure returns (uint256) {
        return t + n * SECONDS_PER_MINUTE;
    }
    function subMinutes(uint256 t, uint256 n) internal pure returns (uint256) {
        return t - n * SECONDS_PER_MINUTE;
    }
    function addHours(uint256 t, uint256 n) internal pure returns (uint256) {
        return t + n * SECONDS_PER_HOUR;
    }
    function subHours(uint256 t, uint256 n) internal pure returns (uint256) {
        return t - n * SECONDS_PER_HOUR;
    }
    function addDays(uint256 t, uint256 n) internal pure returns (uint256) {
        return t + n * SECONDS_PER_DAY;
    }
    function subDays(uint256 t, uint256 n) internal pure returns (uint256) {
        return t - n * SECONDS_PER_DAY;
    }
}

contract C { /* ... */ }
```

## Rationale

- **`uint256` alias, not a struct.** Mirrors how Solidity itself models time and
  how `block.timestamp` is typed, keeping `datetime` values comparable,
  storable, and gas-cheap. It reuses the exact lowering shape SIP-2 established
  for `decimal -> uint256`. A struct-of-fields representation would be expensive
  on-chain, awkward to compare and store, and diverge from Solidity's model.
- **Modelled on BokkyPooBah's DateTimeLibrary.** The injected `LibDateTime`
  matches the API names, semantics, and time constants of
  [BokkyPooBah's DateTimeLibrary](https://github.com/bokkypoobah/BokkyPooBahsDateTimeLibrary)
  (MIT) — the de-facto standard for on-chain date math — so Salty stays
  compatible with a well-audited convention and users can graduate to the full
  library later. We generate a small library rather than importing it because
  Salty has no import/dependency resolver (the same reason SIP-4 injects code).
  For the duration-only scope of this SIP the relevant helpers are trivial
  (`t + n * SECONDS_PER_DAY`), so injecting only those keeps emitted code small.
- **Auto-injected library, reused pattern.** SIP-4 defined "lower to a call +
  inject a helper library, once, only when used." `datetime` is a second user of
  that mechanism, validating it as the way Salty adds utility features without a
  dependency system.
- **`now()` + duration math first.** This is the cheapest and most-used subset.
  Calendar-field extraction (`year()`, `month()`, `day()`, `weekday()`),
  construction (`fromYMD`), and differences (`a.diffDays(b)`) are a natural
  follow-up **SIP-6**, which is exactly where BokkyPooBah's leap-year-correct
  civil date math earns its larger footprint — at which point SIP-6 can inject
  the full BPBDTL.
- **Overflow/underflow.** Under `pragma solidity ^0.8.0`, arithmetic reverts on
  overflow/underflow, so `subDays` past the epoch reverts rather than wrapping.
  This is the safe default and needs no extra checks in the generated code.

## Backwards compatibility

Backwards compatible. `datetime`, `now`, and the `addX`/`subX` methods were not
previously meaningful in Salty. The injected `LibDateTime` is only added when a
duration method is used; programs that never touch datetime are byte-for-byte
unchanged. Note: a user-defined `LibDateTime` would collide with the injected
one — the same caveat as SIP-4's `LibSort`; a future revision may namespace or
detect this.

## Reference Implementation

Implemented in the Salty transpiler:

- The lexer tags `datetime` as a type keyword — `internal/lexer/token.go`.
- Lowering recognises `now()` and rewrites it to `block.timestamp`, and rewrites
  `t.addX(n)` / `t.subX(n)` call expressions to `LibDateTime.addX(t, n)` /
  `LibDateTime.subX(t, n)`, recording that the library is needed —
  `internal/transpiler/lower.go`.
- The emitter injects the `LibDateTime` library once, after the pragma and
  before the first contract, only when a datetime method was used —
  `internal/transpiler/emit.go`.

Example: [`examples/datetime.salty`](../examples/datetime.salty). Follows the
SIP-4 injected-library pattern.
