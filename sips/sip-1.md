---
sip: 1
title: SIP Purpose and Guidelines
status: Living
type: Meta
author: Salty maintainers
created: 2026-09-03
---

## What is a SIP?

SIP stands for **Salty Improvement Proposal**. A SIP is a design document
providing information to the Salty community, or describing a new feature for
the Salty language or its transpiler. The SIP should provide a concise technical
specification and a rationale for the feature.

SIPs are the primary mechanism for proposing new language features, collecting
community input, and documenting design decisions. They are modelled on
[Ethereum Improvement Proposals (EIPs)](https://eips.ethereum.org/).

## SIP types

- **Standards Track** — a change to the Salty language: new syntax, new types,
  new lowering behaviour, or anything affecting emitted Solidity.
- **Meta** — a process or guideline about Salty itself (like this document).

## SIP workflow

1. **Idea → Issue.** Open a GitHub issue using the *SIP proposal* template and
   apply the `sip` label. Discussion happens on the issue.
2. **Draft.** Open a pull request adding `sips/sip-N.md` using the next
   available number and the format below.
3. **Review.** Maintainers and the community review the draft.
4. **Accepted / Final.** Once merged, a Standards Track SIP is `Accepted` until
   implemented, then `Final`. Meta SIPs are `Living`.

## Statuses

| Status | Meaning |
| ------ | ------- |
| Draft | Under discussion, not yet merged. |
| Accepted | Merged and agreed, implementation pending. |
| Final | Merged and implemented. |
| Living | Continually updated (Meta SIPs). |
| Withdrawn | Abandoned. |

## SIP format

Each SIP is a Markdown file `sips/sip-N.md` beginning with a YAML preamble:

```
---
sip: <number>
title: <short title>
status: Draft | Accepted | Final | Living | Withdrawn
type: Standards Track | Meta
author: <name or handle>
created: <YYYY-MM-DD>
---
```

Followed by these sections:

- **Abstract** — one paragraph summary.
- **Motivation** — why this is needed.
- **Specification** — the precise, testable rules.
- **Rationale** — why the design is the way it is; alternatives considered.
- **Backwards Compatibility** — impact on existing Salty code.
- **Reference Implementation** — where the code lives, once implemented.

## Citing SIPs in code

Code implementing a SIP should reference it in a comment so the specification and
implementation stay connected, e.g.:

```go
// SIP-2: decimal(N) lowers to uint256 scaled by 10^N.
```
