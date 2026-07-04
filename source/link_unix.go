//	Copyright © 2026 Jim Collier (ID: 1cv◂‡Vᛦ)
//	Licensed under the GNU General Public License v2.0 or later. Full text at:
//		https://spdx.org/licenses/GPL-2.0-or-later.html
//	SPDX-License-Identifier: GPL-2.0-or-later

//go:build !windows

package main

import (
	"io/fs"
	"os"
)

// classifyLink reports whether path is a symlink and, if so, reads its target.
// Junctions and .lnk shortcuts don't exist here, so symlinks are all we handle.
func classifyLink(path string, d fs.DirEntry) (*LinkEntry, bool, error) {
	if d.Type()&fs.ModeSymlink == 0 {
		return nil, false, nil
	}
	target, err := os.Readlink(path)
	if err != nil {
		return nil, false, err
	}
	isDir := false
	if info, err := os.Stat(path); err == nil { // follows the link
		isDir = info.IsDir()
	}
	return &LinkEntry{Path: path, Kind: KindSymlink, Target: target, IsDir: isDir}, true, nil
}

// writeLinkTarget repoints a symlink by creating a replacement beside it and
// renaming over the original - atomic on POSIX, so the link is never missing.
func writeLinkTarget(entry LinkEntry, newTarget string) error {
	tmp := entry.Path + ".repoint.tmp"
	_ = os.Remove(tmp)
	if err := os.Symlink(newTarget, tmp); err != nil {
		return err
	}
	if err := os.Rename(tmp, entry.Path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}
