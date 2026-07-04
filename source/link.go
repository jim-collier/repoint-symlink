//	Copyright © 2026 Jim Collier (ID: 1cv◂‡Vᛦ)
//	Licensed under the GNU General Public License v2.0 or later. Full text at:
//		https://spdx.org/licenses/GPL-2.0-or-later.html
//	SPDX-License-Identifier: GPL-2.0-or-later

package main

// LinkKind is what sort of link a matched entry is. Symlinks exist everywhere;
// junctions and shortcuts are Windows-only and only ever produced there.
type LinkKind int

const (
	KindSymlink LinkKind = iota
	KindJunction
	KindShortcut // Windows .lnk
)

func (k LinkKind) String() string {
	switch k {
	case KindJunction:
		return "junction"
	case KindShortcut:
		return "shortcut"
	default:
		return "symlink"
	}
}

// LinkEntry is one discovered link: the file that is the link, what kind it is,
// and where it currently points. IsDir marks a directory symlink (matters when
// recreating one on Windows, where file and dir symlinks are distinct).
type LinkEntry struct {
	Path   string
	Kind   LinkKind
	Target string
	IsDir  bool
}
