package legacyauth

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func writeSubjectFixture(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "users.json")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestFixtureStoreRejectsDuplicateBusinessSubjectAcrossLogins(t *testing.T) {
	path := writeSubjectFixture(t, `{
		"schema":"legacy-auth-user-v1",
		"users":[
			{"user_id":"legacy-editor","subject_id":"same-subject","role_code":"ORDER_EDITOR"},
			{"user_id":"legacy-viewer","subject_id":"same-subject","role_code":"ORDER_VIEWER"}
		]
	}`)
	store, err := LoadFixtureStore(path)
	if err == nil || store != nil || !strings.Contains(err.Error(), "subject_id") {
		t.Fatalf("ambiguous subject mapping was accepted: store=%+v err=%v", store, err)
	}
}

func TestFixtureStoreFindBySubjectIsUniqueAndDeterministic(t *testing.T) {
	path := writeSubjectFixture(t, `{
		"schema":"legacy-auth-user-v1",
		"users":[
			{"user_id":"legacy-editor","subject_id":"subject-editor","role_code":"ORDER_EDITOR"},
			{"user_id":"legacy-viewer","subject_id":"subject-viewer","role_code":"ORDER_VIEWER"},
			{"user_id":"legacy-incomplete","subject_id":"","role_code":"ORDER_EDITOR"}
		]
	}`)
	store, err := LoadFixtureStore(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		subjectID, login, role string
		found                  bool
	}{
		{subjectID: "subject-editor", login: "legacy-editor", role: "ORDER_EDITOR", found: true},
		{subjectID: "subject-viewer", login: "legacy-viewer", role: "ORDER_VIEWER", found: true},
		{subjectID: "missing", found: false},
		{subjectID: "", found: false},
	} {
		for i := 0; i < 20; i++ {
			user, found, err := store.FindBySubject(context.Background(), tt.subjectID)
			if err != nil || found != tt.found {
				t.Fatalf("subject=%q lookup[%d]: found=%v err=%v", tt.subjectID, i, found, err)
			}
			if found && (user.LoginID != tt.login || user.RoleCode != tt.role ||
				user.SubjectID != tt.subjectID) {
				t.Fatalf("subject=%q resolved an unrelated login/role: %+v", tt.subjectID, user)
			}
		}
	}
	user, found, err := store.FindByLogin(context.Background(), "legacy-editor")
	if err != nil || !found || user.SubjectID != "subject-editor" {
		t.Fatalf("login lookup changed: %+v found=%v err=%v", user, found, err)
	}
}

func TestReferenceFixtureUsesUniqueBusinessSubjectIndex(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("unable to locate reference fixture")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", ".."))
	store, err := LoadFixtureStore(filepath.Join(root, "reference", "legacy-auth", "users.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		subjectID string
		loginID   string
		role      string
	}{
		{subjectID: "subject-order-editor", loginID: "legacy-order-editor", role: "ORDER_EDITOR"},
		{subjectID: "subject-order-viewer", loginID: "legacy-order-viewer", role: "ORDER_VIEWER"},
		{subjectID: "subject-order-locked", loginID: "legacy-order-locked", role: "ORDER_EDITOR"},
		{subjectID: "subject-order-disabled", loginID: "legacy-order-disabled", role: "ORDER_VIEWER"},
	} {
		user, found, err := store.FindBySubject(context.Background(), tc.subjectID)
		if err != nil || !found || user.LoginID != tc.loginID || user.RoleCode != tc.role {
			t.Fatalf("reference subject %q: user=%+v found=%v err=%v", tc.subjectID, user, found, err)
		}
	}
}
