//	Copyright © 2026 Jim Collier (ID: 1cv◂‡Vᛦ)
//	Licensed under the GNU General Public License v2.0 or later. Full text at:
//		https://spdx.org/licenses/GPL-2.0-or-later.html
//	SPDX-License-Identifier: GPL-2.0-or-later

package main

import "testing"

func compile(t *testing.T, o *options) *filters {
	t.Helper()
	f, err := compileFilters(o)
	if err != nil {
		t.Fatalf("compileFilters: %v", err)
	}
	return f
}

func TestIncludeOr(t *testing.T) {
	f := compile(t, &options{includes: []string{`/a/`, `/b/`}})
	if !f.selects("/x/a/y") || !f.selects("/x/b/y") {
		t.Fatal("include should OR")
	}
	if f.selects("/x/c/y") {
		t.Fatal("non-matching path should be dropped")
	}
}

func TestExcludeRejects(t *testing.T) {
	f := compile(t, &options{excludes: []string{`/backup/`}})
	if f.selects("/srv/backup/l") {
		t.Fatal("exclude should reject")
	}
	if !f.selects("/srv/live/l") {
		t.Fatal("non-excluded should pass")
	}
}

func TestNameGlobs(t *testing.T) {
	f := compile(t, &options{names: []string{"*.conf"}})
	if !f.selects("/x/a.conf") || f.selects("/x/a.txt") {
		t.Fatal("name glob")
	}
	// case sensitivity
	fs := compile(t, &options{names: []string{"*.CONF"}})
	if fs.selects("/x/a.conf") {
		t.Fatal("--name is case-sensitive")
	}
	fi := compile(t, &options{inames: []string{"*.CONF"}})
	if !fi.selects("/x/a.conf") {
		t.Fatal("--iname is case-insensitive")
	}
}

func TestGatesAnd(t *testing.T) {
	// path gate AND name gate: both must hold.
	f := compile(t, &options{includes: []string{`/srv/`}, names: []string{"*.conf"}})
	if !f.selects("/srv/a.conf") {
		t.Fatal("both gates satisfied should pass")
	}
	if f.selects("/srv/a.txt") {
		t.Fatal("name gate fails -> drop")
	}
	if f.selects("/other/a.conf") {
		t.Fatal("path gate fails -> drop")
	}
}

func TestPCREFeatures(t *testing.T) {
	// lookahead is beyond RE2; regexp2 must accept it.
	f := compile(t, &options{includes: []string{`/srv/(?=.*keep)`}})
	if !f.selects("/srv/keep/x") || f.selects("/srv/drop/x") {
		t.Fatal("lookahead not honored")
	}
}
