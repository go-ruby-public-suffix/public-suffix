// Copyright (c) the go-ruby-public-suffix/public-suffix authors
//
// SPDX-License-Identifier: BSD-3-Clause

package publicsuffix

import "strings"

// ruleType enumerates the three kinds of Public Suffix List rule, mirroring
// the PublicSuffix::Rule::{Normal,Wildcard,Exception} subclasses.
type ruleType int

const (
	ruleNormal ruleType = iota
	ruleWildcard
	ruleException
)

// rule is a single Public Suffix List entry. It mirrors PublicSuffix::Rule::Base:
//   - value is the normalized rule name (the "*." / "!" prefix stripped).
//   - length is the matched-label count used to pick the longest match.
//   - private flags a rule from the PRIVATE DOMAINS section.
type rule struct {
	typ     ruleType
	value   string
	length  int
	private bool
}

// factory detects the rule kind from content and builds the normalized rule,
// mirroring PublicSuffix::Rule.factory + the per-subclass build/initialize.
func factory(content string, private bool) rule {
	switch {
	case strings.HasPrefix(content, star):
		// Wildcard.build strips the leading "*." (2 chars); length gets +1
		// for the implicit "*" label.
		value := ""
		if len(content) > 2 {
			value = content[2:]
		}
		return rule{
			typ:     ruleWildcard,
			value:   value,
			length:  labelLen(value) + 1,
			private: private,
		}
	case strings.HasPrefix(content, bang):
		// Exception.build strips the leading "!".
		value := content[1:]
		return rule{
			typ:     ruleException,
			value:   value,
			length:  labelLen(value),
			private: private,
		}
	default:
		return rule{
			typ:     ruleNormal,
			value:   content,
			length:  labelLen(content),
			private: private,
		}
	}
}

// defaultRule returns the "*" rule used when no list rule matches, mirroring
// PublicSuffix::Rule.default (factory("*")).
func defaultRule() rule { return factory(star, false) }

// labelLen is the gem's `value.count(".") + 1` — the number of dot-separated
// labels in value.
func labelLen(value string) int {
	return strings.Count(value, dot) + 1
}

// parts returns the rule's labels, mirroring each subclass's #parts. For an
// exception rule the leftmost label is dropped (per the PSL format spec).
func (r rule) parts() []string {
	if r.value == "" {
		// "*" default / empty wildcard rule: value is empty, split yields [].
		return []string{}
	}
	labels := strings.Split(r.value, dot)
	if r.typ == ruleException {
		return labels[1:]
	}
	return labels
}

// decompose splits name into [left, right] where right is the matched public
// suffix and left is everything before it, mirroring the per-subclass
// #decompose (which the gem implements with a `/^(.*)\.(suffix)$/` regex).
// The bool is false when name does not match (the gem returns [nil, nil]).
func (r rule) decompose(name string) (left, right string, ok bool) {
	parts := r.parts()
	if r.typ == ruleWildcard {
		// suffix = ([".*?"] + parts).join('\.') — i.e. one extra,
		// non-greedy "any label" in front of the rule's labels.
		return splitSuffix(name, append([]string{wildcardLabel}, parts...))
	}
	return splitSuffix(name, parts)
}

// wildcardLabel is a sentinel marking the "*" position in a wildcard rule's
// suffix; splitSuffix matches it against exactly one (non-empty, dot-free)
// label, reproducing the gem's non-greedy `.*?` between anchored `\.`.
const wildcardLabel = "\x00*"

// splitSuffix reproduces the gem's `/^(.*)\.(suffix)$/` match where suffix is
// the dot-joined sequence of labels (with wildcardLabel standing for one
// arbitrary label). It returns the captured left part, the captured suffix,
// and whether the whole pattern matched.
//
// The pattern requires at least one label (the greedy `(.*)` plus the
// mandatory `\.`) before the suffix, so a name equal to just the suffix does
// not match — exactly like the gem.
func splitSuffix(name string, suffixLabels []string) (string, string, bool) {
	nameLabels := splitLabels(name)
	if len(nameLabels) <= len(suffixLabels) {
		// Need at least one label captured by `(.*)\.`.
		return "", "", false
	}
	// The suffix is the rightmost len(suffixLabels) labels of name.
	cut := len(nameLabels) - len(suffixLabels)
	tail := nameLabels[cut:]
	for i, want := range suffixLabels {
		if want == wildcardLabel {
			// Matches any single label; the gem's `.*?` is non-greedy but
			// anchored by `\.` on both sides, so it spans exactly one label.
			continue
		}
		if tail[i] != want {
			return "", "", false
		}
	}
	left := strings.Join(nameLabels[:cut], dot)
	right := strings.Join(tail, dot)
	return left, right, true
}

// splitLabels splits a domain into its dot-separated labels. An empty string
// yields an empty slice (matching Ruby's "".split(".") == []).
func splitLabels(name string) []string {
	if name == "" {
		return nil
	}
	return strings.Split(name, dot)
}
