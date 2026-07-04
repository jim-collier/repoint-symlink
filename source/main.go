//	Copyright © 2026 Jim Collier (ID: 1cv◂‡Vᛦ)
//	Licensed under the GNU General Public License v2.0 or later. Full text at:
//		https://spdx.org/licenses/GPL-2.0-or-later.html
//	SPDX-License-Identifier: GPL-2.0-or-later

package main

import (
	"fmt"
	"os"
)

// version is stamped at build time via -ldflags "-X main.version=...".
var version = "dev"

const (
	appName = "repoint-symlink"
	author  = "Jim Collier (ID: 1cv◂‡Vᛦ)"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(argv []string) int {
	o, err := parseArgs(argv)
	if err != nil {
		errorf("%v", err)
		errorf("try --help")
		return 2
	}

	switch {
	case o.showVersion:
		fmt.Println(version)
		return 0
	case o.showHelp:
		printHelp()
		return 0
	case o.showExamples:
		printExamples()
		return 0
	}

	f, err := compileFilters(o)
	if err != nil {
		errorf("%v", err)
		return 2
	}
	r, err := buildRepointer(o)
	if err != nil {
		errorf("%v", err)
		return 2
	}

	// Start dir must resolve to a directory.
	fi, err := os.Stat(o.dir)
	if err != nil {
		errorf("cannot read start folder %q: %v", o.dir, err)
		return 1
	}
	if !fi.IsDir() {
		errorf("start folder %q is not a directory", o.dir)
		return 1
	}

	entries, err := collectLinks(o.dir, o.maxDepth)
	if err != nil {
		errorf("walk failed: %v", err)
		return 1
	}

	_, failed := process(o, f, r, entries)
	if failed {
		return 1
	}
	return 0
}

func warnf(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "warning: "+format+"\n", a...)
}

func errorf(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", a...)
}

func printHelp() {
	fmt.Printf(`%s %s - find symlinks and repoint where they point.

Usage:
  %s [START] [FROM] [TO] [options]
  %s [START] --from=REGEX --to=TEMPLATE [filters] [options]

Positional (all optional):
  START   folder to search (default: current directory)
  FROM    --from value (regex, or literal with -F)
  TO      --to value (replacement template)

Filters (select which links to act on, matched against the link's own path;
multiples of one kind OR together, different kinds AND):
  --inc[lude]=REGEX   keep links whose path matches (repeatable)
  --exc[lude]=REGEX   drop links whose path matches (repeatable)
  --name=GLOB         keep links whose basename matches glob (case-sensitive)
  --iname=GLOB        same, case-insensitive
  --max-depth=N       limit recursion depth (1 = direct children)

Edit (what to do with the target each matched link points to):
  --from=REGEX        pattern to match in the target (PCRE-level; (?i) etc.)
  --to=TEMPLATE       replacement; $1 / ${name} reference --from's captures
  -F, --fixed         treat --from as a literal string (replace all)
  -n, --dry-run       show what would change; write nothing

With no --from, matching links are just listed. Regex is PCRE-level: lookaround,
backreferences, and inline flags like (?i) are supported.

Other:
  -v, --verbose       also report unchanged matches
  -q, --quiet         only warnings and errors
      --version       print version and exit
      --examples      print usage examples and exit
  -h, --help          this help

Note: %s edits symlink targets in place by default. Use -n to preview first.
On Windows it also handles NTFS junctions and .lnk shortcuts.
`, appName, version, appName, appName, appName)
}

func printExamples() {
	fmt.Printf(`Examples:

  # Preview repointing every symlink under /srv that points into /mnt/old
  %s /srv --from='/mnt/old' --to='/mnt/new' -n

  # Do it for real
  %s /srv --from='/mnt/old' --to='/mnt/new'

  # Positional form (START FROM TO): same as above
  %s /srv '/mnt/old' '/mnt/new'

  # Only *.conf links, skip anything under a 'backup' dir
  %s . --iname='*.conf' --exc='/backup/' --from='v1' --to='v2'

  # Regex capture: rewrite /opt/app-1.2.3 -> /opt/app/1.2.3
  %s /srv --from='/opt/app-(\d+\.\d+\.\d+)' --to='/opt/app/$1'

  # Case-insensitive match via inline flag
  %s . --from='(?i)/OLD/' --to='/new/'

  # Literal replace (no regex), all occurrences
  %s . -F --from='C:\Old' --to='C:\New'

  # Just list every symlink two levels deep
  %s /srv --max-depth=2
`, appName, appName, appName, appName, appName, appName, appName, appName)
}
