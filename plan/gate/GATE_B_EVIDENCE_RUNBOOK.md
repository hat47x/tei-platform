# Gate B Evidence Runbook

## 目的

判定段階Bで残っている作業を、設計の議論に戻さず、**再現可能な実際の検証根拠の取得**として実行するための正式な手順です。

- 必要な検証根拠： [`GATE_B_EVIDENCE_ACQUISITION.md`](GATE_B_EVIDENCE_ACQUISITION.md)
- 合格・不合格の判定： [`GATE_B_READINESS_AUDIT.md`](GATE_B_READINESS_AUDIT.md)
- 判定段階BからCへの引継ぎ： [`GATE_B_TO_C_HANDOVER_CONTRACT.md`](GATE_B_TO_C_HANDOVER_CONTRACT.md)

**判定段階Bで使う具体的なコマンド、実行順序、安全条件、停止条件については、この文書を唯一の正式な実行手順とします。**

変更履歴はGitの履歴・ADR・リポジトリのIssueから追跡し、この文書には旧版の実行契約を混在させません。

## 現行Gate B scope

判定段階Bの合格に必須となる、ソースの版に依存した検証根拠は次の三つです。

1. #310 リポジトリの決定論的な検証根拠
2. #255 生成成果物の再現性に関する検証根拠
3. #308 実際のPostgreSQL提供実装と一時的な試験用環境の検証根拠

三つとも、変更のない同じ候補版`R`を対象に取得します。

```text
#310 ─┐
#255 ─┼─ repositoryRevision = R
#308 ─┘
       ↓
#387 final convergence
       ↓
Gate B Readiness Audit for R
       ↓
Handover Baseline B / revision R
```

実際のOracleに関する#214・#371・#376は、外部資産の検証根拠として引き続き重要ですが、第四の必須検証経路ではありません。取得済みであれば検証済みの外部資産に関する根拠として引き継ぎ、未完了であれば`unresolved`として渡します。

## Evidence natureを相互代替しない

次の検証結果を互いに読み替えません。

- リポジトリの決定論的な検証の成功 → 実際の外部資産での成功
- 生成成果物の一致 → ドライバーのシナリオの成功
- PostgreSQLの一時的な試験用環境での成功 → 本番の既存データベースでの成功
- PostgreSQLでの成功 → 実際のOracleでの成功
- 自動検査の成功 → 製品の検証根拠を取得済み

業務上の結果、実行時エラー、観測、診断結果も互いに読み替えません。

## 0. 実行環境

### TEI checkout

専用に取得したリポジトリのルートディレクトリから始めます。

```bash
git status --short --untracked-files=all --ignored=matching
git ls-files -v
git rev-parse HEAD
go version
```

追跡対象・追跡対象外・無視対象のローカル状態が一つでも表示されたら停止します。さらに、`git ls-files -v`で追跡対象のすべての項目が通常の`H`であることを求めます。`h`（assume-unchanged）、`S`（skip-worktree）、`s`（両方）、その他の非`H`の印がある作業領域は、検証根拠の取得に使いません。判定段階Bではリポジトリ全体を専用に取得します。既存の差分、無視対象のファイル、索引の設定を機械的に破棄せず、必要なローカルの状態を作業領域の外に移すか、専用の作業領域を作り直してから再確認します。

### Go environment

最終確認では、実行器自身をコンパイルする前から次を明示します。

```text
GOENV=off
GOWORK=off
GOFLAGS=
```

外部の`go.work`、利用者単位のGo環境設定ファイル、`-overlay`・`-modfile`・`-tags`などの暗黙のフラグを、候補版のソースの実行条件に持ち込みません。

### PostgreSQL

#308には、**検証根拠の取得専用となる空のPostgreSQLデータベース**を用意します。

使わないもの:

- `postgres`
- `template0`
- `template1`
- 本番用・共用のデータベース
- 既存の利用者定義の関係を含むデータベース
- イベントトリガーを含むデータベース

### DSN secret file

最終確認では、DSNの内容を起動時の環境変数に設定しません。

リポジトリの外にある既存の通常ファイルへ一行のDSNだけを保存し、`TEI_TEST_POSTGRES_DSN_FILE`でそのファイルを指定します。

PR #466以降、実行器は**DSNを保存した秘密情報ファイルを開いて読み込む前**に、PostgreSQLの秘密情報を含まない環境でリポジトリのソース状態を観測します。

```text
HEAD before
  ↓
git status --porcelain --untracked-files=all --ignored=matching
  ↓
git ls-files -v -z
  ↓
HEAD after
  ↓
HEAD stable && local source/index state clean
  ↓ only then
secret file open / read
```

ソースを観測できない場合、観測中にHEADが変化した場合、追跡対象・追跡対象外・無視対象のローカル状態がある場合、またはGitの索引が非標準の状態にある場合は、秘密情報の読込みへ進みません。この事前確認は、後段で行う候補版のソース確認の代わりにはなりません。秘密情報を読み込んだ後にソースが変化した場合も、各検証経路の開始前に再確認して安全側に停止します。

PR #477以降、共有の`tools/internal/repositoryevidence`は、追跡対象外のファイルが周辺のGit設定によって隠れないよう、開始時も終了時も`--untracked-files=all`を明示します。さらに、実際に使う作業領域とは別のリポジトリ・作業ツリー・索引などを観測させるGitの識別情報の上書きも拒否します。`GIT_CONFIG_COUNT`・`GIT_CONFIG_PARAMETERS`・`GIT_CONFIG_KEY_*`・`GIT_CONFIG_VALUE_*`によるコマンド単位の設定注入は、大文字と小文字の違いも含めて検出し、安全側に停止します。

PR #479以降は`--ignored=matching`も明示します。`.gitignore`・`.git/info/exclude`などで無視されるローカルの状態も、候補版`R`には含まれない入力として`WorkingTreeClean=false`にします。無視対象のGoソースが通常の`git status --porcelain --untracked-files=all`には現れないまま`go list`へ渡される事例が実測されているためです。検証専用の作業領域には、`.idea`・`.vscode`・`.DS_Store`・`.tei`のキャッシュなど、無視対象のローカル状態も残しません。

PR #483以降は`git ls-files -v -z`も、同じHEADを観測している間に取得します。追跡対象のソースに`assume-unchanged`または`skip-worktree`を設定すると、変更後のソースがGoの入力になる一方で、通常のGitの状態表示は空のままになる事例が実測されているためです。追跡対象の項目がすべて通常の`H`である場合だけ索引に変更がないと判定します。`h`・`S`・`s`、不正な印や想定外の非`H`、索引の取得失敗はいずれも成功の根拠に含めません。部分取得に伴う`skip-worktree`も許容せず、検証専用にはリポジトリ全体を取得します。

PR #465以降、秘密情報ファイルはシンボリックリンクの解決後もリポジトリ外にある既存の通常ファイルであることを確認します。そのうえで、**検証済みの解決後のパスを一度だけ開きます**。

- 事前に確認したファイルの識別情報と、開いたファイルの記述子を`os.SameFile`で照合する
- 指定された元のパスを読込み前に再解決し、開いたファイルと同じ外部ファイルを指すことを確認する
- DSNのバイト列そのものを、開いたファイルの記述子から読み取る
- 読込み後にも元のパスを再解決し、同じ外部ファイルを指すことを確認する
- パスの差替え・シンボリックリンクの参照先の差替え・リポジトリ内の参照先・通常ファイル以外は、安全側に停止する

リポジトリ外の通常ファイルを指すリポジトリ外のシンボリックリンクは、読込みの前後で同じ外部ファイルを指す場合に限り受理します。

受理する内容:

- 空でない一行のDSN
- 末尾に任意で付く、一つだけのLFまたはCRLF

拒否する内容:

- 空の値
- 複数行の値
- それ以外に残るCRまたはLF
- NUL
- リポジトリ内のファイル
- シンボリックリンクを解決するとリポジトリ内を指すファイル
- 通常ファイル以外
- 読込みの前後で識別情報が変わるファイルまたはパス

DSNの内容、秘密情報ファイルのパス、認証情報は通常の検証根拠に保存しません。

## 1. Canonical Workspace Validation

事前確認では最初に:

```bash
go run ./cmd/tei validate -workspace reference/workspace
```

期待値:

```text
ok
```

検証エラーがあれば、検証根拠の取得へ進みません。

## 2. #310 deterministic Repository Evidence事前確認

変更のない版で実行します。

```bash
go run ./tools/reference-gate-b-repository-evidence \
  > /tmp/tei-gate-b-repository-evidence.json
```

必要な場合だけ:

```bash
go run ./tools/reference-gate-b-repository-evidence -broad \
  > /tmp/tei-gate-b-repository-evidence.json
```

少なくとも次の項目を確認します。

- `evidenceNature = deterministic-repository`
- 作業領域の検証が成功
- 画面を使わない最終シナリオが成功
- 外部の検証根拠の取込みが成功
- リポジトリのソースの境界の準拠確認が成功
- `repositoryRevision`が空でないこと
- `reproducibleRevision = true`
- 実行後のソースを確認済み
- 実行前後でHEADが同一
- 実行後に変更がないこと

UI-less Scenario:

```text
Authentication
 -> browse
 -> record
 -> reference
 -> header-detail
 -> Canonical Validation
 -> Authorization
 -> Command Business Result
 -> separate Observation read
```

合成データを扱う変換器を含むため、実際の外部資産の検証根拠には格上げしません。

## 3. #255 generated artifact事前確認

```bash
go run ./tools/reference-generated-artifact-evidence \
  > /tmp/tei-gate-b-generated-artifact-evidence.json
```

この実行器は変更を加えず、確認だけを行います。

確認点:

- 意味の正本とプラグインの実装固定情報が一致
- 永続的な実行記録が一致
- `repositoryRevision`が対象版と一致
- `reproducibleRevision = true`
- 実行後のソースを確認済み

生成成果物が古い場合に限り、正式な生成器を使います。

```bash
go run ./tools/reference-golden

go run ./tools/reference-durable-run-golden
```

修復が必要な場合に限って`-write`を使い、差分を意味の観点から確認してコミットします。

```bash
go run ./tools/reference-golden -write
git diff -- reference/world.json reference/workspace/.tei/plugin-lock.json

go run ./tools/reference-durable-run-golden -write
git diff -- reference/order_rollout.durable-run.m0.21.start.json
```

禁止:

- 生成成果物を手作業で編集すること
- 照合値を手作業で置き換えること
- 観測結果を固定した試験用データを生成結果に合わせて変更すること
- 秘密情報や一時的な検証根拠をコミットすること

`-write`の直後に作業ツリーへ変更が残っていても、完了の検証根拠にはなりません。変更のない新しい版で、確認だけを再実行します。

## 4. #308 real PostgreSQL事前確認

単独で使うツールでは、従来どおり`TEI_TEST_POSTGRES_DSN`を入力として指定できます。

```bash
TEI_TEST_POSTGRES_DSN='postgres://...' \
go run ./tools/reference-postgres-header-detail-runtime \
  > /tmp/tei-gate-b-postgres-header-detail.json
```

これは、最終確認で使う秘密情報ファイル経由の入力とは別の事前確認です。

### fixture前安全条件

少なくとも次を確認してから、試験用の構成要素を作成します。

- 実行開始時にリポジトリへ変更がないこと
- 作業領域の検証が成功
- PostgreSQL接続成功
- 既定のデータベースではない
- 利用者が定義した関係が0件
- Event Trigger 0件
- `public.order_lines`不在

### fixture ownership / cleanup

試験用の表には、ツールによる所有を示す印を付けます。

後片付けでは表をロックし、データベース上の印が作成時の印と厳密に一致した場合に限って削除します。

同じ名前の構成要素が別の主体に置き換えられた疑いがある場合は、削除せずに安全側に停止します。

### #308 Evidence

同じrunで少なくとも次の項目を確認します。

- `evidenceNature = real-postgres-ephemeral-fixture`
- `repositoryRevision`が対象版と一致
- `reproducibleRevision = true`
- 試験用の構成要素を作成する前の安全条件が成立
- `order_id=1001`で2件
- `order_id=9999`で正常0件
- 試験用の表を削除した後、表が存在しないことを実行時エラーとして扱う
- 元の実行結果から意味上のOrderLineへの対応付けが明示されている
- 補助列`order_id`を意味上の属性に含めない
- 引数の値を検証根拠に保存しない
- 後片付けの前に所有権の確認が成功
- 後片付け後、利用者が定義した関係0件・イベントトリガー0件・試験用構成要素なし
- 実行後もリポジトリが変更のない同じ版

#308は実際のデータベースエンジンと、ツールが管理する試験用の構成要素による検証根拠です。本番の既存データベースで得た検証根拠ではありません。

## 5. source変更を出し切る

#310・#255・#308の事前確認でソースの変更が必要になったら、修正し、意味の変更内容を確認して、コミットまで済ませます。

最終収束前に:

```bash
git status --short --untracked-files=all --ignored=matching
git ls-files -v
git rev-parse HEAD
```

`git status --short --untracked-files=all --ignored=matching`に何も表示されず、`git ls-files -v`で追跡対象のすべての項目が通常の`H`であることを求めます。追跡対象・追跡対象外・無視対象のローカル状態や、非標準の索引状態が残る作業領域を候補版`R`として固定しません。

この変更のない版を候補版`R`とします。

## 6. final runner package直前検査

同じcheckoutで:

```bash
GOENV=off \
GOWORK=off \
GOFLAGS= \
go test \
  ./tools/internal/repositoryevidence \
  ./tools/reference-gate-b-final-evidence \
  ./tools/reference-postgres-header-detail-runtime \
  ./tools/reference-gate-b-readiness-artifact-audit \
  -count=1
```

これは製品の検証根拠そのものではありません。実際に取得した作業領域で、最終確認の実行器、共通のソース確認用部品、実PostgreSQL用の検証実行器、合格判定用の成果物監査をコンパイル・試験できるか、直前に確かめるものです。

ソースを変更した場合や、作業ツリーに変更が残った場合は、最終確認の実行へ進みません。

## 7. final convergence

### 推奨: same-run child Evidence保存先を明示

リポジトリの外に実行用の親ディレクトリを作り、子ディレクトリは実行器に新規作成させます。

```bash
mkdir -m 700 /tmp/tei-gate-b-R

GOENV=off \
GOWORK=off \
GOFLAGS= \
TEI_TEST_POSTGRES_DSN_FILE='/secure/path/to/dsn' \
go run ./tools/reference-gate-b-final-evidence \
  -artifact-dir /tmp/tei-gate-b-R/children \
  > /tmp/tei-gate-b-R/final.json
```

`children`が既に存在する場合、実行器は上書きせず停止します。

`-artifact-dir`を省略すると、実行器がリポジトリ外に所有者だけがアクセスできる一時ディレクトリを作成し、そのパスを標準エラー出力に表示します。

### broad

#310で`go test ./...`まで実行する必要がある場合に限り、次を使用します。

```bash
GOENV=off \
GOWORK=off \
GOFLAGS= \
TEI_TEST_POSTGRES_DSN_FILE='/secure/path/to/dsn' \
go run ./tools/reference-gate-b-final-evidence \
  -artifact-dir /tmp/tei-gate-b-R/children \
  -broad \
  > /tmp/tei-gate-b-R/final.json
```

### final runner launch guard

secret read / lane開始までに少なくとも次の項目を確認します。

- 現在の作業ディレクトリがGitリポジトリのルート
- プロセスの`GOENV`が正確に`off`
- `go env GOWORK`が空、または`off`
- `go env GOFLAGS`が空
- `TEI_TEST_POSTGRES_DSN`を直接設定していない
- DSNの秘密情報ファイルがリポジトリ外の通常ファイル
- 子処理の検証根拠の保存先がリポジトリ外
- リポジトリのソース状態の観測で、追跡対象・追跡対象外・無視対象のローカル状態がなく、追跡対象の索引項目が通常の`H`だけ

現行の実装は、概念上、次の順序で処理します。

```text
Go environment / Repository root確認
        ↓
Repository外child artifact store準備
        ↓
PostgreSQL secretを除いたenvironmentでclean-source preflight
        ↓
Repository外secret fileをstable single-openで読込
        ↓
Repository外owner-only temporary handoff file作成
        ↓
candidate sourceを再確認
        ↓
#310 → #255 → #308
```

秘密情報の読込み前の事前確認と、検証経路の開始時に行うソース確認は別です。前者に成功した後でソースが変わった場合も、後者で停止します。

周辺環境を確認した後で初めて、子処理の環境設定を次のように固定します。

```text
GOENV=off
GOWORK=off
GOFLAGS=
```

へ固定します。

### secret exposure

```text
outer go run / compiler / linker
  -> DSN本文なし
  -> original secret file pathのみ

final runner runtime
  -> secret-free environmentでclean-source preflight
  -> original external secret fileをstable single-open
  -> 読込前後で同一file identityを再確認
  -> normalized DSN本文をmemoryへ保持
  -> Repository外の0700 temporary directoryへ0600 handoff fileを作成
  -> handoff fileにはnormalized DSN本文だけを保存

Go environment probe / Git probe / #310 / #255
  -> DSN本文なし
  -> original secret file pathなし
  -> handoff file pathなし

#308 go run / compiler / linker
  -> DSN本文なし
  -> original secret file pathなし
  -> temporary handoff file pathのみ

#308 runtime
  -> handoff fileからDSN本文を読込
  -> DSN / file path environmentをsource probe前に消去
  -> 以後DSN本文はmemoryのみ

final runner return
  -> temporary handoff directoryをcleanup
```

引継ぎ用ファイルは、最終確認の実行器が通常終了したときに削除します。プロセスの強制終了やホストの停止などに対して、OSレベルで完全に消去できることまでは保証しません。ファイルの存在する間は、リポジトリ外の所有者だけがアクセスできる一時領域に限定します。元の秘密情報ファイルと引継ぎ用ファイルのパスや、DSNの内容は通常の検証根拠に保存しません。

## 8. same-run child Evidence retention

最終確認の実行器は、子処理それぞれの標準出力にある有効な検証用JSONを、元のバイト列のまま、その実行の成果物としてリポジトリ外へ保存します。

filename:

```text
deterministic-repository.json
generated-artifact-reproducibility.json
real-postgres-ephemeral-fixture.json
```

既存のファイルは上書きしません。

各検証経路の最終結果には、次を記録します。

- `artifactFile`
- `evidenceArtifactFingerprint`

を残します。

照合値は、保存した元のバイト列に対するSHA-256です。

成功した最終確認のJSONでは、次を要求します。

```text
childArtifactsRetained = true
```

を要求します。

**同じ版Rで後から再実行した別の成果物を、最終確認の実行器が実際に読み取った成果物と同一だとは推測しません。** 合格判定の監査では、同じ実行で得た子処理の成果物を参照します。

### 保管（採用したrunのartifact）

合格判定に採用した実行の成果物は、リポジトリへコミットせず、次のように保管します（`ISSUE-000057`）。

- 容量が小さい場合（目安: 数MB以下）: `MANIFEST.sha256`（各fileのSHA-256）を同梱した`tar.gz`へ圧縮し、対象revision（`R` / `T`）を指すtagを付けたGitHub Release（prerelease）のassetとして格納する。tag名は`gate-b-evidence-R-<short>` / `gate-c-evidence-T-<short>`とする。
- 容量が大きい場合：WSL上に保管する（WSLは日次バックアップの対象）。
- 格納前にDSN・認証情報・秘密情報のパスが含まれないことを確認する（監査ツールによる拒否に加えて、grepでも確認する）。
- 格納後、成果物を再度ダウンロードしてSHA-256が一致することを確認する。判定文書には保管先と圧縮ファイルのSHA-256を追記する。過去の判定は書き換えず、追記する。

```bash
gh release create <tag> <archive.tar.gz> --target <R/Tの完全なcommit SHA> \
  --title "<title>" --notes-file <notes.md> --prerelease
```

## 9. lane順序

```text
clean candidate revision R
        ↓
#310 deterministic Repository
        ↓ final runner source re-check
#255 generated artifact reproducibility
        ↓ final runner source re-check
#308 real PostgreSQL Provider
        ↓ final runner source re-check
final source check
```

#310または#255が失敗した場合は、#308を開始しません。

各検証経路が成功・失敗した後、最終確認の実行器自身がソースを再確認します。

## 10. final success条件

final JSON:

```text
evidenceNature = gate-b-three-lane-final-convergence
repositoryRevision = R
workingTreeClean = true
reproducibleRevision = true
allLanesSameRevision = true
childArtifactsRetained = true
postRunRepository.checked = true
postRunRepository.revisionStable = true
postRunRepository.workingTreeClean = true

goEnvironment.externalWorkspaceActive = false
goEnvironment.goFlagsEmpty = true
goEnvironment.goEnvDisabled = true
```

各lane:

```text
passed = true
repositoryRevision = R
reproducibleRevision = true
artifactFile != empty
evidenceArtifactFingerprint != empty
```

Expected Evidence nature:

```text
#310 deterministic-repository
#255 generated-artifact-reproducibility
#308 real-postgres-ephemeral-fixture
```

文字列や照合値を手作業で書き換えません。

## 11. Gate B Readiness Audit

最終確認が成功しただけで、判定段階Bの合格を自動的に宣言しません。

まず、#387の最終結果と、その実行で保存した同一実行内の子処理の成果物ディレクトリを、変更を加えない補助ツールで独立して再監査します。

```bash
GOENV=off \
GOWORK=off \
GOFLAGS= \
go run ./tools/reference-gate-b-readiness-artifact-audit \
  -final /tmp/tei-gate-b-R/final.json \
  -artifact-dir /tmp/tei-gate-b-R/children \
  > /tmp/tei-gate-b-R/readiness-artifact-audit.json
```

補助ツールの成功結果を示すJSONでは、少なくとも次を要求します。

```text
evidenceNature = gate-b-readiness-artifact-audit
artifactAuditPassed = true
repositoryRevision = R
```

を要求します。

このツールは、最終結果と保存した子処理の元のバイト列の照合値、同一版Rと実行後のソース状態、#310・#255の所定の確認項目、#308の正本上の対象識別情報・引数・決定論的な意味上の属性、試験用構成要素の作成前後の安全条件を再照合します。秘密情報や物理的な識別情報に関する禁止項目が混入している場合も、安全側に停止して拒否します。

これは第四の必須検証経路ではありません。`artifactAuditPassed = true`だけで判定段階Bの合格を宣言せず、引き続き`GATE_B_READINESS_AUDIT.md`に従って、版Rについて少なくとも次を監査します。

- #310の同一実行内で得た詳細な検証根拠が成立
- #255の同一実行内で得た詳細な検証根拠が成立
- #308の同一実行内で得た詳細な検証根拠が成立
- 三つとも同じ版Rを対象としている
- `childArtifactsRetained = true`
- 最終結果の照合値と、保存した子処理の元のバイト列が一致
- 異なる種類の検証根拠を互いに代用していない
- 外部で観測する必要がある未確認事項を推測で補っていない
- 秘密情報や実行時の引数の値を、通常の検証根拠へ保存していない
- PostgreSQLの試験用構成要素を作る前後の安全条件が成立
- 業務上の結果・実行時エラー・観測を区別している
- 未解決事項を成功として扱っていない

この監査で版Rが合格と判定された場合に限り、引継ぎへ進みます。

## 12. Gate B → Gate C Handover

合格した同じ版Rを、`ISSUE-000030`・#291の引継ぎ基準版として固定します。

引継ぎ記録から、少なくとも次の情報を追跡できるようにします。

- リポジトリの版R
- 正本上のアプリケーション・パターン・データ・関係・問い合わせ・能力
- ドライバーのシナリオ
- データ接続先と権限の境界
- 業務上の結果・実行時エラー・観測
- #310・#255・#308の同一実行内の検証根拠
- 最終確認の成果物
- 版Rについての判定段階Bの最終監査
- 任意に追加する、実際のOracleで取得した検証済みの根拠
- 未解決事項

引継ぎ記録は、意味の第二の正本ではありません。

実際のOracleに関する検証が未完了でも、#214・#371・#376を再び通過の阻害条件とはしません。観測できていない範囲を`unresolved`として明示し、ウェブ側で補いません。

## 停止条件

次のいずれかに当てはまる場合、成功の検証根拠として採用しません。

- 作業領域の検証エラー
- リポジトリのルート以外から最終確認を実行
- 開始時に作業ツリーへ変更がある（追跡対象・追跡対象外・無視対象のローカル状態を含む）
- Gitの索引に`assume-unchanged`・`skip-worktree`その他の非標準の印があり、追跡対象の全項目を通常の`H`にできない
- 部分取得などで`skip-worktree`が設定され、リポジトリ全体を取得していない状態
- 秘密情報の読込み前に、リポジトリのソースが変更されず安定していることを確認できない
- リポジトリのソースを観測する環境に、Gitの識別情報の上書きや`GIT_CONFIG_*`によるコマンド単位の設定注入がある
- ソースを観測できない、またはHEADが変化した
- 検証経路の実行後に変更が残った、または版が変化した
- 必須の検証経路が失敗
- `reproducibleRevision != true`
- lane `repositoryRevision != R`
- 検証根拠の種類が一致しない
- final `allLanesSameRevision != true`
- `childArtifactsRetained != true`
- 子処理の成果物を保存できない
- 成果物ディレクトリがリポジトリ内にある
- 子処理の成果物を既存ファイルへ上書きする必要がある
- 最終的なソースの確認が未完了
- 合格判定に用いる成果物の監査が失敗
- 外部のGo作業領域が有効
- 実効的なGOFLAGSが空でない
- 正確に`GOENV=off`を指定して起動していない
- `TEI_TEST_POSTGRES_DSN`を直接設定している
- DSNの秘密情報ファイルがリポジトリ内にある、シンボリックリンクの参照先がリポジトリ内、通常ファイル以外、空、複数行、NULを含む、または読込み不能
- DSNの秘密情報ファイルのパス、シンボリックリンクの参照先、またはファイルの識別情報が読込み前後で変化する
- PostgreSQL用の引継ぎファイルを、リポジトリ外の所有者だけがアクセスできる一時領域に作成できない
- PostgreSQLが既定のデータベースを指す
- PostgreSQLに既存の利用者定義の関係やイベントトリガーがある
- 試験用の構成要素が最初から存在する
- 試験用の構成要素の所有者を示す印が一致しない
- 後片付け後も試験用の構成要素、利用者定義の関係、イベントトリガーが残る
- 古い生成成果物が修復されていない
- 検証根拠に秘密情報が混入した疑いがある

ソースの修正が必要ならコミットし、変更のない新しい版を新たなRとして、一連の検証を最初からやり直します。

## 非目標

- 実際のOracleを第四の必須検証経路に戻すこと
- Oracleで未確認の事項をPostgreSQLや合成データの検証根拠から推測して補うこと
- 引継ぎ前に最初のウェブ実行先へ着手すること
- 生成成果物を手作業で編集すること
- 秘密情報管理製品を標準化すること
- Goの開発・構築ツール一式やモジュールキャッシュをリポジトリへ取り込んで管理すること
- OSのプロセス全体を隔離環境で実行すること
- GitHub Actionsの成功を判定段階Bの製品検証根拠として格上げすること

## 関連

- `GATE_B_EVIDENCE_ACQUISITION.md`
- `GATE_B_READINESS_AUDIT.md`
- `GATE_B_TO_C_HANDOVER_CONTRACT.md`
- `UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md`
- ISSUE-000025-reference-generated-artifacts-refresh.md
- ISSUE-000029-gate-b-readiness-audit.md
- ISSUE-000030-gate-b-to-c-handover-baseline.md
- ISSUE-000031-gate-b-postgres-header-detail-evidence.md
- ISSUE-000032-gate-b-deterministic-repository-evidence.md
- ISSUE-000037-gate-b-three-lane-final-evidence.md
- `tools/reference-gate-b-final-evidence/README.md`
- `tools/reference-gate-b-readiness-artifact-audit/README.md`