<!-- markdownlint-disable MD007 -- Unordered list indentation -->
<!-- markdownlint-disable MD010 -- No hard tabs -->
<!-- markdownlint-disable MD033 -- No inline html -->
<!-- markdownlint-disable MD055 -- Table pipe style [Expected: leading_and_trailing; Actual: leading_only; Missing trailing pipe] -->
<!-- markdownlint-disable MD041 -- First line in a file should be a top-level heading -->

<!-- TOC ignore:true -->
# Project design

<!-- TOC ignore:true -->
## Table of contents
<!-- TOC -->

- [Goal](#goal)
- [Architecture](#architecture)
	- [Language and stack](#language-and-stack)
	- [Logical code organization](#logical-code-organization)
	- [API](#api)
	- [Decisions](#decisions)

<!-- /TOC -->

## Goal

A small cross-platform CLI that finds symlinks under a start folder, selects a subset with include/exclude regex and name-glob filters, and rewrites their targets via a regex-and-template substitution. On Windows it additionally handles NTFS junctions and `.lnk` shortcuts.

## Architecture

### Language and stack

- Go (single static binary, `CGO_ENABLED=0`, cross-compiled to linux/macOS/windows on amd64+arm64).
- `github.com/dlclark/regexp2` for PCRE-level regex (lookaround, backrefs, inline flags) - RE2 in the stdlib is not enough.
- Windows only: `golang.org/x/sys/windows` (reparse points / junctions) and `github.com/go-ole/go-ole` (WScript.Shell for `.lnk`).

### Logical code organization

The CLI is `package main` under `source/`; the selection engine is a standalone package under `source/filter/`:

| File | Responsibility
| :-- | :--
| `main.go`          | entry, version, help/examples, orchestration
| `args.go`          | custom arg parser (prefix abbreviation, `=`/space values, positional 1/2/3)
| `selection.go`     | map the parsed selection flags onto `filter` rule specs
| `filter/filter.go` | the reusable selection engine: ordered narrow/subtract/re-add pipeline over regex and find-style globs, plus the timeout-bounded regex compiler
| `walk.go`          | recursive traversal collecting `LinkEntry` (no dir-symlink follow)
| `link.go`          | shared `LinkEntry` / `LinkKind` types
| `link_unix.go`     | classify + read/write symlinks (`!windows`)
| `link_windows.go`  | classify + read/write symlinks, junctions, `.lnk` (`windows`)
| `repoint.go`       | from/to transform, per-entry processing, dry-run, summary

### API

Command line only:

```
repoint-symlink [START] [FROM] [TO] [options]
```

See `README.md` for the full flag list. `START`/`FROM`/`TO` are positional aliases for the start folder, `--from`, and `--to`.

### Decisions

- **Apply by default**, `--dry-run` opt-in preview. Destructive-but-reversible; a dry run is one flag away. `--confirm` is the middle ground: it previews the whole plan, then prompts once before writing anything (a full-plan gate, not per-link). `process` builds the plan first, then previews/applies, so the same plan drives dry-run, confirm, and normal apply. `--confirm` and `--print0` (machine output) are mutually exclusive.
- **`--from` is a regex, `--to` a template**; `-F`/`--literal` switches to literal replace-all.
- **Path filters vs target filters.** Most filters match the link's own path; `--from`/`--to` already select by target implicitly (a link whose target does not match is left unchanged). `--inc-target`/`--exc-target` add explicit target selection - the same narrow/subtract engine run over the current target instead of the path. Path and target rules form two independent pipelines and a link must satisfy both. Adding `--inc-target`/`--exc-target` collided with the `--inc`/`--exc` prefix abbreviations, so those short spellings are kept working by a small exact-alias table.
- **Ordered filter pipeline, one fixed effect per flag.** Every selection flag is one rule, kept in command-line order and evaluated left to right. Each flag's effect is fixed and independent of position, so the set reads one flag at a time. Include and the name/wholename globs narrow, exclude subtracts, and re-include re-admits from the original scan. Re-include is the only widener, and the only way to bring back something an exclude dropped. An earlier model let a plain include widen after an exclude; a dedicated widener replaced it so every flag means the same thing wherever it sits.
- **Globs are find-style**, translated to an anchored regex where `*`/`?` span `/` (matches find's `-wholename`); `--[i]wholename` matches the whole (slash-normalized) path, `--[i]name` the basename.
- **Directory symlinks are not followed by default** (loop-safe via `WalkDir`, which never follows). `-L`/`--follow-links` opts into following them with a separate manual walk that records each directory's canonical path in a visited set, so a cycle - or a subtree reachable two ways - is walked exactly once. The default path keeps using `WalkDir` unchanged; following is a distinct, opt-in code path.
- **Target re-normalization is a post-step**, separate from `--from`/`--to`. `--renormal-relative`/`--renormal-absolute` rewrite each (possibly already-rewritten) target relative to the link's own directory, or as a cleaned absolute path. It runs after the substitution and needs the link's own path, so it lives in `process` rather than in `transform`. It normalizes the logical path only (no `EvalSymlinks`), so the link still points at the same place. The two directions are mutually exclusive, and either enables edit mode on its own so it works without `--from`.
- **Symlink rewrite is atomic on POSIX** (create-beside + rename). Windows symlinks are remove+recreate; junctions overwrite the reparse buffer in place; `.lnk` targets are set via the shell object.
- **The selection engine is its own package** (`source/filter/`), with no ties to the CLI. It takes generic rule specs (narrow/subtract/re-add over a regex or a find-style glob) rather than knowing about any flag. This keeps the pipeline testable on its own and lets a later file-lister project reuse it unchanged. The CLI keeps only a thin adapter that turns its flags into those specs.
