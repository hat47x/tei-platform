package main

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"tei/internal/environment"
	"tei/internal/realization"
	"tei/internal/validation"
	"tei/internal/workspace"
)

const testWorkspace = "../../reference/workspace"

func testFacts() observedFacts {
	notNull := false
	return observedFacts{
		RunID:           "AABBCCDD",
		ObservedAt:      time.Date(2026, 10, 12, 0, 0, 0, 0, time.UTC),
		ObjectKind:      "TABLE",
		ObjectIdentity:  "fixture-owner.CUSTOMER",
		InputParameter:  "id",
		InputPhysical:   "NUMBER(10,0)",
		NativeSourceRef: "local-ephemeral-oracle-fixture:AABBCCDD/authored-source",
		SourceRef:       "local-ephemeral-oracle-fixture:AABBCCDD/dictionary",
		ProcessingNote:  "一時的な検証用データの観測。external-observedな実資産ではない",
		ReadOnlyMech:    "READ object privilege / SET TRANSACTION READ ONLY / QueryContextのみ",
		ReadOnlyRef:     "local-ephemeral-oracle-fixture:AABBCCDD/receipt",
		RawColumns: []observedColumn{
			{Name: "CUSTOMER_ID", PhysicalType: "NUMBER(10,0)", Nullable: &notNull},
			{Name: "CUSTOMER_NAME", PhysicalType: "VARCHAR2(100)", Nullable: &notNull},
		},
	}
}

func loadTestWorkspace(t *testing.T) workspace.Result {
	t.Helper()
	loaded, err := workspace.Load(testWorkspace)
	if err != nil {
		t.Fatalf("reference Workspaceを読み込めません: %v", err)
	}
	for _, diagnostic := range validation.Validate(loaded.World) {
		if diagnostic.Severity == validation.Error {
			t.Fatalf("reference Workspaceにvalidation errorがあります: %s", diagnostic.Code)
		}
	}
	return loaded
}

func TestAuthoredRealizationMatchesReferenceContractSignature(t *testing.T) {
	loaded := loadTestWorkspace(t)
	contract, err := findQueryContract(loaded.World, fixtureTarget)
	if err != nil {
		t.Fatalf("reference Workspaceから対象Contract Queryを解決できません: %v", err)
	}
	def := authoredRealization("FIXTURE_OWNER.CUSTOMER")
	if err := def.ValidateContract(contract); err != nil {
		t.Fatalf("authoring済みrealizationがContract signatureと一致しません: %v", err)
	}
	if !strings.Contains(def.Source, ":id") || strings.Contains(def.Source, "FOR UPDATE") {
		t.Fatalf("native sourceが想定と異なります: %s", def.Source)
	}
}

func TestReferenceWorkspaceStillHasNoAuthoredOracleRealization(t *testing.T) {
	// このtoolの検証用データ realizationはWorkspace canonへ追加しない。reference Workspaceが空のままであることで、
	// 既存のacceptance toolが実Oracle 検証根拠に対して「realization無し」で停止する前提を保つ。
	loaded := loadTestWorkspace(t)
	if _, err := findAuthoredRealization(loaded.World.ContractQuerySQLRealizations, fixtureTarget, environment.ProviderOracle); err == nil {
		t.Fatal("reference Workspaceにmaster_dataのOracle realizationが存在します。前提が変わっています")
	}
}

func TestIntakeAcceptsObservedFixtureEvidenceThroughExistingValidation(t *testing.T) {
	loaded := loadTestWorkspace(t)
	def := authoredRealization("FIXTURE_OWNER.CUSTOMER")
	evidence, err := buildEvidence(def, testFacts())
	if err != nil {
		t.Fatal(err)
	}
	result, raw, err := intakeEvidence(loaded.World, []realization.ContractQuerySQLRealization{def}, evidence)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Accepted || result.Error != "" {
		t.Fatalf("既存validationが検証根拠を拒否しました: %s", result.Error)
	}
	if !strings.HasPrefix(result.EvidenceArtifactFingerprint, "sha256:") || result.Target != "master_data/master_data.customer/find_by_id" || result.Provider != "oracle" {
		t.Fatalf("intake結果が想定と異なります: %+v", result)
	}
	if !strings.Contains(string(raw), "external-observedな実資産ではない") {
		t.Fatal("検証根拠のprocessingが検証用データ由来であることを示していません")
	}
}

func TestIntakeStopsWithoutExplicitRealizationAndNeverSynthesizesOne(t *testing.T) {
	loaded := loadTestWorkspace(t)
	def := authoredRealization("FIXTURE_OWNER.CUSTOMER")
	evidence, err := buildEvidence(def, testFacts())
	if err != nil {
		t.Fatal(err)
	}
	result, _, err := intakeEvidence(loaded.World, nil, evidence)
	if err != nil {
		t.Fatal(err)
	}
	if result.Accepted || !strings.Contains(result.Error, "検証根拠からrealizationを生成・補完しません") {
		t.Fatalf("realization無しで停止していません: %+v", result)
	}
}

func TestIntakeRejectsStaleNativeSourceFingerprintAndNonExternalWorld(t *testing.T) {
	loaded := loadTestWorkspace(t)
	def := authoredRealization("FIXTURE_OWNER.CUSTOMER")
	evidence, err := buildEvidence(def, testFacts())
	if err != nil {
		t.Fatal(err)
	}

	edited := def
	edited.Source += " "
	result, _, err := intakeEvidence(loaded.World, []realization.ContractQuerySQLRealization{edited}, evidence)
	if err != nil {
		t.Fatal(err)
	}
	// Source末尾の空白だけで別fingerprintになる（fingerprintはexact textに対するSHA-256）。
	if result.Accepted || !strings.Contains(result.Error, "fingerprintがstale") {
		t.Fatalf("stale fingerprintを受理しています: %+v", result)
	}

	mismatch := evidence
	mismatch.Inputs[0].Parameter = "customer_id"
	result, _, err = intakeEvidence(loaded.World, []realization.ContractQuerySQLRealization{def}, mismatch)
	if err != nil {
		t.Fatal(err)
	}
	if result.Accepted {
		t.Fatal("realizationと異なるphysical parameter名の検証根拠を受理しています")
	}
}

func TestBuildEvidenceRejectsFactsThatDoNotMatchExecution(t *testing.T) {
	def := authoredRealization("FIXTURE_OWNER.CUSTOMER")

	facts := testFacts()
	facts.InputParameter = "other"
	if _, err := buildEvidence(def, facts); err == nil {
		t.Fatal("実行で使われたparameter名がrealizationと違うのに検証根拠を作っています")
	}

	facts = testFacts()
	facts.RawColumns = facts.RawColumns[:1]
	if _, err := buildEvidence(def, facts); err == nil {
		t.Fatal("authoring済みraw列が観測されていないのに検証根拠を作っています")
	}

	facts = testFacts()
	facts.RawColumns = append(facts.RawColumns, observedColumn{Name: "EXTRA"})
	if _, err := buildEvidence(def, facts); err == nil {
		t.Fatal("realizationにないraw列が観測されたのに検証根拠を作っています")
	}
}

func TestFindAuthoredRealizationRequiresExactTargetAndProvider(t *testing.T) {
	def := authoredRealization("T")
	otherProvider := def
	otherProvider.Provider = environment.ProviderPostgres
	otherQuery := def
	otherQuery.Target.Query = "other_query"
	if _, err := findAuthoredRealization([]realization.ContractQuerySQLRealization{otherProvider, otherQuery}, fixtureTarget, environment.ProviderOracle); err == nil {
		t.Fatal("target/providerが違うrealizationを選んでいます")
	}
	if _, err := findAuthoredRealization([]realization.ContractQuerySQLRealization{def, def}, fixtureTarget, environment.ProviderOracle); err == nil {
		t.Fatal("複数のrealizationを曖昧に選んでいます")
	}
	if got, err := findAuthoredRealization([]realization.ContractQuerySQLRealization{otherProvider, def}, fixtureTarget, environment.ProviderOracle); err != nil || got.Provider != environment.ProviderOracle {
		t.Fatalf("完全一致を選べません: %v", err)
	}
}

func TestReceiptIsLabelledAsLocalFixtureAndCarriesNoSecretFields(t *testing.T) {
	r := newReceipt()
	r.Checks = []check{{ID: "x", Outcome: "pass"}}
	raw, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	if !strings.Contains(text, `"runId"`) {
		t.Fatal("実行記録に実行IDの欄がありません")
	}
	if r.EvidenceNature != "local-ephemeral-oracle-fixture" || r.ExternalObserved {
		t.Fatal("実行記録が一時的な検証用データであること・外部で観測した資産でないことを示していません")
	}
	for _, forbidden := range []string{`"password"`, `"dsn"`, `"credential"`, `"secret"`, `"connectionString"`} {
		if strings.Contains(strings.ToLower(text), forbidden) {
			t.Fatalf("実行記録にsecret向けfield %sがあります", forbidden)
		}
	}
	joined := strings.Join(r.NotProven, "\n")
	for _, want := range []string{"外部で観測した実際の", "TCPS", "本番のユーザー権限", "候補の版R", "暫定依存"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("notProvenに%qの明示がありません", want)
		}
	}
	if !strings.Contains(r.Database.Usage, "shared-existing-oracle") || !strings.Contains(r.Database.Note, "使い捨てではなく") || !r.Database.ConfirmedSharedDatabaseUse {
		t.Fatal("接続先が別用途の既存Oracleであることを実行記録が示していません")
	}
	if !strings.Contains(r.Driver.Decision, "#371のドライバー決定ではない") {
		t.Fatal("ドライバーが暫定依存であることを示していません")
	}
	if r.Connection.TLS || r.Connection.Wallet {
		t.Fatal("TLS / Walletを確認済みと示しています")
	}
}

func TestPrivilegeAliasingHidesRunSpecificUserNames(t *testing.T) {
	fx, err := newFixture(adminTarget{Host: "localhost", Port: 1521, Service: "S"}, "AABBCCDD")
	if err != nil {
		t.Fatal(err)
	}
	text := fx.aliasText("TEI_REF_AABBCCDD_RO:READ and TEI_REF_AABBCCDD_OWN.CUSTOMER and TEI_REF_AABBCCDD_SEL")
	if strings.Contains(text, "TEI_REF_") {
		t.Fatalf("run固有のuser名が残っています: %s", text)
	}
	if !strings.Contains(text, "read-only-principal:READ") || !strings.Contains(text, "fixture-owner.CUSTOMER") {
		t.Fatalf("別名への置換が想定と異なります: %s", text)
	}
}

func TestStatementMatrixExpectations(t *testing.T) {
	// READだけのユーザーは、接続の種類にかかわらずSELECT以外を成功させてはならない。
	for _, conn := range []string{connPlain, connReadOnlyTx} {
		for _, statement := range matrixStatements {
			got := expectation(roleReadOnly, conn, statement)
			if statement == stmtSelect && got != expectOK || statement != stmtSelect && got != expectError {
				t.Fatalf("READ ユーザー %s/%sの期待が想定と異なります: %s", conn, statement, got)
			}
		}
	}
	// SELECT privilegeは対照: FOR UPDATEとLOCK TABLEを許す（plain接続）。
	if expectation(roleSelectControl, connPlain, stmtForUpdate) != expectOK || expectation(roleSelectControl, connPlain, stmtLock) != expectOK {
		t.Fatal("SELECT対照ユーザーでFOR UPDATE / LOCK TABLEが通る期待になっていません")
	}
	if expectation(roleOwner, connReadOnlyTx, stmtInsert) != "ORA-01456" {
		t.Fatal("read-only transaction内のownerのINSERTがORA-01456になる期待ではありません")
	}

	met := func(expect, outcome string) *bool { return outcomeMeets(expect, outcome) }
	if m := met(expectOK, "ok"); m == nil || !*m {
		t.Fatal("ok期待がokで満たされていません")
	}
	if m := met(expectError, "ok"); m == nil || *m {
		t.Fatal("error期待が成功で満たされてしまいます")
	}
	if m := met(expectError, "ORA-01031"); m == nil || !*m {
		t.Fatal("error期待がORA-01031で満たされていません")
	}
	if m := met("ORA-01456", "ORA-01031"); m == nil || *m {
		t.Fatal("ORA-code指定期待が別codeで満たされてしまいます")
	}
	if met(expectAny, "ok") != nil {
		t.Fatal("観測のみの項目に合否を付けています")
	}
}

func TestDictionaryPhysicalTypeFormatting(t *testing.T) {
	cases := []struct {
		column dictionaryColumn
		want   string
	}{
		{dictionaryColumn{Type: "NUMBER", Precision: "10", Scale: "0"}, "NUMBER(10,0)"},
		{dictionaryColumn{Type: "NUMBER", Precision: "12", Scale: "2"}, "NUMBER(12,2)"},
		{dictionaryColumn{Type: "VARCHAR2", CharLen: "100"}, "VARCHAR2(100)"},
		{dictionaryColumn{Type: "DATE", CharLen: "0"}, "DATE"},
	}
	for _, c := range cases {
		if got := c.column.physicalType(); got != c.want {
			t.Fatalf("got %q want %q", got, c.want)
		}
	}
}
