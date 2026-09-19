package postgresquery

import (
	"context"
	"database/sql"
	sqldriver "database/sql/driver"
	"errors"
	"io"
	"reflect"
	"testing"
)

type contractRunnerTestState struct {
	query    string
	args     []sqldriver.NamedValue
	columns  []string
	rows     [][]sqldriver.Value
	queryErr error
}

type contractRunnerTestConnector struct{ state *contractRunnerTestState }

func (c contractRunnerTestConnector) Connect(context.Context) (sqldriver.Conn, error) {
	return &contractRunnerTestConn{state: c.state}, nil
}
func (c contractRunnerTestConnector) Driver() sqldriver.Driver { return contractRunnerTestDriver{} }

type contractRunnerTestDriver struct{}

func (contractRunnerTestDriver) Open(string) (sqldriver.Conn, error) {
	return nil, errors.New("contractRunnerTestConnectorを利用してください")
}

type contractRunnerTestConn struct{ state *contractRunnerTestState }

func (c *contractRunnerTestConn) Prepare(string) (sqldriver.Stmt, error) {
	return nil, sqldriver.ErrSkip
}
func (c *contractRunnerTestConn) Close() error                 { return nil }
func (c *contractRunnerTestConn) Begin() (sqldriver.Tx, error) { return nil, sqldriver.ErrSkip }
func (c *contractRunnerTestConn) QueryContext(_ context.Context, query string, args []sqldriver.NamedValue) (sqldriver.Rows, error) {
	c.state.query = query
	c.state.args = append([]sqldriver.NamedValue(nil), args...)
	if c.state.queryErr != nil {
		return nil, c.state.queryErr
	}
	rows := make([][]sqldriver.Value, len(c.state.rows))
	for i := range c.state.rows {
		rows[i] = append([]sqldriver.Value(nil), c.state.rows[i]...)
	}
	return &contractRunnerTestRows{columns: append([]string(nil), c.state.columns...), rows: rows}, nil
}

var _ sqldriver.QueryerContext = (*contractRunnerTestConn)(nil)

type contractRunnerTestRows struct {
	columns []string
	rows    [][]sqldriver.Value
	index   int
}

func (r *contractRunnerTestRows) Columns() []string { return r.columns }
func (r *contractRunnerTestRows) Close() error      { return nil }
func (r *contractRunnerTestRows) Next(dest []sqldriver.Value) error {
	if r.index >= len(r.rows) {
		return io.EOF
	}
	copy(dest, r.rows[r.index])
	r.index++
	return nil
}

func openContractRunnerTestDB(t *testing.T, state *contractRunnerTestState) *sql.DB {
	t.Helper()
	db := sql.OpenDB(contractRunnerTestConnector{state: state})
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestContractRunnerBindsNamedParametersBeforePostgreSQLExecution(t *testing.T) {
	bytesValue := []byte("SKU-01")
	state := &contractRunnerTestState{
		columns: []string{"order_line_id", "item_code"},
		rows:    [][]sqldriver.Value{{int64(11), bytesValue}},
	}
	runner := Runner{DB: openContractRunnerTestDB(t, state)}
	const source = "select order_line_id, item_code from order_line where order_id = :parent_id or original_order_id = :parent_id"

	raw, err := runner.ExecuteSQL(context.Background(), source, map[string]any{"parent_id": int64(1001)})
	if err != nil {
		t.Fatalf("明示されたPostgreSQL SQLを実行できません: %v", err)
	}
	if state.query != "select order_line_id, item_code from order_line where order_id = $1 or original_order_id = $1" {
		t.Fatalf("named parameterをPostgreSQL placeholderへ正しく変換できていません: %q", state.query)
	}
	if len(state.args) != 1 || state.args[0].Value != int64(1001) {
		t.Fatalf("physical parameter値を一度だけ束縛できていません: %#v", state.args)
	}
	if !reflect.DeepEqual(raw.Result.Columns, []string{"order_line_id", "item_code"}) || len(raw.Result.Rows) != 1 {
		t.Fatalf("raw resultが変わっています: %#v", raw.Result)
	}
	itemCode, ok := raw.Result.Rows[0][1].([]byte)
	if !ok || string(itemCode) != "SKU-01" {
		t.Fatalf("raw []byte値を保持できていません: %#v", raw.Result.Rows[0][1])
	}
	bytesValue[0] = 'X'
	if string(itemCode) != "SKU-01" {
		t.Fatalf("driver側bufferの変更が返却済みraw resultへ漏れています: %q", string(itemCode))
	}
	payload, ok := raw.Payload.(Result)
	if !ok || !reflect.DeepEqual(payload.ParameterNames, []string{"parent_id"}) {
		t.Fatalf("runtime値を含めずparameter名だけをEvidenceへ残せていません: %#v", raw.Payload)
	}
}

func TestContractRunnerKeepsZeroRowsAsNormalResult(t *testing.T) {
	state := &contractRunnerTestState{columns: []string{"order_line_id"}}
	raw, err := (Runner{DB: openContractRunnerTestDB(t, state)}).ExecuteSQL(
		context.Background(),
		"select order_line_id from order_line where order_id = :parent_id",
		map[string]any{"parent_id": int64(999)},
	)
	if err != nil {
		t.Fatalf("0 rowをExecution Errorへ変換しています: %v", err)
	}
	if len(raw.Result.Rows) != 0 {
		t.Fatalf("0 rowのraw resultが変わっています: %#v", raw.Result.Rows)
	}
}

func TestContractRunnerKeepsQueryFailureAsExecutionError(t *testing.T) {
	state := &contractRunnerTestState{queryErr: errors.New("postgres unavailable")}
	_, err := (Runner{DB: openContractRunnerTestDB(t, state)}).ExecuteSQL(
		context.Background(),
		"select order_line_id from order_line where order_id = :parent_id",
		map[string]any{"parent_id": int64(1001)},
	)
	if err == nil || !errors.Is(err, state.queryErr) {
		t.Fatalf("Database実行失敗をExecution Errorとして保持できていません: %v", err)
	}
}

func TestContractRunnerRejectsUnusedPhysicalParameter(t *testing.T) {
	state := &contractRunnerTestState{}
	_, err := (Runner{DB: openContractRunnerTestDB(t, state)}).ExecuteSQL(
		context.Background(),
		"select 1",
		map[string]any{"parent_id": int64(1001)},
	)
	if err == nil {
		t.Fatal("SQL sourceで使用していないphysical parameterを黙って無視しています")
	}
}

func TestContractRunnerRequiresInjectedDatabaseAndSQL(t *testing.T) {
	if _, err := (Runner{}).ExecuteSQL(context.Background(), "select 1", nil); err == nil {
		t.Fatal("Database接続を暗黙生成しています")
	}
	runner := Runner{DB: openContractRunnerTestDB(t, &contractRunnerTestState{})}
	if _, err := runner.ExecuteSQL(context.Background(), "   ", nil); err == nil {
		t.Fatal("空のnative SQLを実行しています")
	}
}
