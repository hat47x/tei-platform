# reference-generator-selection（#2016）

受理済みWorkspaceの属性・型に結び付けた一つの参照契約について、複数の生成器候補を同じ期待値fixtureで比較し、実際の出力と出所をreceiptへ記録する参照です。ADR-0034の「生成器を版付き成果物として扱い、由来ではなく適合で選ぶ」の最小PoCです。

```text
go run ./tools/reference-generator-selection/cmd/tei-generator-select \
  -workspace reference/workspace \
  -contract tools/reference-generator-selection/testdata/line-amount/contract.json \
  -fixtures tools/reference-generator-selection/testdata/line-amount/fixtures.json \
  -candidates tools/reference-generator-selection/testdata/candidates
```

registryを使うと、登録済みで有効な版だけを選抜できます。

```text
go run ./tools/reference-generator-selection/cmd/tei-generator-registry \
  validate -registry tools/reference-generator-selection/testdata/registry.json \
  -contract tools/reference-generator-selection/testdata/line-amount/contract.json

go run ./tools/reference-generator-selection/cmd/tei-generator-select \
  -workspace reference/workspace \
  -contract tools/reference-generator-selection/testdata/line-amount/contract.json \
  -fixtures tools/reference-generator-selection/testdata/line-amount/fixtures.json \
  -candidates tools/reference-generator-selection/testdata/candidates \
  -registry tools/reference-generator-selection/testdata/registry.json
```

候補sourceは、契約から固定strategyで生成できます。

```text
go run ./tools/reference-generator-selection/cmd/tei-generator-build \
  -contract tools/reference-generator-selection/testdata/line-amount/contract.json \
  -out /tmp/line-amount-candidates -all
```

版を明示して一つのstrategyだけを生成できます。`-version`は候補manifestとgenerator provenanceへ記録され、source自体は変えません。既存の生成物を置換する場合は、新しい版の候補を別directoryへ生成し、固定fixtureで再選抜します。

```text
go run ./tools/reference-generator-selection/cmd/tei-generator-build \
  -contract tools/reference-generator-selection/testdata/line-amount/contract.json \
  -strategy template -version 2 -out /tmp/line-amount-template-v2
```

同じ契約からPythonの参照候補も生成できます。

```text
go run ./tools/reference-generator-selection/cmd/tei-generator-build \
  -contract tools/reference-generator-selection/testdata/line-amount/contract.json \
  -strategy python-template -out /tmp/line-amount-python
```

JavaScriptの参照候補も生成できます。

```text
go run ./tools/reference-generator-selection/cmd/tei-generator-build \
  -contract tools/reference-generator-selection/testdata/line-amount/contract.json \
  -strategy javascript-template -out /tmp/line-amount-javascript
```

Javaの参照候補も生成できます。

```text
go run ./tools/reference-generator-selection/cmd/tei-generator-build \
  -contract tools/reference-generator-selection/testdata/line-amount/contract.json \
  -strategy java-template -out /tmp/line-amount-java
```

外部モデルが出したsourceは、promptとcontextとともに記録できます。

```text
go run ./tools/reference-generator-selection/cmd/tei-generator-record \
  -id model-candidate -provider example-provider -model example-model-v1 \
  -contract tools/reference-generator-selection/testdata/line-amount/contract.json \
  -source /tmp/model-main.go -prompt /tmp/prompt.txt -context /tmp/context.json \
  -out /tmp/recorded-candidate
```

`-language python`を指定すると、`.py`のsourceを`main.py`として記録できます。

```text
go run ./tools/reference-generator-selection/cmd/tei-generator-record \
  -id model-python -provider example-provider -model example-model-v1 \
  -language python \
  -contract tools/reference-generator-selection/testdata/line-amount/contract.json \
  -source /tmp/model-main.py -prompt /tmp/prompt.txt -context /tmp/context.json \
  -out /tmp/recorded-python
```

`tei-generator-record`はモデルを呼び出しません。生成済みsourceをbyte単位で複製し、prompt・contextを同じdirectoryへ保存して、selectionがdigestを独立に確認できるようにします。`language`とsource拡張子が一致しない場合は拒否します。

外部の生成コマンドを呼び出し、その出力をそのまま記録することもできます。

```text
go run ./tools/reference-generator-selection/cmd/tei-generator-model \
  -id model-candidate -provider example-provider -model example-model-v1 \
  -contract tools/reference-generator-selection/testdata/line-amount/contract.json \
  -prompt /tmp/prompt.txt -context /tmp/context.json \
  -out /tmp/recorded-candidate \
  -command 'your-model-cli ... "$TEI_PROMPT" "$TEI_CONTEXT" > "$TEI_SOURCE"'
```

`tei-generator-model`は、`TEI_CONTRACT`、`TEI_LANGUAGE`、`TEI_PROMPT`、`TEI_CONTEXT`、`TEI_SOURCE`、`TEI_TMPDIR`、`TEI_PROVIDER`、`TEI_MODEL`を環境変数で渡し、コマンドが`TEI_SOURCE`へ書いたsourceを記録します。`-language python`では`TEI_SOURCE`が`.py`になり、記録先も`main.py`です。`TEI_CONTRACT`は候補の契約digestを求める入力であり、コマンドが別契約向けのsourceを出しても、選抜は現在の契約との不一致として拒否します。特定ベンダーのAPI・認証・課金・モデル版管理は埋め込みません。コマンドは信頼済みのものだけを指定し、秘密情報を出力しないでください。

外部の**生成器**が候補sourceを生成する場合は、生成器を深さ1で実行して記録できます。

```text
go run ./tools/reference-generator-selection/cmd/tei-generator-meta \
  -id meta-template -provider recorded-fixture -model meta-fixture-v0 \
  -contract tools/reference-generator-selection/testdata/line-amount/contract.json \
  -generator tools/reference-generator-selection/testdata/meta/generator.go \
  -prompt /tmp/prompt.txt -context /tmp/context.json \
  -out /tmp/meta-candidate
```

`tei-generator-meta`は、生成器をビルドし、`-contract`を渡して1回だけ実行し、その出力を候補sourceとして記録します。生成器のsourceとdigestは`lineage`へ残します。selectionは生成器を再ビルド・再実行し、候補sourceと一致することを確認します。深さ2以上のメタ生成は拒否します。

`passed=true`は、少なくとも1つの候補が固定fixtureをすべて満たしたことだけを示します。意味の受理、生成器の採用、権限の付与、実Providerや本番環境の証明ではありません。

## 対象契約

`testdata/line-amount/contract.json`は、`reference/workspace`で受理済みの次の属性・型を使います。

- `sales.order_line.quantity`（`std.decimal`）
- `sales.order_line.unit_price`（`std.money`）
- 出力 `std.money`
- DECIMAL(8,2)、half-up

数量 × 単価という複合規則そのものは、まだCanonicalな受理済み規則ではありません。契約の`status`は`reference-local`で、受理済みの入力・型に結び付けた参照用の契約として扱います。Workspaceの属性・型が変われば、選抜はfail closedします。

## 候補

`testdata/candidates/`に3つの候補を置いています。

| 候補 | 由来 | 実装 | 期待 |
| --- | --- | --- | --- |
| `template` | template（生成済みsource） | 正確な有理数で計算し、最後にhalf-up | 全fixture合格 |
| `alternative` | handwritten | 10進を整数へ変換し、最後に一度だけ丸める | 全fixture合格 |
| `dangerous` | generated-ai | float64、欠落の0化、通貨・単位無視、精度検査なし | 負例で不合格 |

`testdata/candidates/`の4候補はGoです（`long-running-template`を含む）。Python候補は`-strategy python-template`、JavaScript候補は`-strategy javascript-template`、Java候補は`-strategy java-template`で生成し、同じ`-candidates`directoryへ置いて選抜できます。候補は`candidate.json`で`id`、`language`、`origin`、source path、source digestを宣言します。選抜はsource digestを実byteで再計算し、一致しなければビルド前に拒否します。ビルドに成功したことや、候補の自己申告は合格の根拠にしません。

## 生成AI候補のprovenance

`origin: generated-ai`の候補は、`generation`を必須とします。

- `provider`、`model`
- `contractDigest`（候補を作ったときの契約）
- `promptDigest`、`contextDigest`
- `outputDigest`
- `promptPath`、`contextPath`（任意。記録したprompt・contextの相対path）

選抜は、`contractDigest`が現在の契約と一致すること、`outputDigest`が候補sourceのdigestと一致することを確認します。`generation`のない`generated-ai`候補は拒否します。契約が変わった候補を、別契約の証拠として再利用できません。`promptPath`・`contextPath`がある場合は、そのfileを読み、digestが記録と一致することを確認します。pathは候補directory内の相対pathに限り、シンボリックリンクを拒否します。

`tei-generator-record`は、外部モデルが出したsourceとprompt・contextを一つの候補directoryへ記録します。モデル呼出し自体は行いません。`language`は`go`・`python`・`javascript`・`java`で、source拡張子と記録先（`main.go`/`main.py`/`main.js`/`Main.java`）を一致させます。

`testdata/candidates/dangerous`の`generation`は`recorded-fixture`で、実モデルを呼び出した記録ではありません。生成AIのprovenance境界を試験するための固定fixtureです。実モデル実行では、同じ項目へ実際のprovider・model・prompt・contextのdigestを入れる必要があります。

## 生成器sourceの生成

`tei-generator-build`は、契約JSONと固定strategy（`template`、`alternative`、`dangerous`、`python-template`、`javascript-template`、`java-template`、`long-running-template`、`python-long-running-template`、`javascript-long-running-template`）から、候補sourceを決定的に生成します。

- Goのsourceは`go/format`で整形してから書き出します。Python・JavaScript・Javaのsourceは整形せず、生成したbyte列をそのまま使います。
- `candidate.json`には`language`、`source`（`main.go`/`main.py`/`main.js`）、`generator`を記録します。
  - 生成器のid・version・strategy・language
  - 契約のdigest
  - テンプレートのdigest
- 選抜は、`generator.contractDigest`が現在の契約のdigestと一致するかを確認します。さらに、契約・strategy・versionからsourceを再生成し、候補のsourceとbyte一致することを確認します。別の契約・別のstrategy・別の言語から生成した候補を、現在の契約の証拠として使えません。
- 生成先は未存在のdirectoryに限ります。
- 同じ契約・strategyからは、同じsourceと`sourceDigest`が得られます。

`template`と`dangerous`の生成器は同じ契約パラメータ（通貨、scale、整数桁）を使いますが、実装方式と安全性は異なります。生成器の由来はreceiptへ残しますが、生成AIのモデル・prompt・文脈はまだ記録しません。

## 別言語候補の実行

`python-template`はPython、`javascript-template`はJavaScript、`java-template`はJavaの参照候補を生成します。候補manifestの`language`は`python`/`javascript`/`java`、sourceは`main.py`/`main.js`/`Main.java`です。

- Go候補は`go build`で実行ファイルを作り、`binaryDigest`を記録します。
- Python候補は`python3`または`python`、JavaScript候補は`node`でsourceを実行します。どちらも`binaryDigest`は空です。
- Java候補は`javac -d <classes> Main.java`でcompileし、`java -cp <classes> Main`で実行します。`binaryDigest`は生成された`Main.class`のdigestです。
- receiptの`runtime`へ、言語、起動方法、処理系のversion、利用可否を記録します。Javaは`javac`と`java`の両方が必要です。
- 処理系が見つからない場合は`runtime unavailable`として拒否し、PASSや空の結果へ潰しません。
- source拡張子が`language`と一致しない候補、未知の`language`を拒否します。
- 同じfixtureでGo候補とPython候補、Go候補とJavaScript候補、Go候補とJava候補の出力を比較する試験を持つ。これは同じ参照契約の範囲であり、すべての言語・Runtimeの同等性を証明しません。
- Javaのclassファイルの再現性は`not-tested`です。JVMの版・依存解決・実Targetでの起動は別の境界です。

これらの参照は、別言語の候補を同じ選抜経路へ載せられることの確認であり、Python/JavaScript/Java製品実装の採用や、実Targetでの処理系固定を意味しません。

## long-running候補

`long-running-template`は、標準入力のnewline-delimited JSONを繰り返し読み、1 requestごとに1 JSONを標準出力へ返すGo候補を生成します。単発実行では1 requestを処理して終了し、Target Profileのlong-running probeでは同じprocessへ複数requestを送ります。

- 選抜のfixture実行は、これまでどおりcaseごとにprocessを起動します。
- `tei-target-profile`のlong-running profileは、この候補を1 processで起動し、全requestのresponseをfixtureと比較します。
- processが途中で終了した場合、response数が足りない場合、fixtureと異なる場合は失敗します。
- `python-long-running-template`と`javascript-long-running-template`は、同じ計算・拒否をPython/JavaScriptのnewline-delimited JSON処理として生成します。既存の単発テンプレートは変更せず、別のtemplate digestで識別します。
- Target ProfileのprobeはGo・Python・JavaScript候補を対象とします。Javaのlong-running probeは未対応です。処理系がなければ不合格とし、Python/JavaScriptにはGoのメモリ上限が効かないことをreceiptへ記録します。

## Go候補のbuild evidence

Go候補のreceiptには`buildEvidence`を記録します。

- `toolchain`: `go version`の結果
- `goos`、`goarch`、`cgoEnabled`、`goflags`: build時の環境
- `sourceDigest`: buildしたsource
- `binaryDigest`: 得られた実行ファイル
- `reproducedDigest`: `-trimpath -buildvcs=false`でもう一度buildした実行ファイル
- `buildInputDigest`: build command・source・toolchain・GOOS/GOARCH・CGO・GOFLAGSを結合したdigest
- `reproducibility: same-digest-twice`

同じ候補を`go build -trimpath -buildvcs=false`で2回buildし、`binaryDigest`と`reproducedDigest`が一致する場合だけ合格します。一致しなければ`reproducibility: mismatch`として候補を拒否します。

これは同じ環境・同じsourceでの反復buildの一致であり、別machine・別toolchain・clean cacheでの再現性や、artifact store・署名を証明しません。`buildInputDigest`は、どのsourceとbuild環境から候補を得たかをreceipt内で結び付けるための参照値です。`binaryDigest`はGoの版・OS・build設定に依存します。

## メタ生成（深さ1）

生成器そのものを外部で生成し、その生成器が候補sourceを出す経路を、深さ1に限って扱います。

- 生成器は`-contract FILE`を受け取り、Go sourceを標準出力へ書きます。
- TEIは生成器をビルドし、1回だけ実行します。
- 候補manifestの`lineage`へ、深さ、生成器sourceの相対path・digest・originを記録します。
- 選抜は生成器を再ビルド・再実行し、候補sourceとのbyte一致を確認します。
- `depth`が1以外の場合は拒否します。生成器が別の生成器を生成する再帰は、このPoCの対象外です。
- `testdata/meta/generator.go`は固定fixtureで、実モデルが生成した生成器ではありません。

## 生成器registryとライフサイクル

`testdata/registry.json`（v0）は契約ごとに生成器のid・version・状態・source digestを記録します。`testdata/registry-lifecycle.json`（v1）は、置換と失効の理由まで記録します。

- `status: active`のentryだけを、そのidの現在の版として解決します。
- `status: superseded`は後継の版に置き換えられた版です。`supersedes`で後継元を示し、ちょうど1つの後継を持たなければなりません。
- `status: revoked`は使用してはならない版です。v1では`reason`を必須にします。
- 同じid・versionの重複、契約digestの不一致、source digestの不一致、置換先の欠落・二重の後継・循環を拒否します。
- `tei-generator-select -registry`を使うと、registryにない候補、失効した候補、置換済みの候補、versionやsource digestが一致しない候補を拒否します。

v1のregistryは意味の受理・権限・本番配備の許可ではありません。版の系列と現在のactive版をプロジェクト内で宣言するだけです。

### 候補への影響確認

`tei-generator-reconcile`は、registryと候補directoryを比較し、候補が現在のactive版かを報告します。

```text
go run ./tools/reference-generator-selection/cmd/tei-generator-reconcile \
  -registry tools/reference-generator-selection/testdata/registry-lifecycle.json \
  -contract tools/reference-generator-selection/testdata/line-amount/contract.json \
  -candidates tools/reference-generator-selection/testdata/candidates
```

- `current`: active版とversion・source digestが一致する。
- `stale-version` / `stale-source` / `stale-contract`: 再生成または再選抜が必要。
- `superseded` / `revoked`: その版を使ってはならない。
- `unregistered`: registryにない候補。
- 登録済みのidに現在の候補がなければ、`generators`で不足を示す。

reconcileは候補をビルド・実行・選抜しません。再生成した候補も、固定fixtureでの選抜をやり直す必要があります。自動再生成・自動選抜・失効の承認は行いません。

## receipt

`tei-generator-select`は次を返します。

- 契約のid・version・status
- WorkspaceのSource Indexから求めた`workspaceRevision`
- fixtureのdigestと、契約・fixtureを合わせた`suiteVersion`
- 候補ごとの`language`、`sourceDigest`、`binaryDigest`、`runtime`、`buildEvidence`、`generator`・`generation`（あれば）、各caseの`got`/`want`、`passed`、`rejectReason`
- 各caseの`execution`（`exitCode`、stdout/stderrのdigest、`timedOut`）
- 合格した候補の`qualified`と、不合格の`rejected`
- `grantsRuntimeAuthority: false`、`provesRealProvider: false`

同じWorkspace・契約・fixture・候補sourceからは、同じ`sourceDigest`と結果が得られます。`buildEvidence`は同じ環境での反復build一致を記録しますが、別環境での再現性は未検証です。`execution`はstdout/stderrの内容そのものではなくdigestとexit stateだけを残し、時刻や実行者のidentityは含めません。

## 守っている境界

- 受理済みWorkspaceの属性・型を、契約の入力として実際に解決する。存在しない属性や型の不一致を拒否する。
- 期待値fixtureは候補より先に固定し、候補に合わせて書き換えない。
- 候補のsource digestを実byteで確認する。digest不一致の候補をビルドしない。
- 候補の出力を独立に比較する。候補が「合格」と自己申告しても採用しない。
- 危険な候補を、欠落・通貨・丸め・精度の負例で落とす。
- 生成AI候補のgeneration provenanceを、現在の契約digestへ束縛する。
- 候補の言語とsource拡張子を一致させ、処理系が無い場合は成功扱いにしない。
- 生成器の選択を、意味の受理・権限・本番配備へ接続しない。

## 未実装

- 特定ベンダーのAPI呼出し・認証・課金・モデル版管理。外部コマンドへ委ねており、TEI内には実装していません。
- `promptPath`・`contextPath`を省略した候補では、digestの形式だけを確認し、内容との対応は検証しません。
- 深さ2以上のメタ生成、失効に伴う自動再生成・自動選抜・失効の承認。
- Python/JavaScript/Java sourceを出力するメタ生成器のlineage検証。
- long-runningのJava候補。
- Java以外のJVM言語、.NET/WASM等と、実Targetでの処理系固定・依存解決。
- 実Provider・実Targetでの比較。
- 候補のartifact store、別machine・別toolchain・clean cacheでのbuild再現性（#943 / #1370 / #1104）。
