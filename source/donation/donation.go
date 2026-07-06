//	Copyright © 2026 Jim Collier (ID: 1cv◂‡Vᛦ)
//	Licensed under the GNU General Public License v2.0 or later. Full text at:
//		https://spdx.org/licenses/GPL-2.0-or-later.html
//	SPDX-License-Identifier: GPL-2.0-or-later

// Package donation holds the project's own donation targets, shown by --donate.
//
// SECURITY: these are the project's real donation addresses. A bad edit here - or
// in a merged pull request - could silently redirect donations to someone else.
// Guards, weakest to strongest:
//   - Every change is a visible diff with git blame; keep this file under review.
//   - Values are PLACEHOLDERS until release; a placeholder shows as "not yet
//     configured" and is never printed, so a release can't ask for donations to
//     nothing.
//   - The table is signed with the maintainer's key (kept outside the repo) and
//     the cicd gate refuses to pass if the current table no longer matches that
//     signature - so a swapped address that wasn't re-signed can't reach a
//     release. See packaging/donation-signing.md. CanonicalBytes is what gets
//     signed and verified.
package donation

import "strings"

const placeholderPrefix = "PLACEHOLDER_"

// SignatureNamespace is the ssh-keygen signature namespace; it must match the
// sign helper (packaging/sign-donations.bash) and the verify gate.
const SignatureNamespace = "donation"

// Kind is what a target's value is: a crypto address to copy, or a URL to open.
type Kind int

const (
	Crypto Kind = iota
	Link
)

func (k Kind) String() string {
	if k == Link {
		return "link"
	}
	return "crypto"
}

// Target is one donation destination.
type Target struct {
	Label string
	Kind  Kind
	Value string
}

// Configured reports whether a real address/URL has replaced the placeholder.
func (t Target) Configured() bool { return !strings.HasPrefix(t.Value, placeholderPrefix) }

// Targets is the donation table. Replace every PLACEHOLDER_* with the real
// address/URL before a release, then re-sign (packaging/sign-donations.bash).
// Order is fixed - it is part of the signed content (see CanonicalBytes).
var Targets = []Target{
	{"Bitcoin (BTC)", Crypto, "PLACEHOLDER_BTC_ADDRESS"},
	{"Ethereum (ETH)", Crypto, "PLACEHOLDER_ETH_ADDRESS"},
	{"USD Coin (USDC)", Crypto, "PLACEHOLDER_USDC_ADDRESS"},
	{"Monero (XMR)", Crypto, "PLACEHOLDER_XMR_ADDRESS"},
	{"GitHub Sponsors", Link, "PLACEHOLDER_GITHUB_SPONSORS_URL"},
	{"Ko-fi", Link, "PLACEHOLDER_KOFI_URL"},
}

// HasConfigured reports whether at least one real address/URL has been set.
func HasConfigured() bool {
	for _, t := range Targets {
		if t.Configured() {
			return true
		}
	}
	return false
}

// CanonicalBytes is the exact byte sequence that gets signed and verified - the
// single source of truth for the sign helper and the cicd gate, so they can
// never disagree. One "label\tkind\tvalue" line per target in table order, each
// terminated by '\n'. Reordering or editing the table changes these bytes and
// invalidates the signature.
func CanonicalBytes() []byte {
	var b strings.Builder
	for _, t := range Targets {
		b.WriteString(t.Label)
		b.WriteByte('\t')
		b.WriteString(t.Kind.String())
		b.WriteByte('\t')
		b.WriteString(t.Value)
		b.WriteByte('\n')
	}
	return []byte(b.String())
}
