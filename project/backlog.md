<!-- markdownlint-disable MD007 -- Unordered list indentation -->
<!-- markdownlint-disable MD010 -- No hard tabs -->
<!-- markdownlint-disable MD033 -- No inline html -->
<!-- markdownlint-disable MD055 -- Table pipe style [Expected: leading_and_trailing; Actual: leading_only; Missing trailing pipe] -->
<!-- markdownlint-disable MD041 -- First line in a file should be a top-level heading -->

<!-- TOC ignore:true -->
# Project backlog

This is a product backlog just for pre-v1.0.0 release. After that, bugs, features, and enhancements will be managed in Github Issues, and/or [todo.md](../todo.md)

<!-- TOC ignore:true -->
## Table of contents
<!-- TOC -->

- [Conventions](#conventions)
- [First steps](#first-steps)
- [Backlog](#backlog)
	- [Todo](#todo)
	- [Bugs](#bugs)
	- [New features and enhancements](#new-features-and-enhancements)
	- [Deferred](#deferred)
	- [Canceled](#canceled)
	- [Done](#done)
		- [Done - First steps](#done---first-steps)
		- [Done - Bugs](#done---bugs)
		- [Done - New features and enhancements](#done---new-features-and-enhancements)

<!-- /TOC -->

## Conventions

In each section, items are listed approximately from newest to oldest.

| Icon | Status
| :--: | :--
| 🔘   | Not started
| 🛠️   | Started, and/or partially complete
| ✋   | Defer
| ✅   | Complete
| 🚫   | Canceled

## First steps

## Backlog

### Todo

- 🔘 Verify the Windows paths on real Windows hardware: symlink recreate (file vs dir flag, privilege), junction reparse-buffer write, and `.lnk` `TargetPath` set/save. (Because...built and cross-compiled from Linux.)

### Bugs

### New features and enhancements

- 🔘 CICD process: Rigorous testing of all features, possible combinations (valid and invalid), except not on live files.

- 🔘 `--follow-links` to descend into directory symlinks (with cycle detection).

- 🔘 `-0` / `--print0` NUL-separated output for scripting.

- 🔘 `--renormal-relative` / `--renormal-absolute`: Optional re-normalization of rewritten targets. These can be used with or without renaming options.

- 🔘 `--confirm`: Confirm-before-write mode (interactive `y/n`) as an alternative to blind apply.
	- Behave similar to a shell script that does something like:

		~~~bash
		repoint-symlink --dry-run "$@" | less -FX
		echo; read -r -p "Continue (y|N)?: " answer; echo
		[[ "${answer,,}" == "y" ]] || exit 1
		repoint-symlink "$@"
		~~~

### Deferred

### Canceled

### Done

#### Done - First steps

- ✅ Scaffold Go project, build/cross-compile matrix, CICD engine + config + test harness.
- ✅ Core CLI: custom arg parser (prefix abbreviation, `=`/space values, positional START/FROM/TO).
- ✅ Filters: include/exclude regex, name and wholename globs, and re-include, as one ordered pipeline.
- ✅ Recursive walk collecting symlinks, loop-safe (no dir-symlink follow), `--max-depth`.
- ✅ Repoint engine: regex `--from` + template `--to` (`$1`), literal `--literal`, apply-by-default, or show first with `--dry-run`.
- ✅ POSIX (Linux/BSD/macOS) symlink read/atomic rewrite.
- ✅ Windows symlink + NTFS junction + `.lnk` read/rewrite (compile-verified; see verify item below).
- ✅ Go unit tests (args/filters/transform/walk) + integration harness over scratch symlink trees.

#### Done - Bugs

#### Done - New features and enhancements

- ✅ `--inc-target` / `--exc-target`: select links by their current target (where they point), not just their own path. Reuses the filter engine as a second ordered pipeline over the target. `--inc`/`--exc` short spellings preserved via an exact-alias table.

- ✅ `--no-cross-device`: don't descend into directories on another filesystem (`find -xdev` style). Per-OS device probe (POSIX `st_dev`, Windows volume serial); prunes at directory boundaries during the walk.

- ✅ Selection engine split into its own reusable package (`source/filter/`), independent of the CLI, so a later file-lister project can use it as-is. The CLI keeps a thin flag-to-rule adapter.

- ✅ Fuzz and security test suites: arg parser, glob translator, regex engine, and the selection pipeline under random input; link-follow safety, symlink-cycle safety, and timeout-bounded matching.
