package goemit

import (
	"go/token"
	"go/types"
	"strings"
	"unicode"
)

// Naming rule. A positive rule takes its names from the last segment of its
// Attribute (the part after the last '.'), not from the rule ID or Symbol:
// the rule ID stays the same when an author points the rule at another
// attribute, and the generated names must follow what the rule checks.
//
// The segment is split into words at '_' and '-' and at case boundaries
// ("secondQuantity" -> second, quantity; "HTTPServer" -> http, server).
// From the words:
//
//   - decision function: each word capitalized, then "IsPositive"
//     (sales.order_line.second_quantity -> SecondQuantityIsPositive);
//   - check function: "Check" followed by the decision function name;
//   - parameter: the words in lower camel case (secondQuantity);
//   - file: the words in lower case joined by '_', then "_is_positive.go".
//
// The initialisms ID, URL, HTTP, HTTPS and API are written in upper case
// (customer_id -> CustomerIDIsPositive, parameter customerID). An initialism
// followed by a lower-case 's' at the end of a word is its plural and stays
// one word: userIDs and user_ids both give UserIDsIsPositive, parameter
// userIDs, file user_ids_is_positive.go. Only the case boundaries listed
// above split a word, so some spellings give names that look odd but are
// still predictable:
//
//   - iD splits into i and d (IDIsPositive, parameter iD, file
//     i_d_is_positive.go); write id instead;
//   - HTTPSUrl splits into https and url (HTTPSURLIsPositive, parameter
//     httpsURL, file https_url_is_positive.go).
//
// A name is never changed to make it unique or legal. Decision and check
// functions share one name space, so two rules that give the
// same name produce NAME_COLLISION, a parameter that is a Go keyword or a
// predeclared or imported name produces NAME_RESERVED, and a segment that
// does not give a Go identifier produces NAME_INVALID. The author then renames
// the attribute.

// initialisms are the words written in upper case inside a Go name.
var initialisms = map[string]string{
	"id":    "ID",
	"url":   "URL",
	"http":  "HTTP",
	"https": "HTTPS",
	"api":   "API",
}

// packageNames are the names every generated package already uses at package
// level or as an import. A parameter with one of these names would hide it.
var packageNames = map[string]bool{
	"big":          true,
	"fmt":          true,
	"Result":       true,
	"Satisfied":    true,
	"NotSatisfied": true,
	"Undecided":    true,
	"ParseDecimal": true,
	"notDecimal":   true,
}

// positiveNames are the Go names of one positive rule.
type positiveNames struct {
	Decision  string
	Check     string
	Parameter string
	File      string
}

// splitWords splits a name at '.', '_' and '-' and at case boundaries. It
// returns nil when the name holds anything other than ASCII letters, digits
// and those separators.
func splitWords(name string) []string {
	var words []string
	var current []rune
	flush := func() {
		if len(current) > 0 {
			words = append(words, strings.ToLower(string(current)))
			current = current[:0]
		}
	}
	runes := []rune(name)
	for i, r := range runes {
		switch {
		case r == '.' || r == '_' || r == '-':
			flush()
			continue
		case r > unicode.MaxASCII || !(unicode.IsLetter(r) || unicode.IsDigit(r)):
			return nil
		}
		if unicode.IsUpper(r) && len(current) > 0 {
			previous := current[len(current)-1]
			nextLower := i+1 < len(runes) && unicode.IsLower(runes[i+1])
			if unicode.IsLower(previous) || unicode.IsDigit(previous) ||
				(unicode.IsUpper(previous) && nextLower && !pluralEnd(runes, i+1)) {
				flush()
			}
		}
		current = append(current, r)
	}
	flush()
	return words
}

// pluralEnd reports whether runes[at] is an 's' that ends a word: the last
// rune, or one followed by a separator or an upper-case letter. "IDs" in
// userIDs and userIDsCount is then one word, while "IDsum" still splits.
func pluralEnd(runes []rune, at int) bool {
	if runes[at] != 's' {
		return false
	}
	if at+1 == len(runes) {
		return true
	}
	next := runes[at+1]
	return next == '.' || next == '_' || next == '-' || unicode.IsUpper(next)
}

func exportedWord(word string) string {
	if upper, ok := initialisms[word]; ok {
		return upper
	}
	if singular, ok := strings.CutSuffix(word, "s"); ok {
		if upper, ok := initialisms[singular]; ok {
			return upper + "s"
		}
	}
	return strings.ToUpper(word[:1]) + word[1:]
}

// namePositive derives the names of a positive rule from its attribute. It
// returns false when the attribute segment does not give a Go identifier.
func namePositive(attribute string) (positiveNames, bool) {
	segment := attribute[strings.LastIndex(attribute, ".")+1:]
	words := splitWords(segment)
	if len(words) == 0 || !unicode.IsLetter(rune(words[0][0])) {
		return positiveNames{}, false
	}
	var exported, parameter strings.Builder
	for i, word := range words {
		exported.WriteString(exportedWord(word))
		if i == 0 {
			parameter.WriteString(word)
		} else {
			parameter.WriteString(exportedWord(word))
		}
	}
	decision := exported.String() + "IsPositive"
	return positiveNames{
		Decision:  decision,
		Check:     "Check" + decision,
		Parameter: parameter.String(),
		File:      strings.Join(words, "_") + "_is_positive.go",
	}, true
}

// reservedName reports why name cannot be a parameter, or "" when it can.
func reservedName(name string) string {
	switch {
	case token.IsKeyword(name):
		return "a Go keyword"
	case types.Universe.Lookup(name) != nil:
		return "a predeclared Go identifier"
	case packageNames[name]:
		return "a name the generated package already uses"
	}
	return ""
}
