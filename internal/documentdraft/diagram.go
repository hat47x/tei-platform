package documentdraft

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"sort"
	"strings"

	"tei/internal/sourceindex"
	"tei/internal/threeelement"
)

// DocumentKindThreeElementDiagram projects one three-element analysis model into
// a node-and-edge diagram document.
//
// The input model is a SEPARATE artifact from a TEI Workspace: it is the
// three-element analysis (Data / Function / Business) the model file declares.
// This kind never merges the two, and never decides which declared TEI subject
// counts as Data, Function or Business 窶・the model file states it, and TEI
// reports what the file said. The element mapping itself is an Owner decision
// (THREE_ELEMENT_OWNER_ACCEPTANCE_ITEMS), so a projection that invented a mapping
// would be asserting meaning nobody accepted.
const DocumentKindThreeElementDiagram = "three-element-diagram"

// DiagramNode is one element of the diagram. Layer is the model's own grouping
// (data | function | business), not TEI's judgement about the element.
type DiagramNode struct {
	ID    string `json:"id"`
	Layer string `json:"layer"`
	Name  string `json:"name,omitempty"`
	// AttributeCount and OperationCount name what the element declares, so a
	// reader can see the shape without reading the whole model.
	AttributeCount  int      `json:"attributeCount,omitempty"`
	OperationCount  int      `json:"operationCount,omitempty"`
	ActivityCount   int      `json:"activityCount,omitempty"`
	AuthorityLimits []string `json:"authorityLimits,omitempty"`
}

// DiagramEdge is one declared relation between two nodes. Resolution reports
// whether both endpoints are declared in the same model; an unresolved edge is
// kept and named, never dropped.
type DiagramEdge struct {
	ID         string `json:"id"`
	Kind       string `json:"kind"`
	FromID     string `json:"fromId"`
	FromLayer  string `json:"fromLayer"`
	ToID       string `json:"toId"`
	ToLayer    string `json:"toLayer"`
	Resolution string `json:"resolution"`
	Role       string `json:"role,omitempty"`
	Directed   bool   `json:"directed,omitempty"`
}

// DiagramContent is the generated diagram: nodes, edges and the CRUD definitions
// the model declares. It carries no image: a rendered picture would be a second
// representation of the same structure, and the structure is what the reader can
// check.
type DiagramContent struct {
	ModelID          string           `json:"modelId"`
	ModelFingerprint string           `json:"modelFingerprint"`
	Nodes            []DiagramNode    `json:"nodes"`
	Edges            []DiagramEdge    `json:"edges"`
	CRUDs            []DiagramCRUD    `json:"cruds,omitempty"`
	Diagnostics      []DiagramFinding `json:"diagnostics,omitempty"`
	UnresolvedEdges  []string         `json:"unresolvedEdges,omitempty"`
	AuthorityLimits  []string         `json:"authorityLimits,omitempty"`
}

// DiagramCRUD is one declared Function竊奪ata CRUD mapping.
type DiagramCRUD struct {
	ID         string `json:"id"`
	FunctionID string `json:"functionId"`
	DataID     string `json:"dataId"`
	Operations string `json:"operations"`
	Source     string `json:"source,omitempty"`
}

// DiagramFinding is one diagnostic the model carries. It is reported as-is: the
// document does not resolve, suppress or re-rank it.
type DiagramFinding struct {
	Code       string   `json:"code"`
	Severity   string   `json:"severity"`
	Message    string   `json:"message"`
	SubjectIDs []string `json:"subjectIds,omitempty"`
}

// DiagramAuthorityLimits states what a diagram does not grant. The element
// mapping is named explicitly because it is the Owner decision this projection
// must not appear to have made.
func DiagramAuthorityLimits() []string {
	return []string{
		"diagram-is-a-derived-read-only-projection",
		"diagram-element-mapping-is-model-declared-not-tei-decided",
		"diagram-does-not-assert-business-meaning-or-crud-semantics",
		"unresolved-edge-is-kept-and-named-never-dropped",
		"diagram-diagnostics-are-reported-not-resolved",
	}
}

// requireDiagramInputs refuses a diagram request that names a Workspace subject,
// an application, a section list or an empty model path. The diagram reads a
// separate analysis artifact, so none of the Workspace inputs apply to it.
func requireDiagramInputs(req Request) error {
	if strings.TrimSpace(req.ModelPath) == "" || req.ModelPath != strings.TrimSpace(req.ModelPath) {
		return draftError(CodeInvalidArgument,
			"document kind %s requires an exact model path", DocumentKindThreeElementDiagram)
	}
	if strings.TrimSpace(req.Application) != "" {
		return draftError(CodeInvalidArgument,
			"-application is only meaningful for document kind %s", DocumentKindCRUDTable)
	}
	if len(req.Sections) > 0 {
		return draftError(CodeInvalidArgument,
			"sections are only meaningful for document kind %s", DocumentKindSpecification)
	}
	if strings.TrimSpace(req.Subject.Kind) != "" || strings.TrimSpace(req.Subject.ID) != "" {
		return draftError(CodeInvalidArgument,
			"document kind %s is about an analysis model; -subject-kind and -subject-id do not apply",
			DocumentKindThreeElementDiagram)
	}
	if strings.TrimSpace(req.RelationID) != "" {
		return draftError(CodeInvalidArgument,
			"document kind %s takes its relations from the model, not from -relation-id",
			DocumentKindThreeElementDiagram)
	}
	return nil
}

// buildDiagramDraft reads one three-element analysis model and projects it. No
// Workspace is read: the diagram is about a separate artifact, and reading a
// Workspace here would mix the two models.
func buildDiagramDraft(req Request) (Draft, error) {
	model, err := loadThreeElementModelFile(req.ModelPath)
	if err != nil {
		return Draft{}, err
	}
	draft := Draft{
		SchemaVersion: SchemaVersion,
		Effect:        EffectReadOnlyProjection,
		DocumentKind:  DocumentKindThreeElementDiagram,
		Title:         buildDiagramTitle(req.Title, model),
		Model: ModelRef{
			ModelRevision: model.ModelFingerprint,
		},
		AuthorityLimits: DiagramAuthorityLimits(),
	}
	if err := draft.projectThreeElementDiagram(req.ModelPath); err != nil {
		return Draft{}, err
	}
	draft.DraftID = draftID(draft)
	return draft, nil
}

func buildDiagramTitle(requested string, model threeelement.ThreeElementModel) string {
	if strings.TrimSpace(requested) != "" {
		return requested
	}
	name := strings.TrimSpace(model.Name)
	if name == "" {
		name = model.ModelID
	}
	return "荳芽ｦ∫ｴ蛻・梵蝗ｳ: " + name
}

// projectThreeElementDiagram fills one draft with the nodes and edges of the
// three-element model at modelPath. The order is the model's own, so the same
// model always produces the same diagram.
func (d *Draft) projectThreeElementDiagram(modelPath string) error {
	model, err := loadThreeElementModelFile(modelPath)
	if err != nil {
		return err
	}
	digests := diagramInputDigests(modelPath, model)
	numbering := &occurrenceNumbering{}

	content := DiagramContent{
		ModelID:          model.ModelID,
		ModelFingerprint: model.ModelFingerprint,
		Nodes:            diagramNodes(model),
		Edges:            diagramEdges(model),
		CRUDs:            diagramCRUDs(model),
		AuthorityLimits:  DiagramAuthorityLimits(),
	}
	for _, diagnostic := range model.Diagnostics {
		content.Diagnostics = append(content.Diagnostics, DiagramFinding{
			Code: diagnostic.Code, Severity: diagnostic.Severity,
			Message: diagnostic.Message, SubjectIDs: diagnostic.SubjectIDs,
		})
	}
	// An edge whose endpoint the model does not declare is kept in the diagram and
	// named in unresolvedEdges: dropping it would present a complete-looking
	// diagram of a model that is not complete.
	for _, edge := range content.Edges {
		if edge.Resolution == ResolutionUnresolved {
			content.UnresolvedEdges = append(content.UnresolvedEdges, edge.ID)
		}
	}

	// The model itself is the subject of this document: there is no TEI subject
	// to point at, and pretending there is one would mix the two models.
	d.Subjects = []SubjectRef{{
		Subject: sourceindex.Subject{Kind: "three-element-model", ID: model.ModelID},
		Sources: []SourceBinding{{
			ArtifactPath: modelPath,
			Format:       "three-element-json",
			Fingerprint:  diagramFingerprint(modelPath),
			Role:         "analysis-model",
		}},
	}}
	d.Occurrences = append(d.Occurrences, numbering.add(OccurrenceDiagram,
		[]sourceindex.Subject{d.Subjects[0].Subject},
		conditions(digests, "three-element-diagram-v0"),
		OccurrenceContent{Diagram: &content}))
	return nil
}

// loadThreeElementModelFile reads one three-element model and recomputes its
// fingerprint, exactly as the three-element CLI does. A model that cannot be read
// is an argument failure, not an empty diagram.
func loadThreeElementModelFile(path string) (threeelement.ThreeElementModel, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return threeelement.ThreeElementModel{}, draftError(CodeInvalidArgument,
			"three-element model %s cannot be read: %v", path, err)
	}
	// A leading UTF-8 BOM is a byte-order mark, not JSON text. Windows editors add
	// one often enough that refusing it would report a broken model for an
	// encoding detail.
	raw = bytes.TrimPrefix(raw, []byte{0xEF, 0xBB, 0xBF})
	var model threeelement.ThreeElementModel
	if err := json.Unmarshal(raw, &model); err != nil {
		return threeelement.ThreeElementModel{}, draftError(CodeInvalidArgument,
			"three-element model %s is not readable JSON: %v", path, err)
	}
	if strings.TrimSpace(model.ModelID) == "" {
		return threeelement.ThreeElementModel{}, draftError(CodeInvalidArgument,
			"three-element model %s declares no model ID", path)
	}
	model.ModelFingerprint = threeelement.ModelFingerprint(model)
	return model, nil
}

// diagramNodes lists every element the model declares, in a deterministic order.
// The layer is the model's own grouping.
func diagramNodes(model threeelement.ThreeElementModel) []DiagramNode {
	var nodes []DiagramNode
	for _, data := range model.DataElements {
		nodes = append(nodes, DiagramNode{
			ID: data.ID, Layer: "data", Name: data.Name,
			AttributeCount: len(data.Attributes), AuthorityLimits: data.AuthorityLimits,
		})
	}
	for _, function := range model.FunctionElements {
		nodes = append(nodes, DiagramNode{
			ID: function.ID, Layer: "function", Name: function.Name,
			OperationCount: len(function.Operations), AuthorityLimits: function.AuthorityLimits,
		})
	}
	for _, business := range model.BusinessElements {
		nodes = append(nodes, DiagramNode{
			ID: business.ID, Layer: "business", Name: business.Name,
			ActivityCount: len(business.Activities), AuthorityLimits: business.AuthorityLimits,
		})
	}
	sortDiagramNodes(nodes)
	return nodes
}

func sortDiagramNodes(nodes []DiagramNode) {
	for i := 1; i < len(nodes); i++ {
		for j := i; j > 0; j-- {
			if diagramNodeLess(nodes[j], nodes[j-1]) {
				nodes[j], nodes[j-1] = nodes[j-1], nodes[j]
				continue
			}
			break
		}
	}
}

func diagramNodeLess(a, b DiagramNode) bool {
	if a.Layer != b.Layer {
		return a.Layer < b.Layer
	}
	return a.ID < b.ID
}

// diagramEdges lists every declared cross-reference and resolves its endpoints
// against the same model.
func diagramEdges(model threeelement.ThreeElementModel) []DiagramEdge {
	data, functions, business := model.Index()
	edges := make([]DiagramEdge, 0, len(model.CrossReferences))
	for _, reference := range model.CrossReferences {
		fromLayer, fromFound := layerOf(reference.FromID, data, functions, business)
		toLayer, toFound := layerOf(reference.ToID, data, functions, business)
		resolution := ResolutionResolved
		if !fromFound || !toFound {
			resolution = ResolutionUnresolved
		}
		edges = append(edges, DiagramEdge{
			ID: reference.ID, Kind: reference.Kind,
			FromID: reference.FromID, FromLayer: fromLayer,
			ToID: reference.ToID, ToLayer: toLayer,
			Resolution: resolution, Role: reference.Role, Directed: reference.Directed,
		})
	}
	sort.SliceStable(edges, func(i, j int) bool { return edges[i].ID < edges[j].ID })
	return edges
}

// layerOf reports which layer one ID belongs to. An ID the model does not declare
// has no layer, and the caller keeps the edge unresolved rather than guessing one.
func layerOf(id string, data map[string]threeelement.DataElement,
	functions map[string]threeelement.FunctionElement,
	business map[string]threeelement.BusinessElement,
) (string, bool) {
	if _, ok := data[id]; ok {
		return "data", true
	}
	if _, ok := functions[id]; ok {
		return "function", true
	}
	if _, ok := business[id]; ok {
		return "business", true
	}
	return "", false
}

func diagramCRUDs(model threeelement.ThreeElementModel) []DiagramCRUD {
	cruds := make([]DiagramCRUD, 0, len(model.CRUDDefinitions))
	for _, crud := range model.CRUDDefinitions {
		cruds = append(cruds, DiagramCRUD{
			ID: crud.ID, FunctionID: crud.FunctionID, DataID: crud.DataID,
			Operations: crudOperations(crud), Source: crud.Source,
		})
	}
	sort.SliceStable(cruds, func(i, j int) bool { return cruds[i].ID < cruds[j].ID })
	return cruds
}

// crudOperations names the declared letters in a fixed order, so the same
// declaration always reads the same way.
func crudOperations(crud threeelement.CRUDDefinition) string {
	var letters []string
	if crud.Create {
		letters = append(letters, "C")
	}
	if crud.Read {
		letters = append(letters, "R")
	}
	if crud.Update {
		letters = append(letters, "U")
	}
	if crud.Delete {
		letters = append(letters, "D")
	}
	return strings.Join(letters, "")
}

// diagramFingerprint is the byte identity of the model file the diagram was read
// from.
func diagramFingerprint(path string) string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func diagramInputDigests(path string, model threeelement.ThreeElementModel) map[string]string {
	return map[string]string{
		"modelFingerprint":  model.ModelFingerprint,
		"sourceFingerprint": diagramFingerprint(path),
	}
}
