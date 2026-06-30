// Copyright (c) the go-ruby-public-suffix/public-suffix authors
//
// SPDX-License-Identifier: BSD-3-Clause

package publicsuffix

import (
	"errors"
	"fmt"
	"strings"
)

// Error is the base error type, mirroring PublicSuffix::Error. Use
// [errors.Is] against [ErrDomainInvalid] / [ErrDomainNotAllowed] to classify.
type Error struct {
	// Kind is the sentinel this error wraps (ErrDomainInvalid or
	// ErrDomainNotAllowed).
	Kind error
	// Msg is the gem's human-readable message.
	Msg string
}

func (e *Error) Error() string { return e.Msg }

// Unwrap exposes Kind so errors.Is can match the sentinels. Because
// DomainNotAllowed is a subclass of DomainInvalid in the gem,
// ErrDomainNotAllowed itself unwraps to ErrDomainInvalid (see below), so a
// not-allowed error also satisfies errors.Is(err, ErrDomainInvalid).
func (e *Error) Unwrap() error { return e.Kind }

var (
	// ErrDomainInvalid mirrors PublicSuffix::DomainInvalid — raised when a
	// name is not a valid domain (blank, leading dot, contains a scheme, or
	// no rule matches under a nil default rule).
	ErrDomainInvalid = errors.New("public_suffix: domain invalid")

	// ErrDomainNotAllowed mirrors PublicSuffix::DomainNotAllowed — raised when
	// a rule matches but does not allow the name (e.g. it is itself just a
	// public suffix). In the gem DomainNotAllowed < DomainInvalid, so this
	// sentinel also unwraps to ErrDomainInvalid.
	ErrDomainNotAllowed = fmt.Errorf("public_suffix: domain not allowed: %w", ErrDomainInvalid)
)

func newInvalid(msg string) *Error {
	return &Error{Kind: ErrDomainInvalid, Msg: msg}
}

func newNotAllowed(msg string) *Error {
	return &Error{Kind: ErrDomainNotAllowed, Msg: msg}
}

// Options configures Parse / Domain / Valid, mirroring the gem's keyword
// arguments. The zero value is the gem's default: the default "*" rule applies
// and private domains are honoured.
type Options struct {
	// List is the rule list to search. nil uses the embedded default list.
	List *List

	// NoDefaultRule disables the fallback "*" rule (the gem's
	// default_rule: nil). When set, an unlisted TLD is invalid rather than
	// being split as a single-label suffix.
	NoDefaultRule bool

	// IgnorePrivate skips PRIVATE DOMAINS section rules during matching,
	// mirroring ignore_private: true.
	IgnorePrivate bool
}

func (o *Options) list() *List {
	if o != nil && o.List != nil {
		return o.List
	}
	return defaultList()
}

func (o *Options) defaultRule() (rule, bool) {
	if o != nil && o.NoDefaultRule {
		return rule{}, false
	}
	return defaultRule(), true
}

func (o *Options) ignorePrivate() bool {
	return o != nil && o.IgnorePrivate
}

// normalize mirrors PublicSuffix.normalize: strip surrounding whitespace,
// remove a single trailing dot, and downcase. It returns an error when the
// name is blank, starts with a dot, or contains a scheme ("://").
func normalize(name string) (string, error) {
	name = strings.TrimSpace(name)
	// Ruby's chomp!(".") removes at most one trailing dot.
	name = strings.TrimSuffix(name, dot)
	// Ruby's downcase! is Unicode-aware; strings.ToLower matches it for the
	// case-foldings exercised by domain names.
	name = strings.ToLower(name)

	switch {
	case name == "":
		return "", newInvalid("Name is blank")
	case strings.HasPrefix(name, dot):
		return "", newInvalid("Name starts with a dot")
	case strings.Contains(name, "://"):
		return "", newInvalid(fmt.Sprintf("%s is not expected to contain a scheme", name))
	}
	return name, nil
}

// Parse parses name and returns its decomposition, mirroring
// PublicSuffix.parse. Pass nil opts for the gem defaults.
//
// It returns an error wrapping [ErrDomainInvalid] when the name cannot be
// normalized or no rule matches (with NoDefaultRule), and one wrapping
// [ErrDomainNotAllowed] when a rule matches but the name is not allowed under
// it (e.g. the name is itself only a public suffix).
func Parse(name string, opts *Options) (*Domain, error) {
	what, err := normalize(name)
	if err != nil {
		return nil, err
	}

	list := opts.list()
	def, hasDefault := opts.defaultRule()
	r, found := list.find(what, def, hasDefault, opts.ignorePrivate())
	if !found {
		return nil, newInvalid(fmt.Sprintf("`%s` is not a valid domain", what))
	}
	if _, _, ok := r.decompose(what); !ok {
		return nil, newNotAllowed(fmt.Sprintf("`%s` is not allowed according to Registry policy", what))
	}
	return decompose(what, r), nil
}

// RegistrableDomain returns the registrable domain (sld.tld) for name,
// mirroring PublicSuffix.domain. It returns "" (and never an error) when name
// is not a valid registrable domain — the gem's rescue-to-nil behaviour. Pass
// nil opts for the gem defaults.
//
// (The gem names this PublicSuffix.domain; Go cannot share that name with the
// [Domain] result type, so it is spelled out here.)
func RegistrableDomain(name string, opts *Options) string {
	d, err := Parse(name, opts)
	if err != nil {
		return ""
	}
	return d.DomainName()
}

// Valid reports whether name is an assigned, allowed domain, mirroring
// PublicSuffix.valid?. It does not distinguish domains from subdomains and
// never returns an error. Pass nil opts for the gem defaults.
func Valid(name string, opts *Options) bool {
	what, err := normalize(name)
	if err != nil {
		return false
	}

	list := opts.list()
	def, hasDefault := opts.defaultRule()
	r, found := list.find(what, def, hasDefault, opts.ignorePrivate())
	if !found {
		return false
	}
	_, _, ok := r.decompose(what)
	return ok
}

// decompose builds the Domain from the matched rule, mirroring
// PublicSuffix.decompose. left is split on dots: the last label is the sld and
// the rest (joined) are the trd; right is the tld.
func decompose(name string, r rule) *Domain {
	left, right, _ := r.decompose(name)

	parts := splitLabels(left)
	d := &Domain{TLD: right, hasTLD: true}
	if len(parts) > 0 {
		d.SLD = parts[len(parts)-1]
		d.hasSLD = true
		parts = parts[:len(parts)-1]
	}
	if len(parts) > 0 {
		d.TRD = strings.Join(parts, dot)
		d.hasTRD = true
	}
	return d
}
