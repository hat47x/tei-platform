// Package orderdatarealization runs the technical validation sample
// order-data-realization (sample 3) as one chain: an attribute added to a
// copied workspace travels, bound to one model revision, through the design
// question, the ER/table cross view, the standard function candidates, the
// migration of a real PostgreSQL database, the list and header-detail reads,
// the registration with an independent re-read, and a round trip through
// SQLite.
//
// The package is sample specific and stays out of internal/. Every gap of the
// platform that the chain has to bridge is recorded in the chain record.
package orderdatarealization

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

//go:embed declarations.json
var declarationsJSON []byte

// Declarations are the sample's own declarations for what the platform has no
// place for yet. They are not accepted meaning.
type Declarations struct {
	Schema         string            `json:"schema"`
	Note           string            `json:"note"`
	Attribute      AttributeDecl     `json:"attributeAddition"`
	BusinessKey    BusinessKeyDecl   `json:"businessKey"`
	UniqueDecision UniqueDecl        `json:"uniqueDecision"`
	Migration      MigrationDecl     `json:"migration"`
	Numbering      NumberingDecl     `json:"numbering"`
	SQLite         SQLiteDialectDecl `json:"sqliteDialect"`
	SQLiteNaive    SQLiteDialectDecl `json:"sqliteNaiveDialect"`
}

type AttributeDecl struct {
	Data              string  `json:"data"`
	Symbol            string  `json:"symbol"`
	Attribute         string  `json:"attribute"`
	Type              string  `json:"type"`
	Column            string  `json:"column"`
	Storage           string  `json:"storage"`
	Nullability       string  `json:"nullability"`
	ExistingRowsValue *string `json:"existingRowsValue"`
}

type BusinessKeyDecl struct {
	Data       string   `json:"data"`
	ID         string   `json:"id"`
	Symbol     string   `json:"symbol"`
	Attributes []string `json:"attributes"`
	Question   string   `json:"question"`
	Answer     string   `json:"answer"`
}

type UniqueDecl struct {
	Declare    bool     `json:"declare"`
	Mapping    string   `json:"mapping"`
	Name       string   `json:"name"`
	Attributes []string `json:"attributes"`
}

type MigrationStageDecl struct {
	Name                  string   `json:"name"`
	ChangeKinds           []string `json:"changeKinds"`
	RefuseWhileOldWriters bool     `json:"refuseWhileOldWriters"`
}

type MigrationDecl struct {
	Stages                  []MigrationStageDecl `json:"stages"`
	LockTimeoutMillis       int                  `json:"lockTimeoutMillis"`
	OldWriterValues         map[string]*string   `json:"oldWriterValues"`
	NotAppliedAutomatically []string             `json:"notAppliedAutomatically"`
}

type NumberingDecl struct {
	OrderNumber struct {
		Method string `json:"method"`
		Gaps   string `json:"gaps"`
	} `json:"orderNumber"`
	ExternalNumber struct {
		Source            string `json:"source"`
		OnUnavailable     string `json:"onUnavailable"`
		ProvisionalPrefix string `json:"provisionalPrefix"`
	} `json:"externalNumber"`
}

type KnownDifference struct {
	Storage string `json:"storage"`
	Kind    string `json:"kind"`
	Detail  string `json:"detail"`
}

type SQLiteDialectDecl struct {
	Note             string            `json:"note,omitempty"`
	Placement        string            `json:"placement,omitempty"`
	Storage          map[string]string `json:"storage"`
	Encoding         map[string]string `json:"encoding,omitempty"`
	KnownDifferences []KnownDifference `json:"knownDifferences,omitempty"`
}

// Policy values for the external number issuer.
const (
	PolicyProvisional = "provisional-then-reassign"
	PolicyStop        = "stop"
)

// LoadDeclarations decodes the embedded declarations strictly and returns them
// with the SHA-256 of their bytes.
func LoadDeclarations() (Declarations, string, error) {
	return decodeDeclarations(declarationsJSON)
}

func decodeDeclarations(raw []byte) (Declarations, string, error) {
	var d Declarations
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&d); err != nil {
		return Declarations{}, "", fmt.Errorf("declarations: %w", err)
	}
	if d.Schema != "tei.reference.order-data-realization-declarations/v0" {
		return Declarations{}, "", fmt.Errorf("declarations: unexpected schema %q", d.Schema)
	}
	switch d.Numbering.ExternalNumber.OnUnavailable {
	case PolicyProvisional, PolicyStop:
	default:
		return Declarations{}, "", fmt.Errorf("declarations: unknown external number policy %q", d.Numbering.ExternalNumber.OnUnavailable)
	}
	if len(d.Migration.Stages) == 0 || d.Migration.LockTimeoutMillis <= 0 {
		return Declarations{}, "", fmt.Errorf("declarations: migration stages and lock timeout are required")
	}
	for _, s := range []string{"int64", "varchar", "decimal", "datetime-tz"} {
		if d.SQLite.Storage[s] == "" || d.SQLiteNaive.Storage[s] == "" {
			return Declarations{}, "", fmt.Errorf("declarations: SQLite storage %s is not declared", s)
		}
	}
	sum := sha256.Sum256(raw)
	return d, "sha256:" + hex.EncodeToString(sum[:]), nil
}

// stageOf returns the declared migration stage for a change kind.
func (m MigrationDecl) stageOf(kind string) (MigrationStageDecl, bool) {
	for _, s := range m.Stages {
		for _, k := range s.ChangeKinds {
			if k == kind {
				return s, true
			}
		}
	}
	return MigrationStageDecl{}, false
}
