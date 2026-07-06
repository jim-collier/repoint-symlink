//	Copyright © 2026 Jim Collier (ID: 1cv◂‡Vᛦ)
//	Licensed under the GNU General Public License v2.0 or later. Full text at:
//		https://spdx.org/licenses/GPL-2.0-or-later.html
//	SPDX-License-Identifier: GPL-2.0-or-later

//go:build windows

package main

import "golang.org/x/sys/windows"

// statDevice returns the volume serial number backing path, the Windows analog
// of a POSIX device id. ok is false if it can't be read, so --no-cross-device
// degrades to a no-op. BACKUP_SEMANTICS lets us open a directory handle.
func statDevice(path string) (dev uint64, ok bool) {
	ptr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return 0, false
	}
	handle, err := windows.CreateFile(ptr, 0,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
		nil, windows.OPEN_EXISTING, windows.FILE_FLAG_BACKUP_SEMANTICS, 0)
	if err != nil {
		return 0, false
	}
	defer windows.CloseHandle(handle)
	var info windows.ByHandleFileInformation
	if err := windows.GetFileInformationByHandle(handle, &info); err != nil {
		return 0, false
	}
	return uint64(info.VolumeSerialNumber), true
}
