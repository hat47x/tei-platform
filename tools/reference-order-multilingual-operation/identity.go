// Package ordermultilingualoperation runs the technical validation sample
// order-multilingual-operation (sample 8, stages 1-6) as one through record.
//
// The package is sample specific and lives under tools/. It calls the
// existing multilingual reference assets unchanged and keeps every identifier
// that the assets used to synthesize separately (owner plugin, environment,
// source language, field name) in one place: Unified. Nothing here is a
// product translation service, host authorization or acceptance.
package ordermultilingualoperation

import (
	"tei/internal/core"
	"tei/internal/modelref"
	"tei/internal/plugin"
	"tei/internal/subject"
)

// Unified is the single identity set the through record uses for every
// stage. Environment and application come from reference/workspace. The
// workspace has no plugin that owns the order business definitions, no
// std.message and no language plugin; those are added to the copy by the
// operator (stage 1 and 2) and are listed in PlatformGaps.
type Unified struct {
	Workspace    string          `json:"workspace"`
	Environment  core.CanonicalID `json:"environment"`
	Application  core.CanonicalID `json:"application"`
	Owner        plugin.Identity  `json:"owner"`
	Message      plugin.Identity  `json:"message"`
	SourceLocale string           `json:"sourceLocale"`
	Scope        string           `json:"scope"`
	MessageField string           `json:"messageField"`
	DisplayField string           `json:"displayField"`
	Store        string           `json:"store"`
	Tenant       string           `json:"tenant"`
	Edition      string           `json:"edition"`
}

// The language plugins added by the operator in stage 2. lang_en is the
// English plugin under test; lang_de is another language; partner.lang_en is
// another English provider (for a different message, so no override is
// needed).
var (
	LangEN     = plugin.Identity{ID: "std.lang_en", Version: "1.0.0"}
	LangDE     = plugin.Identity{ID: "std.lang_de", Version: "1.0.0"}
	PartnerEN  = plugin.Identity{ID: "partner.lang_en", Version: "1.0.0"}
	Languages  = []plugin.Identity{LangEN, LangDE, PartnerEN}
	ProviderBy = map[string]plugin.Identity{"std.lang_en": LangEN, "std.lang_de": LangDE, "partner.lang_en": PartnerEN}
)

// DefaultUnified is the identity set of the through record.
func DefaultUnified() Unified {
	return Unified{
		Workspace:    "tei_reference",
		Environment:  "development",
		Application:  "order_entry",
		Owner:        plugin.Identity{ID: "sales.order_entry", Version: "1.0.0"},
		Message:      plugin.Identity{ID: "std.message", Version: "1.0.0"},
		SourceLocale: "ja-JP",
		Scope:        "tei_reference",
		MessageField: "message-body", // fixed inside reference/message-localization
		DisplayField: "display-name", // the only field reference/multilingual-overlays accepts
		Store:        "translations",
		Tenant:       "order_entry",
		Edition:      "catalog",
	}
}

// SubjectKind maps the source declaration's kind to the Derived Index kind
// (internal/modelref/derivedbridge). Display names of data and function names
// of capabilities are both the overlay's display-name field.
func SubjectKind(kind string) string {
	switch kind {
	case "data":
		return "semantic-data"
	case "capability":
		return "semantic-capability"
	}
	return ""
}

// SubjectKey is the typed key of a display target in the unified scope.
func (u Unified) SubjectKey(kind, id string) modelref.Key {
	return modelref.Key{Scope: u.Scope, Subject: subject.Subject{Kind: SubjectKind(kind), ID: id}}
}

// AssetIdentity is one row of the identifier map: what an existing asset
// used on its own and what the through record uses instead.
type AssetIdentity struct {
	Identifier string `json:"identifier"`
	Asset      string `json:"asset"`
	AssetValue string `json:"assetValue"`
	Through    string `json:"through"`
	Source     string `json:"source"` // observed-by-call, read-from-file, read-from-test-source
}
