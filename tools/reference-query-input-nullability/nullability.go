// Package queryinputnullability is a reference for distinguishing a missing
// Database Contract Query input from an explicit JSON null before SQL execution.
// It is a proposed contract shape, not an adopted product API.
package queryinputnullability

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

const (
	ContractSchema = "tei.reference.query-input-nullability-contract/v0"
	RequestSchema  = "tei.reference.query-input-nullability-request/v0"
	ReceiptSchema  = "tei.reference.query-input-nullability-receipt/v0"
)

const (
	PresenceRequired = "required"
	PresenceOptional = "optional"

	NullForbidden = "forbidden"
	NullAllowed   = "allowed"
)

type Input struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Presence string `json:"presence"`
	Null     string `json:"null"`
}

type Contract struct {
	Schema  string  `json:"schema"`
	ID      string  `json:"id"`
	Version string  `json:"version"`
	Inputs  []Input `json:"inputs"`
}

type Request struct {
	Schema string                     `json:"schema"`
	ID     string                     `json:"id"`
	Inputs map[string]json.RawMessage `json:"inputs"`
}

type BoundInput struct {
	Name     string  `json:"name"`
	Type     string  `json:"type"`
	State    string  `json:"state"`
	Value    *string `json:"value,omitempty"`
	SQLValue any     `json:"sqlValue"`
	Bound    bool    `json:"bound"`
}

type Diagnostic struct {
	Code   string `json:"code"`
	Input  string `json:"input,omitempty"`
	Detail string `json:"detail,omitempty"`
}

type Receipt struct {
	Schema           string         `json:"schema"`
	ContractID       string         `json:"contractId"`
	ContractVersion  string         `json:"contractVersion"`
	RequestID        string         `json:"requestId"`
	Inputs           []BoundInput   `json:"inputs"`
	SQLParameters    map[string]any `json:"sqlParameters"`
	Rejected         []Diagnostic   `json:"rejected,omitempty"`
	Passed           bool           `json:"passed"`
	QueryNotExecuted bool           `json:"queryNotExecuted"`
	Notes            []string       `json:"notes"`
}

func readJSON(path string, into any) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(into); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	if decoder.More() {
		return fmt.Errorf("%s: trailing JSON value", path)
	}
	return nil
}

func LoadContract(path string) (Contract, error) {
	var contract Contract
	if err := readJSON(path, &contract); err != nil {
		return Contract{}, err
	}
	if err := ValidateContract(contract); err != nil {
		return Contract{}, err
	}
	return contract, nil
}

func LoadRequest(path string) (Request, error) {
	var request Request
	if err := readJSON(path, &request); err != nil {
		return Request{}, err
	}
	if err := ValidateRequest(request); err != nil {
		return Request{}, err
	}
	return request, nil
}

func ValidateContract(contract Contract) error {
	if contract.Schema != ContractSchema {
		return fmt.Errorf("contract schema %q is not %q", contract.Schema, ContractSchema)
	}
	if strings.TrimSpace(contract.ID) == "" || strings.TrimSpace(contract.Version) == "" {
		return fmt.Errorf("contract id and version are required")
	}
	if len(contract.Inputs) == 0 {
		return fmt.Errorf("contract inputs must not be empty")
	}
	seen := map[string]bool{}
	for _, input := range contract.Inputs {
		if strings.TrimSpace(input.Name) == "" || strings.TrimSpace(input.Type) == "" {
			return fmt.Errorf("input name and type are required")
		}
		if seen[input.Name] {
			return fmt.Errorf("duplicate input %q", input.Name)
		}
		seen[input.Name] = true
		if input.Presence != PresenceRequired && input.Presence != PresenceOptional {
			return fmt.Errorf("input %s must declare presence %q or %q", input.Name, PresenceRequired, PresenceOptional)
		}
		if input.Null != NullForbidden && input.Null != NullAllowed {
			return fmt.Errorf("input %s must declare null %q or %q", input.Name, NullForbidden, NullAllowed)
		}
	}
	return nil
}

func ValidateRequest(request Request) error {
	if request.Schema != RequestSchema {
		return fmt.Errorf("request schema %q is not %q", request.Schema, RequestSchema)
	}
	if strings.TrimSpace(request.ID) == "" {
		return fmt.Errorf("request id is required")
	}
	return nil
}

// Bind decides whether each declared input is present, missing, or explicit
// null. It never coerces a JSON number, boolean, object, or array into a string.
// A rejected request produces no SQL parameters, so a caller cannot execute it
// accidentally.
func Bind(contract Contract, request Request) Receipt {
	receipt := Receipt{
		Schema: ReceiptSchema, ContractID: contract.ID, ContractVersion: contract.Version,
		RequestID: request.ID, Inputs: []BoundInput{}, SQLParameters: map[string]any{},
		Notes: []string{
			"missing inputと明示nullを別の状態として扱う参照binderです。",
			"拒否時はSQL parameterを作らず、SQL実行前に停止する想定です。",
			"実PostgreSQLのNULL比較・型復元・Provider差は、このreceiptでは検証していません。",
			"値の暗黙変換は行いません。JSON stringだけをpresentとして受け付けます。",
		},
	}
	var rejected []Diagnostic
	params := map[string]any{}
	declared := map[string]bool{}
	for _, input := range contract.Inputs {
		declared[input.Name] = true
		raw, exists := request.Inputs[input.Name]
		bound := BoundInput{Name: input.Name, Type: input.Type, Bound: false}
		switch {
		case !exists:
			bound.State = "missing"
			if input.Presence == PresenceRequired {
				rejected = append(rejected, Diagnostic{
					Code: "MISSING_REQUIRED_INPUT", Input: input.Name,
					Detail: "required input is absent",
				})
			}
		case isJSONNull(raw):
			bound.State = "null"
			if input.Null == NullForbidden {
				rejected = append(rejected, Diagnostic{
					Code: "NULL_NOT_ALLOWED", Input: input.Name,
					Detail: "explicit null is forbidden by the contract",
				})
				break
			}
			bound.Bound = true
			bound.SQLValue = nil
			params[input.Name] = nil
		default:
			var value string
			if err := json.Unmarshal(raw, &value); err != nil {
				bound.State = "unsupported-value"
				rejected = append(rejected, Diagnostic{
					Code: "UNSUPPORTED_VALUE", Input: input.Name,
					Detail: "only a JSON string is accepted; no implicit coercion",
				})
				break
			}
			bound.State = "present"
			bound.Value = &value
			bound.Bound = true
			bound.SQLValue = value
			params[input.Name] = value
		}
		receipt.Inputs = append(receipt.Inputs, bound)
	}
	var unknown []string
	for name := range request.Inputs {
		if !declared[name] {
			unknown = append(unknown, name)
		}
	}
	sort.Strings(unknown)
	for _, name := range unknown {
		rejected = append(rejected, Diagnostic{
			Code: "UNKNOWN_INPUT", Input: name,
			Detail: "request input is not declared by the contract",
		})
	}
	if len(rejected) > 0 {
		receipt.Rejected = rejected
		receipt.Passed = false
		receipt.QueryNotExecuted = true
		receipt.SQLParameters = map[string]any{}
		return receipt
	}
	receipt.Passed = true
	receipt.SQLParameters = params
	return receipt
}

func BindFiles(contractPath, requestPath string) (Receipt, error) {
	contract, err := LoadContract(contractPath)
	if err != nil {
		return Receipt{}, err
	}
	request, err := LoadRequest(requestPath)
	if err != nil {
		return Receipt{}, err
	}
	return Bind(contract, request), nil
}

func isJSONNull(raw json.RawMessage) bool {
	return bytes.Equal(bytes.TrimSpace(raw), []byte("null"))
}
