//	Copyright © 2026 Jim Collier (ID: 1cv◂‡Vᛦ)
//	Licensed under the GNU General Public License v2.0 or later. Full text at:
//		https://spdx.org/licenses/GPL-2.0-or-later.html
//	SPDX-License-Identifier: GPL-2.0-or-later

package main

import (
	"testing"

	"github.com/jim-collier/repoint-symlink/filter"
)

// specFor must translate each flag kind to the right op / glob / basename / fold.
func TestSpecForMapping(t *testing.T) {
	cases := []struct {
		kind selKind
		want filter.Spec
	}{
		{selInclude, filter.Spec{Op: filter.Narrow}},
		{selExclude, filter.Spec{Op: filter.Subtract}},
		{selReInclude, filter.Spec{Op: filter.Readd}},
		{selIncTarget, filter.Spec{Op: filter.Narrow}},
		{selExcTarget, filter.Spec{Op: filter.Subtract}},
		{selName, filter.Spec{Op: filter.Narrow, Glob: true, Base: true}},
		{selIName, filter.Spec{Op: filter.Narrow, Glob: true, Base: true, Fold: true}},
		{selWholename, filter.Spec{Op: filter.Narrow, Glob: true}},
		{selIWholename, filter.Spec{Op: filter.Narrow, Glob: true, Fold: true}},
	}
	for _, c := range cases {
		got := specFor(selRule{kind: c.kind, pat: "p"})
		c.want.Pattern = "p"
		c.want.Label = "--" + c.kind.flag()
		if got != c.want {
			t.Fatalf("%s -> %+v, want %+v", c.kind.flag(), got, c.want)
		}
	}
}

// compileFilters must produce a working pipeline: --iname folds, --exclude
// subtracts, and command-line order is preserved.
func TestCompileFiltersEndToEnd(t *testing.T) {
	path, target, err := compileFilters(&options{rules: []selRule{
		{selIName, "*.CONF"},
		{selExclude, "/backup/"},
	}})
	if err != nil {
		t.Fatalf("compileFilters: %v", err)
	}
	if !path.Selects("/srv/app.conf") {
		t.Fatal("iname should fold and match app.conf")
	}
	if path.Selects("/srv/backup/app.conf") {
		t.Fatal("exclude should drop the backup path")
	}
	if !target.Selects("/anything") {
		t.Fatal("an empty target pipeline should keep everything")
	}
}

// The --inc-target/--exc-target pipeline selects on the current target, not the
// link's own path.
func TestCompileTargetFilters(t *testing.T) {
	_, target, err := compileFilters(&options{targetRules: []selRule{
		{selIncTarget, "/mnt/old/"},
		{selExcTarget, "/mnt/old/skip/"},
	}})
	if err != nil {
		t.Fatalf("compileFilters: %v", err)
	}
	if !target.Selects("/mnt/old/data") {
		t.Fatal("inc-target should keep a target under /mnt/old/")
	}
	if target.Selects("/mnt/old/skip/data") {
		t.Fatal("exc-target should drop the skipped target")
	}
	if target.Selects("/elsewhere/data") {
		t.Fatal("inc-target should narrow out unrelated targets")
	}
}
