# Donation address signing

`--donate` shows the project's donation addresses (`source/donation/donation.go`). Anyone can edit that file, so the table is signed: the cicd gate refuses to pass unless the current table carries a valid signature made with the maintainer's key. A changed address that isn't re-signed fails the gate, so a stray or malicious edit can't quietly redirect donations in a release.

The addresses stay visible - they are meant to be seen. What the signature protects is that they are the maintainer's, not a substitute.

## What lives where

- **Private signing key** - `private/donation_keys/donation_ed25519`, outside the git repo (the repo is `github/`; `private/` is its cloud-synced sibling, so the key is passphrase-protected and only ever stored encrypted). Never commit it.
- **Trust anchor** - `private/donation_keys/allowed_signers`, also outside the repo. The verify gate reads the public key from here, not from the repo, so a pull request can't swap the key along with an address.
- **Signature** - `source/donation/donation.sig`, committed alongside `donation.go`.
- **Signed content** - `donation.CanonicalBytes()`: the label, kind and value of every entry, in order. Reordering or editing the table changes these bytes and invalidates the signature.

## Generate the key (one time)

```bash
mkdir -p private/donation_keys && chmod 700 private/donation_keys
ssh-keygen -t ed25519 -C "repoint-symlink donation signing" -f private/donation_keys/donation_ed25519
printf 'donation namespaces="donation" %s\n' "$(cat private/donation_keys/donation_ed25519.pub)" > private/donation_keys/allowed_signers
```

Enter a strong passphrase at the prompt (the folder is cloud-synced). Back up the passphrase-protected key somewhere safe - without it you can never update the addresses again.

## Set addresses and sign

1. Replace the `PLACEHOLDER_*` values in `source/donation/donation.go` with the real addresses/URLs.
2. Sign: `packaging/sign-donations.bash` (prompts for the key passphrase; writes `donation.sig`).
3. Commit `donation.go` and `donation.sig` together.

## How the gate works

The Go test `TestDonationTableSigned` (in `source/donation/`) runs in the gating cicd test stage (`go test ./...`). It:

- skips while every address is still a placeholder (nothing to protect yet);
- skips when `ssh-keygen` or the trust anchor is absent (e.g. a fresh clone without the key dir) - so it never breaks a contributor's test run;
- otherwise verifies `donation.sig` over the current table against the anchor, and fails if it doesn't match.

Point it at a different anchor with `REPOINT_DONATION_ALLOWED_SIGNERS` if needed.

## Rotate / update

Editing an address is the same loop: change `donation.go`, re-run `sign-donations.bash`, commit both. Replacing the key means generating a new one and re-running the `allowed_signers` line above, then re-signing.
