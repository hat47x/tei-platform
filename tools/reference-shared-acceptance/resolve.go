package main

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"tei/internal/core"
	"tei/internal/knowledgecontext"
	"tei/internal/sourceindex"
	"tei/internal/subject"
	"tei/internal/syntax"
)

var (
	semanticFilePattern    = regexp.MustCompile(`^semantic/(?:[^/]+/)*[^/]+\.tei$`)
	applicationFilePattern = regexp.MustCompile(`^applications/([^/]+)/application\.tei$`)
	patternFilePattern     = regexp.MustCompile(`^applications/([^/]+)/patterns/[^/]+\.tei$`)
)

// worldIndexは、resolveが「既に存在するか」を確認するための、Canonical Worldのidentity一覧である。
type worldIndex struct {
	namespaces   map[core.CanonicalID]bool
	dataIDs      map[core.CanonicalID]bool
	validations  map[core.CanonicalID]bool
	attributes   map[core.CanonicalID]bool
	relations    map[core.CanonicalID]bool
	applications map[core.CanonicalID]string
	patterns     map[core.CanonicalID]bool
	subjects     map[string]bool
	// environmentsはEnvironment IDからそれを宣言したsource file、
	// environmentBindingsは既存binding（`<env>/<app>.<alias>`）の集合である。
	environments        map[core.CanonicalID]string
	environmentBindings map[string]bool
	// applicationValidationsは既存の`uses validation`宣言（`<application>/<validation>`）の集合である。
	applicationValidations map[string]bool
}

func indexWorld(s snapshot) worldIndex {
	idx := worldIndex{
		namespaces:   map[core.CanonicalID]bool{},
		dataIDs:      map[core.CanonicalID]bool{},
		validations:  map[core.CanonicalID]bool{},
		attributes:   map[core.CanonicalID]bool{},
		relations:    map[core.CanonicalID]bool{},
		applications: map[core.CanonicalID]string{},
		patterns:     map[core.CanonicalID]bool{},
		subjects:     map[string]bool{},

		environments:        map[core.CanonicalID]string{},
		environmentBindings: map[string]bool{},

		applicationValidations: map[string]bool{},
	}
	for _, env := range s.loaded.World.Environments {
		if location, ok := s.loaded.Sources["environment:"+string(env.ID)]; ok {
			idx.environments[env.ID] = location.File
		}
		idx.subjects[string(env.ID)] = true
		for _, binding := range env.Bindings {
			idx.environmentBindings[string(env.ID)+"/"+string(binding.Application)+"."+string(binding.Alias)] = true
		}
	}
	world := s.loaded.World
	for _, ns := range world.Semantics {
		idx.namespaces[ns.ID] = true
		idx.subjects[string(ns.ID)] = true
		for _, t := range ns.Types {
			idx.subjects[string(t.ID)] = true
		}
		for _, v := range ns.Validations {
			idx.validations[v.ID] = true
			idx.subjects[string(v.ID)] = true
		}
		for _, c := range ns.Capabilities {
			idx.subjects[string(c.ID)] = true
		}
		for _, d := range ns.Data {
			idx.dataIDs[d.ID] = true
			idx.subjects[string(d.ID)] = true
			for _, a := range d.Attributes {
				idx.attributes[a.ID] = true
				idx.subjects[string(a.ID)] = true
			}
			for _, r := range d.Relations {
				idx.relations[r.ID] = true
				idx.subjects[string(r.ID)] = true
			}
		}
	}
	for _, app := range world.Applications {
		idx.applications[app.ID] = s.loaded.Sources["application:"+string(app.ID)].File
		idx.subjects[string(app.ID)] = true
		for _, id := range app.Validations {
			idx.applicationValidations[string(app.ID)+"/"+string(id)] = true
		}
		for _, p := range app.Patterns {
			idx.patterns[p.ID] = true
			idx.subjects[string(p.ID)] = true
		}
	}
	return idx
}

// resolveCandidateは、candidateが指す対象・追加するidの未使用・placementの規則・Evidenceの実在を、
// sourceを書き換える前に確認する。Application / Pattern / Dataの参照先が意味として正しいかは、
// overlay適用後の共有Validationが判断する（別のvalidatorを持たない）。
func resolveCandidate(s snapshot, c Candidate) []Diagnostic {
	var out []Diagnostic
	idx := indexWorld(s)
	fingerprints := s.fingerprints()

	deltaApplications := map[core.CanonicalID]string{}
	for _, op := range c.Delta {
		switch op.Op {
		case opAddApplication:
			deltaApplications[core.CanonicalID(op.Application.ID)] = op.Placement
		}
	}
	// 同じdeltaで追加するValidationは、`uses validation`の参照先として既に存在するものと同じに扱う（ADR-0029の確かめ方1）。
	// ここで確かめるのは、参照先のIDが基準のWorldのValidation、または同じdeltaのadd-validationのIDと完全一致することだけである。
	// overlay適用後のreloadとvalidate（UNKNOWN_APPLICATION_VALIDATION）は別の確認で、多層防御として残す。
	deltaValidations := map[core.CanonicalID]bool{}
	for _, op := range c.Delta {
		if op.Op == opAddValidation {
			deltaValidations[core.CanonicalID(op.ID)] = true
		}
	}

	planned := map[string]string{}
	seenNamespaces := map[core.CanonicalID]bool{}
	for _, op := range c.Delta {
		key := operationKey(op)
		diag := func(code, format string, args ...any) {
			out = append(out, Diagnostic{Code: code, Message: fmt.Sprintf(format, args...), Subject: key})
		}
		switch op.Op {
		case opAddValidation:
			if !idx.namespaces[core.CanonicalID(op.Namespace)] {
				diag("NAMESPACE_NOT_FOUND", "semantic namespace %s は存在しません", op.Namespace)
			}
			if idx.validations[core.CanonicalID(op.ID)] {
				diag("VALIDATION_ALREADY_EXISTS", "validation %s は既に存在します", op.ID)
			}
			if !idx.attributes[core.CanonicalID(op.Attribute)] {
				diag("ATTRIBUTE_NOT_FOUND", "attribute %s は存在しません", op.Attribute)
			}
			if op.Relation != "" && !idx.relations[core.CanonicalID(op.Relation)] {
				diag("RELATION_NOT_FOUND", "relation %s は存在しません", op.Relation)
			}
			out = append(out, checkPlacementOwnership(s, op)...)

		case opAddData:
			ns := core.CanonicalID(op.Data.Namespace)
			if idx.namespaces[ns] || seenNamespaces[ns] {
				diag("NAMESPACE_ALREADY_EXISTS", "semantic namespace %s は既に存在します（このReferenceは新しいnamespaceへのData追加だけを扱います）", ns)
			}
			seenNamespaces[ns] = true
			out = append(out, checkNewFile(s, fingerprints, planned, key, op.Placement, layerSemantic, semanticFilePattern, "semantic/<...>/<file>.tei")...)

		case opAddEnvironmentBinding:
			// 層をまたぐ正しさ（unknown application / alias / instance、target mismatch、duplicate）は
			// 共有Validationが見る。ここではEnvironmentの存在とplacementの所有関係だけを確認する。
			b := op.EnvironmentBinding
			file, exists := idx.environments[core.CanonicalID(b.Environment)]
			if !exists {
				diag("ENVIRONMENT_NOT_FOUND", "environment %s は存在しません", b.Environment)
				break
			}
			if _, ok := fingerprints[op.Placement]; !ok {
				diag("PLACEMENT_NOT_MANAGED_SOURCE", "placement %s は管理対象sourceではありません", op.Placement)
				break
			}
			if op.Placement != file {
				diag("PLACEMENT_NOT_OWNED", "environment %s を宣言しているのは %s で、placement %s ではありません", b.Environment, file, op.Placement)
			}
			if idx.environmentBindings[b.Environment+"/"+b.Application+"."+b.Alias] {
				diag("ENVIRONMENT_BINDING_ALREADY_EXISTS", "environment %s には %s.%s のbindingが既にあります", b.Environment, b.Application, b.Alias)
			}

		case opAddApplicationValidationUse:
			// そのnamespaceをApplicationが使っているか等、層をまたぐ正しさは共有Validationが見る。
			// ここでは対象の存在、重複宣言、placementの所有関係だけを確認する。
			u := op.ApplicationValidationUse
			file, exists := idx.applications[core.CanonicalID(u.Application)]
			if !exists {
				diag("APPLICATION_NOT_FOUND", "application %s は存在しません（%s は既存のApplicationへ宣言を追加します）", u.Application, opAddApplicationValidationUse)
				break
			}
			if !idx.validations[core.CanonicalID(u.Validation)] && !deltaValidations[core.CanonicalID(u.Validation)] {
				diag("VALIDATION_NOT_FOUND", "validation %s は存在しません", u.Validation)
			}
			if _, ok := fingerprints[op.Placement]; !ok {
				diag("PLACEMENT_NOT_MANAGED_SOURCE", "placement %s は管理対象sourceではありません", op.Placement)
				break
			}
			if op.Placement != file {
				diag("PLACEMENT_NOT_OWNED", "application %s を宣言しているのは %s で、placement %s ではありません", u.Application, file, op.Placement)
			}
			if idx.applicationValidations[u.Application+"/"+u.Validation] {
				diag("APPLICATION_VALIDATION_USE_ALREADY_EXISTS", "application %s は`uses validation %s`を既に宣言しています", u.Application, u.Validation)
			}
		case opAddRelation:
			if !idx.dataIDs[core.CanonicalID(op.AddRelation.Data)] {
				diag("DATA_NOT_FOUND", "data %s は存在しません（add-relationは既存のDataへRelationを追加します）", op.AddRelation.Data)
			}
			if idx.relations[core.CanonicalID(op.AddRelation.ID)] {
				diag("RELATION_ALREADY_EXISTS", "relation %s は既に存在します", op.AddRelation.ID)
			}

		case opAddApplication:
			if _, exists := idx.applications[core.CanonicalID(op.Application.ID)]; exists {
				diag("APPLICATION_ALREADY_EXISTS", "application %s は既に存在します", op.Application.ID)
			}
			out = append(out, checkNewFile(s, fingerprints, planned, key, op.Placement, layerApplication, applicationFilePattern, "applications/<dir>/application.tei")...)

		case opAddReferencePattern:
			if idx.patterns[core.CanonicalID(op.Pattern.ID)] {
				diag("PATTERN_ALREADY_EXISTS", "pattern %s は既に存在します", op.Pattern.ID)
			}
			out = append(out, checkNewFile(s, fingerprints, planned, key, op.Placement, layerApplication, patternFilePattern, "applications/<dir>/patterns/<file>.tei")...)
			out = append(out, checkPatternOwner(key, op, idx, deltaApplications)...)
		}
	}

	out = append(out, validateEvidence(s, c.Evidence)...)
	return out
}

func validateEvidence(s snapshot, evidence []Evidence) []Diagnostic {
	var out []Diagnostic
	idx := indexWorld(s)
	fingerprints := s.fingerprints()
	for i, e := range evidence {
		if e.Artifact != nil {
			for _, diagnostic := range validateEvidenceArtifact(e.Artifact) {
				diagnostic.Message = fmt.Sprintf("evidence[%d].artifact: %s", i, diagnostic.Message)
				out = append(out, diagnostic)
			}
			continue
		}
		if e.Subject != "" {
			if e.SubjectKind == "" {
				if !idx.subjects[e.Subject] {
					out = append(out, Diagnostic{Code: "EVIDENCE_SUBJECT_NOT_FOUND", Message: fmt.Sprintf("evidence[%d].subject %s はCanonical Worldに存在しません", i, e.Subject)})
				}
			} else {
				exact := sourceindex.Subject{Kind: e.SubjectKind, ID: e.Subject}
				refs := s.index.ReferencesFor(exact)
				if len(refs) == 0 {
					out = append(out, Diagnostic{Code: "EVIDENCE_EXACT_SUBJECT_NOT_FOUND", Message: fmt.Sprintf("evidence[%d]のexact subject %s:%s はcurrent Source Indexに存在しません", i, e.SubjectKind, e.Subject)})
				} else if e.Source != "" {
					matched := false
					for _, ref := range refs {
						if ref.ArtifactPath == e.Source {
							matched = true
							break
						}
					}
					if !matched {
						out = append(out, Diagnostic{Code: "EVIDENCE_SUBJECT_SOURCE_MISMATCH", Message: fmt.Sprintf("evidence[%d]のexact subject %s:%s はsource %sにより宣言・authorされていません", i, e.SubjectKind, e.Subject, e.Source)})
					}
				}
			}
		}
		if e.Source != "" {
			current, ok := fingerprints[e.Source]
			if !ok {
				out = append(out, Diagnostic{Code: "EVIDENCE_SOURCE_NOT_FOUND", Message: fmt.Sprintf("evidence[%d].source %s は管理対象sourceではありません", i, e.Source)})
				continue
			}
			if e.SourceFingerprint != "" && e.SourceFingerprint != current {
				out = append(out, Diagnostic{Code: "EVIDENCE_SOURCE_FINGERPRINT_MISMATCH", Message: fmt.Sprintf("evidence[%d].sourceFingerprintはcurrent sourceと一致しません: source=%s have=%s current=%s", i, e.Source, e.SourceFingerprint, current)})
			}
		}
		if e.QualificationRequirements != nil {
			ctx := knowledgecontext.Build(
				s.loaded.World,
				s.index,
				subject.Subject{Kind: e.SubjectKind, ID: e.Subject},
			)
			qualified, err := knowledgecontext.QualifyWithSources(ctx, s.index, *e.QualificationRequirements)
			if err != nil {
				out = append(out, Diagnostic{
					Code:    "EVIDENCE_QUALIFICATION_INVALID",
					Message: fmt.Sprintf("evidence[%d]のqualification requirementsを評価できません: %v", i, err),
				})
				continue
			}
			if qualified.Status != knowledgecontext.QualificationApplicable {
				out = append(out, Diagnostic{
					Code:    "EVIDENCE_QUALIFICATION_NOT_APPLICABLE",
					Message: fmt.Sprintf("evidence[%d]のexact subject %s:%s はcurrent contextで%sです: %+v", i, e.SubjectKind, e.Subject, qualified.Status, qualified.Reasons),
				})
				continue
			}
			matchedQualifiedEvidence := false
			for _, currentEvidence := range qualified.Evidence {
				if currentEvidence.SubjectKind == e.SubjectKind &&
					currentEvidence.Subject == e.Subject &&
					currentEvidence.Source == e.Source &&
					currentEvidence.SourceFingerprint == e.SourceFingerprint {
					matchedQualifiedEvidence = true
					break
				}
			}
			if !matchedQualifiedEvidence {
				out = append(out, Diagnostic{
					Code:    "EVIDENCE_QUALIFICATION_PROVENANCE_MISMATCH",
					Message: fmt.Sprintf("evidence[%d]のsource / fingerprintはcurrent qualificationが返したauthoritative Evidenceと一致しません", i),
				})
			}
			if !sameQualificationEvidence(qualified.DependencyEvidence, e.QualificationDependencyEvidence) {
				out = append(out, Diagnostic{
					Code:    "EVIDENCE_QUALIFICATION_DEPENDENCY_STALE",
					Message: fmt.Sprintf("evidence[%d]のrequired dependency Evidenceはcurrent qualificationと一致しません", i),
				})
			}
		}
	}
	return out
}

// checkNewFileは、新しいfileを作るoperationのplacementが、意味を所有する層のpath形式に合い、
// まだ存在せず、他のoperationと衝突しないことを確認する。
func checkNewFile(s snapshot, fingerprints map[string]string, planned map[string]string, key, placement, layer string, shape *regexp.Regexp, shapeText string) []Diagnostic {
	if !strings.HasPrefix(placement, layerPrefix(layer)) {
		return []Diagnostic{{
			Code:    "LAYER_VIOLATION",
			Message: fmt.Sprintf("この意味は%s層（%s）でauthorします。placement %s は別の層です", layer, layerPrefix(layer), placement),
			Subject: key,
		}}
	}
	if !shape.MatchString(placement) {
		return []Diagnostic{{
			Code:    "PLACEMENT_SHAPE_INVALID",
			Message: fmt.Sprintf("placement %s はloaderが読む形式（%s）ではありません", placement, shapeText),
			Subject: key,
		}}
	}
	var out []Diagnostic
	if _, managed := fingerprints[placement]; managed || fileExists(s.root, placement) {
		out = append(out, Diagnostic{Code: "DESTINATION_ALREADY_EXISTS", Message: fmt.Sprintf("作成先 %s は既に存在します", placement), Subject: key})
	}
	if other, ok := planned[placement]; ok {
		out = append(out, Diagnostic{Code: "PLACEMENT_COLLISION", Message: fmt.Sprintf("placement %s は %s とも重なっています", placement, other), Subject: key})
	}
	planned[placement] = key
	return out
}

// checkPatternOwnerは、PatternのplacementがそのPatternを所有するApplicationのdirectoryにあることを確認する。
// loaderはApplication manifestと同じdirectoryのpatterns/にあるPatternをそのApplicationのものとして読む。
func checkPatternOwner(key string, op Operation, idx worldIndex, deltaApplications map[core.CanonicalID]string) []Diagnostic {
	owner := core.CanonicalID(op.Pattern.Application)
	manifest, exists := idx.applications[owner]
	if !exists {
		manifest, exists = deltaApplications[owner]
	}
	if !exists {
		return []Diagnostic{{
			Code:    "PATTERN_OWNER_APPLICATION_NOT_FOUND",
			Message: fmt.Sprintf("pattern %s のowner application %s は、既存のWorldにも同じdeltaにもありません", op.Pattern.ID, owner),
			Subject: key,
		}}
	}
	match := patternFilePattern.FindStringSubmatch(op.Placement)
	if match == nil {
		return nil
	}
	if path.Dir(manifest) != "applications/"+match[1] {
		return []Diagnostic{{
			Code:    "PATTERN_OWNER_DIRECTORY_MISMATCH",
			Message: fmt.Sprintf("pattern %s はowner application %s のdirectory（%s）ではなく applications/%s に置かれています", op.Pattern.ID, owner, path.Dir(manifest), match[1]),
			Subject: key,
		}}
	}
	return nil
}

// checkPlacementOwnershipは、add-validationの明示placementがValidationを所有するsemantic層のfileであること、
// そのfileが同じnamespaceを宣言していることを、sourceを書き換える前に確認する。
func checkPlacementOwnership(s snapshot, op Operation) []Diagnostic {
	if op.Placement == "" {
		return nil
	}
	key := "validation:" + op.ID
	if !strings.HasPrefix(op.Placement, layerPrefix(layerSemantic)) {
		return []Diagnostic{{
			Code:    "LAYER_VIOLATION",
			Message: fmt.Sprintf("Validationは意味を所有するsemantic層（%s）でauthorします。placement %s は別の層です", layerPrefix(layerSemantic), op.Placement),
			Subject: key,
		}}
	}
	content, err := os.ReadFile(filepath.Join(s.root, filepath.FromSlash(op.Placement)))
	if err != nil {
		return []Diagnostic{{Code: "PLACEMENT_NOT_MANAGED_SOURCE", Message: fmt.Sprintf("placement %s を読めません: %v", op.Placement, err), Subject: key}}
	}
	ns, err := syntax.NewParser(string(content)).ParseSemanticFile()
	if err != nil {
		return []Diagnostic{{Code: "PLACEMENT_NOT_SEMANTIC_FILE", Message: fmt.Sprintf("placement %s はsemantic fileとして読めません: %v", op.Placement, err), Subject: key}}
	}
	if string(ns.ID) != op.Namespace {
		return []Diagnostic{{
			Code:    "PLACEMENT_NAMESPACE_MISMATCH",
			Message: fmt.Sprintf("placement %s はnamespace %s を宣言しており、%s ではありません", op.Placement, ns.ID, op.Namespace),
			Subject: key,
		}}
	}
	return nil
}
