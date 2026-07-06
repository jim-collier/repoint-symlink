<!-- markdownlint-disable MD007 -- Unordered list indentation -->
<!-- markdownlint-disable MD010 -- No hard tabs -->
<!-- markdownlint-disable MD033 -- No inline html -->
<!-- markdownlint-disable MD055 -- Table pipe style [Expected: leading_and_trailing; Actual: leading_only; Missing trailing pipe] -->
<!-- markdownlint-disable MD041 -- First line in a file should be a top-level heading -->

<!-- TOC ignore:true -->
# Project backlog

This is a product backlog just for pre-v1.0.0 release. After that, bugs, features, and enhancements will be mananged in Github Issues, and/or [todo.md](../todo.md)

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

- 🔘 `--no-cross-dev[ice]`: Don't cross underlying filesystem devices.

- 🔘 `--inc-target` / `--exc-target`: Optional *target*-matching regex filters to select links by where they currently point, not just their own path.

- 🔘 `--follow-links` to descend into directory symlinks (with cycle detection).

- 🔘 `-0` / `--print0` NUL-separated output for scripting.

- 🔘 `--renormal-relative` / `--renormal-absolute`: Optional re-normalization of rewritten targets.

- 🔘 `--scan-then-confirm`: Confirm-before-write mode (interactive `y/n`) as an alternative to blind apply.
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
- ✅ Filters: include/exclude PCRE-level regex + name/iname globs (OR within a kind, AND across kinds).
- ✅ Recursive walk collecting symlinks, loop-safe (no dir-symlink follow), `--max-depth`.
- ✅ Repoint engine: regex `--from` + template `--to` (`$1`), literal `-F`, apply-by-default, or show first with `--dry-run`.
- ✅ POSIX (Linux/BSD/macOS) symlink read/atomic rewrite.
- ✅ Windows symlink + NTFS junction + `.lnk` read/rewrite (compile-verified; see verify item below).
- ✅ Go unit tests (args/filters/transform/walk) + integration harness over scratch symlink trees.

#### Done - Bugs

#### Done - New features and enhancements
