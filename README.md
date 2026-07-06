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

- Select which links to touch, matched against the link's own path.
	- Name and path globs: `--name` / `--iname`, `--wholename` / `--iwholename`.
	- Repeatable `--include` / `--exclude` / `--re-include` regexes.
	- Flags apply left to right, so you reason about them one at a time (see [Filters](#filters)).
	- PCRE-level regex: lookaround, backreferences, inline `(?i)` case flag.

- Rewrite targets with a regex `--from='findstr'` and `--to='replacestr'`.
	- `$1`, `${name}` capture references in `--to`.
	- `-F` / `--literal` matches `--from` as a plain literal instead of a regex, replacing every occurrence.
		- Handy for Windows paths, where `\` and `:` would otherwise be regex-special and need escaping - e.g. `-F --from='C:\Old' --to='C:\New'` just works.

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

Filters pick which links to act on. Each one matches the link's own path, not its target.

How the pipeline reads:

- Every filter flag is one rule.
- Rules run left to right, starting from "all links kept".
- Each flag's effect is fixed, so you read them one at a time.
- Order still matters: a later rule acts on whatever the earlier ones left.

What each kind does:

- **Narrow** - keep only links that also match.
	- `--include` (regex), and every name/wholename glob.
- **Subtract** - drop links that match.
	- `--exclude` (regex).
- **Widen** - re-admit links from the original scan.
	- `--re-include` (regex), the only widener.
	- Brings back even a link a previous `--exclude` dropped.

Globs vs regexes:

- Globs are [`find`](https://man7.org/linux/man-pages/man1/find.1.html)-style.
	- `*` and `?` span `/`, so `--wholename` behaves like find's `-wholename`.
	- Quote them so the shell doesn't expand them.
- Regexes are PCRE-level (lookaround, backreferences, inline `(?i)`).

| Flag | Matches
| :-- | :--
| `--inc[lude]=REGEX`    | Keep only links whose path also matches (repeatable; narrows).
| `--exc[lude]=REGEX`    | Drop links whose path matches (repeatable; subtracts).
| `--re-inc[lude]=REGEX` | Re-add links matching this from the original scan (repeatable; widens).
| `--name=GLOB`          | Keep only links whose basename matches, case-sensitive.
| `--iname=GLOB`         | Same, case-insensitive.
| `--wholename=GLOB`     | Keep only links whose whole path matches, case-sensitive.
| `--iwholename=GLOB`    | Same, case-insensitive.
| `--max-depth=N`        | Limit recursion depth (1 = direct children).

### Editing the target

| Flag | Effect
| :-- | :--
| `--from=REGEX`  | pattern to match inside each target
| `--to=TEMPLATE` | replacement; `$1` / `${name}` reference `--from` captures
| `-F`, `--literal` | treat `--from` as a plain literal, not a regex (replace all occurrences)
| `-n`, `--dry-run` | preview; write nothing

## Examples

```sh
# Preview repointing every symlink under /srv that points into /mnt/old
repoint-symlink /srv --from='/mnt/old' --to='/mnt/new' -n

# Do it for real
repoint-symlink /srv --from='/mnt/old' --to='/mnt/new'

# Only *.conf links, skip anything under a 'backup' dir
repoint-symlink . --iname='*.conf' --exc='/backup/' --from='v1' --to='v2'

# Whole-path glob (find style): links anywhere under an 'etc' dir
repoint-symlink / --wholename='*/etc/*'

# Drop every .tmp link, then rescue the ones under an 'assets' dir
repoint-symlink /srv --exc='\.tmp$' --re-inc='/assets/.*\.tmp$'

# Regex capture: /opt/app-1.2.3 -> /opt/app/1.2.3
repoint-symlink /srv --from='/opt/app-(\d+\.\d+\.\d+)' --to='/opt/app/$1'

# Literal replace (no regex) - handy for Windows paths
repoint-symlink . -F --from='C:\Old' --to='C:\New'

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
