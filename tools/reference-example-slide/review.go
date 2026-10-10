package exampleslide

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
)

// ReviewPacket is a derived, read-only handoff for human design review.
// No Question or Issue here is an accepted semantic decision or a Patch.
type ReviewPacket struct {
	CaseID        string          `json:"caseId"`
	ModelRevision string          `json:"modelRevision"`
	Authored      []Question      `json:"authoredQuestions,omitempty"`
	Issues        []ReviewIssue   `json:"issues,omitempty"`
}

type ReviewIssue struct {
	ID              string   `json:"id"`
	Stage           string   `json:"stage"`
	FrameIDs        []string `json:"frameIds"`
	RelationID      string   `json:"relationId"`
	SourceRecordID  string   `json:"sourceRecordId"`
	Result          string   `json:"result"`
	Reason          string   `json:"reason"`
	Question        string   `json:"question"`
	EvidenceNeeded  []string `json:"evidenceNeeded"`
	ReviewTarget    string   `json:"reviewTarget"`
}

// Review builds a new projection from its input rather than trusting a
// caller-supplied diagnostic. This function never updates the TEI World.
func Review(input Input) (ReviewPacket, error) {
	projection, err := Build(input)
	if err != nil {
		return ReviewPacket{}, err
	}
	return reviewProjection(input, projection), nil
}

func reviewProjection(input Input, projection Projection) ReviewPacket {
	packet := ReviewPacket{
		CaseID: input.Example.ID, ModelRevision: input.Catalog.Revision,
		Authored: append([]Question(nil), input.Example.Questions...),
	}
	byID := map[string]int{}
	for _, slide := range projection.Slides {
		for _, finding := range slide.RelationFindings {
			if finding.Result == "supported" {
				continue
			}
			question, needed, target := proposedQuestion(finding)
			id := reviewIssueID(input.Example.ID, input.Catalog.Revision,
				slide.Stage, finding.RelationID, finding.SourceRecordID, finding.Reason)
			if index, exists := byID[id]; exists {
				packet.Issues[index].FrameIDs =
					append(packet.Issues[index].FrameIDs, slide.ID)
				continue
			}
			byID[id] = len(packet.Issues)
			packet.Issues = append(packet.Issues, ReviewIssue{
				ID: id, Stage: slide.Stage, FrameIDs: []string{slide.ID},
				RelationID: finding.RelationID,
				SourceRecordID: finding.SourceRecordID,
				Result: finding.Result, Reason: finding.Reason,
				Question: question, EvidenceNeeded: needed,
				ReviewTarget: target,
			})
		}
	}
	// Stable representation despite differences in presentation ordering.
	sort.Slice(packet.Issues, func(i, j int) bool {
		left, right := packet.Issues[i], packet.Issues[j]
		if left.Stage != right.Stage { return left.Stage < right.Stage }
		if left.RelationID != right.RelationID { return left.RelationID < right.RelationID }
		if left.SourceRecordID != right.SourceRecordID { return left.SourceRecordID < right.SourceRecordID }
		return left.ID < right.ID
	})
	for i := range packet.Issues {
		sort.Strings(packet.Issues[i].FrameIDs)
	}
	return packet
}

func reviewIssueID(caseID, revision, stage, relationID, recordID, reason string) string {
	// JSON encoding is unambiguous even if the individual IDs contain colons,
	// slashes or Unicode text. Revision changes invalidate review handoffs.
	fields, _ := json.Marshal([]string{caseID, revision, stage, relationID, recordID, reason})
	hash := sha256.Sum256(fields)
	return fmt.Sprintf("review-%s", hex.EncodeToString(hash[:12]))
}

func proposedQuestion(f RelationFinding) (string, []string, string) {
	switch f.Reason {
	case "relation-keys-not-declared":
		return "この関連で、データ間の対応を決める属性は確定していますか。キーを定義すべき関連ですか。",
			[]string{"accepted Relation/RelationKey definition", "business identity and correspondence rule"},
			"semantic-relation"
	case "source-key-missing-or-unusable":
		return "このレコードの対応キーが不足しています。Caseの省略ですか、業務上の任意値ですか。",
			[]string{"example record source", "requiredness / optionality decision"},
			"example-data"
	case "unresolved-target-key":
		return "参照先候補の対応キーが不足しています。どの値を使えば安全に照合できますか。",
			[]string{"candidate target record values", "source and target attribute definitions"},
			"example-data"
	case "target-table-is-partial":
		return "この例に含まれる参照先レコードは全件ですか。全件だと確認できる根拠はありますか。",
			[]string{"dataset extraction scope", "completeness evidence for target table"},
			"example-completeness"
	case "target-absence-requires-business-review":
		return "対応する参照先が見つかりません。関連先の存在は業務上必須ですか。",
			[]string{"business optionality rule", "example scope and intended absence"},
			"business-meaning"
	case "multiple-targets-for-one":
		return "一対一方向の関連で複数の参照先が一致します。識別方法または具体例のどちらを修正すべきですか。",
			[]string{"declared relation cardinality", "matching example records", "business identity/key evidence"},
			"cross-view-contradiction"
	default:
		// Unknown reason is still reviewable, not silently "resolved".
		return "この関連の評価を確定できません。理由と根拠を確認してください。",
			[]string{"relation finding provenance", "applicable business/source evidence"},
			"unresolved"
	}
}
