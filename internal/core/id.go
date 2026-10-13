// Package core retains the old import path during the incremental migration.
// The CanonicalID, LocalName, Kind and Ref implementations are owned by
// core/identity, not duplicated here.
package core

import "tei/core/identity"

type CanonicalID = identity.CanonicalID
type LocalName = identity.LocalName
type Kind = identity.Kind

const (
	KindSemantic    Kind = identity.KindSemantic
	KindApplication Kind = identity.KindApplication
	KindDatabase    Kind = identity.KindDatabase
	KindContract    Kind = identity.KindContract
	KindEnvironment Kind = identity.KindEnvironment
	KindCapability  Kind = identity.KindCapability
)

type Ref = identity.Ref

func ParseCanonicalID(s string) (CanonicalID, error) {
	return identity.ParseCanonicalID(s)
}

func MustCanonicalID(s string) CanonicalID {
	return identity.MustCanonicalID(s)
}

func ParseLocalName(s string) (LocalName, error) {
	return identity.ParseLocalName(s)
}

func MustLocalName(s string) LocalName {
	return identity.MustLocalName(s)
}

func SymbolToSegment(symbol string) (string, error) {
	return identity.SymbolToSegment(symbol)
}

func MustSymbolSegment(symbol string) string {
	return identity.MustSymbolSegment(symbol)
}
