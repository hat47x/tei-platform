# セルフビルドのGo Module供給境界（SB-0b・設計候補）

- 対象：[#2831](https://github.com/hat47x/tei-platform/issues/2831) の SB-0b、親Gate [#1119](https://github.com/hat47x/tei-platform/issues/1119) G4 / CP-5
- 関連：[#943](https://github.com/hat47x/tei-platform/issues/943)、[#979](https://github.com/hat47x/tei-platform/issues/979)、[#1088](https://github.com/hat47x/tei-platform/issues/1088)、[#1090](https://github.com/hat47x/tei-platform/issues/1090)、[#1096](https://github.com/hat47x/tei-platform/issues/1096)、[#1103](https://github.com/hat47x/tei-platform/issues/1103)、[#1109](https://github.com/hat47x/tei-platform/issues/1109)
- 作成日：2026-11-07
- 状態：**設計候補。production仕様の採用・実装・qualified PASSではない**
- 実観測：[Go Module供給の実観測](../evidence/TEI_GO_MODULE_SUPPLY_OBSERVATION_2026-11-07.md)
- 既存の判定用ノート：[プラグインセルフビルドのproduction最小契約](PLUGIN_SELF_BUILD_PRODUCTION_CONTRACT_2026-10.md)（本稿はこの §2.2・§2.3・§5 の層別判定を置き換えず、その未決事項のうちGo Module供給に限定して差分を詰める）
- 計画側の順序：`plan/workstreams/TEI_PRODUCT_CRITICAL_PATH_2026-10.md` の SB-0a/SB-0b は、同じ計画ファイルへ §35 を追加する[Draft PR #2877](https://github.com/hat47x/tei-platform/pull/2877)が所有する。本稿はその計画文を複製せず、**SB-0bの契約だけ**を切り出す

## 1. この文書が解く問題

SB-0aの観測で、`cmd/tei`は隔離`GOMODCACHE`では`missing-module`、Host module cacheを事前供給すると`external-unresolved`になった。つまり次に止まるのは「Moduleが足りない」ではなく、**「供給されたModuleのbyteが正しいと、誰が何で確認したか」**である。

現行の`tools/internal/gobuilder`は外部Module根を`ResolveSourceClosure`で一律に拒否する。これは安全側だが、`cmd/tei`のような実Targetを一度も通せない。かといって拒否を外すだけでは、**Host cacheの中身を暗黙の入力にする**という、`GOPROXY=off`と隔離`GOMODCACHE`が防いでいた危険をそのまま戻すことになる。

本稿の主張は一つである。**外部Moduleを入力として認める条件は「供給元」ではなく「byteの照合値」で決める。** 供給元の経路（Host cache、社内mirror、同梱tarball）は来歴の記録であり、受理の条件ではない。

## 2. 実測から確定する三つの事実

いずれも[実観測](../evidence/TEI_GO_MODULE_SUPPLY_OBSERVATION_2026-11-07.md)の値である。文書上の推定ではない。

### 2.1 供給があれば`cmd/tei`は解決できる

Host cacheの`cache/download`を丸ごと隔離`GOMODCACHE`へ複製した状態で、SB-0aのpreflightは`external-unresolved`、解決package 109、未解決0、外部Module根12件を返した。**SB-0aの`missing-module`は供給の有無だけの差**であり、Sourceや構文の問題ではなかった。

### 2.2 壊れた供給は`unknown`へ落ちるが、区別できない

Module archive 1件のbyteを1byteだけ変えた供給では、preflightは`unknown`（`unzip ... flate: corrupt input before offset 44`）を返した。fail-closedとしては正しい。しかし`unknown`は「Sourceの誤り」と同じ箱であり、**供給の破損**という別の打ち手を指せない。ここは分類を一つ足す必要がある（§4）。

### 2.3 成果物のdigest一致は、供給byteの同一性を証明しない

同じ checkout を、隔離`GOMODCACHE`内の**展開済みModuleのソースファイル**だけ書き換えて`go build -trimpath`すると、次の二通りが起きた。

| 改変 | `-trimpath`の成果物digest | 解釈 |
| --- | --- | --- |
| コメント行を1行追加 | 未改変の供給と**同一**（`sha256:D60C5FCC…`） | 成果物一致では改変を検出できない |
| 定数の値を1つ変える | **異なる**（`sha256:504DF5E4…`） | 成果物が静かに変わる。照合が無ければ気づけない |

前者が決定的である。**同じ成果物byteから「同じ供給で構築した」と推論できない。** 二回ビルドのbit一致（`plan/design/PLUGIN_SELF_BUILD_PRODUCTION_CONTRACT_2026-10.md` §5 のSB-1層）は、供給manifestを伴わない限り、供給の同一性の証拠にならない。

### 2.4 既存toolchainで、供給の改変を検出できる手段はすでにある

同じ供給に対して`go mod verify`は次の結果を返した。

- 未改変：`all modules verified`
- 展開済みディレクトリ改変：`golang.org/x/text v0.41.0: dir has been modified`

`go mod verify`は`go.sum`の`h1:`（archiveのハッシュ）と、展開済みの内容を突き合わせる。ネットワークを使わない。**SB-0bはこの既存機構を再利用し、第二のハッシュ実装を作らない。**

## 3. SB-0bの最小契約（案）

### 3.1 供給manifest（入力）

一回のBuild試行に対して、次を宣言する。名称は提案であり、production型の確定ではない。

~~~text
ModuleSupplyManifest
  schemaVersion        : 版
  target               : 対象package path（例 ./cmd/tei）
  moduleDefinition     : go.mod digest, go.sum digest（存在・不在を区別）
  supplyOrigin         : 供給元の種別と識別（host-cache-copy / internal-mirror / bundled-archive）
  entries[]            : module path, version,
                         archiveDigest     : 供給したarchive byteのsha256
                         moduleH1          : go.sumの当該行の h1: 値
                         goModH1           : go.sumの /go.mod 行の h1: 値
                         ziphashDigest     : 供給元の .ziphash を写したもの（あれば）
                         extractedTreeDigest : 展開済みツリーのpath+digest集合の集約
~~~

`archiveDigest`は**供給元の物理パスを入れない**。§2.3のとおり、同じbyteなら同じ値になるべきで、Hostのディレクトリ名は内容同一性ではない。物理パスは観測ログ側に残す。

### 3.2 事前条件（拒否条件）

次のいずれかが成り立つとき、Buildを開始しない。

1. `go.sum`に当該`module@version`の`h1:`行が無い（`-mod=readonly`では解決できない）
2. 供給archiveのsha256から計算した`h1:`が`go.sum`の値と一致しない
3. `go.mod`行の`h1:`が一致しない
4. `go mod verify`が非0、または`all modules verified`以外の出力を返す
5. 外部Module根が`GOMODCACHE`の外を指す（`GOMODCACHE`内の`module@version`配置だけを許す）
6. `GOPROXY`が`off`以外、`GOWORK`が`off`以外、`GOTOOLCHAIN`が`local`以外
7. 試行と同じ`GOMODCACHE`を別の試行が共有している

3と4は冗長に見えるが、役割が違う。`h1:`照合は**供給時点のarchive**の検査、`go mod verify`は**展開後（または展開済み供給）のツリー**の検査である。§2.3のコメント改変は展開後ツリーの改変なので、archiveのh1だけでは捕まらない。両方を残す。

### 3.3 供給元とTrustを分ける

`supplyOrigin`は「どこから来たか」であり、`Trust`ではない。Host cacheのbyteが正しいことは、そのbyteが**誰のBuildで作られたか**を証明しない。前者は§3.2の照合で閉じ、後者は既存のAttestation/Trust判定（`PLUGIN_SELF_BUILD_PRODUCTION_CONTRACT_2026-10.md` §2.2）へ渡す。**SB-0bはTrust判定を作らない。**

同じ理由で、`go.sum`の存在は何も証明しない。`go.sum`は**照合の期待値**であり、期待値と実際のbyteを突き合わせた記録が無ければ、期待値だけが残る。

### 3.4 既存実装との接続（2026-11-07 追補・統合後）

本節の契約案を、次の実装が分担して持つ。**分担は重なっておらず、本稿の§3.1・§3.2・§4の全部を一つの実装が満たしているわけではない。**

| 実装 | 担当範囲 | 状態 |
| --- | --- | --- |
| `gobuilder`の`corrupt-supply`分類（[PR #3144](https://github.com/hat47x/tei-platform/pull/3144)、main `e3e94204f`） | §4の分類変更。`go list -e`の既知文面から供給破損を分ける | **統合済み**。単体試験と実観測を伴う |
| `tools/reference-go-module-supply-verify`（[PR #3142](https://github.com/hat47x/tei-platform/pull/3142)、main `8b7edf867`） | §3.1・§3.2の照合部分。`go.sum`の`h1:`を独立に再計算し、実際に選択されたModuleだけへ適用する。宣言したが選択されないModuleは不合格にしない | **統合済み** |
| `tools/reference-go-build-with-supply`（[PR #3156](https://github.com/hat47x/tei-platform/pull/3156)、main `5ad04cbdf`） | §6のL1側。供給だけを使う隔離環境で実Buildし、artifact digestとSource閉包の資格を別欄で返す | **統合済み** |

前者二つは「どの打ち手か」を選ぶ分類と「供給が記録と合うか」の照合であり、三つ目は「Buildできること」と「閉包の資格」の分離である。§5の`ResolveSourceClosure`変更は、いずれも行っていない。

PR #3142から本稿の§3.1へ取り込むべき差分が一つある。**照合の対象は`go.mod`の宣言Moduleではなく、`go list`が実際に選択したModuleに限る。** `cmd/tei`は17件を宣言し12件を選択するため、宣言全体を対象にすると選択されない5件の欠落で正しい供給を不合格にする。本稿の§3.1の`entries[]`は「供給した」の記録であり、照合の分母は実選択側で決める、と読むべきである。

### 3.5 供給が通っても閉包の資格は成立しない（X1の実測）

`cmd/tei`について、供給を渡した実Buildと、同じadapterの`ResolveSourceClosure`を同じ条件で走らせた結果は次である（[X1の記録](../evidence/TEI_GO_BUILD_WITH_SUPPLY_X1_2026-11-07.md)）。

| 欄 | 値 |
| --- | --- |
| `buildSucceeded` | `true`（exit 0、artifact digest `sha256:a512ffba…`） |
| `sourceClosureState` | `rejected` |
| `artifactProducedDespiteUnqualifiedClosure` | `true` |

拒否理由は二段階ある。第一に`ResolveSourceClosure`自身が`GOPROXY=off`で走るため、供給が渡っていない。第二に、**供給を渡しても`moduleDir != root`で外部Module根を拒否する**設計だからである。したがって理由の形が変わっても`rejected`であり続ける。

これは本稿§5の変更が**未着手であること**の実測であり、同時に、§6のL1とL2を同じ「成立した」でまとめてはいけないことの実測でもある。artifactが在ることは、閉包の資格の証明ではない。

### 3.6 次の判断点：§5を行うか、`rejected`のまま進めるか

二つの道があり、どちらも本稿では決めない。

| 選択肢 | 内容 | 必要な作業 | リスク |
| --- | --- | --- | --- |
| A: §5を行う | `ResolveSourceClosure`を、照合済み供給を前提に外部Module入力を認める形へ変える | 本稿§5の1〜5、§7のSBP-24〜30 | 安全側の停止線を外す。既存の`SourceClosureDigest`の意味を保つ必要がある |
| B: `rejected`のまま進める | artifactと閉包を別Claimとして扱い、閉包の資格を要求しない対象でSB-A/SB-Bを進める | 対象の選び直し（外部Moduleを選ばない対象） | 資格の無い閉包を前提にしたEvidenceが増える |

方針の所有者は[#1090](https://github.com/hat47x/tei-platform/issues/1090)である。本稿はAの手順を§5に書いたが、**採用はしていない**。Bを選ぶ場合も、§3.5の三欄を残す限り、artifactと資格の混同は検出できる。

## 4. 分類の変更（SB-0a → SB-0b）

SB-0aの四分類に、供給の失敗を区別する一項を足す。

| 分類 | 意味 | 次の打ち手 |
| --- | --- | --- |
| `supported` | 対象Module根のみ | 既存`ResolveSourceClosure`へ |
| `external-supplied-verified` | 外部Module根があり、§3.2の全照合が通った | 外部Module入力つきBuildへ（新規） |
| `missing-module` | 内容が無い、または`go.sum`行が無い | §3.1の供給を用意する |
| `corrupt-supply` | 内容はあるが`h1:`不一致・`go mod verify`失敗・archive破損 | 供給を差し替える（新規。Source修正ではない） |
| `unknown` | 上記以外の解決失敗 | 分類しない。Source・環境の調査が先 |

`corrupt-supply`を足す理由は、`unknown`が「Sourceを直す」と「供給を差し替える」を畳んでいることにある。`gobuilder.Environment`の方針上、供給の破損は**呼び出し側が直せる**失敗であり、Sourceの誤りは**実装者が直す**失敗である。同じ箱に入れると、CIの失敗メッセージがどちらを指すか読めない。

## 5. 外部Moduleを認める条件（既存Closureの変更方向）

`ResolveSourceClosure`は現在、`moduleDir != root`で停止する。SB-0b後の変更は次の順序にする。**拒否の撤去ではなく、照合済み供給の存在を前提条件にする。**

1. `ModuleSupplyManifest`の各entryが§3.2を満たす
2. 外部Module根がmanifestの`module path @ version`に一致し、`GOMODCACHE`配下にある
3. 外部Moduleの選択ファイルを`SourceMember`に含めるとき、**keyの空間を分ける**。対象repoのfileはrepo相対path、外部Moduleのfileは`module@version:path`の形にして、同一path文字列の衝突を防ぐ
4. 対象Moduleのclosure digest（既存`SourceClosureDigest`）と、外部Module入力のdigestを**別の値として記録**する。混ぜた一つのdigestを既存名称へ代入しない
5. 外部ModuleのSource変更は、対象repoのSource digestを変えない。逆に、対象repoのSource変更は外部Module digestを変えない。各試行のmanifestとclosureは、それぞれの入力側で失効する

4を明示するのは、既存の`SourceClosureDigest`がPluginとRuntime Engineの照合値として固定されているためである。外部Moduleを含めた新しい集約を同じ名前で返すと、過去のEvidenceと比較できない。

## 6. Claimの層別（既存計画への追補）

`plan/workstreams/TEI_PRODUCT_CRITICAL_PATH_2026-10.md` §35.3は、L1（Go Toolchainによる実artifact）、L2（TEI標準Build Graphからの実実行）、L3（Stage1生成TEI自身によるStage2の駆動）を分けている。Draft PR #2877が所有するこの区分を本稿は変更しない。SB-0bが関係するのはL1とL2の**前提**だけである。

| 主張 | SB-0bの関与 | 関与しないこと |
| --- | --- | --- |
| L1 実artifact | 供給manifestとclosure記録が、artifactの入力側の根拠になる | 供給byteのTrust、seedのTrust |
| L2 Graphからの実実行 | 外部Module入力もGraphの入力roleとして宣言できる | Graph完全性、Owner Acceptance |
| L3 Stage再入 | 何も加えない | — |

§2.3に従い、**L1のartifact digestだけで供給の同一性を主張しない**。L1を報告するときは、§3.1のmanifestと§3.2の照合結果を同じpacketへ入れる。

## 7. 追加の負例（試験仕様。actual PASSの主張ではない）

既存のSBP-01〜23（`PLUGIN_SELF_BUILD_PRODUCTION_CONTRACT_2026-10.md` §4）に、SB-0b固有の行を足す。

| ID | Given / 操作 | Expected |
| --- | --- | --- |
| SBP-24 | 供給archiveの1byte破損 | `corrupt-supply`で停止。`unknown`や`missing-module`へ畳まない |
| SBP-25 | 展開済みModuleのコメントを1行改変（成果物digestは同一） | `go mod verify`で拒否。**artifact digest一致を供給同一性の根拠にしない** |
| SBP-26 | 展開済みModuleの定数を改変（成果物digestが変わる） | 照合で拒否。成果物が偶然一致した場合も受理しない |
| SBP-27 | `go.sum`に行があるがarchiveのbyteが別物 | `h1:`照合で拒否。`go.sum`行の存在を成功条件にしない |
| SBP-28 | 同じ供給を二つの独立した`GOMODCACHE`へ与え、二回build | 成果物bit一致は**supply manifest一致と併せて**報告。片方だけを再現性として主張しない |
| SBP-29 | Host cacheを供給元にして成功 | 供給元を`host-cache-copy`として記録。Trust判定は別工程。供給成功を`accepted`と表現しない |
| SBP-30 | 外部Moduleを含むclosure digest | 既存`SourceClosureDigest`の名称・形式を外部Module込みへ流用しない。対象Module側と外部入力側を別値で残す |

SBP-25〜26は本稿の実測（§2.3）から直接作った負例であり、合成ではなく実Goの観測に基づく。ただし§7全体は**試験仕様**であり、SBP-24〜30のactual PASSは未取得である。

## 8. 未決事項

- `ModuleSupplyManifest`の正確なschema・format version・置き場所
- `h1:`計算をGo toolchainの出力に委ねるか、自前で再計算するか（自前実装は第二のハッシュ実装になり得るため、第一候補は`go mod verify`と `go mod download -json` の出力利用）
- 供給元の識別語彙と、`supplied-verified` を Evidence のどの層に置くか
- 同じ供給を複数Targetが共有する場合の失効単位
- 中間生成物（`dist`、`vendor`）を許すか。`vendor`は`go.sum`照合の対象が変わるため、別の契約として扱う
- Go以外のbuilder（npm、Maven等）への一般化。本稿はGo 1.27の観測に限定される

## 9. この文書の終了条件

- [ ] SBP-24〜30のownerと実装先を各Issueへ同期
- [x] `corrupt-supply`分類の実装と単体試験（[PR #3144](https://github.com/hat47x/tei-platform/pull/3144)。main `e3e94204f`）
- [x] `go.sum`照合を実選択へ限定する部分の実装（[PR #3142](https://github.com/hat47x/tei-platform/pull/3142)。main `8b7edf867`）。§3.1のmanifestへの接続は未着手
- [x] 供給下の実Buildと閉包資格を別欄で返す実装と実測（[PR #3156](https://github.com/hat47x/tei-platform/pull/3156)。main `5ad04cbdf`、[X1の記録](../evidence/TEI_GO_BUILD_WITH_SUPPLY_X1_2026-11-07.md)）
- [ ] §3.6のA/Bのどちらを採るかの決定（ownerは[#1090](https://github.com/hat47x/tei-platform/issues/1090)）
- [ ] 外部Module入力つきclosure digestの別値化と、既存照合値への非流用の回帰試験
- [ ] 供給manifestを伴うL1 packetの一枚目
- [ ] 別Go版・別OSでの再観測

本書は計画の正本を増やすものではない。SB-0bの**失敗の分類と照合の順序**を固定し、`#2831`のP0順序（SB-0a → SB-0b → SB-A → SB-B → SB-C）のうちSB-0bの部分だけを詰めた設計候補である。最終判断は元計画・ADR・actual Evidenceへ返す。
