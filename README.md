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

Moving a directory, renaming a mount, or restructuring a tree leaves a scatter of symlinks pointing at the old location. Fixing them by hand is tedious and error-prone. `repoint-symlink` finds them all and repoints them in one pass, with a dry run to check first.

## Features

- Recursive search from a start folder (default: current directory), never following directory symlinks, so it is loop-safe.
- Select links with repeatable `--include` / `--exclude` regexes (PCRE-level: lookaround, backreferences, inline `(?i)` case flag) and `--name` / `--iname` filename globs.
- Rewrite targets with a regex `--from` and a template `--to` (`$1`, `${name}` capture references), or a literal replace with `-F`.
- Applies by default; `--dry-run` previews every before/after with nothing written.
- Cross-platform. On Windows it also repoints NTFS junctions and `.lnk` shortcut targets.

## Usage

```
repoint-symlink [START] [FROM] [TO] [options]
```

`START`, `FROM`, and `TO` may be given positionally (in that order) or as `--from` / `--to`. With no `--from`, matching links are just listed.

### Filters

Filters select which links to act on and match against the link's own path. Multiples of one kind OR together; different kinds AND.

| Flag | Matches
| :-- | :--
| `--inc[lude]=REGEX` | keep links whose path matches (repeatable)
| `--exc[lude]=REGEX` | drop links whose path matches (repeatable)
| `--name=GLOB`       | keep links whose basename matches, case-sensitive
| `--iname=GLOB`      | same, case-insensitive
| `--max-depth=N`     | limit recursion depth (1 = direct children)

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
