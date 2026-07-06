//	Copyright © 2026 Jim Collier (ID: 1cv◂‡Vᛦ)
//	Licensed under the GNU General Public License v2.0 or later. Full text at:
//		https://spdx.org/licenses/GPL-2.0-or-later.html
//	SPDX-License-Identifier: GPL-2.0-or-later

// Package filter selects a subset of paths through an ordered pipeline of rules.
// Each rule narrows, subtracts, or re-adds, and its effect is fixed regardless
// of where it sits, so a rule set reads one rule at a time. Regexes are
// PCRE-level (regexp2: lookaround, backrefs, inline (?i)); globs are find-style,
// where '*' and '?' span '/'. Nothing here is tied to a particular CLI or file
// type, so the same engine can back any path selector.
package filter

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/dlclark/regexp2"
)

// MatchTimeout caps a single regex match so a pathological pattern (one that
// backtracks catastrophically) fails instead of hanging. Normal patterns finish
// in microseconds, so this never bites real use. A var, not a const, so a caller
// or test can dial it down.
var MatchTimeout = 5 * time.Second

// Op is what a rule does to the running keep-state for a path.
type Op int

const (
	Narrow   Op = iota // keep = keep AND match
	Subtract           // keep = keep AND NOT match
	Readd              // keep = keep OR match (re-admit from the original set)
)

// Spec describes one rule before compilation. Pattern is a regex when Glob is
// false, or a find-style glob when true. For a glob, Base matches the basename
// only (otherwise the whole slash-normalized path) and Fold makes it
// case-insensitive. Base and Fold are ignored for a regex, which carries its own
// inline flags. Label, if set, names the rule in a compile error (else "rule N").
type Spec struct {
	Op      Op
	Pattern string
	Glob    bool
	Base    bool
	Fold    bool
	Label   string
}

// Set is a compiled, ready-to-apply rule pipeline.
type Set struct {
	rules []rule
}

type rule struct {
	op    Op
	match func(path string) bool
}

// Compile turns specs into a Set, preserving their order. A bad pattern stops
// compilation and reports which rule failed.
func Compile(specs []Spec) (*Set, error) {
	set := &Set{}
	for i, spec := range specs {
		label := spec.Label
		if label == "" {
			label = fmt.Sprintf("rule %d", i+1)
		}
		r := rule{op: spec.Op}
		if spec.Glob {
			re, err := compileGlob(spec.Pattern, spec.Fold)
			if err != nil {
				return nil, fmt.Errorf("%s: bad glob %q: %w", label, spec.Pattern, err)
			}
			r.match = globMatcher(re, spec.Base)
		} else {
			re, err := CompileRegex(spec.Pattern, false)
			if err != nil {
				return nil, fmt.Errorf("%s: bad regex %q: %w", label, spec.Pattern, err)
			}
			r.match = reMatcher(re)
		}
		set.rules = append(set.rules, r)
	}
	return set, nil
}

// Selects reports whether path survives the whole pipeline. An empty Set keeps
// everything.
func (s *Set) Selects(path string) bool {
	keep := true
	for _, r := range s.rules {
		hit := r.match(path)
		switch r.op {
		case Narrow:
			keep = keep && hit
		case Subtract:
			keep = keep && !hit
		case Readd:
			keep = keep || hit
		}
	}
	return keep
}

// CompileRegex compiles a regexp2 pattern with MatchTimeout applied. fold adds
// case-insensitivity on top of any inline flags already in the pattern.
func CompileRegex(pattern string, fold bool) (*regexp2.Regexp, error) {
	opt := regexp2.None
	if fold {
		opt = regexp2.IgnoreCase
	}
	re, err := regexp2.Compile(pattern, opt)
	if err != nil {
		return nil, err
	}
	re.MatchTimeout = MatchTimeout
	return re, nil
}

func reMatcher(re *regexp2.Regexp) func(string) bool {
	return func(path string) bool {
		ok, _ := re.MatchString(path)
		return ok
	}
}

// globMatcher matches a compiled glob against the basename (base) or the whole
// path. Whole-path matching is slash-normalized so a '/' pattern works on
// Windows paths too.
func globMatcher(re *regexp2.Regexp, base bool) func(string) bool {
	return func(path string) bool {
		subject := filepath.ToSlash(path)
		if base {
			subject = filepath.Base(path)
		}
		ok, _ := re.MatchString(subject)
		return ok
	}
}

func compileGlob(glob string, fold bool) (*regexp2.Regexp, error) {
	return CompileRegex(globToRegex(glob), fold)
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
