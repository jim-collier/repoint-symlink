//	Copyright © 2026 Jim Collier (ID: 1cv◂‡Vᛦ)
//	Licensed under the GNU General Public License v2.0 or later. Full text at:
//		https://spdx.org/licenses/GPL-2.0-or-later.html
//	SPDX-License-Identifier: GPL-2.0-or-later

package main

import (
	"fmt"

	"github.com/jim-collier/repoint-symlink/donation"
)

const projectURL = "https://github.com/jim-collier/repoint-symlink"

// printDonate lists the project's donation targets (--donate). Placeholders are
// never printed, so a build without real addresses says so instead of showing a
// copyable but bogus value.
func printDonate() {
	fmt.Printf("Support %s\n\n", appName)
	if !donation.HasConfigured() {
		fmt.Println("Donation addresses are not yet configured in this build.")
		fmt.Printf("Project: %s\n", projectURL)
		return
	}
	fmt.Println("If you find this tool useful, please consider a donation:")
	fmt.Println()
	for _, t := range donation.Targets {
		if !t.Configured() {
			continue // never show a placeholder
		}
		fmt.Printf("  %-18s %s\n", t.Label, t.Value)
	}
	fmt.Printf("\nProject: %s\nThank you!\n", projectURL)
}
