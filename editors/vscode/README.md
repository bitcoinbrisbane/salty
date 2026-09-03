# Salty for VS Code

Language support for [Salty](https://github.com/bitcoinbrisbane/salty) — a
Solidity-like language that transpiles to Solidity.

## Features

- **Syntax highlighting** for `.salty` files: keywords, built-in types,
  `decimal` fixed-point literals ([SIP-2](https://github.com/bitcoinbrisbane/salty/blob/main/sips/sip-2.md)),
  strings, and comments.
- **Transpile command** — `Salty: Transpile to Solidity` (also on the editor
  title bar for `.salty` files). Runs the `salty` CLI on the current file and
  opens the generated Solidity beside it.

## Requirements

The transpile command shells out to the `salty` CLI. Install it:

```bash
go install github.com/bitcoinbrisbane/salty/cmd/salty@latest
```

If `salty` isn't on your `PATH`, set the path in settings:

```json
"salty.cliPath": "/absolute/path/to/salty"
```

## Settings

| Setting | Default | Description |
| ------- | ------- | ----------- |
| `salty.cliPath` | `salty` | Path to the `salty` CLI executable. |

## Development

```bash
npm install
npm run compile      # or: npm run watch
```

Press <kbd>F5</kbd> in VS Code to launch an Extension Development Host with the
extension loaded, then open any file from `examples/`.

## Packaging & installing

Build a shareable `.vsix`:

```bash
npm run package      # produces salty-<version>.vsix
```

Install it into VS Code:

```bash
code --install-extension salty-0.1.0.vsix
```

…or in VS Code: **Extensions** view → `…` menu → **Install from VSIX…**.

To publish to the Marketplace (requires a publisher and token):

```bash
npm run publish
```

The TextMate grammar (`syntaxes/salty.tmLanguage.json`) mirrors the keyword and
type sets in the transpiler's lexer; keep them in sync when the language grows.
