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

Everything is `package main` under `source/`:

| File | Responsibility
| :-- | :--
| `main.go`         | entry, version, help/examples, orchestration
| `args.go`         | custom arg parser (prefix abbreviation, `=`/space values, positional 1/2/3)
| `filter.go`       | compile + apply include/exclude/name/iname selection
| `walk.go`         | recursive traversal collecting `LinkEntry` (no dir-symlink follow)
| `link.go`         | shared `LinkEntry` / `LinkKind` types
| `link_unix.go`    | classify + read/write symlinks (`!windows`)
| `link_windows.go` | classify + read/write symlinks, junctions, `.lnk` (`windows`)
| `repoint.go`      | from/to transform, per-entry processing, dry-run, summary

### API

Command line only:

```
repoint-symlink [START] [FROM] [TO] [options]
```

See `README.md` for the full flag list. `START`/`FROM`/`TO` are positional aliases for the start folder, `--from`, and `--to`.

### Decisions

- **Apply by default**, `--dry-run` opt-in preview. Destructive-but-reversible; a dry run is one flag away.
- **`--from` is a regex, `--to` a template**; `-F` switches to literal replace-all.
- **Filters match the link's own path**, not its target - `--from`/`--to` already select by target implicitly (a link whose target does not match is left unchanged). Optional target-matching filters are a backlog item.
- **Include = OR, exclude = subtract; different filter kinds AND.** Standard grep-style mental model.
- **Symlink rewrite is atomic on POSIX** (create-beside + rename). Windows symlinks are remove+recreate; junctions overwrite the reparse buffer in place; `.lnk` targets are set via the shell object.
