// Package goemit turns selected TEI rules into Go source that builds and runs
// with the Go standard library alone. The emitted package imports nothing
// under tei/, so a team can keep the code after it stops using TEI.
//
// Emit returns the file contents, one trace line per rule, and diagnostics.
// It writes no files. Each rule gives two functions:
//
//   - a decision function, such as QuantityIsPositive, which receives only a
//     present value and answers true or false;
//   - a check function, such as CheckQuantityIsPositive, for callers outside
//     TEI, which answers Undecided when the value is missing.
//
// The package also receives result.go (Result and its three values) and
// decimal.go (ParseDecimal). Both are standard parts kept as ordinary Go
// files under internal/stdpart, so they are compiled and tested here before
// they are copied into generated code.
//
// Names come from the rule's Attribute as described in naming.go. Element
// IDs appear only in comments and trace lines, never in identifiers.
package goemit
