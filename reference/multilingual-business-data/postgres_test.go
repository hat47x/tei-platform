package multilingualbusinessdata

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

// The PostgreSQL database is an ephemeral CI service, not a user database.
// Ordinary go test runs skip this test unless the DSN is explicitly supplied.
func TestPostgresRelationalRoundTrip(t *testing.T) {
	dsn := os.Getenv("TEI_MULTILINGUAL_PG_DSN")
	if dsn == "" {
		t.Skip("requires ephemeral PostgreSQL service")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(context.Background())
	schema, err := os.ReadFile("testdata/schema.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, sql := range strings.Split(string(schema), ";") {
		if strings.TrimSpace(sql) == "" {
			continue
		}
		if _, err := conn.Exec(ctx, sql); err != nil {
			t.Fatalf("schema: %v in %s", err, sql)
		}
	}

	key := []any{"store-a", "product:P-100", "catalog", "attribute", "catalog.product.description"}
	insert := func(locale, revision, body, origin, review string, sourceLocale, sourceRevision any) {
		t.Helper()
		_, err := conn.Exec(ctx, `INSERT INTO tei_multilingual_revision
		(tenant_id,record_id,field_scope,field_kind,field_id,locale,edition,revision,body,origin_kind,review_state,source_locale,source_revision)
		VALUES ($1,$2,$3,$4,$5,$6,'catalog',$7,$8,$9,$10,$11,$12)`,
			key[0], key[1], key[2], key[3], key[4], locale, revision, body, origin, review, sourceLocale, sourceRevision)
		if err != nil {
			t.Fatal(err)
		}
	}
	publish := func(locale, revision string) {
		t.Helper()
		_, err := conn.Exec(ctx, `INSERT INTO tei_multilingual_publication
		(tenant_id,record_id,field_scope,field_kind,field_id,locale,edition,revision)
		VALUES ($1,$2,$3,$4,$5,$6,'catalog',$7)`,
			key[0], key[1], key[2], key[3], key[4], locale, revision)
		if err != nil {
			t.Fatal(err)
		}
	}
	visible := func(tenant, locale string) (string, string) {
		t.Helper()
		var rev, body string
		err := conn.QueryRow(ctx, `SELECT revision,body FROM tei_multilingual_current
			WHERE tenant_id=$1 AND record_id=$2 AND field_scope=$3
			AND field_kind=$4 AND field_id=$5 AND locale=$6 AND edition='catalog'`,
			tenant, key[1], key[2], key[3], key[4], locale).Scan(&rev, &body)
		if err == pgx.ErrNoRows {
			return "", ""
		}
		if err != nil {
			t.Fatal(err)
		}
		return rev, body
	}

	insert("ja-JP", "ja-1", "安全に保管してください。", "original", "approved", nil, nil)
	insert("en-US", "en-1", "Store safely.", "translation", "approved", "ja-JP", "ja-1")
	publish("ja-JP", "ja-1")
	publish("en-US", "en-1")
	if r, b := visible("store-a", "en-US"); r != "en-1" || b != "Store safely." {
		t.Fatalf("English value %q %q", r, b)
	}
	if r, _ := visible("store-b", "en-US"); r != "" {
		t.Fatal("cross-tenant data exposed")
	}
	if r, _ := visible("store-a", "en-GB"); r != "" {
		t.Fatal("implicit language fallback")
	}
	var n int
	if err := conn.QueryRow(ctx, `SELECT count(*) FROM tei_multilingual_current
		WHERE tenant_id=$1 AND locale='en-US' AND position('Store' in body)>0`, key[0]).Scan(&n); err != nil || n != 1 {
		t.Fatalf("search: count=%d err=%v", n, err)
	}

	insert("ja-JP", "ja-2", "冷暗所に保管してください。", "original", "approved", nil, nil)
	_, err = conn.Exec(ctx, `UPDATE tei_multilingual_publication SET revision='ja-2'
		WHERE tenant_id=$1 AND locale='ja-JP'`, key[0])
	if err != nil {
		t.Fatal(err)
	}
	if r, _ := visible("store-a", "en-US"); r != "" {
		t.Fatal("stale translation still public")
	}
	if err := conn.QueryRow(ctx, `SELECT count(*) FROM tei_multilingual_revision
		WHERE tenant_id=$1 AND locale='en-US' AND revision='en-1'`, key[0]).Scan(&n); err != nil || n != 1 {
		t.Fatalf("history missing: count=%d err=%v", n, err)
	}
	insert("en-US", "en-2", "Keep in a cool, dark place.", "translation", "approved", "ja-JP", "ja-2")

	// Lock the source publication in the same transaction as target CAS.
	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(context.Background())
	var sourceRevision string
	err = tx.QueryRow(ctx, `SELECT revision FROM tei_multilingual_publication
		WHERE tenant_id=$1 AND record_id=$2 AND field_scope=$3 AND field_kind=$4
		AND field_id=$5 AND locale='ja-JP' AND edition='catalog' FOR UPDATE`, key...).Scan(&sourceRevision)
	if err != nil || sourceRevision != "ja-2" {
		t.Fatalf("source changed %q: %v", sourceRevision, err)
	}
	tag, err := tx.Exec(ctx, `UPDATE tei_multilingual_publication SET revision='en-2'
		WHERE tenant_id=$1 AND record_id=$2 AND field_scope=$3 AND field_kind=$4
		AND field_id=$5 AND locale='en-US' AND edition='catalog' AND revision='en-1'`, key...)
	if err != nil || tag.RowsAffected() != 1 {
		t.Fatalf("target CAS: %v %v", tag, err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if r, b := visible("store-a", "en-US"); r != "en-2" || b != "Keep in a cool, dark place." {
		t.Fatalf("new translation %q %q", r, b)
	}
	tag, err = conn.Exec(ctx, `UPDATE tei_multilingual_publication SET revision='en-1'
		WHERE tenant_id=$1 AND record_id=$2 AND field_scope=$3 AND field_kind=$4
		AND field_id=$5 AND locale='en-US' AND edition='catalog' AND revision='en-1'`, key...)
	if err != nil || tag.RowsAffected() != 0 {
		t.Fatalf("stale CAS accepted: %v %v", tag, err)
	}

	insert("en-US", "en-candidate", "Unsigned AI text", "translation", "candidate", "ja-JP", "ja-2")
	_, err = conn.Exec(ctx, `UPDATE tei_multilingual_publication SET revision='en-candidate'
		WHERE tenant_id=$1 AND locale='en-US'`, key[0])
	if err != nil {
		t.Fatal(err)
	}
	if r, _ := visible("store-a", "en-US"); r != "" {
		t.Fatal("unapproved candidate visible")
	}
	insert("en-US", "en-independent", "Binding English agreement", "independent", "approved", nil, nil)
	_, err = conn.Exec(ctx, `UPDATE tei_multilingual_publication SET revision='en-independent'
		WHERE tenant_id=$1 AND locale='en-US'`, key[0])
	if err != nil {
		t.Fatal(err)
	}
	insert("ja-JP", "ja-3", "独立した日本語版", "independent", "approved", nil, nil)
	_, err = conn.Exec(ctx, `UPDATE tei_multilingual_publication SET revision='ja-3'
		WHERE tenant_id=$1 AND locale='ja-JP'`, key[0])
	if err != nil {
		t.Fatal(err)
	}
	if r, _ := visible("store-a", "en-US"); r != "en-independent" {
		t.Fatal("independent text incorrectly expired")
	}
	_, err = conn.Exec(ctx, `INSERT INTO tei_multilingual_revision VALUES
		($1,$2,$3,$4,$5,'fr-FR','catalog','fr-1','Bonjour','translation','approved','ja-JP','absent')`, key...)
	if err == nil {
		t.Fatal("missing translated source foreign key accepted")
	}
}
