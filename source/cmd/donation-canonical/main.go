//	Copyright © 2026 Jim Collier (ID: 1cv◂‡Vᛦ)
//	Licensed under the GNU General Public License v2.0 or later. Full text at:
//		https://spdx.org/licenses/GPL-2.0-or-later.html
//	SPDX-License-Identifier: GPL-2.0-or-later

// Command donation-canonical writes the donation table's canonical bytes to
// stdout. The sign helper feeds these to ssh-keygen so the signer and the verify
// gate sign/verify the exact same bytes. Not part of the shipped binary.
package main

import (
	"os"

	"github.com/jim-collier/repoint-symlink/donation"
)

func main() {
	if _, err := os.Stdout.Write(donation.CanonicalBytes()); err != nil {
		os.Exit(1)
	}
}
