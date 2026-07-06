//	Copyright © 2026 Jim Collier (ID: 1cv◂‡Vᛦ)
//	Licensed under the GNU General Public License v2.0 or later. Full text at:
//		https://spdx.org/licenses/GPL-2.0-or-later.html
//	SPDX-License-Identifier: GPL-2.0-or-later

package main

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/dlclark/regexp2"
)

// filters selects which discovered links to act on. Every selection flag
// (--include/--exclude/--re-include regexes and --[i]name/--[i]wholename globs)
// is one rule here, kept in command-line order, and matched against the link's
// own path - --from/--to handle the target side.
//
// Each rule has ONE intrinsic effect, independent of its neighbours, so the set
// can be reasoned about sequentially, one flag at a time:
//   - --include and the name/wholename globs NARROW: keep = keep AND match.
//   - --exclude SUBTRACTS: keep = keep AND NOT match.
//   - --re-include RE-ADMITS from the original scan: keep = keep OR match. It
//     brings back any link matching it, even one a prior --exclude dropped.
//
// Narrow and subtract only ever shrink the set; --re-include is the only widener.
// Order still matters (a later narrow/subtract applies to whatever a re-include
// widened). Regexes are regexp2 (PCRE-level: lookaround, backrefs, inline (?i));
// globs are find-style (--wholename == find -wholename), where a '*' spans '/'
// and patterns should be quoted to survive the shell.
type filters struct {
	rules []rule
}

// op is what a rule does to the running keep-state for a given link.
type op int

const (
	opNarrow   op = iota // keep = keep AND match  (include / name globs)
	opSubtract           // keep = keep AND NOT match  (exclude)
	opReadd              // keep = keep OR match  (re-include, from original scan)
)

// rule is one compiled selection flag: its operator and a matcher over the path.
type rule struct {
	op    op
	match func(linkPath string) bool
}

// kindOp maps a selection flag to its set operator.
func kindOp(k selKind) op {
	switch k {
	case selExclude:
		return opSubtract
	case selReInclude:
		return opReadd
	default:
		return opNarrow
	}
}

func compileFilters(opts *options) (*filters, error) {
	f := &filters{}
	for _, sr := range opts.rules {
		r := rule{op: kindOp(sr.kind)}
		if sr.kind.isGlob() {
			fold := sr.kind == selIName || sr.kind == selIWholename
			onBase := sr.kind == selName || sr.kind == selIName
			re, err := compileGlob(sr.pat, fold)
			if err != nil {
				return nil, fmt.Errorf("bad --%s glob %q: %w", sr.kind.flag(), sr.pat, err)
			}
			r.match = globMatcher(re, onBase)
		} else {
			re, err := regexp2.Compile(sr.pat, regexp2.None)
			if err != nil {
				return nil, fmt.Errorf("bad --%s regex %q: %w", sr.kind.flag(), sr.pat, err)
			}
			r.match = reMatcher(re)
		}
		f.rules = append(f.rules, r)
	}
	return f, nil
}

func (f *filters) selects(linkPath string) bool {
	keep := true
	for _, r := range f.rules {
		hit := r.match(linkPath)
		switch r.op {
		case opNarrow:
			keep = keep && hit
		case opSubtract:
			keep = keep && !hit
		case opReadd:
			keep = keep || hit
		}
	}
	return keep
}

func reMatcher(re *regexp2.Regexp) func(string) bool {
	return func(linkPath string) bool {
		ok, _ := re.MatchString(linkPath)
		return ok
	}
}

// globMatcher matches a compiled glob against the basename (onBase) or the whole
// path. Whole-path matching is slash-normalized so a '/' pattern works on
// Windows paths too.
func globMatcher(re *regexp2.Regexp, onBase bool) func(string) bool {
	return func(linkPath string) bool {
		subject := filepath.ToSlash(linkPath)
		if onBase {
			subject = filepath.Base(linkPath)
		}
		ok, _ := re.MatchString(subject)
		return ok
	}
}

func compileGlob(glob string, fold bool) (*regexp2.Regexp, error) {
	opt := regexp2.None
	if fold {
		opt = regexp2.IgnoreCase
	}
	return regexp2.Compile(globToRegex(glob), opt)
}

// globToRegex translates a find-style glob to an anchored regex. Unlike shell
// path globbing, '*' and '?' span '/' (find's -wholename convention). '\' escapes
// the next char; '[...]' is a character class ('!' negates).
func globToRegex(glob string) string {
	var b strings.Builder
	b.WriteString(`\A(?:`)
	runes := []rune(glob)
	for i := 0; i < len(runes); i++ {
		switch c := runes[i]; c {
		case '*':
			b.WriteString(`.*`)
		case '?':
			b.WriteString(`.`)
		case '\\':
			if i+1 < len(runes) {
				i++
				b.WriteString(regexp.QuoteMeta(string(runes[i])))
			} else {
				b.WriteString(regexp.QuoteMeta(`\`))
			}
		case '[':
			if class, next, ok := globClass(runes, i); ok {
				b.WriteString(class)
				i = next
			} else {
				b.WriteString(regexp.QuoteMeta("["))
			}
		default:
			b.WriteString(regexp.QuoteMeta(string(c)))
		}
	}
	b.WriteString(`)\z`)
	return b.String()
}

// globClass parses a '[...]' character class starting at runes[open]. It returns
// the regex class, the index of the closing ']', and whether a class was found
// (an unterminated '[' is treated as a literal by the caller).
func globClass(runes []rune, open int) (string, int, bool) {
	i := open + 1
	neg := false
	if i < len(runes) && runes[i] == '!' {
		neg = true
		i++
	}
	start := i
	if i < len(runes) && runes[i] == ']' { // a ']' right after '[' is a literal member
		i++
	}
	for i < len(runes) && runes[i] != ']' {
		i++
	}
	if i >= len(runes) {
		return "", 0, false // no closing ']'
	}
	var b strings.Builder
	b.WriteByte('[')
	if neg {
		b.WriteByte('^')
	}
	for _, m := range runes[start:i] {
		switch m {
		case '\\', ']', '^':
			b.WriteByte('\\')
		}
		b.WriteRune(m)
	}
	b.WriteByte(']')
	return b.String(), i, true
}
