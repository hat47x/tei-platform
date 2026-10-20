// Package designoutput derives one deterministic design-output bundle from an
// accepted Workspace: table and function definition documents, declared
// Application x Data matrices in both directions, and the DDL each database
// Provider can render. The bundle is an artifact of accepted meaning, not a
// second source: editing its files never changes the Workspace, and a
// candidate that has not been accepted is never included.
package designoutput

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"tei/internal/database"
	"tei/internal/environment"
	"tei/internal/ir"
	"tei/internal/sourceindex"
	"tei/internal/workspace"
	accessmatrix "tei/tools/reference-access-matrix"
	ddlentry "tei/tools/reference-ddl-entry"
	funcdoc "tei/tools/reference-function-definition-document"
	tabledoc "tei/tools/reference-table-definition-document"
)

const Schema = "tei.reference.design-output-bundle/v0"

// Artifact describes one rendered file. Digest is over the rendered bytes so a
// reader can verify the file, while StructureRevision identifies the accepted
// structure the artifact was derived from. Neither value is an approval or an
// execution grant.
type Artifact struct {
	Name              string   `json:"name"`
	Kind              string   `json:"kind"`
	Schema            string   `json:"schema"`
	Path              string   `json:"path"`
	Bytes             int      `json:"bytes"`
	Digest            string   `json:"digest"`
	StructureRevision string   `json:"structureRevision,omitempty"`
	Unresolved        []string `json:"unresolved,omitempty"`
}

// Unsupported records a requested artifact that this bundle cannot produce.
// It is kept separate from an empty artifact so a missing dimension cannot be
// read as "nothing to report".
type Unsupported struct {
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

// Manifest binds every artifact to one accepted Source revision.
type Manifest struct {
	Schema         string        `json:"schema"`
	SourceRevision string        `json:"sourceRevision"`
	Provider       string        `json:"provider"`
	Artifacts      []Artifact    `json:"artifacts"`
	Unsupported    []Unsupported `json:"unsupported,omitempty"`
	Notes          []string      `json:"notes"`
}

type File struct {
	Path    string
	Content []byte
}

type Bundle struct {
	Manifest Manifest
	Files    []File
}

func digestOf(content []byte) string {
	sum := sha256.Sum256(content)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// Build derives the bundle from one in-memory World and the Source revision it
// was loaded from. It performs no Workspace or database write.
func Build(world ir.World, sourceRevision string, provider environment.DatabaseProvider) (Bundle, error) {
	if strings.TrimSpace(sourceRevision) == "" {
		return Bundle{}, fmt.Errorf("source revision is required")
	}
	if provider == "" {
		provider = environment.ProviderPostgres
	}
	bundle := Bundle{Manifest: Manifest{
		Schema: Schema, SourceRevision: sourceRevision, Provider: string(provider),
		Artifacts: []Artifact{},
		Notes: []string{
			"受理済みの意味から導出した成果物です。成果物を編集しても、意味の正本には反映されません。",
			"一覧は宣言された内容であり、実際の副作用・権限・実行結果の観測ではありません。",
			"導出元の主体は認証していません。sourceRevisionと生成手段だけを記録します。",
		},
	}}
	add := func(name, kind, schema, structureRevision, path string, content []byte, unresolved []string) {
		bundle.Files = append(bundle.Files, File{Path: path, Content: content})
		artifact := Artifact{
			Name: name, Kind: kind, Schema: schema, Path: path,
			Bytes: len(content), Digest: digestOf(content),
			StructureRevision: structureRevision,
		}
		if len(unresolved) > 0 {
			artifact.Unresolved = append([]string(nil), unresolved...)
			sort.Strings(artifact.Unresolved)
		}
		bundle.Manifest.Artifacts = append(bundle.Manifest.Artifacts, artifact)
	}

	tableDoc, err := tabledoc.Render(world)
	if err != nil {
		return Bundle{}, fmt.Errorf("render table definitions: %w", err)
	}
	add("table-definitions", "table-definition-document", tabledoc.Schema,
		tableDoc.StructureRevision, "table-definitions.md", []byte(tableDoc.Markdown), tableDoc.Unresolved)

	functionDoc, err := funcdoc.Render(world)
	if err != nil {
		return Bundle{}, fmt.Errorf("render function definitions: %w", err)
	}
	add("function-definitions", "function-definition-document", funcdoc.Schema,
		functionDoc.StructureRevision, "function-definitions.md", []byte(functionDoc.Markdown), functionDoc.Unresolved)

	matrix, err := accessmatrix.Derive(world)
	if err != nil {
		return Bundle{}, fmt.Errorf("derive declared access matrix: %w", err)
	}
	add("application-data-matrix", "declared-access-matrix", accessmatrix.Schema,
		matrix.StructureRevision, "application-data-matrix.md", []byte(matrix.Markdown), gapStrings(matrix))

	inverse := renderDataApplicationMatrix(matrix)
	add("data-application-matrix", "data-access-matrix", accessmatrix.Schema,
		matrix.StructureRevision, "data-application-matrix.md", []byte(inverse), gapStrings(matrix))

	databases := append([]database.Database(nil), world.Databases...)
	sort.Slice(databases, func(i, j int) bool { return databases[i].ID < databases[j].ID })
	for _, db := range databases {
		id := string(db.ID)
		result, err := ddlentry.Render(world, id, provider)
		if err != nil {
			bundle.Manifest.Unsupported = append(bundle.Manifest.Unsupported, Unsupported{
				Name: "ddl/" + id, Reason: err.Error(),
			})
			continue
		}
		unresolved := make([]string, 0, len(result.Unresolved))
		for _, u := range result.Unresolved {
			unresolved = append(unresolved, u.Table+": "+u.Reason)
		}
		add("ddl/"+id, "ddl-script", ddlentry.Schema, "",
			"ddl/"+id+".sql", renderDDL(result, sourceRevision), unresolved)
	}
	bundle.Manifest.Unsupported = append(bundle.Manifest.Unsupported, Unsupported{
		Name: "business-cross-view",
		Reason: "受理済みのBusiness Procedure sourceがまだ提供されていません（#1125）。" +
			"業務・機能・データ間の対応一覧は、非CanonicalなDFDや候補から生成しません。",
	})
	bundle.Manifest.Notes = append(bundle.Manifest.Notes,
		"index.htmlは成果物の一覧とリンクを表示するナビゲーションであり、manifestのartifactには含めません。")
	sort.Slice(bundle.Manifest.Unsupported, func(i, j int) bool { return bundle.Manifest.Unsupported[i].Name < bundle.Manifest.Unsupported[j].Name })
	sort.Slice(bundle.Manifest.Artifacts, func(i, j int) bool { return bundle.Manifest.Artifacts[i].Name < bundle.Manifest.Artifacts[j].Name })
	bundle.Files = append(bundle.Files, File{Path: "index.html", Content: renderIndex(bundle.Manifest)})
	sort.Slice(bundle.Files, func(i, j int) bool { return bundle.Files[i].Path < bundle.Files[j].Path })
	return bundle, nil
}

func gapStrings(m accessmatrix.Matrix) []string {
	out := make([]string, 0, len(m.Gaps))
	for _, g := range m.Gaps {
		out = append(out, g.Code+": "+g.Subject)
	}
	return out
}

// renderDataApplicationMatrix reads the same declared cells from the Data side.
// It adds no fact the Application-facing matrix did not already contain.
func renderDataApplicationMatrix(m accessmatrix.Matrix) string {
	var b strings.Builder
	b.WriteString("# 宣言された利用の対応表（Data × Application）\n\n")
	b.WriteString("> 受理済みの意味から導出した文書です。この文書を編集しても、意味の正本には反映されません。\n\n")
	fmt.Fprintf(&b, "- 導出元の構造の版: `%s`\n", m.StructureRevision)
	b.WriteString("- Application × Dataの対応表を、Data側から読み直したものです。列の文字は Create / Read / Update の宣言です。\n")
	b.WriteString("- 削除は現在のモデルで表現できないため、この表には現れません。空欄を「削除しない」と読まないでください。\n")
	b.WriteString("- 宣言であり、実際の副作用の観測ではありません。\n\n")
	if len(m.Data) == 0 {
		b.WriteString("宣言された利用はありません。\n")
		return b.String()
	}
	byData := map[string][]string{}
	for _, c := range m.Cells {
		byData[c.Data] = append(byData[c.Data], fmt.Sprintf("`%s`（%s）", c.Application, strings.Join(c.Access, "")))
	}
	b.WriteString("| Data | 利用するApplication（宣言） |\n| --- | --- |\n")
	for _, d := range m.Data {
		parts := append([]string(nil), byData[d]...)
		sort.Strings(parts)
		if len(parts) == 0 {
			parts = []string{"—"}
		}
		fmt.Fprintf(&b, "| `%s` | %s |\n", d, strings.Join(parts, "、"))
	}
	if len(m.Gaps) > 0 {
		b.WriteString("\n## 読み取れなかった宣言\n\n")
		for _, g := range m.Gaps {
			fmt.Fprintf(&b, "- %s: `%s`\n", g.Code, g.Subject)
		}
	}
	return b.String()
}

// renderDDL formats what the Provider returned. It adds no SQL of its own; the
// provider owns statement rendering (ADR-0015). Unresolved tables stay in the
// manifest rather than being filled in here.
func renderDDL(result ddlentry.Result, sourceRevision string) []byte {
	var b strings.Builder
	b.WriteString("-- TEI design output: fresh-create DDL\n")
	fmt.Fprintf(&b, "-- database: %s\n", result.Database)
	fmt.Fprintf(&b, "-- provider: %s\n", result.Provider)
	fmt.Fprintf(&b, "-- source revision: %s\n", sourceRevision)
	b.WriteString("-- Derived from declared persistence mappings. This is not a migration for an existing database.\n")
	b.WriteString("-- Tables the Provider could not render are listed in manifest.json.\n\n")
	if len(result.Tables) == 0 {
		b.WriteString("-- no table could be rendered; see manifest.json\n")
		return []byte(b.String())
	}
	for _, table := range result.Tables {
		fmt.Fprintf(&b, "-- table %s\n", table.Name)
		for _, statement := range table.Statements {
			b.WriteString(statement)
			if !strings.HasSuffix(statement, "\n") {
				b.WriteString("\n")
			}
		}
		b.WriteString("\n")
	}
	return []byte(b.String())
}

// Write loads one Workspace, derives the bundle, and publishes it into a new
// directory. The directory must not exist. Files are written into a staging
// directory beside it and published with one Rename, so a failed run does not
// leave a partially written bundle. A concurrent creator that appears between
// the absence check and the Rename is an explicit limitation, not a guarantee.
func Write(workspaceDir, outDir string, provider environment.DatabaseProvider) (Manifest, error) {
	if strings.TrimSpace(workspaceDir) == "" || strings.TrimSpace(outDir) == "" {
		return Manifest{}, fmt.Errorf("workspace and out are required")
	}
	loaded, err := workspace.Load(workspaceDir)
	if err != nil {
		return Manifest{}, fmt.Errorf("load Workspace: %w", err)
	}
	index, err := workspace.BuildSourceIndex(workspaceDir, loaded)
	if err != nil {
		return Manifest{}, fmt.Errorf("build Source Index: %w", err)
	}
	bundle, err := Build(loaded.World, sourceindex.ModelRevision(index), provider)
	if err != nil {
		return Manifest{}, err
	}

	manifestJSON, err := json.MarshalIndent(bundle.Manifest, "", "  ")
	if err != nil {
		return Manifest{}, err
	}
	manifestJSON = append(manifestJSON, '\n')
	files := append([]File(nil), bundle.Files...)
	files = append(files, File{Path: "manifest.json", Content: manifestJSON})

	absolute, err := filepath.Abs(outDir)
	if err != nil {
		return Manifest{}, err
	}
	if _, err := os.Lstat(absolute); err == nil {
		return Manifest{}, fmt.Errorf("output directory already exists: %s", absolute)
	} else if !errors.Is(err, os.ErrNotExist) {
		return Manifest{}, fmt.Errorf("inspect output directory: %w", err)
	}
	parent := filepath.Dir(absolute)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return Manifest{}, fmt.Errorf("create output parent: %w", err)
	}
	stage, err := os.MkdirTemp(parent, ".tei-design-output-*")
	if err != nil {
		return Manifest{}, fmt.Errorf("create staging directory: %w", err)
	}
	defer os.RemoveAll(stage)
	for _, file := range files {
		target := filepath.Join(stage, filepath.FromSlash(file.Path))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return Manifest{}, fmt.Errorf("create artifact directory: %w", err)
		}
		f, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if err != nil {
			return Manifest{}, fmt.Errorf("create artifact %s: %w", file.Path, err)
		}
		_, writeErr := f.Write(file.Content)
		closeErr := f.Close()
		if writeErr != nil {
			return Manifest{}, fmt.Errorf("write artifact %s: %w", file.Path, writeErr)
		}
		if closeErr != nil {
			return Manifest{}, fmt.Errorf("close artifact %s: %w", file.Path, closeErr)
		}
	}
	// Re-check immediately before publication; the Rename still refuses a
	// non-empty existing directory and the staging directory stays separate.
	if _, err := os.Lstat(absolute); err == nil {
		return Manifest{}, fmt.Errorf("output directory appeared before publication: %s", absolute)
	} else if !errors.Is(err, os.ErrNotExist) {
		return Manifest{}, fmt.Errorf("inspect output directory: %w", err)
	}
	if err := os.Rename(stage, absolute); err != nil {
		return Manifest{}, fmt.Errorf("publish design-output bundle: %w", err)
	}
	return bundle.Manifest, nil
}
