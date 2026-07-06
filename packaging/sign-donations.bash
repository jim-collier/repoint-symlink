#!/bin/bash

##	Purpose:
##		- Signs the donation address table so cicd can detect tampering. The signature
##		  covers the donation package's CanonicalBytes() - the label, kind and value of
##		  every entry in source/donation/donation.go, in order - and the cicd verify gate
##		  (the Go test TestDonationTableSigned) re-checks it against the out-of-repo trust
##		  anchor.
##		- Run this after editing the addresses. Only the holder of the private key can
##		  produce a signature that verifies, so a pull request that changes an address
##		  without re-signing fails the gate.
##	Usage:
##		packaging/sign-donations.bash
##		- Signing key: $DONATION_SIGNING_KEY, else ../private/donation_keys/donation_ed25519
##		  (outside the repo, passphrase-protected; ssh-keygen prompts for the passphrase).
##		- Output: source/donation/donation.sig - commit it alongside donation.go.
##	History: At bottom of script.

##	Copyright © 2026 Jim Collier (ID: 1cv◂‡Vᛦ)
##	Licensed under The MIT License (MIT). Full text at:
##		https://mit-license.org/
##	SPDX-License-Identifier: MIT

set -euo pipefail

root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)   # the github/ repo dir
cd "$root"

key="${DONATION_SIGNING_KEY:-$root/../private/donation_keys/donation_ed25519}"
namespace="donation"
sig="$root/source/donation/donation.sig"

[[ -f "$key" ]] || {
	echo "Signing key not found: $key" >&2
	echo "Generate it first (see packaging/donation-signing.md), or point DONATION_SIGNING_KEY at it." >&2
	exit 1
}

## Canonical bytes come from the app itself, so signer and verifier never disagree.
canon=$(mktemp)
trap 'rm -f "$canon" "$canon.sig"' EXIT
( cd "$root/source" && go run ./cmd/donation-canonical ) > "$canon"

ssh-keygen -Y sign -f "$key" -n "$namespace" "$canon"
mv -f "$canon.sig" "$sig"

echo "Signed donation table -> $sig"


##	History:
##		- 2026-07-06 JC: Created (ported from the rapid-photo-downloader-pro sister project).
