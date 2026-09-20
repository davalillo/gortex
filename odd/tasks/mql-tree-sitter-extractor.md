# Feature: MQL tree-sitter extractor (bespoke tier)

Status: complete (PR 1 scope). Commits on feat/mql-tree-sitter-extractor:
e7b281ce (fork pin + shim), 3fea73ae (extractor + registration + tests),
c29e260a (docs). PR opening is the user's decision.
Environmental note: the full languages-package test suite aborts with a
pre-existing glibc heap corruption inside tree-sitter cgo (moving crash
site), reproduced identically at base commit 2b5480bf — unrelated to
this feature; MQL tests pass in isolation and all meta-tests pass.
Worth a separate issue.
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

1. [x] Publish grammar fork: davalillo/tree-sitter-mql5 v0.1.0
   (gortexhq had no push access for this account; migration there is a
   one-line go.mod change if ever wanted). Go binding via cgo with a
   C-only scanner stub (MQL5 has no raw strings; avoids g++ dep).
   Corpus-driven audit result: stock grammar 46/53 corpus files clean,
   37 ERROR nodes -> after extensions 47/53 clean, 10 ERROR nodes, all
   remaining ones macro-expansion-only fixtures (EA_INPUT/EVENT_MAP,
   deliberately invalid pre-expansion source) -> error tolerance by
   design, documented in fork README. Extensions added: sinput,
   interface_specifier, color_literal, datetime_literal. 98/98
   tree-sitter tests pass incl. all upstream C++ corpus (no C++
   regressions). Tagged v0.1.0.
   Commit: e7b281ce (go.mod pin + tsitter/mql shim, branch
   feat/mql-tree-sitter-extractor).
   NOTE: toolchain for regeneration is pinned to tree-sitter-cli
   0.20.8-era DSL; node_modules for cpp/c grammars resolved manually
   (see fork README).
2. [x] Add internal/parser/tsitter/mql/mql.go shim re-exporting the
   binding (dart.go pattern). Full CGO build passes; tsitter tests
   pass. Commit: e7b281ce.
3. [x] Write internal/parser/languages/mql.go: combined query (classes,
   structs, enums, interfaces, methods with member_of, free + template
   functions, includes, macros, calls), input/sinput/extern variables
   (Meta storage_class), dialect stamping on the file node
   (mql-language-server Mql5Tokens markers; .mqh default mql4).
   Registered in register.go before forest languages. Commit: 3fea73ae.
4. [x] Tests: mql_test.go — happy path, empty input, dialect table;
   7/7 green in isolation. Full languages suite blocked by the
   pre-existing environmental SIGABRT (see status note).
   Constructor-as-method matches C#/Java behavior (accepted).
   Commit: 3fea73ae.
5. [x] Docs: languages.md — count 257, bespoke tier ~31, core matrix
   row, at-a-glance updates, MQL specifics section with "not covered"
   boundaries. Commit: c29e260a.
6. [x] Close: work-unit commits e7b281ce, 3fea73ae, c29e260a recorded
   as evidence. PR 1 opening is the user's decision.

## Evidence

(commits appended per task)

## Follow-ups

- PR 2: LSP ServerSpec `mql-lsp-server` (.NET 10 dependency documented),
  LanguageIDs routing .mq4->mql4 / .mq5->mql5 / .mqh -> server-side
  LanguageDetection (its unknown-languageId fallback already handles
  mqh sniffing), extension routing, docs/lsp.md server matrix + install.
- Optional differential harness vs mql-language-server documentSymbol.
