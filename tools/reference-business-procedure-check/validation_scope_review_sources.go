package main

import (
	"sort"

	"tei/internal/sourceindex"
	"tei/internal/subject"
)

// validationReviewSource は正本を複製せず、Source Indexの定義位置を参照する。
// fingerprintはファイルの指紋であり、人が内容を承認した証拠ではない。
type validationReviewSource struct {
	Kind        string `json:"kind"`
	ID          string `json:"id"`
	Path        string `json:"path"`
	Line        int    `json:"line"`
	Fingerprint string `json:"fingerprint"`
}

// validationScopeSourcesNeeded は比較の対象がある場合だけ追加のSource解析を求める。
// 文脈や参照が未解決の行は、出典探索によって意味を補完しない。
func validationScopeSourcesNeeded(rows []validationScopeReviewRow) bool {
	for _, row := range rows {
		if row.Status == "review-needed" || row.Status == "no-set-difference" {
			return true
		}
	}
	return false
}

// attachValidationScopeSources は差の原因となるApplicationとValidation定義を
// 既存Source Indexへ戻す。曖昧・欠落時に名前からSourceを補完しない。
func attachValidationScopeSources(rows []validationScopeReviewRow, index sourceindex.Index) {
	artifacts := make(map[string]string, len(index.Artifacts))
	for _, artifact := range index.Artifacts {
		if _, duplicated := artifacts[artifact.Path]; duplicated {
			// An ambiguous artifact path cannot identify a trustworthy fingerprint.
			artifacts[artifact.Path] = ""
			continue
		}
		artifacts[artifact.Path] = artifact.Fingerprint
	}
	for i := range rows {
		row := &rows[i]
		// Derived evidence is a fresh projection of this Index, not an append-only log.
		row.SourceEvidence = nil
		row.UnresolvedSource = nil
		if row.Status != "review-needed" && row.Status != "no-set-difference" {
			continue
		}

		subjects := []subject.Subject{{Kind: "application", ID: row.Application}}
		seenID := make(map[string]struct{})
		for _, id := range append(append([]string(nil), row.ApplicationOnly...), row.ProcedureOnly...) {
			if _, seen := seenID[id]; seen {
				continue
			}
			seenID[id] = struct{}{}
			subjects = append(subjects, subject.Subject{Kind: "semantic-validation", ID: id})
		}
		for _, target := range subjects {
			var definitions []sourceindex.Reference
			for _, ref := range index.ReferencesFor(target) {
				if ref.Role == sourceindex.RoleDefinition {
					definitions = append(definitions, ref)
				}
			}
			if len(definitions) != 1 {
				row.UnresolvedSource = append(row.UnresolvedSource, target.Kind+":"+target.ID)
				continue
			}
			ref := definitions[0]
			fingerprint := artifacts[ref.ArtifactPath]
			if fingerprint == "" || ref.Range.Start.Line < 1 {
				row.UnresolvedSource = append(row.UnresolvedSource, target.Kind+":"+target.ID)
				continue
			}
			row.SourceEvidence = append(row.SourceEvidence, validationReviewSource{
				Kind:        target.Kind,
				ID:          target.ID,
				Path:        ref.ArtifactPath,
				Line:        ref.Range.Start.Line,
				Fingerprint: fingerprint,
			})
		}
		sort.Slice(row.SourceEvidence, func(a, b int) bool {
			if row.SourceEvidence[a].Kind != row.SourceEvidence[b].Kind {
				return row.SourceEvidence[a].Kind < row.SourceEvidence[b].Kind
			}
			return row.SourceEvidence[a].ID < row.SourceEvidence[b].ID
		})
		sort.Strings(row.UnresolvedSource)
	}
}
