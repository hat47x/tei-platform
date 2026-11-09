package queryinputnullability

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5"
)

const PostgresEvidenceSchema = "tei.reference.query-input-nullability-postgres-evidence/v0"

type PostgresCase struct {
	Name             string         `json:"name"`
	RequestID        string         `json:"requestId"`
	Rejected         []Diagnostic   `json:"rejected,omitempty"`
	QueryNotExecuted bool           `json:"queryNotExecuted"`
	SQLParameters    map[string]any `json:"sqlParameters"`
	ObservedNull     *bool          `json:"observedNull,omitempty"`
	ObservedValue    *string        `json:"observedValue,omitempty"`
	Passed           bool           `json:"passed"`
	Detail           string         `json:"detail,omitempty"`
}

type PostgresReceipt struct {
	Schema           string         `json:"schema"`
	EvidenceNature   string         `json:"evidenceNature"`
	DatabaseProvider string         `json:"databaseProvider"`
	ContractID       string         `json:"contractId"`
	ContractVersion  string         `json:"contractVersion"`
	Cases            []PostgresCase `json:"cases"`
	Passed           bool           `json:"passed"`
	Notes            []string       `json:"notes"`
}

// RunPostgresEvidence binds the reference contract, then performs the allowed
// and rejected cases against an ephemeral PostgreSQL database. It does not prove
// production behavior or equivalence with other Providers.
func RunPostgresEvidence(ctx context.Context, dsn, contractPath string) (PostgresReceipt, error) {
	contract, err := LoadContract(contractPath)
	if err != nil {
		return PostgresReceipt{}, err
	}
	receipt := PostgresReceipt{
		Schema: PostgresEvidenceSchema, EvidenceNature: "real-postgres-ephemeral-fixture",
		DatabaseProvider: "postgres", ContractID: contract.ID, ContractVersion: contract.Version,
		Cases: []PostgresCase{},
		Notes: []string{
			"一時PostgreSQLのtemp tableに対する実bind・NULL観測である。",
			"DSN、実顧客データ、本番DB、Provider間の同等性は記録・主張しない。",
			"拒否caseはbinderで停止し、SQLを実行していないことを行数0で確認する。",
		},
	}
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return receipt, err
	}
	defer conn.Close(context.Background())
	if _, err := conn.Exec(ctx, `CREATE TEMP TABLE tei_query_nullability_fixture (
		customer_id text PRIMARY KEY,
		nickname text
	)`); err != nil {
		return receipt, err
	}
	defer conn.Exec(context.Background(), `DROP TABLE IF EXISTS tei_query_nullability_fixture`)

	cases := []struct {
		name      string
		request   Request
		insert    string
		args      []any
		countID   string
		wantNull  *bool
		wantValue *string
	}{
		{
			name: "explicit-null-allowed",
			request: Request{Inputs: map[string]json.RawMessage{
				"customer_id": json.RawMessage(`"C-NULL"`), "nickname": json.RawMessage(`null`),
			}},
			insert:   `INSERT INTO tei_query_nullability_fixture (customer_id, nickname) VALUES ($1, $2)`,
			args:     []any{"C-NULL", nil},
			wantNull: boolPointer(true),
		},
		{
			name: "omitted-optional",
			request: Request{Inputs: map[string]json.RawMessage{
				"customer_id": json.RawMessage(`"C-OMITTED"`),
			}},
			insert:   `INSERT INTO tei_query_nullability_fixture (customer_id) VALUES ($1)`,
			args:     []any{"C-OMITTED"},
			wantNull: boolPointer(true),
		},
		{
			name: "empty-string-is-value",
			request: Request{Inputs: map[string]json.RawMessage{
				"customer_id": json.RawMessage(`"C-EMPTY"`), "nickname": json.RawMessage(`""`),
			}},
			insert:    `INSERT INTO tei_query_nullability_fixture (customer_id, nickname) VALUES ($1, $2)`,
			args:      []any{"C-EMPTY", ""},
			wantNull:  boolPointer(false),
			wantValue: stringPointer(""),
		},
		{
			name: "forbidden-null",
			request: Request{Inputs: map[string]json.RawMessage{
				"customer_id": json.RawMessage(`"C-FORBIDDEN"`), "region": json.RawMessage(`null`),
			}},
			countID: "C-FORBIDDEN",
		},
		{
			name: "missing-required",
			request: Request{Inputs: map[string]json.RawMessage{
				"nickname": json.RawMessage(`"x"`),
			}},
			countID: "C-MISSING",
		},
	}
	for _, testCase := range cases {
		bound := Bind(contract, testCase.request)
		caseReceipt := PostgresCase{
			Name: testCase.name, RequestID: testCase.request.ID, SQLParameters: bound.SQLParameters,
			Rejected: bound.Rejected, QueryNotExecuted: bound.QueryNotExecuted,
		}
		if !bound.Passed {
			var count int
			if err := conn.QueryRow(ctx, `SELECT count(*) FROM tei_query_nullability_fixture WHERE customer_id = $1`,
				testCase.countID).Scan(&count); err != nil {
				return receipt, err
			}
			caseReceipt.Passed = len(bound.Rejected) > 0 && count == 0
			caseReceipt.Detail = "rejected before SQL; zero rows observed"
			receipt.Cases = append(receipt.Cases, caseReceipt)
			continue
		}
		if _, err := conn.Exec(ctx, testCase.insert, testCase.args...); err != nil {
			return receipt, err
		}
		var nickname *string
		if err := conn.QueryRow(ctx, `SELECT nickname FROM tei_query_nullability_fixture WHERE customer_id = $1`,
			testCase.args[0]).Scan(&nickname); err != nil {
			return receipt, err
		}
		isNull := nickname == nil
		caseReceipt.ObservedNull = &isNull
		caseReceipt.ObservedValue = nickname
		caseReceipt.Passed = isNull == *testCase.wantNull
		if testCase.wantValue != nil {
			caseReceipt.Passed = caseReceipt.Passed && nickname != nil && *nickname == *testCase.wantValue
		}
		receipt.Cases = append(receipt.Cases, caseReceipt)
	}
	receipt.Passed = true
	for _, caseReceipt := range receipt.Cases {
		if !caseReceipt.Passed {
			receipt.Passed = false
		}
	}
	return receipt, nil
}

func boolPointer(value bool) *bool {
	return &value
}

func stringPointer(value string) *string {
	return &value
}
