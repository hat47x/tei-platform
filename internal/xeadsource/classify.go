package xeadsource

import "sort"

type CandidateRule struct {
	Classifications []string `json:"classifications"`
	Extension       string   `json:"extension,omitempty"`
	AuthorityLimits []string `json:"authorityLimits,omitempty"`
}

type Candidate struct {
	Locator         string   `json:"locator"`
	SourceName      string   `json:"sourceName"`
	Classifications []string `json:"classifications"`
	Extension       string   `json:"extension,omitempty"`
	AuthorityLimits []string `json:"authorityLimits,omitempty"`
	Unresolved      bool     `json:"unresolved"`
}

type ClassificationPolicy map[string]CandidateRule

// Classify applies an explicitly supplied candidate policy to source facts.
// The policy is intentionally external: parsing source bytes must not grant
// canonical meaning or make the adapter a second semantic authority.
func Classify(doc Document, policy ClassificationPolicy) []Candidate {
	out := make([]Candidate, 0)
	var walk func(Node)
	walk = func(n Node) {
		rule, ok := policy[localName(n.Name)]
		c := Candidate{
			Locator:    n.Locator,
			SourceName: n.Name,
			Unresolved: !ok,
		}
		if ok {
			c.Classifications = append([]string(nil), rule.Classifications...)
			sort.Strings(c.Classifications)
			c.Extension = rule.Extension
			c.AuthorityLimits = append([]string(nil), rule.AuthorityLimits...)
			sort.Strings(c.AuthorityLimits)
		}
		out = append(out, c)
		for _, child := range n.Children {
			walk(child)
		}
	}
	walk(doc.Root)
	return out
}
