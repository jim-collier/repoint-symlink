//	Copyright © 2026 Jim Collier (ID: 1cv◂‡Vᛦ)
//	Licensed under the GNU General Public License v2.0 or later. Full text at:
//		https://spdx.org/licenses/GPL-2.0-or-later.html
//	SPDX-License-Identifier: GPL-2.0-or-later

package main

import "testing"

func xform(t *testing.T, o *options, in string) string {
	t.Helper()
	r, err := buildRepointer(o)
	if err != nil {
		t.Fatalf("buildRepointer: %v", err)
	}
	out, err := r.transform(in)
	if err != nil {
		t.Fatalf("transform: %v", err)
	}
	return out
}

func TestRegexCapture(t *testing.T) {
	o := &options{fromSet: true, from: `/opt/app-(\d+\.\d+\.\d+)`, to: `/opt/app/$1`}
	if got := xform(t, o, "/opt/app-1.2.3"); got != "/opt/app/1.2.3" {
		t.Fatalf("capture: %q", got)
	}
}

func TestRegexReplaceAll(t *testing.T) {
	o := &options{fromSet: true, from: `/old`, to: `/new`}
	if got := xform(t, o, "/old/a/old"); got != "/new/a/new" {
		t.Fatalf("replace-all: %q", got)
	}
}

func TestCaseInsensitiveInline(t *testing.T) {
	o := &options{fromSet: true, from: `(?i)/OLD/`, to: `/new/`}
	if got := xform(t, o, "/Old/x"); got != "/new/x" {
		t.Fatalf("(?i): %q", got)
	}
}

func TestLiteralReplace(t *testing.T) {
	o := &options{fromSet: true, literal: true, from: `a.b.c`, to: `X`}
	if got := xform(t, o, "/data/a.b.c/a.b.c"); got != "/data/X/X" {
		t.Fatalf("literal replace: %q", got)
	}
	// dot must NOT act as regex wildcard in literal mode.
	if got := xform(t, o, "/data/axbxc"); got != "/data/axbxc" {
		t.Fatalf("literal should not match wildcard: %q", got)
	}
}

func TestListOnlyNoFrom(t *testing.T) {
	r, err := buildRepointer(&options{})
	if err != nil {
		t.Fatal(err)
	}
	if r.editMode {
		t.Fatal("no --from should be list-only")
	}
	if got, _ := r.transform("/anything"); got != "/anything" {
		t.Fatalf("list-only must not change target: %q", got)
	}
}

func TestBadFromRegex(t *testing.T) {
	if _, err := buildRepointer(&options{fromSet: true, from: `(`}); err == nil {
		t.Fatal("expected bad regex error")
	}
}
