//	Copyright © 2026 Jim Collier (ID: 1cv◂‡Vᛦ)
//	Licensed under the GNU General Public License v2.0 or later. Full text at:
//		https://spdx.org/licenses/GPL-2.0-or-later.html
//	SPDX-License-Identifier: GPL-2.0-or-later

package main

import (
	"io"
	"os"
	"testing"

	"github.com/jim-collier/repoint-symlink/filter"
)

// captureStdout runs fn with os.Stdout redirected and returns what it printed.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	fn()
	w.Close()
	os.Stdout = old
	out, _ := io.ReadAll(r)
	return string(out)
}

func keepAll(t *testing.T) *filter.Set {
	t.Helper()
	set, err := filter.Compile(nil)
	if err != nil {
		t.Fatal(err)
	}
	return set
}

// In list mode --print0 emits just the paths, NUL-terminated, no summary.
func TestPrint0ListMode(t *testing.T) {
	entries := []LinkEntry{
		{Path: "/a/one", Target: "/t/1", Kind: KindSymlink},
		{Path: "/a/two", Target: "/t/2", Kind: KindSymlink},
	}
	opts := &options{print0: true}
	all := keepAll(t)
	out := captureStdout(t, func() {
		process(opts, all, all, &repointer{}, entries)
	})
	if out != "/a/one\x00/a/two\x00" {
		t.Fatalf("print0 list output = %q", out)
	}
}

// In edit mode --print0 emits a record only for links that actually change.
func TestPrint0EditMode(t *testing.T) {
	entries := []LinkEntry{
		{Path: "/a/one", Target: "/mnt/old/1", Kind: KindSymlink},
		{Path: "/a/keep", Target: "/mnt/other/x", Kind: KindSymlink}, // no match -> unchanged
	}
	opts := &options{print0: true, dryRun: true} // dry-run so we don't touch disk
	rp, err := buildRepointer(&options{fromSet: true, from: "/mnt/old", to: "/mnt/new"})
	if err != nil {
		t.Fatal(err)
	}
	all := keepAll(t)
	out := captureStdout(t, func() {
		process(opts, all, all, rp, entries)
	})
	if out != "/a/one\x00" {
		t.Fatalf("print0 edit output = %q (only the changed link, no summary)", out)
	}
}
