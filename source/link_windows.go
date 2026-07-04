//	Copyright © 2026 Jim Collier (ID: 1cv◂‡Vᛦ)
//	Licensed under the GNU General Public License v2.0 or later. Full text at:
//		https://spdx.org/licenses/GPL-2.0-or-later.html
//	SPDX-License-Identifier: GPL-2.0-or-later

//go:build windows

// Windows link handling: symlinks, NTFS junctions (mount-point reparse points),
// and .lnk shortcuts. Untested on real Windows hardware from this build host -
// see the "verify on Windows" backlog item.

package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"unicode/utf16"

	ole "github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
	"golang.org/x/sys/windows"
)

const (
	fsctlGetReparsePoint = 0x000900A8
	fsctlSetReparsePoint = 0x000900A4
	maxReparse           = 16 * 1024

	// Not exported by x/sys/windows in all versions.
	symlinkFlagUnprivileged = 0x2
)

func classifyLink(path string, d fs.DirEntry) (*LinkEntry, bool, error) {
	// .lnk shortcuts are ordinary files.
	if !d.IsDir() && strings.EqualFold(filepath.Ext(path), ".lnk") {
		target, err := readShortcut(path)
		if err != nil {
			return nil, false, fmt.Errorf("read shortcut: %w", err)
		}
		return &LinkEntry{Path: path, Kind: KindShortcut, Target: target}, true, nil
	}

	// Everything else we handle is a reparse point (symlink or junction).
	if !isReparsePoint(path) {
		return nil, false, nil
	}
	tag, err := reparseTag(path)
	if err != nil {
		return nil, false, nil // unreadable/other reparse type - skip quietly
	}
	target, err := os.Readlink(path) // handles both symlink and mount-point targets
	if err != nil {
		return nil, false, fmt.Errorf("read reparse target: %w", err)
	}
	switch tag {
	case windows.IO_REPARSE_TAG_MOUNT_POINT:
		return &LinkEntry{Path: path, Kind: KindJunction, Target: target, IsDir: true}, true, nil
	case windows.IO_REPARSE_TAG_SYMLINK:
		isDir := false
		if info, err := os.Stat(path); err == nil {
			isDir = info.IsDir()
		}
		return &LinkEntry{Path: path, Kind: KindSymlink, Target: target, IsDir: isDir}, true, nil
	default:
		return nil, false, nil // some other reparse type (dedup, etc.)
	}
}

func writeLinkTarget(e LinkEntry, newTarget string) error {
	switch e.Kind {
	case KindShortcut:
		return writeShortcut(e.Path, newTarget)
	case KindJunction:
		return writeJunction(e.Path, newTarget)
	default:
		return writeSymlink(e, newTarget)
	}
}

func isReparsePoint(path string) bool {
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return false
	}
	attrs, err := windows.GetFileAttributes(p)
	if err != nil {
		return false
	}
	return attrs&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0
}

func reparseTag(path string) (uint32, error) {
	h, err := openReparse(path, 0)
	if err != nil {
		return 0, err
	}
	defer windows.CloseHandle(h)
	buf := make([]byte, maxReparse)
	var ret uint32
	if err := windows.DeviceIoControl(h, fsctlGetReparsePoint, nil, 0, &buf[0], uint32(len(buf)), &ret, nil); err != nil {
		return 0, err
	}
	if ret < 4 {
		return 0, fmt.Errorf("short reparse data")
	}
	return binary.LittleEndian.Uint32(buf[:4]), nil
}

func openReparse(path string, access uint32) (windows.Handle, error) {
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return windows.InvalidHandle, err
	}
	return windows.CreateFile(p, access,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
		nil, windows.OPEN_EXISTING,
		windows.FILE_FLAG_BACKUP_SEMANTICS|windows.FILE_FLAG_OPEN_REPARSE_POINT, 0)
}

func writeSymlink(e LinkEntry, newTarget string) error {
	if err := os.Remove(e.Path); err != nil {
		return err
	}
	tgt, err := windows.UTF16PtrFromString(newTarget)
	if err != nil {
		return err
	}
	lnk, err := windows.UTF16PtrFromString(e.Path)
	if err != nil {
		return err
	}
	var flags uint32 = symlinkFlagUnprivileged
	if e.IsDir {
		flags |= windows.SYMBOLIC_LINK_FLAG_DIRECTORY
	}
	return windows.CreateSymbolicLink(lnk, tgt, flags)
}

// writeJunction overwrites an existing junction's mount-point reparse buffer.
// The directory is reused in place, so the junction is never missing.
func writeJunction(path, target string) error {
	target = filepath.Clean(target)
	if !filepath.IsAbs(target) {
		return fmt.Errorf("junction target must be absolute: %q", target)
	}
	subst := `\??\` + target
	sw := utf16.Encode([]rune(subst))
	pw := utf16.Encode([]rune(target))

	var pathBuf bytes.Buffer
	writeU16(&pathBuf, sw)
	pathBuf.Write([]byte{0, 0}) // null terminator
	writeU16(&pathBuf, pw)
	pathBuf.Write([]byte{0, 0})
	pb := pathBuf.Bytes()

	var b bytes.Buffer
	wu32 := func(v uint32) { binary.Write(&b, binary.LittleEndian, v) }
	wu16 := func(v uint16) { binary.Write(&b, binary.LittleEndian, v) }
	wu32(windows.IO_REPARSE_TAG_MOUNT_POINT)
	wu16(uint16(8 + len(pb)))       // ReparseDataLength: the four USHORTs + PathBuffer
	wu16(0)                         // Reserved
	wu16(0)                         // SubstituteNameOffset
	wu16(uint16(len(sw) * 2))       // SubstituteNameLength (bytes, no null)
	wu16(uint16((len(sw) + 1) * 2)) // PrintNameOffset
	wu16(uint16(len(pw) * 2))       // PrintNameLength
	b.Write(pb)
	data := b.Bytes()

	h, err := openReparse(path, windows.GENERIC_WRITE)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(h)
	var ret uint32
	return windows.DeviceIoControl(h, fsctlSetReparsePoint, &data[0], uint32(len(data)), nil, 0, &ret, nil)
}

func writeU16(w *bytes.Buffer, vs []uint16) {
	for _, v := range vs {
		w.Write([]byte{byte(v), byte(v >> 8)})
	}
}

// readShortcut / writeShortcut drive a .lnk via WScript.Shell (IDispatch), whose
// TargetPath property is exactly the "where it points" we edit.
func readShortcut(path string) (string, error) {
	var result string
	err := withCOM(func(wsh *ole.IDispatch) error {
		lnkV, err := oleutil.CallMethod(wsh, "CreateShortcut", path)
		if err != nil {
			return err
		}
		lnk := lnkV.ToIDispatch()
		defer lnk.Release()
		tp, err := oleutil.GetProperty(lnk, "TargetPath")
		if err != nil {
			return err
		}
		result = tp.ToString()
		return nil
	})
	return result, err
}

func writeShortcut(path, target string) error {
	return withCOM(func(wsh *ole.IDispatch) error {
		lnkV, err := oleutil.CallMethod(wsh, "CreateShortcut", path)
		if err != nil {
			return err
		}
		lnk := lnkV.ToIDispatch()
		defer lnk.Release()
		if _, err := oleutil.PutProperty(lnk, "TargetPath", target); err != nil {
			return err
		}
		_, err = oleutil.CallMethod(lnk, "Save")
		return err
	})
}

func withCOM(fn func(wsh *ole.IDispatch) error) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	// S_FALSE (already initialized on this thread) is not a real error.
	_ = ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED)
	defer ole.CoUninitialize()

	unknown, err := oleutil.CreateObject("WScript.Shell")
	if err != nil {
		return err
	}
	defer unknown.Release()
	wsh, err := unknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return err
	}
	defer wsh.Release()
	return fn(wsh)
}
