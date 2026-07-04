//	Copyright © 2026 Jim Collier (ID: 1cv◂‡Vᛦ)
//	Licensed under the GNU General Public License v2.0 or later. Full text at:
//		https://spdx.org/licenses/GPL-2.0-or-later.html
//	SPDX-License-Identifier: GPL-2.0-or-later

package main

import (
	"fmt"
	"sort"
	"strings"
)

// options is the parsed command line.
type options struct {
	dir      string   // start folder (positional 1, default ".")
	from     string   // regex (or literal with -F); positional 2
	to       string   // replacement template; positional 3
	fromSet  bool     // was --from / positional 2 given? (enables edit mode)
	includes []string // --inc, repeatable
	excludes []string // --exc, repeatable
	names    []string // --name, repeatable
	inames   []string // --iname, repeatable
	maxDepth int      // --max-depth, -1 = unlimited
	fixed    bool     // -F: treat --from as a literal string
	dryRun   bool     // -n: preview, do not write
	verbose  bool     // -v
	quiet    bool     // -q
	// terminal actions
	showVersion  bool
	showHelp     bool
	showExamples bool
}

// long flags that take a value, and the min-unambiguous-prefix minimum length.
// Prefix abbreviation is allowed down to 3 chars (so --inc, --exc resolve),
// but an exact spelling always wins regardless of length (so --to works).
const minPrefix = 3

var valueFlags = []string{"include", "exclude", "name", "iname", "from", "to", "max-depth"}
var boolFlags = []string{"dry-run", "fixed", "verbose", "quiet", "version", "help", "examples"}

func parseArgs(argv []string) (*options, error) {
	o := &options{dir: ".", maxDepth: -1}
	var pos []string
	seenFrom, seenTo := false, false

	i := 0
	for i < len(argv) {
		a := argv[i]
		i++

		if a == "--" { // rest are positional
			pos = append(pos, argv[i:]...)
			break
		}

		// Short single-dash bool flags (bundling allowed: -nv).
		if len(a) >= 2 && a[0] == '-' && a[1] != '-' {
			if err := parseShort(a, o); err != nil {
				return nil, err
			}
			continue
		}

		// Long flags.
		if strings.HasPrefix(a, "--") {
			name, val, hasVal := strings.Cut(a[2:], "=")
			canon, err := resolveLong(name)
			if err != nil {
				return nil, err
			}
			if isBool(canon) {
				b, err := boolValue(canon, val, hasVal)
				if err != nil {
					return nil, err
				}
				setBool(o, canon, b)
				continue
			}
			// value flag
			if !hasVal {
				if i >= len(argv) {
					return nil, fmt.Errorf("--%s needs a value", canon)
				}
				val = argv[i]
				i++
			}
			if err := setValue(o, canon, val, &seenFrom, &seenTo); err != nil {
				return nil, err
			}
			continue
		}

		pos = append(pos, a)
	}

	// Positional fill: dir, from, to. Explicit flags win over positionals.
	if len(pos) >= 1 {
		o.dir = pos[0]
	}
	if len(pos) >= 2 && !seenFrom {
		o.from = pos[1]
		o.fromSet = true
	}
	if len(pos) >= 3 && !seenTo {
		o.to = pos[2]
	}
	if len(pos) > 3 {
		return nil, fmt.Errorf("too many positional arguments: %s", strings.Join(pos[3:], " "))
	}
	return o, nil
}

func parseShort(a string, o *options) error {
	for _, c := range a[1:] {
		switch c {
		case 'n':
			o.dryRun = true
		case 'F':
			o.fixed = true
		case 'v':
			o.verbose = true
		case 'q':
			o.quiet = true
		case 'h':
			o.showHelp = true
		default:
			return fmt.Errorf("unknown flag -%c", c)
		}
	}
	return nil
}

// resolveLong maps a (possibly abbreviated) long-flag name to its canonical
// spelling. Exact match wins; otherwise a prefix >= minPrefix that matches
// exactly one canonical name is accepted; ambiguity is an error.
func resolveLong(name string) (string, error) {
	all := append(append([]string{}, valueFlags...), boolFlags...)
	for _, c := range all {
		if name == c {
			return c, nil
		}
	}
	if len(name) < minPrefix {
		return "", fmt.Errorf("unknown flag --%s", name)
	}
	var hits []string
	for _, c := range all {
		if strings.HasPrefix(c, name) {
			hits = append(hits, c)
		}
	}
	switch len(hits) {
	case 1:
		return hits[0], nil
	case 0:
		return "", fmt.Errorf("unknown flag --%s", name)
	default:
		sort.Strings(hits)
		return "", fmt.Errorf("ambiguous flag --%s (matches: %s)", name, strings.Join(hits, ", "))
	}
}

func isBool(canon string) bool {
	for _, b := range boolFlags {
		if b == canon {
			return true
		}
	}
	return false
}

func boolValue(canon, val string, hasVal bool) (bool, error) {
	if !hasVal {
		return true, nil
	}
	switch strings.ToLower(val) {
	case "true", "1", "yes", "on":
		return true, nil
	case "false", "0", "no", "off":
		return false, nil
	default:
		return false, fmt.Errorf("--%s: not a boolean: %q", canon, val)
	}
}

func setBool(o *options, canon string, b bool) {
	switch canon {
	case "dry-run":
		o.dryRun = b
	case "fixed":
		o.fixed = b
	case "verbose":
		o.verbose = b
	case "quiet":
		o.quiet = b
	case "version":
		o.showVersion = b
	case "help":
		o.showHelp = b
	case "examples":
		o.showExamples = b
	}
}

func setValue(o *options, canon, val string, seenFrom, seenTo *bool) error {
	switch canon {
	case "include":
		o.includes = append(o.includes, val)
	case "exclude":
		o.excludes = append(o.excludes, val)
	case "name":
		o.names = append(o.names, val)
	case "iname":
		o.inames = append(o.inames, val)
	case "from":
		o.from = val
		o.fromSet = true
		*seenFrom = true
	case "to":
		o.to = val
		*seenTo = true
	case "max-depth":
		n, err := parseInt(val)
		if err != nil {
			return fmt.Errorf("--max-depth: %w", err)
		}
		o.maxDepth = n
	}
	return nil
}

func parseInt(s string) (int, error) {
	n := 0
	if s == "" {
		return 0, fmt.Errorf("empty number")
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("not a number: %q", s)
		}
		n = n*10 + int(c-'0')
	}
	return n, nil
}
