<p align="center"><img src="https://raw.githubusercontent.com/go-ruby-public-suffix/brand/main/social/go-ruby-public-suffix-public-suffix.png" alt="go-ruby-public-suffix/public-suffix" width="720"></p>

# public-suffix — go-ruby-public-suffix

[![Docs](https://img.shields.io/badge/docs-mkdocs--material-DC2626)](https://go-ruby-public-suffix.github.io/docs/)
[![License](https://img.shields.io/badge/license-BSD--3--Clause-blue)](LICENSE)
[![Go](https://img.shields.io/badge/go-1.26.4%2B-00ADD8)](https://go.dev/dl/)
[![Coverage](https://img.shields.io/badge/coverage-100%25-1a7f37)](#tests--coverage)

**A pure-Go (no cgo) reimplementation of Ruby's [public_suffix] gem** — a domain
name parser based on the Mozilla [Public Suffix List]. Given a hostname it
splits it into the `trd` (subdomain), `sld` (second-level domain) and `tld`
(public suffix), **byte-for-byte faithful** to the gem's decomposition, so
`PublicSuffix.domain("www.example.co.uk")` and friends behave identically —
**without any Ruby runtime**.

The Public Suffix List itself is **embedded** (`go:embed` of a committed
`public_suffix_list.dat`), so the module is complete and works entirely
**offline** with no network access at run time.

It is the public-suffix backend for
[go-embedded-ruby](https://github.com/go-embedded-ruby/ruby), but is a
**standalone, reusable** module with no dependency on the Ruby runtime — a
sibling of [go-ruby-uri](https://github.com/go-ruby-uri/uri) and
[go-ruby-ipaddr](https://github.com/go-ruby-ipaddr/ipaddr).

## Features

Faithful port of the gem's `parse` / `domain` / `valid?`, validated against the
real `public_suffix` gem on every supported platform:

- **Full decomposition** — `Parse` returns a `*Domain` with `TLD`, `SLD`, `TRD`,
  plus `DomainName()` (`sld.tld`), `Subdomain()` (`trd.sld.tld`), `Name()`,
  `IsDomain()` and `IsSubdomain()`, mirroring `PublicSuffix::Domain`.
- **Every rule type** — **normal** (`com`), **wildcard** (`*.ck`) and
  **exception** (`!www.ck`) rules, with the gem's longest-match selection and
  exception short-circuit.
- **ICANN vs PRIVATE** sections — the `IgnorePrivate` option skips the
  PRIVATE DOMAINS section (`ignore_private: true`).
- **Default `*` rule** for unlisted TLDs, toggled off with `NoDefaultRule`
  (`default_rule: nil`) for strict checking.
- **Normalization** — Unicode-aware downcasing, a single trailing dot stripped,
  and the gem's blank / leading-dot / scheme rejections.
- **IDN & punycode** labels pass through untouched, exactly as the gem (which
  performs no IDN conversion itself).
- **Gem-faithful errors** — `ErrDomainInvalid` and `ErrDomainNotAllowed`
  (a subtype, per the gem's `DomainNotAllowed < DomainInvalid`).

CGO-free, dependency-free, **100% test coverage**, `gofmt` + `go vet` clean, and
green across the six 64-bit Go targets (amd64, arm64, riscv64, loong64, ppc64le,
s390x) and three operating systems (Linux, macOS, Windows).

## Install

```sh
go get github.com/go-ruby-public-suffix/public-suffix
```

## Usage

```go
package main

import (
	"fmt"

	publicsuffix "github.com/go-ruby-public-suffix/public-suffix"
)

func main() {
	// Registrable domain (PublicSuffix.domain).
	fmt.Println(publicsuffix.RegistrableDomain("www.example.co.uk", nil)) // example.co.uk

	// Full decomposition (PublicSuffix.parse).
	d, _ := publicsuffix.Parse("www.example.co.uk", nil)
	fmt.Println(d.TLD, d.SLD, d.TRD)   // co.uk example www
	fmt.Println(d.DomainName())        // example.co.uk
	fmt.Println(d.Subdomain())         // www.example.co.uk

	// Validation (PublicSuffix.valid?).
	fmt.Println(publicsuffix.Valid("example.com", nil)) // true

	// Strict checking, no default "*" rule.
	fmt.Println(publicsuffix.Valid("example.tldnotlisted",
		&publicsuffix.Options{NoDefaultRule: true})) // false

	// Skip the PRIVATE DOMAINS section.
	pd, _ := publicsuffix.Parse("foo.blogspot.com",
		&publicsuffix.Options{IgnorePrivate: true})
	fmt.Println(pd.TLD) // com  (instead of blogspot.com)
}
```

## API

| Go | Ruby gem |
| --- | --- |
| `Parse(name, opts) (*Domain, error)` | `PublicSuffix.parse(name, ...)` |
| `RegistrableDomain(name, opts) string` | `PublicSuffix.domain(name, ...)` |
| `Valid(name, opts) bool` | `PublicSuffix.valid?(name, ...)` |
| `Domain{TLD, SLD, TRD}` + `Name/DomainName/Subdomain/IsDomain/IsSubdomain` | `PublicSuffix::Domain` |
| `Options{List, NoDefaultRule, IgnorePrivate}` | the `list:` / `default_rule:` / `ignore_private:` keywords |
| `ErrDomainInvalid` / `ErrDomainNotAllowed` | `DomainInvalid` / `DomainNotAllowed` |

`opts` may be `nil` for the gem defaults (default list, `*` default rule,
private domains honoured).

## Tests & coverage

```sh
go test ./...                 # deterministic suite + embedded-list vectors
go test -race -cover ./...    # 100% statement coverage
```

The suite runs the official Public Suffix List `checkPublicSuffix` test vectors
(`testdata/psl_tests.txt`) and a **differential oracle** against the real
`public_suffix` gem (which it skips automatically when `ruby` or the gem is
absent — the Windows and qemu cross-arch lanes — so the deterministic tests
alone hold the 100% gate there).

## Refreshing the list

The embedded `public_suffix_list.dat` is a committed copy of the canonical list.
To update it:

```sh
curl -sSL https://publicsuffix.org/list/public_suffix_list.dat -o public_suffix_list.dat
```

## License

[BSD-3-Clause](LICENSE) © the go-ruby-public-suffix/public-suffix authors.

[public_suffix]: https://rubygems.org/gems/public_suffix
[Public Suffix List]: https://publicsuffix.org
