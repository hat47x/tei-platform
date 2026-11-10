# TEIを動かす

この文書では、TEIを取得して動作を確認するまでの手順を説明します。設計の考え方から知りたい場合は、[はじめてTEIを知る方へ](for-readers.md)を参照してください。

## 前提

TEIは現在、設計上の境界をGoのコードで検証している段階です。この文書では完成した画面製品の操作ではなく、参照実装を使った動作確認を説明します。

主要な手順を試すには、Goの開発環境が必要です。必要なGoの版は、リポジトリ直下の`go.mod`にある`go`行が示す版以上です。この文書は版の数値を複製しません。`go.mod`の更新のたびに文書が古くなるためです。2026-11-07時点の`go.mod`は`go 1.26.10`を要求します。数値は必ず`go.mod`で確認してください。


`tools/` の参照ツールのうち、外部コマンドを `bash -c` で起動する試験（`tools/reference-cognition-port`・`tools/reference-generator-selection`）は **POSIXシェルを前提**にしています。2026-11-09以降は、Windowsでも同じ試験を実行します（#3461）。Windowsの`bash`はWSLの`bash.exe`が多く、`C:\...`のパスを開けず、`-c`文字列中の`$NAME`をWindowsの環境変数で先に展開します。そのため両ツールは、コマンドを一時ディレクトリをworking directoryにして実行し、入力をそこへ複製し、パス変数をコマンド文字列へ直接埋め込みます。`bash`が見つからない環境だけがskipします。skip は失敗ではなく「この環境では検査していない」という記録です。`go test ./...` の結果を読むときは、skip の理由も確認してください。
```bash
grep '^go ' go.mod
go version
```

2つの出力を見比べ、`go version`の版が`go.mod`の要求以上であることを確認してください。

版が足りない場合、`GOTOOLCHAIN=local`を設定した環境では、`go run ./cmd/tei version`が`go: go.mod requires go >= ...`で失敗します。TEIのCIの多くは`GOTOOLCHAIN: local`で動かし、版は`go-version-file: go.mod`から取ります（`.github/workflows/*.yml`に`go-version:`の直指定を残さないことを`tools/repository-architecture-check`の試験が検査します）。`GOTOOLCHAIN`が`auto`（既定）の環境では、Goがネットワークから必要な版を取得することがあります。ダウンロードを許可しない環境では、`go.mod`の要求以上のGoを事前に用意してください。この失敗はTEIの検証結果ではなく、実行環境の不足です。

### Node.jsを使う試験の前提

リポジトリ全体のGo試験は、`go test ./...`の一部として、どのworkflowからも参照されていなかったNode.js試験5ファイルを実行します（`tools/repository-node-tests/`）。Node.jsが無い環境ではskipし、GitHub Actionsでは必須です。手元でNode.js試験を再現する場合はNode.jsを用意してください。

個別の参照ツールは、それぞれの`package.json`の`engines`が要求版を示します。2026-11-07時点では、`reference/tei-studio-independent-consumer`が`>=22.16.0`、`reference/tei-studio-workbench`が`>=24`です。workflowには`22`・`22.16.0`・`24`・`24.21.0`が混在しており、単一の版へ統一していません。使うツールの`package.json`を確認し、版が合わない場合はそのツールの試験を実行しないでください。この混在を解消するかは未決です（[#2953](https://github.com/hat47x/tei-platform/issues/2953)）。

### Windowsでの`go test ./...`

2026-11-10時点のmain（`fb766fc5f`系）で、Windows 11・go1.27.0の`go test ./...`は完走し、**CIの`windows-2025`でも `scope=full` が緑**になりました（run [42694376462](https://github.com/hat47x/tei-platform/actions/runs/42694376462)、`GOTEST_EXIT=0`、`-count=1 -timeout 90m`で90分規模、skip 288件）。2026-11-09の初回実行では18件が失敗しており、その原因は「8.3形式の短縮名（`C:\Users\RUNNER~1\...`）によるパス同一性の誤判定」「`core.autocrlf=true` による生バイトのCRLF化」「POSIXシェル資格フロー」「OS固有のエラー文言」でした（#3783）。ただし、次の2種類の注意があります。

**1. Windowsでは理由付きでskipする検査があります。**skipは通過と同じ意味ではありません。

| 種類 | 対象 | skipの理由 |
| --- | --- | --- |
| POSIXのパーミッションビット | `internal/access`・`internal/effectlog`・`internal/jsonl`・`internal/runlog`・`internal/workspace`など | Windowsは`0600`で作ったファイルを`0666`と報告する。ACLの確認はHostの責務 |
| POSIXのinode置換 | `tools/reference-build-artifact-store` | 読み取り専用の既存ファイルへのrenameが`Access is denied`になり、置換を同じ意味で表せない（[#3460](https://github.com/hat47x/tei-platform/issues/3460)） |
| ~~`bash -c` + `cp`の外部コマンド~~ | ~~`tools/reference-cognition-port`・`tools/reference-generator-selection`~~ | 2026-11-09に解消（[#3461](https://github.com/hat47x/tei-platform/issues/3461)）。両ツールはコマンドを一時ディレクトリで実行し、パス変数をコマンド文字列へ埋め込むため、WindowsのWSL `bash.exe`でも動く。`bash`が見つからない場合だけskipする |
| 空白を含む実行パス | `tools/reference-composition-process-bridge` | 2026-11-09に解消（#3460）。参照のstdio launcherが空白入りの実行パスを拒否するのは契約なので、試験は**ホストの一時ディレクトリに空白がある場合だけ**skipする。Windowsでも既定の`%TEMP%`なら実行される |
| OpenSSHの`-Y verify verify-time` | `tools/reference-authoring-witness-openssh` | 同梱のOpenSSHが`verify-time`非対応で、スクリプトが`exit 77`（「No result is claimed」）を返す |
| シンボリックリンク・JVM等 | `internal/workspace`・`tools/internal/gobuilder`・`tools/reference-target-profile`など | シンボリックリンクは、作成を試して失敗したときだけskipする（理由にOSのエラーを出す）。開発者モード等で作成できる環境では実行される。JVM・node・pythonは未導入の環境でskip |

skipの総数は32件（`t.Skip`の理由にWindowsを含むもの）。Linuxで実行した結果を正とします。

**Windowsで落ちやすい4つの型と、その避け方**（#3783・#3971 で解消した記録から）:

1. **パスの同一性は両辺を解決してから判定する。** `filepath.EvalSymlinks` した子と未解決の root を `filepath.Rel` で比べると、短縮名の環境で「外に出た」と誤判定します。`os.SameFile` を使うのが確実です。
2. **生バイトを比較する試験は LF 固定に頼る。** GitHubのWindowsランナーは `core.autocrlf=true` で、リポジトリは `.gitattributes` の `* text=auto eol=lf` でLFに固定しています。生バイトを保持したいパスは従来どおり `-text` を列挙します。
3. **OS固有のエラー文言ではなく、エラー値で判定する。** `"no such file"` はUnixの文言で、Windowsは "The system cannot find the file specified." です。`errors.Is(err, fs.ErrNotExist)` を使います。

4. **wasm候補は最小の環境で起動する。** Goのwasmランタイムは環境変数全体をインスタンスへコピーします。CIランナーの環境ブロックが大きいと `total length of command line and environment variables exceeds limit` で起動に失敗し、プロセスは終了0のまま出力が空になるため、receiptには「全ケース空」とだけ残ります（#3971）。`reference-generator-selection` は `SystemRoot`・`TEMP`・`TMP`・`HOME`・`PATH` だけを渡しています。

**2. 比較試験はチェックアウトがcleanであることを前提にします。**`tools/reference-plugin-self-build`と`tools/reference-transform-runtime-engine-build`のhistorical比較は、`git status --porcelain=v1 --untracked-files=all`が空であることを確かめてから実行します。`go test ./...`はパッケージを並列に実行するため、別パッケージがcheckout内へ書いた直後にこの前提を読むと落ちることがあります（[#3637](https://github.com/hat47x/tei-platform/issues/3637)）。失敗時は汚れていたpathがメッセージに出るので、原因のパッケージを特定できます。単体で`go test ./tools/reference-plugin-self-build -count=1`を実行した場合は落ちません。

PRごとのWindows確認は`tei-windows-go-test-survey.yml`が`workflow_dispatch`と本workflow自身の変更時に行い、既定（`scope=survey`）では`./internal/plugin/runtime`・`./internal/desktophost`・`./internal/operations`・`./cmd/tei`・`./internal/workspace`・`./internal/modelref`・`./reference/application-composition`・`./tools/reference-developer-context`・`./tools/reference-build-catalog-authoring`・`./tools/reference-gate-b-final-evidence`・`./tools/reference-target-package-evidence`を対象にします。`scope=full`で`go test ./...`全体（90分規模）を手動実行できます。全パッケージは時間がかかるため必須検査にしていません。

## 最初の小さな作業領域から始める

大きな参照用作業領域である`reference/workspace`を調べる前に、独立した小さな作業領域で、業務上の意味を一か所変更し、検証と入力元索引で結果を確認できます。

手順は[最初の意味変更を試す](../examples/first-workspace/README.md)に記載しています。Goの開発環境だけで実行できます。ただし、これはモデルの検証例であり、画面の生成やデータの登録・照会・更新・削除はできません。


### 外部の呼出し元が書く最小の作業領域

外部の呼出し元が文書だけを読んで作れる最小の形は、次の2ファイルです。`workspace:` の名前は任意で、`formatVersion` と `sources` は必須です（`formatVersion` がないと `WORKSPACE_LOAD_FAILED`、`sources` が空だと `sources must contain at least one source root`）。

```text
my-workspace/
  tei.workspace.yaml
  semantic/model.tei
```

```yaml
# tei.workspace.yaml
workspace: scn001
formatVersion: 1
sources:
  - semantic
```

```tei
namespace scn

data RuleCandidate {
    attribute sourceVersion type std.text
}
```

この2ファイルで `go run ./cmd/tei validate -workspace my-workspace -json` は `ok:true`、`declarationCounts` は `namespaces:1`・`data:1`・`attributes:1`・`total:3` になります（`types` は `std` しか無いため0）。`model catalog` の `entries` には `semantic` / `scn`・`semantic-data` / `scn.rule_candidate`（`symbol:RuleCandidate`）・`semantic-attribute` / `scn.rule_candidate.source_version`（`symbol:sourceVersion`）が出ます。標準定義の `std` は `prelude:true` で印を付けます。

**宣言名はCanonical IDへ変換されます。** identifier（namespace・data・attributeなど）は、宣言した `symbol` ではなく、次の規則で作ったCanonical IDを `subject.id` として持ちます（`core/identity`）。

| 宣言 | 変換 | 例 |
| --- | --- | --- |
| namespace | 名前をそのまま小文字のまま使う | `namespace scn` → `scn` |
| data | `namespace.` + symbolをsnake_caseへ | `data RuleCandidate` → `scn.rule_candidate` |
| attribute | 親dataのID + `.` + symbolをsnake_caseへ | `attribute sourceVersion` → `scn.rule_candidate.source_version` |
| relation・candidate-key・functional-dependency | 親dataのID + `.` + symbolをsnake_caseへ | `relation OrderLines` → `scn.order.order_lines` |
| capability・validation・type | `namespace.` + symbolをsnake_caseへ | `type OrderStatus` → `scn.order_status` |

snake_caseへの変換は、大文字の前に`_`を置き、`-`・空白・連続する`_`を1つの`_`にまとめ、前後の`_`を落とす操作です。`sourceVersion` は `source_version`、`SourceVersion` も同じ `source_version` になります。**大文字小文字だけが違う2つの宣言は同じIDへ潰れ、`DUPLICATE_SEMANTIC_ELEMENT` になります**。`model catalog` の `entries[].symbol` は宣言した綴り、`subject.id` は変換後のIDです。呼出し元が宣言と応答を突き合わせるときは、`symbol` ではなくこの変換を経た `subject.id` を使ってください。

## コマンドラインを生成AIや自動化から使う

当面はコマンドラインから`tei`を操作する方法を基本とします。現在使える主な読み取り専用コマンドは、次のコマンドが返すJSON形式の一覧で確認できます。

```bash
go run ./cmd/tei commands
```

結果を記録して後から再現できるようにするには、使った`tei`の識別も一緒に残します。`go build`で作った実行ファイルに対して次を実行すると、埋め込まれたコミット（`vcsRevision`）、未コミット変更の有無（`vcsModified`）、Goのバージョン、OSとCPUの種類が返ります。

```bash
go build -o tei ./cmd/tei
./tei version -json
```

`go run`や`-buildvcs=false`でbuildした場合はコミットが埋め込まれないため、`source`が`unavailable`になります。識別は再現の手掛かりであり、配布の承認や署名ではありません。

checkoutの外でbuildするなど、toolchainがコミットを埋め込めない場合は、buildする側がrevisionを注入できます。

```bash
go build -buildvcs=false \
  -ldflags="-X main.teiDeclaredRevision=$(git rev-parse HEAD)" \
  -o tei ./cmd/tei
./tei version -json
```

この場合、`source`は`ldflags`になり、値は`vcsRevision`ではなく`declaredRevision`へ入ります。`declaredRevision`は**buildした側の自己申告**で、実行物がそのcommitのsourceから作られたことを証明するものではありません。toolchainの`vcsRevision`がある場合は、より強い証拠として`source`は`build-info`のままです（両方を持つ場合も`declaredRevision`は残ります）。注入しないbuildは、従来どおり`source`が`unavailable`になります。

モデルや定義の整合性を調べる場合、生成AIなどの呼出し元は次のようにJSON形式の診断結果を取得できます。

```bash
go run ./cmd/tei validate \
  -workspace reference/workspace \
  -json > /tmp/tei-validation.json
```

結果には`schemaVersion`、`ok`、`diagnostics`が含まれます。構文の誤りも読込みの失敗もなく読めた場合は、`worldFingerprint`と`sourceIndexFingerprint`も返します。値は`model catalog`・`inspect`・`trace`と同じ計算で、同じファイル内容なら一致し、内容が変われば変わります。どの版のWorkspaceを検証した結果かを後から照合する証拠であり、内容が正しいことや受理されたことの証拠ではありません。構文に誤りがある場合と読込みに失敗した場合は、版が存在しないため両項目を省きます。構文に誤りがあれば元ファイルの位置を示し、意味の検証では可能な範囲で対象要素と入力元索引の位置を返します。`-json`を指定しない従来の`validate -workspace`は、テキスト形式で結果を返します。警告だけの場合、テキストは警告行を出力した後に`ok`を出力し、終了コードは`0`です。標準出力全体が`ok`と一致することを成功条件にすると、警告付きの成功を失敗と誤判定します。成否は終了コード、または`-json`の`ok`で判定してください。`ok`が`true`でも`diagnostics`に警告が残ることがあります。

能力の入出力については、宣言した型の実在を検査しますが、ある能力の出力と次の能力の入力が接続できることは検査しません（[参照例](../reference/product-contracts/README.md#境界)）。

意味要素の重複診断では、`-json`が同じ識別子を持つローカルの宣言位置を候補として複数返すことがあります。各候補の`subject`はその宣言の種類と識別子です。候補の順序はファイルと位置による表示順であり、どの宣言を採用・拒否すべきかは示しません。標準定義（prelude）との重複では、ローカルの候補に加えて`unresolvedSourceOrigins`へ`standard`を返し、標準定義側の位置を実在しないpathで埋めません。importがある場合でも、ローカル宣言同士の重複候補は省略しません。importとローカル宣言が同じ名前空間を共有する場合は読込み時に拒否されるため、重複診断には到達しません。位置を取得できない宣言が残る場合は`unknown`を返します。テキスト形式では一つの候補だけを代表位置として表示しません。

版を固定する値は、checkoutの改行（CRLF/LF）の受け取り方にも依存します。`sourceIndexFingerprint`と`modelRevision`（`model catalog`が返す、管理対象Sourceの版。Shared Acceptanceが照合する`baseModelRevision`と同じ値）は、Sourceの生バイトをsha256で識別するため、行末だけが違うcheckoutでも別の版になります。`worldFingerprint`は正規化したモデルの内容を識別しますが、元定義に書かれた本文をそのまま載せる欄（移行のDDLや問い合わせのSQLなど）があると、その本文の改行も反映されます。2026-10-31にmainで実測した結果では、`reference/workspace`（`databases/sales-core/migrations/20260826_001/migration.tei`と`persistence/order.tei`がSQL本文を持つ）ではCRLFのcheckoutで`worldFingerprint`も変わり、その種の欄を持たない`examples/first-workspace`では`worldFingerprint`は変わらず`sourceIndexFingerprint`と`modelRevision`だけが変わりました。別の環境で作った版固定の値を照合するときは、比較する側のcheckoutの改行をそろえてください。リポジトリ内のWorkspaceの木（`reference/workspace`、`examples/`配下、`reference/product-contracts`ほか、`tei.workspace.yaml`のある8木）は`.gitattributes`で`-text`に固定してあり、新しいcheckoutでは改行が書き換わらない。`core.autocrlf`が有効な環境で作った古いcheckoutで該当pathが`modified`に見えるときは、差分を確認し、必要な変更を保存したうえで、意図したファイルだけを復元してください。`git restore`は未保存の変更を破棄します。[#2399](https://github.com/hat47x/tei-platform/issues/2399)（2026-10-31クローズ）では、生バイトでの版識別を維持し、改行正規化後のバイトで計算する契約への変更を見送る仮決定をしました。この変更は、Shared Acceptanceの版を張り替える時点で改めて判断します。

`workspace-files`の各要素は`readByLoader`を持ちます。`true`はそのファイルのバイト列をLoad中にdomain loaderが読んだこと、`false`は管理対象だが解釈されていないことを示します。`workspace-files`の`sourceRootFindings`は、未知・存在しないsource rootを名指しします。`readByLoader:true`だけではParserの意味受理や整合性合格を意味しません。

`validate -json`の応答には`sourceRootFindings`が入ります。`tei.workspace.yaml`の`sources`に存在しないrootがあれば`severity:"error"`（`ok:false`、終了コード1）、Loaderが解釈しない未知のrootなら`severity:"warning"`です。`workspace-files`に載るがLoaderが読まない近似名のファイル（`.tei.bak`・`Application.tei`など）も`file`付きのwarningとして返します。これらはCanonical Subjectを持たないため`diagnostics`ではなく別の欄です。

`tei.workspace.yaml`はYAML全体ではなく、次の部分集合です。表に無い書き方は`WORKSPACE_LOAD_FAILED`（終了コード2）で拒否され、エラー文に行番号が付きます（2026-11-09から）。

| 要素 | 受理する形 | 拒否・注意 |
| --- | --- | --- |
| ファイル先頭 | UTF-8。先頭のBOMは1つだけ無視する | 2つ目のBOM、途中のBOMは拒否する（#3556） |
| `workspace:` | 名前（省略可）。`"…"`または`'…'`で囲んだ値も受理し、引用符を外して応答の`workspace.name`に返す | 閉じ引用符の欠落、引用符内の同じ引用符の未エスケープ、閉じ引用符の後ろの文字は拒否する |
| `formatVersion:` | `1`または`2`の**引用符なし**の整数。`sources`より前に置く | `"1"`のような引用符付き、0以下、非整数、重複、`sources`の後ろは拒否する |
| `sources:` | 次の行に`- `で始まるrootを1行1つ。最小1つ | インライン配列（`sources: [semantic]`）は拒否する。`- "semantic"`は引用符を外してroot名にする |
| `imports:` | `- alias:`で始め、続く`path:`・`namespace:`・`semanticDigest:`を書く | 各値は引用符を外せる。未知の項目は拒否する |
| コメント | 行頭の`#`、または空白の後ろの`#`から行末まで | 値の途中の`#`は値の一部として扱う（`workspace: scn#1`の名前は`scn#1`）（#3557） |
| 未知のキー | — | 拒否する（`description: hi`など） |

引用符の中のエスケープ（`\"`・`\n`など）は解釈しません。引用符を外した文字列をそのまま使うため、引用符内で同じ種類の引用符を使うときは値を引用符なしで書いてください。`#`を含む名前に空白の後ろの`#`を使うと、そこからコメントになります。`workspace: "scn #1"`のように引用符で囲むと、`#`以降も名前の一部として残ります。

Workspaceがモデル要素を1つも宣言していない場合、`validate`は`WORKSPACE_DECLARES_NO_MODEL_ELEMENTS`（warning）を返します。`ok:true`のまま終了コード0ですが、`diagnostics`にこのwarningがあるため「宣言が読み込まれて検証を通った」と読み違えないでください。標準意味型（`std`）は宣言として数えません。

`validate -json`の応答には`declarationCounts`が入ります。これはWorkspace自身が宣言した要素の件数で、プロセス内の標準意味型（`std`）を含みません。`ok:true`でも`declarationCounts.total`が`0`なら、そのWorkspaceはモデル要素を何も宣言していません。`model catalog`の応答にも同じ`declarationCounts`があります。標準定義だけの13件の`entries`は、利用者の宣言を意味しません。`model catalog`の各`entries`要素は、標準意味型（`std`）に属するものを`prelude:true`で示します（2026-11-08から）。`prelude`が無い要素が、Workspace自身の宣言です（#3347）。

終了コードは`0`が検証成功（警告を含む）、`1`が検証不成立、`2`が入力・読込みなどの実行失敗です。`-json`で返す結果を利用するときも終了コードを確認してください。`commands`はすべてのコマンドや拡張機能を列挙する完全な登録簿ではありません。詳細とウェブやデスクトップへの接続方針は[コマンドライン優先のインターフェース設計](../plan/design/CLI_FIRST_INTERFACE_DESIGN.md)を参照してください。

作業領域のモデル要素を一覧で確認するには、`model catalog`を使います。返された`entries[].subject.kind`と`entries[].subject.id`をそのまま`inspect`へ渡せます。`ambiguous=true`の項目は重複宣言があるため、詳細取得を拒否します。

`-namespace`には意味名前空間のIDだけを指定します。存在しないID、およびapplicationやqueryのIDを渡した場合は、空の成功ではなく`NAMESPACE_NOT_FOUND`（終了コード1）を返します。`namespaceMatched`と`namespaceMatchKind`は、要求がどの種類のSubjectに一致したかを示します。宣言はあるが要素を持たない名前空間は、`namespaceMatched=true`かつ`entries`が名前空間自身だけになります。

`model catalog`の応答には`validation`（`ok`・`errorCount`・`warningCount`）が入ります（2026-11-09から）。これは`validate -workspace ... -json`と同じ規則の要約で、error重大度の診断もerror重大度のsource-root findingも無いときに`ok:true`になります。**`model catalog`自体は検証NGのWorkspaceでも`ok:true`・終了コード 0 を返します**。一覧は編集中でも読める投影であり、受理判断ではないためです。応答の`ok`とは別の値なので、catalogだけを読む呼出し元が「このモデルは健全か」を判定するときは`validation.ok`を読んでください。診断のコード・位置・提案は複製せず、`validate -json`で読みます（#3674）。

`model catalog`の`modelRevision`は、管理対象Sourceのファイルpathと生バイトの組をsha256で並べた値です。**読み込んだWorldの内容の版ではありません。** 管理対象Sourceが1件も無いWorkspaceでは、空入力のSHA-256（`sha256:e3b0c442…b855`）になります。標準意味型（`std`）の`entries`は`modelRevision`の計算に入らないため、`entries`が0件でなくてもこの値になり得ます。空かどうかを判別するには、`modelRevision`ではなく`declarationCounts.total`を使ってください。`modelRevision`は「管理対象Sourceを1バイトも読まなかった」ことの証拠であり、モデルの同一性・受理・整合性の証拠ではありません。

`validate -json`・`model catalog`・`inspect`・`trace`・`workspace-files`・`view -format json`・`source-index`・`knowledge-context`・`requirements`・`plugin-lock`・`plugin-resolve`・`plugin-runtime-plan`は、結果に`workspace`を返します。`-world`ファイルだけを入力したコマンドは、manifestを読んでいないため`workspace`を返しません。`name`は`tei.workspace.yaml`の`workspace:`に書かれた値、`formatVersion`はmanifestの版、`manifest.sha256`は読んだmanifestのバイトの照合値です。同じCanonical IDが複数のWorkspaceに別の意味で存在し得るため、`worldFingerprint`は内容の同一性を示しますが、どのWorkspaceかは示しません。両者を組み合わせて、結果の混在を防いでください。`workspace:`を書かないmanifestは読み込まれ、`name`は空になります（パスから名前を作りません）。これらは読んだ版の識別子であり、受理・権限・排他ロックではありません。`source-index`の参照一覧は、0件でも`[]`で返り、`null`にはなりません。

```bash
go run ./cmd/tei model catalog -workspace reference/workspace
go run ./cmd/tei model catalog -workspace reference/workspace -namespace sales
```

一覧と詳細は読取り専用です。元ファイルを変更する権限や、変更を受理した証拠は与えません。一覧の取得と詳細の取得は別の処理なので、同じ版かどうかは返された`worldFingerprint`と`sourceIndexFingerprint`を比較します。

一つの要素がモデル全体のどこにあるかを確かめるには、`model locate`を使います（2026-11-10から）。`model catalog`の`entries[].subject`をそのまま渡すと、次を返します。

- `path`：最上位から親までの経路
- `children`・`siblings`：子と兄弟。`-limit`件（既定50、上限500）で切り、`total`・`omitted`・種類ごとの件数`kindCounts`を添える
- `outgoing`・`incoming`：宣言された参照。Relationの参照先、属性の型、Patternが使うData・Relation・Query・Capabilityを、`role`で区別して返す。参照先のIDが存在しなければ`resolved:false`として残す

`coverage.notDerived`に挙がる関係（業務イベントの流れ、状態遷移、Queryのデータ依存、実装・試験との対応）は、まだ導出していません。`incoming`が空でも「その種類の関係が無い」とは読まないでください。存在しない要素は`SUBJECT_NOT_FOUND`、重複宣言された要素は`SUBJECT_AMBIGUOUS`（どちらも終了コード1）で返し、どちらかの宣言を選んで位置を答えることはしません。`worldFingerprint`・`sourceIndexFingerprint`・`modelRevision`は`model catalog`と同じ読込みから出るため、図表とこの結果が同じ版かを照合できます。MCPの参照adapterでは`tei_model_locate`が同じ結果を返します。設計上の位置づけは[三要素Method §19.8](../plan/design/THREE_ELEMENT_METHOD_DIAGRAM_CONTRACT_2026-10.md)を参照してください。

```bash
go run ./cmd/tei model locate -workspace reference/workspace -subject-kind semantic-data -subject-id sales.order
```

`model locate`の結果には、その要素と配下全体の要注意の件数`attention`も入ります（`model overview`と同じ数え方です）。

モデル全体の目次と、どこに問題があるかを先に見るには、`model overview`を使います（2026-11-10から）。最上位の要素（意味名前空間、Application、Query）を区画として並べ、区画ごとに次を返します。

- `descendants`・`kindCounts`：配下の要素数と種類別の件数
- `attention`：配下のどこかにある要注意の件数。validationのerror・warning、参照先が存在しない宣言参照（`role`別）、重複宣言、Source参照を辿れないWorkspace宣言要素を、別々に数えます。`needsAttention`はどれかが0でないときに`true`です

区画の一覧は`-limit`件（既定100、上限1000）で切りますが、`totals`は切った区画も含めた合計です。どの要素にも属さないfinding（対象の無い診断、catalogに無い種類の診断、Source rootのfinding）は`unattributed`に残します。`totals`と`unattributed`のerror・warningを足すと、`model catalog`の`validation`の件数と一致します。`attention`は案内のための件数で、検証の合否ではありません。合否は`validation.ok`か`validate -json`で読みます。カスタム・Extension Slot、業務イベント・状態遷移の欠落、実装との対応、利用者ごとの開示判定はまだ集約していません（`coverage.notDerived`）。MCPの参照adapterでは`tei_model_overview`が同じ結果を返します。

```bash
go run ./cmd/tei model overview -workspace reference/workspace
```

正規化された業務モデルから特定の要素だけを確認するには、`inspect`を使います。現在対応するのは`semantic`、`semantic-type`、`semantic-data`、`semantic-attribute`、`semantic-relation`、`semantic-candidate-key`、`semantic-functional-dependency`、`semantic-capability`、`semantic-validation`、`application`、`application-pattern`、`query`の12種類です。対象の種類と正本の識別子を、完全一致で指定します。

```bash
go run ./cmd/tei inspect \
  -workspace reference/workspace \
  -subject-kind semantic-data \
  -subject-id sales.order
```

`inspect`は`tei.cli.inspect/v0`形式のJSONを返します。成功時の`result.canonical`は既存の正規化した全体モデルから選択した要素そのもの、`result.sources`は既存入力元索引の参照と成果物の照合値です。`worldFingerprint`は読取り時点の正本となる全体モデルの内容を識別するためのハッシュであり、競合更新を防ぐ版を示す値や変更の受理証跡ではありません。`sourceStatus=not-indexed`は入力元索引に参照がないことを意味し、正本となる定義の不存在とは区別します。

業務モデルの要素が何を参照し、何から参照されているかを調べるには、読み取り専用の`trace`を使います。`inspect`と同じ種類・識別子（例：`semantic-relation: sales.order.customer`）を指定して、意味の情報と参照関係を確認できます。現在扱えるのは、既存の派生索引で表現できる、宣言済みの直接参照だけです。

```bash
go run ./cmd/tei trace \
  -workspace reference/workspace \
  -scope reference-observation \
  -subject-kind semantic-relation \
  -subject-id sales.order.customer \
  -direction both -mode summary

go run ./cmd/tei trace \
  -workspace reference/workspace \
  -scope reference-observation \
  -subject-kind semantic-relation \
  -subject-id sales.order.customer \
  -direction both -mode full
```

`-scope`には、呼出し元が**関係図を作る際の文脈を識別する値**を指定します。作業領域名や業務上の所属名前空間、受理の状態、認可の範囲を自動的に証明するものではありません。`-mode summary`は件数・未解決件数を取得し、`-mode full`では宣言された直接の参照を省略せず返します。大規模な結果を分割する場合のみ`-limit 100`のように指定し、`result.trace.nextOffset`と**`result.snapshotFingerprint`（作業領域全体の継続用トークン）**を次回の`-offset`、`-expected-fingerprint`へ渡してください。`result.trace.graphFingerprint`（2026-11-09より前の名前は`result.trace.snapshotFingerprint`。旧名も同じ値で残る）は関係図、関係規則、選択条件の照合値であり、コマンドラインで結果の取得を続けるためには使いません。継続に使う値は`nextOffset`の隣の`result.trace.continuation.fingerprint`（`field`は`result.snapshotFingerprint`）にも書かれているので、そこから取れます。入力した関係図、対象や方向、正規化した全体モデル、入力元索引（元ファイルの内容や場所を含む）が変化している場合、`STALE_SNAPSHOT`で継続を拒否します。

結果に含まれる`worldFingerprint`と`sourceIndexFingerprint`は、`inspect`が返す同名の項目と比較できます。意味の正本を識別する前者が同じでも、入力元ファイルの移動や注記の変更があれば、後者は変わります。結果の`source`は選択要素の元定義、`relatedSources`は全件出力のそのページに登場した参照先・参照元の情報源の候補を示します。`projectionUnresolved`は、派生索引を作った時点で解決できなかった参照を示します。関係の`resolved`は宣言の解決に成功したことのみを意味し、実行の成功・人間による承認・書込み許可を意味しません。入力元索引と原ファイルの間で生じる同時更新は、一つの処理として固定できていません。

成功時の終了コードは`0`、不存在・重複宣言は`1`、引数不正・未対応の種類・作業領域読込み失敗などは`2`です。`inspect`が読める種類のうち、`semantic-candidate-key`と`semantic-functional-dependency`は派生索引の関係図に投影されないため、`trace`は`UNSUPPORTED_SUBJECT_KIND`（終了コード`2`）を返し、`SUBJECT_NOT_FOUND`（終了コード`1`）とは区別します。後者は種類は投影されるが指定した識別子の宣言が無いことを示します。失敗時も可能な範囲で`ok=false`と`error.code`を持つJSONを出力します。現時点で対応するのは宣言済みの直接参照であり、複数段の依存探索、すべての種類の要素取得、受理済み業務フローとの完全な横断参照には対応しません。

## 1. テスト

```bash
go test ./...
```

すべてのパッケージのテストが成功したか確認します。

## 2. 参照用ワークスペースを検証する

```bash
go run ./cmd/tei validate -workspace reference/workspace
```

参照用データには過去の配布版を意図的に残しています。そのため、過去の契約に関する警告が表示されます。最後に`ok`と表示されれば、検証は成功です。

入力元索引で位置を解決できる診断結果には、`path:line:column`を伴って表示されます。

## 3. 正規化した全体モデルを見る

```bash
go run ./cmd/tei world \
  -workspace reference/workspace > /tmp/tei-world.json
```

この結果には、意味、アプリケーション、データベース、環境、配布版、拡張機能などの正規化したモデルが含まれます。

`reference/world.json`は、同じ参照用作業領域から再生成できる基準データです。自動検査でも、生成結果が一致することを確認しています。

正本となる全体モデルには、入力ファイルの場所を含めません。元ファイルの場所は、次に説明する入力元索引で確認します。

## 4. 入力元索引で定義元を確認する

入力元索引の全体を見るには、次のコマンドを実行します。

```bash
go run ./cmd/tei source-index \
  -workspace reference/workspace
```

特定の意味上の対象だけを確認する場合は、種類と識別子を指定します。

```bash
go run ./cmd/tei source-index \
  -workspace reference/workspace \
  -subject-kind query \
  -subject-id order_entry.recent_orders
```

参照用問い合わせでは、同じ`query:order_entry.recent_orders`に対して次の二つが表示されます。

- `definition`: `applications/order-entry/queries/recent_orders.tei`
- `authoring`: `applications/order-entry/queries/recent_orders.sql`

`.tei`には問い合わせの宣言を、`.sql`には正本となるSQL本文を記述します。成果物には内容の照合値も含まれます。

`recent_orders.tei`自体はSQL本文を埋め込まず、次の形で元形式のソースを参照しています。

```tei
query order_entry.recent_orders {
    datasource sales
    sql postgres file "recent_orders.sql"
}
```

SQLファイルを移動し、参照先だけを更新した場合でも、正本となる問い合わせの識別子と正規化した全体モデルは変わらず、入力元索引の場所だけが変わることを試験しています。

この境界の詳細は [準拠確認 — M0.23 入力元から正本への追跡可能性](conformance/source-traceability-m0.23.md) を参照してください。

## 5. 拡張機能の固定情報と実行計画

```bash
go run ./cmd/tei plugin-lock \
  -workspace reference/workspace -environment production > /tmp/tei-plugin-lock-result.json

go run ./cmd/tei plugin-runtime-plan \
  -workspace reference/workspace \
  -environment production \
  -plugin-lock /tmp/tei-plugin-lock-result.json
```

この結果から、環境が利用するプラグインとその版、提供される機能、必要な権限、資源の対応付け、隔離に関する要求を確認できます。

`plugin-lock`は結果をJSON envelopeで返します。`-plugin-lock`は、その envelope をそのまま受け付けます（内側の`result`だけを渡しても同じ結果になります）。`-environment production`を指定したLockはproductionだけを対象とし、別の環境には使えません。この指定を省略した従来の操作は全環境のLockを作ります。どちらの場合もWorkspace全体の検証は省略されず、Lockの作成・選択は実行権限や成果物バイト列の検証ではありません。

リポジトリに保存した参照用プラグインの実装固定情報についても、作業領域から再生成して一致することを自動検査で確認しています。

## 6. 配備の参照経路

参照用作業領域と観測試験用データを使い、保存済みの配備実行記録を再生成できます。

```bash
go run ./cmd/tei deployment-start \
  -workspace reference/workspace \
  -solution order_rollout \
  -version 2026.8.27 \
  -deployment reference/observed/production.deployment-state.json \
  -run-id order-rollout-prod-m0.21 \
  -plugin-lock reference/workspace/.tei/plugin-lock.json
```

保存済みの `reference/order_rollout.durable-run.m0.21.start.json` はこの入力から再生成可能な参照試験用データです。詳しくは [reference/README.md](../reference/README.md) を参照してください。

## 7. 外部プロセスとして動くプラグインの参照例

M0.24では、参照用のプラグインをOS上の子プロセスとして起動できます。最初に補助プログラムを構築します。

```bash
go build -o /tmp/tei-reference-process-plugin ./tools/reference-process-plugin
```

次に、既存の`DeploymentExecutor`接続部から、標準入出力で通信するプロセスへ命令を渡します。

```bash
go run ./examples/process-plugin \
  -plugin /tmp/tei-reference-process-plugin
```

出力の`outcome`は`accepted`、拡張機能の呼び出しの`boundary`は`process`になります。参照用の拡張機能には`--tei-reference-marker=argv value preserved`という空白を含む一つの引数要素も渡され、子プロセス側がその境界を確認してから成功応答を返します。`accepted`は実行役の受け付け通知であり、外部状態が整ったことの証明ではありません。

この例では、ローカルのプロセスを一度だけ起動します。プロセスIDを実行状態の識別情報として永続化していないため、実行環境の再起動後や別の環境から再接続できることを実証したものではありません。実プロセス境界は [M0.24実プロセス 準拠確認](conformance/process-runtime-m0.24.md)、実行ファイルと引数の境界は [M0.25 準拠確認](conformance/process-command-argv-m0.25.md) を参照してください。

## 8. PostgreSQLで期待状態と実際の状態を比較する

PostgreSQLを用意して、接続文字列を実行環境の環境変数として渡すと、参照用の連携試験を実行できます。

```bash
TEI_TEST_POSTGRES_DSN='postgres://user:password@127.0.0.1:5432/database?sslmode=disable' \
  go run ./tools/reference-postgres-integration
```

この試験では専用の試験用構造を準備し、データ保存の対応付けから期待状態を生成してから、実際のデータベースの定義情報を読み取ります。基準状態では差分がない状態を要求し、その後実際の状態側だけに列を追加してずれ検出を確認します。試験用の構造定義は連携試験用の仕組みであり、TEIの自動的な構造変更機能ではありません。

詳細は [M0.26 準拠確認](conformance/postgres-actual-schema-m0.26.md) を参照してください。

## 9. 実行中のプロセスを別の実行環境から引き継ぐ例

```bash
go run ./examples/runtime-reattach
```

この例では、実行環境Aが管理していた処理を、実行環境Bが新しい識別値を使って引き継ぎます。Aが使っていた古い識別値による操作は、実行環境側で拒否されます。

重要なのは、実行状態の識別情報が`missing`だった場合に自動再実行しないことです。外部副作用の有無を確定できないため、復旧処理へ送ります。

## 10. コードを読む順番

実装を初めて読む場合は、次の順にたどると全体の関係を把握しやすくなります。

1. `reference/workspace/tei.workspace.yaml`
2. `reference/workspace/semantic/`
3. `reference/workspace/applications/`
4. `reference/workspace/databases/`
5. `reference/workspace/environments/`
6. `reference/workspace/releases/`
7. `internal/ir/`, `internal/workspace/`, `internal/sourceindex/`, `internal/validation/`
8. `internal/database/`, `internal/query/`
9. `internal/deployment/`
10. `internal/plugin/runtime/`, `internal/resource/`

ソースと意味の境界を読む場合は、[準拠確認 — M0.23 入力元から正本への追跡可能性](conformance/source-traceability-m0.23.md) がコードとテストの対応表として使えます。

拡張機能と実行環境の安全境界を読む場合は、[準拠確認 — M0.22.1](conformance/plugin-runtime-m0.22.1.md) を参照してください。

## 注意

現段階の参照用接続部は、設計上の契約を検証するためのものです。本番で利用する場合は、Vault、Kubernetes、etcdなど、実際の運用環境に合わせた接続部が必要です。
