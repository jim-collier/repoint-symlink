//	Copyright © 2026 Jim Collier (ID: 1cv◂‡Vᛦ)
//	Licensed under the GNU General Public License v2.0 or later. Full text at:
//		https://spdx.org/licenses/GPL-2.0-or-later.html
//	SPDX-License-Identifier: GPL-2.0-or-later

package filter

import (
	"strings"
	"testing"
	"time"
)

// Rule-spec builders for the pipeline tests: reg is a regex rule; name/iname and
// whole/iwhole are basename and whole-path globs (the i-forms fold case).
func reg(op Op, pat string) Spec { return Spec{Op: op, Pattern: pat} }
func name(pat string) Spec       { return Spec{Op: Narrow, Pattern: pat, Glob: true, Base: true} }
func iname(pat string) Spec {
	return Spec{Op: Narrow, Pattern: pat, Glob: true, Base: true, Fold: true}
}
func whole(pat string) Spec  { return Spec{Op: Narrow, Pattern: pat, Glob: true} }
func iwhole(pat string) Spec { return Spec{Op: Narrow, Pattern: pat, Glob: true, Fold: true} }

func mustCompile(t *testing.T, specs ...Spec) *Set {
	t.Helper()
	set, err := Compile(specs)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	return set
}

// Consecutive narrows AND, they do not OR.
func TestConsecutiveNarrowsAND(t *testing.T) {
	set := mustCompile(t, reg(Narrow, `/a/`), reg(Narrow, `/b/`))
	if !set.Selects("/x/a/b/y") {
		t.Fatal("path matching both narrows should pass")
	}
	if set.Selects("/x/a/y") || set.Selects("/x/b/y") {
		t.Fatal("consecutive narrows must AND, not OR")
	}
}

func TestSubtractRejects(t *testing.T) {
	set := mustCompile(t, reg(Subtract, `/backup/`))
	if set.Selects("/srv/backup/l") {
		t.Fatal("subtract should reject a match")
	}
	if !set.Selects("/srv/live/l") {
		t.Fatal("a non-match should pass")
	}
}

// A narrow only ever narrows - even after a subtract it cannot re-admit what the
// subtract dropped.
func TestNarrowAfterSubtractStillNarrows(t *testing.T) {
	set := mustCompile(t,
		reg(Narrow, `/srv/`),
		reg(Subtract, `/srv/vendor/`),
		reg(Narrow, `/keep/`),
	)
	if set.Selects("/srv/vendor/keep/x") {
		t.Fatal("a narrow must not re-admit a subtracted path")
	}
	if set.Selects("/srv/live/x") {
		t.Fatal("third narrow drops a non-matching path")
	}
	if !set.Selects("/srv/keep/x") {
		t.Fatal("path surviving the subtract and matching every narrow should pass")
	}
}

// Readd re-admits from the original set what a subtract dropped.
func TestReaddAfterSubtractExpands(t *testing.T) {
	set := mustCompile(t,
		reg(Narrow, `/srv/`),
		reg(Subtract, `/srv/vendor/`),
		reg(Readd, `/srv/vendor/keep/`),
	)
	if set.Selects("/srv/vendor/x") {
		t.Fatal("a subtracted path not matched by the readd should stay dropped")
	}
	if !set.Selects("/srv/vendor/keep/x") {
		t.Fatal("readd should bring the subtracted path back")
	}
	if !set.Selects("/srv/live/x") {
		t.Fatal("a narrowed-in path should still pass")
	}
}

// Readd pulls from the whole original set, not the narrowed one - it can admit a
// path that never matched any prior narrow.
func TestReaddPullsFromOriginal(t *testing.T) {
	set := mustCompile(t, reg(Narrow, `/srv/`), reg(Readd, `/other/`))
	if !set.Selects("/other/x") {
		t.Fatal("readd should admit a path outside the narrow set")
	}
}

func TestNameGlobs(t *testing.T) {
	set := mustCompile(t, name("*.conf"))
	if !set.Selects("/x/a.conf") || set.Selects("/x/a.txt") {
		t.Fatal("name glob")
	}
	if mustCompile(t, name("*.CONF")).Selects("/x/a.conf") {
		t.Fatal("name glob is case-sensitive")
	}
	if !mustCompile(t, iname("*.CONF")).Selects("/x/a.conf") {
		t.Fatal("folded name glob is case-insensitive")
	}
}

// Basename globs match only the last element; whole-path globs span '/'.
func TestWholenameGlobs(t *testing.T) {
	if mustCompile(t, name("*/etc/*")).Selects("/srv/etc/a.conf") {
		t.Fatal("a basename glob matches the last element only, not the path")
	}
	w := mustCompile(t, whole("*/etc/*"))
	if !w.Selects("/srv/etc/a.conf") || !w.Selects("/a/b/etc/deep/c") {
		t.Fatal("a whole-path '*' should span '/' (find style)")
	}
	if w.Selects("/srv/other/a.conf") {
		t.Fatal("whole-path glob should not match a path without the segment")
	}
	if !mustCompile(t, iwhole("*/ETC/*")).Selects("/srv/etc/a.conf") {
		t.Fatal("folded whole-path glob is case-insensitive")
	}
}

// A narrow glob after a subtract narrows like any other keep-rule; it does not
// re-admit what the subtract dropped.
func TestGlobAfterSubtractNarrows(t *testing.T) {
	set := mustCompile(t, reg(Narrow, `/srv/`), reg(Subtract, `/srv/`), name("keep.conf"))
	if set.Selects("/srv/keep.conf") {
		t.Fatal("a narrow glob must not re-admit a subtracted path")
	}
}

func TestPCREFeatures(t *testing.T) {
	// lookahead is beyond RE2; regexp2 must accept it.
	set := mustCompile(t, reg(Narrow, `/srv/(?=.*keep)`))
	if !set.Selects("/srv/keep/x") || set.Selects("/srv/drop/x") {
		t.Fatal("lookahead not honored")
	}
}

func TestEmptySetKeepsAll(t *testing.T) {
	if !mustCompile(t).Selects("/anything/at/all") {
		t.Fatal("no rules should keep every path")
	}
}

func TestBadPatternReportsLabel(t *testing.T) {
	_, err := Compile([]Spec{{Op: Narrow, Pattern: `(`, Label: "--include"}})
	if err == nil || !strings.Contains(err.Error(), "--include") {
		t.Fatalf("compile error should name the rule label, got %v", err)
	}
}

// A catastrophic-backtracking pattern must fail within MatchTimeout rather than
// hanging, which proves the timeout is wired through CompileRegex.
func TestReDoSBounded(t *testing.T) {
	saved := MatchTimeout
	MatchTimeout = 150 * time.Millisecond
	defer func() { MatchTimeout = saved }()

	re, err := CompileRegex("(a+)+$", false)
	if err != nil {
		t.Fatalf("pattern should compile: %v", err)
	}
	input := strings.Repeat("a", 40) + "!"

	start := time.Now()
	ok, matchErr := re.MatchString(input)
	elapsed := time.Since(start)

	if elapsed > 2*time.Second {
		t.Fatalf("match was not bounded by the timeout: took %v", elapsed)
	}
	if ok {
		t.Fatal("the pathological input should not match")
	}
	if matchErr == nil {
		t.Fatal("a timed-out match should report an error")
	}
}
