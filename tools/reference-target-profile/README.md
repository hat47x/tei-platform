# reference-target-profile（#2019）

同じ受理済み契約・fixture・生成器候補を、複数のTarget Profileで実行し、profileの宣言と実行結果を一つのreceiptへ記録する参照です。組込み/OS寄りのone-shot profileと、サーバ向けのlong-running profileを並べ、意味の検証が同じであることを確かめます。

```text
go run ./tools/reference-target-profile/cmd/tei-target-profile \
  validate -profile tools/reference-target-profile/testdata/server.json

go run ./tools/reference-target-profile/cmd/tei-target-profile \
  run -profile tools/reference-target-profile/testdata/server.json \
  -workspace reference/workspace \
  -contract tools/reference-generator-selection/testdata/line-amount/contract.json \
  -fixtures tools/reference-generator-selection/testdata/line-amount/fixtures.json \
  -candidates tools/reference-generator-selection/testdata/candidates

go run ./tools/reference-target-profile/cmd/tei-target-profile \
  compare \
  -profiles tools/reference-target-profile/testdata/server.json,tools/reference-target-profile/testdata/embedded.json \
  -workspace reference/workspace \
  -contract tools/reference-generator-selection/testdata/line-amount/contract.json \
  -fixtures tools/reference-generator-selection/testdata/line-amount/fixtures.json \
  -candidates tools/reference-generator-selection/testdata/candidates

go run ./tools/reference-target-profile/cmd/tei-target-profile \
  deploy -profile tools/reference-target-profile/testdata/desktop-replace.json \
  -candidates tools/reference-generator-selection/testdata/candidates \
  -root /tmp/tei-target-profile-root

go run ./tools/reference-target-profile/cmd/tei-target-profile \
  rollback -root /tmp/tei-target-profile-root
```

## Profileの項目

| 項目 | 内容 |
| --- | --- |
| `processModel` | `oneshot`（caseごとに別process）、`long-running`（1 processで複数request）、`batch`（未対応） |
| `storage` | `none`、`filesystem`、`sqlite`、`remote`。`filesystem`だけdeploy参照で扱う |
| `network` | `none`、`loopback`、`allowed` |
| `update` | `replace`、`in-place`、`rolling`。deploy参照で扱う |
| `memoryLimitBytes` | 正の整数 |
| `observability` | 観測手段の一覧。`stdout`・`stderr`・`exit-code`はcase実行から観測する |
| `unsupported` | このprofileで扱わない機能 |
| `generatorId` | このprofileで選ぶ生成器のid（任意） |

`testdata/server.json`はlong-running・filesystem・network allowed・rolling updateで`long-running-template`を選び、`testdata/embedded.json`はone-shot・sqlite・network none・replace updateで`alternative`を選びます。serverの`audit-log`はこのReferenceでは提供しないため、`unsupported`へ明示しています。`testdata/desktop-replace.json`はfilesystem・replace updateで`template`を選び、ローカルartifact配備の参照に使います。

## 実行とreceipt

`run`は次を一つのreceiptへまとめます。

- profileのid・versionと全項目
- 選択した契約・fixture・候補の選抜receipt
- 実行した制約と、実際に強制した方法（`enforcement`）
- 観測したchannelと、未対応channel（`observation`）
- processModelの実行証拠（`processModel`）
- profileが扱う・扱わない機能（`unsupported`）
- 同じfixtureで合格したか

memory limitは`GOMEMLIMIT`としてselectionのサブプロセスへ渡します。これはGo runtimeだけに効くsoft limitであり、OSや実機のメモリ制約を証明しません。Python/JavaScript等の候補には効かないため、`enforcement.memory`へ`not-enforced`と記録します。

`observability`の`stdout`・`stderr`・`exit-code`は、選択候補の各case実行からdigest・exit codeを観測します。`audit-log`等の未対応channelを宣言したprofileは`observation.unsupported`へ記録し、`passed=false`にします。暗黙に別の観測手段で代替しません。

`processModel`は、oneshotならcaseごとのprocess実行、long-runningなら1 processへ全requestを送り全responseを比較します。long-running probeはGo・Python・JavaScript候補を対象とし、processが途中で終了した場合、終了codeが0以外の場合、responseの欠落・過剰・期待値との不一致、末尾を含むJSONの破損がある場合は`passed=false`にします。全responseをEOFまで読み、末尾の空白だけは許容します。batchは未対応として返します。

## Python/JavaScriptの常駐候補

`python-long-running-template`と`javascript-long-running-template`は、既存の単発候補と同じ計算・拒否をnewline-delimited JSONで繰り返す候補です。既存の単発テンプレートのsourceとtemplate digestは変えていません。

```bash
go run ./tools/reference-generator-selection/cmd/tei-generator-build \
  -contract tools/reference-generator-selection/testdata/line-amount/contract.json \
  -strategy python-long-running-template -out /tmp/tei-script-candidates/python-long-running-template
```

JavaScriptは`-strategy javascript-long-running-template`で生成します。Profileの`processModel`を`long-running`、`generatorId`を生成したstrategyと同じ値にして、`run`または`compare`の`-candidates`へ候補の親directoryを指定します。

- 選抜を通った候補だけを常駐probeで実行し、単発実行の選抜と常駐実行の両方を合格条件にします。
- Pythonは`python3`/`python`、JavaScriptは`node`を探し、実行可能な処理系のcommand・versionを`processModel.runtime`へ記録します。処理系がなければ不合格です。
- 1 processで全requestを処理したこと、応答数、case結果、終了code、stderr digestを記録します。単発候補を選んでも常駐の合格にはなりません。
- 処理系の隔離、依存解決、OSのメモリ上限、実Targetのsupervisorは扱いません。

## ローカルartifact配備

`deploy`は、選択したGo候補をbuildし、ローカルfilesystemのrootへartifactとして配置します。実Targetへの配備ではなく、`storage: filesystem`と`update`の最小の挙動を確認する参照です。

- `replace`: 新しいartifactを同じrootへrenameし、既存artifactは`history/<digest>`へ保持します。
- `in-place`: 既存artifactを上書きし、上書き前のartifactを`history/<digest>`へ保持します。
- `rolling`: `slots/a`と`slots/b`の2 slotを使い、非active slotへ新artifactを置いてから`current` pointerをatomicに切り替えます。旧slotは保持し、`rollback`でcurrent/previousを入れ替えます。
- 配備後にartifact digestを再計算し、build結果と一致することを確認します。history/slotのdigestも照合します。
- `sqlite`・`remote` storage、Go以外の候補は未対応として`passed=false`にします。
- 実Targetのsupervisor・health check・traffic drain・電源断耐性・複数ホスト配備は扱いません。

## Profile間の意味比較

`compare`は、複数のprofileを同じ契約・fixture・候補集合で順に実行し、各profileが選んだ生成器の**case別出力**を比較します。

- 各caseの`state`・`amount`・`currency`・`code`・`passed`が全profileで一致する場合だけ`equal: true`。
- あるprofileでcase結果が欠けた場合、またはprofileの実行が不合格の場合は`semanticsEqual: false`。
- `passed`は、全profileの実行が合格し、かつ`semanticsEqual`がtrueの場合だけtrue。
- profileの宣言差（processModel、storage、network、update、memoryLimitBytes）は`profileDifferences`へ記録し、意味の差と混同しない。
- `unsupported`の差は`unsupportedDifferences`、強制方法の差は`enforcementDifferences`へ記録する。
- 比較は宣言の強制・実機動作・本番同等性を証明しない。`grantsRuntimeAuthority`と`provesRealTarget`は常にfalse。

## 守っている境界

- profileは意味を変えない。契約・fixture・候補は同じものを使う。
- profileは`generatorId`で生成器を一つ選ぶ。選ばれなかった候補は`not selected by target profile`として拒否し、別の生成器へ暗黙にフォールバックしない。
- `processModel`はoneshotのprocess-per-case、long-runningのsingle-processを実行で確認する。batchは未対応。
- `storage`の`filesystem`と`update`の`replace`/`in-place`/`rolling`は、`deploy`のローカルfilesystem参照で限定的に確認する。`network`、`sqlite`/`remote` storageは宣言または未対応のまま。
- `observability`は`stdout`・`stderr`・`exit-code`だけをcase実行から観測する。未対応channelは`unsupported`として返し、合格にしない。
- profileの合格や`semanticsEqual`は、実機・実OS・実Targetでの動作を証明しない。
- 組込み固有のリアルタイム性・安全認証・電源断耐性は扱わない。

## 未実装

- OS・CPU・RTOS別の実行環境と、実機での資源測定。
- `network`の強制、`sqlite`/`remote` storage。
- batch processModel、long-runningのJava候補。
- `audit-log`・分散trace・metrics等の観測channel。
- 実Targetへの配備、ロールバック、長期運用。
- 異なるfixture集合・異なる期待値でのprofile比較。現在は同じfixtureの範囲だけを比較する。
