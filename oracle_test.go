// Copyright (c) the go-ruby-public-suffix/public-suffix authors
//
// SPDX-License-Identifier: BSD-3-Clause

package publicsuffix

import (
	"os/exec"
	"strings"
	"testing"
)

// rubyBin locates a usable `ruby` with the public_suffix gem once. The oracle
// tests skip themselves when ruby (or the gem) is absent — the qemu cross-arch
// lanes and the Windows lane — so the deterministic suite alone drives the
// 100% gate there.
func rubyBin(t *testing.T) string {
	t.Helper()
	path, err := exec.LookPath("ruby")
	if err != nil {
		t.Skip("ruby not on PATH; skipping public_suffix gem oracle")
	}
	// Confirm the gem is installed; skip cleanly if not.
	if err := exec.Command(path, "-e", "require 'public_suffix'").Run(); err != nil {
		t.Skip("public_suffix gem not installed; skipping oracle")
	}
	return path
}

// rubyDomain returns `PublicSuffix.domain(name)` from the real gem ("" for
// nil). $stdout.binmode keeps Windows text-mode from mangling the bytes (the
// go-ruby-erb lesson); the script prints exactly the domain followed by "\n".
func rubyDomain(t *testing.T, bin, name string) string {
	t.Helper()
	script := `$stdout.binmode
require 'public_suffix'
name = ARGV[0]
d = (PublicSuffix.domain(name) rescue nil)
print(d.nil? ? "" : d)`
	out, err := exec.Command(bin, "-e", script, "--", name).Output()
	if err != nil {
		t.Fatalf("ruby error for %q: %v", name, err)
	}
	return string(out)
}

// rubyValid returns `PublicSuffix.valid?(name)` from the real gem.
func rubyValid(t *testing.T, bin, name string) bool {
	t.Helper()
	script := `$stdout.binmode
require 'public_suffix'
print(PublicSuffix.valid?(ARGV[0]) ? "true" : "false")`
	out, err := exec.Command(bin, "-e", script, "--", name).Output()
	if err != nil {
		t.Fatalf("ruby error for %q: %v", name, err)
	}
	return strings.TrimSpace(string(out)) == "true"
}

// oracleNames is a representative spread across every rule type and edge.
var oracleNames = []string{
	"google.com",
	"www.google.com",
	"a.b.google.com",
	"google.com.",
	"www.google.com.",
	"example.uk.com",
	"a.example.uk.com",
	"b.c.mm",       // wildcard
	"a.b.c.mm",     // wildcard + sub
	"city.kobe.jp", // exception
	"www.city.kobe.jp",
	"test.ck",              // wildcard *.ck
	"www.ck",               // exception !www.ck
	"foo.blogspot.com",     // private section
	"example.tldnotlisted", // default * rule
	"WwW.Example.COM",      // mixed case
	"食狮.com.cn",            // unicode idn
	"食狮.公司.cn",
	"com", // bare suffix -> not allowed
	"uk.com",
	"",                      // blank
	".com",                  // leading dot
	"http://www.google.com", // scheme
	"x.yz",                  // unlisted 2-label
}

// TestOracleDomain checks RegistrableDomain against the real gem for every
// oracle name.
func TestOracleDomain(t *testing.T) {
	bin := rubyBin(t)
	for _, name := range oracleNames {
		want := rubyDomain(t, bin, name)
		got := RegistrableDomain(name, nil)
		if got != want {
			t.Errorf("RegistrableDomain(%q) = %q; gem = %q", name, got, want)
		}
	}
}

// TestOracleValid checks Valid against the real gem for every oracle name.
func TestOracleValid(t *testing.T) {
	bin := rubyBin(t)
	for _, name := range oracleNames {
		want := rubyValid(t, bin, name)
		got := Valid(name, nil)
		if got != want {
			t.Errorf("Valid(%q) = %v; gem = %v", name, got, want)
		}
	}
}

// TestOracleDecomposition checks the full tld/sld/trd split against the gem's
// Domain object for the names that parse successfully.
func TestOracleDecomposition(t *testing.T) {
	bin := rubyBin(t)
	script := `$stdout.binmode
require 'public_suffix'
d = PublicSuffix.parse(ARGV[0])
print [d.tld, d.sld, d.trd].map { |x| x.nil? ? "" : x }.join("\t")`

	for _, name := range oracleNames {
		// Only compare names the gem parses without raising.
		if RegistrableDomain(name, nil) == "" && !Valid(name, nil) {
			continue
		}
		out, err := exec.Command(bin, "-e", script, "--", name).Output()
		if err != nil {
			// Gem raised (e.g. bare suffix); skip — error parity is covered
			// by the deterministic suite.
			continue
		}
		fields := strings.Split(string(out), "\t")
		if len(fields) != 3 {
			t.Fatalf("unexpected gem output for %q: %q", name, out)
		}
		d, err := Parse(name, nil)
		if err != nil {
			t.Errorf("Parse(%q) errored but gem parsed: %v", name, err)
			continue
		}
		if d.TLD != fields[0] || d.SLD != fields[1] || d.TRD != fields[2] {
			t.Errorf("Parse(%q) = tld=%q sld=%q trd=%q; gem = tld=%q sld=%q trd=%q",
				name, d.TLD, d.SLD, d.TRD, fields[0], fields[1], fields[2])
		}
	}
}
