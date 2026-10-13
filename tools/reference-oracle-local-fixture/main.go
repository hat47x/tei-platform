// reference-oracle-local-fixtureは、手元の使い捨てOracle fixtureに対して、Oracle driver、
// named bind、datatype、層別のread-only確認、Evidence intakeを実行し、secretを含まないJSON receiptを出す。
//
// これはexternal-observedな実際のmaster_data資産のEvidenceではない。
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
	"time"

	"tei/internal/database/externalqueryevidence"
	"tei/internal/environment"
	"tei/internal/realization"
	"tei/internal/validation"
	"tei/internal/workspace"
	"tei/tools/internal/repositoryevidence"
)

// evidenceNatureは、このreceiptの種類を示す。external-observedではない。
const evidenceNature = "local-ephemeral-oracle-fixture"

type driverFacts struct {
	Module       string `json:"module"`
	Version      string `json:"version"`
	RegisteredAs string `json:"registeredAs"`
	GoVersion    string `json:"goVersion"`
	Isolation    string `json:"isolation"`
}

type connectionFacts struct {
	Transport string `json:"transport"`
	TLS       bool   `json:"tls"`
	Wallet    bool   `json:"wallet"`
	Loopback  bool   `json:"loopback"`
	Note      string `json:"note"`
}

type receipt struct {
	EvidenceNature       string                     `json:"evidenceNature"`
	ExternalObserved     bool                       `json:"externalObserved"`
	Completed            bool                       `json:"completed"`
	AllChecksPassed      bool                       `json:"allChecksPassed"`
	FailedChecks         []string                   `json:"failedChecks"`
	Error                string                     `json:"error,omitempty"`
	RepositoryRevision   string                     `json:"repositoryRevision"`
	WorkingTreeClean     bool                       `json:"workingTreeClean"`
	ReproducibleRevision bool                       `json:"reproducibleRevision"`
	PostRunRepository    repositoryevidence.PostRun `json:"postRunRepository"`
	Driver               driverFacts                `json:"driver"`
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

// notProvenは、このreceiptが証明しないことを常に明示する。
var notProven = []string{
	"external-observedな実際のmaster_data資産（実Oracle）のphysical facts、型、NULL扱い、native realizationは観測していない",
	"実資産のprovenance・加工履歴、実資産に対するread-only principalのgrant構成は取得していない",
	"TCPS / Wallet / 認証方式は未確認（このrunはloopbackのTCP平文、password認証だけ）",
	"本番principalのgrantではない。使い捨てprincipalに対するgrantの観測であり、実外部assetのEvidenceへ読み替えない",
	"候補revision Rでの最終actual readではない。fixtureはこのrunのために作成して破棄した",
	"read-only transactionをpoolでどう運用するかの方針は決めていない。接続単位で持続させる方式はsnapshotが接続開始時点に固定される",
}

func main() {
	workspacePath := flag.String("workspace", "reference/workspace", "Contract Queryを解決するWorkspace directory")
	evidenceOut := flag.String("evidence-out", "", "受入れたexternal Evidence JSONを書き出すpath（Repository外を指定すること）")
	cleanupRunID := flag.String("cleanup", "", "run idを指定すると、そのrunが作成したfixture userだけを削除する（中断後の後始末用）")
	timeout := flag.Duration("timeout", 10*time.Minute, "全体のtimeout")
	flag.Parse()

	if *cleanupRunID != "" {
		os.Exit(runCleanup(*cleanupRunID))
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	result, err := run(ctx, *workspacePath, *evidenceOut)
	if err != nil {
		result.Error = err.Error()
	}
	encoded, marshalErr := json.MarshalIndent(result, "", "  ")
	if marshalErr != nil {
		fmt.Fprintln(os.Stderr, "receiptを出力できません")
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
			"fixture-owner":            "使い捨てfixture tableの所有者。CREATE SESSIONとCREATE TABLEだけ",
			"read-only-principal":      "fixture tableへのREAD object privilegeとCREATE SESSIONだけ",
			"select-control-principal": "READとの対照用。fixture tableへのSELECT object privilegeとCREATE SESSIONだけ",
		},
		Connection: connectionFacts{
			Transport: "TCP",
			TLS:       false,
			Wallet:    false,
			Loopback:  true,
			Note:      "loopbackのTCP平文とpassword認証だけを確認した。TCPS / Wallet / 他の認証方式は未確認",
		},
		Driver: driverInfo(),
	}
}

func driverInfo() driverFacts {
	facts := driverFacts{
		Module:       "github.com/sijms/go-ora/v2",
		RegisteredAs: "oracle",
		GoVersion:    runtime.Version(),
		Isolation:    "driverのimportはこのtool packageだけ。internal/配下のCoreとOracle Provider / QueryRuntimeはdriverを知らない",
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

func run(ctx context.Context, workspacePath, evidenceOut string) (receipt, error) {
	result := newReceipt()

	source, err := repositoryevidence.InspectStart()
	if err != nil {
		return result, err
	}
	result.RepositoryRevision = source.RepositoryRevision
	result.WorkingTreeClean = source.WorkingTreeClean
	if !source.WorkingTreeClean {
		return result, fmt.Errorf("receiptをclean revisionへ固定するため、working treeに未commit差分がある状態では実行しません")
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
		return result, fmt.Errorf("Workspace %sを読み込めません", workspacePath)
	}
	for _, diagnostic := range validation.Validate(loaded.World) {
		if diagnostic.Severity == validation.Error {
			return result, fmt.Errorf("Workspaceにvalidation errorがあります: %s", diagnostic.Code)
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

	runErr := s.execute(admin, evidenceOut, &result)

	// 失敗や中断の場合も、作成した使い捨てuserを必ず削除して結果を再確認する。
	cleanupCtx, cancelCleanup := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancelCleanup()
	cleanup := fx.teardown(cleanupCtx, admin, red)
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
		runErr = fmt.Errorf("使い捨てfixtureの削除を確認できませんでした")
	}
	if runErr == nil && !result.ReproducibleRevision {
		runErr = fmt.Errorf("実行中にrepositoryのrevisionまたはworking treeが変化したため、単一のclean revisionへ固定できません")
	}
	return result, runErr
}

// executeは、使い捨てfixtureの作成から各確認、Evidence intakeまでを行う。teardownは呼び出し側が行う。
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
	s.record("provision.read-privilege-grantable", layerGrant, "fixture ownerがread-only principalへREAD object privilegeをgrantできた（Oracle 12.1.0.2以降の機能）", true, "")

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
		s.record("intake.facts", layerIntake, "観測した事実をEvidenceへ写せる", false, err.Error())
		return nil
	}
	evidence, err := buildEvidence(s.def, facts)
	if err != nil {
		s.record("intake.build", layerIntake, "観測した事実とauthoring済みrealizationからEvidenceを組み立てられる", false, err.Error())
		return nil
	}
	definitions := []realization.ContractQuerySQLRealization{s.def}
	intake, raw, err := intakeEvidence(s.world, definitions, evidence)
	if err != nil {
		return err
	}
	result.Intake = &intake
	s.record("intake.accepted", layerIntake, "既存のexternalqueryevidenceの厳格loaderとValidateAgainstがEvidenceを受理する", intake.Accepted, intake.Error)

	// SQL realizationがないtarget / providerは、Evidenceから補完せず停止する（既存tool・既存規則と同じ境界）。
	_, stopErr := findAuthoredRealization(nil, evidence.Target, evidence.Provider)
	s.record("intake.stops-without-realization", layerIntake, "authoring済みrealizationがない場合はEvidenceから補完せず停止する", stopErr != nil, "")

	// providerが異なるrealizationしかない場合も補完しない。
	other := s.def
	other.Provider = environment.ProviderPostgres
	_, stopErr = findAuthoredRealization([]realization.ContractQuerySQLRealization{other}, evidence.Target, evidence.Provider)
	s.record("intake.provider-must-match", layerIntake, "provider違いのrealizationを選ばない", stopErr != nil, "")

	// native sourceのfingerprintが違えばstaleとして拒否する。
	stale := evidence
	stale.NativeSource.Fingerprint = "sha256:" + fmt.Sprintf("%064d", 0)
	if err := externalqueryevidence.ValidateAgainst(s.world, s.def, stale); err != nil {
		s.record("intake.rejects-stale-fingerprint", layerIntake, "native source fingerprintが違うEvidenceを拒否する", true, "")
	} else {
		s.record("intake.rejects-stale-fingerprint", layerIntake, "native source fingerprintが違うEvidenceを拒否する", false, "")
	}

	if evidenceOut != "" && intake.Accepted {
		if err := os.WriteFile(evidenceOut, append(raw, '\n'), 0o600); err != nil {
			return fmt.Errorf("Evidence JSONを書き出せません")
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
		ProcessingNote:  "このtoolが作成した使い捨てfixture tableのdata dictionaryとdriver metadataを観測した。external-observedな実資産の観測ではなく、実資産のEvidenceへ読み替えない",
		ReadOnlyMech:    "READ object privilegeだけのprincipal、SET TRANSACTION READ ONLY(connector)、QueryContextだけのRunnerの多層。使い捨てfixtureでの観測",
		ReadOnlyRef:     "local-ephemeral-oracle-fixture:" + s.fx.RunID + "/receipt#grant,transaction,statementMatrix",
	}
	if facts.ObjectKind == "" {
		return facts, fmt.Errorf("fixture objectの種類をdictionaryから観測できていません")
	}
	idColumn, ok := s.dictionaryColumn("CUSTOMER_ID")
	if !ok {
		return facts, fmt.Errorf("CUSTOMER_ID列をdictionaryから観測できていません")
	}
	facts.InputPhysical = idColumn.physicalType()
	if len(s.Metadata) == 0 {
		return facts, fmt.Errorf("actual readでraw列metadataを観測できていません")
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
	result := fx.teardown(ctx, admin, red)
	encoded, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(encoded))
	if !result.Verified {
		return 1
	}
	return 0
}
