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
	opts, err := parseArgs(argv)
	if err != nil {
		errorf("%v", err)
		errorf("try --help")
		return 2
	}

	switch {
	case opts.showVersion:
		fmt.Println(version)
		return 0
	case opts.showHelp:
		printHelp()
		return 0
	case opts.showExamples:
		printExamples()
		return 0
	}

	filt, targetFilt, err := compileFilters(opts)
	if err != nil {
		errorf("%v", err)
		return 2
	}
	rp, err := buildRepointer(opts)
	if err != nil {
		errorf("%v", err)
		return 2
	}

	// Start dir must resolve to a directory.
	info, err := os.Stat(opts.dir)
	if err != nil {
		errorf("cannot read start folder %q: %v", opts.dir, err)
		return 1
	}
	if !info.IsDir() {
		errorf("start folder %q is not a directory", opts.dir)
		return 1
	}

	entries, err := collectLinks(opts.dir, opts.maxDepth, opts.noCrossDev, opts.followLinks)
	if err != nil {
		errorf("walk failed: %v", err)
		return 1
	}

	_, failed := process(opts, filt, targetFilt, rp, entries)
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

Filters (select which links to act on, matched against the link's own path).
Every filter is one rule in an ordered pipeline - their order matters:
  --inc[lude]=REGEX     keep only links whose path also matches (narrows)
  --exc[lude]=REGEX     drop links whose path matches (subtracts)
  --re-inc[lude]=REGEX  re-add links matching this from the original scan (widens)
  --inc-target=REGEX    keep only links whose current target matches (narrows)
  --exc-target=REGEX    drop links whose current target matches (subtracts)
  --name=GLOB           keep only links whose basename matches glob (case-sensitive)
  --iname=GLOB          same, case-insensitive
  --wholename=GLOB      keep only links whose whole path matches glob (case-sensitive)
  --iwholename=GLOB     same, case-insensitive
  --max-depth=N         limit recursion depth (1 = direct children)
  --no-cross-device     don't descend into dirs on another filesystem
  -L, --follow-links    descend into directory symlinks (loop-safe)

Each flag has one fixed effect, so you can reason left to right one at a time.
--include and the name/wholename globs narrow (keep only what also matches);
--exclude subtracts. Both only ever shrink the set. --re-include is the only
widener: it re-admits any link from the original scan matching its regex, even
one a prior --exclude dropped. Globs are find-style ('*' spans '/'); quote them.
--inc-target/--exc-target apply the same narrow/subtract logic to each link's
current target (where it points) instead of its own path.

Edit (what to do with the target each matched link points to):
  --from=REGEX        pattern to match in the target (PCRE-level; (?i) etc.)
  --to=TEMPLATE       replacement; $1 / ${name} reference --from's captures
  -F, --literal       treat --from as a plain literal, not a regex (replace all)
  --renormal-relative rewrite each target relative to the link's own directory
  --renormal-absolute rewrite each target as a cleaned absolute path
  -n, --dry-run       show what would change; write nothing
      --confirm       preview the whole plan, then prompt once before writing
  -0, --print0        machine output: one link path per NUL, no summary

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

  # Whole-path glob (find style): links anywhere under an 'etc' dir
  %s / --wholename='*/etc/*'

  # Order matters: drop every .tmp, then rescue the ones under assets/
  %s /srv --exc='\.tmp$' --re-inc='/assets/.*\.tmp$'

  # Regex capture: rewrite /opt/app-1.2.3 -> /opt/app/1.2.3
  %s /srv --from='/opt/app-(\d+\.\d+\.\d+)' --to='/opt/app/$1'

  # Case-insensitive match via inline flag
  %s . --from='(?i)/OLD/' --to='/new/'

  # Literal replace (no regex), all occurrences
  %s . -F --from='C:\Old' --to='C:\New'

  # Just list every symlink two levels deep
  %s /srv --max-depth=2
`, appName, appName, appName, appName, appName, appName, appName, appName, appName, appName)
}
