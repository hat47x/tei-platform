// Package plancheck holds repository-level consistency checks for the planning
// documents (Issue / ADR numbering, Issue states, Markdown relative links).
//
// The checks are ordinary Go tests so that `go test ./...`, and therefore
// tools/repository-health-check.sh, catches them without CI.
package plancheck
