package main

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	go_ora "github.com/sijms/go-ora/v2"
)

// fixtureは、このtoolが作る専用ユーザー三つと検証用の表、およびそのパスワードを保持する。
// パスワードはプロセス内だけに存在し、実行記録や診断へ出さない。
// createdは、この実行が実際に作成に成功したユーザー名だけを持つ。削除はcreatedに記録した名前だけを対象とし、
// 名前の衝突などで作成しなかったユーザー（他の用途の既存ユーザーを含む）には一切触れない。
type fixture struct {
	RunID      string
	Target     adminTarget
	Tablespace string

	Owner         string
	ReadOnly      string
	SelectControl string

	created []string

	ownerPassword         string
	readOnlyPassword      string
	selectControlPassword string
}

func newFixture(target adminTarget, runID string) (*fixture, error) {
	f := &fixture{RunID: runID, Target: target}
	var err error
	if f.Owner, err = fixtureUserName(runID, roleOwner); err != nil {
		return nil, err
	}
	if f.ReadOnly, err = fixtureUserName(runID, roleReadOnly); err != nil {
		return nil, err
	}
	if f.SelectControl, err = fixtureUserName(runID, roleSelectControl); err != nil {
		return nil, err
	}
	return f, nil
}

func (f *fixture) generatePasswords(red *redactor) error {
	for _, dst := range []*string{&f.ownerPassword, &f.readOnlyPassword, &f.selectControlPassword} {
		password, err := generatePassword()
		if err != nil {
			return err
		}
		*dst = password
		red.add(password)
	}
	return nil
}

func (f *fixture) users() []string {
	return []string{f.Owner, f.ReadOnly, f.SelectControl}
}

// dsnは、指定した専用ユーザーの接続DSNをHost側で組み立てる。
func (f *fixture) dsn(user string) string {
	var password string
	switch user {
	case f.Owner:
		password = f.ownerPassword
	case f.ReadOnly:
		password = f.readOnlyPassword
	case f.SelectControl:
		password = f.selectControlPassword
	}
	return go_ora.BuildUrl(f.Target.Host, f.Target.Port, f.Target.Service, user, password, nil)
}

// qualifiedTableは、検証用の表の完全修飾名である。所有者名は検証済みの固定形式だけを取る。
func (f *fixture) qualifiedTable() string {
	return f.Owner + ".CUSTOMER"
}

// aliasesは、実行記録へ実行固有のユーザー名を残さないための別名である。
func (f *fixture) aliases() map[string]string {
	return map[string]string{
		f.Owner:         "fixture-owner",
		f.ReadOnly:      "read-only-principal",
		f.SelectControl: "select-control-principal",
	}
}

func (f *fixture) aliasText(text string) string {
	for name, alias := range f.aliases() {
		text = strings.ReplaceAll(text, name, alias)
	}
	return text
}

const createTableSQL = `CREATE TABLE CUSTOMER (
  CUSTOMER_ID NUMBER(10) NOT NULL,
  CUSTOMER_NAME VARCHAR2(100 CHAR) NOT NULL,
  CUSTOMER_KANA NVARCHAR2(100),
  NOTE VARCHAR2(200 CHAR),
  AMOUNT NUMBER(12,2),
  REGISTERED_ON DATE,
  CONSTRAINT CUSTOMER_PK PRIMARY KEY (CUSTOMER_ID)
)`

const insertCustomerSQL = `INSERT INTO CUSTOMER (CUSTOMER_ID, CUSTOMER_NAME, CUSTOMER_KANA, NOTE, AMOUNT, REGISTERED_ON)
VALUES (:1, :2, :3, :4, :5, :6)`

// 固定の検証用データのidである。
const (
	idJapanese      int64 = 1
	idNullable      int64 = 2
	idCharsetFirst  int64 = 101
	idNoSuchRow     int64 = 999999
	idProbeInsert   int64 = 900001
	idProbeAutocmt  int64 = 900002
	idProbeFirstSQL int64 = 900003
)

const (
	japaneseName = "山田太郎"
	japaneseKana = "ヤマダタロウ"
	japaneseNote = "備考"
)

// charsetCaseは、JA16SJISTILDEのデータベースとGo / go-oraの間の文字往復を観測するための文字列である。
type charsetCase struct {
	Label string
	Text  string
}

var charsetCases = []charsetCase{
	{"kanji", "山田太郎"},
	{"halfwidth-katakana", "ﾔﾏﾀﾞﾀﾛｳ"},
	{"wave-dash-U+301C", "〜"},
	{"fullwidth-tilde-U+FF5E", "～"},
	{"circled-digit-U+2460", "①"},
	{"ibm-ext-kanji-U+9AD9", "髙"},
	{"fullwidth-hyphen-minus-U+FF0D", "－"},
	{"minus-sign-U+2212", "−"},
	{"emoji-U+1F600", "😀"},
}

func charsetCaseID(index int) int64 { return idCharsetFirst + int64(index) }

func execAdmin(ctx context.Context, db *sql.DB, red *redactor, stage, statement string) error {
	if _, err := db.ExecContext(ctx, statement); err != nil {
		return fmt.Errorf("%sに失敗しました: %s", stage, red.describeError(err))
	}
	return nil
}

// createUsersは、衝突確認の後に専用ユーザー三つを作成する。
// 名前がすでに存在する場合は何も作成せず停止する。CREATE USERが成功した直後に限り、その名前を
// f.createdへ記録する。作成に失敗したユーザー（ORA-01920などの競合を含む）は記録しないため、
// 後始末の対象にならない。
func (f *fixture) createUsers(ctx context.Context, admin *sql.DB, red *redactor) error {
	for _, user := range f.users() {
		if err := requireFixtureUser(user); err != nil {
			return err
		}
		var count int
		if err := admin.QueryRowContext(ctx, `SELECT COUNT(*) FROM dba_users WHERE username = :u`, sql.Named("u", user)).Scan(&count); err != nil {
			return fmt.Errorf("専用ユーザー名の衝突確認に失敗しました: %s", red.describeError(err))
		}
		if count != 0 {
			return fmt.Errorf("専用ユーザー名がすでに存在するため作成しません。既存のユーザーには触れません")
		}
	}

	if err := admin.QueryRowContext(ctx, `SELECT property_value FROM database_properties WHERE property_name = 'DEFAULT_PERMANENT_TABLESPACE'`).Scan(&f.Tablespace); err != nil {
		return fmt.Errorf("既定の表領域を確認できません: %s", red.describeError(err))
	}
	if err := requireIdentifier(f.Tablespace); err != nil {
		return fmt.Errorf("既定の表領域名: %w", err)
	}

	steps := []struct {
		stage, statement string
		creates          string // CREATE USERの場合だけ、作成するユーザー名
	}{
		{"所有者の作成", fmt.Sprintf(`CREATE USER "%s" IDENTIFIED BY "%s" DEFAULT TABLESPACE "%s" QUOTA 5M ON "%s"`, f.Owner, f.ownerPassword, f.Tablespace, f.Tablespace), f.Owner},
		{"読取り専用ユーザーの作成", fmt.Sprintf(`CREATE USER "%s" IDENTIFIED BY "%s"`, f.ReadOnly, f.readOnlyPassword), f.ReadOnly},
		{"対照ユーザーの作成", fmt.Sprintf(`CREATE USER "%s" IDENTIFIED BY "%s"`, f.SelectControl, f.selectControlPassword), f.SelectControl},
		{"所有者への権限付与", fmt.Sprintf(`GRANT CREATE SESSION, CREATE TABLE TO "%s"`, f.Owner), ""},
		{"読取り専用ユーザーへの権限付与", fmt.Sprintf(`GRANT CREATE SESSION TO "%s"`, f.ReadOnly), ""},
		{"対照ユーザーへの権限付与", fmt.Sprintf(`GRANT CREATE SESSION TO "%s"`, f.SelectControl), ""},
	}
	for _, step := range steps {
		if err := execAdmin(ctx, admin, red, step.stage, step.statement); err != nil {
			return err
		}
		if step.creates != "" {
			f.created = append(f.created, step.creates)
		}
	}
	return nil
}

// provisionは、専用ユーザーを作り、所有者で検証用の表を作成して権限を付与する。
// すでに存在するオブジェクトやデータは読み書きしない。
func (f *fixture) provision(ctx context.Context, admin *sql.DB, red *redactor, notes *provisionNotes) error {
	if err := f.createUsers(ctx, admin, red); err != nil {
		return err
	}

	owner := newDB(f.dsn(f.Owner), false)
	defer owner.Close()
	if err := execAdmin(ctx, owner, red, "検証用の表の作成", createTableSQL); err != nil {
		return err
	}
	if err := f.insertFixtureRows(ctx, owner, red, notes); err != nil {
		return err
	}
	// 対象表への付与は表の所有者が行う。読取り専用ユーザーにはREADだけを与える。
	for _, step := range []struct{ stage, statement string }{
		{"READ権限の付与", fmt.Sprintf(`GRANT READ ON CUSTOMER TO "%s"`, f.ReadOnly)},
		{"SELECT権限の付与（対照）", fmt.Sprintf(`GRANT SELECT ON CUSTOMER TO "%s"`, f.SelectControl)},
	} {
		if err := execAdmin(ctx, owner, red, step.stage, step.statement); err != nil {
			return err
		}
	}
	return nil
}

// provisionNotesは、作成中に観測した事実のうち、成否に関わらず記録するものを保持する。
type provisionNotes struct {
	CharsetInsertOutcomes map[string]string
}

func (f *fixture) insertFixtureRows(ctx context.Context, owner *sql.DB, red *redactor, notes *provisionNotes) error {
	registered := time.Date(2026, 10, 12, 0, 0, 0, 0, time.UTC)
	if _, err := owner.ExecContext(ctx, insertCustomerSQL,
		idJapanese, japaneseName, go_ora.NVarChar(japaneseKana), japaneseNote, 1234.5, registered); err != nil {
		return fmt.Errorf("検証用の行の投入に失敗しました: %s", red.describeError(err))
	}
	if _, err := owner.ExecContext(ctx, insertCustomerSQL,
		idNullable, "佐藤花子", nil, nil, nil, nil); err != nil {
		return fmt.Errorf("検証用の行の投入に失敗しました: %s", red.describeError(err))
	}

	notes.CharsetInsertOutcomes = map[string]string{}
	for i, c := range charsetCases {
		var kana any
		if c.Label == "emoji-U+1F600" {
			kana = go_ora.NVarChar(c.Text)
		}
		_, err := owner.ExecContext(ctx, insertCustomerSQL, charsetCaseID(i), c.Text, kana, nil, nil, nil)
		if err != nil {
			// 文字集合上表現できない値の拒否は、失敗ではなく観測事実として残す。
			notes.CharsetInsertOutcomes[c.Label] = "rejected:" + firstNonEmpty(oraCode(err), "driver-error")
			continue
		}
		notes.CharsetInsertOutcomes[c.Label] = "inserted"
	}
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// adoptExistingForCleanupは、実行IDを指定した後始末で、すでに存在する同名の専用ユーザーを削除対象に加える。
// 名前は実行ごとの乱数を含む固定形式で、利用者が明示した実行IDからだけ決まる。
func (f *fixture) adoptExistingForCleanup(ctx context.Context, admin *sql.DB, red *redactor) error {
	for _, user := range f.users() {
		if err := requireFixtureUser(user); err != nil {
			return err
		}
		var count int
		if err := admin.QueryRowContext(ctx, `SELECT COUNT(*) FROM dba_users WHERE username = :u`, sql.Named("u", user)).Scan(&count); err != nil {
			return fmt.Errorf("専用ユーザーの存在確認に失敗しました: %s", red.describeError(err))
		}
		if count != 0 {
			f.created = append(f.created, user)
		}
	}
	return nil
}

// cleanupResultは、後始末の後の再確認結果である。
type cleanupResult struct {
	DroppedUsers          []string `json:"droppedUsers"`
	RemainingFixtureUsers int      `json:"remainingFixtureUsers"`
	RemainingFixtureObjs  int      `json:"remainingFixtureObjects"`
	Verified              bool     `json:"verified"`
	Errors                []string `json:"errors,omitempty"`
}

// teardownは、この実行が作成したと記録した専用ユーザーだけをDROP USER CASCADEし、消えたことを再確認する。
// 呼び出し側は、中断やシグナルで取り消されたcontextではなく、新しいcontextを渡すこと。
func (f *fixture) teardown(ctx context.Context, admin *sql.DB, red *redactor) cleanupResult {
	result := cleanupResult{DroppedUsers: []string{}}
	for _, user := range f.created {
		if err := requireFixtureUser(user); err != nil {
			result.Errors = append(result.Errors, err.Error())
			continue
		}
		statement := fmt.Sprintf(`DROP USER "%s" CASCADE`, user)
		var err error
		for attempt := 0; attempt < 10; attempt++ {
			_, err = admin.ExecContext(ctx, statement)
			// ORA-01940: 接続中のユーザーは削除できない。切断の反映を短く待って再試行する。
			if err == nil || oraCode(err) != "ORA-01940" {
				break
			}
			time.Sleep(time.Second)
		}
		switch {
		case err == nil:
			result.DroppedUsers = append(result.DroppedUsers, f.aliasText(user))
		case oraCode(err) == "ORA-01918":
			// すでに存在しないユーザーは、削除済みの状態である。
		default:
			result.Errors = append(result.Errors, "DROP USER: "+f.aliasText(red.describeError(err)))
		}
	}

	pattern := "TEI\\_REF\\_" + f.RunID + "\\_%"
	if err := admin.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM dba_users WHERE username LIKE :p ESCAPE '\'`, sql.Named("p", pattern),
	).Scan(&result.RemainingFixtureUsers); err != nil {
		result.Errors = append(result.Errors, "残存ユーザー確認: "+red.describeError(err))
	}
	if err := admin.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM dba_objects WHERE owner LIKE :p ESCAPE '\'`, sql.Named("p", pattern),
	).Scan(&result.RemainingFixtureObjs); err != nil {
		result.Errors = append(result.Errors, "残存オブジェクト確認: "+red.describeError(err))
	}
	result.Verified = len(result.Errors) == 0 && result.RemainingFixtureUsers == 0 && result.RemainingFixtureObjs == 0
	return result
}
