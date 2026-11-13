// Package xteathreeview is the read-only X-TEA three-view plugin.
//
// It reads one X-TEA design source of a Workspace and answers the common
// three-view navigation Operation (tei/internal/operations) with the Data,
// Function and Business subjects, their relation occurrences, the function
// level CRUD matrix and the item-level references of that source.
//
// Everything specific to the X-TEA source format lives here and in the
// source reader it uses: which construct is placed in which view, how a
// source relation is named, how a source position is addressed. The Operation
// contract, the Studio Host and the Bridge know none of it.
package xteathreeview

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"errors"
	"path"
	"sort"
	"strings"

	"tei/internal/operations"
	"tei/internal/sourceindex"
	"tei/internal/threeelement"
	"tei/internal/workspace"
	"tei/internal/xeadsource"
)

const (
	// ProviderID is the identifier a Host selects this plugin by.
	ProviderID = "tei.plugin.xtea-three-view"
	// Entry is the linked symbol of Navigate. A Host compares it with the
	// function value it is about to call.
	Entry = "tei/cmd/tei-studio/plugins/xteathreeview.Navigate"
	// Revision identifies this plugin's projection rules.
	Revision = "tei.plugin.xtea-three-view/v0"

	// SourceLimitBytes bounds one source read.
	SourceLimitBytes int64 = 4 << 20
)

//go:embed reading-rules.json
var readingRules []byte

// The classifications of reading-rules.json this plugin places in a view.
var viewByClassification = map[string]string{
	"three-view-data":     operations.ThreeViewData,
	"three-view-function": operations.ThreeViewFunction,
	"three-view-business": operations.ThreeViewBusiness,
}

const fieldClassification = "three-view-field"

// The reader's relation kinds, renamed to the vocabulary of this projection.
// A reader kind that is absent here has ends that are not three-view
// subjects; such relations are counted in the coverage and not shown.
var relationKinds = map[string]string{
	"task-action":                               "business-contains-action",
	"action-function-use":                       "action-uses-function",
	"function-table-operation":                  operations.ThreeViewRelationFunctionDataOperation,
	"action-function-table-potential-operation": "action-data-potential-operation",
}

var navigationAuthorityLimits = []string{
	"three-view-navigation-is-a-read-only-projection-not-a-canonical-model",
	"occurrence-is-an-address-in-exact-source-bytes-not-an-identity",
	"source-id-is-a-source-fact-not-a-canonical-identity",
	"view-placement-and-derived-paths-are-structural-candidates",
	"no-accepted-meaning-is-bound-by-this-projection",
	"function-level-crud-is-not-business-action-specific-crud",
	"declared-crud-is-not-an-observed-runtime-effect",
	"absence-from-this-projection-is-not-absence-from-the-system",
	"projection-does-not-authorize-edits-or-execution",
}

func failed(code string) (operations.ThreeViewNavigation, error) {
	return operations.ThreeViewNavigation{}, &operations.ThreeViewError{Code: code}
}

// Navigate answers the three-view navigation Operation for one X-TEA design
// source below trustedRoot. It reads; it writes nothing and accepts nothing.
func Navigate(trustedRoot string, request operations.ThreeViewRequest) (operations.ThreeViewNavigation, error) {
	normalized, err := sourceindex.NormalizeWorkspacePath(request.SourcePath)
	if err != nil || normalized != request.SourcePath {
		return failed(operations.ThreeViewSourceReadFailed)
	}
	if !strings.EqualFold(path.Ext(normalized), ".xead") {
		return failed(operations.ThreeViewUnsupportedSource)
	}
	content, err := workspace.ReadSource(trustedRoot, normalized, SourceLimitBytes)
	if err != nil {
		if errors.Is(err, workspace.ErrSourceTooLarge) {
			return failed(operations.ThreeViewSourceTooLarge)
		}
		return failed(operations.ThreeViewSourceReadFailed)
	}
	// A source that exceeds the tree budget is refused as a whole. Showing the
	// part that fitted would present a truncated model as the model.
	if _, err := xeadsource.ParseWithLimits(xeadsource.FormatXead, content, xeadsource.ParseLimits{
		MaxBytes: SourceLimitBytes, MaxDepth: 128, MaxElements: 100000,
	}); err != nil {
		if errors.Is(err, xeadsource.ErrSourceLimitExceeded) {
			return failed(operations.ThreeViewSourceTooLarge)
		}
		return failed(operations.ThreeViewSourceRejected)
	}
	inspection, err := xeadsource.InspectWithContract(xeadsource.FormatXead, content, readingRules)
	if err != nil {
		return failed(operations.ThreeViewSourceRejected)
	}
	projection, err := xeadsource.BuildCrossViewProjection(xeadsource.FormatXead, content, readingRules)
	if err != nil {
		return failed(operations.ThreeViewProjectionFailed)
	}
	manifest, err := workspace.ObserveManifestIdentity(trustedRoot)
	if err != nil {
		return failed(operations.ThreeViewSourceReadFailed)
	}

	nodes := xeadsource.CrossViewSourceNodes(inspection.Document)
	candidates := make(map[string]xeadsource.Candidate, len(inspection.Candidates))
	for _, candidate := range inspection.Candidates {
		candidates[candidate.Locator] = candidate
	}
	tokenByLocator := make(map[string]string, len(nodes))
	nodeByToken := make(map[string]xeadsource.CrossViewSourceNode, len(nodes))
	// A source ID singles an element out only among elements of its own kind.
	idCount := map[[2]string]int{}
	for _, node := range nodes {
		tokenByLocator[node.Locator] = node.Token
		nodeByToken[node.Token] = node
		if node.Known && node.LegacyIDPresent {
			idCount[[2]string{node.Name, node.LegacyID}]++
		}
	}
	sourceFact := func(node xeadsource.CrossViewSourceNode) operations.ThreeViewSourceFact {
		fact := operations.ThreeViewSourceFact{
			Kind: node.Name, Locator: node.Locator, Name: node.DisplayName,
			SourceIDStatus: operations.ThreeViewSourceIDMissing,
		}
		if node.LegacyIDPresent {
			fact.SourceID = node.LegacyID
			fact.SourceIDStatus = operations.ThreeViewSourceIDUnique
			if idCount[[2]string{node.Name, node.LegacyID}] > 1 {
				fact.SourceIDStatus = operations.ThreeViewSourceIDDuplicated
			}
		}
		return fact
	}

	value := operations.ThreeViewNavigation{
		SchemaVersion: operations.ThreeViewNavigationSchemaVersion,
		Effect:        "read-only",
		Workspace:     &workspace.WorkspaceIdentity{Manifest: manifest},
		Source: operations.ThreeViewSource{
			Path:                  normalized,
			Format:                string(xeadsource.FormatXead),
			ArtifactDigest:        projection.SourceArtifactDigest,
			MappingFingerprint:    projection.MappingContractFingerprint,
			DerivationRevision:    Revision + "+" + projection.DerivationPolicyRevision,
			ProjectionFingerprint: projection.ProjectionFingerprint,
			Coverage:              operations.ThreeViewCoverage{Scope: operations.ThreeViewScopeSingleSource},
		},
		Subjects:        []operations.ThreeViewSubject{},
		Relations:       []operations.ThreeViewRelation{},
		CRUD:            []operations.ThreeViewCRUDRow{},
		AuthorityLimits: append([]string(nil), navigationAuthorityLimits...),
	}
	coverage := &value.Source.Coverage
	omit := func(kind string) {
		if coverage.OmittedRelations == nil {
			coverage.OmittedRelations = map[string]int{}
		}
		coverage.OmittedRelations[kind]++
	}

	// Subjects and fields, in document order.
	subjectView := map[string]string{}
	sameName := map[[3]string][]string{}
	var fieldNodes []xeadsource.CrossViewSourceNode
	for _, node := range nodes {
		if !node.Known {
			coverage.UnknownConstructs++
			continue
		}
		candidate := candidates[node.Locator]
		if candidate.Unresolved {
			coverage.UnplacedConstructs++
			continue
		}
		view := ""
		isField := false
		for _, classification := range candidate.Classifications {
			if placed, ok := viewByClassification[classification]; ok {
				view = placed
			}
			isField = isField || classification == fieldClassification
		}
		if isField {
			fieldNodes = append(fieldNodes, node)
		}
		if view == "" {
			continue
		}
		subject := operations.ThreeViewSubject{
			Occurrence: node.Token,
			View:       view,
			SourceFact: sourceFact(node),
			Candidate: operations.ThreeViewCandidate{
				Nature:          operations.ThreeViewNatureStructuralCandidate,
				Classifications: append([]string(nil), candidate.Classifications...),
				AuthorityLimits: append([]string(nil), candidate.AuthorityLimits...),
			},
			Meaning: operations.ThreeViewMeaning{Status: operations.ThreeViewMeaningNotBound},
		}
		if len(subject.Candidate.AuthorityLimits) == 0 {
			return failed(operations.ThreeViewProjectionFailed)
		}
		if _, nested := subjectView[node.ParentToken]; nested {
			subject.ParentOccurrence = node.ParentToken
		}
		subjectView[node.Token] = view
		if node.DisplayName != "" {
			key := [3]string{view, node.Name, node.DisplayName}
			sameName[key] = append(sameName[key], node.Token)
		}
		value.Subjects = append(value.Subjects, subject)
	}
	for i := range value.Subjects {
		subject := &value.Subjects[i]
		for _, other := range sameName[[3]string{subject.View, subject.SourceFact.Kind, subject.SourceFact.Name}] {
			if other != subject.Occurrence {
				subject.SameNameOccurrences = append(subject.SameNameOccurrences, other)
			}
		}
	}
	fieldsByOwner := map[string][]operations.ThreeViewField{}
	for _, node := range fieldNodes {
		if subjectView[node.ParentToken] != operations.ThreeViewData {
			coverage.UnplacedConstructs++
			continue
		}
		field := operations.ThreeViewField{
			Occurrence: node.Token, OwnerOccurrence: node.ParentToken, SourceFact: sourceFact(node),
		}
		value.Fields = append(value.Fields, field)
		fieldsByOwner[node.ParentToken] = append(fieldsByOwner[node.ParentToken], field)
	}

	// Relations. Source-backed edges are source facts; edges the reader joined
	// from several facts are structural candidates and keep their rule.
	seenRelation := map[string]bool{}
	declared := map[[2]string][]string{}
	model := threeelement.NewThreeElementModel(Revision, normalized)
	appendEdges := func(edges []xeadsource.CrossViewEdge, nature string) error {
		for _, edge := range edges {
			kind, shown := relationKinds[edge.Kind]
			_, fromSubject := subjectView[edge.FromToken]
			_, toSubject := subjectView[edge.ToToken]
			if !shown || !fromSubject || !toSubject {
				omit(edge.Kind)
				continue
			}
			relation := operations.ThreeViewRelation{
				Kind: kind, Nature: nature, From: edge.FromToken, To: edge.ToToken,
				Operations:                edge.Operations,
				SourceOccurrences:         append([]string(nil), edge.SourceTokens...),
				DerivationRuleFingerprint: edge.DerivationRuleFingerprint,
				AuthorityLimits:           append([]string(nil), edge.AuthorityLimits...),
			}
			relation.Occurrence = relationOccurrence(relation)
			if seenRelation[relation.Occurrence] {
				return errors.New("relation occurrence is not unique")
			}
			seenRelation[relation.Occurrence] = true
			value.Relations = append(value.Relations, relation)
			if kind == operations.ThreeViewRelationFunctionDataOperation && nature == operations.ThreeViewNatureSourceFact {
				pair := [2]string{edge.FromToken, edge.ToToken}
				declared[pair] = append(declared[pair], relation.Occurrence)
				model.CRUDDefinitions = append(model.CRUDDefinitions, threeelement.CRUDDefinition{
					ID: relation.Occurrence, FunctionID: edge.FromToken, DataID: edge.ToToken,
					Create: strings.Contains(edge.Operations, "C"), Read: strings.Contains(edge.Operations, "R"),
					Update: strings.Contains(edge.Operations, "U"), Delete: strings.Contains(edge.Operations, "D"),
					Source: "declared",
				})
			}
		}
		return nil
	}
	if err := appendEdges(projection.SourceEdges, operations.ThreeViewNatureSourceFact); err != nil {
		return failed(operations.ThreeViewProjectionFailed)
	}
	if err := appendEdges(projection.DerivedEdges, operations.ThreeViewNatureStructuralCandidate); err != nil {
		return failed(operations.ThreeViewProjectionFailed)
	}

	// The CRUD matrix is the standard three-element CRUD projection over the
	// function and data occurrences. Element IDs of this throwaway model are
	// occurrences, so two elements that share a name or a source ID stay two.
	for _, subject := range value.Subjects {
		switch subject.View {
		case operations.ThreeViewData:
			model.DataElements = append(model.DataElements, threeelement.DataElement{
				ID: subject.Occurrence, Name: subject.SourceFact.Name,
			})
		case operations.ThreeViewFunction:
			model.FunctionElements = append(model.FunctionElements, threeelement.FunctionElement{
				ID: subject.Occurrence, Name: subject.SourceFact.Name,
			})
		}
	}
	for _, row := range threeelement.GenerateCRUDTable(model) {
		relations := declared[[2]string{row.FunctionID, row.DataID}]
		if len(relations) == 0 {
			return failed(operations.ThreeViewProjectionFailed)
		}
		value.CRUD = append(value.CRUD, operations.ThreeViewCRUDRow{
			Function: row.FunctionID, FunctionName: row.FunctionName,
			Data: row.DataID, DataName: row.DataName,
			Create: row.Create, Read: row.Read, Update: row.Update, Delete: row.Delete,
			Scope:     operations.ThreeViewCRUDScopeFunctionDeclared,
			Relations: append([]string(nil), relations...),
			// Several declarations for one pair are shown as one row. Whether
			// they agree is not decided here, so the row is flagged for review.
			Conflicting: row.ConflictingDeclarations || len(relations) > 1,
		})
	}

	// Item-level references: the members of a declared key refer to fields.
	for _, key := range inspection.RelationalFacts.LegacyKeys {
		owner := nodeByToken[tokenByLocator[key.Locator]].ParentToken
		for _, member := range key.Members {
			if subjectView[owner] != operations.ThreeViewData {
				omit("key-member-outside-data-subject")
				continue
			}
			reference := operations.ThreeViewFieldReference{
				Occurrence: tokenByLocator[member.Locator], Kind: "key-member",
				Nature: operations.ThreeViewNatureSourceFact, Owner: owner,
				ReferencedID: member.FieldLegacyID,
				AuthorityLimits: []string{
					"key-membership-is-a-source-fact-not-a-business-identity",
					"key-membership-is-not-a-functional-dependency",
				},
			}
			var matches []string
			for _, field := range fieldsByOwner[owner] {
				if member.FieldLegacyID != "" && field.SourceFact.SourceID == member.FieldLegacyID {
					matches = append(matches, field.Occurrence)
				}
			}
			switch {
			case member.FieldLegacyID == "":
				reference.Resolution = "referenced-id-missing"
			case len(matches) == 1:
				reference.Resolution = "resolved"
				reference.Field = matches[0]
			case len(matches) == 0:
				reference.Resolution = "field-not-found"
			default:
				// Two fields carry the referenced ID. Neither is chosen.
				reference.Resolution = "field-ambiguous"
			}
			value.FieldReferences = append(value.FieldReferences, reference)
		}
	}

	for _, unresolved := range projection.Unresolved {
		value.Unresolved = append(value.Unresolved, operations.ThreeViewUnresolved{
			Code: unresolved.Code, RelationKind: unresolved.RelationKind,
			SourceOccurrences: append([]string(nil), unresolved.SourceTokens...),
		})
	}
	coverage.UnresolvedRelations = len(value.Unresolved)
	coverage.Status = operations.ThreeViewCoverageComplete
	if coverage.UnplacedConstructs != 0 || coverage.UnknownConstructs != 0 ||
		coverage.UnresolvedRelations != 0 || len(coverage.OmittedRelations) != 0 {
		coverage.Status = operations.ThreeViewCoveragePartial
	}

	if request.Pin != nil {
		state := operations.ThreeViewPinState{ThreeViewPin: *request.Pin}
		_, isSubject := subjectView[request.Pin.Occurrence]
		switch {
		case request.Pin.SourceArtifactDigest != value.Source.ArtifactDigest:
			// The pinned bytes are no longer the bytes of this source. The pin
			// is reported stale and is not moved to a similar-looking element.
			state.Status = operations.ThreeViewPinStaleSourceChanged
		case isSubject:
			state.Status = operations.ThreeViewPinHeld
		default:
			state.Status = operations.ThreeViewPinOccurrenceNotFound
		}
		value.Pin = &state
	}
	return value, nil
}

func relationOccurrence(relation operations.ThreeViewRelation) string {
	tokens := append([]string(nil), relation.SourceOccurrences...)
	sort.Strings(tokens)
	h := sha256.New()
	for _, part := range append([]string{
		Revision, relation.Kind, relation.Nature, relation.From, relation.To, relation.Operations,
	}, tokens...) {
		h.Write([]byte(part))
		h.Write([]byte{0})
	}
	return "relation:sha256:" + hex.EncodeToString(h.Sum(nil))
}
