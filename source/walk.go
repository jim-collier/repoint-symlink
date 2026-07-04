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

// collectLinks walks root and returns every link it handles. It does not
// descend into symlinked directories (WalkDir never follows them), which also
// keeps it loop-safe. maxDepth < 0 means unlimited; depth 1 = direct children.
// classifyLink is platform-specific (see link_unix.go / link_windows.go).
func collectLinks(root string, maxDepth int) ([]LinkEntry, error) {
	var links []LinkEntry

	// If the start dir is itself a symlink to a directory, resolve it so we walk
	// its contents rather than stopping at the link.
	walkRoot := root
	if info, err := os.Lstat(root); err == nil && info.Mode()&fs.ModeSymlink != 0 {
		if resolved, err := filepath.EvalSymlinks(root); err == nil {
			walkRoot = resolved
		}
	}

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

func entryDepth(root, path string) int {
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == "." {
		return 0
	}
	return strings.Count(rel, string(filepath.Separator)) + 1
}
