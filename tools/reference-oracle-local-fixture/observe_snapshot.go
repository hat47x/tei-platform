package main

import (
	"fmt"
	"time"
)

func (s *session) snapshotProbeSQL() string {
	return fmt.Sprintf("SELECT CUSTOMER_ID FROM %s WHERE CUSTOMER_ID = %d", s.fx.qualifiedTable(), idJapanese)
}

// waitForReadOnlySnapshotは、fixtureのDDL / GRANT直後にread-only transactionを開始すると
// ORA-01466（table definition has changed）で読めないことがあるため、読めるようになるまで待つ。
// 待った事実自体を観測として残す。
func (s *session) waitForReadOnlySnapshot() {
	const limit = 60 * time.Second
	started := time.Now()
	attempts := 0
	firstCode := ""
	for {
		attempts++
		db := newDB(s.fx.dsn(s.fx.ReadOnly), true)
		_, err := queryStrings(s.ctx, db, s.snapshotProbeSQL())
		db.Close()
		if err == nil {
			break
		}
		code := oraCode(err)
		if firstCode == "" {
			firstCode = firstNonEmpty(code, "error")
		}
		if code != "ORA-01466" || time.Since(started) > limit || s.ctx.Err() != nil {
			s.record("tx.readable-after-ddl-settles", layerTransaction,
				"fixture作成直後でも、待てばread-only transactionでfixture tableを読める", false, "last="+firstNonEmpty(code, s.errText(err)))
			return
		}
		time.Sleep(time.Second)
	}
	s.record("tx.readable-after-ddl-settles", layerTransaction,
		"fixture作成直後でも、待てばread-only transactionでfixture tableを読める", true,
		fmt.Sprintf("attempts=%d waitedMs=%d firstErrorCode=%s", attempts, time.Since(started).Milliseconds(), firstNonEmpty(firstCode, "none")))
}

// observeDDLAfterSnapshotは、read-only transactionを持つ長寿命接続がある間にfixture tableへDDLが入った場合の挙動を観測する。
// 他のread-only確認がfixtureのDDL時刻に影響されないよう、最後に実行する。
func (s *session) observeDDLAfterSnapshot() {
	reader := newDB(s.fx.dsn(s.fx.ReadOnly), true)
	reader.SetMaxOpenConns(1)
	defer reader.Close()

	_, before := queryStrings(s.ctx, reader, s.snapshotProbeSQL())

	owner := newDB(s.fx.dsn(s.fx.Owner), false)
	_, commentErr := owner.ExecContext(s.ctx, "COMMENT ON TABLE CUSTOMER IS 'tei local fixture'")
	_, afterComment := queryStrings(s.ctx, reader, s.snapshotProbeSQL())
	_, alterErr := owner.ExecContext(s.ctx, "ALTER TABLE CUSTOMER ADD (DDL_PROBE NUMBER)")
	owner.Close()

	_, sameConn := queryStrings(s.ctx, reader, s.snapshotProbeSQL())

	// 新しい接続なら読めるようになるまでの時間。
	started := time.Now()
	recovered := false
	for time.Since(started) < 60*time.Second && s.ctx.Err() == nil {
		fresh := newDB(s.fx.dsn(s.fx.ReadOnly), true)
		_, err := queryStrings(s.ctx, fresh, s.snapshotProbeSQL())
		fresh.Close()
		if err == nil {
			recovered = true
			break
		}
		time.Sleep(time.Second)
	}
	s.observed("tx.ddl-after-snapshot", layerTransaction,
		"read-only transactionを持つ長寿命接続の途中でfixture tableにDDL(COMMENT ON TABLE、続けてALTER TABLE ADD COLUMN)が入った場合の読取り",
		fmt.Sprintf("firstQuery=%s commentDdl=%s sameConnectionAfterComment=%s alterDdl=%s sameConnectionAfterAlter=%s freshConnectionRecovered=%t afterMs=%d",
			firstNonEmpty(oraCode(before), okOr(before)), okOr(commentErr), firstNonEmpty(oraCode(afterComment), okOr(afterComment)),
			okOr(alterErr), firstNonEmpty(oraCode(sameConn), okOr(sameConn)), recovered, time.Since(started).Milliseconds()))
}
