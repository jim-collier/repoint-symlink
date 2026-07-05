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
// (--include/--exclude regexes and --[i]name/--[i]wholename globs) is one rule
// here, kept in command-line order, and matched against the link's own path -
// --from/--to handle the target side.
//
// Evaluation walks the rules left to right, starting from "everything kept":
//   - a positive rule (include/name/iname/wholename/iwholename) that follows
//     another positive rule (or is first) NARROWS: keep = keep AND match.
//   - a positive rule that follows an --exclude can EXPAND: keep = keep OR match,
//     bringing links a prior exclude had dropped back in.
//   - --exclude always narrows: keep = keep AND NOT match.
//
// So order matters: an include after an exclude means something different from
// an include before it. Regexes are regexp2 (PCRE-level: lookaround, backrefs,
// inline (?i)); globs are find-style (--wholename == find -wholename), where a
// '*' spans '/' and patterns should be quoted to survive the shell.
type filters struct {
	rules []rule
}

// rule is one compiled selection flag: whether it adds or subtracts, and a
// matcher closure over the link path.
type rule struct {
	positive bool
	match    func(linkPath string) bool
}

func compileFilters(opts *options) (*filters, error) {
	f := &filters{}
	for _, sr := range opts.rules {
		r := rule{positive: sr.kind.positive()}
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
	prevExclude := false
	for _, r := range f.rules {
		hit := r.match(linkPath)
		if r.positive {
			if prevExclude {
				keep = keep || hit // expand: bring back what an exclude dropped
			} else {
				keep = keep && hit // narrow
			}
			prevExclude = false
		} else {
			keep = keep && !hit // exclude only ever narrows
			prevExclude = true
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
