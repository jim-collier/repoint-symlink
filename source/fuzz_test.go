//	Copyright © 2026 Jim Collier (ID: 1cv◂‡Vᛦ)
//	Licensed under the GNU General Public License v2.0 or later. Full text at:
//		https://spdx.org/licenses/GPL-2.0-or-later.html
//	SPDX-License-Identifier: GPL-2.0-or-later

package main

import (
	"strings"
	"testing"
	"time"

	"github.com/jim-collier/repoint-symlink/filter"
)

// The fuzz targets below run their seed corpus during a normal 'go test', which
// already guards against regressions; 'go test -fuzz=...' drives them harder.
// The goal is robustness: no input, however malformed, should panic or hang.
// The glob translator and regex engine are fuzzed in package filter.

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

		re, err := filter.CompileRegex("(.*)/(.*)", false)
		if err != nil {
			t.Fatalf("seed pattern should compile: %v", err)
		}
		re.MatchTimeout = 200 * time.Millisecond
		rx := &repointer{editMode: true, fromRE: re, to: "$2/$1"}
		_, _ = rx.transform(target)
	})
}
