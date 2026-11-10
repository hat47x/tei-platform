package productcontracts_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// externalContractPin is the machine-readable record of which EKI contract
// revision this reference workspace assumes, and which parts of it the bindings
// rely on (#3836).
//
// It exists because the assumption used to live only as prose in the README
// ("f8047c1 で確認"): nothing read it, so a later EKI change could make the
// claim stale without any test noticing. The pin is the single place the
// revision is stated; the README, the bindings and the optional live check all
// point at it.
type externalContractPin struct {
	Schema   string `json:"schema"`
	Producer struct {
		Repository     string `json:"repository"`
		Commit         string `json:"commit"`
		SpecPath       string `json:"specPath"`
		OpenAPIVersion string `json:"openapiVersion"`
		APIVersion     string `json:"apiVersion"`
	} `json:"producer"`
	ConsumedBy struct {
		Package string `json:"package"`
		Readme  string `json:"readme"`
	} `json:"consumedBy"`
	Operations []struct {
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
	} `json:"operations"`
	NotConsumed []string `json:"notConsumed"`
	Notes       []string `json:"notes"`
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	return filepath.Join("..", "..")
}

func loadExternalContractPin(t *testing.T) externalContractPin {
	t.Helper()
	path := filepath.Join(repositoryRoot(t), "reference", "product-contracts", "external-contract-pin.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read the external contract pin: %v", err)
	}
	var pin externalContractPin
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	// An unknown field means the file and this reader disagree about what was
	// pinned, which is exactly the drift the pin is supposed to prevent.
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&pin); err != nil {
		t.Fatalf("decode the external contract pin: %v", err)
	}
	return pin
}

// The pin must say enough to be checked, and must stay a *partial* record: the
// issue asks that EKI's contract body is not reproduced here.
func TestExternalContractPinRecordsTheRevisionAndThePartsReliedOn(t *testing.T) {
	pin := loadExternalContractPin(t)
	if pin.Schema != "tei.external-contract-pin/v0" {
		t.Fatalf("schema = %q", pin.Schema)
	}
	if pin.Producer.Repository == "" || pin.Producer.Commit == "" || pin.Producer.SpecPath == "" {
		t.Fatalf("the producer is not identified: %+v", pin.Producer)
	}
	// A short commit is what the README quotes; refuse a moving reference such
	// as "main" or a branch, which cannot pin anything.
	if strings.ContainsAny(pin.Producer.Commit, "/ ") || pin.Producer.Commit == "main" || pin.Producer.Commit == "HEAD" {
		t.Fatalf("the commit is not a fixed revision: %q", pin.Producer.Commit)
	}
	if len(pin.Producer.Commit) < 7 {
		t.Fatalf("the commit is too short to be unique: %q", pin.Producer.Commit)
	}
	if len(pin.Operations) == 0 {
		t.Fatal("no operation is pinned")
	}
	for _, operation := range pin.Operations {
		if operation.Method == "" || operation.Path == "" {
			t.Fatalf("an operation has no route: %+v", operation)
		}
		if operation.AcceptedSchema == "" || len(operation.AcceptedRequiredKeys) == 0 {
			t.Fatalf("%s %s does not record the accepted shape", operation.Method, operation.Path)
		}
		// The keys are the reason the pin is useful: they are what the binding
		// and the README disagree about today.
		if !contains(operation.AcceptedRequiredKeys, "task_id") ||
			!contains(operation.AcceptedRequiredKeys, "status") {
			t.Fatalf("%s %s does not name task_id and status: %v",
				operation.Method, operation.Path, operation.AcceptedRequiredKeys)
		}
		// Reproducing the schema body would go beyond a pin.
		if len(operation.AcceptedRequiredKeys) > 24 {
			t.Fatalf("%s %s looks like a full schema copy (%d keys)",
				operation.Method, operation.Path, len(operation.AcceptedRequiredKeys))
		}
	}
}

// The README is where a reader learns the revision, so it must quote the same
// one as the pin. Before this, the README was the only place it appeared and
// nothing tied it to anything.
func TestReadmeQuotesThePinnedRevision(t *testing.T) {
	pin := loadExternalContractPin(t)
	path := filepath.Join(repositoryRoot(t), pin.ConsumedBy.Readme)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", pin.ConsumedBy.Readme, err)
	}
	readme := string(raw)
	if !strings.Contains(readme, pin.Producer.Commit) {
		t.Fatalf("%s does not mention the pinned revision %q", pin.ConsumedBy.Readme, pin.Producer.Commit)
	}
	if !strings.Contains(readme, filepath.Base("external-contract-pin.json")) {
		t.Fatalf("%s does not point at the pin file, so a reader cannot check the revision",
			pin.ConsumedBy.Readme)
	}
}

// The pin must agree with what the bindings actually require. The 202 fixture
// in the bindings package is written by hand; if the contract gains a required
// key, the fixture and the pin have to move together.
func TestBindingsFixtureCoversEveryKeyThePinRequires(t *testing.T) {
	pin := loadExternalContractPin(t)
	fixturePath := filepath.Join(repositoryRoot(t), "reference", "product-contracts",
		"bindings", "testdata", "task_accepted.json")
	raw, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read %s: %v", fixturePath, err)
	}
	var accepted map[string]any
	if err := json.Unmarshal(raw, &accepted); err != nil {
		t.Fatalf("decode %s: %v", fixturePath, err)
	}
	for _, operation := range pin.Operations {
		for _, key := range operation.AcceptedRequiredKeys {
			if _, ok := accepted[key]; !ok {
				t.Errorf("the %s fixture is missing %q, which the pin says EKI requires",
					operation.AcceptedSchema, key)
			}
		}
		if got, ok := accepted["status"]; ok && operation.AcceptedStatusConst != "" {
			if got != operation.AcceptedStatusConst {
				t.Errorf("the fixture status = %v, but the pin says %q", got, operation.AcceptedStatusConst)
			}
		}
		if got, ok := accepted["schema"]; ok && operation.AcceptedSchemaConst != "" {
			if got != operation.AcceptedSchemaConst {
				t.Errorf("the fixture schema = %v, but the pin says %q", got, operation.AcceptedSchemaConst)
			}
		}
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
