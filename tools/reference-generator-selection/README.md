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

外部モデルが出したsourceは、promptとcontextとともに記録できます。

```text
go run ./tools/reference-generator-selection/cmd/tei-generator-record \
  -id model-candidate -provider example-provider -model example-model-v1 \
  -source /tmp/model-main.go -prompt /tmp/prompt.txt -context /tmp/context.json \
  -out /tmp/recorded-candidate
```

`tei-generator-record`はモデルを呼び出しません。生成済みsourceをbyte単位で複製し、prompt・contextを同じdirectoryへ保存して、selectionがdigestを独立に確認できるようにします。

外部の生成コマンドを呼び出し、その出力をそのまま記録することもできます。

```text
go run ./tools/reference-generator-selection/cmd/tei-generator-model \
  -id model-candidate -provider example-provider -model example-model-v1 \
  -prompt /tmp/prompt.txt -context /tmp/context.json \
  -out /tmp/recorded-candidate \
  -command 'your-model-cli ... "$TEI_PROMPT" "$TEI_CONTEXT" > "$TEI_SOURCE"'
```

`tei-generator-model`は、`TEI_PROMPT`、`TEI_CONTEXT`、`TEI_SOURCE`、`TEI_TMPDIR`、`TEI_PROVIDER`、`TEI_MODEL`を環境変数で渡し、コマンドが`TEI_SOURCE`へ書いたGo sourceを記録します。特定ベンダーのAPI・認証・課金・モデル版管理は埋め込みません。コマンドは信頼済みのものだけを指定し、秘密情報を出力しないでください。

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

候補は`candidate.json`で`id`、`origin`、source path、source digestを宣言します。選抜はsource digestを実byteで再計算し、一致しなければビルド前に拒否します。ビルドに成功したことや、候補の自己申告は合格の根拠にしません。

## 生成AI候補のprovenance

`origin: generated-ai`の候補は、`generation`を必須とします。

- `provider`、`model`
- `promptDigest`、`contextDigest`
- `outputDigest`
- `promptPath`、`contextPath`（任意。記録したprompt・contextの相対path）

選抜は、`outputDigest`が候補sourceのdigestと一致することを確認します。`generation`のない`generated-ai`候補は拒否します。`promptPath`・`contextPath`がある場合は、そのfileを読み、digestが記録と一致することを確認します。pathは候補directory内の相対pathに限り、シンボリックリンクを拒否します。

`tei-generator-record`は、外部モデルが出したsourceとprompt・contextを一つの候補directoryへ記録します。モデル呼出し自体は行いません。

`testdata/candidates/dangerous`の`generation`は`recorded-fixture`で、実モデルを呼び出した記録ではありません。生成AIのprovenance境界を試験するための固定fixtureです。実モデル実行では、同じ項目へ実際のprovider・model・prompt・contextのdigestを入れる必要があります。

## 生成器sourceの生成

`tei-generator-build`は、契約JSONと固定strategy（`template`、`alternative`、`dangerous`）から、Goの候補sourceを決定的に生成します。

- 生成したsourceは`go/format`で整形してから書き出します。
- `candidate.json`には`generator`を記録します。
  - 生成器のid・version・strategy
  - 契約のdigest
  - テンプレートのdigest
- 選抜は、`generator.contractDigest`が現在の契約のdigestと一致するかを確認します。さらに、契約とstrategyからsourceを再生成し、候補のsourceとbyte一致することを確認します。別の契約・別のstrategyから生成した候補を、現在の契約の証拠として使えません。
- 生成先は未存在のdirectoryに限ります。
- 同じ契約・strategyからは、同じsourceと`sourceDigest`が得られます。

`template`と`dangerous`の生成器は同じ契約パラメータ（通貨、scale、整数桁）を使いますが、実装方式と安全性は異なります。生成器の由来はreceiptへ残しますが、生成AIのモデル・prompt・文脈はまだ記録しません。

## 生成器registryと失効

`testdata/registry.json`は、契約ごとに生成器のid・version・状態・source digestを記録します。

- `status: active`のentryだけを、そのidの現在の版として解決します。
- `status: revoked`のentryは選抜で拒否します。
- 同じid・versionの重複、契約digestの不一致、source digestの不一致を拒否します。
- `tei-generator-select -registry`を使うと、registryにない候補、失効した候補、versionやsource digestが一致しない候補を拒否します。

registryは選抜の制約であり、意味の受理・権限・本番配備の許可ではありません。失効履歴の自動追跡や、失効に伴う生成物の再生成は未実装です。

## receipt

`tei-generator-select`は次を返します。

- 契約のid・version・status
- WorkspaceのSource Indexから求めた`workspaceRevision`
- fixtureのdigestと、契約・fixtureを合わせた`suiteVersion`
- 候補ごとの`sourceDigest`、`binaryDigest`、`generator`・`generation`（あれば）、各caseの`got`/`want`、`passed`、`rejectReason`
- 合格した候補の`qualified`と、不合格の`rejected`
- `grantsRuntimeAuthority: false`、`provesRealProvider: false`

同じWorkspace・契約・fixture・候補sourceからは、同じ`sourceDigest`と結果が得られます。`binaryDigest`はGoの版やビルド環境に依存します。時刻や実行者のidentityは含めません。

## 守っている境界

- 受理済みWorkspaceの属性・型を、契約の入力として実際に解決する。存在しない属性や型の不一致を拒否する。
- 期待値fixtureは候補より先に固定し、候補に合わせて書き換えない。
- 候補のsource digestを実byteで確認する。digest不一致の候補をビルドしない。
- 候補の出力を独立に比較する。候補が「合格」と自己申告しても採用しない。
- 危険な候補を、欠落・通貨・丸め・精度の負例で落とす。
- 生成器の選択を、意味の受理・権限・本番配備へ接続しない。

## 未実装

- 特定ベンダーのAPI呼出し・認証・課金・モデル版管理。外部コマンドへ委ねており、TEI内には実装していません。
- `promptPath`・`contextPath`を省略した候補では、digestの形式だけを確認し、内容との対応は検証しません。
- 生成器を生成する生成器（メタ生成）の停止条件、失効履歴・再生成の自動追跡。
- 実Provider・実Target・別言語（.NET/JVM/WASM）での比較。
- 候補の署名、build provenance（#943 / #1370）。
