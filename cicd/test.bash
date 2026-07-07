#!/usr/bin/env bash

#  shellcheck disable=2001
#  shellcheck disable=2016
#  shellcheck disable=2086
#  shellcheck disable=2155
#  shellcheck disable=2181

##	Purpose:
##		- Test harness for repoint-symlink. Runs the Go unit tests, then drives the
##		  built binary against scratch symlink trees and asserts the results.
##		- Binary under test: $CICDTEST_EXE if set (the cicd engine sets it to the
##		  staged build); otherwise 'make -C source local' is built and used.
##		- Set CICDTEST_DO_LONGTEST=1 for the larger, exhaustive tree.
##	History: At bottom of script.

##	Copyright © 2026 Jim Collier (ID: 1cv◂‡Vᛦ)
##	Licensed under the GNU General Public License v2.0 or later. Full text at:
##		https://spdx.org/licenses/GPL-2.0-or-later.html
##	SPDX-License-Identifier: GPL-2.0-or-later


set -Eeuo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
root="$(cd "${here}/.." && pwd)"
srcdir="${root}/source"

## Output helpers.
b=$'\e[1m'; grn=$'\e[32m'; red=$'\e[31m'; rst=$'\e[0m'
section(){ echo; printf '%s[ %s ]%s\n' "${b}" "$*" "${rst}"; }
pass(){ printf '  %sok%s   %s\n' "${grn}" "${rst}" "$*"; PASSED=$((PASSED+1)); }
fail(){ printf '  %sFAIL%s %s\n' "${red}" "${rst}" "$*" >&2; FAILED=$((FAILED+1)); }
die(){  printf '\n%sTEST ABORTED: %s%s\n' "${red}" "$*" "${rst}" >&2; exit 1; }

PASSED=0; FAILED=0
LONG="${CICDTEST_DO_LONGTEST:-0}"

## Resolve the binary under test.
EXE="${CICDTEST_EXE:-}"
if [[ -z "${EXE}" ]]; then
	section "Building binary under test"
	make -C "${srcdir}" local >/dev/null || die "build failed"
	EXE="${srcdir}/repoint-symlink"
fi
[[ -x "${EXE}" ]] || die "binary not executable: ${EXE}"

## Go unit tests first - they cover the arg parser, filters, transform logic,
## and the security properties (link-follow safety, cycle safety, bounded match).
section "Go unit tests"
if ( cd "${srcdir}" && go test ./... ); then
	pass "go test ./..."
else
	fail "go test ./..."
fi

## Fuzz each target for a short burst (longer under CICDTEST_DO_LONGTEST). These
## hammer the arg parser, the glob translator, the regex engine, the selection
## pipeline, and the transform with random input, looking for panics or hangs.
## Targets are listed as 'package:Target' since the glob/regex/selection fuzzers
## live in the reusable filter package.
section "Go fuzz (short burst per target)"
fuzztime="3s"; ((LONG)) && fuzztime="20s"
for pair in .:FuzzParseArgs .:FuzzTransform ./filter:FuzzGlobToRegex ./filter:FuzzCompileAndMatch ./filter:FuzzSelects; do
	pkg="${pair%%:*}"; fz="${pair#*:}"
	if ( cd "${srcdir}" && go test -run='^$' -fuzz="^${fz}\$" -fuzztime="${fuzztime}" "${pkg}" >/dev/null 2>&1 ); then
		pass "fuzz ${fz} (${fuzztime})"
	else
		fail "fuzz ${fz}"
	fi
done

## Assertions.
assert_target(){ ## link expected msg
	local got; got="$(readlink "$1" 2>/dev/null || true)"
	if [[ "${got}" == "$2" ]]; then pass "$3"; else fail "$3 (want '$2', got '${got}')"; fi
}
assert_grep(){ ## haystack needle msg
	if grep -qF -- "$2" <<<"$1"; then pass "$3"; else fail "$3 (missing '$2')"; fi
}
assert_rc(){ ## actual expected msg
	if [[ "$1" == "$2" ]]; then pass "$3"; else fail "$3 (want rc $2, got $1)"; fi
}

## Fresh scratch tree per scenario.
mktree(){
	local t; t="$(mktemp -d)"
	mkdir -p "${t}/a/b" "${t}/backup"
	ln -s /mnt/old/data  "${t}/a/one.conf"
	ln -s /mnt/old/logs  "${t}/a/b/two.conf"
	ln -s /mnt/keep/x    "${t}/a/three.txt"
	ln -s /opt/app-1.2.3 "${t}/a/app"
	ln -s /mnt/old/z     "${t}/backup/old.conf"
	echo "${t}"
}

section "Integration: dry-run writes nothing"
T="$(mktree)"
out="$("${EXE}" "${T}" --from='/mnt/old' --to='/mnt/new' -n)"
assert_grep "${out}" "would repoint" "dry-run announces changes"
assert_target "${T}/a/one.conf" "/mnt/old/data" "dry-run left target untouched"
rm -rf "${T}"

section "Integration: apply repoints matching links"
T="$(mktree)"
"${EXE}" "${T}" --from='/mnt/old' --to='/mnt/new' >/dev/null
assert_target "${T}/a/one.conf"    "/mnt/new/data" "one.conf repointed"
assert_target "${T}/a/b/two.conf"  "/mnt/new/logs" "nested two.conf repointed"
assert_target "${T}/a/three.txt"   "/mnt/keep/x"   "non-matching left alone"
rm -rf "${T}"

section "Integration: filters (iname + exclude)"
T="$(mktree)"
"${EXE}" "${T}" --iname='*.conf' --exc='/backup/' --from='/mnt/old' --to='/mnt/new' >/dev/null
assert_target "${T}/a/one.conf"      "/mnt/new/data" "*.conf matched"
assert_target "${T}/backup/old.conf" "/mnt/old/z"    "excluded backup untouched"
rm -rf "${T}"

section "Integration: ordered filters - re-include after exclude expands"
T="$(mktree)"
ln -s /mnt/old/w "${T}/a/b/other.conf"   # a second link under a/b to prove exclusion
out="$("${EXE}" "${T}" --inc='/a/' --exc='/a/b/' --re-inc='/a/b/two')"
assert_grep "${out}"   "/a/one.conf"   "under /a/ but not /a/b/ stays in"
assert_grep "${out}"   "/a/b/two.conf" "re-include brings two.conf back"
if grep -qF '/a/b/other.conf' <<<"${out}"; then fail "excluded other.conf wrongly back"; else pass "other.conf stays excluded"; fi
if grep -qF '/backup/old.conf' <<<"${out}"; then fail "backup wrongly included"; else pass "backup never included"; fi
rm -rf "${T}"

section "Integration: ordered filters - plain include narrows, does not re-admit"
T="$(mktree)"
ln -s /mnt/old/w "${T}/a/b/other.conf"
out="$("${EXE}" "${T}" --inc='/a/' --exc='/a/b/' --inc='/a/b/two')"
if grep -qF '/a/b/two.conf' <<<"${out}"; then fail "plain include wrongly re-admitted excluded link"; else pass "plain include cannot undo an --exclude"; fi
rm -rf "${T}"

section "Integration: ordered filters - two includes narrow (AND)"
T="$(mktree)"
out="$("${EXE}" "${T}" --inc='/a/' --inc='two')"
assert_grep "${out}" "/a/b/two.conf" "matches both includes"
if grep -qF '/a/one.conf' <<<"${out}"; then fail "one.conf should be narrowed out"; else pass "consecutive includes AND, not OR"; fi
rm -rf "${T}"

section "Integration: wholename spans '/', name is basename-only"
T="$(mktree)"
out="$("${EXE}" "${T}" --wholename='*/a/b/*')"
assert_grep "${out}" "/a/b/two.conf" "wholename '*' spans '/'"
if grep -qF '/a/one.conf' <<<"${out}"; then fail "wholename matched wrong path"; else pass "wholename anchored to the subtree"; fi
out="$("${EXE}" "${T}" --name='*/a/b/*')"
if grep -qF 'two.conf' <<<"${out}"; then fail "name should not see '/' in basename"; else pass "name matches basename only"; fi
rm -rf "${T}"

section "Integration: regex capture template"
T="$(mktree)"
"${EXE}" "${T}" --name='app' --from='/opt/app-(\d+\.\d+\.\d+)' --to='/opt/app/$1' >/dev/null
assert_target "${T}/a/app" "/opt/app/1.2.3" "capture group rewritten"
rm -rf "${T}"

section "Integration: literal (-F) replace"
T="$(mktree)"
ln -s '/data/a.b.c' "${T}/lit"
"${EXE}" "${T}" -F --name='lit' --from='a.b.c' --to='X' >/dev/null
assert_target "${T}/lit" "/data/X" "literal dots not treated as regex"
rm -rf "${T}"

section "Integration: max-depth limits recursion"
T="$(mktree)"
out="$("${EXE}" "${T}/a" --max-depth=1)"
assert_grep "${out}" "one.conf" "depth-1 child listed"
if grep -qF 'two.conf' <<<"${out}"; then fail "depth-2 wrongly listed"; else pass "depth-2 pruned"; fi
rm -rf "${T}"

section "Integration: idempotency"
T="$(mktree)"
"${EXE}" "${T}" --from='/mnt/old' --to='/mnt/new' >/dev/null
out="$("${EXE}" "${T}" --from='/mnt/old' --to='/mnt/new')"
assert_grep "${out}" "Repointed 0 of" "second run changes nothing"
rm -rf "${T}"

section "Integration: exit codes"
T="$(mktree)"
set +e
"${EXE}" "${T}" --from='/mnt/old' --to='/mnt/new' >/dev/null 2>&1; rc=$?
set -e
assert_rc "${rc}" "0" "success exits 0"
set +e
"${EXE}" "/no/such/dir" >/dev/null 2>&1; rc=$?
set -e
assert_rc "${rc}" "1" "missing start dir exits 1"
set +e
"${EXE}" "${T}" --bogusflag >/dev/null 2>&1; rc=$?
set -e
assert_rc "${rc}" "2" "bad flag exits 2"
rm -rf "${T}"

section "Integration: no args prints help (exit 0)"
set +e
out="$("${EXE}" 2>&1)"; rc=$?
set -e
assert_rc "${rc}" "0" "no args exits 0"
assert_grep "${out}" "Usage:" "no args prints help"

section "Security: rewrite replaces the link, does not write through it"
T="$(mktree)"
printf 'SECRET\n' > "${T}/realfile"
ln -s "${T}/realfile" "${T}/a/pointer"
"${EXE}" "${T}/a" --name='pointer' -F --from="${T}/realfile" --to='/mnt/new/z' >/dev/null
assert_target "${T}/a/pointer" "/mnt/new/z" "pointer repointed"
if [[ "$(cat "${T}/realfile")" == "SECRET" ]]; then pass "pointed-at file left untouched"; else fail "pointed-at file was modified"; fi
rm -rf "${T}"

section "Security: symlink cycle does not hang the walk"
T="$(mktemp -d)"
ln -s "${T}"       "${T}/self"   # directory self-loop
ln -s "${T}/b"     "${T}/a"      # mutual file links
ln -s "${T}/a"     "${T}/b"
set +e
timeout 20 "${EXE}" "${T}" >/dev/null 2>&1; rc=$?
set -e
if ((rc == 0)); then pass "cyclic tree walked and terminated"; else fail "cyclic tree hung or errored (rc ${rc})"; fi
rm -rf "${T}"

section "Integration: --no-cross-device is a no-op on one filesystem"
T="$(mktree)"; T2="$(mktree)"
plain="$("${EXE}" "${T}"  --from='/mnt/old' --to='/mnt/new' -n | grep -c 'would repoint')"
xdev="$("${EXE}" "${T2}" --from='/mnt/old' --to='/mnt/new' -n --no-cross-device | grep -c 'would repoint')"
if [[ "${plain}" == "${xdev}" ]]; then pass "same match count with/without --no-cross-device"; else fail "counts differ (plain=${plain} xdev=${xdev})"; fi
rm -rf "${T}" "${T2}"

section "Integration: --inc-target / --exc-target select by current target"
T="$(mktree)"
out="$("${EXE}" "${T}" --inc-target='^/mnt/old/')"
assert_grep "${out}" "one.conf" "inc-target keeps a /mnt/old link"
if grep -qF 'three.txt' <<<"${out}"; then fail "inc-target should drop the /mnt/keep link"; else pass "inc-target narrows out /mnt/keep"; fi
out="$("${EXE}" "${T}" --inc-target='^/mnt/old/' --exc-target='/z$')"
if grep -qF 'backup/old.conf' <<<"${out}"; then fail "exc-target should drop /mnt/old/z"; else pass "exc-target subtracts by target"; fi
out="$("${EXE}" "${T}" --inc='/a/')"
assert_grep "${out}" "one.conf" "--inc alias still resolves after adding --inc-target"
rm -rf "${T}"

section "Integration: --follow-links reaches behind a dir symlink, loop-safe"
T="$(mktemp -d)"; EXT="$(mktemp -d)"
ln -s /mnt/old/deep "${EXT}/inner.conf"   # link only reachable through the dir symlink
ln -s "${EXT}"      "${T}/dirlink"
out_no="$("${EXE}" "${T}")"
if grep -qF 'inner.conf' <<<"${out_no}"; then fail "should not find a link behind an unfollowed dir symlink"; else pass "default walk stops at the dir symlink"; fi
out_yes="$("${EXE}" "${T}" -L)"
assert_grep "${out_yes}" "inner.conf" "--follow-links reaches the link behind the dir symlink"
rm -rf "${T}" "${EXT}"
T="$(mktemp -d)"; mkdir -p "${T}/sub"
ln -s "${T}"       "${T}/sub/loop"   # cycle back to root
ln -s /mnt/old/x   "${T}/sub/l"
set +e; timeout 20 "${EXE}" "${T}" -L >/dev/null 2>&1; rc=$?; set -e
assert_rc "${rc}" "0" "--follow-links terminates on a dir-symlink cycle"
rm -rf "${T}"

section "Integration: --print0 emits NUL-separated paths, no summary"
T="$(mktree)"
n="$("${EXE}" "${T}" --inc-target='^/mnt/old/' -0 | tr -cd '\0' | wc -c)"
assert_rc "${n}" "3" "one NUL record per /mnt/old match (3)"
if "${EXE}" "${T}" -0 | grep -qa 'matching'; then fail "print0 leaked the summary line"; else pass "print0 suppresses the summary"; fi
rm -rf "${T}"

section "Integration: --renormal-relative / --renormal-absolute rewrite target spelling"
T="$(mktemp -d)"; mkdir -p "${T}/app"
ln -s "${T}/app/data" "${T}/app/abs"
"${EXE}" "${T}/app" --name='abs' --renormal-relative >/dev/null
assert_target "${T}/app/abs" "data" "absolute in-tree target made relative"
ln -s ../shared/x "${T}/app/rel"
"${EXE}" "${T}/app" --name='rel' --renormal-absolute >/dev/null
assert_target "${T}/app/rel" "${T}/shared/x" "relative target made absolute"
rm -rf "${T}"

section "Integration: --confirm applies on yes, aborts on no"
T="$(mktemp -d)"; ln -s /mnt/old/z "${T}/l"
printf 'y\n' | "${EXE}" "${T}" --from='/mnt/old' --to='/mnt/new' --confirm >/dev/null
assert_target "${T}/l" "/mnt/new/z" "confirm + yes applied the change"
T2="$(mktemp -d)"; ln -s /mnt/old/z "${T2}/l"
printf 'n\n' | "${EXE}" "${T2}" --from='/mnt/old' --to='/mnt/new' --confirm >/dev/null
assert_target "${T2}/l" "/mnt/old/z" "confirm + no wrote nothing"
rm -rf "${T}" "${T2}"

section "Integration: combined filters + target-filter + renormal (dry-run)"
T="$(mktree)"
out="$("${EXE}" "${T}" --iname='*.conf' --exc='/backup/' --inc-target='^/mnt/old/' --from='/mnt/old' --to='/mnt/new' --renormal-absolute -n)"
assert_grep "${out}" "would repoint" "combined run previews changes"
assert_grep "${out}" "one.conf"      "combined run keeps the expected link"
if grep -qF 'backup/old.conf' <<<"${out}"; then fail "excluded link leaked into the combined run"; else pass "combined run honors --exclude"; fi
rm -rf "${T}"

section "Integration: invalid flag combinations exit 2"
T="$(mktree)"
chk(){ set +e; "${EXE}" "$@" >/dev/null 2>&1; local rc=$?; set -e; echo "${rc}"; }
assert_rc "$(chk "${T}" --renormal-relative --renormal-absolute)" "2" "conflicting renormal directions rejected"
assert_rc "$(chk "${T}" --confirm --print0)"                      "2" "--confirm with --print0 rejected"
assert_rc "$(chk "${T}" --from='(' --to='y')"                     "2" "bad --from regex rejected"
assert_rc "$(chk "${T}" --inc-target='(')"                        "2" "bad --inc-target regex rejected"
assert_rc "$(chk "${T}" --renormal)"                              "2" "ambiguous --renormal rejected"
rm -rf "${T}"

if ((LONG)); then
	section "Long: large tree round-trip"
	T="$(mktemp -d)"
	for i in $(seq 1 500); do
		d="${T}/d$((i % 20))"; mkdir -p "${d}"
		ln -s "/mnt/old/item-${i}" "${d}/link-${i}"
	done
	"${EXE}" "${T}" --from='/mnt/old' --to='/mnt/new' >/dev/null
	miss=0
	for l in $(find "${T}" -type l); do
		[[ "$(readlink "${l}")" == /mnt/new/* ]] || miss=$((miss+1))
	done
	if ((miss == 0)); then pass "all 500 links repointed"; else fail "${miss} links not repointed"; fi
	rm -rf "${T}"
fi

section "Summary"
printf '  passed: %s%d%s   failed: %s%d%s\n' "${grn}" "${PASSED}" "${rst}" "$( ((FAILED)) && echo "${red}" || echo "${grn}")" "${FAILED}" "${rst}"
((FAILED == 0)) || exit 1


##	History:
##		- 2026-07-04 JC: Created. Go unit tests + integration scenarios over scratch symlink trees.
##		- 2026-07-06 JC: Added a fuzz burst per target and end-to-end security scenarios.
##		- 2026-07-06 JC: Broad feature/combination coverage: device, target filters, follow-links,
##		  print0, renormal, confirm, a combined run, and invalid-combination exit codes.
##		- 2026-07-06 JC: no-args-prints-help check.
