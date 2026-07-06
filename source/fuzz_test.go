//	Copyright © 2026 Jim Collier (ID: 1cv◂‡Vᛦ)
//	Licensed under the GNU General Public License v2.0 or later. Full text at:
//		https://spdx.org/licenses/GPL-2.0-or-later.html
//	SPDX-License-Identifier: GPL-2.0-or-later

package main

import (
	"strings"
	"testing"
	"time"

	"github.com/dlclark/regexp2"
)

// The fuzz targets below run their seed corpus during a normal 'go test', which
// already guards against regressions; 'go test -fuzz=...' drives them harder.
// The goal is robustness: no input, however malformed, should panic or hang.

// FuzzParseArgs throws arbitrary argument vectors at the parser. It must always
// return cleanly (options or error), never panic.
func FuzzParseArgs(f *testing.F) {
	seeds := []string{
		"", "-n", "-nvq", "--include=.*", "--exc=/x/ --re-inc=/y/",
		"--from=(a)(b) --to=$2$1", "-F --from=C:\\Old --to=C:\\New",
		"--max-depth=3", "-- --from", "--iwholename=*/etc/*", "--bogus",
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		argv := strings.Fields(raw)
		_, _ = parseArgs(argv) // must not panic
	})
}

// FuzzGlobToRegex feeds arbitrary globs through the find-style translator and
// the compiler. The translation must never panic, and a glob that does compile
// must be safe to match against a sample path.
func FuzzGlobToRegex(f *testing.F) {
	seeds := []string{"", "*", "?", "*.conf", "*/etc/*", "[a-z]*", "[!x]", "\\*", "a[", "]", "[]a]"}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, glob string) {
		re, err := compileGlob(globToRegex(glob), false)
		if err != nil {
			return // an uncompilable translation is acceptable; a panic is not
		}
		re.MatchTimeout = 200 * time.Millisecond
		_, _ = re.MatchString("/some/sample/path.conf")
	})
}

// FuzzCompileAndMatch exercises the regexp2 library itself with arbitrary
// patterns. Invalid patterns return a compile error; valid ones match against a
// sample under a short timeout so catastrophic backtracking cannot hang the run.
func FuzzCompileAndMatch(f *testing.F) {
	seeds := []string{".*", "(a+)+$", "(?i)old", "a|b|c", "\\d+\\.\\d+", "(?=.*x)", "[", "(unclosed"}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, pattern string) {
		re, err := regexp2.Compile(pattern, regexp2.None)
		if err != nil {
			return
		}
		re.MatchTimeout = 200 * time.Millisecond
		_, _ = re.MatchString("aaaaaaaaaaaaaaaaaaaaaaaa!/mnt/old/data")
	})
}

// FuzzTransform runs arbitrary targets through both replace modes. Literal mode
// must never panic; regex mode is bounded by the match timeout.
func FuzzTransform(f *testing.F) {
	seeds := []string{"/mnt/old/x", "", "C:\\Old\\path", "no-match-here", "/a/b/c/d"}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, target string) {
		lit := &repointer{editMode: true, literal: true, fromLit: "old", to: "new"}
		_, _ = lit.transform(target)

		re, err := compileRE("(.*)/(.*)", regexp2.None)
		if err != nil {
			t.Fatalf("seed pattern should compile: %v", err)
		}
		re.MatchTimeout = 200 * time.Millisecond
		rx := &repointer{editMode: true, fromRE: re, to: "$2/$1"}
		_, _ = rx.transform(target)
	})
}
