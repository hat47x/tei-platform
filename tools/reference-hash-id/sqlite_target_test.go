package hashid

import (
	"database/sql"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	_ "modernc.org/sqlite"
)

// SQLite is only one concrete physical target. This test verifies that a
// declared ASCII case-folding profile can be backed by an actual UNIQUE index;
// it does not claim equivalence for PostgreSQL, filesystem paths, or Unicode.
func TestSQLiteNOCASEUniqueIndexPreventsPhysicalNameReuse(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ids.sqlite")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	_, err = db.Exec(`CREATE TABLE bindings (
		namespace TEXT NOT NULL,
		subject TEXT NOT NULL,
		physical_id TEXT NOT NULL COLLATE NOCASE,
		UNIQUE(namespace, subject),
		UNIQUE(namespace, physical_id)
	)`)
	if err != nil {
		t.Fatal(err)
	}
	s := fixtureScheme()
	id, err := Derive(s, []byte("source-one"))
	if err != nil {
		t.Fatal(err)
	}
	p := TargetProfile{
		Namespace:       "example.sqlite.target",
		ProfileVersion:  "ascii-nocase-v1",
		AllowedAlphabet: "0123456789abcdefABCDEF",
		ExactLength:     64,
		CaseFoldASCII:   true,
	}
	if _, err := CheckTargetBinding(id, "qualified:one", p, nil); err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec("INSERT INTO bindings(namespace,subject,physical_id) VALUES(?,?,?)", p.Namespace, "qualified:one", id.Text)
	if err != nil {
		t.Fatal(err)
	}
	// This distinct subject has the same physical key under the *actual*
	// SQLite NOCASE collation, even though the text bytes differ.
	_, err = db.Exec("INSERT INTO bindings(namespace,subject,physical_id) VALUES(?,?,?)",
		p.Namespace, "qualified:two", strings.ToUpper(id.Text))
	if err == nil {
		t.Fatal("SQLite NOCASE unique index accepted a case-folded collision")
	}
	var n int
	if err := db.QueryRow("SELECT COUNT(*) FROM bindings").Scan(&n); err != nil || n != 1 {
		t.Fatalf("expected one binding after collision: count=%d error=%v", n, err)
	}
}

// Concurrent caller requests on one SQLite connection are serialized at that
// database boundary; the UNIQUE constraint still prevents duplicate physical
// assignments. This does NOT test multi-connection / multi-node transactions.
func TestSQLiteUniqueIndexSerializesConcurrentCallers(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "races.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	_, err = db.Exec(`CREATE TABLE bindings (
		namespace TEXT NOT NULL,
		subject TEXT NOT NULL,
		physical_id TEXT NOT NULL,
		UNIQUE(namespace, physical_id)
	)`)
	if err != nil {
		t.Fatal(err)
	}
	const callers = 8
	start := make(chan struct{})
	ok := make(chan bool, callers)
	var wg sync.WaitGroup
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func(subject int) {
			defer wg.Done()
			<-start
			_, err := db.Exec("INSERT INTO bindings(namespace,subject,physical_id) VALUES(?,?,?)",
				"example.concurrent", subject, "same-physical-key")
			ok <- (err == nil)
		}(i)
	}
	close(start)
	wg.Wait()
	close(ok)
	successes := 0
	for success := range ok {
		if success {
			successes++
		}
	}
	if successes != 1 {
		t.Fatalf("unique index admitted %d allocations, want exactly one", successes)
	}
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM bindings").Scan(&count); err != nil || count != 1 {
		t.Fatalf("expected one stored row, count=%d error=%v", count, err)
	}
}
