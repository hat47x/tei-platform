package generatorselection

import (
    "bytes"
    "encoding/json"
    "fmt"
    "math/big"
    "os"
    "os/exec"
    "path/filepath"
    "strings"
    "testing"

    "tei/internal/core"
    "tei/internal/semantic"
    "tei/internal/semantic/validationeval"
    "tei/internal/sourceindex"
    "tei/internal/validation"
    "tei/internal/workspace"
)

// This probe witnesses logical Source requests only, not physical Provider I/O.
type oneFactorReadProbe struct{ requested []core.CanonicalID }
func (p *oneFactorReadProbe) Number(id core.CanonicalID) (*big.Rat, error) {
    p.requested = append(p.requested, id)
    switch id {
    case "sales.order_line.quantity": return big.NewRat(2, 1), nil
    case "sales.order_line.approved_quantity": return big.NewRat(-2, 1), nil
    default: return nil, fmt.Errorf("unexpected read %s", id)
    }
}
func (*oneFactorReadProbe) RelationKeys(relation, attribute core.CanonicalID) ([]string, error) {
    return nil, fmt.Errorf("unexpected relation read %s/%s", relation, attribute)
}

// Both Workspaces contain exactly the same Data model with two std.decimal
// attributes. Only the selected Markdown Rule reference differs.
func oneFactorRuleWorkspace(t *testing.T, attribute string) (string, []byte) {
    t.Helper()
    root := t.TempDir()
    fixture := filepath.Join(repoRoot(t), "internal", "workspace", "testdata", "tei-md-rule")
    var markdown []byte
    for _, path := range []string{"tei.workspace.yaml", "semantic/sales/order.tei", "semantic/sales/order-line-rules.tei.md"} {
        raw, err := os.ReadFile(filepath.Join(fixture, filepath.FromSlash(path)))
        if err != nil { t.Fatal(err) }
        if strings.HasSuffix(path, "order.tei") {
            const old = "attribute quantity type std.decimal"
            if strings.Count(string(raw), old) != 1 { t.Fatal("fixture numeric declaration changed") }
            raw = []byte(strings.Replace(string(raw), old,
                old+"\n    attribute approvedQuantity type std.decimal", 1))
        }
        if strings.HasSuffix(path, ".tei.md") {
            const old = "sales.order_line.quantity is positive"
            if strings.Count(string(raw), old) != 1 { t.Fatal("fixture rule declaration changed") }
            raw = []byte(strings.Replace(string(raw), old, attribute+" is positive", 1))
            markdown = raw
        }
        target := filepath.Join(root, filepath.FromSlash(path))
        if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil { t.Fatal(err) }
        if err := os.WriteFile(target, raw, 0600); err != nil { t.Fatal(err) }
    }
    return root, markdown
}

func oneFactorRuleContract(t *testing.T, attribute, input string) string {
    t.Helper()
    raw, err := os.ReadFile(validationTestPaths(t).contract)
    if err != nil { t.Fatal(err) }
    var c Contract
    if err := json.Unmarshal(raw, &c); err != nil { t.Fatal(err) }
    if len(c.Inputs) != 1 || c.Validation == nil { t.Fatal("single-input reference contract changed") }
    c.Inputs[0].Attribute, c.Validation.Input = attribute, input
    raw, err = json.MarshalIndent(c, "", "  ")
    if err != nil { t.Fatal(err) }
    path := filepath.Join(t.TempDir(), "contract.json")
    if err := os.WriteFile(path, append(raw, '\n'), 0600); err != nil { t.Fatal(err) }
    return path
}

func oneFactorRuleGenerate(t *testing.T, workspaceDir, contractPath string) (string, []byte, error) {
    t.Helper()
    out := filepath.Join(t.TempDir(), "candidate")
    paths := validationTestPaths(t)
    _, err := GenerateFromTransformSource(TransformGeneratorOptions{
        ID: "go-positive", WorkspaceDir: workspaceDir, ContractPath: contractPath,
        GeneratorSourcePath: filepath.Join(paths.generators, generatorSources["go-positive"]),
        OutDir: out,
    })
    if err != nil { return out, nil, err }
    if err := VerifyTransformCandidate(out, workspaceDir, contractPath); err != nil {
        return out, nil, err
    }
    raw, err := os.ReadFile(filepath.Join(out, "main.go"))
    return out, raw, err
}

func oneFactorRuleRunGo(t *testing.T, mainFile string) string {
    t.Helper()
    binary := filepath.Join(t.TempDir(), "generated-positive")
    cmd := exec.Command("go", "build", "-trimpath", "-o", binary, mainFile)
    if out, err := cmd.CombinedOutput(); err != nil {
        t.Fatalf("generated-Go separate build: %v: %s", err, out)
    }
    cmd = exec.Command(binary)
    // One independent input for BOTH generated programs; the reference
    // Source and the external process must choose different actual fields.
    cmd.Stdin = strings.NewReader("{\"quantity\":\"2\",\"approvedQuantity\":\"-2\"}")
    raw, err := cmd.CombinedOutput()
    if err != nil { t.Fatalf("generated-Go separate run: %v: %s", err, raw) }
    var result map[string]any
    if err := json.Unmarshal(raw, &result); err != nil { t.Fatal(err) }
    state, _ := result["state"].(string)
    return state
}

func TestSelectedRuleOneFactorChangeReachesExistingGeneratorAndSeparateGoProcess(t *testing.T) {
    const ruleID = "sales.order_line.quantity_positive"
    cases := []struct{ name, attribute, input, expected string }{
        {"original", "sales.order_line.quantity", "quantity", "satisfied"},
        {"changed", "sales.order_line.approved_quantity", "approvedQuantity", "not-satisfied"},
    }
    var firstMarkdown, firstGo []byte
    for i, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            root, markdown := oneFactorRuleWorkspace(t, tc.attribute)
            if i == 0 { firstMarkdown = markdown } else {
                want := strings.Replace(string(firstMarkdown),
                    "sales.order_line.quantity is positive",
                    "sales.order_line.approved_quantity is positive", 1)
                if want != string(markdown) { t.Fatal("change was not limited to one Rule reference") }
            }
            loaded, err := workspace.Load(root)
            if err != nil { t.Fatalf("selected Markdown product Load: %v", err) }
            if len(loaded.AuthoringSources) != 1 ||
                loaded.AuthoringSources[0].Language.LanguageID != "tei.rule-authoring" ||
                loaded.AuthoringSources[0].Language.ContractVersion != "v0" {
                t.Fatalf("product did not select built-in Reader: %+v", loaded.AuthoringSources)
            }
            rule, ok := findValidation(loaded.World, ruleID)
            if !ok || string(rule.Attribute) != tc.attribute || rule.Kind != semantic.ValidationPositive {
                t.Fatalf("typed candidate changed incorrectly: %+v / %v", rule, ok)
            }
            for _, d := range validation.Validate(loaded.World) {
                if d.Code == "SEMANTIC_VALIDATION_POSITIVE_REQUIRES_NUMERIC_ATTRIBUTE" ||
                    d.Code == "UNKNOWN_SEMANTIC_VALIDATION_ATTRIBUTE" {
                    t.Fatalf("numeric Rule failed existing Validator: %+v", d)
                }
            }
            index, err := workspace.BuildSourceIndexFromLoad(root, loaded)
            if err != nil { t.Fatal(err) }
            refs := index.ReferencesFor(sourceindex.Subject{Kind:"semantic-validation", ID:ruleID})
            if len(refs) != 1 || refs[0].ArtifactPath != "semantic/sales/order-line-rules.tei.md" ||
                string(markdown[refs[0].Range.Start.Offset:refs[0].Range.End.Offset]) != ruleID {
                t.Fatalf("selected Rule lost its original Source span: %+v", refs)
            }
            probe := &oneFactorReadProbe{}
            state, err := validationeval.Evaluate(rule, probe)
            if err != nil || string(state) != tc.expected || len(probe.requested) != 1 ||
                string(probe.requested[0]) != tc.attribute {
                t.Fatalf("meaning/read: %s %v %v, want %s %s", state, err, probe.requested, tc.expected, tc.attribute)
            }
            contract := oneFactorRuleContract(t, tc.attribute, tc.input)
            output, generated, err := oneFactorRuleGenerate(t, root, contract)
            if err != nil { t.Fatalf("selected Rule -> existing Generator: %v", err) }
            if !bytes.Contains(generated, []byte(tc.attribute)) ||
                !bytes.Contains(generated, []byte("inputKey     = \""+tc.input+"\"")) {
                t.Fatal("generated Go lost the explicit Rule / input binding")
            }
            if i == 0 { firstGo = generated } else if bytes.Equal(firstGo, generated) {
                t.Fatal("one-factor Rule edit did not change generated Go bytes")
            }
            if observed := oneFactorRuleRunGo(t, filepath.Join(output, "main.go")); observed != tc.expected {
                t.Fatalf("separate Go process: got %s want independent Expected %s", observed, tc.expected)
            }
        })
    }

    // Holding the old contract fixed against a changed Rule must fail closed.
    changed, _ := oneFactorRuleWorkspace(t, "sales.order_line.approved_quantity")
    oldContract := oneFactorRuleContract(t, "sales.order_line.quantity", "quantity")
    _, _, err := oneFactorRuleGenerate(t, changed, oldContract)
    if err == nil || !strings.Contains(err.Error(), "contract input is") {
        t.Fatalf("stale contract silently followed a changed Rule: %v", err)
    }

    // A superficially similar reference edit to std.text parses but fails the
    // existing numeric meaning check; it must not qualify a Go Generator.
    textRoot, _ := oneFactorRuleWorkspace(t, "sales.order_line.item_name")
    textWorld, err := workspace.Load(textRoot)
    if err != nil { t.Fatal(err) }
    found := false
    for _, d := range validation.Validate(textWorld.World) {
        if d.Code == "SEMANTIC_VALIDATION_POSITIVE_REQUIRES_NUMERIC_ATTRIBUTE" { found = true }
    }
    if !found { t.Fatal("text attribute was not refused by existing Validator") }
    _, _, err = oneFactorRuleGenerate(t, textRoot, oneFactorRuleContract(t, "sales.order_line.item_name", "itemName"))
    if err == nil || !strings.Contains(err.Error(), "resolves to std.text") {
        t.Fatalf("std.text unexpectedly passed the numeric Generator contract: %v", err)
    }
}

// R1-M reference scope only: this does not show real Builder consumption,
// a selected product Target, Provider I/O, acceptance, or Studio self-build.
