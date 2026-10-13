package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"tei/internal/database"
	oraclequery "tei/internal/driver/queryruntime/oracle"
	sqlquery "tei/internal/driver/queryruntime/sql"
	"tei/internal/environment"
	"tei/internal/ir"
	"tei/internal/realization"
)

// checkは、一時的な検証用データに対して実行した確認一件の結果である。
// Outcomeは"pass" / "fail" / "observed"のいずれかで、observedは合否基準を置かず事実だけを記録する。
type check struct {
	ID          string `json:"id"`
	Layer       string `json:"layer"`
	Description string `json:"description"`
	Outcome     string `json:"outcome"`
	Detail      string `json:"detail,omitempty"`
}

// 確認のlayer名。read-onlyを一層だけで主張しないため、層を分けて記録する。
const (
	layerDriver      = "driver"
	layerBind        = "named-bind"
	layerDatatype    = "datatype-charset"
	layerGrant       = "grant"
	layerTransaction = "transaction"
	layerRuntime     = "runtime"
	layerErrors      = "error-mapping"
	layerTimeout     = "timeout-cancel"
	layerIntake      = "evidence-intake"
)

type session struct {
	ctx      context.Context
	fx       *fixture
	red      *redactor
	world    ir.World
	contract database.QueryContract
	def      realization.ContractQuerySQLRealization
	notes    *provisionNotes
	// skipTimeoutCheckは、共有データベースへCPU負荷をかける時間切れの確認を実施しないときに真にする。
	skipTimeoutCheck bool

	checks []check

	Instance   instanceFacts
	Privileges privilegeEvidence
	Matrix     []matrixCell
	Datatypes  []datatypeObservation
	Charset    []charsetObservation
	Dictionary []dictionaryColumn
	Metadata   []oraclequery.RawResultColumnMetadata

	objectType string
}

func (s *session) record(id, layer, description string, ok bool, detail string) {
	outcome := "pass"
	if !ok {
		outcome = "fail"
	}
	s.checks = append(s.checks, check{ID: id, Layer: layer, Description: description, Outcome: outcome, Detail: detail})
}

func (s *session) observed(id, layer, description, detail string) {
	s.checks = append(s.checks, check{ID: id, Layer: layer, Description: description, Outcome: "observed", Detail: detail})
}

func (s *session) failures() []string {
	var ids []string
	for _, c := range s.checks {
		if c.Outcome == "fail" {
			ids = append(ids, c.ID)
		}
	}
	return ids
}

func (s *session) errText(err error) string {
	return s.fx.aliasText(s.red.describeError(err))
}

// rootMessageは、包まれたerrorの最も内側（ドライバーが返した本文）を診断用に返す。
func (s *session) rootMessage(err error) string {
	if err == nil {
		return ""
	}
	for {
		next := errors.Unwrap(err)
		if next == nil {
			break
		}
		err = next
	}
	return s.errText(err)
}

func (s *session) executor(def realization.ContractQuerySQLRealization, db *sql.DB) sqlquery.Executor {
	return sqlquery.Executor{
		Provider:     environment.ProviderOracle,
		Realizations: []realization.ContractQuerySQLRealization{def},
		Runner:       oraclequery.Runner{DB: db},
	}
}

// ---- instance facts ----

type instanceFacts struct {
	Banner               string `json:"banner"`
	Version              string `json:"version"`
	Container            string `json:"container"`
	NLSCharacterSet      string `json:"nlsCharacterSet"`
	NLSNCharCharacterSet string `json:"nlsNcharCharacterSet"`
}

func (s *session) observeInstance(admin *sql.DB) error {
	q := func(dst *string, statement string) error {
		if err := admin.QueryRowContext(s.ctx, statement).Scan(dst); err != nil {
			return fmt.Errorf("Oracle instance factを取得できません: %s", s.errText(err))
		}
		return nil
	}
	if err := q(&s.Instance.Banner, `SELECT banner_full FROM v$version WHERE ROWNUM = 1`); err != nil {
		return err
	}
	if err := q(&s.Instance.Version, `SELECT version_full FROM product_component_version WHERE ROWNUM = 1`); err != nil {
		return err
	}
	if err := q(&s.Instance.Container, `SELECT sys_context('USERENV','CON_NAME') FROM dual`); err != nil {
		return err
	}
	if err := q(&s.Instance.NLSCharacterSet, `SELECT value FROM nls_database_parameters WHERE parameter = 'NLS_CHARACTERSET'`); err != nil {
		return err
	}
	return q(&s.Instance.NLSNCharCharacterSet, `SELECT value FROM nls_database_parameters WHERE parameter = 'NLS_NCHAR_CHARACTERSET'`)
}

// ---- helpers ----

func queryStrings(ctx context.Context, db *sql.DB, statement string, args ...any) ([][]string, error) {
	rows, err := db.QueryContext(ctx, statement, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	var out [][]string
	for rows.Next() {
		values := make([]any, len(columns))
		dests := make([]any, len(columns))
		for i := range values {
			dests[i] = &values[i]
		}
		if err := rows.Scan(dests...); err != nil {
			return nil, err
		}
		row := make([]string, len(columns))
		for i, v := range values {
			if v == nil {
				row[i] = "<NULL>"
			} else {
				row[i] = fmt.Sprint(v)
			}
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func column0(rows [][]string) []string {
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		out = append(out, row[0])
	}
	sort.Strings(out)
	return out
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// toInt64は、go-oraがNUMBERをどのGo型で返しても整数として比較するための補助である。
func toInt64(v any) (int64, bool) {
	switch n := v.(type) {
	case int64:
		return n, true
	case int:
		return int64(n), true
	case int32:
		return int64(n), true
	case float64:
		return int64(n), n == float64(int64(n))
	case string:
		var i int64
		if _, err := fmt.Sscanf(n, "%d", &i); err == nil {
			return i, true
		}
	}
	return 0, false
}

func goType(v any) string {
	if v == nil {
		return "nil"
	}
	return fmt.Sprintf("%T", v)
}

// ---- named bind / runtime path through the Contract Query executor ----

func (s *session) observeContractQueryPath(roDB *sql.DB) {
	exec := s.executor(s.def, roDB)

	if err := roDB.PingContext(s.ctx); err != nil {
		s.record("driver.ping", layerDriver, "読取り専用ユーザーのTCP接続でPingできる", false, s.errText(err))
		return
	}
	s.record("driver.ping", layerDriver, "読取り専用ユーザーのTCP接続でPingできる", true, "")

	result, err := exec.ExecuteContractQuery(s.ctx, fixtureTarget, s.contract, map[string]any{"id": idJapanese})
	if err != nil {
		s.record("bind.found", layerBind, "Contract Query inputがnamed bind :idとして渡り1件取得できる", false, s.errText(err))
	} else {
		ok := len(result.Rows) == 1
		var idMatches, nameMatches bool
		if ok {
			attrs := result.Rows[0].Attributes
			gotID, idOK := toInt64(attrs[attributeCustomerID])
			idMatches = idOK && gotID == idJapanese
			gotName, nameOK := attrs[attributeCustomerName].(string)
			nameMatches = nameOK && gotName == japaneseName
		}
		payload, _ := result.Payload.(oraclequery.Result)
		namesOK := equalStrings(payload.ParameterNames, []string{"id"}) &&
			equalStrings(payload.RawResultColumns, []string{"CUSTOMER_ID", "CUSTOMER_NAME"})
		s.Metadata = payload.RawResultColumnMetadata
		s.record("bind.found", layerBind,
			"Contract Query inputがnamed bind :idとして渡り、1件がSemantic attributeへmappingされ、日本語名が往復する",
			ok && idMatches && nameMatches && namesOK,
			fmt.Sprintf("rows=%d idMatches=%t japaneseNameMatches=%t parameterNames/rawColumnsAsExpected=%t", len(result.Rows), idMatches, nameMatches, namesOK))
	}

	result, err = exec.ExecuteContractQuery(s.ctx, fixtureTarget, s.contract, map[string]any{"id": idNoSuchRow})
	s.record("result.zero-rows", layerBind, "0件を正常な0件として扱う（errorにしない）",
		err == nil && len(result.Rows) == 0, detailOf(s, err, fmt.Sprintf("rows=%d", len(result.Rows))))

	// 文字列bindをNUMBER列へ渡した場合の観測。合否基準を置かない。
	runner := oraclequery.Runner{DB: roDB}
	if raw, err := runner.ExecuteSQL(s.ctx, s.def.Source, map[string]any{"id": "1"}); err != nil {
		s.observed("bind.string-to-number", layerBind, "文字列bindをNUMBER列へ渡す", "error "+firstNonEmpty(oraCode(err), s.errText(err)))
	} else {
		s.observed("bind.string-to-number", layerBind, "文字列bindをNUMBER列へ渡す", fmt.Sprintf("rows=%d", len(raw.Result.Rows)))
	}

	// 宣言と異なるnamed parameter名。実行時errorとして保持されるかを確認する。
	_, err = runner.ExecuteSQL(s.ctx, s.def.Source, map[string]any{"other": int64(1)})
	s.record("error.bind-name-mismatch", layerErrors, "SQLが要求しないnamed parameter名を渡すとExecution Errorとして保持される",
		err != nil && strings.Contains(err.Error(), "native SQLを実行できません"), "code="+firstNonEmpty(oraCode(err), "none")+" cause="+s.rootMessage(err))
	// bind不足。
	_, err = runner.ExecuteSQL(s.ctx, s.def.Source, nil)
	s.record("error.bind-missing", layerErrors, "必要なnamed parameterを渡さないとExecution Errorとして保持される",
		err != nil && strings.Contains(err.Error(), "native SQLを実行できません"), "code="+firstNonEmpty(oraCode(err), "none")+" cause="+s.rootMessage(err))

	// 存在しないobject。
	missing := s.def
	missing.Source = "SELECT CUSTOMER_ID, CUSTOMER_NAME FROM " + s.fx.ReadOnly + ".NO_SUCH_TABLE WHERE CUSTOMER_ID = :id"
	_, err = s.executor(missing, roDB).ExecuteContractQuery(s.ctx, fixtureTarget, s.contract, map[string]any{"id": idJapanese})
	s.record("error.missing-object", layerErrors, "存在しないobjectへのQueryはExecution Errorとして保持される（ORA-00942）",
		err != nil && oraCode(err) == "ORA-00942" && strings.Contains(err.Error(), "物理SQL実行に失敗しました"), "code="+firstNonEmpty(oraCode(err), "none"))

	// NULLをnilのまま保持するか、datatype probe。
	s.observeDatatypes(roDB)
	s.observeCharset(roDB)
}

func detailOf(s *session, err error, base string) string {
	if err != nil {
		return base + " error=" + s.errText(err)
	}
	return base
}

// ---- datatype ----

type datatypeObservation struct {
	Column            string  `json:"column"`
	DatabaseTypeName  *string `json:"databaseTypeName"`
	Nullable          *bool   `json:"nullable"`
	Length            *int64  `json:"length"`
	Precision         *int64  `json:"precision"`
	Scale             *int64  `json:"scale"`
	GoTypeWhenPresent string  `json:"goTypeWhenPresent"`
	GoTypeWhenNull    string  `json:"goTypeWhenNull"`
}

const datatypeProbeTemplate = "SELECT CUSTOMER_ID, CUSTOMER_NAME, CUSTOMER_KANA, NOTE, AMOUNT, REGISTERED_ON FROM %s WHERE CUSTOMER_ID = :id"

func (s *session) observeDatatypes(roDB *sql.DB) {
	runner := oraclequery.Runner{DB: roDB}
	source := fmt.Sprintf(datatypeProbeTemplate, s.fx.qualifiedTable())
	present, err1 := runner.ExecuteSQL(s.ctx, source, map[string]any{"id": idJapanese})
	null, err2 := runner.ExecuteSQL(s.ctx, source, map[string]any{"id": idNullable})
	if err1 != nil || err2 != nil {
		s.record("datatype.probe", layerDatatype, "全列を取得するdatatype probeを実行できる", false, s.errText(errors.Join(err1, err2)))
		return
	}
	payload, _ := present.Payload.(oraclequery.Result)
	rowsOK := len(present.Result.Rows) == 1 && len(null.Result.Rows) == 1
	if !rowsOK || len(payload.RawResultColumnMetadata) != len(present.Result.Rows[0]) {
		s.record("datatype.probe", layerDatatype, "全列を取得するdatatype probeを実行できる", false, "row数または列metadata件数が想定と異なります")
		return
	}
	nullsAreNil := true
	for i, meta := range payload.RawResultColumnMetadata {
		obs := datatypeObservation{
			Column:            meta.Name,
			DatabaseTypeName:  meta.DatabaseTypeName,
			Nullable:          meta.Nullable,
			Length:            meta.Length,
			Precision:         meta.Precision,
			Scale:             meta.Scale,
			GoTypeWhenPresent: goType(present.Result.Rows[0][i]),
			GoTypeWhenNull:    goType(null.Result.Rows[0][i]),
		}
		// 非NULL列(CUSTOMER_ID / CUSTOMER_NAME)以外は、id=2のrowでSQL NULLがnilのまま保持される。
		if i >= 2 && obs.GoTypeWhenNull != "nil" {
			nullsAreNil = false
		}
		s.Datatypes = append(s.Datatypes, obs)
	}
	s.record("datatype.probe", layerDatatype, "全列を取得し、ドライバーが提供したcolumn metadataとGo型を記録できる", true,
		fmt.Sprintf("columns=%d", len(s.Datatypes)))
	s.record("datatype.null-as-nil", layerDatatype, "SQL NULLが別値へ変換されずnilとして保持される", nullsAreNil, "")
}

// ---- charset ----

type charsetObservation struct {
	Label          string `json:"label"`
	InsertOutcome  string `json:"insertOutcome"`
	RoundTripEqual *bool  `json:"roundTripEqual"`
	StoredBytes    string `json:"storedBytes,omitempty"`
}

func (s *session) observeCharset(roDB *sql.DB) {
	runner := oraclequery.Runner{DB: roDB}
	source := fmt.Sprintf("SELECT CUSTOMER_NAME, DUMP(CUSTOMER_NAME, 1016) FROM %s WHERE CUSTOMER_ID = :id", s.fx.qualifiedTable())
	allKanji := true
	for i, c := range charsetCases {
		obs := charsetObservation{Label: c.Label, InsertOutcome: s.notes.CharsetInsertOutcomes[c.Label]}
		raw, err := runner.ExecuteSQL(s.ctx, source, map[string]any{"id": charsetCaseID(i)})
		if err == nil && len(raw.Result.Rows) == 1 {
			got, _ := raw.Result.Rows[0][0].(string)
			equal := got == c.Text
			obs.RoundTripEqual = &equal
			if dump, ok := raw.Result.Rows[0][1].(string); ok {
				obs.StoredBytes = dump
			}
		}
		s.Charset = append(s.Charset, obs)
		if c.Label == "kanji" && (obs.RoundTripEqual == nil || !*obs.RoundTripEqual) {
			allKanji = false
		}
	}
	s.record("charset.japanese-roundtrip", layerDatatype,
		"JA16SJISTILDE Databaseで漢字のVARCHAR2文字列が書込み・読取りで往復する", allKanji, "")

	// NVARCHAR2列の往復。
	kana := oraclequery.Runner{DB: roDB}
	raw, err := kana.ExecuteSQL(s.ctx, fmt.Sprintf("SELECT CUSTOMER_KANA FROM %s WHERE CUSTOMER_ID = :id", s.fx.qualifiedTable()), map[string]any{"id": idJapanese})
	got := ""
	if err == nil && len(raw.Result.Rows) == 1 {
		got, _ = raw.Result.Rows[0][0].(string)
	}
	s.record("charset.nvarchar-roundtrip", layerDatatype, "NVARCHAR2列のカタカナ文字列が往復する", err == nil && got == japaneseKana,
		"goType="+goType(firstValue(raw)))
}

func firstValue(raw sqlquery.RawExecution) any {
	if len(raw.Result.Rows) == 0 || len(raw.Result.Rows[0]) == 0 {
		return nil
	}
	return raw.Result.Rows[0][0]
}

// ---- grant evidence ----

type privilegeEvidence struct {
	// ReadOnlyPrincipalSession は、読取り専用ユーザー自身から見えるsession中の権限である。
	SessionPrivileges   []string `json:"sessionPrivileges"`
	SessionRoles        []string `json:"sessionRoles"`
	UserSystemPrivs     []string `json:"userSystemPrivileges"`
	UserRolePrivs       []string `json:"userRolePrivileges"`
	UserObjectPrivs     []string `json:"userObjectPrivileges"`
	AllObjectPrivsOnObj []string `json:"visibleObjectPrivilegesOnFixtureTable"`
	// DBA側から見たgrant。indirect roleやPUBLICを含めて確認する。
	DBAObjectPrivsOnObj    []string `json:"dbaObjectPrivilegesOnFixtureTable"`
	DBASystemPrivs         []string `json:"dbaSystemPrivilegesOfPrincipal"`
	DBARolePrivs           []string `json:"dbaRolesOfPrincipal"`
	PublicSystemPrivileges []string `json:"publicSystemPrivileges"`
	PublicHasWriteOrWide   bool     `json:"publicHasDmlOrAnyPrivilege"`
}

func (s *session) observeGrants(admin *sql.DB) {
	roPlain := newDB(s.fx.dsn(s.fx.ReadOnly), false)
	defer roPlain.Close()

	get := func(db *sql.DB, statement string, args ...any) []string {
		rows, err := queryStrings(s.ctx, db, statement, args...)
		if err != nil {
			s.record("grant.query", layerGrant, "権限dictionaryを参照できる", false, s.errText(err))
			return nil
		}
		out := make([]string, 0, len(rows))
		for _, row := range rows {
			out = append(out, s.fx.aliasText(strings.Join(row, ":")))
		}
		sort.Strings(out)
		return out
	}

	p := &s.Privileges
	p.SessionPrivileges = get(roPlain, `SELECT privilege FROM session_privs`)
	p.SessionRoles = get(roPlain, `SELECT role FROM session_roles`)
	p.UserSystemPrivs = get(roPlain, `SELECT privilege FROM user_sys_privs`)
	p.UserRolePrivs = get(roPlain, `SELECT granted_role FROM user_role_privs`)
	p.UserObjectPrivs = get(roPlain, `SELECT table_name || ':' || privilege || ':grantable=' || grantable FROM user_tab_privs`)
	p.AllObjectPrivsOnObj = get(roPlain, `SELECT grantee || ':' || privilege FROM all_tab_privs WHERE table_schema = :o AND table_name = 'CUSTOMER'`, sql.Named("o", s.fx.Owner))
	p.DBAObjectPrivsOnObj = get(admin, `SELECT grantee || ':' || privilege FROM dba_tab_privs WHERE owner = :o AND table_name = 'CUSTOMER'`, sql.Named("o", s.fx.Owner))
	p.DBASystemPrivs = get(admin, `SELECT privilege FROM dba_sys_privs WHERE grantee = :g`, sql.Named("g", s.fx.ReadOnly))
	p.DBARolePrivs = get(admin, `SELECT granted_role FROM dba_role_privs WHERE grantee = :g`, sql.Named("g", s.fx.ReadOnly))
	p.PublicSystemPrivileges = get(admin, `SELECT privilege FROM dba_sys_privs WHERE grantee = 'PUBLIC'`)
	for _, privilege := range p.PublicSystemPrivileges {
		if strings.Contains(privilege, "ANY") || strings.Contains(privilege, "INSERT") || strings.Contains(privilege, "UPDATE") || strings.Contains(privilege, "DELETE") {
			p.PublicHasWriteOrWide = true
		}
	}

	only := func(values []string, want ...string) bool { return equalStrings(values, want) }
	readOnlyAlias := s.fx.aliasText(s.fx.ReadOnly)
	s.record("grant.session-privs-create-session-only", layerGrant,
		"読取り専用ユーザーのsession権限がCREATE SESSIONだけである", only(p.SessionPrivileges, "CREATE SESSION"), "")
	s.record("grant.no-enabled-roles", layerGrant, "読取り専用ユーザーに有効なroleがない",
		len(p.SessionRoles) == 0 && len(p.DBARolePrivs) == 0, "")
	var readOnlyGrants []string
	for _, grant := range p.DBAObjectPrivsOnObj {
		if strings.HasPrefix(grant, readOnlyAlias+":") {
			readOnlyGrants = append(readOnlyGrants, grant)
		}
	}
	s.record("grant.object-read-only", layerGrant,
		"検証用の表に対する読取り専用ユーザーのobject privilegeがREADだけである（DBA viewで確認。SELECTや他の権限がない）",
		only(readOnlyGrants, readOnlyAlias+":READ"), "")
	// user_tab_privsにはOracleが既定で付ける自分自身へのINHERIT PRIVILEGESも現れるため、検証用の表の行だけを見る。
	var tableGrants []string
	for _, grant := range p.UserObjectPrivs {
		if strings.HasPrefix(grant, "CUSTOMER:") {
			tableGrants = append(tableGrants, grant)
		}
	}
	s.record("grant.read-visible-to-principal", layerGrant, "読取り専用ユーザー自身のuser_tab_privsで検証用の表への権限がREADだけに見える",
		only(tableGrants, "CUSTOMER:READ:grantable=NO"), "INHERIT PRIVILEGES（自分自身のuserに対する既定の付与）はuser_tab_privsに別行として現れる")
	s.record("grant.no-public-dml", layerGrant, "PUBLICに検証用の表へのobject privilegeやDML / ANY系system privilegeがない",
		!p.PublicHasWriteOrWide && !containsGrantee(p.DBAObjectPrivsOnObj, "PUBLIC"), "")

	// 全権限が仮にDBA側から追加されても分かるよう、READ privilege対応をdictionaryでも確認する。
	rows, err := queryStrings(s.ctx, admin, `SELECT privilege FROM dba_tab_privs WHERE owner = :o AND table_name = 'CUSTOMER' AND grantee = :g`,
		sql.Named("o", s.fx.Owner), sql.Named("g", s.fx.ReadOnly))
	s.record("grant.read-privilege-supported", layerGrant, "このOracle versionでREAD object privilegeをgrantでき、dictionaryにREADとして記録される",
		err == nil && len(rows) == 1 && rows[0][0] == "READ", "")
}

func containsGrantee(grants []string, grantee string) bool {
	for _, g := range grants {
		if strings.HasPrefix(g, grantee+":") {
			return true
		}
	}
	return false
}

type dictionaryColumn struct {
	Column    string `json:"column"`
	Type      string `json:"type"`
	Precision string `json:"precision,omitempty"`
	Scale     string `json:"scale,omitempty"`
	CharLen   string `json:"charLength,omitempty"`
	Nullable  string `json:"nullable"`
}

func (s *session) observeDictionary() {
	roPlain := newDB(s.fx.dsn(s.fx.ReadOnly), false)
	defer roPlain.Close()
	rows, err := queryStrings(s.ctx, roPlain,
		`SELECT column_name, data_type, NVL(TO_CHAR(data_precision),''), NVL(TO_CHAR(data_scale),''), NVL(TO_CHAR(char_length),''), nullable
		 FROM all_tab_columns WHERE owner = :o AND table_name = 'CUSTOMER' ORDER BY column_id`, sql.Named("o", s.fx.Owner))
	if err != nil {
		s.record("dictionary.columns", layerGrant, "READ privilegeだけのユーザーからall_tab_columnsで検証用の表の列を観測できる", false, s.errText(err))
		return
	}
	// Oracleは空文字列をNULLとして返すため、NULL表記は「値なし」として空文字列へ正規化する。
	blank := func(v string) string {
		if v == "<NULL>" {
			return ""
		}
		return v
	}
	for _, row := range rows {
		s.Dictionary = append(s.Dictionary, dictionaryColumn{Column: row[0], Type: row[1], Precision: blank(row[2]), Scale: blank(row[3]), CharLen: blank(row[4]), Nullable: row[5]})
	}
	s.record("dictionary.columns", layerGrant, "READ privilegeだけのユーザーからall_tab_columnsで検証用の表の列を観測できる", len(s.Dictionary) == 6, fmt.Sprintf("columns=%d", len(s.Dictionary)))

	objects, err := queryStrings(s.ctx, roPlain, `SELECT object_type FROM all_objects WHERE owner = :o AND object_name = 'CUSTOMER'`, sql.Named("o", s.fx.Owner))
	if err == nil && len(objects) == 1 {
		s.objectType = objects[0][0]
	}
	s.record("dictionary.object-type", layerGrant, "READ privilegeだけのユーザーからall_objectsで検証用の対象の種類を観測できる", s.objectType == "TABLE", "type="+s.objectType)
}

// physicalTypeは、dictionaryから観測した列の型を一つの文字列にする。
func (c dictionaryColumn) physicalType() string {
	switch {
	case c.Precision != "":
		if c.Scale != "" {
			return fmt.Sprintf("%s(%s,%s)", c.Type, c.Precision, c.Scale)
		}
		return fmt.Sprintf("%s(%s)", c.Type, c.Precision)
	case c.CharLen != "" && c.CharLen != "0":
		return fmt.Sprintf("%s(%s)", c.Type, c.CharLen)
	}
	return c.Type
}

func (s *session) dictionaryColumn(name string) (dictionaryColumn, bool) {
	for _, c := range s.Dictionary {
		if c.Column == name {
			return c, true
		}
	}
	return dictionaryColumn{}, false
}

// ---- statement matrix ----

type matrixCell struct {
	Principal   string `json:"principal"`
	Connection  string `json:"connection"`
	Statement   string `json:"statement"`
	Outcome     string `json:"outcome"`
	Expectation string `json:"expectation"`
	Met         *bool  `json:"met,omitempty"`
}

const (
	stmtSelect    = "SELECT"
	stmtForUpdate = "SELECT_FOR_UPDATE"
	stmtLock      = "LOCK_TABLE_EXCLUSIVE"
	stmtInsert    = "INSERT"
	stmtUpdate    = "UPDATE"
	stmtDelete    = "DELETE"

	connPlain      = "plain"
	connReadOnlyTx = "set-transaction-read-only"

	expectOK    = "ok"
	expectError = "error"
	expectAny   = "observe"
)

var matrixStatements = []string{stmtSelect, stmtForUpdate, stmtLock, stmtInsert, stmtUpdate, stmtDelete}

func (s *session) statementSQL(statement string) string {
	table := s.fx.qualifiedTable()
	switch statement {
	case stmtSelect:
		return fmt.Sprintf("SELECT CUSTOMER_ID FROM %s WHERE CUSTOMER_ID = 1", table)
	case stmtForUpdate:
		return fmt.Sprintf("SELECT CUSTOMER_ID FROM %s WHERE CUSTOMER_ID = 1 FOR UPDATE NOWAIT", table)
	case stmtLock:
		return fmt.Sprintf("LOCK TABLE %s IN EXCLUSIVE MODE NOWAIT", table)
	case stmtInsert:
		return fmt.Sprintf("INSERT INTO %s (CUSTOMER_ID, CUSTOMER_NAME) VALUES (%d, 'matrix')", table, idProbeInsert)
	case stmtUpdate:
		return fmt.Sprintf("UPDATE %s SET CUSTOMER_NAME = 'matrix' WHERE CUSTOMER_ID = %d", table, idProbeInsert)
	case stmtDelete:
		return fmt.Sprintf("DELETE FROM %s WHERE CUSTOMER_ID = %d", table, idProbeInsert)
	}
	return ""
}

// runStatementは、probe文を一回だけ実行して結果を"ok"またはORA-codeで返す。
// このprobeはtoolの確認用コードであり、既存Oracle Runnerのwrite経路ではない。
func (s *session) runStatement(db *sql.DB, statement string) string {
	text := s.statementSQL(statement)
	var err error
	if statement == stmtSelect || statement == stmtForUpdate {
		var rows *sql.Rows
		rows, err = db.QueryContext(s.ctx, text)
		if err == nil {
			for rows.Next() {
			}
			err = rows.Err()
			rows.Close()
		}
	} else {
		_, err = db.ExecContext(s.ctx, text)
	}
	if err == nil {
		return "ok"
	}
	return firstNonEmpty(oraCode(err), "error:"+s.errText(err))
}

func expectation(principal, connection, statement string) string {
	switch principal {
	case roleReadOnly:
		if statement == stmtSelect {
			return expectOK
		}
		return expectError
	case roleSelectControl:
		switch {
		case statement == stmtSelect:
			return expectOK
		case connection == connPlain && (statement == stmtForUpdate || statement == stmtLock):
			// #376が述べるとおり、SELECT privilegeはFOR UPDATEとLOCK TABLEを許す。READとの対照として使う。
			return expectOK
		case statement == stmtInsert || statement == stmtUpdate || statement == stmtDelete:
			return expectError
		}
	case roleOwner:
		if connection == connReadOnlyTx {
			switch statement {
			case stmtSelect:
				return expectOK
			case stmtInsert, stmtUpdate, stmtDelete:
				return "ORA-01456"
			}
		}
	}
	return expectAny
}

func outcomeMeets(expect, outcome string) *bool {
	var met bool
	switch {
	case expect == expectAny:
		return nil
	case expect == expectOK:
		met = outcome == "ok"
	case expect == expectError:
		met = outcome != "ok"
	default:
		met = outcome == expect
	}
	return &met
}

func (s *session) observeMatrix() {
	principals := []struct {
		role string
		user string
	}{
		{roleReadOnly, s.fx.ReadOnly},
		{roleSelectControl, s.fx.SelectControl},
		{roleOwner, s.fx.Owner},
	}
	allMet := true
	for _, principal := range principals {
		for _, connection := range []string{connPlain, connReadOnlyTx} {
			if principal.role == roleOwner && connection == connPlain {
				// 書込み可能なownerをplain接続でprobeしても検証用データを書き換えるだけで、確認になるのは
				// read-only transaction側である。plain側は実行しない。
				continue
			}
			for _, statement := range matrixStatements {
				db := newDB(s.fx.dsn(principal.user), connection == connReadOnlyTx)
				outcome := s.runStatement(db, statement)
				db.Close()
				expect := expectation(principal.role, connection, statement)
				cell := matrixCell{
					Principal:   s.fx.aliasText(principal.user),
					Connection:  connection,
					Statement:   statement,
					Outcome:     outcome,
					Expectation: expect,
					Met:         outcomeMeets(expect, outcome),
				}
				if cell.Met != nil && !*cell.Met {
					allMet = false
				}
				s.Matrix = append(s.Matrix, cell)
			}
		}
	}
	s.record("matrix.expectations", layerGrant,
		"読取り専用ユーザー(READのみ)はSELECT以外を全て拒否され、SELECT付与の対照ユーザーはFOR UPDATE / LOCK TABLEが通り、ownerはread-only transaction内でDMLがORA-01456になる",
		allMet, "セルごとの結果はstatementMatrixを参照")

	// 試験用のwrite文が実際には何も残していないことを、ownerから再確認する。
	owner := newDB(s.fx.dsn(s.fx.Owner), false)
	defer owner.Close()
	rows, err := queryStrings(s.ctx, owner, fmt.Sprintf("SELECT COUNT(*) FROM CUSTOMER WHERE CUSTOMER_ID IN (%d, %d, %d)", idProbeInsert, idProbeAutocmt, idProbeFirstSQL))
	s.record("matrix.no-residual-rows", layerGrant, "probe用のwrite文が検証用の表へrowを残していない",
		err == nil && len(rows) == 1 && rows[0][0] == "0", "")
}

// ---- transaction layer ----

func (s *session) observeTransaction() {
	// 1. read-only connectorが複数のquery後もread-only transactionを保つ。
	owner := newDB(s.fx.dsn(s.fx.Owner), true)
	owner.SetMaxOpenConns(1)
	_, e1 := queryStrings(s.ctx, owner, `SELECT COUNT(*) FROM CUSTOMER`)
	_, e2 := queryStrings(s.ctx, owner, `SELECT COUNT(*) FROM CUSTOMER`)
	_, e3 := owner.ExecContext(s.ctx, fmt.Sprintf("INSERT INTO CUSTOMER (CUSTOMER_ID, CUSTOMER_NAME) VALUES (%d, 'x')", idProbeInsert))
	owner.Close()
	s.record("tx.persists-across-statements", layerTransaction,
		"同一接続で複数のquery後もread-only transactionが維持され、書込み権限を持つownerのINSERTがORA-01456になる",
		e1 == nil && e2 == nil && oraCode(e3) == "ORA-01456", "insert="+firstNonEmpty(oraCode(e3), "succeeded"))

	// 2. 既定のauto commit modeで、db.ExecContextからSET TRANSACTION READ ONLYを発行した場合。
	plain := newDB(s.fx.dsn(s.fx.Owner), false)
	plain.SetMaxOpenConns(1)
	_, errSet := plain.ExecContext(s.ctx, setTransactionReadOnly)
	_, errIns := plain.ExecContext(s.ctx, fmt.Sprintf("INSERT INTO CUSTOMER (CUSTOMER_ID, CUSTOMER_NAME) VALUES (%d, 'x')", idProbeAutocmt))
	if errIns == nil {
		_, _ = plain.ExecContext(s.ctx, fmt.Sprintf("DELETE FROM CUSTOMER WHERE CUSTOMER_ID = %d", idProbeAutocmt))
	}
	plain.Close()
	s.observed("tx.autocommit-mode-set-transaction", layerTransaction,
		"go-oraの既定auto commit modeでdb.ExecContextから発行したSET TRANSACTION READ ONLYが、直後のINSERTを止めるか",
		fmt.Sprintf("setTransaction=%s followingInsert=%s", firstNonEmpty(oraCode(errSet), okOr(errSet)), firstNonEmpty(oraCode(errIns), okOr(errIns))))

	// 同じauto commit modeで、SET TRANSACTION READ ONLYの後にqueryを挟んでもtransactionが残るか。
	plainSelects := newDB(s.fx.dsn(s.fx.Owner), false)
	plainSelects.SetMaxOpenConns(1)
	_, _ = plainSelects.ExecContext(s.ctx, setTransactionReadOnly)
	_, selErr1 := queryStrings(s.ctx, plainSelects, "SELECT COUNT(*) FROM CUSTOMER")
	_, selErr2 := queryStrings(s.ctx, plainSelects, "SELECT COUNT(*) FROM CUSTOMER")
	_, insAfter := plainSelects.ExecContext(s.ctx, fmt.Sprintf("INSERT INTO CUSTOMER (CUSTOMER_ID, CUSTOMER_NAME) VALUES (%d, 'x')", idProbeAutocmt))
	if insAfter == nil {
		_, _ = plainSelects.ExecContext(s.ctx, fmt.Sprintf("DELETE FROM CUSTOMER WHERE CUSTOMER_ID = %d", idProbeAutocmt))
	}
	plainSelects.Close()
	s.observed("tx.autocommit-mode-after-queries", layerTransaction,
		"go-oraの既定auto commit modeで、SET TRANSACTION READ ONLYの後に2回のqueryを挟んでもINSERTを止めるか",
		fmt.Sprintf("queries=%s insertAfterQueries=%s", okOr(errors.Join(selErr1, selErr2)), firstNonEmpty(oraCode(insAfter), okOr(insAfter))))

	// 3. database/sqlのReadOnly TxOptionsをgo-oraが受け付けるか。
	ro := newDB(s.fx.dsn(s.fx.ReadOnly), false)
	_, errTx := ro.BeginTx(s.ctx, &sql.TxOptions{ReadOnly: true})
	ro.Close()
	s.observed("tx.sql-readonly-txoptions", layerTransaction,
		"database/sqlのTxOptions{ReadOnly:true}をgo-ora/v2が受け付けるか", "error="+s.errText(errTx))

	// 4. SET TRANSACTION READ ONLYがtransactionの最初のSQLでなければならない。
	own := newDB(s.fx.dsn(s.fx.Owner), false)
	conn, err := own.Conn(s.ctx)
	var errFirst error
	if err == nil {
		tx, beginErr := conn.BeginTx(s.ctx, nil)
		if beginErr == nil {
			_, _ = tx.ExecContext(s.ctx, fmt.Sprintf("INSERT INTO CUSTOMER (CUSTOMER_ID, CUSTOMER_NAME) VALUES (%d, 'x')", idProbeFirstSQL))
			_, errFirst = tx.ExecContext(s.ctx, setTransactionReadOnly)
			_ = tx.Rollback()
		} else {
			errFirst = beginErr
		}
		conn.Close()
	} else {
		errFirst = err
	}
	own.Close()
	s.record("tx.first-statement-rule", layerTransaction,
		"DML後のSET TRANSACTION READ ONLYがORA-01453で拒否される（transactionの最初のSQLでなければならない）",
		oraCode(errFirst) == "ORA-01453", "code="+firstNonEmpty(oraCode(errFirst), "none"))

	// 5. read-only transactionのsnapshotは接続開始時点に固定される。
	frozen := newDB(s.fx.dsn(s.fx.ReadOnly), true)
	frozen.SetMaxOpenConns(1)
	before, f1 := queryStrings(s.ctx, frozen, fmt.Sprintf("SELECT CUSTOMER_NAME FROM %s WHERE CUSTOMER_ID = %d", s.fx.qualifiedTable(), idNullable))
	writer := newDB(s.fx.dsn(s.fx.Owner), false)
	_, f2 := writer.ExecContext(s.ctx, fmt.Sprintf("UPDATE CUSTOMER SET CUSTOMER_NAME = 'updated' WHERE CUSTOMER_ID = %d", idNullable))
	after, f3 := queryStrings(s.ctx, frozen, fmt.Sprintf("SELECT CUSTOMER_NAME FROM %s WHERE CUSTOMER_ID = %d", s.fx.qualifiedTable(), idNullable))
	fresh := newDB(s.fx.dsn(s.fx.ReadOnly), true)
	freshRows, f4 := queryStrings(s.ctx, fresh, fmt.Sprintf("SELECT CUSTOMER_NAME FROM %s WHERE CUSTOMER_ID = %d", s.fx.qualifiedTable(), idNullable))
	_, f5 := writer.ExecContext(s.ctx, fmt.Sprintf("UPDATE CUSTOMER SET CUSTOMER_NAME = '佐藤花子' WHERE CUSTOMER_ID = %d", idNullable))
	fresh.Close()
	writer.Close()
	frozen.Close()
	ok := errors.Join(f1, f2, f3, f4, f5) == nil && len(before) == 1 && len(after) == 1 && len(freshRows) == 1
	staleKept := ok && before[0][0] == after[0][0] && freshRows[0][0] == "updated"
	s.record("tx.snapshot-fixed-at-connection", layerTransaction,
		"read-only transactionを持つ接続は、他sessionのcommit後もtransaction開始時点のsnapshotを読み続け、新しい接続は更新後を読む",
		staleKept, "この性質は、pool内の長寿命接続でread-only transactionを持続させる方式の制約として記録する")
}

func okOr(err error) string {
	if err == nil {
		return "ok"
	}
	return "error"
}

// ---- timeout / cancel ----

// waitQueryは、取消しまでCPUを使い続ける、実表を読まない問合せである。実行中の問合せをcontextの期限と
// 取消しで中断できるかを確認するために使う。CPU負荷は取消しまでの1〜2秒に限られる。
// 共有しているデータベースでこの負荷を避けるときは、-skip-timeout-checkで実施しない。
// DBMS_PIPEやDBMS_LOCKは、作成したばかりの読取り専用ユーザーに実行権限がないため使わない。
func (s *session) waitQuery() string {
	return "SELECT COUNT(*) FROM (SELECT LEVEL FROM dual CONNECT BY LEVEL <= 20000) a, (SELECT LEVEL FROM dual CONNECT BY LEVEL <= 20000) b"
}

func (s *session) observeTimeoutAndCancel(roDB *sql.DB) {
	if s.skipTimeoutCheck {
		s.observed("timeout.skipped", layerTimeout, "時間切れと取消しの確認", "-skip-timeout-checkの指定により実施しなかった")
		return
	}
	runner := oraclequery.Runner{DB: roDB}

	timeoutCtx, cancel := context.WithTimeout(s.ctx, 2*time.Second)
	started := time.Now()
	_, err := runner.ExecuteSQL(timeoutCtx, s.waitQuery(), nil)
	elapsed := time.Since(started)
	contextErr := timeoutCtx.Err()
	cancel()
	// ドライバーのerrorがcontext errorを包まない場合があるため、呼び出し側がctx.Err()でも原因を判別できることを併せて記録する。
	s.record("timeout.deadline", layerTimeout, "context deadline(2秒)で長時間queryが中断され、Execution Errorとして返る",
		err != nil && contextErr != nil && oraCode(err) == "ORA-01013" && elapsed < 15*time.Second,
		fmt.Sprintf("elapsedMs=%d errorWrapsDeadlineExceeded=%t contextErrAfterReturn=%v code=%s", elapsed.Milliseconds(), errors.Is(err, context.DeadlineExceeded), contextErr, firstNonEmpty(oraCode(err), "none")))

	// 中断後のconnection poolが以降のqueryに使えること。
	recovered, err := s.executor(s.def, roDB).ExecuteContractQuery(s.ctx, fixtureTarget, s.contract, map[string]any{"id": idJapanese})
	s.record("timeout.recovers", layerTimeout, "deadline中断後も同じ*sql.DBで通常のContract Queryを実行できる",
		err == nil && len(recovered.Rows) == 1, detailOf(s, err, ""))

	cancelCtx, cancelNow := context.WithCancel(s.ctx)
	timer := time.AfterFunc(time.Second, cancelNow)
	started = time.Now()
	_, err = runner.ExecuteSQL(cancelCtx, s.waitQuery(), nil)
	elapsed = time.Since(started)
	contextErr = cancelCtx.Err()
	timer.Stop()
	cancelNow()
	s.record("cancel.explicit", layerTimeout, "contextの明示cancel(1秒後)で長時間queryが中断され、Execution Errorとして返る",
		err != nil && contextErr != nil && oraCode(err) == "ORA-01013" && elapsed < 15*time.Second,
		fmt.Sprintf("elapsedMs=%d errorWrapsCanceled=%t contextErrAfterReturn=%v code=%s", elapsed.Milliseconds(), errors.Is(err, context.Canceled), contextErr, firstNonEmpty(oraCode(err), "none")))
	recovered, err = s.executor(s.def, roDB).ExecuteContractQuery(s.ctx, fixtureTarget, s.contract, map[string]any{"id": idJapanese})
	s.record("cancel.recovers", layerTimeout, "cancel後も同じ*sql.DBで通常のContract Queryを実行できる",
		err == nil && len(recovered.Rows) == 1, detailOf(s, err, ""))
}
