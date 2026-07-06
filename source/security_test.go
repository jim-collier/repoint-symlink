//	Copyright © 2026 Jim Collier (ID: 1cv◂‡Vᛦ)
//	Licensed under the GNU General Public License v2.0 or later. Full text at:
//		https://spdx.org/licenses/GPL-2.0-or-later.html
//	SPDX-License-Identifier: GPL-2.0-or-later

//go:build !windows

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Repointing a link must replace the link itself, never write through it to the
// file it points at.
func TestWriteDoesNotFollowLink(t *testing.T) {
	root := t.TempDir()
	realFile := filepath.Join(root, "real.txt")
	if err := os.WriteFile(realFile, []byte("SECRET"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "link")
	if err := os.Symlink(realFile, link); err != nil {
		t.Fatal(err)
	}

	entry := LinkEntry{Path: link, Kind: KindSymlink, Target: realFile}
	if err := writeLinkTarget(entry, "/mnt/new/x"); err != nil {
		t.Fatalf("writeLinkTarget: %v", err)
	}

	if got, _ := os.Readlink(link); got != "/mnt/new/x" {
		t.Fatalf("link should now point at the new target, got %q", got)
	}
	if data, _ := os.ReadFile(realFile); string(data) != "SECRET" {
		t.Fatalf("the pointed-at file must be untouched, got %q", data)
	}
	info, err := os.Lstat(link)
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatal("link must still be a symlink, not a regular file")
	}
}

// A leftover .repoint.tmp beside the link would be a security and cleanliness
// problem; a successful rewrite must not leave one.
func TestTmpFileCleanedUp(t *testing.T) {
	root := t.TempDir()
	link := filepath.Join(root, "link")
	if err := os.Symlink("/mnt/old/x", link); err != nil {
		t.Fatal(err)
	}
	entry := LinkEntry{Path: link, Kind: KindSymlink, Target: "/mnt/old/x"}
	if err := writeLinkTarget(entry, "/mnt/new/x"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(link + ".repoint.tmp"); !os.IsNotExist(err) {
		t.Fatal("temporary file should have been renamed away")
	}
}

// The walk must not descend into a directory symlink, so a link reachable only
// through an aliased directory is never collected twice or from outside the tree.
func TestNoDescendIntoDirSymlink(t *testing.T) {
	root := t.TempDir()
	realDir := filepath.Join(root, "real")
	if err := os.MkdirAll(realDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/mnt/old/inner", filepath.Join(realDir, "inner")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(realDir, filepath.Join(root, "alias")); err != nil {
		t.Fatal(err)
	}

	entries, err := collectLinks(root, -1)
	if err != nil {
		t.Fatal(err)
	}
	inner := 0
	for _, e := range entries {
		if e.Target == "/mnt/old/inner" {
			inner++
		}
		if strings.Contains(e.Path, string(filepath.Separator)+"alias"+string(filepath.Separator)) {
			t.Fatalf("walk descended into the directory symlink: %s", e.Path)
		}
	}
	if inner != 1 {
		t.Fatalf("inner link should be collected exactly once, got %d", inner)
	}
}

// A symlink cycle (both a directory self-loop and a pair of file links pointing
// at each other) must not make the walk hang.
func TestSymlinkCycleTerminates(t *testing.T) {
	root := t.TempDir()
	if err := os.Symlink(root, filepath.Join(root, "self")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "b"), filepath.Join(root, "a")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "a"), filepath.Join(root, "b")); err != nil {
		t.Fatal(err)
	}

	done := make(chan error, 1)
	go func() { _, err := collectLinks(root, -1); done <- err }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("walk over a cyclic tree errored: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("walk did not terminate on a cyclic symlink tree")
	}
}
