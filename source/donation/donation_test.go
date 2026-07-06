//	Copyright © 2026 Jim Collier (ID: 1cv◂‡Vᛦ)
//	Licensed under the GNU General Public License v2.0 or later. Full text at:
//		https://spdx.org/licenses/GPL-2.0-or-later.html
//	SPDX-License-Identifier: GPL-2.0-or-later

package donation

import (
	"strings"
	"testing"
)

func TestConfigured(t *testing.T) {
	if (Target{Value: "PLACEHOLDER_BTC_ADDRESS"}).Configured() {
		t.Fatal("a placeholder must not count as configured")
	}
	if !(Target{Value: "bc1qrealaddress"}).Configured() {
		t.Fatal("a real value must count as configured")
	}
}

// CanonicalBytes must be one tab-separated line per target, in order, each ending
// in a newline - the format the sign helper and verify gate both rely on.
func TestCanonicalBytesFormat(t *testing.T) {
	got := string(CanonicalBytes())
	if !strings.HasSuffix(got, "\n") {
		t.Fatal("canonical bytes must end with a newline")
	}
	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	if len(lines) != len(Targets) {
		t.Fatalf("got %d lines, want %d", len(lines), len(Targets))
	}
	for i, line := range lines {
		parts := strings.Split(line, "\t")
		if len(parts) != 3 {
			t.Fatalf("line %d not label\\tkind\\tvalue: %q", i, line)
		}
		if parts[0] != Targets[i].Label || parts[1] != Targets[i].Kind.String() || parts[2] != Targets[i].Value {
			t.Fatalf("line %d = %q, does not match target %+v", i, line, Targets[i])
		}
	}
}
