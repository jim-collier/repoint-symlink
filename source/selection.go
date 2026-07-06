//	Copyright © 2026 Jim Collier (ID: 1cv◂‡Vᛦ)
//	Licensed under the GNU General Public License v2.0 or later. Full text at:
//		https://spdx.org/licenses/GPL-2.0-or-later.html
//	SPDX-License-Identifier: GPL-2.0-or-later

package main

import "github.com/jim-collier/repoint-symlink/filter"

// compileFilters maps the parsed selection flags onto the reusable filter
// engine. The pipeline semantics live in package filter; here we only translate
// each flag's kind into a rule spec (which op, regex vs glob, basename, fold).
func compileFilters(opts *options) (*filter.Set, error) {
	specs := make([]filter.Spec, 0, len(opts.rules))
	for _, sr := range opts.rules {
		specs = append(specs, specFor(sr))
	}
	return filter.Compile(specs)
}

func specFor(sr selRule) filter.Spec {
	spec := filter.Spec{Op: filter.Narrow, Pattern: sr.pat, Label: "--" + sr.kind.flag()}
	switch sr.kind {
	case selExclude:
		spec.Op = filter.Subtract
	case selReInclude:
		spec.Op = filter.Readd
	}
	if sr.kind.isGlob() {
		spec.Glob = true
		spec.Base = sr.kind == selName || sr.kind == selIName
		spec.Fold = sr.kind == selIName || sr.kind == selIWholename
	}
	return spec
}
