package main

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"

	// ADR-0007に従い、Oracle ドライバーの選択と登録はHost / Reference assembly（このtool）だけが行う。
	// internal/配下のCoreやOracle Provider / QueryRuntimeはこのドライバーをimportしない。
	go_ora "github.com/sijms/go-ora/v2"
)

const setTransactionReadOnly = "SET TRANSACTION READ ONLY"

// newDBは、go-ora/v2 ドライバーのconnectorからdatabase/sql接続を作る。
// readOnlyTransactionがtrueなら、各物理接続の最初のSQLとして読取り専用transactionを開始する。
func newDB(dsn string, readOnlyTransaction bool) *sql.DB {
	var connector driver.Connector = go_ora.NewConnector(dsn)
	if readOnlyTransaction {
		connector = readOnlyConnector{inner: connector}
	}
	return sql.OpenDB(connector)
}

// readOnlyConnectorは、新しい物理接続ごとに`SET TRANSACTION READ ONLY`を、その接続の最初のSQLとして発行する。
//
// go-ora/v2は`database/sql`の`BeginTx`に`ReadOnly`を指定すると「readonly transaction is not supported」で拒否する。
// 既存のOracle Runnerは`*sql.DB`だけを受け取り、`QueryContext`しか呼ばない。そのためRunnerを変更せず、
// 接続を開く時点でSQLとして発行する方式を採る。
//
// 持続するtransactionは物理接続の寿命と同じため、pool内の同一接続では読取りsnapshotが接続開始時点に固定される。
// これは一時的な検証用データの確認用であり、production向けの接続・transaction方針を決めるものではない。
// 戻す接続は包むせず、go-ora/v2の接続をそのまま返す（ドライバーのinterface検出を保つため）。
type readOnlyConnector struct {
	inner driver.Connector
}

func (c readOnlyConnector) Connect(ctx context.Context) (driver.Conn, error) {
	conn, err := c.inner.Connect(ctx)
	if err != nil {
		return nil, err
	}
	if err := execOnConn(ctx, conn, setTransactionReadOnly); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("%sを発行できません: %w", setTransactionReadOnly, err)
	}
	return conn, nil
}

func (c readOnlyConnector) Driver() driver.Driver {
	return c.inner.Driver()
}

func execOnConn(ctx context.Context, conn driver.Conn, query string) error {
	if execer, ok := conn.(driver.ExecerContext); ok {
		_, err := execer.ExecContext(ctx, query, nil)
		return err
	}
	stmt, err := conn.Prepare(query)
	if err != nil {
		return err
	}
	defer stmt.Close()
	//nolint:staticcheck // driver.StmtExecContext非対応ドライバー向けの代替
	_, err = stmt.Exec(nil)
	return err
}
