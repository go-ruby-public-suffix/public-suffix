// Copyright (c) the go-ruby-public-suffix/public-suffix authors
//
// SPDX-License-Identifier: BSD-3-Clause

package publicsuffix

import (
	"bufio"
	"os"
	"strings"
	"testing"
)

// gemDivergentVectors are the official `tests.txt` lines that the
// public_suffix gem itself does NOT satisfy, because the list it ships (the
// canonical public_suffix_list.dat we embed) carries the IDN suffix `公司.cn`
// only in its Unicode form and not as the punycode mirror `xn--55qx5d.cn`.
// The gem performs no IDN/punycode conversion (it only downcases), so a
// punycoded input is matched against the Unicode-only list and falls back to
// `cn`. Our library reproduces the gem byte-for-byte on these inputs; the map
// records the value the gem actually returns so we still assert exact parity.
var gemDivergentVectors = map[string]string{
	"xn--85x722f.xn--55qx5d.cn":     "xn--55qx5d.cn",
	"www.xn--85x722f.xn--55qx5d.cn": "xn--55qx5d.cn",
	"shishi.xn--55qx5d.cn":          "xn--55qx5d.cn",
	"xn--55qx5d.cn":                 "xn--55qx5d.cn",
}

// TestPSLCheckVectors runs the official Public Suffix List `tests.txt`
// `checkPublicSuffix(input, expected)` vectors. Each line is
// `<input> <expected-registrable-domain>` where `null` means the input has no
// registrable domain. The reference semantics: lower-case the input, then the
// registrable domain is RegistrableDomain(input). A `null` expected value must
// yield "".
//
// For the handful of punycode IDN vectors the upstream gem cannot satisfy with
// its shipped list (see gemDivergentVectors), we assert parity with the gem's
// real output rather than the raw vector, so the suite tracks gem behaviour
// exactly.
func TestPSLCheckVectors(t *testing.T) {
	f, err := os.Open("testdata/psl_tests.txt")
	if err != nil {
		t.Fatalf("open vectors: %v", err)
	}
	defer f.Close()

	total, checked := 0, 0
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		total++
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		input, expected, ok := parseVector(line)
		if !ok {
			t.Fatalf("malformed vector line: %q", line)
		}
		checked++

		if want, ok := gemDivergentVectors[input]; ok {
			expected = want
		}

		got := RegistrableDomain(input, nil)
		if got != expected {
			t.Errorf("RegistrableDomain(%q) = %q; want %q", input, got, expected)
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("scan vectors: %v", err)
	}
	if checked == 0 {
		t.Fatal("no vectors were checked")
	}
	t.Logf("checked %d PSL vectors (of %d lines)", checked, total)
}

// parseVector splits a `checkPublicSuffix` line. The expected token `null`
// becomes "". The input token `null` is itself a real vector (null input ->
// null) and is handled by the caller via normalize returning an error, so we
// keep it literally; RegistrableDomain("null", nil) returns "" since "null"
// is an unlisted single-label name with no registrable domain.
func parseVector(line string) (input, expected string, ok bool) {
	fields := strings.Fields(line)
	switch len(fields) {
	case 2:
		input = fields[0]
		expected = fields[1]
	default:
		return "", "", false
	}
	if input == "null" {
		input = ""
	}
	if expected == "null" {
		expected = ""
	}
	return input, expected, true
}
