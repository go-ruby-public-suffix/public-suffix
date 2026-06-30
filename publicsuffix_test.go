// Copyright (c) the go-ruby-public-suffix/public-suffix authors
//
// SPDX-License-Identifier: BSD-3-Clause

package publicsuffix

import (
	"errors"
	"strings"
	"testing"
)

// --- Parse: decomposition across rule types --------------------------------

func TestParseDecomposition(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		tld, sld, trd string
		hasTRD        bool
		domain        string
		subdomain     string
	}{
		// Normal rule.
		{"plain domain", "google.com", "com", "google", "", false, "google.com", ""},
		{"subdomain", "www.google.com", "com", "google", "www", true, "google.com", "www.google.com"},
		{"deep subdomain", "a.b.google.com", "com", "google", "a.b", true, "google.com", "a.b.google.com"},
		// Trailing dot is stripped (one dot only).
		{"fqdn trailing dot", "google.com.", "com", "google", "", false, "google.com", ""},
		{"fqdn subdomain dot", "www.google.com.", "com", "google", "www", true, "google.com", "www.google.com"},
		// Multi-level normal rule.
		{"two-level tld", "example.uk.com", "uk.com", "example", "", false, "example.uk.com", ""},
		{"two-level tld sub", "a.example.uk.com", "uk.com", "example", "a", true, "example.uk.com", "a.example.uk.com"},
		// Wildcard rule (*.mm): b.c.mm -> tld=c.mm.
		{"wildcard", "b.c.mm", "c.mm", "b", "", false, "b.c.mm", ""},
		{"wildcard sub", "a.b.c.mm", "c.mm", "b", "a", true, "b.c.mm", "a.b.c.mm"},
		// Exception rule (!city.kobe.jp): city.kobe.jp -> tld=kobe.jp.
		{"exception", "city.kobe.jp", "kobe.jp", "city", "", false, "city.kobe.jp", ""},
		{"exception sub", "www.city.kobe.jp", "kobe.jp", "city", "www", true, "city.kobe.jp", "www.city.kobe.jp"},
		// Default "*" rule for unlisted TLD.
		{"unlisted tld", "example.tldnotlisted", "tldnotlisted", "example", "", false, "example.tldnotlisted", ""},
		{"unlisted tld sub", "www.example.tldnotlisted", "tldnotlisted", "example", "www", true, "example.tldnotlisted", "www.example.tldnotlisted"},
		// Unicode IDN labels stay as-is.
		{"unicode idn", "食狮.com.cn", "com.cn", "食狮", "", false, "食狮.com.cn", ""},
		// Mixed case is downcased.
		{"mixed case", "WwW.Example.COM", "com", "example", "www", true, "example.com", "www.example.com"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d, err := Parse(tc.input, nil)
			if err != nil {
				t.Fatalf("Parse(%q) error: %v", tc.input, err)
			}
			if d.TLD != tc.tld {
				t.Errorf("TLD = %q; want %q", d.TLD, tc.tld)
			}
			if d.SLD != tc.sld {
				t.Errorf("SLD = %q; want %q", d.SLD, tc.sld)
			}
			if d.TRD != tc.trd {
				t.Errorf("TRD = %q; want %q", d.TRD, tc.trd)
			}
			if d.HasTRD() != tc.hasTRD {
				t.Errorf("HasTRD() = %v; want %v", d.HasTRD(), tc.hasTRD)
			}
			if got := d.DomainName(); got != tc.domain {
				t.Errorf("DomainName() = %q; want %q", got, tc.domain)
			}
			if got := d.Subdomain(); got != tc.subdomain {
				t.Errorf("Subdomain() = %q; want %q", got, tc.subdomain)
			}
		})
	}
}

// --- Parse: error branches -------------------------------------------------

func TestParseErrors(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		opts   *Options
		kind   error
		substr string
	}{
		{"blank", "", nil, ErrDomainInvalid, "Name is blank"},
		{"whitespace only", "   ", nil, ErrDomainInvalid, "Name is blank"},
		{"leading dot", ".com", nil, ErrDomainInvalid, "Name starts with a dot"},
		{"scheme", "http://www.google.com", nil, ErrDomainInvalid, "not expected to contain a scheme"},
		// A bare public suffix is not allowed (rule matches, decompose nil).
		{"bare suffix com", "com", nil, ErrDomainNotAllowed, "not allowed according to Registry policy"},
		{"bare suffix multi", "uk.com", nil, ErrDomainNotAllowed, "not allowed according to Registry policy"},
		// Wildcard suffix itself (e.g. "c.mm") is not allowed.
		{"bare wildcard suffix", "c.mm", nil, ErrDomainNotAllowed, "not allowed according to Registry policy"},
		// With NoDefaultRule an unlisted TLD becomes invalid.
		{"unlisted strict", "example.tldnotlisted", &Options{NoDefaultRule: true}, ErrDomainInvalid, "is not a valid domain"},
		// Single unlisted label with strict checking: no rule at all.
		{"single label strict", "tldnotlisted", &Options{NoDefaultRule: true}, ErrDomainInvalid, "is not a valid domain"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d, err := Parse(tc.input, tc.opts)
			if err == nil {
				t.Fatalf("Parse(%q) = %v; want error", tc.input, d)
			}
			if !errors.Is(err, tc.kind) {
				t.Errorf("Parse(%q) error kind = %v; want %v", tc.input, err, tc.kind)
			}
			if !strings.Contains(err.Error(), tc.substr) {
				t.Errorf("Parse(%q) error %q; want substring %q", tc.input, err.Error(), tc.substr)
			}
		})
	}
}

// DomainNotAllowed must also satisfy errors.Is(ErrDomainInvalid) — the gem's
// subclass relationship.
func TestNotAllowedIsInvalid(t *testing.T) {
	_, err := Parse("com", nil)
	if !errors.Is(err, ErrDomainNotAllowed) {
		t.Fatalf("want not-allowed, got %v", err)
	}
	if !errors.Is(err, ErrDomainInvalid) {
		t.Fatalf("not-allowed must also be invalid (gem subclass); got %v", err)
	}
}

// --- RegistrableDomain -----------------------------------------------------

func TestRegistrableDomain(t *testing.T) {
	tests := []struct {
		input string
		opts  *Options
		want  string
	}{
		{"google.com", nil, "google.com"},
		{"www.google.com", nil, "google.com"},
		{"google.com.", nil, "google.com"},
		{"example.tldnotlisted", nil, "example.tldnotlisted"},
		// Errors collapse to "".
		{"com", nil, ""},
		{"", nil, ""},
		{".com", nil, ""},
		{"http://x.com", nil, ""},
		{"example.tldnotlisted", &Options{NoDefaultRule: true}, ""},
	}
	for _, tc := range tests {
		if got := RegistrableDomain(tc.input, tc.opts); got != tc.want {
			t.Errorf("RegistrableDomain(%q, %+v) = %q; want %q", tc.input, tc.opts, got, tc.want)
		}
	}
}

// --- Valid -----------------------------------------------------------------

func TestValid(t *testing.T) {
	tests := []struct {
		input string
		opts  *Options
		want  bool
	}{
		{"example.com", nil, true},
		{"www.example.com", nil, true},
		{"example.tldnotlisted", nil, true},
		{"google.com.", nil, true},
		{"www.google.com.", nil, true},
		// Strict checking rejects unlisted TLD.
		{"example.tldnotlisted", &Options{NoDefaultRule: true}, false},
		// Not a valid domain at all.
		{"http://www.example.com", nil, false},
		{"", nil, false},
		{".com", nil, false},
		// A bare suffix is assigned but not allowed.
		{"com", nil, false},
	}
	for _, tc := range tests {
		if got := Valid(tc.input, tc.opts); got != tc.want {
			t.Errorf("Valid(%q, %+v) = %v; want %v", tc.input, tc.opts, got, tc.want)
		}
	}
}

// --- ignore_private --------------------------------------------------------

func TestIgnorePrivate(t *testing.T) {
	// Find a PRIVATE-section rule to exercise the ignore_private path. Use a
	// well-known one: blogspot.com is in the PRIVATE section.
	const host = "foo.blogspot.com"

	// Default: private rule applies -> tld = blogspot.com.
	d, err := Parse(host, nil)
	if err != nil {
		t.Fatalf("Parse(%q): %v", host, err)
	}
	if d.TLD != "blogspot.com" {
		t.Fatalf("with private: TLD = %q; want blogspot.com", d.TLD)
	}

	// ignore_private: the private rule is skipped, only ICANN "com" remains.
	d2, err := Parse(host, &Options{IgnorePrivate: true})
	if err != nil {
		t.Fatalf("Parse ignore_private(%q): %v", host, err)
	}
	if d2.TLD != "com" {
		t.Fatalf("ignore_private: TLD = %q; want com", d2.TLD)
	}
	if d2.SLD != "blogspot" {
		t.Fatalf("ignore_private: SLD = %q; want blogspot", d2.SLD)
	}
}

// --- Domain value-model accessors (the non-parse branches) -----------------

func TestDomainAccessors(t *testing.T) {
	// TLD only.
	tldOnly := &Domain{TLD: "com", hasTLD: true}
	if tldOnly.Name() != "com" {
		t.Errorf("Name = %q; want com", tldOnly.Name())
	}
	if tldOnly.String() != "com" {
		t.Errorf("String = %q; want com", tldOnly.String())
	}
	if tldOnly.IsDomain() {
		t.Error("tld-only IsDomain() should be false")
	}
	if tldOnly.IsSubdomain() {
		t.Error("tld-only IsSubdomain() should be false")
	}
	if tldOnly.DomainName() != "" {
		t.Errorf("tld-only DomainName = %q; want empty", tldOnly.DomainName())
	}
	if tldOnly.Subdomain() != "" {
		t.Errorf("tld-only Subdomain = %q; want empty", tldOnly.Subdomain())
	}
	if !tldOnly.HasTLD() || tldOnly.HasSLD() || tldOnly.HasTRD() {
		t.Error("tld-only Has* flags wrong")
	}

	// TLD + SLD (domain, not subdomain).
	dom := &Domain{TLD: "com", SLD: "google", hasTLD: true, hasSLD: true}
	if dom.Name() != "google.com" {
		t.Errorf("Name = %q; want google.com", dom.Name())
	}
	if !dom.IsDomain() {
		t.Error("IsDomain() should be true")
	}
	if dom.IsSubdomain() {
		t.Error("IsSubdomain() should be false")
	}
	if dom.DomainName() != "google.com" {
		t.Errorf("DomainName = %q", dom.DomainName())
	}
	if dom.Subdomain() != "" {
		t.Errorf("Subdomain = %q; want empty", dom.Subdomain())
	}

	// Full subdomain.
	sub := &Domain{TLD: "com", SLD: "google", TRD: "www", hasTLD: true, hasSLD: true, hasTRD: true}
	if !sub.IsSubdomain() {
		t.Error("IsSubdomain() should be true")
	}
	if sub.Subdomain() != "www.google.com" {
		t.Errorf("Subdomain = %q", sub.Subdomain())
	}
	if !sub.HasSLD() || !sub.HasTRD() {
		t.Error("Has* flags wrong on full subdomain")
	}

	// Defensive: a hand-built Domain that carries only a trd still joins it in
	// Name() (covers the trd branch independently of sld/tld).
	trdOnly := &Domain{TRD: "www", hasTRD: true}
	if trdOnly.Name() != "www" {
		t.Errorf("trd-only Name = %q; want www", trdOnly.Name())
	}
}

// --- List internals: private_domains:false parse stop ----------------------

func TestParsePrivateDomainsFalse(t *testing.T) {
	withPrivate := parse(rawList, true)
	withoutPrivate := parse(rawList, false)
	if withoutPrivate.size() >= withPrivate.size() {
		t.Fatalf("private:false (%d) should be smaller than private:true (%d)",
			withoutPrivate.size(), withPrivate.size())
	}
	// blogspot.com is a PRIVATE rule: absent when private domains dropped.
	if _, ok := withoutPrivate.rules["blogspot.com"]; ok {
		t.Error("blogspot.com should be absent when private domains dropped")
	}
	if _, ok := withPrivate.rules["blogspot.com"]; !ok {
		t.Error("blogspot.com should be present with private domains")
	}
}

// Custom list via Options.List and the inline comment/blank-line branches.
func TestCustomList(t *testing.T) {
	const src = `
// a comment
com

// wildcard + exception + private
*.ck
!www.ck

===BEGIN PRIVATE DOMAINS===
priv.example
`
	l := parse(src, true)
	opts := &Options{List: l}

	if got := RegistrableDomain("foo.com", opts); got != "foo.com" {
		t.Errorf("foo.com => %q; want foo.com", got)
	}
	// Wildcard: a.b.ck -> tld a.ck.
	d, err := Parse("a.b.ck", opts)
	if err != nil {
		t.Fatalf("a.b.ck: %v", err)
	}
	if d.TLD != "b.ck" {
		t.Errorf("a.b.ck TLD = %q; want b.ck", d.TLD)
	}
	// Exception !www.ck -> www.ck is a registrable domain with tld=ck.
	d2, err := Parse("www.ck", opts)
	if err != nil {
		t.Fatalf("www.ck: %v", err)
	}
	if d2.TLD != "ck" || d2.SLD != "www" {
		t.Errorf("www.ck => tld=%q sld=%q; want ck/www", d2.TLD, d2.SLD)
	}
	// Private rule present with default options.
	if !Valid("x.priv.example", opts) {
		t.Error("x.priv.example should be valid with private rule")
	}
	// Dropped under ignore_private -> falls to default "*" rule on "example".
	d3, err := Parse("x.priv.example", &Options{List: l, IgnorePrivate: true})
	if err != nil {
		t.Fatalf("ignore_private x.priv.example: %v", err)
	}
	if d3.TLD != "example" {
		t.Errorf("ignore_private tld = %q; want example", d3.TLD)
	}
}

// parse must stop at the private marker when private_domains:false (covers the
// early-return branch).
func TestParsePrivateMarkerStop(t *testing.T) {
	const src = `com
===BEGIN PRIVATE DOMAINS===
priv.example
`
	l := parse(src, false)
	if _, ok := l.rules["priv.example"]; ok {
		t.Error("priv.example must be absent when private domains skipped")
	}
	if _, ok := l.rules["com"]; !ok {
		t.Error("com must be present")
	}
}

// --- rule.go direct edge cases ---------------------------------------------

func TestRuleFactoryAndParts(t *testing.T) {
	// Wildcard with empty value ("*"): length = 0-label + 1.
	star := factory("*", false)
	if star.typ != ruleWildcard {
		t.Fatalf("factory(*) typ = %v; want wildcard", star.typ)
	}
	if star.value != "" {
		t.Errorf("factory(*) value = %q; want empty", star.value)
	}
	if got := star.parts(); len(got) != 0 {
		t.Errorf("star.parts() = %v; want empty", got)
	}
	// Wildcard *.co.uk: value = co.uk, length = 2 + 1.
	w := factory("*.co.uk", false)
	if w.value != "co.uk" || w.length != 3 {
		t.Errorf("*.co.uk => value=%q length=%d; want co.uk/3", w.value, w.length)
	}
	// Exception !foo.bar.uk: value foo.bar.uk, parts drop leftmost.
	e := factory("!foo.bar.uk", false)
	if e.typ != ruleException || e.value != "foo.bar.uk" {
		t.Errorf("exception => typ=%v value=%q", e.typ, e.value)
	}
	if got := e.parts(); len(got) != 2 || got[0] != "bar" || got[1] != "uk" {
		t.Errorf("exception parts = %v; want [bar uk]", got)
	}
	// Exception with single label "!uk": parts drops the only label -> [].
	e2 := factory("!uk", false)
	if got := e2.parts(); len(got) != 0 {
		t.Errorf("!uk parts = %v; want empty", got)
	}
	// Normal multi-label.
	n := factory("co.uk", false)
	if n.typ != ruleNormal || n.length != 2 {
		t.Errorf("normal => typ=%v length=%d", n.typ, n.length)
	}
}

// decompose must return ok=false when the name equals the suffix exactly (the
// gem's `(.*)\.` needs a captured left label).
func TestDecomposeNoMatch(t *testing.T) {
	r := factory("com", false)
	if _, _, ok := r.decompose("com"); ok {
		t.Error("decompose(com) on rule com should not match (no left part)")
	}
	if left, right, ok := r.decompose("a.com"); !ok || left != "a" || right != "com" {
		t.Errorf("decompose(a.com) = %q,%q,%v; want a,com,true", left, right, ok)
	}
	// Wildcard rule where the name is too short to fill the wildcard label.
	w := factory("*.mm", false)
	if _, _, ok := w.decompose("c.mm"); ok {
		t.Error("decompose(c.mm) on *.mm should not match (wildcard needs a label)")
	}
	if left, right, ok := w.decompose("x.c.mm"); !ok || left != "x" || right != "c.mm" {
		t.Errorf("decompose(x.c.mm) = %q,%q,%v; want x,c.mm,true", left, right, ok)
	}
	// Suffix labels mismatch (right labels differ).
	if _, _, ok := r.decompose("a.net"); ok {
		t.Error("decompose(a.net) on rule com should not match")
	}
}

// splitLabels on empty string returns nil.
func TestSplitLabelsEmpty(t *testing.T) {
	if got := splitLabels(""); got != nil {
		t.Errorf("splitLabels(\"\") = %v; want nil", got)
	}
	if got := splitLabels("a.b"); len(got) != 2 {
		t.Errorf("splitLabels(a.b) = %v; want 2", got)
	}
}

// selectRules on an empty name returns nil (labels empty branch).
func TestSelectRulesEmpty(t *testing.T) {
	l := defaultList()
	if got := l.selectRules("", false); got != nil {
		t.Errorf("selectRules(\"\") = %v; want nil", got)
	}
}

// default list is memoized.
func TestDefaultListMemoized(t *testing.T) {
	a := defaultList()
	b := defaultList()
	if a != b {
		t.Error("defaultList() must return the same memoized instance")
	}
	if a.size() == 0 {
		t.Error("default list must be non-empty")
	}
}

// Error.Error/Unwrap direct coverage.
func TestErrorType(t *testing.T) {
	e := newInvalid("boom")
	if e.Error() != "boom" {
		t.Errorf("Error() = %q", e.Error())
	}
	if !errors.Is(e, ErrDomainInvalid) {
		t.Error("newInvalid must wrap ErrDomainInvalid")
	}
	na := newNotAllowed("nope")
	if !errors.Is(na, ErrDomainNotAllowed) || !errors.Is(na, ErrDomainInvalid) {
		t.Error("newNotAllowed must wrap both sentinels")
	}
}
