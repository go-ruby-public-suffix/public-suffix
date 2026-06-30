// Copyright (c) the go-ruby-public-suffix/public-suffix authors
//
// SPDX-License-Identifier: BSD-3-Clause

package publicsuffix

import (
	_ "embed"
	"strings"
	"sync"
)

// rawList is the committed copy of the Public Suffix List, embedded so the
// package is complete and works offline. Refresh it from
// https://publicsuffix.org/list/public_suffix_list.dat.
//
//go:embed public_suffix_list.dat
var rawList string

// List is a parsed collection of Public Suffix List rules, mirroring
// PublicSuffix::List. Rules are indexed by their normalized value, preserving
// the gem's hash-of-value semantics (the last rule with a given value wins).
type List struct {
	rules map[string]rule
}

// parse parses the content of a Public Suffix List, mirroring
// PublicSuffix::List.parse. When privateDomains is false the scan stops at the
// "===BEGIN PRIVATE DOMAINS===" marker, dropping every private rule.
func parse(input string, privateDomains bool) *List {
	const (
		commentToken = "//"
		privateToken = "===BEGIN PRIVATE DOMAINS==="
	)
	// section: 0 == ICANN (default), 2 == PRIVATE — mirrors the gem (which
	// leaves the ICANN sentinel as nil and never uses 1).
	section := 0
	l := &List{rules: make(map[string]rule)}

	// Ruby's String#each_line splits on "\n", keeping the separator; after
	// strip! only the content matters, so a plain line split is equivalent.
	for _, line := range strings.Split(input, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case line == "":
			continue
		case strings.Contains(line, privateToken):
			if !privateDomains {
				// break out of the each_line loop entirely.
				return l
			}
			section = 2
		case strings.HasPrefix(line, commentToken):
			continue
		default:
			l.add(factory(line, section == 2))
		}
	}
	return l
}

// add inserts rule, keyed by its value (mirrors PublicSuffix::List#add).
func (l *List) add(r rule) {
	l.rules[r.value] = r
}

// size returns the number of indexed rules (mirrors PublicSuffix::List#size).
func (l *List) size() int { return len(l.rules) }

// selectRules returns every rule whose value is a suffix of name, walking from
// the rightmost label inward, mirroring PublicSuffix::List#select. When
// ignorePrivate is true, private rules are skipped (but still traversed).
func (l *List) selectRules(name string, ignorePrivate bool) []rule {
	labels := splitLabels(name)
	// Reverse to walk right-to-left.
	for i, j := 0, len(labels)-1; i < j; i, j = i+1, j-1 {
		labels[i], labels[j] = labels[j], labels[i]
	}
	if len(labels) == 0 {
		return nil
	}

	var rules []rule
	query := labels[0]
	for index := 0; ; {
		if r, present := l.rules[query]; present {
			if !ignorePrivate || !r.private {
				rules = append(rules, r)
			}
		}
		index++
		if index >= len(labels) {
			break
		}
		query = labels[index] + dot + query
	}
	return rules
}

// find returns the rule corresponding to the longest matching public suffix,
// mirroring PublicSuffix::List#find. An exception rule short-circuits and wins
// immediately; otherwise the longest (by label count) normal/wildcard rule
// wins. The supplied def rule is returned when nothing matches; when def has
// hasDefault false (the gem's nil default_rule) the bool is false.
func (l *List) find(name string, def rule, hasDefault, ignorePrivate bool) (rule, bool) {
	matches := l.selectRules(name, ignorePrivate)

	var best rule
	have := false
	for _, r := range matches {
		if r.typ == ruleException {
			return r, true
		}
		if !have || r.length > best.length {
			best = r
			have = true
		}
	}
	if have {
		return best, true
	}
	return def, hasDefault
}

var (
	defaultListOnce sync.Once
	defaultListVal  *List
)

// defaultList lazily parses the embedded list (private domains included),
// mirroring PublicSuffix::List.default's memoization.
func defaultList() *List {
	defaultListOnce.Do(func() {
		defaultListVal = parse(rawList, true)
	})
	return defaultListVal
}
