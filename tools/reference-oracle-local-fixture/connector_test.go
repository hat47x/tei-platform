package main

import (
	"context"
	"database/sql/driver"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fakeConn struct {
	calls   *[]string
	execErr error
}

func (c *fakeConn) Prepare(string) (driver.Stmt, error) { return nil, errors.New("使いません") }
func (c *fakeConn) Close() error                        { *c.calls = append(*c.calls, "close"); return nil }
func (c *fakeConn) Begin() (driver.Tx, error)           { *c.calls = append(*c.calls, "begin"); return nil, nil }
func (c *fakeConn) ExecContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Result, error) {
	*c.calls = append(*c.calls, "exec:"+query)
	return nil, c.execErr
}

type fakeConnector struct{ conn driver.Conn }

func (f fakeConnector) Connect(context.Context) (driver.Conn, error) { return f.conn, nil }
func (f fakeConnector) Driver() driver.Driver                        { return nil }

func TestReadOnlyConnectorIssuesSetTransactionAsTheOnlyFirstSQL(t *testing.T) {
	var calls []string
	conn := &fakeConn{calls: &calls}
	got, err := readOnlyConnector{inner: fakeConnector{conn}}.Connect(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got != driver.Conn(conn) {
		t.Fatal("物理接続をwrapせず、そのまま返す必要があります（driverのinterface検出を保つため）")
	}
	if strings.Join(calls, "|") != "exec:SET TRANSACTION READ ONLY" {
		t.Fatalf("接続直後の呼び出し=%v。SET TRANSACTION READ ONLYだけを最初のSQLとして発行する必要があります", calls)
	}
}

func TestReadOnlyConnectorClosesConnectionWhenReadOnlyCannotBeEstablished(t *testing.T) {
	var calls []string
	failing := &fakeConn{calls: &calls, execErr: errors.New("ORA-00900")}
	if _, err := (readOnlyConnector{inner: fakeConnector{failing}}).Connect(context.Background()); err == nil {
		t.Fatal("SET TRANSACTION READ ONLYの失敗を握りつぶしています")
	}
	if calls[len(calls)-1] != "close" {
		t.Fatalf("read-onlyにできなかった接続を返さずcloseする必要があります: %v", calls)
	}
}

// ADR-0007: driver選択はHost / Reference assemblyだけが行う。internal/配下のCoreがdriverをimportしないことを確認する。
func TestOracleDriverIsImportedOnlyByThisToolNotByInternalCore(t *testing.T) {
	const driverPath = "github.com/sijms/go-ora"
	root := filepath.Join("..", "..", "internal")
	fset := token.NewFileSet()
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		file, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, spec := range file.Imports {
			if strings.Contains(spec.Path.Value, driverPath) {
				t.Errorf("internal/配下の%sがOracle driverをimportしています", path)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// 既存のOracle Runnerは読取り専用経路でありQueryContextだけを使う（#376）。
// Exec系・Begin系・Prepare系へ変更された場合に、このfixtureの前提が崩れることを検出する。
func TestExistingOracleRunnerUsesQueryContextOnly(t *testing.T) {
	path := filepath.Join("..", "..", "internal", "driver", "queryruntime", "oracle", "runner.go")
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	file, err := parser.ParseFile(token.NewFileSet(), path, source, 0)
	if err != nil {
		t.Fatal(err)
	}
	called := map[string]bool{}
	ast.Inspect(file, func(n ast.Node) bool {
		if sel, ok := n.(*ast.SelectorExpr); ok {
			called[sel.Sel.Name] = true
		}
		return true
	})
	if !called["QueryContext"] {
		t.Fatal("RunnerがQueryContextを使っていません。前提が変わっています")
	}
	for _, forbidden := range []string{"Exec", "ExecContext", "Begin", "BeginTx", "Prepare", "PrepareContext", "QueryRowContext", "Conn"} {
		if called[forbidden] {
			t.Fatalf("RunnerがQueryContext以外の%sを使っています", forbidden)
		}
	}
}
