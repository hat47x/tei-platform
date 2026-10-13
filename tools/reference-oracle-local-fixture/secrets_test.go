package main

import (
	"errors"
	"strings"
	"testing"
	"unicode"
)

func TestRedactorRemovesEverySecretAndKeepsOneShortLine(t *testing.T) {
	r := &redactor{}
	r.add("hunter2secret")
	r.add("oracle://SYSTEM:hunter2secret@localhost/S")
	r.add("")

	err := errors.New("ORA-01017: invalid credential oracle://SYSTEM:hunter2secret@localhost/S\nsecond line hunter2secret")
	text := r.describeError(err)
	if strings.Contains(text, "hunter2secret") {
		t.Fatalf("secretが残っています: %s", text)
	}
	if strings.Contains(text, "\n") {
		t.Fatalf("複数行のままです: %q", text)
	}
	if !strings.Contains(text, "ORA-01017") {
		t.Fatalf("ORA-codeまで消えています: %s", text)
	}
	if got := (&redactor{}).describeError(nil); got != "" {
		t.Fatalf("nil errorに診断文が付いています: %q", got)
	}
	long := errors.New(strings.Repeat("あ", 500))
	if n := len([]rune(r.describeError(long))); n > 210 {
		t.Fatalf("診断文が長すぎます: %d文字", n)
	}
}

func TestOraCodeExtractsFirstCode(t *testing.T) {
	if got := oraCode(errors.New("x ORA-00942: table or view does not exist ORA-00001")); got != "ORA-00942" {
		t.Fatalf("got %q", got)
	}
	if oraCode(errors.New("no code")) != "" || oraCode(nil) != "" {
		t.Fatal("codeがないerrorで空文字列を返していません")
	}
}

func TestGeneratePasswordIsRandomAndSafeInQuotedIdentifier(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 50; i++ {
		password, err := generatePassword()
		if err != nil {
			t.Fatal(err)
		}
		if len(password) != 24 {
			t.Fatalf("長さ=%d", len(password))
		}
		if !unicode.IsLetter(rune(password[0])) {
			t.Fatalf("先頭が英字ではありません: %q", password[:1])
		}
		for _, c := range password {
			if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9') {
				t.Fatalf("DDLへ埋め込むと構文を壊し得る文字を含みます: %q", c)
			}
		}
		if seen[password] {
			t.Fatal("同じpasswordが重複して生成されました")
		}
		seen[password] = true
	}
}

func TestFixtureUserNamesAreConstrainedToToolOwnedPattern(t *testing.T) {
	id, err := newRunID()
	if err != nil || !runIDPattern.MatchString(id) {
		t.Fatalf("run idが規則に合いません: %q %v", id, err)
	}
	for _, role := range []string{roleOwner, roleReadOnly, roleSelectControl} {
		name, err := fixtureUserName(id, role)
		if err != nil {
			t.Fatal(err)
		}
		if err := requireFixtureUser(name); err != nil {
			t.Fatalf("このtoolが作る名前を拒否しています: %s", name)
		}
	}
	if _, err := fixtureUserName("zzzzzzzz", roleOwner); err == nil {
		t.Fatal("不正なrun idを受理しています")
	}
	if _, err := fixtureUserName(id, "DBA"); err == nil {
		t.Fatal("未知のroleを受理しています")
	}

	// DROP USERの対象は、このtool固有の名前形式に限る。他projectのuserやinjection文字列を通さない。
	for _, victim := range []string{
		"SYSTEM", "SYS", "PDBADMIN", "HR", "TEI_REF_", "TEI_REF_AABBCCDD", "TEI_REF_AABBCCDD_DBA",
		"tei_ref_aabbccdd_own", `TEI_REF_AABBCCDD_OWN" CASCADE; DROP USER "HR`, "TEI_REF_AABBCCDD_OWN ",
	} {
		if err := requireFixtureUser(victim); err == nil {
			t.Fatalf("破壊的DDLの対象に%qを許しています", victim)
		}
	}
}

func TestRequireIdentifierRejectsAnythingButSimpleUppercase(t *testing.T) {
	for _, ok := range []string{"USERS", "TEI_DATA1", "A$B#C"} {
		if err := requireIdentifier(ok); err != nil {
			t.Fatalf("%qを拒否しています", ok)
		}
	}
	for _, bad := range []string{"", "users", `USERS" QUOTA`, "1USERS", "A B", strings.Repeat("A", 31)} {
		if err := requireIdentifier(bad); err == nil {
			t.Fatalf("%qを受理しています", bad)
		}
	}
}
