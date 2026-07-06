<!-- markdownlint-disable MD010 -- No hard tabs -->
<!-- markdownlint-disable MD024 -- No duplicate headings [OK with no TOC] -->
# Changelog

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## v0.1.0 (unreleased) - WIP

### Added

- Initial working tool: recursively find symlinks under a start folder and repoint their targets.
- Filters: repeatable `--include`/`--exclude`/`--re-include` (PCRE-level regex), `--name`/`--iname` basename globs, `--wholename`/`--iwholename` whole-path globs; `--max-depth`. Every filter is one rule in a single ordered pipeline, each with one fixed effect: include and the globs narrow, `--exclude` subtracts, `--re-include` re-admits from the original scan (the only widener). Globs are find-style (`*` spans `/`).
- Regex `--from` + template `--to` (`$1`, `${name}`); literal replace with `-F` / `--literal`.
- Apply-by-default with `--dry-run` preview; list-only mode when no `--from`.
- Regex matching is bounded by a timeout, so a pathological pattern fails instead of hanging.
- `--inc-target` / `--exc-target`: select links by their current target (where they point), a second ordered pipeline reusing the filter engine.
- Traversal controls: `--no-cross-device` (alias `--xdev`, `find -xdev` style) and `-L` / `--follow-links` (descend into directory symlinks, loop-safe).
- `--renormal-relative` / `--renormal-absolute`: normalize each target's spelling (relative to the link, or cleaned absolute); usable without `--from`.
- `--confirm`: preview the whole plan, then prompt once before writing. `-0` / `--print0`: NUL-separated output for scripting.
- `--donate`: show the project's donation addresses. The address table is signed with the maintainer's key (kept outside the repo) and a cicd gate rejects a release whose table was edited without re-signing.
- Windows: NTFS junction and `.lnk` shortcut targets in addition to symlinks (built and cross-compiled, pending run-test on real Windows).
- Cross-compile matrix for linux/macOS/windows on amd64+arm64; Go unit tests and an integration test harness.
