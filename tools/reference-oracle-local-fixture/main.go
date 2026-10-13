// reference-oracle-local-fixtureは、別用途の既存Oracleに専用ユーザーと検証用の表だけを一時的に作成し、
// Oracleドライバー、名前付きバインド、型、層別の読取り専用確認、検証根拠の受入れを実行して、
// 秘密を含まないJSONの実行記録を出す。実行の最後に、作成した専用ユーザーだけを削除する。
//
// これは外部で観測した実際のmaster_data資産の検証根拠ではない。
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"
	"syscall"
	"time"

	"tei/internal/database/externalqueryevidence"
	"tei/internal/environment"
	"tei/internal/realization"
	"tei/internal/validation"
	"tei/internal/workspace"
	"tei/tools/internal/repositoryevidence"
)

// evidenceNatureは、この実行記録の種類を示す。外部で観測した資産の検証根拠ではない。
const evidenceNature = "local-ephemeral-oracle-fixture"

type driverFacts struct {
	Module       string `json:"module"`
	Version      string `json:"version"`
	RegisteredAs string `json:"registeredAs"`
	GoVersion    string `json:"goVersion"`
	Isolation    string `json:"isolation"`
	Decision     string `json:"decision"`
}

type connectionFacts struct {
	Transport string `json:"transport"`
	TLS       bool   `json:"tls"`
	Wallet    bool   `json:"wallet"`
	Loopback  bool   `json:"loopback"`
	Note      string `json:"note"`
}

// databaseFactsは、接続先が使い捨てではなく別用途の既存Oracleであることと、このtoolが触れた範囲を示す。
type databaseFacts struct {
	Usage                      string `json:"usage"`
	ConfirmedSharedDatabaseUse bool   `json:"confirmedSharedDatabaseFixture"`
	Note                       string `json:"note"`
}

type receipt struct {
	EvidenceNature       string                     `json:"evidenceNature"`
	ExternalObserved     bool                       `json:"externalObserved"`
	RunID                string                     `json:"runId"`
	Completed            bool                       `json:"completed"`
	AllChecksPassed      bool                       `json:"allChecksPassed"`
	FailedChecks         []string                   `json:"failedChecks"`
	Error                string                     `json:"error,omitempty"`
	RepositoryRevision   string                     `json:"repositoryRevision"`
	WorkingTreeClean     bool                       `json:"workingTreeClean"`
	ReproducibleRevision bool                       `json:"reproducibleRevision"`
	PostRunRepository    repositoryevidence.PostRun `json:"postRunRepository"`
	Driver               driverFacts                `json:"driver"`
	Database             databaseFacts              `json:"database"`
	Oracle               *instanceFacts             `json:"oracle,omitempty"`
	Connection           connectionFacts            `json:"connection"`
	Principals           map[string]string          `json:"principals"`
	Checks               []check                    `json:"checks"`
	StatementMatrix      []matrixCell               `json:"statementMatrix,omitempty"`
	Privileges           *privilegeEvidence         `json:"privileges,omitempty"`
	DictionaryColumns    []dictionaryColumn         `json:"dictionaryColumns,omitempty"`
	Datatypes            []datatypeObservation      `json:"datatypes,omitempty"`
	Charset              []charsetObservation       `json:"charset,omitempty"`
	Intake               *intakeResult              `json:"externalEvidenceIntake,omitempty"`
	Cleanup              *cleanupResult             `json:"cleanup,omitempty"`
	NotProven            []string                   `json:"notProven"`
}

// notProvenは、この実行記録が証明しないことを常に明示する。
var notProven = []string{
	"外部で観測した実際のmaster_data資産（実Oracle）の物理的な事実、型、NULL扱い、native SQLは観測していない",
	"実資産の由来・加工履歴、実資産に対する読取り専用ユーザーの権限構成は取得していない",
	"TCPS、Wallet、認証方式は未確認（この実行はloopbackのTCP平文とパスワード認証だけ）",
	"本番のユーザー権限ではない。このtoolが一時的に作った専用ユーザーに対する権限の観測であり、実外部資産の検証根拠へ読み替えない",
	"候補の版Rでの最終的な実読取りではない。検証用データはこの実行のために作成して削除した",
	"読取り専用トランザクションを接続プールでどう運用するかの方針は決めていない。接続単位で持続させる方式では、スナップショットが接続開始時点に固定される",
	"go-ora/v2は検証用の暫定依存であり、#371のドライバー決定ではない",
}

func main() {
	opts, err := parseOptions(os.Args[1:], os.Stderr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if opts.CleanupRun != "" {
		os.Exit(runCleanup(opts.CleanupRun))
	}

	// SIGINTとSIGTERMで実行を中断する。後始末はこのcontextではなく、新しいcontextで行う。
	signalCtx, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignals()
	ctx, cancel := context.WithTimeout(signalCtx, opts.Timeout)
	defer cancel()
	result, err := run(ctx, opts)
	if err != nil {
		result.Error = err.Error()
	}
	encoded, marshalErr := json.MarshalIndent(result, "", "  ")
	if marshalErr != nil {
		fmt.Fprintln(os.Stderr, "実行記録を出力できません")
		os.Exit(1)
	}
	fmt.Println(string(encoded))
	if err != nil || !result.AllChecksPassed {
		os.Exit(1)
	}
}

func newReceipt() receipt {
	return receipt{
		EvidenceNature:   evidenceNature,
		ExternalObserved: false,
		FailedChecks:     []string{},
		Checks:           []check{},
		NotProven:        notProven,
		Principals: map[string]string{
			"fixture-owner":            "検証用の表の所有者。CREATE SESSIONとCREATE TABLEだけ",
			"read-only-principal":      "検証用の表へのREAD object privilegeとCREATE SESSIONだけ",
			"select-control-principal": "READとの対照用。検証用の表へのSELECT object privilegeとCREATE SESSIONだけ",
		},
		Connection: connectionFacts{
			Transport: "TCP",
			TLS:       false,
			Wallet:    false,
			Loopback:  true,
			Note:      "loopbackのTCP平文とパスワード認証だけを確認した。TCPS、Wallet、他の認証方式は未確認",
		},
		Database: databaseFacts{
			Usage:                      "shared-existing-oracle-temporary-fixture",
			ConfirmedSharedDatabaseUse: true,
			Note:                       "別用途の既存Oracle 19cに、このtoolが専用ユーザーと検証用の表だけを一時的に作成・削除した。Oracleそのものは使い捨てではなく、既存のオブジェクトとデータは読み書きしていない",
		},
		Driver: driverInfo(),
	}
}

func driverInfo() driverFacts {
	facts := driverFacts{
		Module:       "github.com/sijms/go-ora/v2",
		RegisteredAs: "oracle",
		GoVersion:    runtime.Version(),
		Isolation:    "ドライバーのimportはこのtoolのpackageだけ。internal/配下の中核とOracleの提供元・問合せ実行環境はドライバーを知らない",
		Decision:     "検証用の暫定依存であり、#371のドライバー決定ではない",
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, dep := range info.Deps {
			if dep.Path == facts.Module {
				facts.Version = dep.Version
			}
		}
	}
	return facts
}

func run(ctx context.Context, opts options) (receipt, error) {
	workspacePath, evidenceOut := opts.Workspace, opts.EvidenceOut
	result := newReceipt()

	source, err := repositoryevidence.InspectStart()
	if err != nil {
		return result, err
	}
	result.RepositoryRevision = source.RepositoryRevision
	result.WorkingTreeClean = source.WorkingTreeClean
	if !source.WorkingTreeClean {
		return result, fmt.Errorf("実行記録をclean revisionへ固定するため、working treeに未commit差分がある状態では実行しません")
	}

	dsn, err := loadAdminDSN(os.LookupEnv, os.ReadFile)
	if err != nil {
		return result, err
	}
	if err := clearAdminDSNEnvironment(); err != nil {
		return result, err
	}
	target, err := parseAdminDSN(dsn)
	if err != nil {
		return result, err
	}
	red := &redactor{}
	red.add(dsn)
	red.add(target.Password)

	loaded, err := workspace.Load(workspacePath)
	if err != nil {
		return result, fmt.Errorf("作業領域 %sを読み込めません", workspacePath)
	}
	for _, diagnostic := range validation.Validate(loaded.World) {
		if diagnostic.Severity == validation.Error {
			return result, fmt.Errorf("作業領域に検証エラーがあります: %s", diagnostic.Code)
		}
	}
	contract, err := findQueryContract(loaded.World, fixtureTarget)
	if err != nil {
		return result, err
	}

	runID, err := newRunID()
	if err != nil {
		return result, err
	}
	// 実行IDは、ユーザーを作成する前に標準エラーへ出す。強制終了後でも-cleanupで後始末できるようにするためである。
	result.RunID = runID
	fmt.Fprintf(os.Stderr, "runId=%s\n", runID)
	fmt.Fprintf(os.Stderr, "中断された場合の後始末: -cleanup %s -%s（管理DSNは同じ方法で渡す）\n", runID, confirmFlag)

	fx, err := newFixture(target, runID)
	if err != nil {
		return result, err
	}

	admin := newDB(dsn, false)
	defer admin.Close()
	if err := admin.PingContext(ctx); err != nil {
		return result, fmt.Errorf("管理接続のPingに失敗しました: %s", red.describeError(err))
	}

	s := &session{
		ctx:      ctx,
		fx:       fx,
		red:      red,
		world:    loaded.World,
		contract: contract,
		def:      authoredRealization(fx.qualifiedTable()),
		notes:    &provisionNotes{},
	}
	s.skipTimeoutCheck = opts.SkipTimeoutCheck

	// panicなどで通常の後始末へ到達しなかった場合も、作成済みのユーザーを新しいcontextで削除する。
	tornDown := false
	defer func() {
		if tornDown {
			return
		}
		panicCtx, cancelPanic := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancelPanic()
		cleanup := fx.teardown(panicCtx, admin, red)
		fmt.Fprintf(os.Stderr, "異常終了のため専用ユーザーを削除しました: verified=%t\n", cleanup.Verified)
	}()

	runErr := s.execute(admin, evidenceOut, &result)

	// 失敗や中断（取り消されたcontextを含む）の場合も、この実行が作成した専用ユーザーを新しいcontextで必ず削除し、結果を再確認する。
	cleanupCtx, cancelCleanup := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancelCleanup()
	cleanup := fx.teardown(cleanupCtx, admin, red)
	tornDown = true
	result.Cleanup = &cleanup

	result.Checks = s.checks
	result.FailedChecks = s.failures()
	if result.FailedChecks == nil {
		result.FailedChecks = []string{}
	}
	result.Oracle = &s.Instance
	result.StatementMatrix = s.Matrix
	result.Privileges = &s.Privileges
	result.DictionaryColumns = s.Dictionary
	result.Datatypes = s.Datatypes
	result.Charset = s.Charset

	postRun, postErr := repositoryevidence.InspectPost(source.RepositoryRevision)
	if postErr != nil && runErr == nil {
		runErr = postErr
	}
	result.PostRunRepository = postRun
	result.ReproducibleRevision = repositoryevidence.IsReproducible(source, postRun)

	result.Completed = runErr == nil && cleanup.Verified && result.ReproducibleRevision
	result.AllChecksPassed = result.Completed && len(result.FailedChecks) == 0 && result.Intake != nil && result.Intake.Accepted
	if runErr == nil && !cleanup.Verified {
		runErr = fmt.Errorf("作成した専用ユーザーの削除を確認できませんでした")
	}
	if runErr == nil && !result.ReproducibleRevision {
		runErr = fmt.Errorf("実行中にrepositoryのrevisionまたはworking treeが変化したため、単一のclean revisionへ固定できません")
	}
	return result, runErr
}

// executeは、専用ユーザーと検証用の表の作成から各確認、検証根拠の受入れまでを行う。後始末は呼び出し側が行う。
func (s *session) execute(admin *sql.DB, evidenceOut string, result *receipt) error {
	if err := s.observeInstance(admin); err != nil {
		return err
	}
	if err := s.fx.generatePasswords(s.red); err != nil {
		return err
	}
	if err := s.fx.provision(s.ctx, admin, s.red, s.notes); err != nil {
		return err
	}
	s.record("provision.read-privilege-grantable", layerGrant, "検証用の表の所有者が、読取り専用ユーザーへREAD object privilegeを付与できた（Oracle 12.1.0.2以降の機能）", true, "")

	s.observeDictionary()
	s.waitForReadOnlySnapshot()

	roDB := newDB(s.fx.dsn(s.fx.ReadOnly), true)
	s.observeContractQueryPath(roDB)
	s.observeGrants(admin)
	s.observeMatrix()
	s.observeTransaction()
	s.observeTimeoutAndCancel(roDB)
	roDB.Close()
	s.observeDDLAfterSnapshot()

	return s.intake(evidenceOut, result)
}

func (s *session) intake(evidenceOut string, result *receipt) error {
	facts, err := s.collectEvidenceFacts()
	if err != nil {
		s.record("intake.facts", layerIntake, "観測した事実を検証根拠へ写せる", false, err.Error())
		return nil
	}
	evidence, err := buildEvidence(s.def, facts)
	if err != nil {
		s.record("intake.build", layerIntake, "観測した事実と作者が明示したSQL実現から検証根拠を組み立てられる", false, err.Error())
		return nil
	}
	definitions := []realization.ContractQuerySQLRealization{s.def}
	intake, raw, err := intakeEvidence(s.world, definitions, evidence)
	if err != nil {
		return err
	}
	result.Intake = &intake
	s.record("intake.accepted", layerIntake, "既存のexternalqueryevidenceの厳格な読込みとValidateAgainstが検証根拠を受理する", intake.Accepted, intake.Error)

	// SQL実現がない対象と提供元は、検証根拠から補完せず停止する（既存のtoolと既存の規則と同じ境界）。
	_, stopErr := findAuthoredRealization(nil, evidence.Target, evidence.Provider)
	s.record("intake.stops-without-realization", layerIntake, "作者が明示したSQL実現がない場合は検証根拠から補完せず停止する", stopErr != nil, "")

	// 提供元が異なるSQL実現しかない場合も補完しない。
	other := s.def
	other.Provider = environment.ProviderPostgres
	_, stopErr = findAuthoredRealization([]realization.ContractQuerySQLRealization{other}, evidence.Target, evidence.Provider)
	s.record("intake.provider-must-match", layerIntake, "提供元が違うSQL実現を選ばない", stopErr != nil, "")

	// native SQLの照合値が違えば古い検証根拠として拒否する。
	stale := evidence
	stale.NativeSource.Fingerprint = "sha256:" + fmt.Sprintf("%064d", 0)
	if err := externalqueryevidence.ValidateAgainst(s.world, s.def, stale); err != nil {
		s.record("intake.rejects-stale-fingerprint", layerIntake, "native SQLの照合値が違う検証根拠を拒否する", true, "")
	} else {
		s.record("intake.rejects-stale-fingerprint", layerIntake, "native SQLの照合値が違う検証根拠を拒否する", false, "")
	}

	if evidenceOut != "" && intake.Accepted {
		if err := os.WriteFile(evidenceOut, append(raw, '\n'), 0o600); err != nil {
			return fmt.Errorf("検証根拠のJSONを書き出せません")
		}
	}
	return nil
}

func (s *session) collectEvidenceFacts() (observedFacts, error) {
	facts := observedFacts{
		RunID:           s.fx.RunID,
		ObservedAt:      time.Now(),
		ObjectKind:      s.objectType,
		ObjectIdentity:  s.fx.aliasText(s.fx.qualifiedTable()),
		InputParameter:  "id",
		NativeSourceRef: "local-ephemeral-oracle-fixture:" + s.fx.RunID + "/authored-source",
		SourceRef:       "local-ephemeral-oracle-fixture:" + s.fx.RunID + "/data-dictionary-and-driver-metadata",
		ProcessingNote:  "このtoolが一時的に作成した検証用の表について、データ辞書とドライバーのメタデータを観測した。外部で観測した実資産の観測ではなく、実資産の検証根拠へ読み替えない",
		ReadOnlyMech:    "READ object privilegeだけの専用ユーザー、接続時のSET TRANSACTION READ ONLY、QueryContextだけを使う実行部の多層。一時的な検証用データでの観測",
		ReadOnlyRef:     "local-ephemeral-oracle-fixture:" + s.fx.RunID + "/receipt#grant,transaction,statementMatrix",
	}
	if facts.ObjectKind == "" {
		return facts, fmt.Errorf("検証用の対象の種類をデータ辞書から観測できていません")
	}
	idColumn, ok := s.dictionaryColumn("CUSTOMER_ID")
	if !ok {
		return facts, fmt.Errorf("CUSTOMER_ID列をデータ辞書から観測できていません")
	}
	facts.InputPhysical = idColumn.physicalType()
	if len(s.Metadata) == 0 {
		return facts, fmt.Errorf("実読取りで結果列のメタデータを観測できていません")
	}
	for _, meta := range s.Metadata {
		column := observedColumn{Name: meta.Name, Nullable: meta.Nullable}
		if dict, ok := s.dictionaryColumn(meta.Name); ok {
			column.PhysicalType = dict.physicalType()
		}
		facts.RawColumns = append(facts.RawColumns, column)
	}
	return facts, nil
}

// runCleanupは、実行IDを指定して、その実行が作成した専用ユーザーだけを削除する。
// 実行IDから決まる三つの名前のうち、現に存在するものだけを対象とする。
func runCleanup(runID string) int {
	dsn, err := loadAdminDSN(os.LookupEnv, os.ReadFile)
	if err == nil {
		err = clearAdminDSNEnvironment()
	}
	var target adminTarget
	if err == nil {
		target, err = parseAdminDSN(dsn)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	red := &redactor{}
	red.add(dsn)
	red.add(target.Password)
	fx, err := newFixture(target, runID)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	admin := newDB(dsn, false)
	defer admin.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if err := fx.adoptExistingForCleanup(ctx, admin, red); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	result := fx.teardown(ctx, admin, red)
	encoded, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(encoded))
	if !result.Verified {
		return 1
	}
	return 0
}
