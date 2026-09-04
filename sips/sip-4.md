---
sip: 4
title: Array sort via an auto-injected library
status: Draft
type: Standards Track
author: Salty maintainers
created: 2026-09-04
requires: 3
discussion: https://github.com/bitcoinbrisbane/salty/issues/3
---

## Abstract

This SIP adds a `sort` operation on Salty arrays. `a.sort()` lowers to a call to
a generated Solidity library, `LibSort`, which Salty prepends to the emitted
output when (and only when) sorting is used. The library implements an in-place
ascending sort over `uint256[]`. This establishes a reusable pattern for Salty:
a language feature that lowers to *a call plus an auto-injected helper library*.

## Motivation

Solidity has no built-in sort, so every project reimplements one or pulls in a
dependency (Solady's `LibSort`, OpenZeppelin `Arrays`). Salty can make sorting a
first-class, one-line operation — `a.sort()` — while keeping the emitted Solidity
clean and dependency-free by generating the sort routine itself. Requires dynamic
arrays (SIP-3).

## Specification

### Syntax

- `a.sort()` sorts the array `a` in place, ascending. `a` must be a `uint256[]`
  (after SIP-2/SIP-3 lowering). Support for other element types and a descending
  variant may be added by a later SIP.
- `a.sort()` is a statement-level operation; it returns nothing.

```
// Salty
uint[] xs;
function order() public {
    xs.sort();
}
```

### Lowering

`a.sort()` lowers to `LibSort.sort(a)`:

```solidity
function order() public {
    LibSort.sort(xs);
}
```

### Injected library

When at least one `.sort()` appears in a compilation unit, Salty prepends a
single `library LibSort` after the pragma and before the first contract. It is
emitted at most once per output regardless of how many call sites use it, and is
omitted entirely when unused.

```solidity
// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

// Auto-injected by Salty for SIP-4 (array sort).
library LibSort {
    // In-place ascending insertion sort over a memory/storage uint256 array.
    function sort(uint256[] storage a) internal {
        for (uint256 i = 1; i < a.length; i++) {
            uint256 key = a[i];
            uint256 j = i;
            while (j > 0 && a[j - 1] > key) {
                a[j] = a[j - 1];
                j--;
            }
            a[j] = key;
        }
    }
}

contract C { /* ... */ }
```

### Algorithm

The library uses **insertion sort**. Rationale in the next section.

## Rationale

- **Auto-injected library, not inline.** Emitting `LibSort.sort(a)` keeps call
  sites readable and emits the algorithm once, no matter how many times sort is
  used. Generating the library ourselves keeps the output self-contained — no
  import resolver or external dependency, which Salty does not yet have.
- **Insertion sort.** On the EVM, gas is proportional to executed operations and
  contract arrays are typically small. Insertion sort has a tiny code footprint,
  no recursion (so no stack-depth risk), and is optimal on small or
  nearly-sorted inputs. Quicksort/heapsort add code and recursion for asymptotic
  gains that rarely matter at contract array sizes; a future SIP may offer them
  for large arrays.
- **`uint256[]` first.** Establishes the mechanism with the most common case.
  Generic element types and comparators are a natural follow-up.
- **Reusable pattern.** "Lower to a call + inject a helper library, once, only
  when used" is exactly what future utility SIPs (safe math, string helpers)
  will want. SIP-4 defines that pattern.

## Backwards compatibility

Backwards compatible. `.sort()` was not previously meaningful. The injected
`LibSort` is only added when sort is used; programs that never sort are
byte-for-byte unchanged. Note: a user-defined `LibSort` would collide with the
injected one; a future revision may namespace or detect this.

## Reference Implementation

Not yet implemented. Planned:

- Recognise `a.sort()` call expressions in lowering and rewrite to
  `LibSort.sort(a)` — `internal/transpiler/lower.go`.
- Track whether sort was used and, if so, emit the `LibSort` library once after
  the pragma — `internal/transpiler/emit.go`.

Depends on SIP-3 (dynamic arrays).
