<!-- markdownlint-disable MD007 -- Unordered list indentation -->
<!-- markdownlint-disable MD010 -- No hard tabs -->
<!-- markdownlint-disable MD033 -- No inline html -->
<!-- markdownlint-disable MD055 -- Table pipe style [Expected: leading_and_trailing; Actual: leading_only; Missing trailing pipe] -->
<!-- markdownlint-disable MD041 -- First line in a file should be a top-level heading -->
<div align="center">

![Go](https://img.shields.io/badge/Go-00ADD8?logo=go&logoColor=white)
![License: GPL v2](https://img.shields.io/badge/License-GPLv2-blue.svg)
![Lifecycle](https://img.shields.io/badge/Lifecycle-Alpha-orange)
![Support](https://img.shields.io/badge/Support-Maintained-brightgreen)
![Status: Passing](https://img.shields.io/badge/Status-Passing-brightgreen)

</div>

<!-- TOC ignore:true -->
# repoint-symlink

Find symlinks anywhere under a folder and rewrite where they point. Filter which links to touch with any number of include/exclude regexes and name globs, then edit their targets with a regex-and-template substitution. Linux, macOS, and Windows; on Windows it also handles NTFS junctions and `.lnk` shortcuts.

<!-- TOC ignore:true -->
## Table of contents

<!-- TOC -->

- [Why](#why)
- [Features](#features)
- [Usage](#usage)
	- [Filters](#filters)
	- [Editing the target](#editing-the-target)
- [Examples](#examples)
- [Installing](#installing)
- [Building from source](#building-from-source)
- [Copyright and license](#copyright-and-license)

<!-- /TOC -->

## Why

Moving a directory, renaming a mount, or restructuring a tree leaves a scatter of symlinks pointing at the old location. Fixing them by hand is tedious and error-prone. `repoint-symlink` finds them all and repoints them in one pass - with an optional dry run to check first.

## Features

- Recursive search from a start folder (default: current directory).

- Select links by name, via one or more of:
	- `--name='*Name*'` / `--iname='*name*'` filename wildcards.
	- `--wholename='*/Path*/*name'` / `--iwholename='*/path*/*name'` full-path wildcards.
	- Repeatable, nested `--include=""` / `--exclude=""` regexes:
		- The first `--include`, or following only other `--include`s, can only narrow results for a given path or leave the same, but not expand.
		- But an `--include` following an `--exclude` can actually *expand* the resulting list of files, by bringing previously excluded matches back in, for example further down a filesystem hierarchy.
		- `--exclude` can only ever narrow a result set or leave it the same.
		- The regex engine is PCRE-level - supporting lookaround, backreferences, inline `(?i)` case flag, etc.

- Rewrite targets with a regex `--from='findstr'` and `--to='replacestr'`.
	- Including `$1`, `${name}` capture references.
	- Literal replace with `-F`.

- `--dry-run` previews every before/after with nothing written.
	- By default, renames are applied without preview.

- Doesn't follow directory symlinks (without a flag), so it is loop-safe.

- Cross-platform. On Windows it also repoints NTFS junctions and `.lnk` shortcut targets.

## Usage

```bash
repoint-symlink [START] [FROM] [TO] [options]
```

`START`, `FROM`, and `TO` may be given positionally (in that order) or as `--from` / `--to`. With no `--from`, matching links are just listed.

### Filters

Filters select which links to act on and match against the link's own path. Every filter is one rule in a single ordered pipeline, evaluated left to right from "everything kept" - so their order on the command line matters:

- A keep-rule (`--include` / `--name` / `--iname` / `--wholename` / `--iwholename`) that follows another keep-rule, or is first, **narrows** the set (result AND match).
- A keep-rule that follows an `--exclude` can **expand** the set (result OR match), bringing back links a previous exclude had dropped - for example a single branch further down a hierarchy.
- `--exclude` only ever **narrows** (result AND NOT match).

Globs are [`find`](https://man7.org/linux/man-pages/man1/find.1.html)-style: `*` and `?` span `/` (so `--wholename` behaves like find's `-wholename`), and patterns must be quoted so the shell doesn't expand them. Regexes are PCRE-level.

| Flag | Matches
| :-- | :--
| `--inc[lude]=REGEX` | Keep links whose path matches (repeatable; narrows, or expands after an `--exclude`).
| `--exc[lude]=REGEX` | Drop links whose path matches (repeatable; only narrows).
| `--name=GLOB`       | Keep links whose basename matches, case-sensitive.
| `--iname=GLOB`      | Same, case-insensitive.
| `--wholename=GLOB`  | Keep links whose whole path matches, case-sensitive.
| `--iwholename=GLOB` | Same, case-insensitive.
| `--max-depth=N`     | Limit recursion depth (1 = direct children).

### Editing the target

| Flag | Effect
| :-- | :--
| `--from=REGEX`  | pattern to match inside each target
| `--to=TEMPLATE` | replacement; `$1` / `${name}` reference `--from` captures
| `-F`, `--fixed` | treat `--from` as a literal string (replace all occurrences)
| `-n`, `--dry-run` | preview; write nothing

## Examples

```sh
# Preview repointing every symlink under /srv that points into /mnt/old
repoint-symlink /srv --from='/mnt/old' --to='/mnt/new' -n

# Do it for real
repoint-symlink /srv --from='/mnt/old' --to='/mnt/new'

# Only *.conf links, skip anything under a 'backup' dir
repoint-symlink . --iname='*.conf' --exc='/backup/' --from='v1' --to='v2'

# Regex capture: /opt/app-1.2.3 -> /opt/app/1.2.3
repoint-symlink /srv --from='/opt/app-(\d+\.\d+\.\d+)' --to='/opt/app/$1'

# List every symlink two levels deep
repoint-symlink /srv --max-depth=2
```

## Installing

Grab a prebuilt archive for your platform from the releases, or build from source below. The binary is self-contained with no runtime dependencies.

## Building from source

Requires Go (see `.go-version`).

```sh
cd source
make local      # native build -> ./repoint-symlink
make test       # go test ./...
make release    # cross-compile every platform into ./dist
```

## Copyright and license

> Copyright © 2026 Jim Collier (ID: 1cv◂‡Vᛦ)<br />
> Licensed under [GNU GPL v2 Or Later License](https://spdx.org/licenses/GPL-2.0-or-later.html) license. No warranty.
