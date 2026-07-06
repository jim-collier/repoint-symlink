//	Copyright © 2026 Jim Collier (ID: 1cv◂‡Vᛦ)
//	Licensed under the GNU General Public License v2.0 or later. Full text at:
//		https://spdx.org/licenses/GPL-2.0-or-later.html
//	SPDX-License-Identifier: GPL-2.0-or-later

package main

import (
	"fmt"
	"strings"

	"github.com/dlclark/regexp2"
)

// repointer turns a current target into a new one per --from/--to. In regex
// mode --from is a pattern and --to a template ($1, ${name}); with -F it is a
// plain literal and every occurrence is replaced.
type repointer struct {
	fromRE   *regexp2.Regexp
	fromLit  string
	to       string
	literal  bool
	editMode bool
}

func buildRepointer(opts *options) (*repointer, error) {
	rp := &repointer{to: opts.to, literal: opts.literal}
	if !opts.fromSet || opts.from == "" {
		if opts.fromSet && opts.from == "" {
			warnf("--from is empty; listing matches only")
		}
		return rp, nil // list-only
	}
	rp.editMode = true
	if opts.literal {
		rp.fromLit = opts.from
		return rp, nil
	}
	re, err := compileRE(opts.from, regexp2.None)
	if err != nil {
		return nil, fmt.Errorf("bad --from regex %q: %w", opts.from, err)
	}
	rp.fromRE = re
	return rp, nil
}

func (r *repointer) transform(target string) (string, error) {
	if !r.editMode {
		return target, nil
	}
	if r.literal {
		return strings.ReplaceAll(target, r.fromLit, r.to), nil
	}
	return r.fromRE.Replace(target, r.to, -1, -1)
}

// process finds, reports, and (unless dry-run) rewrites matching links.
// Returns the number of links changed and whether any write failed.
func process(opts *options, filt *filters, rp *repointer, entries []LinkEntry) (changed int, failed bool) {
	matched := 0
	for _, entry := range entries {
		if !filt.selects(entry.Path) {
			continue
		}
		matched++

		if !rp.editMode {
			if !opts.quiet {
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
		if newTarget == entry.Target {
			if opts.verbose {
				fmt.Printf("unchanged: %s -> %s\n", entry.Path, entry.Target)
			}
			continue
		}

		verb := "repointed"
		if opts.dryRun {
			verb = "would repoint"
		}
		if !opts.quiet {
			fmt.Printf("%s: %s  [%s]\n    %s -> %s\n", verb, entry.Path, entry.Kind, entry.Target, newTarget)
		}
		if opts.dryRun {
			changed++
			continue
		}
		if err := writeLinkTarget(entry, newTarget); err != nil {
			warnf("%s: write failed: %v", entry.Path, err)
			failed = true
			continue
		}
		changed++
	}

	printSummary(opts, rp, matched, changed)
	return changed, failed
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
