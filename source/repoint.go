//	Copyright © 2026 Jim Collier (ID: 1cv◂‡Vᛦ)
//	Licensed under the GNU General Public License v2.0 or later. Full text at:
//		https://spdx.org/licenses/GPL-2.0-or-later.html
//	SPDX-License-Identifier: GPL-2.0-or-later

package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
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

	if opts.confirm && opts.print0 {
		return nil, fmt.Errorf("--confirm cannot be combined with --print0 (interactive vs machine output)")
	}

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

// change is one planned rewrite: a link and the target it should point to.
type change struct {
	entry     LinkEntry
	newTarget string
}

// process finds, reports, and (unless dry-run or a declined confirm) rewrites
// matching links. Returns the number changed and whether any write failed.
func process(opts *options, filt, targetFilt *filter.Set, rp *repointer, entries []LinkEntry) (changed int, failed bool) {
	matched := 0
	var plan []change
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
		plan = append(plan, change{entry, newTarget})
	}

	if !rp.editMode {
		if !opts.print0 {
			printSummary(opts, rp, matched, changed)
		}
		return changed, failed
	}

	// Dry run: show the plan (or emit records), write nothing.
	if opts.dryRun {
		for _, c := range plan {
			if opts.print0 {
				emitRecord(c.entry.Path)
			} else {
				showChange("would repoint", c, opts)
			}
		}
		changed = len(plan)
		if !opts.print0 {
			printSummary(opts, rp, matched, changed)
		}
		return changed, failed
	}

	// Confirm: preview the whole plan, then one prompt before writing anything.
	if opts.confirm && len(plan) > 0 {
		for _, c := range plan {
			showChange("would repoint", c, opts)
		}
		if !askProceed(len(plan)) {
			fmt.Fprintln(os.Stderr, "Aborted; nothing written.")
			return 0, false
		}
	}

	for _, c := range plan {
		// Non-confirm human runs echo each change as it is applied; a confirmed
		// run already previewed the whole plan above, so don't repeat it.
		if !opts.confirm && !opts.print0 {
			showChange("repointed", c, opts)
		}
		if err := writeLinkTarget(c.entry, c.newTarget); err != nil {
			warnf("%s: write failed: %v", c.entry.Path, err)
			failed = true
			continue
		}
		if opts.print0 {
			emitRecord(c.entry.Path)
		}
		changed++
	}

	if !opts.print0 {
		printSummary(opts, rp, matched, changed)
	}
	return changed, failed
}

// showChange prints one planned/applied rewrite in the human format.
func showChange(verb string, c change, opts *options) {
	if opts.quiet || opts.print0 {
		return
	}
	fmt.Printf("%s: %s  [%s]\n    %s -> %s\n", verb, c.entry.Path, c.entry.Kind, c.entry.Target, c.newTarget)
}

// confirmReader is where askProceed reads the y/N answer; overridable in tests.
var confirmReader io.Reader = os.Stdin

// askProceed prompts on stderr and reads one line; only "y"/"yes" proceeds.
func askProceed(n int) bool {
	fmt.Fprintf(os.Stderr, "\nApply %d %s? (y|N): ", n, plural(n, "change", "changes"))
	line, _ := bufio.NewReader(confirmReader).ReadString('\n')
	switch strings.TrimSpace(strings.ToLower(line)) {
	case "y", "yes":
		return true
	default:
		return false
	}
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
