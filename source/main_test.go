//	Copyright © 2026 Jim Collier (ID: 1cv◂‡Vᛦ)
//	Licensed under the GNU General Public License v2.0 or later. Full text at:
//		https://spdx.org/licenses/GPL-2.0-or-later.html
//	SPDX-License-Identifier: GPL-2.0-or-later

package main

import (
	"strings"
	"testing"
)

// No args behaves like --help: exit 0, print the usage text, touch nothing.
func TestNoArgsShowsHelp(t *testing.T) {
	var code int
	out := captureStdout(t, func() { code = run(nil) })
	if code != 0 {
		t.Fatalf("run(nil) = %d, want 0", code)
	}
	if !strings.Contains(out, "Usage:") {
		t.Fatalf("run(nil) did not print help; got:\n%s", out)
	}
}
