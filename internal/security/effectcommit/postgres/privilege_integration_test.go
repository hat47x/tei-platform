package postgres

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"tei/internal/security/effectcommit"
)

const authorityFunctionSignature = ".lock_authority(TEXT,TEXT,TEXT,TEXT,TEXT,TEXT,BIGINT,BIGINT,BIGINT)"

// PostgreSQLの実認証で、保存先の処理権限と認可正本の変更権限を分ける。
// 接続文字列を使い回してSET ROLEで切り替えるだけの試験にはしない。
func TestPostgresWriterCannotMutateAuthority(t *testing.T) {
	f := openFixture(t)
	ctx := context.Background()
	writerRole := f.schema + "_writer"
	ownerRole := f.schema + "_owner"
	password := "tei_scoped_test_writer"

	for _, stmt := range []string{
		"CREATE ROLE " + ownerRole + " NOLOGIN",
		"CREATE ROLE " + writerRole + " LOGIN PASSWORD '" + password + "'",
		"GRANT USAGE, CREATE ON SCHEMA " + f.schema + " TO " + ownerRole,
		"GRANT SELECT, UPDATE ON " + f.schema + ".authority TO " + ownerRole,
		"ALTER FUNCTION " + f.schema + authorityFunctionSignature + " OWNER TO " + ownerRole,
		"GRANT USAGE ON SCHEMA " + f.schema + " TO " + writerRole,
		"GRANT EXECUTE ON FUNCTION " + f.schema + authorityFunctionSignature + " TO " + writerRole,
		"GRANT SELECT, UPDATE ON " + f.schema + ".objects TO " + writerRole,
		"GRANT SELECT, INSERT ON " + f.schema + ".receipts TO " + writerRole,
	} {
		if _, err := f.pool.Exec(ctx, stmt); err != nil {
			t.Fatalf("DB権限の分離を設定できません: %v: %s", err, stmt)
		}
	}
	t.Cleanup(func() {
		if _, err := f.pool.Exec(context.Background(), "REASSIGN OWNED BY "+ownerRole+" TO CURRENT_USER"); err != nil {
			t.Errorf("試験用関数の所有権を戻せません: %v", err)
		}
		if _, err := f.pool.Exec(context.Background(), "DROP OWNED BY "+ownerRole+", "+writerRole); err != nil {
			t.Errorf("試験用権限を解放できません: %v", err)
		}
		for _, role := range []string{ownerRole, writerRole} {
			if _, err := f.pool.Exec(context.Background(), "DROP ROLE "+role); err != nil {
				t.Errorf("試験用DB主体を削除できません: %v", err)
			}
		}
	})

	cfg, err := pgxpool.ParseConfig(os.Getenv("TEI_EFFECT_POSTGRES_DSN"))
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.User = writerRole
	cfg.ConnConfig.Password = password
	writer, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer writer.Close()
	if err := writer.Ping(ctx); err != nil {
		t.Fatalf("制限付きの別DBログインに失敗: %v", err)
	}

	for _, stmt := range []string{
		"UPDATE " + f.schema + ".authority SET enabled=TRUE, membership=membership+1",
		"SELECT * FROM " + f.schema + ".authority",
		"UPDATE " + f.schema + ".subjects SET role='EDITOR'",
		"ALTER FUNCTION " + f.schema + authorityFunctionSignature + " OWNER TO " + writerRole,
		"DROP TRIGGER update_membership ON " + f.schema + ".subjects",
	} {
		_, err := writer.Exec(ctx, stmt)
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.Code != "42501" {
			t.Fatalf("認可正本を操作するSQLを制限付き接続で実行できました: %q / %v", stmt, err)
		}
	}

	grant := f.grant(t) // 読取専用の認可経路は別の信頼済みDB接続
	provider, err := New(writer, f.schema)
	if err != nil {
		t.Fatal(err)
	}
	runner := effectcommit.Runner{Provider: provider}
	result, err := runner.Execute(ctx, f.req, grant)
	if err != nil || result.Status != effectcommit.StatusCommitted {
		t.Fatalf("最小権限の接続から確定できません: %+v %v", result, err)
	}
	if result.Receipt.ResourceVersion != 1 {
		t.Fatal("確定記録が正しく保存されませんでした")
	}
	// 認可更新は信頼された管理経路から行い、保存先には変更権限がない。
	if _, err := f.pool.Exec(ctx, "UPDATE "+f.schema+".subjects SET disabled=TRUE WHERE subject=$1", f.req.Scope.Subject); err != nil {
		t.Fatal(err)
	}
	replayed, err := runner.Lookup(ctx, f.req, grant)
	if err != nil || replayed.Status != effectcommit.StatusDenied || replayed.Receipt != (effectcommit.Receipt{}) {
		t.Fatalf("失効後の過去記録が露出: %+v %v", replayed, err)
	}
}

func TestPostgresAuthorityLockFunctionMissingFailsClosed(t *testing.T) {
	f := openFixture(t)
	grant := f.grant(t)
	if _, err := f.pool.Exec(context.Background(), "DROP FUNCTION "+f.schema+authorityFunctionSignature); err != nil {
		t.Fatal(err)
	}
	runner := effectcommit.Runner{Provider: f.provider}
	result, err := runner.Execute(context.Background(), f.req, grant)
	if err == nil || result != (effectcommit.Result{}) {
		t.Fatalf("認可正本を照合できないのに書込みました: %+v %v", result, err)
	}
	if value, version := f.readObject(t); value != "original" || version != 0 {
		t.Fatalf("関数欠落後に副作用が発生: %q %d", value, version)
	}
}
