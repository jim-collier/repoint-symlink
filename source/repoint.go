//	Copyright © 2026 Jim Collier (ID: 1cv◂‡Vᛦ)
//	Licensed under the GNU General Public License v2.0 or later. Full text at:
//		https://spdx.org/licenses/GPL-2.0-or-later.html
//	SPDX-License-Identifier: GPL-2.0-or-later

package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/dlclark/regexp2"
	"github.com/jim-collier/repoint-symlink/filter"
)

// renormMode re-normalizes a target after any --from/--to rewrite. It can be
// used on its own (no --from) to just tidy existing targets.
type renormMode int

const (
	renormNone     renormMode = iota
	renormRelative            // rewrite the target relative to the link's own dir
	renormAbsolute            // rewrite the target as a cleaned absolute path
)

// repointer turns a current target into a new one per --from/--to. In regex
// mode --from is a pattern and --to a template ($1, ${name}); with -F it is a
// plain literal and every occurrence is replaced. An optional renormal step
// then rewrites the result relative to, or absolute from, the link's own dir.
type repointer struct {
	fromRE   *regexp2.Regexp
	fromLit  string
	to       string
	literal  bool
	hasFrom  bool
	renormal renormMode
	editMode bool
}

func buildRepointer(opts *options) (*repointer, error) {
	rp := &repointer{to: opts.to, literal: opts.literal}

	switch {
	case opts.renormRel && opts.renormAbs:
		return nil, fmt.Errorf("--renormal-relative and --renormal-absolute are mutually exclusive")
	case opts.renormRel:
		rp.renormal = renormRelative
	case opts.renormAbs:
		rp.renormal = renormAbsolute
	}

	if opts.fromSet && opts.from != "" {
		rp.hasFrom = true
		if opts.literal {
			rp.fromLit = opts.from
		} else {
			re, err := filter.CompileRegex(opts.from, false)
			if err != nil {
				return nil, fmt.Errorf("bad --from regex %q: %w", opts.from, err)
			}
			rp.fromRE = re
		}
	} else if opts.fromSet && opts.from == "" && rp.renormal == renormNone {
		warnf("--from is empty; listing matches only")
	}

	// A renormal-only run still edits (rewrites targets) even without --from.
	rp.editMode = rp.hasFrom || rp.renormal != renormNone
	return rp, nil
}

// transform applies the --from/--to (or literal) substitution. Renormalization
// is a separate step (renormalizeTarget) since it needs the link's own path.
func (r *repointer) transform(target string) (string, error) {
	if !r.editMode || !r.hasFrom {
		return target, nil
	}
	if r.literal {
		return strings.ReplaceAll(target, r.fromLit, r.to), nil
	}
	return r.fromRE.Replace(target, r.to, -1, -1)
}

// renormalizeTarget rewrites target to an absolute cleaned path or to one
// relative to the link's own directory. It normalizes the logical path only -
// it does not resolve symlinks in the target, so the link keeps pointing at the
// same place, just spelled differently.
func renormalizeTarget(linkPath, target string, mode renormMode) (string, error) {
	if mode == renormNone {
		return target, nil
	}
	dir := filepath.Dir(linkPath)
	abs := target
	if !filepath.IsAbs(abs) {
		abs = filepath.Join(dir, abs)
	}
	abs, err := filepath.Abs(abs)
	if err != nil {
		return target, err
	}
	if mode == renormAbsolute {
		return abs, nil
	}
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return target, err
	}
	rel, err := filepath.Rel(absDir, abs)
	if err != nil {
		return target, err
	}
	return rel, nil
}

// process finds, reports, and (unless dry-run) rewrites matching links.
// Returns the number of links changed and whether any write failed.
func process(opts *options, filt, targetFilt *filter.Set, rp *repointer, entries []LinkEntry) (changed int, failed bool) {
	matched := 0
	for _, entry := range entries {
		if !filt.Selects(entry.Path) || !targetFilt.Selects(entry.Target) {
			continue
		}
		matched++

		if !rp.editMode {
			switch {
			case opts.print0:
				emitRecord(entry.Path)
			case !opts.quiet:
				fmt.Printf("%s -> %s  [%s]\n", entry.Path, entry.Target, entry.Kind)
			}
			continue
		}

		newTarget, err := rp.transform(entry.Target)
		if err != nil {
			warnf("%s: transform failed: %v", entry.Path, err)
			failed = true
			continue
		}
		if rp.renormal != renormNone {
			normalized, nerr := renormalizeTarget(entry.Path, newTarget, rp.renormal)
			if nerr != nil {
				warnf("%s: renormalize failed: %v", entry.Path, nerr)
				failed = true
				continue
			}
			newTarget = normalized
		}
		if newTarget == entry.Target {
			if opts.verbose && !opts.print0 {
				fmt.Printf("unchanged: %s -> %s\n", entry.Path, entry.Target)
			}
			continue
		}

		verb := "repointed"
		if opts.dryRun {
			verb = "would repoint"
		}
		if !opts.quiet && !opts.print0 {
			fmt.Printf("%s: %s  [%s]\n    %s -> %s\n", verb, entry.Path, entry.Kind, entry.Target, newTarget)
		}
		if opts.dryRun {
			if opts.print0 {
				emitRecord(entry.Path)
			}
			changed++
			continue
		}
		if err := writeLinkTarget(entry, newTarget); err != nil {
			warnf("%s: write failed: %v", entry.Path, err)
			failed = true
			continue
		}
		if opts.print0 {
			emitRecord(entry.Path)
		}
		changed++
	}

	if !opts.print0 {
		printSummary(opts, rp, matched, changed)
	}
	return changed, failed
}

// emitRecord writes one machine-readable record: the link path, NUL-terminated
// (find -print0 / xargs -0 convention). Used only with --print0.
func emitRecord(path string) {
	fmt.Printf("%s\x00", path)
}

func printSummary(opts *options, rp *repointer, matched, changed int) {
	if opts.quiet {
		return
	}
	if !rp.editMode {
		fmt.Printf("\n%d matching %s.\n", matched, plural(matched, "link", "links"))
		return
	}
	if opts.dryRun {
		fmt.Printf("\nDry run: %d of %d matched %s would change (nothing written).\n",
			changed, matched, plural(matched, "link", "links"))
		return
	}
	fmt.Printf("\nRepointed %d of %d matched %s.\n", changed, matched, plural(matched, "link", "links"))
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}
