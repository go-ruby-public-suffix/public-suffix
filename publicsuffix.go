// Copyright (c) the go-ruby-public-suffix/public-suffix authors
//
// SPDX-License-Identifier: BSD-3-Clause

// Package publicsuffix is a pure-Go (no cgo) reimplementation of Ruby's
// [public_suffix] gem — a domain-name parser based on the Mozilla
// [Public Suffix List]. It is byte-for-byte faithful to the gem's
// decomposition: given a hostname it splits it into the trd (subdomain),
// sld (second-level domain) and tld (public suffix), honouring the list's
// normal, wildcard (*) and exception (!) rule grammar, the ICANN / PRIVATE
// section split (ignore_private), Unicode and punycode labels, the trailing
// dot and the default "*" rule for unlisted TLDs.
//
// The list itself is embedded (go:embed) from a committed copy of
// public_suffix_list.dat, so the package is complete and works entirely
// offline with no network access at run time.
//
// It is the public-suffix backend for [go-embedded-ruby], but is a
// standalone, reusable module with no dependency on the Ruby runtime.
//
// [public_suffix]: https://rubygems.org/gems/public_suffix
// [Public Suffix List]: https://publicsuffix.org
// [go-embedded-ruby]: https://github.com/go-embedded-ruby/ruby
package publicsuffix

import (
	"strings"
)

// Rule grammar tokens, mirroring the gem's PublicSuffix module constants.
const (
	dot  = "."
	bang = "!"
	star = "*"
)

// Domain is the decomposition of a hostname into its three significant
// levels, mirroring PublicSuffix::Domain. Any of the fields may be empty
// ("") when the corresponding level is absent; the gem models that level as
// nil. Use [Domain.HasTLD], [Domain.HasSLD] and [Domain.HasTRD] (or the
// zero value of each field) to distinguish "absent" from "present".
type Domain struct {
	// TLD is the public suffix (the matched rule's labels), e.g. "com".
	TLD string
	// SLD is the second-level (registrable) label, e.g. "google".
	SLD string
	// TRD is the third-and-higher level labels joined by dots (the
	// subdomain), e.g. "www" or "a.b".
	TRD string

	hasTLD bool
	hasSLD bool
	hasTRD bool
}

// HasTLD reports whether the TLD level is present (gem: tld != nil).
func (d *Domain) HasTLD() bool { return d.hasTLD }

// HasSLD reports whether the SLD level is present (gem: sld != nil).
func (d *Domain) HasSLD() bool { return d.hasSLD }

// HasTRD reports whether the TRD level is present (gem: trd != nil).
func (d *Domain) HasTRD() bool { return d.hasTRD }

// Name returns the full domain name — the present levels (trd, sld, tld)
// joined by dots. Mirrors PublicSuffix::Domain#name / #to_s.
func (d *Domain) Name() string {
	parts := make([]string, 0, 3)
	if d.hasTRD {
		parts = append(parts, d.TRD)
	}
	if d.hasSLD {
		parts = append(parts, d.SLD)
	}
	if d.hasTLD {
		parts = append(parts, d.TLD)
	}
	return strings.Join(parts, dot)
}

// String returns Name, so a *Domain prints like the gem's #to_s.
func (d *Domain) String() string { return d.Name() }

// IsDomain reports whether self looks like a domain — it has both a tld and
// an sld. Mirrors PublicSuffix::Domain#domain?.
func (d *Domain) IsDomain() bool { return d.hasTLD && d.hasSLD }

// IsSubdomain reports whether self looks like a subdomain — it has a tld, an
// sld and a trd. Mirrors PublicSuffix::Domain#subdomain?.
func (d *Domain) IsSubdomain() bool { return d.hasTLD && d.hasSLD && d.hasTRD }

// DomainName returns the registrable domain "sld.tld" when IsDomain,
// otherwise "". Mirrors PublicSuffix::Domain#domain.
func (d *Domain) DomainName() string {
	if !d.IsDomain() {
		return ""
	}
	return d.SLD + dot + d.TLD
}

// Subdomain returns the fully-qualified subdomain "trd.sld.tld" when
// IsSubdomain, otherwise "". Mirrors PublicSuffix::Domain#subdomain.
func (d *Domain) Subdomain() string {
	if !d.IsSubdomain() {
		return ""
	}
	return d.TRD + dot + d.SLD + dot + d.TLD
}
