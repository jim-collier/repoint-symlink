//	Copyright © 2026 Jim Collier (ID: 1cv◂‡Vᛦ)
//	Licensed under the GNU General Public License v2.0 or later. Full text at:
//		https://spdx.org/licenses/GPL-2.0-or-later.html
//	SPDX-License-Identifier: GPL-2.0-or-later

package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// collectLinks walks root and returns every link it handles. maxDepth < 0 means
// unlimited; depth 1 = direct children. noCrossDev prunes any directory on a
// different underlying device (find -xdev). followLinks descends into directory
// symlinks (loop-safe via a visited-canonical-path set); without it the walk
// never follows a symlink, which is inherently loop-safe.
// classifyLink is platform-specific (see link_unix.go / link_windows.go).
func collectLinks(root string, maxDepth int, noCrossDev, followLinks bool) ([]LinkEntry, error) {
	// If the start dir is itself a symlink to a directory, resolve it so we walk
	// its contents rather than stopping at the link.
	walkRoot := root
	if info, err := os.Lstat(root); err == nil && info.Mode()&fs.ModeSymlink != 0 {
		if resolved, err := filepath.EvalSymlinks(root); err == nil {
			walkRoot = resolved
		}
	}

	var rootDev uint64
	haveRootDev := false
	if noCrossDev {
		if rootDev, haveRootDev = statDevice(walkRoot); !haveRootDev {
			warnf("--no-cross-device: cannot determine device of %s; not pruning", walkRoot)
		}
	}
	// sameDevice is true unless the path is on a known, different device (an
	// unreadable device never prunes).
	sameDevice := func(path string) bool {
		if !haveRootDev {
			return true
		}
		dev, ok := statDevice(path)
		return !ok || dev == rootDev
	}

	if followLinks {
		return collectFollow(walkRoot, maxDepth, sameDevice), nil
	}

	var links []LinkEntry
	err := filepath.WalkDir(walkRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			warnf("cannot access %s: %v", path, err)
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if maxDepth >= 0 && entryDepth(walkRoot, path) > maxDepth {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		// Don't descend into a directory that sits on a different device.
		if d.IsDir() && path != walkRoot && !sameDevice(path) {
			return fs.SkipDir
		}
		entry, ok, classifyErr := classifyLink(path, d)
		if classifyErr != nil {
			warnf("%s: %v", path, classifyErr)
			return nil
		}
		if ok {
			links = append(links, *entry)
		}
		return nil
	})
	return links, err
}

// collectFollow is the --follow-links walk: a manual recursion that descends
// into directory symlinks. visited holds the canonical path of every directory
// already entered, so a cycle (or a subtree reachable two ways) is walked once.
func collectFollow(root string, maxDepth int, sameDevice func(string) bool) []LinkEntry {
	var links []LinkEntry
	visited := map[string]bool{}
	if canon, err := filepath.EvalSymlinks(root); err == nil {
		visited[canon] = true
	} else {
		visited[root] = true
	}

	var walk func(dir string, depth int)
	descend := func(path string, depth int) {
		canon, err := filepath.EvalSymlinks(path)
		if err != nil {
			return // dangling or a self-referential link (ELOOP) - just don't enter
		}
		if visited[canon] || !sameDevice(canon) {
			return
		}
		visited[canon] = true
		walk(path, depth)
	}
	walk = func(dir string, depth int) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			warnf("cannot access %s: %v", dir, err)
			return
		}
		for _, de := range entries {
			path := filepath.Join(dir, de.Name())
			childDepth := depth + 1
			if maxDepth >= 0 && childDepth > maxDepth {
				continue // beyond the depth limit: neither collect nor descend
			}
			if entry, ok, classifyErr := classifyLink(path, de); classifyErr != nil {
				warnf("%s: %v", path, classifyErr)
			} else if ok {
				links = append(links, *entry)
			}
			isSymlink := de.Type()&fs.ModeSymlink != 0
			switch {
			case de.IsDir() && !isSymlink: // real directory
				descend(path, childDepth)
			case isSymlink && isDirLink(path): // symlink to a directory
				descend(path, childDepth)
			}
		}
	}
	walk(root, 0)
	return links
}

// isDirLink reports whether path (a symlink) resolves to a directory.
func isDirLink(path string) bool {
	info, err := os.Stat(path) // follows the link
	return err == nil && info.IsDir()
}

func entryDepth(root, path string) int {
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == "." {
		return 0
	}
	return strings.Count(rel, string(filepath.Separator)) + 1
}
