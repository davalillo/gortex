# Feature: MQL tree-sitter extractor (bespoke tier)

Status: planned — not started
Scope decision: two PRs. This feature covers PR 1 (extractor). PR 2 (LSP
spec for mql-lsp-server in `internal/semantic/lsp/registry.go` +
`docs/lsp.md`) is a separate change, tracked separately.
Grammar decision: fork mskelton/tree-sitter-mql5 under the `gortexhq` org
with a Go binding added (same precedent as tree-sitter-dart/swift/sql
already in go.mod). The fork's grammar.js is extended EMPIRICALLY, not
speculatively: the stock grammar (tree-sitter-cpp + `input`) is run
over a real .mq5/.mqh corpus (mql-language-server test corpus), ERROR
nodes are counted per construct, and only proven-broken constructs get
new rules — each with a fork-side test. Mql5Grammar.g4 (the user's
ANTLR grammar) is the checklist spec for the audit. Known audit
candidates: sinput, interface, `input group "..."`, #property, #import
DLL blocks.

## Background

- Gortex bespoke tier: `internal/parser/tsitter/<lang>/` is a ~10-line
  shim re-exporting the grammar's Go bindings; the extractor in
  `internal/parser/languages/<lang>.go` runs one combined S-expression
  query and dispatches on capture names. Reference: golang.go,
  cpp.go.
- mskelton/tree-sitter-mql5 ships node+rust bindings only, so a fork
  must add `bindings/go/binding.go` + its own go.mod, then be tagged.
- Extensions `.mq4` / `.mq5` / `.mqh` are unclaimed by all existing
  extractors (no collisions). Modern .mq4 shares MQL5 syntax
  (unified compiler since build 600), so one grammar serves both.
  Pre-600 legacy .mq4 is a C subset and parses as such.
- Dialect (mql4 vs mql5): syntactically irrelevant — the dialect split
  in mql-language-server is semantic (builtins, misuse rules), proven
  by its content-sniffing markers all being tokens the C++ grammar
  parses fine. PR 1 optionally stamps `Meta["dialect"]` (extension
  based; .mqh sniffed with the server's Mql5Tokens markers). The real
  dual support lives in PR 2's LanguageIDs routing (.mq4->mql4,
  .mq5->mql5, .mqh -> server-side LanguageDetection).
- Rust/go.mod module path of this repo is `github.com/zzet/gortex`.

## Tasks

1. [ ] Publish grammar fork: gortexhq/tree-sitter-mql5 with
   `bindings/go` (binding.go + go.mod), tag v0.1.0. May require user's
   push access / gh CLI. Then add module to Gortex go.mod; `go build`
   passes with CGO.
   1a. [ ] Corpus-driven precision audit: parse mql-language-server
       corpus + real EA/indicator samples (MIXED .mq4, .mq5, .mqh, incl.
       one legacy pre-600 .mq4) with the stock grammar, count ERROR
       nodes per construct, diff against Mql5Grammar.g4 surface;
       extend grammar.js only for proven gaps, with per-rule fork tests;
       re-tag before pinning the version in go.mod.
2. [ ] Add `internal/parser/tsitter/mql/mql.go` shim re-exporting the
   new binding (pattern: tsitter/dart/dart.go).
3. [ ] Write `internal/parser/languages/mql.go`: combined query for
   functions (incl. event handlers), class/struct/enum/interface types,
   methods with member_of, `input`/`sinput`/`extern` variables,
   `#include` -> imports edges, calls (plain + method calls), macro
   invocations excluded from call edges. Register in register.go
   before forest languages with extensions .mq4/.mq5/.mqh.
4. [ ] Tests: happy path, empty input (mandated), golden corpus of
   real EA/indicator snippets (classes, templates, `input` params,
   Controls event-map macros from the user's LSP corpus).
5. [ ] Docs: languages.md — add MQL to core/bespoke table + dedicated
   section ("What is not covered" honesty block); update extension
   collision table if needed.
6. [ ] Close: work-unit commits per task recorded as evidence here;
   PR 1 opened by user decision.

## Evidence

(commits appended per task)

## Follow-ups

- PR 2: LSP ServerSpec `mql-lsp-server` (.NET 10 dependency documented),
  LanguageIDs routing .mq4->mql4 / .mq5->mql5 / .mqh -> server-side
  LanguageDetection (its unknown-languageId fallback already handles
  mqh sniffing), extension routing, docs/lsp.md server matrix + install.
- Optional differential harness vs mql-language-server documentSymbol.
