//	Copyright © 2026 Jim Collier (ID: 1cv◂‡Vᛦ)
//	Licensed under the GNU General Public License v2.0 or later. Full text at:
//		https://spdx.org/licenses/GPL-2.0-or-later.html
//	SPDX-License-Identifier: GPL-2.0-or-later

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// oneLink lays down a real symlink and returns its path plus the matching entry.
func oneLink(t *testing.T, target string) (string, LinkEntry) {
	t.Helper()
	link := filepath.Join(t.TempDir(), "l")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	return link, LinkEntry{Path: link, Target: target, Kind: KindSymlink}
}

func runConfirm(t *testing.T, answer string, link string, entry LinkEntry) (out string, changed int) {
	t.Helper()
	rp, err := buildRepointer(&options{fromSet: true, from: "/mnt/old", to: "/mnt/new"})
	if err != nil {
		t.Fatal(err)
	}
	all := keepAll(t)
	saved := confirmReader
	confirmReader = strings.NewReader(answer)
	defer func() { confirmReader = saved }()
	out = captureStdout(t, func() {
		changed, _ = process(&options{confirm: true}, all, all, rp, []LinkEntry{entry})
	})
	return out, changed
}

// A "y" applies the whole plan after the preview.
func TestConfirmYesApplies(t *testing.T) {
	link, entry := oneLink(t, "/mnt/old/x")
	out, changed := runConfirm(t, "y\n", link, entry)
	if !strings.Contains(out, "would repoint") {
		t.Fatalf("confirm should preview the plan first, got %q", out)
	}
	if changed != 1 {
		t.Fatalf("expected 1 change on yes, got %d", changed)
	}
	if got, _ := os.Readlink(link); got != "/mnt/new/x" {
		t.Fatalf("link not rewritten after yes: %q", got)
	}
}

// Anything other than yes writes nothing.
func TestConfirmNoAborts(t *testing.T) {
	link, entry := oneLink(t, "/mnt/old/x")
	_, changed := runConfirm(t, "n\n", link, entry)
	if changed != 0 {
		t.Fatalf("expected 0 changes on no, got %d", changed)
	}
	if got, _ := os.Readlink(link); got != "/mnt/old/x" {
		t.Fatalf("link should be untouched after no: %q", got)
	}
}

// --confirm and --print0 are mutually exclusive.
func TestConfirmPrint0Conflict(t *testing.T) {
	if _, err := buildRepointer(&options{confirm: true, print0: true}); err == nil {
		t.Fatal("--confirm with --print0 should error")
	}
}
