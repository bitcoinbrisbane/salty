# Salty Improvement Proposals (SIPs)

SIPs are design documents that describe new features for the Salty language and
its transpiler. They are modelled on [Ethereum's EIPs](https://eips.ethereum.org/).

## How to propose a SIP

1. **Open a design issue.** Use the *SIP proposal* issue template on GitHub to
   describe the idea, motivation, and a sketch of the design. Apply the `sip`
   label. This is where discussion happens.
2. **Draft the SIP.** Once there's rough consensus, open a pull request adding
   `sips/sip-N.md` (next free number), following the format in
   [SIP-1](./sip-1.md).
3. **Review and merge.** Maintainers review; when accepted, the SIP is merged
   with status `Final` (or `Accepted` if implementation is pending).
4. **Implement.** Code that implements a SIP cites it in comments, e.g.
   `// SIP-2: decimal fixed-point type`.

## Index

| SIP | Title | Status |
| --- | ----- | ------ |
| [1](./sip-1.md) | SIP Purpose and Guidelines | Living |
| [2](./sip-2.md) | Decimal fixed-point type | Final |
