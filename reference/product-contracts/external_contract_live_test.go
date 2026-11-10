package productcontracts_test

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// consumedShape holds only the parts of EKI's contract this pin claims: for one
// operation, the method, the accepted status, the accepted schema name, its
// required keys and the constants of the two fields the bindings read. Copying
// the whole schema would reproduce EKI's document, which the pin must not do.
type consumedShape struct {
	Method            string
	AcceptedStatus    int
	AcceptedSchema    string
	AcceptedSchemaRef string
	RequiredKeys      []string
	SchemaConst       string
	StatusConst       string
}

// readConsumedShape pulls the consumed part of one operation out of an
// OpenAPI document.
func readConsumedShape(spec []byte, operationPath string) (consumedShape, error) {
	var document struct {
		Paths map[string]map[string]struct {
			Responses map[string]struct {
				Content map[string]struct {
					Schema struct {
						Ref string `json:"$ref"`
					} `json:"schema"`
				} `json:"content"`
			} `json:"responses"`
		} `json:"paths"`
		Components struct {
			Schemas map[string]struct {
				Required   []string                   `json:"required"`
				Properties map[string]json.RawMessage `json:"properties"`
			} `json:"schemas"`
		} `json:"components"`
	}
	if err := json.Unmarshal(spec, &document); err != nil {
		return consumedShape{}, fmt.Errorf("decode the OpenAPI document: %w", err)
	}
	methods, ok := document.Paths[operationPath]
	if !ok {
		return consumedShape{}, fmt.Errorf("the document has no %s", operationPath)
	}
	post, ok := methods["post"]
	if !ok {
		return consumedShape{}, fmt.Errorf("%s has no post operation", operationPath)
	}
	accepted, ok := post.Responses["202"]
	if !ok {
		return consumedShape{}, fmt.Errorf("%s has no 202 response", operationPath)
	}
	content, ok := accepted.Content["application/json"]
	if !ok {
		return consumedShape{}, fmt.Errorf("%s 202 response is not application/json", operationPath)
	}
	const prefix = "#/components/schemas/"
	if !strings.HasPrefix(content.Schema.Ref, prefix) {
		return consumedShape{}, fmt.Errorf("%s 202 schema is not a component reference: %q",
			operationPath, content.Schema.Ref)
	}
	name := strings.TrimPrefix(content.Schema.Ref, prefix)
	schema, ok := document.Components.Schemas[name]
	if !ok {
		return consumedShape{}, fmt.Errorf("the 202 reference names %q, which is not defined", name)
	}
	shape := consumedShape{
		Method: "POST", AcceptedStatus: 202, AcceptedSchema: name,
		AcceptedSchemaRef: content.Schema.Ref,
		RequiredKeys:      append([]string(nil), schema.Required...),
	}
	for _, field := range []struct {
		key    string
		target *string
	}{{"schema", &shape.SchemaConst}, {"status", &shape.StatusConst}} {
		raw, ok := schema.Properties[field.key]
		if !ok {
			continue
		}
		var property struct {
			Const *string `json:"const"`
		}
		if err := json.Unmarshal(raw, &property); err == nil && property.Const != nil {
			*field.target = *property.Const
		}
	}
	slices.Sort(shape.RequiredKeys)
	return shape, nil
}

// contractDrift compares the pin with an EKI checkout and reports why it does
// not hold. It returns nil only when the pinned revision matches the pin and the
// consumed subset is unchanged at HEAD.
func contractDrift(t *testing.T, repo string, pin externalContractPin) error {
	t.Helper()
	if !revisionExists(t, repo, pin.Producer.Commit) {
		return fmt.Errorf("the pinned revision %q does not exist in %s", pin.Producer.Commit, repo)
	}
	pinned, err := specAt(t, repo, pin.Producer.Commit, pin.Producer.SpecPath)
	if err != nil {
		return err
	}
	head, err := specAt(t, repo, "HEAD", pin.Producer.SpecPath)
	if err != nil {
		return err
	}
	for _, operation := range pin.Operations {
		atPin, err := readConsumedShape(pinned, operation.Path)
		if err != nil {
			return err
		}
		// The pin must describe the revision it names: a mismatch here is a
		// wrong record, not a contract change.
		if atPin.Method != operation.Method {
			return fmt.Errorf("%s at %s is %s, pinned as %s",
				operation.Path, pin.Producer.Commit, atPin.Method, operation.Method)
		}
		if atPin.AcceptedSchema != operation.AcceptedSchema {
			return fmt.Errorf("%s 202 at %s is %s, pinned as %s",
				operation.Path, pin.Producer.Commit, atPin.AcceptedSchema, operation.AcceptedSchema)
		}
		want := append([]string(nil), operation.AcceptedRequiredKeys...)
		slices.Sort(want)
		if !slices.Equal(atPin.RequiredKeys, want) {
			return fmt.Errorf("%s required keys at %s = %v, pinned as %v",
				operation.Path, pin.Producer.Commit, atPin.RequiredKeys, want)
		}
		if atPin.SchemaConst != operation.AcceptedSchemaConst {
			return fmt.Errorf("%s schema const at %s = %q, pinned as %q",
				operation.Path, pin.Producer.Commit, atPin.SchemaConst, operation.AcceptedSchemaConst)
		}
		if atPin.StatusConst != operation.AcceptedStatusConst {
			return fmt.Errorf("%s status const at %s = %q, pinned as %q",
				operation.Path, pin.Producer.Commit, atPin.StatusConst, operation.AcceptedStatusConst)
		}
		// The part the issue is about: has the contract moved since the pin?
		atHead, err := readConsumedShape(head, operation.Path)
		if err != nil {
			return err
		}
		if !slices.Equal(atHead.RequiredKeys, atPin.RequiredKeys) ||
			atHead.SchemaConst != atPin.SchemaConst ||
			atHead.StatusConst != atPin.StatusConst ||
			atHead.AcceptedSchema != atPin.AcceptedSchema {
			return fmt.Errorf("EKI's %s changed since %s: pinned %v/%q/%q, HEAD %v/%q/%q; "+
				"re-verify the bindings, then update external-contract-pin.json, "+
				"bindings/testdata/task_accepted.json and the README together",
				operation.Path, pin.Producer.Commit,
				atPin.RequiredKeys, atPin.SchemaConst, atPin.StatusConst,
				atHead.RequiredKeys, atHead.SchemaConst, atHead.StatusConst)
		}
	}
	return nil
}

func specAt(t *testing.T, repo, revision, specPath string) ([]byte, error) {
	t.Helper()
	command := exec.Command("git", "-C", repo, "show", revision+":"+specPath)
	out, err := command.Output()
	if err != nil {
		var exit *exec.ExitError
		if ok := asExitError(err, &exit); ok {
			return nil, fmt.Errorf("git show %s:%s in %s: %w: %s",
				revision, specPath, repo, err, strings.TrimSpace(string(exit.Stderr)))
		}
		return nil, fmt.Errorf("git show %s:%s in %s: %w", revision, specPath, repo, err)
	}
	return out, nil
}

func asExitError(err error, target **exec.ExitError) bool {
	if exit, ok := err.(*exec.ExitError); ok {
		*target = exit
		return true
	}
	return false
}

func revisionExists(t *testing.T, repo, revision string) bool {
	t.Helper()
	command := exec.Command("git", "-C", repo, "cat-file", "-e", revision+"^{commit}")
	return command.Run() == nil
}

func runGit(t *testing.T, repo string, args ...string) string {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", repo}, args...)...)
	out, err := command.Output()
	if err != nil {
		t.Fatalf("git %v in %s: %v", args, repo, err)
	}
	return string(out)
}

// TestLiveExternalContractPin compares the pin with a real EKI checkout. It is
// skipped unless EKI_CONTRACT_REPO is set, because TEI's CI has no EKI checkout
// and requiring one would make the health run depend on another repository
// (#3836). A skip is not evidence that the pin still holds; the manual record in
// the README is where a comparison is written down.
func TestLiveExternalContractPin(t *testing.T) {
	repo := strings.TrimSpace(os.Getenv("EKI_CONTRACT_REPO"))
	if repo == "" {
		t.Skip("EKI_CONTRACT_REPO is not set; the pin is compared with a real EKI checkout only on request")
	}
	if _, err := os.Stat(repo); err != nil {
		t.Fatalf("EKI_CONTRACT_REPO=%q is not usable: %v", repo, err)
	}
	pin := loadExternalContractPin(t)
	if err := contractDrift(t, repo, pin); err != nil {
		t.Fatalf("the pin no longer holds: %v", err)
	}
	head := strings.TrimSpace(runGit(t, repo, "rev-parse", "HEAD"))
	t.Logf("pin holds: %s at %s, and the consumed subset is unchanged at HEAD %s",
		pin.Producer.SpecPath, pin.Producer.Commit, head)
}

// The comparison is only useful if it can fail. These cases build a synthetic
// repository (no EKI content) and show that each drift the pin is meant to
// catch is reported.
func TestContractDriftDetectsEachChange(t *testing.T) {
	pin := externalContractPin{Producer: struct {
		Repository     string `json:"repository"`
		Commit         string `json:"commit"`
		SpecPath       string `json:"specPath"`
		OpenAPIVersion string `json:"openapiVersion"`
		APIVersion     string `json:"apiVersion"`
	}{Commit: "pinned", SpecPath: "spec/openapi.json"}}
	pin.Operations = append(pin.Operations, struct {
		OperationID          string   `json:"operationId"`
		Method               string   `json:"method"`
		Path                 string   `json:"path"`
		RequestSchema        string   `json:"requestSchema"`
		RequestSchemaConst   string   `json:"requestSchemaConst"`
		AcceptedStatus       int      `json:"acceptedStatus"`
		AcceptedSchema       string   `json:"acceptedSchema"`
		AcceptedSchemaConst  string   `json:"acceptedSchemaConst"`
		AcceptedRequiredKeys []string `json:"acceptedRequiredKeys"`
		AcceptedStatusConst  string   `json:"acceptedStatusConst"`
	}{
		Method: "POST", Path: "/v1/tasks", AcceptedStatus: 202,
		AcceptedSchema: "TaskAccepted", AcceptedSchemaConst: "eki.task-accepted/v1",
		AcceptedRequiredKeys: []string{"schema", "task_id", "status"},
		AcceptedStatusConst:  "queued",
	})

	cases := []struct {
		name       string
		pinnedSpec string
		headSpec   string
		wantIn     string
	}{
		{
			name:       "unchanged",
			pinnedSpec: syntheticSpec("eki.task-accepted/v1", "queued", "schema", "task_id", "status"),
			headSpec:   syntheticSpec("eki.task-accepted/v1", "queued", "schema", "task_id", "status"),
		},
		{
			name:       "required-key-added-at-head",
			pinnedSpec: syntheticSpec("eki.task-accepted/v1", "queued", "schema", "task_id", "status"),
			headSpec:   syntheticSpec("eki.task-accepted/v1", "queued", "schema", "task_id", "status", "attempt_id"),
			wantIn:     "changed since",
		},
		{
			name:       "status-const-changed-at-head",
			pinnedSpec: syntheticSpec("eki.task-accepted/v1", "queued", "schema", "task_id", "status"),
			headSpec:   syntheticSpec("eki.task-accepted/v1", "running", "schema", "task_id", "status"),
			wantIn:     "changed since",
		},
		{
			name:       "schema-const-changed-at-head",
			pinnedSpec: syntheticSpec("eki.task-accepted/v1", "queued", "schema", "task_id", "status"),
			headSpec:   syntheticSpec("eki.task-accepted/v2", "queued", "schema", "task_id", "status"),
			wantIn:     "changed since",
		},
		{
			// The pin itself is wrong: the revision it names does not say this.
			name:       "pin-disagrees-with-its-own-revision",
			pinnedSpec: syntheticSpec("eki.task-accepted/v1", "queued", "schema", "task_id"),
			headSpec:   syntheticSpec("eki.task-accepted/v1", "queued", "schema", "task_id"),
			wantIn:     "pinned as",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			repo := syntheticCheckout(t, testCase.pinnedSpec, testCase.headSpec)
			err := contractDrift(t, repo, pin)
			if testCase.wantIn == "" {
				if err != nil {
					t.Fatalf("an unchanged contract was reported as drift: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("drift was not detected")
			}
			if !strings.Contains(err.Error(), testCase.wantIn) {
				t.Fatalf("the reason does not mention %q: %v", testCase.wantIn, err)
			}
		})
	}
}

// A missing pinned revision is a broken record, and must be reported rather than
// treated as "nothing to compare".
func TestContractDriftReportsAnUnknownRevision(t *testing.T) {
	repo := syntheticCheckout(t,
		syntheticSpec("eki.task-accepted/v1", "queued", "schema", "task_id", "status"),
		syntheticSpec("eki.task-accepted/v1", "queued", "schema", "task_id", "status"))
	pin := loadExternalContractPin(t)
	pin.Producer.Commit = "0123456789abcdef0123456789abcdef01234567"
	err := contractDrift(t, repo, pin)
	if err == nil || !strings.Contains(err.Error(), "does not exist") {
		t.Fatalf("an unknown revision was not reported: %v", err)
	}
}

// syntheticSpec builds a minimal OpenAPI document with only the parts the pin
// reads, so the detection can be tested without reproducing EKI's document.
func syntheticSpec(schemaConst, statusConst string, required ...string) string {
	properties := map[string]any{}
	for _, key := range required {
		switch key {
		case "schema":
			properties[key] = map[string]any{"type": "string", "const": schemaConst}
		case "status":
			properties[key] = map[string]any{"type": "string", "const": statusConst}
		default:
			properties[key] = map[string]any{"type": "string"}
		}
	}
	document := map[string]any{
		"openapi": "3.2.1",
		"paths": map[string]any{
			"/v1/tasks": map[string]any{
				"post": map[string]any{
					"responses": map[string]any{
						"202": map[string]any{
							"content": map[string]any{
								"application/json": map[string]any{
									"schema": map[string]any{"$ref": "#/components/schemas/TaskAccepted"},
								},
							},
						},
					},
				},
			},
		},
		"components": map[string]any{
			"schemas": map[string]any{
				"TaskAccepted": map[string]any{
					"type":       "object",
					"required":   required,
					"properties": properties,
				},
			},
		},
	}
	raw, err := json.Marshal(document)
	if err != nil {
		panic(err)
	}
	return string(raw)
}

// syntheticCheckout makes a repository whose pinned revision and HEAD hold the
// given documents, with the pinned revision tagged so `git show pinned:...`
// resolves.
func syntheticCheckout(t *testing.T, pinnedSpec, headSpec string) string {
	t.Helper()
	repo := t.TempDir()
	runGit(t, repo, "init", "-q")
	runGit(t, repo, "config", "user.name", "TEI Test")
	runGit(t, repo, "config", "user.email", "test@example.invalid")
	if err := os.MkdirAll(filepath.Join(repo, "spec"), 0o755); err != nil {
		t.Fatal(err)
	}
	specPath := filepath.Join(repo, "spec", "openapi.json")
	if err := os.WriteFile(specPath, []byte(pinnedSpec), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, repo, "add", "spec/openapi.json")
	runGit(t, repo, "commit", "-qm", "pinned")
	// The pin names the revision "pinned", which is a tag rather than a hash, so
	// the synthetic checkout resolves it the same way a short hash resolves.
	runGit(t, repo, "tag", "pinned")
	if err := os.WriteFile(specPath, []byte(headSpec), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, repo, "add", "spec/openapi.json")
	// The head may hold the same bytes as the pinned revision; that case is the
	// "unchanged" one, so the commit must be allowed to be empty.
	runGit(t, repo, "commit", "-qm", "head", "--allow-empty")
	return repo
}
