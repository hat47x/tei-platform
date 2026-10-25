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

go run ./tools/reference-target-profile/cmd/tei-target-profile \
  rollback -root /tmp/tei-target-profile-root -to sha256:...
```

## Profileの項目

| 項目 | 内容 |
| --- | --- |
| `processModel` | `oneshot`（caseごとに別process）、`long-running`（1 processで複数request）、`batch`（1 processで全requestを処理して終了） |
| `storage` | `none`、`filesystem`、`sqlite`、`remote`。`filesystem`だけdeploy参照で扱う |
| `network` | `none`、`loopback`、`allowed` |
| `update` | `replace`、`in-place`、`rolling`。deploy参照で扱う |
| `memoryLimitBytes` | 正の整数 |
| `cpuTimeLimitSeconds` | 任意の非負整数。0・省略は未宣言。`-enforce-cpu` では正の値が必要 |
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

`processModel`は、oneshotならcaseごとのprocess実行、long-running/batchなら1 processへ全requestを送り全responseを比較します。long-running/batch probeはGo・Python・JavaScript・Java・WASM・WASI候補を対象とし、processが途中で終了した場合、終了codeが0以外の場合、responseの欠落・過剰・期待値との不一致、末尾を含むJSONの破損がある場合は`passed=false`にします。全responseをEOFまで読み、末尾の空白だけは許容します。batchはsingle ephemeral processとして扱い、scheduler・job queue・再試行は実装しません。WindowsのNode wasm_execはmulti-request streamの終了時に非0で終了するため、WASMのlong-running/batch probeはLinux CIだけで検証し、Windowsでは未対応として扱います。WASIはNodeの`node:wasi`（preview1）で実行します。

常駐probeはdirectory名ではなく、選抜receiptの候補IDに一致するmanifestを探します。候補IDの重複、languageやsource digestの変更、実際のsource bytesと選抜digestの不一致は、processを起動する前に拒否します。照合した単一sourceのbytesを専用の一時directoryへコピーし、そのコピーをbuildまたは実行します。候補directoryの隣接ファイルは取り込みません。

Goは選抜と同じ `go build -trimpath -buildvcs=false -o ... main.go` でbuildし、binary digestも選抜receiptと一致した場合だけ実行します。同じsourceでもtoolchainやbuild環境の違いでbinaryが変われば拒否します。Javaは`javac -d <classes> Main.java`でcompileし、`Main.class`のdigestが選抜receiptと一致した場合だけ実行します。WASMは`GOOS=js`、WASIは`GOOS=wasip1`で`.wasm`へbuildし、wasm binaryのdigestを選抜receiptと照合します。`processModel`には `candidateId`、`sourceDigest`、`binaryDigest`、`binding=source-and-binary-digest`を記録します。Python/JavaScriptは `binding=source-digest`を記録し、interpreterや依存ライブラリの同一性までは保証しません。

対象は単一の通常ファイルです。Go/WASM/WASIは `main.go`、Pythonは `.py`、JavaScriptは `.js`、Javaは `Main.java`を対象とし、sourceのdirectory参照・symlink・複数ファイル構成は対応しません。これは参照probeの実行対象を選抜証拠と結び付ける確認であり、署名済み来歴や実Targetへの配備受理ではありません。ローカル配備のGo buildも同じ単一 `main.go` のbuild方法を共有しますが、配備へ選抜receiptとの照合を追加したものではありません。

## 実行時間とCPU時間の観測

`processModel` はprocessの実行回数とともに時間の観測値を残します。

- oneshot: `cases[].measurement` にcaseごとの経過時間・CPU時間を記録します。複数processの合計を単一processの測定値として示さず、先の選抜で観測した値を使います。
- long-running: `measurement` にprobeの1 processについて記録します。build後、`Start` の直前から全応答の読取り・終了待ちの完了までを測ります。request送信、応答のJSON解釈・case比較、終了待ちを含む参照実行器から見た経過時間です。
- long-runningの `firstResponseNanos`: `Start` の直前から、JSONとして解釈できた最初の応答までの経過時間です。起動・wrapper・最初のrequestの処理・入出力・JSON解釈を含み、純粋な起動時間ではありません。応答がなければ省略します。応答の存在と意味の合格は別で、不正な値を返した候補にも観測値は残ります。

各 `measurement` の `elapsedNanos`、`userCpuNanos`、`systemCpuNanos` の単位はnanosecondです。CPU時間はOSのprocess stateを通じて取得し、GoのAPIが返すprocess・子processの会計範囲に従います。wrapperを使えばその処理も含み得ます。起動失敗などでprocess stateがなければmeasurementを省略します。CPU時間0はOSの時間分解能内での観測値であり、未観測の代わりではありません。

`goos`、`goarch`、`harnessGoVersion` は参照実行器の環境です。候補のsource/binary digest、build情報、Python/JavaScriptの処理系情報は既存のreceiptに残します。`compare` も各profileの `processModelEvidence` にこれらの値を保持します。意味の一致判定は従来どおりcaseの結果を比較し、時間の差を意味の差や合否条件にはしません。

これらはローカル参照実行器の単一試行の観測値です。CPU・メモリ上限の強制、メモリ使用量、子孫process全体の監視、純粋な起動時間、実Targetや機種間の性能比較、実配備時の更新時間は今回の測定範囲に含みません。機種・負荷・OSの会計方法が異なる値から、性能順位やリアルタイム性を判断できません。

## CPU時間上限の適用（Linux、明示指定時）

`run` / `compare` の `-enforce-cpu` は、profileの `cpuTimeLimitSeconds` を候補の各processへ適用します。既定は未宣言なら `enforcement.cpu=not-declared`、正の値を宣言しただけなら `declared-only` です。

```text
go run ./tools/reference-target-profile/cmd/tei-target-profile \
  run -profile tools/reference-target-profile/testdata/cpu-batch.json \
  -workspace reference/workspace \
  -contract tools/reference-generator-selection/testdata/line-amount/contract.json \
  -fixtures tools/reference-generator-selection/testdata/line-amount/fixtures.json \
  -candidates tools/reference-generator-selection/testdata/candidates \
  -enforce-cpu
```

- `prlimit --cpu=N:N --` でsoft・hard双方のCPU時間上限を秒単位で設定します。起動前に同じwrapper経由で `/proc/self/limits` を読み、要求値を確認します。利用不可・不一致なら候補の選抜前に `passed=false` を返します。適用を外して実行し直す経路はありません。
- `cpu` に要求値、観測したsoft/hard値、wrapper argv、利用可否、失敗理由、stderr digestを記録します。成功時の `enforcement.cpu` は `linux:prlimit-cpu` です。compareの各profileにもCPU宣言と証拠を残します。
- 選抜の各caseと、追加のlong-running/batch probeに同じ制限を適用します。oneshotでは各processが個別に上限を持ち、常駐・batch probeでは全requestを処理する1 processに適用します。上限到達で候補が非0終了すれば、既存の終了検査で不合格になります。
- `-enforce-network` と併用できます。CPU wrapperの内側にnetwork namespace wrapperを置き、組み合わせた起動方法でCPU上限を確認します。実際のargvは `selection.execution.commandPrefix` に残ります。別runnerを使うlineage検証は、既存のwrapper制約に従って拒否します。
- `-enforce-cpu` 指定時の省略・0、Linux以外、utility不在、probe失敗は成功として扱いません。負値はprofile検証で拒否します。このflagは `run` / `compare` のみで利用できます。

[LinuxのRLIMIT_CPU](https://man7.org/linux/man-pages/man2/getrlimit.2.html)はprocess単位のCPU時間上限です。経過時間、CPU利用率・コア数、fixture全体や子孫process群の合算予算を制御するものではありません。forkしたprocessは上限を継承しても各processのCPU時間は別に計数されます。compile・処理系検出・メモリ・process数の制約は今回のwrapperの対象外です。既存の30秒の実行期限とGoのsoft memory limitは別に扱います。

実動試験は、同じ合成workloadが制限なしでCPU時間2秒に到達し、1秒の上限を適用すると親の経過時間によるtimeoutより先にsignalで終了することを対照で確認します。同じ業務fixtureをoneshot・long-running・batchで比較し、network wrapperとの併用も確認します。ローカル参照実行器の検証であり、正式なHost権限、特権processへの制約、実Targetのsupervisor、リアルタイム保証や本番配備の証拠ではありません。利用できないOS・utility・namespaceの実動試験はskipし、環境不足を実動成功として数えません。CI全体のsuccessだけでは実動試験の実行とskipを区別できません。

## Python/JavaScript/Java/WASIの常駐候補

`python-long-running-template`、`javascript-long-running-template`、`java-long-running-template`、`wasi-long-running-template`は、既存の単発候補と同じ計算・拒否をnewline-delimited JSONで繰り返す候補です。既存の単発テンプレートのsourceとtemplate digestは変えていません。

```bash
go run ./tools/reference-generator-selection/cmd/tei-generator-build \
  -contract tools/reference-generator-selection/testdata/line-amount/contract.json \
  -strategy python-long-running-template -out /tmp/tei-script-candidates/python-long-running-template
```

JavaScriptは`-strategy javascript-long-running-template`、Javaは`-strategy java-long-running-template`、WASIは`-strategy wasi-long-running-template`で生成します。Profileの`processModel`を`long-running`、`generatorId`を生成したstrategyと同じ値にして、`run`または`compare`の`-candidates`へ候補の親directoryを指定します。

- 選抜を通った候補だけを常駐probeで実行し、単発実行の選抜と常駐実行の両方を合格条件にします。
- Pythonは`python3`/`python`、JavaScript/WASIは`node`、Javaは`javac`/`java`を探し、実行可能な処理系のcommand・versionを`processModel.runtime`へ記録します。処理系がなければ不合格です。
- 1 processで全requestを処理したこと、応答数、case結果、終了code、stderr digestを記録します。単発候補を選んでも常駐の合格にはなりません。
- 処理系の隔離、依存解決、OSのメモリ上限、実Targetのsupervisorは扱いません。

## Linuxでのnetwork隔離

`run`/`compare`に`-enforce-network`を付けると、`network: none`の候補をLinuxのuser/network namespace内で実行します。既定は従来どおり宣言だけです。`loopback`/`allowed`の強制は未対応として拒否し、指定を無視して実行しません。`validate`/`deploy`/`rollback`への指定も拒否します。

```bash
go run ./tools/reference-target-profile/cmd/tei-target-profile run \
  -enforce-network \
  -profile tools/reference-target-profile/testdata/embedded.json \
  -workspace reference/workspace \
  -contract tools/reference-generator-selection/testdata/line-amount/contract.json \
  -fixtures tools/reference-generator-selection/testdata/line-amount/fixtures.json \
  -candidates tools/reference-generator-selection/testdata/candidates
```

- `unshare --user --map-root-user --net --`を使用します。起動前に`readlink /proc/self/ns/net`を隔離下で実行し、親とは別のnetwork namespaceに入ったことを確認します。wrapperのargv、親とprobeのnamespace、失敗理由、stderr digestを`network`へ記録します。
- 非Linux、コマンド不在、kernelによる拒否、親と同じnamespace等では、候補を一つも実行せず`passed=false`と`enforcement.network: unavailable`を返します。候補実行時にwrapperが失敗しても隔離なしへ切り替えません。
- 選抜のcase実行と、追加のlong-running probeの両方を隔離します。`selection.execution.commandPrefix`に使用したwrapperを記録します。oneshotはcaseごとに別namespace、long-running probeは1 namespace内の1 processで全要求を処理します。
- 実行対象のlineage検証は別のrunnerを使うため、隔離指定中は未対応として拒否します。
- 隔離は候補実行時のIP network namespaceの分離です。ビルド・処理系の検出、filesystem、AF_UNIX、ホスト側serviceを介する代理通信、CPU・メモリ・子孫processの管理は範囲外です。新しい実行先としての汎用sandboxや本番Hostの安全性を証明しません。
- 実動試験では同じホストのIPv4 loopback listenerへの接続が隔離なしでは成立し、隔離中は成立せず、同じ業務fixtureがoneshot/long-runningで成功することを確認します。Python/JavaScriptの隔離下でのfixture比較も行います。IPv6や外部ネットワークの実通信は試験していません。
- namespaceや処理系を利用できない環境では実動試験を理由付きでskipします。そのskipをnetwork隔離の成功とは扱いません。利用できないときの拒否は別の負例で検証します。

## ローカルartifact配備

`deploy`は、選択したGo候補をbuildし、ローカルfilesystemのrootへartifactとして配置します。実Targetへの配備ではなく、`storage: filesystem`と`update`の最小の挙動を確認する参照です。

- `replace`: 新しいartifactを同じrootへrenameし、既存artifactは`history/<digest>`へ保持します。
- `in-place`: 既存artifactを上書きし、上書き前のartifactを`history/<digest>`へ保持します。
- `rolling`: `slots/a`と`slots/b`の2 slotを使い、非active slotへ新artifactを置いてから`current` pointerをatomicに切り替えます。旧slotは保持し、`rollback`でcurrent/previousを入れ替えます。
- `rollback -to <digest>`: `replace`/`in-place`の`history/<digest>`から指定artifactを選び、digestを照合してからcurrentへatomicに戻します。戻す前のcurrentはhistoryへ保持します。
- currentの退避先historyが既にある場合は、内容のdigestを照合します。読めない・一致しない場合はcurrentの置換前に拒否し、壊れたhistoryも上書きしません。この確認は電源断耐性を提供しません。
- 配備後にartifact digestを再計算し、build結果と一致することを確認します。history/slotのdigestも照合します。
- `sqlite`・`remote` storage、Go以外の候補は未対応として`passed=false`にします。
- 実Targetのsupervisor・health check・traffic drain・電源断耐性・複数ホスト配備は扱いません。
- `rollback -to`は`replace`/`in-place`のhistoryだけを対象とし、rollingのslot rollbackは`-to`なしの`rollback`を使います。

### 同じ配備先への同時更新

`deploy`（replace/in-place/rolling）と両方の`rollback`は、root内の`.update-lock` directoryを排他的に作成します。配備はbuild前、rollbackは履歴・pointerの読取り前に取得し、結果の照合まで保持します。取得できなければ待機・再試行せず`passed=false`を返し、artifact・history・pointer・stagingを変更しません。別rootの更新は妨げません。

正常終了・通常のエラーではlockを解放します。強制終了・電源断で残ったlockは自動削除しません。復旧時は、同じrootに対する更新processが動いていないことと、artifact・history・pointerの状態を確認してから、空の`.update-lock` directoryを手動で削除します。稼働中のlockを削除すると排他が失われます。

これは、この手順に従うローカルfilesystemのwriter間の排他です。古い版のツール、外部からの書込み、lock/rootの差替え、ネットワークfilesystem、分散lock、電源断後の整合性や自動復旧は保証しません。実Targetでの実行権限やhealth checkを与えるものでもありません。

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
- `processModel`はoneshotのprocess-per-case、long-running/batchのsingle-processを実行で確認する。batchはGo候補に加え、Java候補とWASI候補でも確認する。
- `storage`の`filesystem`と`update`の`replace`/`in-place`/`rolling`は、`deploy`のローカルfilesystem参照で限定的に確認する。`network`は以下のopt-inの範囲で強制し、`sqlite`/`remote` storageは宣言または未対応のまま。
- `observability`は`stdout`・`stderr`・`exit-code`だけをcase実行から観測する。未対応channelは`unsupported`として返し、合格にしない。
- profileの合格や`semanticsEqual`は、実機・実OS・実Targetでの動作を証明しない。
- 組込み固有のリアルタイム性・安全認証・電源断耐性は扱わない。

## 未実装

- OS・CPU・RTOS別の実行環境と、実機での資源測定。
- `loopback`等のnetwork方針の強制、Linux namespace以外の実行器、`sqlite`/`remote` storage。
- batch scheduler・job queue・再試行。
- WindowsでのWASM long-running/batch probe。Node wasm_execの終了挙動に依存する。
- WASIのwasmtime・browser・embedded runtimeでのlong-running/batch。Nodeの`node:wasi`だけを検証する。
- `audit-log`・分散trace・metrics等の観測channel。
- 実Targetへの配備、ロールバック、長期運用。
- 異なるfixture集合・異なる期待値でのprofile比較。現在は同じfixtureの範囲だけを比較する。
