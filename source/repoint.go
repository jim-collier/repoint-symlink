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
	fixed    bool
	editMode bool
}

func buildRepointer(o *options) (*repointer, error) {
	r := &repointer{to: o.to, fixed: o.fixed}
	if !o.fromSet || o.from == "" {
		if o.fromSet && o.from == "" {
			warnf("--from is empty; listing matches only")
		}
		return r, nil // list-only
	}
	r.editMode = true
	if o.fixed {
		r.fromLit = o.from
		return r, nil
	}
	re, err := regexp2.Compile(o.from, regexp2.None)
	if err != nil {
		return nil, fmt.Errorf("bad --from regex %q: %w", o.from, err)
	}
	r.fromRE = re
	return r, nil
}

func (r *repointer) transform(target string) (string, error) {
	if !r.editMode {
		return target, nil
	}
	if r.fixed {
		return strings.ReplaceAll(target, r.fromLit, r.to), nil
	}
	return r.fromRE.Replace(target, r.to, -1, -1)
}

// process finds, reports, and (unless dry-run) rewrites matching links.
// Returns the number of links changed and whether any write failed.
func process(o *options, f *filters, r *repointer, entries []LinkEntry) (changed int, failed bool) {
	matched := 0
	for _, e := range entries {
		if !f.selects(e.Path) {
			continue
		}
		matched++

		if !r.editMode {
			if !o.quiet {
				fmt.Printf("%s -> %s  [%s]\n", e.Path, e.Target, e.Kind)
			}
			continue
		}

		newTarget, err := r.transform(e.Target)
		if err != nil {
			warnf("%s: transform failed: %v", e.Path, err)
			failed = true
			continue
		}
		if newTarget == e.Target {
			if o.verbose {
				fmt.Printf("unchanged: %s -> %s\n", e.Path, e.Target)
			}
			continue
		}

		verb := "repointed"
		if o.dryRun {
			verb = "would repoint"
		}
		if !o.quiet {
			fmt.Printf("%s: %s  [%s]\n    %s -> %s\n", verb, e.Path, e.Kind, e.Target, newTarget)
		}
		if o.dryRun {
			changed++
			continue
		}
		if err := writeLinkTarget(e, newTarget); err != nil {
			warnf("%s: write failed: %v", e.Path, err)
			failed = true
			continue
		}
		changed++
	}

	printSummary(o, r, matched, changed)
	return changed, failed
}

func printSummary(o *options, r *repointer, matched, changed int) {
	if o.quiet {
		return
	}
	if !r.editMode {
		fmt.Printf("\n%d matching %s.\n", matched, plural(matched, "link", "links"))
		return
	}
	if o.dryRun {
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
