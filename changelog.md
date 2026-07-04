<!-- markdownlint-disable MD010 -- No hard tabs -->
<!-- markdownlint-disable MD024 -- No duplicate headings [OK with no TOC] -->
# Changelog

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## v0.1.0 (unreleased) - WIP

### Added

- Initial working tool: recursively find symlinks under a start folder and repoint their targets.
- Filters: repeatable `--include`/`--exclude` (PCRE-level regex) and `--name`/`--iname` globs; `--max-depth`.
- Regex `--from` + template `--to` (`$1`, `${name}`); literal replace with `-F`.
- Apply-by-default with `--dry-run` preview; list-only mode when no `--from`.
- Windows: NTFS junction and `.lnk` shortcut targets in addition to symlinks (built and cross-compiled, pending run-test on real Windows).
- Cross-compile matrix for linux/macOS/windows on amd64+arm64; Go unit tests and an integration test harness.
