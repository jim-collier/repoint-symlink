//	Copyright © 2026 Jim Collier (ID: 1cv◂‡Vᛦ)
//	Licensed under the GNU General Public License v2.0 or later. Full text at:
//		https://spdx.org/licenses/GPL-2.0-or-later.html
//	SPDX-License-Identifier: GPL-2.0-or-later

package filter

import (
	"testing"
	"time"

	"github.com/dlclark/regexp2"
)

// The fuzz targets below run their seed corpus during a normal 'go test', which
// already guards against regressions; 'go test -fuzz=...' drives them harder.
// The goal is robustness: no input, however malformed, should panic or hang.

// FuzzGlobToRegex feeds arbitrary globs through the find-style translator and
// the compiler. The translation must never panic, and a glob that does compile
// must be safe to match against a sample path.
func FuzzGlobToRegex(f *testing.F) {
	seeds := []string{"", "*", "?", "*.conf", "*/etc/*", "[a-z]*", "[!x]", "\\*", "a[", "]", "[]a]"}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, glob string) {
		re, err := compileGlob(glob, false)
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

// FuzzSelects drives arbitrary paths through a mixed pipeline (a narrow, a
// subtract, a readd, and a glob). Selection must never panic, whatever the path.
func FuzzSelects(f *testing.F) {
	seeds := []string{"", "/", "/srv/a.conf", "/a/b/c", "C:\\x\\y", "no-slashes"}
	for _, s := range seeds {
		f.Add(s)
	}
	set, err := Compile([]Spec{
		{Op: Narrow, Pattern: `/`},
		{Op: Subtract, Pattern: `backup`},
		{Op: Readd, Pattern: `keep`},
		{Op: Narrow, Pattern: `*`, Glob: true},
	})
	if err != nil {
		f.Fatalf("seed pipeline should compile: %v", err)
	}
	f.Fuzz(func(t *testing.T, path string) {
		_ = set.Selects(path) // must not panic
	})
}
