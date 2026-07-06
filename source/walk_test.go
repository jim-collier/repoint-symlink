//	Copyright © 2026 Jim Collier (ID: 1cv◂‡Vᛦ)
//	Licensed under the GNU General Public License v2.0 or later. Full text at:
//		https://spdx.org/licenses/GPL-2.0-or-later.html
//	SPDX-License-Identifier: GPL-2.0-or-later

package main

import (
	"os"
	"path/filepath"
	"testing"
)

// buildTree lays down a small symlink tree and returns its root.
func buildTree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(os.MkdirAll(filepath.Join(root, "a", "b"), 0o755))
	must(os.Symlink("/mnt/old/one", filepath.Join(root, "a", "one")))         // depth 2
	must(os.Symlink("/mnt/old/two", filepath.Join(root, "a", "b", "two")))    // depth 3
	must(os.WriteFile(filepath.Join(root, "a", "plain"), []byte("x"), 0o644)) // not a link
	return root
}

func targetsByBase(entries []LinkEntry) map[string]string {
	m := map[string]string{}
	for _, e := range entries {
		m[filepath.Base(e.Path)] = e.Target
	}
	return m
}

func TestCollectAll(t *testing.T) {
	root := buildTree(t)
	entries, err := collectLinks(root, -1, false)
	if err != nil {
		t.Fatal(err)
	}
	got := targetsByBase(entries)
	if len(got) != 2 || got["one"] != "/mnt/old/one" || got["two"] != "/mnt/old/two" {
		t.Fatalf("collect all: %+v", got)
	}
	for _, e := range entries {
		if e.Kind != KindSymlink {
			t.Fatalf("expected symlink kind, got %v", e.Kind)
		}
	}
}

// On a single filesystem --no-cross-device must be a no-op: every link the plain
// walk finds is still found. (A real cross-device prune needs two mounts, which
// a unit test can't fabricate; the integration harness covers that shape.)
func TestNoCrossDeviceSingleFS(t *testing.T) {
	root := buildTree(t)
	if dev, ok := statDevice(root); !ok || dev == 0 {
		t.Skip("device id unavailable on this platform")
	}
	entries, err := collectLinks(root, -1, true)
	if err != nil {
		t.Fatal(err)
	}
	got := targetsByBase(entries)
	if len(got) != 2 || got["one"] == "" || got["two"] == "" {
		t.Fatalf("no-cross-device dropped links on one fs: %+v", got)
	}
}

func TestCollectMaxDepth(t *testing.T) {
	root := buildTree(t)
	entries, err := collectLinks(root, 2, false) // exclude the depth-3 link
	if err != nil {
		t.Fatal(err)
	}
	got := targetsByBase(entries)
	if _, ok := got["one"]; !ok {
		t.Fatal("depth-2 link should be present")
	}
	if _, ok := got["two"]; ok {
		t.Fatal("depth-3 link should be pruned by max-depth=2")
	}
}
