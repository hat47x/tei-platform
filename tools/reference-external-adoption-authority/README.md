# reference-external-adoption-authority

技術検証用サンプル[`external-adoption-authority`](../../reference/external-adoption-authority/README.md)(サンプル9)の工程1〜4を、一つの通しの記録として実行する実装です。設計の正本は[技術検証用サンプルの統合設計 4.9節](../../plan/design/TECH_VALIDATION_SAMPLES_2026-10.md#49-外部アプリの契約を意味として取り込み案件で受理するまでexternal-adoption-authority)です。

この実装はサンプル固有の処理で、`internal/`の基盤には入れていません。既存資産は変更せずに呼びます。固定入力(`reference/sui-adoption`、拡張カタログの`testdata/extensions`)は作業ディレクトリへ複製してから使い、リポジトリには書きません。SUIやSEIのリポジトリとネットワークへは取りに行きません。外部側の変化は、固定した入力の複製を書き換えて模擬します。

## 構成

| 役 | 実体 | 担当 |
| --- | --- | --- |
| 試験(編成) | `go test`のプロセス | 工程1〜4、失敗注入、通しの照合、完了報告 |
| 外部側 | `reference/sui-adoption/README.md`の固定表の複製(`pinned-source.json`) | SUIの版と依存blob4件。失敗注入ではこの複製を書き換える |
| 工程1・2の資産 | `tools/reference-sui-adoption`と`tools/reference-semantic-extension-catalog`を試験の中で`go build`したCLI | 影の検査、拡張の解決・意味の固定記録・検証計画・名前空間所有者の判定 |
| 信頼の状態 | 試験のプロセス内の`TrustHistory`(`trust.go`) | 論理的な時点の列。各時点が所有者の信頼付与、署名者の信頼付与、署名者の鍵を持つ。判定は必ず時点を名指しして行う |
| ホスト | 試験のプロセス内の`Host`(`stage4.go`) | ソース受理と外部の名前空間の選択と外部意味の受理を、一つの決定として記録する合成のホスト |

時刻は論理的な値です(`BaseTime`、各時点は1時間ずつ進む)。壁時計を読まないため、WSL2の時計の跳びに影響されません。署名者の鍵は固定の種から導くため、毎回同じ署名と照合値になります。

## 既存資産を呼べるか

| 工程 | 既存資産 | 呼べるか | 理由と扱い |
| --- | --- | --- | --- |
| 1 | `reference/sui-adoption` | 固定入力として使う | 作業領域とM1〜M4を複製する。READMEの固定表を、外部側の固定コピーとして読む(`ReadPinnedTable`) |
| 1 | `tools/reference-sui-adoption` | CLIで呼べる | `package main`のため関数は呼べない。CLIを複製の中で四つの事例に実行し、報告(schema v2)を読む。報告は版と依存blobを定数で書くだけで照合しないため、外部側との照合はサンプル側(`CheckExternalPin`)で行う |
| 2 | `tools/reference-semantic-extension-catalog` | CLIで呼べる(評価は呼べない) | 拡張の解決、名前空間の権限、信頼付与、評価器の束縛、意味の固定記録と検証計画の出力、固定記録の再生と証跡の再照合、検証計画の再照合を、CLIの引数で呼ぶ。制約の評価(`evaluateSemanticConstraintWithPlan`)はCLIに出ていない |
| 2 | `tools/reference-sui-adoption/hierarchy`、`tools/reference-semantic-extension-catalog/acyclicgraph` | ライブラリとして呼べる | 工程1と同じ判定を共有パッケージで評価し、判定を二重に実装しない |
| 2〜4 | `tools/reference-authority-evidence-trust/trust` | ライブラリとして呼べる | 名前空間所有者(カタログの中)と署名者(サンプルのLookup)を同じ`Verify`で判定する |
| 3 | `reference/semantic-distribution` | ライブラリとして呼べる | `MaterializeReference`、`InspectSelectedReference` |
| 3 | `reference/semantic-build-bridge` | ライブラリとして呼べる | `InspectLockedExport`で構築計画と固定記録を照合してから読む |
| 3 | `reference/semantic-composition-preflight` | ライブラリとして呼べる | `SemanticPublicationMessage`と`InspectPublished`。現在時刻の引数には論理時刻を渡す |
| 4 | `reference/semantic-composition-preflight` | ライブラリとして呼べる(パス取込みだけ) | `InspectCurrentProjectMeaning`を合成ホストで呼ぶ。データのみの配布物は受理照合の対象にならないため、利用側はパス取込みで提供元の複製を読み、その意味の照合値が配布物の意味の照合値と同じことをサンプル側で照合する。`PreviewExternalSelectionChange`は使っていない |

## 工程の流れ

1. 工程1(`runStage1`、`stage1.go`): `reference/sui-adoption`を複製し、影の検査のCLIを四つの事例に実行します。四つの報告が固定版、写像、制約の保持範囲で一致することを確かめ、報告の固定版を外部側(固定表の複製)と照合します。依存blobが1件でも違えば入力の不一致として止まり、損失の記録も渡すものも作りません。照合が通れば、報告から損失の記録(`loss-record.json`、`lossless=false`、親の不在の意味`unresolved`、制約ごとの保持範囲)を作り、そのダイジェストと写像報告の照合値、入力の照合値を渡します。
2. 工程2(`runStage2`、`stage2.go`): 工程1の入力・報告・損失の記録を照合値で確かめてから(入口の照合)、保持範囲が`shadow-only`の`acyclic-hierarchy`に対応する拡張`tei.reference.sui-acyclic`を、工程1の対象`semantic-data:sui.island`へ後付けします。カタログのCLIで、名前空間`reference.sui.acyclic`の所有者をその時点の信頼付与で判定し、意味の固定記録と検証計画を出力し、両方を再照合します。親の不在(`parent-presence`)は拡張では埋めず、未解決のまま渡します。
3. 工程3(`runStage3`、`stage3.go`): 意味の固定記録と名前空間`sui`の意味の照合値を確かめ、所有者をその時点で判定し直します。`sui`をデータのみの配布物にし、構築計画の入力に名前空間の意味、意味の固定記録、写像報告、損失の記録、固定版の照合値を並べて固定記録(BuildArtifactLock)を作ります。構築の橋渡しで読み戻し、署名して、署名つきの前検査を通します。署名は計画の照合値を含むため、損失の記録を差し替えると署名が通りません。
4. 工程4(`runStage4`、`stage4.go`): 配布物が固定した成果物で、運ばれた意味を持つことを確かめます。利用側の作業領域(`testdata/consumer`)が提供元をパス取込みし、合成ホストがソース受理と選択と外部意味の受理を一つの決定として記録します。損失の記録は作り直さず、工程1のバイト列をダイジェストで照合します。判定(`rejudge`)は、意味(`InspectCurrentProjectMeaning`)、所有者(カタログのCLI)、署名者(`InspectPublished`と鍵の世代)、損失の記録を、名指しした時点で確かめます。決定に残る時点は履歴で、判定には使いません。
5. 通しの照合(`VerifyThrough`、`chain.go`)と完了報告(`BuildCompletionReport`、`VerifyCompletionReport`、`report.go`): 工程の記録を前の記録の照合値でつなぎ、読むだけで照合します。完了報告は照合を通った記録からだけ作ります。

## 失敗注入と合否

観測値は、下の「安定性の記録」の実行で得た値です。試験はすべて`TestThroughExternalAdoptionAuthority`の中で行い、記録(`chain-record.json`)の`faultInjections`に残ります。

| 要件 | 注入の識別子 | 注入 | 合否の観測 |
| --- | --- | --- | --- |
| P-GOV-01(工程1) | `pgov01-blob-*`(4件) | 外部側の固定表の複製で、依存blob4件を1件ずつ書き換える | 4件とも`input mismatch: SUI dependency blobs differ from the pinned version`で停止。損失の記録は書かれず、報告の出力も消える |
| P-GOV-01(工程1) | `pgov01-revision-only` | blobは変えず、リビジョンだけを進める | 停止しない。`revisionAdvanced=true`を記録し、写像報告と損失の記録の照合値は元と同じ(鮮度はblobに束縛するという`reference/sui-adoption`の規則のとおり) |
| P-INT-03(工程2) | `pint03-two-claimants-without-authority` | 同じ名前空間を名乗る拡張(`…-rival`)を加え、`-require`の順序と記述子のファイル名の順序を入れ替える | 両方の順序で終了状態1、意味の固定記録は書かれず、未解決は同じ`namespace-ownership-conflict … multiple owners: tei.reference.sui-acyclic, tei.reference.sui-acyclic-rival` |
| P-INT-03(工程2) | `pint03-two-trusted-authorities` | 二つの提供元のそれぞれに、信頼された発行者の権限の主張と有効な信頼付与を与え、主張の順序を入れ替える | 両方の順序で`conflicting authority grants`として拒否。誤り文に並ぶ二つの名前は同じ集合で、並びだけが入力の順序に従う |
| P-INT-03(工程2) | `pint03-ambiguous-trust-grant` | 同じ発行者と範囲に有効な信頼付与を二つ与え、順序を入れ替える | 両方の順序で同じ誤り文`trust grant is ambiguous`で拒否 |
| P-OPS-02(工程2) | `pops02-stage2-owner-revoked`、`…-owner-absent` | 所有者の信頼付与を失効させた時点、付与のない時点で工程2を実行する | `trust grant is revoked`、`no current trust grant`で停止し、意味の固定記録は書かれない |
| P-OPS-02(工程3) | `pops02-stage3-owner-revoked` | 工程2は時点1で適格だった所有者を、失効した時点2で工程3が判定する | `namespace owner is not qualified at trust point 2`で停止。工程2の適格性を根拠にしない |
| P-OPS-02(工程3) | `pops02-stage3-signer-revoked`、`…-signer-key-rotated`、`…-publication-expired` | 署名者の信頼付与の失効、鍵の交換(版2の鍵へ。版1の付与は失効)、署名の期限後の論理時刻 | それぞれ`Host has not approved publication signer: trust grant is revoked`、`publisher signature or Host allowance rejected`、`expired publisher binding`で拒否 |
| P-OPS-02(工程3) | `pops02-stage3-key-rotated-during-read` | 前検査の一回目の問合せは時点1、二回目は鍵交換後の時点に答える | `Host publisher trust changed during inspection`で拒否(問合せ2回) |
| P-OPS-02(工程4) | `pops02-stage4-rejudged-at-each-point` | 時点1で記録した決定を、時点1〜7で判定し直す | 成立は時点1と、所有者の付与を出し直した時点6だけ。所有者の失効・不在、署名者の失効・鍵交換、期限後では成立しない。決定そのもの(照合値)は全時点で変わらない |
| 固有(工程1〜4) | `lossless-asset-reports-false` | なし(陽性の確認) | 工程1の四つの報告がすべて`lossless=false`、`parentPresenceSemantics=unresolved` |
| 固有(工程1〜4) | `lossless-chain-rewritten-true-at-stage-2/3/4`、`lossless-chain-dropped-at-stage-4` | 通しの記録の工程2・3・4で`lossless`を`true`へ書き換え、またはなくし、記録のつながりを計算し直す(偽造者の立場) | 通しの照合が`stage N reads lossless as true`、`stage 4 does not carry lossless`で不合格(陰性対照) |
| 固有(工程1〜3) | `lossless-loss-record-rewritten` | 損失の記録のファイルを`lossless: true`へ書き換える | 運ばれた照合値との照合で不一致、書き換えた記録自身の照合値でも`reads lossless=true`で拒否。その照合値で構築計画を作り直すと計画の照合値が変わり、構築の橋渡しは元の固定記録と合わず、署名つきの前検査は署名を拒否する |
| 固有(報告) | `report-*`(4件) | 完了報告で、損失なし、現行の外部アプリへの適合、実環境での確認、採用の成立を主張させる | すべて完了報告の照合で不一致として検出 |
| 固有(工程4) | `meaning-provider-meaning-changed` | 受理した後、提供元の`IslandId`を`std.identifier`から`std.text`へ変える(利用側のソースは変えない) | 利用側のソース受理だけの照合は`ok`のまま、受理の判定は`PROJECT_MEANING_SEMANTIC_DIGEST_MISMATCH`で成立しない |
| 固有(工程4) | `meaning-provider-file-moved` | 受理した後、提供元のファイルを`semantic/sui/hierarchy/island.tei`へ移す | 意味の照合値は同じで、受理の判定は成立したまま |
| 通し | `substitution-stage1-report-from-another-revision` | 工程1の報告の1件のリビジョンを書き換えて工程2へ渡す | 工程2が`input mismatch: mapping report digest`で停止 |
| 通し | `substitution-stage3-provider-meaning-swapped` | 意味を変えた提供元を工程3へ渡す | 工程3が`input mismatch: namespace sui semantic digest`で停止 |
| 通し | `substitution-stage4-distribution-swapped` | 同じ意味で版だけ違う配布物(1.0.1)を工程4へ渡す | 工程4が`input mismatch: distribution digest`で停止 |
| 通し | `through-*`(5件) | 通しの記録から工程2を除く。工程4の成果物、工程3の意味の照合値、工程2の固定版を差し替えてつながりを計算し直す。工程3をつながりを計算し直さずに書き換える | いずれも通しの照合が不合格(`missingStages=[2]`、各項目の不一致、`does not match its digest`) |

## 通しの完了条件

`TestThroughExternalAdoptionAuthority`が、五つの条件をすべて確かめます(記録の`completionConditions`)。

- `one-pinned-version-carried-to-acceptance`: 一つの固定したSUI版(リビジョン`725e9d9c…`と依存blob4件)、入力の照合値、写像報告の照合値、損失の記録の照合値が工程1から4まで同じ値で、工程2からは名前空間の意味の照合値(`sha256:411a2b80…`)、意味の集合の照合値、意味の固定記録と検証計画の照合値、名前空間所有者が、工程3からは配布物の照合値(`sha256:d086fef2…`)、構築計画の照合値、署名者と鍵の指紋が、工程4まで同じ値です。途中の差し替えは後続の工程が入力の不一致として止まり、記録の書き換えは通しの照合で検出されます。
- `lossless-false-carried-to-completion-report`: `lossless=false`と親の不在の意味`unresolved`が、配布(構築計画の入力)、受理(外部意味の受理)、完了報告まで運ばれ、どこで`true`へ読み替えても検出されます。
- `trust-judged-at-the-judgment-point`: 所有者と署名者の適格性は、判定のたびに名指しした時点の信頼付与で判定され、失効、鍵交換、曖昧、不在は工程2〜4のどこでも適格とされません。
- `acceptance-one-to-one-and-meaning-change-refused`: 外部意味の受理は、ソース受理と同じホストの決定に、取込み1件に対して選択1件で記録されます。提供元の意味が変われば成立せず、ファイルの移動だけなら成立が保たれます。
- `completion-report-distinguishes-and-does-not-overclaim`: 完了報告(`completionReport`)は、固定版への適合(`pinned-version-and-dependency-blobs-only`)、現行の外部アプリへの適合(`not-claimed`)、損失あり、確かめ方の別(既存資産を変更せずに固定入力で呼んだもの、合成、模擬)、実環境での確認(なし)、採用の成立(なし)を分けて記し、過大な主張を加えると照合で検出されます。

## 実行

作業ブランチのコミットを浅いcloneした清浄な複製の中で実行します。

```bash
git clone --depth 1 --branch <作業ブランチ> file:///<リポジトリ> /tmp/tei-eaa
cd /tmp/tei-eaa
python3 tools/verify-external-adoption-authority.py --out /tmp/tei-eaa-receipt
```

- スクリプトは`go env GOROOT`(`GOPROXY=off`で、ツールチェーンを取得しない)から正確なGoの実行体を選び、`TEI_EAA_REPO`、`TEI_EAA_GO`、`TEI_EAA_OUT_DIR`と`GOPROXY=off`、`GOFLAGS=-mod=readonly`、`GOTOOLCHAIN=local`を与えて`go test -json -count=1`を実行します。全試験がPASSでskipが0件、記録が合格、試験の後も作業ツリーが清浄なときだけ合格です。DockerとPostgreSQLは使いません。
- `TEI_EAA_GO`がないと、通しの試験はskipします(単体試験はどの環境でも動きます)。
- 記録は`<out>/chain-record.json`、受領記録は`<out>/receipt.json`です。

## 安定性の記録

(反復の結果は下に追記します。)

## 基盤の不足

通しの実装で観測した不足と、サンプル側での扱いです。記録の`platformGaps`と同じ内容です。どれも基盤では解消していません。

| 要件 | 領域 | 不足 | サンプル側の扱い |
| --- | --- | --- | --- |
| P-GOV-01 | ツール | 影の検査は、SUIの版と依存blob4件を定数として報告へ書くだけで、外部の実物とも固定コピーとも照合しない | サンプル側で、報告の固定値を固定表の複製と照合し、blobが1件でも違えば工程1を止める。SUIの実物は取得していない |
| 固有 | コア | 関係は一対一と一対多だけで、親の不在の意味を表せない。損失を後続へ運ぶ型がない。配布物の封筒は未知の項目を拒否するため、損失の記録を載せられない | 工程1の報告から損失の記録を作り、そのダイジェストを各工程の記録と構築計画の入力に載せる |
| 固有 | プラグイン | 意味の固定記録と、配布側の成果物の固定記録を結ぶ形がない | 構築計画の入力に、意味の固定記録・写像報告・損失の記録・固定版のダイジェストを並べ、計画の照合値を署名の対象に含める。正式な接続ではない |
| 固有 | 実行時 | 受理照合はパス取込みだけを扱い、データのみの配布物を受理しない。外部意味の受理を製品のホストが記録する手段がない | パス取込みの意味の照合値を配布物の値と突き合わせる。受理は合成ホスト |
| P-OPS-02 | プラグイン | 署名つき配布の前検査は抽象のLookupだけで、名前空間所有者の判定とは別の判定。信頼付与の運用・失効の伝播がない | サンプルのLookupが署名者の鍵の世代ごとに`authoritytrust.Verify`で判定する(認可の種別だけ別)。信頼の状態は論理的な時点の列 |
| P-INT-03 | ツール | 信頼された権限の主張が二つあると両方の順序で拒否するが、誤り文の名前の並びは入力の順序に従う | 名前の集合が同じことを照合し、並びの違いは記録だけする |
| 固有 | コア | 拡張カタログのCLIは制約の評価を公開しない。評価器の成果物のダイジェストに当たる実物がない | 工程2は共有パッケージを直接呼ぶ。評価器の束縛のダイジェストは`acyclicgraph/graph.go`のダイジェストで代用する |
| 固有 | 検証根拠 | 第二の外部アプリで同じ欠落が繰り返されるかを比べる入力がない | この通しの検証では比べない |

## 範囲外

- SUIのカード移動の操作境界、SEIの判断継続、認識状態の評価、言語・演算の選択(4.9節と5章の保留)。
- 第二の外部アプリでの欠落の再現の比較。
- 実際のSUIリポジトリとの照合、実際の発行者・信頼ストア、製品の受理、サーバ間の失効の伝播。
- 拡張`sui-hierarchy-integrity`(親の存在、親子の整合など)の後付け。工程2は第二の利用者でも共有できた循環検査だけを後付けし、残りの保持範囲は`coverageNotExtended`として運びます。
