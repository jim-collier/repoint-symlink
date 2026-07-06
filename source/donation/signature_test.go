//	Copyright © 2026 Jim Collier (ID: 1cv◂‡Vᛦ)
//	Licensed under the GNU General Public License v2.0 or later. Full text at:
//		https://spdx.org/licenses/GPL-2.0-or-later.html
//	SPDX-License-Identifier: GPL-2.0-or-later

// Gate: once real donation addresses are set, the table must carry a valid
// signature. A pull request can edit an address, but only the maintainer's key
// (kept outside the repo) produces a signature that verifies against the
// out-of-repo trust anchor, so a swapped address fails this test and thus the
// pipeline before a release is built. It skips harmlessly while the addresses are
// placeholders, when ssh-keygen is unavailable, or when the trust anchor is not
// on this machine (e.g. a fresh clone without the private key dir).

package donation

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// signIdentity is the -I value used when signing and verifying.
const signIdentity = "donation"

// pkgDir is the directory of this package's source (where donation.sig lives).
func pkgDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate package source dir")
	}
	return filepath.Dir(file)
}

// anchorPath resolves the out-of-repo allowed_signers trust anchor. It sits
// beside the private signing key, three levels above the module (repo is
// github/, its parent holds private/donation_keys/).
func anchorPath(t *testing.T) string {
	if override := os.Getenv("REPOINT_DONATION_ALLOWED_SIGNERS"); override != "" {
		return override
	}
	return filepath.Join(pkgDir(t), "..", "..", "..", "private", "donation_keys", "allowed_signers")
}

func isFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func TestDonationTableSigned(t *testing.T) {
	if !HasConfigured() {
		t.Skip("donation addresses are still placeholders - nothing to protect yet")
	}
	if _, err := exec.LookPath("ssh-keygen"); err != nil {
		t.Skip("ssh-keygen not available")
	}
	anchor := anchorPath(t)
	if !isFile(anchor) {
		t.Skipf("donation trust anchor not present (%s) - cannot verify here", anchor)
	}

	sig := filepath.Join(pkgDir(t), "donation.sig")
	if !isFile(sig) {
		t.Fatal("real donation addresses are set but donation.sig is missing - run packaging/sign-donations.bash")
	}

	canonical := filepath.Join(t.TempDir(), "donation.canonical")
	if err := os.WriteFile(canonical, CanonicalBytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	data, err := os.Open(canonical)
	if err != nil {
		t.Fatal(err)
	}
	defer data.Close()

	cmd := exec.Command("ssh-keygen", "-Y", "verify",
		"-f", anchor, "-I", signIdentity, "-n", SignatureNamespace, "-s", sig)
	cmd.Stdin = data
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("donation signature failed to verify: %v: %s\n"+
			"the table was changed without re-signing, or the signature is invalid", err, out)
	}
}
