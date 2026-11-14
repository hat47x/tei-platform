package browsegen

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"tei/tools/reference-application-binary/internal/appgen"
)

//go:embed templates/main.go.tmpl
var templates embed.FS

type lit struct{ Literal string }

type genData struct {
	GeneratorID, FunctionID, SourceRevision, ModelDigest string
	FunctionIDLiteral, BoundSQLLiteral, UsageLiteral     string
	Params, Columns                                      []lit
}

// Manifest is what the binary embeds as its provenance. It names every input
// the Function is a function of and every Source declaration not generated.
type Manifest struct {
	Schema         string            `json:"schema"`
	Generator      string            `json:"generator"`
	TemplateSHA256 string            `json:"templateSha256"`
	Plan           *Plan             `json:"plan"`
	GeneratedFiles map[string]string `json:"generatedFiles"`
}

// Generate writes main.go, go.mod and manifest.json into outDir and returns
// the manifest. Nothing is written when the plan cannot be realised.
func Generate(p *Plan, outDir string) (*Manifest, error) {
	raw, err := templates.ReadFile("templates/main.go.tmpl")
	if err != nil {
		return nil, err
	}
	tpl, err := template.New("main").Parse(string(raw))
	if err != nil {
		return nil, err
	}
	d := genData{GeneratorID: GeneratorID, FunctionID: p.FunctionID, SourceRevision: p.SourceRevision, ModelDigest: p.ModelDigest,
		FunctionIDLiteral: strconv.Quote(p.FunctionID), BoundSQLLiteral: strconv.Quote(p.BoundSQL)}
	usage := ""
	for _, n := range p.Params {
		d.Params = append(d.Params, lit{strconv.Quote(n)})
		usage += " --" + n + "=<value>"
	}
	d.UsageLiteral = strconv.Quote(usage)
	for _, c := range p.Columns {
		d.Columns = append(d.Columns, lit{strconv.Quote(c.Name)})
	}
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, d); err != nil {
		return nil, err
	}
	goMod := "module app/" + strings.ReplaceAll(p.ApplicationID, "_", "-") + "\n\ngo 1.25.0\n\nrequire github.com/jackc/pgx/v5 v5.9.2\n"
	m := &Manifest{Schema: "tei.browse-binary.manifest/v0", Generator: GeneratorID, TemplateSHA256: sha(raw), Plan: p,
		GeneratedFiles: map[string]string{"main.go": sha(buf.Bytes()), "go.mod": sha([]byte(goMod))}}
	mj, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return nil, err
	}
	mj = append(mj, '\n')
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, err
	}
	for name, b := range map[string][]byte{"main.go": buf.Bytes(), "go.mod": []byte(goMod), "manifest.json": mj} {
		if err := os.WriteFile(filepath.Join(outDir, name), b, 0o644); err != nil {
			return nil, err
		}
	}
	return m, nil
}

// Receipt pairs the manifest with the binary actually built from it.
type Receipt struct {
	Schema         string `json:"schema"`
	FunctionID     string `json:"functionId"`
	ManifestSHA256 string `json:"manifestSha256"`
	BinarySHA256   string `json:"binarySha256"`
	BinarySize     int64  `json:"binarySize"`
	Toolchain      string `json:"toolchain"`
}

// Build compiles the generated module and returns the receipt that binds the
// manifest bytes to the binary bytes.
func Build(outDir, binaryPath string) (*Receipt, error) {
	mj, err := os.ReadFile(filepath.Join(outDir, "manifest.json"))
	if err != nil {
		return nil, err
	}
	res, err := appgen.BuildModule(outDir, binaryPath, "")
	if err != nil {
		return nil, err
	}
	var m Manifest
	if err := json.Unmarshal(mj, &m); err != nil {
		return nil, err
	}
	tc, _ := exec.Command("go", "env", "GOVERSION").Output()
	return &Receipt{Schema: "tei.browse-binary.receipt/v0", FunctionID: m.Plan.FunctionID, ManifestSHA256: sha(mj),
		BinarySHA256: strings.TrimPrefix(res.Digest, "sha256:"), BinarySize: res.BinarySize,
		Toolchain: fmt.Sprintf("%s", strings.TrimSpace(string(tc)))}, nil
}

// Verify checks that the binary on disk is the one built from the manifest on
// disk: the receipt's digests match both files, and the binary reports the same
// manifest bytes it was built with. A stale binary or a regenerated Source with
// no rebuild fails here.
func Verify(outDir, binaryPath string, rc *Receipt) error {
	mj, err := os.ReadFile(filepath.Join(outDir, "manifest.json"))
	if err != nil {
		return err
	}
	if got := sha(mj); got != rc.ManifestSHA256 {
		return fmt.Errorf("STALE_BINARY: manifest %s differs from the manifest the binary was built from (%s)", got, rc.ManifestSHA256)
	}
	bb, err := os.ReadFile(binaryPath)
	if err != nil {
		return err
	}
	if got := sha(bb); got != rc.BinarySHA256 {
		return fmt.Errorf("BINARY_DIGEST_MISMATCH: %s != %s", got, rc.BinarySHA256)
	}
	out, err := exec.Command(binaryPath, "provenance").Output()
	if err != nil {
		return fmt.Errorf("provenance: %w", err)
	}
	if got := sha(out); got != rc.ManifestSHA256 {
		return fmt.Errorf("PROVENANCE_MISMATCH: binary embeds manifest %s, receipt expects %s", got, rc.ManifestSHA256)
	}
	return nil
}
