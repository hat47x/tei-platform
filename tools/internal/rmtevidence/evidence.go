package rmtevidence

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"

	"tei/internal/sourceindex"
)

const Schema = "tei.reference.rmt-transcription-candidate-evidence/v0"

type MappingStatus string

const (
	MappingMapped   MappingStatus = "mapped"
	MappingUnmapped MappingStatus = "unmapped"
)

type Evidence struct {
	Schema                     string        `json:"schema"`
	SourceFormat               string        `json:"sourceFormat"`
	SourceFormatVersion        string        `json:"sourceFormatVersion"`
	SourceArtifactDigest       string        `json:"sourceArtifactDigest"`
	SourceToken                string        `json:"sourceToken"`
	SourceConstructKnown       bool          `json:"sourceConstructKnown"`
	SourceConstruct            string        `json:"sourceConstruct,omitempty"`
	MappingContractFingerprint string        `json:"mappingContractFingerprint"`
	MappingStatus              MappingStatus `json:"mappingStatus"`
	Classification             []string      `json:"classification"`
	Extension                  string        `json:"extension,omitempty"`
	AuthorityLimits            []string      `json:"authorityLimits"`
	EvidenceFingerprint        string        `json:"evidenceFingerprint"`
}

var fingerprintPattern = regexp.MustCompile(`^sha256:[0-9a-f]{64}package rmtevidence

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"

	"tei/internal/sourceindex"
)

const Schema = "tei.reference.rmt-transcription-candidate-evidence/v0"

type MappingStatus string

const (
	MappingMapped   MappingStatus = "mapped"
	MappingUnmapped MappingStatus = "unmapped"
)

type Evidence struct {
	Schema                     string        `json:"schema"`
	SourceFormat               string        `json:"sourceFormat"`
	SourceFormatVersion        string        `json:"sourceFormatVersion"`
	SourceArtifactDigest       string        `json:"sourceArtifactDigest"`
	SourceToken                string        `json:"sourceToken"`
	SourceConstructKnown       bool          `json:"sourceConstructKnown"`
	SourceConstruct            string        `json:"sourceConstruct,omitempty"`
	MappingContractFingerprint string        `json:"mappingContractFingerprint"`
	MappingStatus              MappingStatus `json:"mappingStatus"`
	Classification             []string      `json:"classification"`
	Extension                  string        `json:"extension,omitempty"`
	AuthorityLimits            []string      `json:"authorityLimits"`
	EvidenceFingerprint        string        `json:"evidenceFingerprint"`
}

)

var sourceFormatVersionPattern = regexp.MustCompile(`^[0-9]+(?:\.[0-9]+)*package rmtevidence

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"

	"tei/internal/sourceindex"
)

const Schema = "tei.reference.rmt-transcription-candidate-evidence/v0"

type MappingStatus string

const (
	MappingMapped   MappingStatus = "mapped"
	MappingUnmapped MappingStatus = "unmapped"
)

type Evidence struct {
	Schema                     string        `json:"schema"`
	SourceFormat               string        `json:"sourceFormat"`
	SourceFormatVersion        string        `json:"sourceFormatVersion"`
	SourceArtifactDigest       string        `json:"sourceArtifactDigest"`
	SourceToken                string        `json:"sourceToken"`
	SourceConstructKnown       bool          `json:"sourceConstructKnown"`
	SourceConstruct            string        `json:"sourceConstruct,omitempty"`
	MappingContractFingerprint string        `json:"mappingContractFingerprint"`
	MappingStatus              MappingStatus `json:"mappingStatus"`
	Classification             []string      `json:"classification"`
	Extension                  string        `json:"extension,omitempty"`
	AuthorityLimits            []string      `json:"authorityLimits"`
	EvidenceFingerprint        string        `json:"evidenceFingerprint"`
}

)

var standardConstructsByFormat = map[string]map[string]struct{}{
	"xead": stringSet(
		"System", "Subsystem", "Department", "DataType",
		"Table", "TableField", "TableKey", "TableKeyField", "Relationship",
		"FunctionType", "Function", "FunctionUsedByThis", "IOTable",
		"IOPanel", "IOPanelField", "IOPanelStyle", "IOSpool", "IOSpoolStyle",
		"Role", "Task", "TaskAction", "TaskFunctionIO", "SubjectArea",
		"DataflowNode", "DataflowLine", "Instance",
		"TableType", "SubsystemTable", "SubsystemRelationship", "MaintenanceLog",
	),
	"xeaf": stringSet(
		"System", "Subsystem", "Table", "Function", "Script",
		"Field", "Column", "Button", "Filter", "Refer", "Detail", "Key", "Menu",
		"HeaderPhrase", "Phrase", "Option", "PrintFont", "BatchField",
		"AddRowListColumn", "AddRowListButton", "MaintenanceLog",
	),
}

var allowedClassifications = stringSet(
	"canonical-candidate",
	"derived-projection",
	"native-preserved",
	"presentation-only",
	"provenance",
	"realization",
	"source-metadata",
	"unresolved",
)

var RequiredAuthorityLimits = []string{
	"source-observation-is-not-canonical-authority",
	"mapping-is-candidate-not-semantic-acceptance",
	"rmt-evidence-does-not-grant-runtime-authority",
	"rmt-evidence-content-fingerprint-is-not-observer-conformance",
}

func Decode(data []byte) (Evidence, error) {
	var value Evidence
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return value, fmt.Errorf("RMT transcription EvidenceをJSONとして読めません: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err == io.EOF {
		// expected
	} else if err != nil {
		return value, err
	} else {
		return value, fmt.Errorf("RMT transcription EvidenceのJSONの後ろに余分な内容があります")
	}
	return Verify(value)
}

func Prepare(value Evidence) (Evidence, error) {
	normalized, err := normalize(value)
	if err != nil {
		return normalized, err
	}
	normalized.EvidenceFingerprint = Fingerprint(normalized)
	return normalized, nil
}

func Verify(value Evidence) (Evidence, error) {
	recorded := strings.TrimSpace(value.EvidenceFingerprint)
	normalized, err := normalize(value)
	if err != nil {
		return normalized, err
	}
	if !fingerprintPattern.MatchString(recorded) {
		return normalized, fmt.Errorf("RMT transcription Evidence.evidenceFingerprint %q は exact sha256 fingerprintではありません", recorded)
	}
	calculated := Fingerprint(normalized)
	if recorded != calculated {
		return normalized, fmt.Errorf(
			"RMT transcription Evidence fingerprintが内容と一致しません: recorded=%s calculated=%s",
			recorded,
			calculated,
		)
	}
	normalized.EvidenceFingerprint = recorded
	return normalized, nil
}

func CanonicalBytes(value Evidence) ([]byte, error) {
	verified, err := Verify(value)
	if err != nil {
		return nil, err
	}
	return json.Marshal(verified)
}

func Fingerprint(value Evidence) string {
	value.EvidenceFingerprint = ""
	encoded, _ := json.Marshal(value)
	return sourceindex.Fingerprint(encoded)
}

func normalize(value Evidence) (Evidence, error) {
	value.Schema = strings.TrimSpace(value.Schema)
	value.SourceFormat = strings.TrimSpace(value.SourceFormat)
	value.SourceFormatVersion = strings.TrimSpace(value.SourceFormatVersion)
	value.SourceArtifactDigest = strings.TrimSpace(value.SourceArtifactDigest)
	value.SourceToken = strings.TrimSpace(value.SourceToken)
	value.SourceConstruct = strings.TrimSpace(value.SourceConstruct)
	value.MappingContractFingerprint = strings.TrimSpace(value.MappingContractFingerprint)
	value.Extension = strings.TrimSpace(value.Extension)
	value.EvidenceFingerprint = strings.TrimSpace(value.EvidenceFingerprint)

	if value.Schema != Schema {
		return value, fmt.Errorf("RMT transcription Evidence schema %q は未対応です", value.Schema)
	}
	switch value.SourceFormat {
	case "xead", "xeaf":
	default:
		return value, fmt.Errorf("RMT transcription Evidence.sourceFormat %q は xead / xeaf のいずれかである必要があります", value.SourceFormat)
	}
	if !sourceFormatVersionPattern.MatchString(value.SourceFormatVersion) {
		return value, fmt.Errorf(
			"RMT transcription Evidence.sourceFormatVersion %q は公開Evidenceで扱える数値versionではありません",
			value.SourceFormatVersion,
		)
	}
	for _, item := range []struct {
		name  string
		value string
	}{
		{"sourceArtifactDigest", value.SourceArtifactDigest},
		{"sourceToken", value.SourceToken},
		{"mappingContractFingerprint", value.MappingContractFingerprint},
	} {
		if !fingerprintPattern.MatchString(item.value) {
			return value, fmt.Errorf("RMT transcription Evidence.%s %q は exact sha256 fingerprintではありません", item.name, item.value)
		}
	}

	switch value.MappingStatus {
	case MappingMapped, MappingUnmapped:
	default:
		return value, fmt.Errorf("RMT transcription Evidence.mappingStatus %q は mapped / unmapped のいずれかである必要があります", value.MappingStatus)
	}

	if value.SourceConstructKnown {
		if value.SourceConstruct == "" {
			return value, fmt.Errorf("known source constructにはsourceConstructが必要です")
		}
		known := standardConstructsByFormat[value.SourceFormat]
		if _, ok := known[value.SourceConstruct]; !ok {
			return value, fmt.Errorf(
				"sourceConstruct %q はsourceFormat %sの公開可能な標準constructではありません",
				value.SourceConstruct,
				value.SourceFormat,
			)
		}
	} else if value.SourceConstruct != "" {
		return value, fmt.Errorf("unknown source constructの名前をRMT transcription Evidenceへ公開できません")
	}

	classification, err := normalizeStrings("classification", value.Classification)
	if err != nil {
		return value, err
	}
	for _, item := range classification {
		if _, ok := allowedClassifications[item]; !ok {
			return value, fmt.Errorf("RMT transcription Evidence.classification %q は未対応です", item)
		}
	}
	value.Classification = classification
	limits, err := normalizeStrings("authorityLimits", append(append([]string(nil), value.AuthorityLimits...), RequiredAuthorityLimits...))
	if err != nil {
		return value, err
	}
	value.AuthorityLimits = limits

	if value.MappingStatus == MappingMapped {
		if !value.SourceConstructKnown {
			return value, fmt.Errorf("unknown source constructをmapped Evidenceへ昇格できません")
		}
		if value.Extension == "" {
			return value, fmt.Errorf("mapped EvidenceにはExtensionが必要です")
		}
	} else {
		if value.Extension != "" {
			return value, fmt.Errorf("unmapped EvidenceへExtensionを割り当てられません")
		}
		if len(value.Classification) != 1 || value.Classification[0] != "unresolved" {
			return value, fmt.Errorf("unmapped Evidenceのclassificationはunresolvedだけである必要があります")
		}
	}

	return value, nil
}

func normalizeStrings(name string, values []string) ([]string, error) {
	if len(values) == 0 {
		return nil, fmt.Errorf("RMT transcription Evidence.%s が空です", name)
	}
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for i, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			return nil, fmt.Errorf("RMT transcription Evidence.%s[%d] が空です", name, i)
		}
		if _, duplicate := seen[value]; duplicate {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out, nil
}

func stringSet(values ...string) map[string]struct{} {
	out := make(map[string]struct{}, len(values))
	for _, value := range values {
		out[value] = struct{}{}
	}
	return out
}
