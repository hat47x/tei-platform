package oraclequery

import (
	"context"
	"database/sql"
	sqldriver "database/sql/driver"
	"errors"
	"io"
	"reflect"
	"testing"
)

type runnerTestState struct {
	query    string
	args     []sqldriver.NamedValue
	columns  []string
	rows     [][]sqldriver.Value
	queryErr error
	rowErr   error
}

type runnerTestConnector struct {
	state *runnerTestState
}

func (c runnerTestConnector) Connect(context.Context) (sqldriver.Conn, error) {
	return &runnerTestConn{state: c.state}, nil
}

func (c runnerTestConnector) Driver() sqldriver.Driver {
	return runnerTestDriver{}
}

type runnerTestDriver struct{}

func (runnerTestDriver) Open(string) (sqldriver.Conn, error) {
	return nil, errors.New("runnerTestConnectorを利用してください")
}

type runnerTestConn struct {
	state *runnerTestState
}

func (c *runnerTestConn) Prepare(string) (sqldriver.Stmt, error) {
	return nil, sqldriver.ErrSkip
}

func (c *runnerTestConn) Close() error { return nil }

func (c *runnerTestConn) Begin() (sqldriver.Tx, error) {
	return nil, sqldriver.ErrSkip
}

func (c *runnerTestConn) QueryContext(
	_ context.Context,
	query string,
	args []sqldriver.NamedValue,
) (sqldriver.Rows, error) {
	c.state.query = query
	c.state.args = append([]sqldriver.NamedValue(nil), args...)
	if c.state.queryErr != nil {
		return nil, c.state.queryErr
	}
	rows := make([][]sqldriver.Value, len(c.state.rows))
	for i := range c.state.rows {
		rows[i] = append([]sqldriver.Value(nil), c.state.rows[i]...)
	}
	return &runnerTestRows{
		columns: append([]string(nil), c.state.columns...),
		rows:    rows,
		rowErr:  c.state.rowErr,
	}, nil
}

var _ sqldriver.QueryerContext = (*runnerTestConn)(nil)

type runnerTestRows struct {
	columns []string
	rows    [][]sqldriver.Value
	index   int
	rowErr  error
}

func (r *runnerTestRows) Columns() []string { return r.columns }
func (r *runnerTestRows) Close() error      { return nil }

func (r *runnerTestRows) Next(dest []sqldriver.Value) error {
	if r.index < len(r.rows) {
		copy(dest, r.rows[r.index])
		r.index++
		return nil
	}
	if r.rowErr != nil {
		err := r.rowErr
		r.rowErr = nil
		return err
	}
	return io.EOF
}

func openRunnerTestDB(t *testing.T, state *runnerTestState) *sql.DB {
	t.Helper()
	db := sql.OpenDB(runnerTestConnector{state: state})
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestRunnerExecutesExplicitSQLWithStableNamedParameters(t *testing.T) {
	bytesValue := []byte("Alice")
	state := &runnerTestState{
		columns: []string{"legacy_customer_id", "legacy_name"},
		rows: [][]sqldriver.Value{
			{int64(42), bytesValue},
		},
	}
	runner := Runner{DB: openRunnerTestDB(t, state)}
	const source = "select legacy_customer_id, legacy_name from observed_customer_view where legacy_customer_id = :legacy_id and tenant_code = :tenant"

	raw, err := runner.ExecuteSQL(
		context.Background(),
		source,
		map[string]any{
			"tenant":    "A",
			"legacy_id": int64(42),
		},
	)
	if err != nil {
		t.Fatalf("明示されたOracle SQLを実行できません: %v", err)
	}
	if state.query != source {
		t.Fatalf("native SQL sourceを変更しています: %q", state.query)
	}
	if len(state.args) != 2 || state.args[0].Name != "legacy_id" || state.args[1].Name != "tenant" {
		t.Fatalf("physical parameterを安定したnamed argumentとして渡せていません: %#v", state.args)
	}
	if state.args[0].Value != int64(42) || state.args[1].Value != "A" {
		t.Fatalf("physical parameter値が変わっています: %#v", state.args)
	}

	if !reflect.DeepEqual(raw.Result.Columns, []string{"legacy_customer_id", "legacy_name"}) {
		t.Fatalf("raw result列が変わっています: %#v", raw.Result.Columns)
	}
	if len(raw.Result.Rows) != 1 || raw.Result.Rows[0][0] != int64(42) {
		t.Fatalf("raw result rowが変わっています: %#v", raw.Result.Rows)
	}
	name, ok := raw.Result.Rows[0][1].([]byte)
	if !ok || string(name) != "Alice" {
		t.Fatalf("raw []byte値を保持できていません: %#v", raw.Result.Rows[0][1])
	}
	bytesValue[0] = 'X'
	if string(name) != "Alice" {
		t.Fatalf("driver側bufferの変更が返却済みraw resultへ漏れています: %q", string(name))
	}

	payload, ok := raw.Payload.(Result)
	if !ok {
		t.Fatalf("Oracle Runner Evidenceの型が変わっています: %#v", raw.Payload)
	}
	if !reflect.DeepEqual(payload.ParameterNames, []string{"legacy_id", "tenant"}) {
		t.Fatalf("Evidenceへ保存するparameter名が不安定です: %#v", payload.ParameterNames)
	}
	if !reflect.DeepEqual(payload.RawResultColumns, []string{"legacy_customer_id", "legacy_name"}) {
		t.Fatalf("Evidenceへ保存するraw result列が変わっています: %#v", payload.RawResultColumns)
	}
}

func TestRunnerKeepsZeroRowsAsNormalRawResult(t *testing.T) {
	state := &runnerTestState{columns: []string{"legacy_customer_id"}}
	runner := Runner{DB: openRunnerTestDB(t, state)}

	raw, err := runner.ExecuteSQL(
		context.Background(),
		"select legacy_customer_id from observed_customer_view where legacy_customer_id = :legacy_id",
		map[string]any{"legacy_id": int64(999)},
	)
	if err != nil {
		t.Fatalf("0 rowをExecution Errorへ変換しています: %v", err)
	}
	if len(raw.Result.Rows) != 0 {
		t.Fatalf("0 rowのraw resultが別の結果へ変わっています: %#v", raw.Result.Rows)
	}
}

func TestRunnerKeepsQueryFailureAsExecutionError(t *testing.T) {
	state := &runnerTestState{queryErr: errors.New("oracle unavailable")}
	runner := Runner{DB: openRunnerTestDB(t, state)}

	_, err := runner.ExecuteSQL(
		context.Background(),
		"select legacy_customer_id from observed_customer_view where legacy_customer_id = :legacy_id",
		map[string]any{"legacy_id": int64(42)},
	)
	if err == nil || !errors.Is(err, state.queryErr) {
		t.Fatalf("Database実行失敗をExecution Errorとして保持できていません: %v", err)
	}
}

func TestRunnerKeepsRowReadFailureAsExecutionError(t *testing.T) {
	state := &runnerTestState{
		columns: []string{"legacy_customer_id"},
		rows:    [][]sqldriver.Value{{int64(42)}},
		rowErr:  errors.New("row stream interrupted"),
	}
	runner := Runner{DB: openRunnerTestDB(t, state)}

	_, err := runner.ExecuteSQL(
		context.Background(),
		"select legacy_customer_id from observed_customer_view where legacy_customer_id = :legacy_id",
		map[string]any{"legacy_id": int64(42)},
	)
	if err == nil || !errors.Is(err, state.rowErr) {
		t.Fatalf("row取得中の失敗を正常結果として扱っています: %v", err)
	}
}

func TestRunnerRequiresInjectedDatabaseAndSQL(t *testing.T) {
	if _, err := (Runner{}).ExecuteSQL(context.Background(), "select 1 from dual", nil); err == nil {
		t.Fatal("Database接続を暗黙生成しています")
	}

	runner := Runner{DB: openRunnerTestDB(t, &runnerTestState{})}
	if _, err := runner.ExecuteSQL(context.Background(), "   ", nil); err == nil {
		t.Fatal("空のnative SQLを実行しています")
	}
}
