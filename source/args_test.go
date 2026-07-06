//	Copyright © 2026 Jim Collier (ID: 1cv◂‡Vᛦ)
//	Licensed under the GNU General Public License v2.0 or later. Full text at:
//		https://spdx.org/licenses/GPL-2.0-or-later.html
//	SPDX-License-Identifier: GPL-2.0-or-later

package main

import "testing"

func mustParse(t *testing.T, argv ...string) *options {
	t.Helper()
	o, err := parseArgs(argv)
	if err != nil {
		t.Fatalf("parseArgs(%v): %v", argv, err)
	}
	return o
}

func TestPositional(t *testing.T) {
	o := mustParse(t, "/srv", "/old", "/new")
	if o.dir != "/srv" || o.from != "/old" || o.to != "/new" || !o.fromSet {
		t.Fatalf("positional fill wrong: %+v", o)
	}
	o = mustParse(t) // no args -> current dir, list-only
	if o.dir != "." || o.fromSet {
		t.Fatalf("defaults wrong: %+v", o)
	}
}

func TestFlagBeatsPositional(t *testing.T) {
	o := mustParse(t, "/srv", "/old", "/new", "--from=/x", "--to=/y")
	if o.from != "/x" || o.to != "/y" {
		t.Fatalf("explicit flag should win: %+v", o)
	}
}

func TestPrefixAbbrev(t *testing.T) {
	// Abbreviations resolve, and rules keep command-line order + kind.
	o := mustParse(t, ".", "--inc=a", "--incl=b", "--exc=c", "--re-inc=f", "--who=d", "--iwh=e")
	want := []selRule{
		{selInclude, "a"}, {selInclude, "b"}, {selExclude, "c"},
		{selReInclude, "f"}, {selWholename, "d"}, {selIWholename, "e"},
	}
	if len(o.rules) != len(want) {
		t.Fatalf("rule count wrong: %+v", o.rules)
	}
	for i, r := range want {
		if o.rules[i] != r {
			t.Fatalf("rule %d = %+v, want %+v", i, o.rules[i], r)
		}
	}
}

func TestAmbiguousAndUnknown(t *testing.T) {
	if _, err := parseArgs([]string{"--ex"}); err == nil {
		t.Fatal("expected --ex ambiguous (exclude vs examples)")
	}
	if _, err := parseArgs([]string{"--nope"}); err == nil {
		t.Fatal("expected unknown flag error")
	}
	// exa and exc are unambiguous at length 3.
	if o := mustParse(t, "--exa"); !o.showExamples {
		t.Fatal("--exa should resolve to examples")
	}
}

func TestValueForms(t *testing.T) {
	a := mustParse(t, ".", "--from=/a", "--to", "/b")
	if a.from != "/a" || a.to != "/b" {
		t.Fatalf("= and space value forms: %+v", a)
	}
}

func TestShortBundling(t *testing.T) {
	o := mustParse(t, ".", "-nv")
	if !o.dryRun || !o.verbose {
		t.Fatalf("bundled shorts: %+v", o)
	}
	o = mustParse(t, ".", "-F")
	if !o.literal {
		t.Fatal("-F should set literal")
	}
}

func TestMaxDepthAndBadInt(t *testing.T) {
	o := mustParse(t, ".", "--max-depth=3")
	if o.maxDepth != 3 {
		t.Fatalf("max-depth: %d", o.maxDepth)
	}
	if _, err := parseArgs([]string{".", "--max-depth=x"}); err == nil {
		t.Fatal("expected bad int error")
	}
}

func TestDoubleDash(t *testing.T) {
	o := mustParse(t, "--", "-weird-dir", "-f", "-t")
	if o.dir != "-weird-dir" || o.from != "-f" || o.to != "-t" {
		t.Fatalf("-- terminator: %+v", o)
	}
}

func TestTooManyPositional(t *testing.T) {
	if _, err := parseArgs([]string{"a", "b", "c", "d"}); err == nil {
		t.Fatal("expected too-many-positional error")
	}
}
