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
)

// filters decides which discovered links to act on. All match against the
// link's own path (not its target - --from/--to handle the target side).
//
// Selection = pathGate AND nameGate AND not(any exclude), where:
//   - pathGate: no --inc given -> pass; else the full path matches ANY --inc.
//   - nameGate: no --name/--iname given -> pass; else the basename matches ANY.
//   - exclude:  the full path matching ANY --exc rejects the link.
//
// Multiples of the same kind OR together; different kinds AND. Regexes are
// regexp2 (PCRE-level: lookaround, backrefs, inline (?i) case flag).
type filters struct {
	includes []*regexp2.Regexp // path regex, OR
	excludes []*regexp2.Regexp // path regex, any match rejects
	names    []string          // basename globs, case-sensitive, OR
	inames   []string          // basename globs, case-insensitive, OR
}

func compileFilters(opts *options) (*filters, error) {
	filt := &filters{names: opts.names, inames: opts.inames}
	for _, pattern := range opts.includes {
		re, err := regexp2.Compile(pattern, regexp2.None)
		if err != nil {
			return nil, fmt.Errorf("bad --include regex %q: %w", pattern, err)
		}
		filt.includes = append(filt.includes, re)
	}
	for _, pattern := range opts.excludes {
		re, err := regexp2.Compile(pattern, regexp2.None)
		if err != nil {
			return nil, fmt.Errorf("bad --exclude regex %q: %w", pattern, err)
		}
		filt.excludes = append(filt.excludes, re)
	}
	// Validate name globs up front so a typo fails loudly, not silently.
	for _, glob := range append(append([]string{}, opts.names...), opts.inames...) {
		if _, err := filepath.Match(glob, ""); err != nil {
			return nil, fmt.Errorf("bad glob %q: %w", glob, err)
		}
	}
	return filt, nil
}

func (f *filters) selects(linkPath string) bool {
	if len(f.includes) > 0 && !anyMatch(f.includes, linkPath) {
		return false
	}
	if len(f.names) > 0 || len(f.inames) > 0 {
		base := filepath.Base(linkPath)
		if !globMatch(f.names, base, false) && !globMatch(f.inames, base, true) {
			return false
		}
	}
	if anyMatch(f.excludes, linkPath) {
		return false
	}
	return true
}

func anyMatch(patterns []*regexp2.Regexp, subject string) bool {
	for _, re := range patterns {
		if ok, _ := re.MatchString(subject); ok {
			return true
		}
	}
	return false
}

func globMatch(globs []string, name string, fold bool) bool {
	for _, glob := range globs {
		foldGlob, foldName := glob, name
		if fold {
			foldGlob, foldName = strings.ToLower(glob), strings.ToLower(name)
		}
		if ok, _ := filepath.Match(foldGlob, foldName); ok {
			return true
		}
	}
	return false
}
