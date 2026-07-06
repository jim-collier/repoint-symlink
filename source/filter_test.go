//	Copyright © 2026 Jim Collier (ID: 1cv◂‡Vᛦ)
//	Licensed under the GNU General Public License v2.0 or later. Full text at:
//		https://spdx.org/licenses/GPL-2.0-or-later.html
//	SPDX-License-Identifier: GPL-2.0-or-later

package main

import "testing"

// rules builds an options with selection rules in the given order.
func rules(rs ...selRule) *options { return &options{rules: rs} }

func compile(t *testing.T, o *options) *filters {
	t.Helper()
	f, err := compileFilters(o)
	if err != nil {
		t.Fatalf("compileFilters: %v", err)
	}
	return f
}

// Consecutive includes narrow (AND), they do not OR.
func TestConsecutiveIncludesNarrow(t *testing.T) {
	f := compile(t, rules(selRule{selInclude, `/a/`}, selRule{selInclude, `/b/`}))
	if !f.selects("/x/a/b/y") {
		t.Fatal("path matching both includes should pass")
	}
	if f.selects("/x/a/y") || f.selects("/x/b/y") {
		t.Fatal("consecutive includes must AND, not OR")
	}
}

func TestExcludeNarrows(t *testing.T) {
	f := compile(t, rules(selRule{selExclude, `/backup/`}))
	if f.selects("/srv/backup/l") {
		t.Fatal("exclude should reject")
	}
	if !f.selects("/srv/live/l") {
		t.Fatal("non-excluded should pass")
	}
}

// A plain include only ever narrows - even after an exclude it cannot re-admit
// what the exclude dropped.
func TestIncludeAfterExcludeNarrows(t *testing.T) {
	f := compile(t, rules(
		selRule{selInclude, `/srv/`},
		selRule{selExclude, `/srv/vendor/`},
		selRule{selInclude, `/keep/`},
	))
	if f.selects("/srv/vendor/keep/x") {
		t.Fatal("plain include must not re-admit an excluded path")
	}
	if f.selects("/srv/live/x") {
		t.Fatal("third include narrows: non-matching path drops out")
	}
	if !f.selects("/srv/keep/x") {
		t.Fatal("path surviving exclude and matching every include should pass")
	}
}

// --re-include re-admits from the original scan what an exclude dropped.
func TestReincludeAfterExcludeExpands(t *testing.T) {
	f := compile(t, rules(
		selRule{selInclude, `/srv/`},
		selRule{selExclude, `/srv/vendor/`},
		selRule{selReInclude, `/srv/vendor/keep/`},
	))
	if f.selects("/srv/vendor/x") {
		t.Fatal("excluded path not matched by --re-include should stay dropped")
	}
	if !f.selects("/srv/vendor/keep/x") {
		t.Fatal("--re-include should bring the excluded path back")
	}
	if !f.selects("/srv/live/x") {
		t.Fatal("plainly included path should pass")
	}
}

// --re-include pulls from the whole original scan, not the narrowed set - it can
// admit a link that never matched any prior include.
func TestReincludePullsFromOriginal(t *testing.T) {
	f := compile(t, rules(
		selRule{selInclude, `/srv/`},
		selRule{selReInclude, `/other/`},
	))
	if !f.selects("/other/x") {
		t.Fatal("--re-include should admit a path outside the include set")
	}
}

func TestNameGlobs(t *testing.T) {
	f := compile(t, rules(selRule{selName, "*.conf"}))
	if !f.selects("/x/a.conf") || f.selects("/x/a.txt") {
		t.Fatal("name glob")
	}
	fs := compile(t, rules(selRule{selName, "*.CONF"}))
	if fs.selects("/x/a.conf") {
		t.Fatal("--name is case-sensitive")
	}
	fi := compile(t, rules(selRule{selIName, "*.CONF"}))
	if !fi.selects("/x/a.conf") {
		t.Fatal("--iname is case-insensitive")
	}
}

// Basename globs match only the last path element; whole-path globs span '/'.
func TestWholenameGlobs(t *testing.T) {
	byName := compile(t, rules(selRule{selName, "*/etc/*"}))
	if byName.selects("/srv/etc/a.conf") {
		t.Fatal("--name matches the basename only, not the path")
	}
	whole := compile(t, rules(selRule{selWholename, "*/etc/*"}))
	if !whole.selects("/srv/etc/a.conf") || !whole.selects("/a/b/etc/deep/c") {
		t.Fatal("--wholename '*' should span '/' (find style)")
	}
	if whole.selects("/srv/other/a.conf") {
		t.Fatal("--wholename should not match a path without the segment")
	}
	ci := compile(t, rules(selRule{selIWholename, "*/ETC/*"}))
	if !ci.selects("/srv/etc/a.conf") {
		t.Fatal("--iwholename is case-insensitive")
	}
}

// A name glob after an exclude narrows like any other keep-rule; it does not
// re-admit what the exclude dropped.
func TestGlobAfterExcludeNarrows(t *testing.T) {
	f := compile(t, rules(
		selRule{selInclude, `/srv/`},
		selRule{selExclude, `/srv/`},
		selRule{selName, "keep.conf"},
	))
	if f.selects("/srv/keep.conf") {
		t.Fatal("name glob must not re-admit an excluded path")
	}
}

func TestPCREFeatures(t *testing.T) {
	// lookahead is beyond RE2; regexp2 must accept it.
	f := compile(t, rules(selRule{selInclude, `/srv/(?=.*keep)`}))
	if !f.selects("/srv/keep/x") || f.selects("/srv/drop/x") {
		t.Fatal("lookahead not honored")
	}
}

func TestNoRulesKeepsAll(t *testing.T) {
	f := compile(t, rules())
	if !f.selects("/anything/at/all") {
		t.Fatal("no filters should keep every link")
	}
}
