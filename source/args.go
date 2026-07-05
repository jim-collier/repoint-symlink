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

// selKind is one selection flag's flavor. Every selection flag becomes a rule
// in one ordered pipeline (see filter.go) - their command-line order matters.
type selKind int

const (
	selInclude    selKind = iota // --include REGEX (positive)
	selExclude                   // --exclude REGEX (negative)
	selName                      // --name GLOB, basename, case-sensitive (positive)
	selIName                     // --iname GLOB, basename, case-insensitive (positive)
	selWholename                 // --wholename GLOB, full path, case-sensitive (positive)
	selIWholename                // --iwholename GLOB, full path, case-insensitive (positive)
)

// positive reports whether the rule adds/keeps (everything but --exclude).
func (k selKind) positive() bool { return k != selExclude }

// isGlob reports whether the rule is a find-style glob (vs a regex).
func (k selKind) isGlob() bool { return k >= selName }

func (k selKind) flag() string {
	switch k {
	case selInclude:
		return "include"
	case selExclude:
		return "exclude"
	case selName:
		return "name"
	case selIName:
		return "iname"
	case selWholename:
		return "wholename"
	default:
		return "iwholename"
	}
}

// selRule is one selection flag as given, kept in command-line order.
type selRule struct {
	kind selKind
	pat  string
}

// options is the parsed command line.
type options struct {
	dir      string    // start folder (positional 1, default ".")
	from     string    // regex (or literal with -F); positional 2
	to       string    // replacement template; positional 3
	fromSet  bool      // was --from / positional 2 given? (enables edit mode)
	rules    []selRule // --inc/--exc/--[i]name/--[i]wholename, in order
	maxDepth int       // --max-depth, -1 = unlimited
	fixed    bool      // -F: treat --from as a literal string
	dryRun   bool      // -n: preview, do not write
	verbose  bool      // -v
	quiet    bool      // -q
	// terminal actions
	showVersion  bool
	showHelp     bool
	showExamples bool
}

// long flags that take a value, and the min-unambiguous-prefix minimum length.
// Prefix abbreviation is allowed down to 3 chars (so --inc, --exc resolve),
// but an exact spelling always wins regardless of length (so --to works).
const minPrefix = 3

var valueFlags = []string{"include", "exclude", "name", "iname", "wholename", "iwholename", "from", "to", "max-depth"}
var boolFlags = []string{"dry-run", "fixed", "verbose", "quiet", "version", "help", "examples"}

func parseArgs(argv []string) (*options, error) {
	opts := &options{dir: ".", maxDepth: -1}
	var pos []string
	seenFrom, seenTo := false, false

	i := 0
	for i < len(argv) {
		arg := argv[i]
		i++

		if arg == "--" { // rest are positional
			pos = append(pos, argv[i:]...)
			break
		}

		// Short single-dash bool flags (bundling allowed: -nv).
		if len(arg) >= 2 && arg[0] == '-' && arg[1] != '-' {
			if err := parseShort(arg, opts); err != nil {
				return nil, err
			}
			continue
		}

		// Long flags.
		if strings.HasPrefix(arg, "--") {
			name, val, hasVal := strings.Cut(arg[2:], "=")
			canon, err := resolveLong(name)
			if err != nil {
				return nil, err
			}
			if isBool(canon) {
				enabled, err := boolValue(canon, val, hasVal)
				if err != nil {
					return nil, err
				}
				setBool(opts, canon, enabled)
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
			if err := setValue(opts, canon, val, &seenFrom, &seenTo); err != nil {
				return nil, err
			}
			continue
		}

		pos = append(pos, arg)
	}

	// Positional fill: dir, from, to. Explicit flags win over positionals.
	if len(pos) >= 1 {
		opts.dir = pos[0]
	}
	if len(pos) >= 2 && !seenFrom {
		opts.from = pos[1]
		opts.fromSet = true
	}
	if len(pos) >= 3 && !seenTo {
		opts.to = pos[2]
	}
	if len(pos) > 3 {
		return nil, fmt.Errorf("too many positional arguments: %s", strings.Join(pos[3:], " "))
	}
	return opts, nil
}

func parseShort(arg string, opts *options) error {
	for _, ch := range arg[1:] {
		switch ch {
		case 'n':
			opts.dryRun = true
		case 'F':
			opts.fixed = true
		case 'v':
			opts.verbose = true
		case 'q':
			opts.quiet = true
		case 'h':
			opts.showHelp = true
		default:
			return fmt.Errorf("unknown flag -%c", ch)
		}
	}
	return nil
}

// resolveLong maps a (possibly abbreviated) long-flag name to its canonical
// spelling. Exact match wins; otherwise a prefix >= minPrefix that matches
// exactly one canonical name is accepted; ambiguity is an error.
func resolveLong(name string) (string, error) {
	allFlags := append(append([]string{}, valueFlags...), boolFlags...)
	for _, flag := range allFlags {
		if name == flag {
			return flag, nil
		}
	}
	if len(name) < minPrefix {
		return "", fmt.Errorf("unknown flag --%s", name)
	}
	var matches []string
	for _, flag := range allFlags {
		if strings.HasPrefix(flag, name) {
			matches = append(matches, flag)
		}
	}
	switch len(matches) {
	case 1:
		return matches[0], nil
	case 0:
		return "", fmt.Errorf("unknown flag --%s", name)
	default:
		sort.Strings(matches)
		return "", fmt.Errorf("ambiguous flag --%s (matches: %s)", name, strings.Join(matches, ", "))
	}
}

func isBool(canon string) bool {
	for _, flag := range boolFlags {
		if flag == canon {
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

func setBool(opts *options, canon string, enabled bool) {
	switch canon {
	case "dry-run":
		opts.dryRun = enabled
	case "fixed":
		opts.fixed = enabled
	case "verbose":
		opts.verbose = enabled
	case "quiet":
		opts.quiet = enabled
	case "version":
		opts.showVersion = enabled
	case "help":
		opts.showHelp = enabled
	case "examples":
		opts.showExamples = enabled
	}
}

func setValue(opts *options, canon, val string, seenFrom, seenTo *bool) error {
	switch canon {
	case "include":
		opts.rules = append(opts.rules, selRule{selInclude, val})
	case "exclude":
		opts.rules = append(opts.rules, selRule{selExclude, val})
	case "name":
		opts.rules = append(opts.rules, selRule{selName, val})
	case "iname":
		opts.rules = append(opts.rules, selRule{selIName, val})
	case "wholename":
		opts.rules = append(opts.rules, selRule{selWholename, val})
	case "iwholename":
		opts.rules = append(opts.rules, selRule{selIWholename, val})
	case "from":
		opts.from = val
		opts.fromSet = true
		*seenFrom = true
	case "to":
		opts.to = val
		*seenTo = true
	case "max-depth":
		depth, err := parseInt(val)
		if err != nil {
			return fmt.Errorf("--max-depth: %w", err)
		}
		opts.maxDepth = depth
	}
	return nil
}

func parseInt(s string) (int, error) {
	n := 0
	if s == "" {
		return 0, fmt.Errorf("empty number")
	}
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return 0, fmt.Errorf("not a number: %q", s)
		}
		n = n*10 + int(ch-'0')
	}
	return n, nil
}
