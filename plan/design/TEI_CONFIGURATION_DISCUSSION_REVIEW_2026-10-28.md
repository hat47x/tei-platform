# TEI構成議論の再精査と次の比較（2026-10-28）

- 文書の役割：複数のIssueにまたがる設計検討の追補。現在の実装説明は[全体構成](../../docs/architecture.md)と[実装状況](../../docs/status.md)、要求と採用理由は既存Issue・ADRを参照する。
- 状態：検討用。以下の比較方法や参照記録の形を、新しいCore契約・API・採用済みADRとして定めるものではない。
- 照合版：main `21f4e986581d483b494fe5bdf363a0b4309cf8f8`。
- 対象：業務Validation、構築・候補記録、Java入力と成果物、表示・引用、認知順位付け。過去の経緯は[10月24日から26日の整理](TEI_CONFIGURATION_DISCUSSION_SYNTHESIS_2026-10-24.md)へ残す。

今回の要点は、既存の正常系を正しく取り込み、未確認の範囲を狭めることにある。三規則から試験用Commandへ進む合成経路は既に存在する。次に確かめるのは、規則が不成立のときの停止、規則を適用する行集合、成果物のどの構成ファイルまで照合したか、実際に観測した操作の範囲である。複数の成功記録を、それらが確かめていない一つの保証へまとめない。

## 前回から統合されたもの

| 根拠 | 取り込む進展 | 残る確認 |
| --- | --- | --- |
| [#2240](https://github.com/hat47x/tei-platform/pull/2240) | 過去の構成議論整理がmainへ統合された | 文書の統合は、そこに記載した案や業務意味の採用ではない |
| [#2197](https://github.com/hat47x/tei-platform/pull/2197)・[#2196の記録](https://github.com/hat47x/tei-platform/issues/2196#issuecomment-6469076926) | 開発文脈探索の参照PoCとtools索引が統合された | 実モデルの精度・遅延・必要根拠の包含率、既定利用の価値は未測定 |
| [#2254](https://github.com/hat47x/tei-platform/pull/2254) | read-onlyの最初の作業と分野間の証拠引渡しが計画へ統合された | 10/26時点のDraft状態を現在も未統合とは記さない。製品資格は別 |
| [#2313](https://github.com/hat47x/tei-platform/pull/2313)・[#2314](https://github.com/hat47x/tei-platform/pull/2314) | DDLの外部キー参照順と、以前の実PostgreSQL記録が統合された | 実DB記録は旧版`4cc6489d7`のもの。修正後にB1が解消したことの再実行証拠ではない |
| [#2315](https://github.com/hat47x/tei-platform/pull/2315) | 一プラグインの構築意図、構築・保管、自己構築、再取得byteの呼出しを工程間で照合するサンプルが統合された | 工程の一部は既存CLIの直接接続ではなくサンプル側で実現。汎用の計画入力API、遠隔保管庫、配備、OS隔離の成立へ一般化しない |
| [#2154](https://github.com/hat47x/tei-platform/pull/2154)・[#2273](https://github.com/hat47x/tei-platform/pull/2273) | Driver呼出し・確定境界とeffectcommit橋渡しの参照検証が統合された | 合成/memory経路から原典の業務対応・実DB効果・現在認可を導かない |

[#2315のPR記録](https://github.com/hat47x/tei-platform/pull/2315)は、清浄なcloneでの複数回の検証、再取得したbyteの実呼出し、失効・強制終了・不一致の検査を報告している。一方、部分出力や二回構築の不一致は模擬構築器で起こしたもので、実際のGo構築の非決定性の観測ではない。今回こちらでこれらを再実行してはいない。

#2226は引き続きOpen/Draft、#2293はOpenで未統合、#2294のGraph設計文書は統合済みと確認した。設計文書の統合をConsumerの実装・実UI受入の完了へ置き換えない。

## 三規則の合成正常系を取り込む訂正

[#1125の訂正](https://github.com/hat47x/tei-platform/issues/1125#issuecomment-6405752403)を現mainの`internal/driver/final_gate_b_scenario_test.go`（blob `7fa27d31cac4fb27f10ee977300efa1632eaf048`）へ静的に照合した。三規則の評価、認可、Pattern解決、試験用Command、合成Observationが同じ正常系にある。

このコードでは`quantity`と`lineNumber`が先頭明細、`lineNumbers`が複数明細から作られる。したがって次の四つは別に扱う。

1. 三規則から試験用Commandへ進む合成正常系の存在。
2. 同じ三規則経路で、第3規則不成立時に認可・Commandを呼ばないこと。
3. 編集対象の一明細か、提出対象の全明細かという適用範囲。
4. 正式Provider、DB効果、独立Observation、Ownerが受理した業務責務。

現在の静的根拠は1を支える。2〜4を同時に実証したとはしない。判別入力の案は、同じ二明細の片方だけ行番号を0にして順序を入れ替えること。先頭行・選択行・全行のどの範囲を評価するかを見分ける。今回の実行結果ではなく、担当へ渡す未実行の比較案である。

[横断整理の訂正を#1125へ追記](https://github.com/hat47x/tei-platform/issues/1125#issuecomment-6470719087)した。過去に未発見だった事実は履歴として残し、現在も合成経路が無いという含意は維持しない。

## 成果物の本体と、それを使う範囲

[Java担当の反例](https://github.com/hat47x/tei-platform/issues/2077#issuecomment-6407878151)では、同じJSONでも重複キーやトークン型の解釈により、JavaとGoのAdapterが異なる存在状態・数値を作る可能性が指摘された。これは旧照合版に対する静的報告と未実行の予測であり、計算関数の非同値を実測したものではない。Adapterで同じ型付き入力へ到達した範囲を先に固定して、独立期待値と計算を比較する。

同じ記録では、`Main.class`のdigest一致が`Main$Input.class`等を含むclass集合の一致ではない点も示された。Javaの最初の独立Consumerでは、実際のclasspath、classの相対pathとbyte、JDK条件を記録する案を問い直す。JAR配布の完成は、この小さな呼出しの先行条件にしない。

これは[SHOの表示依存](https://github.com/hat47x/tei-platform/issues/1803#issuecomment-6405713981)とも共通する。Markdown本体Dが同じでも参照SVGのEが変われば、画面の再現は別の確認になる。引用本体、今回の表示で読む資源、人の説明の根拠を分け、未探索と取得不能を混ぜない。汎用の依存閉包エンジンを先に作る要求にはしない。

[引用の根拠に関する追補](https://github.com/hat47x/tei-platform/issues/1803#issuecomment-6405740102)は、manifestのUnsupported/NotesやartifactのUnresolvedが変わる場合も区別する。本体の再発見が成功しても、説明の前提が現在も同じとは限らない。旧manifestの原文が無ければ、hash差だけから「引用外追加のみ」と理由を確定しない。

## 表示・観測・副作用を分けて再開する

[Graphの構造と表示](https://github.com/hat47x/tei-platform/issues/982#issuecomment-6405710330)では、入力に別IDの多重辺を保持できても、既存SVGの直線が重なり、向きや多重性の視認性は未成立と報告された。最初の案は、Graphと辺一覧のread-only詳細でID、向き、関係を確かめること。表示ラベルから意味やscopeを補わず、元Graphとの対応をConsumer局所に保つ。

[Source変更後の表示失効](https://github.com/hat47x/tei-platform/issues/1124#issuecomment-6413492794)は、旧応答の後着拒否だけでなく、既に表示した古い評価も別に失効させる案である。現行APIがreadyを返すと主張するものではなく、将来の受入入力としての検討である。[未呼出しと判定済みの区別](https://github.com/hat47x/tei-platform/issues/1124#issuecomment-6413575758)も保つ。guardで止めてSourceBridgeを呼ばなかったなら、そのBridgeが曖昧診断を返したとは表示しない。

[Accept後にDecision記録が失敗する表示](https://github.com/hat47x/tei-platform/issues/1124#issuecomment-6467978094)では、非ゼロ終了と有効なaccepted結果、Source変更、ledger未保存を別々に扱う。現行read-only UIがAcceptを実装済みとはしない。将来のConsumerに、Source不変と決めつけた無条件の再Acceptを促さない表示と再読取りを求める案である。

[Record回復の検討](https://github.com/hat47x/tei-platform/issues/988#issuecomment-6405734635)でも、四ファイルが揃うことと凍結入力が一致することは別である。Freeze時C1に対しRecordがC2を読んだなら、保存成功だけで同じ試行の来歴を確定しない。元dirを保持して読み取り照合し、外部操作のACK不明をこの局所回復で解消したことにはしない。

## 認知順位と実Harnessの証拠

#2197のmain統合後も、`Rerank`のrequestはpath・SHA・候補metadataで、本文や変更意図を含まない。現mainの同関数を確認し、blobは以前と同じ`dc69dc3a204085801ca509bdda2e2e27b91fa204`だった。

[評価の独立性に関する追補](https://github.com/hat47x/tei-platform/issues/2196#issuecomment-6405773963)は、baselineRankの入力と同じReportの再利用を区別する。現行CLIの一回呼出しを不具合と決めず、比較では毎回新しいReportと同じ候補集合を使う。pathだけの順位、kind/basisも使う順位、同じmetadataでのモデル順位を分け、入力情報の差をモデル効果へ混ぜない。

[SEI担当の訂正](https://github.com/hat47x/tei-platform/issues/2021#issuecomment-6411220296)も、三規則正常系の新証拠で支持された主張だけを更新する案である。以前の未発見というMethod履歴は保つが、それを現在の不存在判定に使わない。

[実HarnessのWitness案](https://github.com/hat47x/tei-platform/issues/2016#issuecomment-6411212892)では、用意した入力、実際に渡した入力、生成されたファイル、Recordした入力、対象候補の検証を別の辺として照合する。Context Aを渡してBをRecordしたなら、Bが渡された証拠にはならない。CLI終了や最終メッセージ、provider/modelのcaller申告も、実ファイル・実観測・現在資格と分ける。ここで述べるのは証拠の対応付けの案であり、特定CLIの運用設定を検証した報告ではない。

## 各担当へ送る判別質問

| 担当チャット | 今回絞る問い |
| --- | --- |
| TEI設計への取り込み検討 | 二明細の並べ替えで、先頭行・選択行・全行の評価を見分ける |
| TEI Go多言語対応調査 | 主classと実行class集合、Adapterと型付き計算の比較範囲 |
| TEI Desktop仕様 | 表示済み値の失効、旧応答の後着、Acceptと記録失敗 |
| TEIデータ様式 | 現rendererの限界を辺一覧で補うread-onlyの最小表示 |
| TEI認知基盤構成検討 | 新鮮なReportと固定入力で順位の寄与を比較する |
| TEIプラグイン管理整理 | Record保存完了と凍結入力一致を区別する |
| TEI SHO命名とMMI再定義 | 本体一致と説明の前提変更を分ける |
| TEIハーネス#3 | 新証拠で支持された主張だけを更新する |
| TEIハーネス#4 | DeliveredとRecordedの因果的な対応 |
| TEIハーネス#7 | Markdown本体と表示に必要な画像の再開 |

質問の受付、回答取得、Issueへの保存、設計採用は別に記録する。今回の照合はGitHub記録と公開コードの静的確認であり、Go/Java/モデル/DB/実UIの試験を再実行していない。非公開原典の固有情報は転載せず、意味・権限・本番配備の採用判断は既存担当へ残す。


## 2026-10-29追補：設計書から評価へ進むための条件

照合mainは引き続き `21f4e986581d483b494fe5bdf363a0b4309cf8f8`。前回質問への応答だけでなく、各チャットで独立して進んだ設計書中心の構成案、形式検証、Oracle、知識の確認履歴も読み取った。以下は検討の整理であり、製品要求やADRの採用変更ではない。

### Graphの同一性と評価条件

[#982のRule集合・順序の追補](https://github.com/hat47x/tei-platform/issues/982#issuecomment-6472383027)では、同一Graphでも評価Ruleの違いで診断や影響候補が変わり得ると整理された。現mainの [modelref/model.go](https://github.com/hat47x/tei-platform/blob/21f4e986581d483b494fe5bdf363a0b4309cf8f8/internal/modelref/model.go)（blob `4de4beb2db3a03720f653deb6eb44d2bb8275893`）を静的に再照合した。

- `Validate`は同じRuleキーの競合を診断し、後着の異なるRuleを採用しない。補助的な循環診断は、先に保持したRuleの条件に左右され得る。
- `Impact`は競合したRuleを経路に使用しない。空の返却だけでは、評価前提が成立して影響がなかったのか、競合等で経路を採用できなかったのか分からない。
- `Diff`はNodeだけでなく辺構造と辺のEvidence変更も検出する。一方、Rule集合は引数に含まれない。**辺差分が未対応という整理は誤りであり、Rule評価条件がGraph差分の外にあることが今回の論点である。**

したがって、Consumerが評価結果を再利用するなら、元Graph、評価へ渡したRuleと順序、変更集合、診断を対応付ける案を検討する。「影響0件」を無条件に再確認不要へ変換しない。新しいRule正本やCore契約は作らず、一枚のread-only設計書で、元情報・描画範囲・評価不能を読者が区別できるかを先に確かめる。

### 形式検証と独立Oracleの役割

「TEI Lean活用のメリット」では、人間が読める設計表現から評価・実行へ進む構成案が深まった。この方針は今回採用した事実ではなく、設計候補として取り込む。

次の比較は、全明細の数量正値という一つの命題を、設計文、形式化、対象集合、仮定、証明対象へ対応付けるものとする。先頭行へ射影した後の証明では、全明細への適用を保証できない。証明が成立しても、設計文から命題への翻訳、対象集合の完全性、実Providerの鮮度、業務Ownerの受理は別に残る。既存Goの判別試験と比べてLeanが何を追加できるかを問い、導入そのものを完了条件にしない。

[#1063の既存handwritten負例案](https://github.com/hat47x/tei-platform/issues/1063#issuecomment-6471935105)では、half-even契約と旧half-up fixtureの組合せを、固定Generatorの拒否とは分けて扱う。#2291の対応済み問題を再提案せず、fixtureに合格することと契約意味への適合を区別する。同一候補に二種類のfixtureを与える比較と、正しい実装が誤fixtureで落ちる場合の扱いを整理する。いずれも今回実行していない。Adapter入力状態が違う比較を、丸め演算の非同値へ混ぜない。

### 局所評価と業務完了

「TEI汎用フレームワーク脱却構想」では、問いごとに必要な意味情報が異なると整理された。通貨未決でも数量規則を評価できる場合がある一方、数量検証成功から業務金額や提出可能性は導けない。通貨未決・一方の明細だけ数量0という一入力に対し、数量規則、純粋な数値積、業務金額、提出可否の四問を別々に記す比較案を依頼した。

[#1125の行順判別案](https://github.com/hat47x/tei-platform/issues/1125#issuecomment-6471589471)も取り込む。実Reference SQLには行番号等のORDER BYがあるため、同じ二行の順序反転は実DBで自然に得られる結果ではなく、Executor境界の合成攪乱入力である。SQLの順序・完全性等の前提が揃う場合の行番号判定と、全明細の数量判定は別である。三規則の正常系、負例での後続停止、適用対象集合、実Providerの証拠を一つへまとめない。

### 確認履歴と現在用途への適用

「TEI-EKI接続」では、確認操作時の古いEvidence拒否と、確認履歴を読み取って現在用途へ再利用する評価が別であるという所見が進んだ。これは同チャットのコード調査結果として受領したもので、今回EKIリポジトリを独立検証した結果ではない。

次の問いは、同じ命題・対象・文脈でEvidenceだけE1からE2へ変わる場合に、参考資料なのか成立に必要な根拠なのかを区別できるかである。依存の意味が未宣言なら、過去の確認履歴を消したり、現在の業務有効性を自動で認めたりせず、現在用途への適用が未確認であると示す案を検討する。署名Witnessの検証は、確認内容の理解や現在の業務適用を代行しない。TEI以外の製品への変更はこの追補の範囲に含めない。

### Payload検証と確定した効果

「TEI構成検討」の新所見は、結果ModeだけでなくPayloadと実効果の整合を分けることである。現mainの [capabilityinvoke/invocation.go](https://github.com/hat47x/tei-platform/blob/21f4e986581d483b494fe5bdf363a0b4309cf8f8/internal/semantic/capabilityinvoke/invocation.go)（blob `413ce5ba1ab0e369cd2211956356cc67d1a73628`）には、Payloadの具体的な値表現・型変換はAdapterの責務と明記されている。従って、汎用関数に型検査がないという事実だけを欠陥と判定しない。具体的Adapterの検査範囲と実際に不足する証拠を先に確認する。

[既存の確定後Result不正の比較](https://github.com/hat47x/tei-platform/issues/1174#issuecomment-6471937593)と、mainの `TestDriverRejectsPostCommitInvalidBusinessResultWithoutErasingEffect` の存在も静的に確認した。これはmemory Providerで確定後に不正Modeを返す合成負例であり、既にある試験を再実装する必要はない。正常Modeで誤Payloadの場合、Commit前に検査できる内容、確定後のReceipt・独立Observationとの整合、Ownerの完了条件を分ける。検出を強化しても確定済み効果が消えることにはならない。

### 追加質問と回収状態

| チャット | 新たに絞った問い |
| --- | --- |
| TEIデータ様式 | Rule競合時の診断とImpact空結果、Graph差分と評価条件 |
| TEI Lean活用のメリット | 設計文から証明への対応、全明細の完全性と証明対象外 |
| TEIテスト設計 | 誤fixtureによる偽陽性・偽陰性と独立期待値 |
| TEI汎用フレームワーク脱却構想 | 同一入力への四問で局所評価と未決を分ける |
| TEI-EKI接続 | Evidence変更後の履歴と現在用途への適用 |
| TEI構成検討 | AdapterのPayload責務と確定効果の整合 |

送信受付と回答本文の取得は別に扱う。取得制限やタイムアウトは未確認として配信台帳へ残す。今回は公開ソースの静的照合と議論整理であり、Lean/Go/DB/UI試験を新規実行していない。非公開原典の詳細は転載せず、ADRの採用状態も変更しない。


## 2026-10-29第2追補：回答の回収と実装進展の取り込み

今回の照合版はmain `4fde7e46e73cb0f0eaa6694d655662d094842bb3`。上の追補の照合版とは区別する。チャット本文の取得がタイムアウトした範囲は、担当が既存Issueへ保存した読取り可能な回答で補った。受付・本文取得・Issue保存・設計採用を別々に記録する。

### 前回の未確認をどう整理したか

[Graph担当の回答](https://github.com/hat47x/tei-platform/issues/982#issuecomment-6476555702)は、前回のDiff前提の訂正を明示的に参照している。したがって、訂正メッセージ自体の受付は未確認の履歴として残しつつ、**訂正内容が担当の回答へ反映されたこと**は確認できた。訂正の再送や同じ反例の再作成は不要である。

「TEI汎用フレームワーク脱却構想」への四問の依頼は、前回の送信台帳では未確認のまま。既存回答の不存在とは断定せず、再確認する場合は受領済みなら参照だけを返すように依頼する。本文を取得できなかった他のチャットへは、受付済みの同一依頼を繰り返さない。

### 候補が見つかったことと網羅できたこと

[#982の追加反例](https://github.com/hat47x/tei-platform/issues/982#issuecomment-6476664267)は、影響候補が正数でも網羅性が未確認となる場合を具体化した。非競合の関係からN3が候補になっても、競合して経路へ採用できなかった別関係が残る。表示案は「候補N3 1件／競合関係の影響は未判定」であり、「N3だけが影響対象」とはしない。

次の問いは、この区別をConsumerがどの観測から説明できるかである。既存評価器の戻り値、Validateの診断、Consumerが保持する評価条件を分け、全経路の被覆追跡を実装済みとしない。Ruleだけが変わった場合はGraph差分が空でも評価条件は変わるため、過去の評価表示の再照合を検討する。これは表示・再利用条件の案であり、新しいRule正本や変更種別の採用ではない。

### 条件付き証明から対象集合の欠落へ

[Lean担当の回答](https://github.com/hat47x/tei-platform/issues/1063#issuecomment-6475396003)は、有限リスト全件検査の性質と、実Providerが返した集合の完全性を分けた。全明細命題は `P(L) := ∀ l∈L, q(l)>0`、返却リストRを検査する形式モデルの正しさは、そのモデル内の性質である。日本語からの翻訳、Rと業務対象Lの一致、鮮度、Ownerの適用範囲、生成実装の忠実性は証明対象外としている。Leanの実装・実行はまだない。

次の判別例は、Lに正値のAと数量0のBがあるのに、RがAだけを返す場合である。Rの全件検査が成立してもLの業務命題は成立しない。空集合・NULL・未取得の意味も未受理のまま保持する。形式モデルの定理を追加することと、実データ取得の欠落を検出することを別の確認へ戻す。

### 検査Source、ビルド入力、保存後の実行

[ハーネス#4のS4-01/S4-02](https://github.com/hat47x/tei-platform/issues/2016#issuecomment-6476652969)を読み、現照合版の `selection.go` が引き続きblob `61c8f3b0d25c1a462e346c372c38e72f3184efa5` であることを確認した。初回Sourceの読取り・digest検査後に、二回のBuild用コピーが元directoryから別々に読み取る。二回Buildの一致は、初回検査Sourceとの一致を自動では示さない。静的な競合仮説であり、誤PASSの実測ではない。

次は単一Go候補のS4-01へ絞り、試験用の同期点で差替え時点を固定し、初回検査D1・コピー1・コピー2・実行Binaryを別々に観測する案を問う。試験用同期点を製品hook/APIへ一般化しない。Oracleが正しくても評価対象が違えば、その候補の適合証拠として使えない。

[Java担当の新しいStore所見](https://github.com/hat47x/tei-platform/issues/2077#issuecomment-6482440631)は、保存成功と復元実行の閉包を分ける。`store.go` は現照合版でもblob `be9f5bfffa2931fea61e92abe660074cefdc00a4`。主classだけの保存・取得のbyte整合を、関連class一式が必要なConsumer起動の成功と同一視しない。次は全classの直接起動と、Put/Getで戻した保存物だけの隔離起動を対照にする。既存classesや外部classpathから不足分が紛れ込む条件も分ける。同SourceでBuild条件が異なる保存衝突は、拒否が既存物を保護する効用と、複数の実行物を保持できる範囲を別に検討する。新しいStore schemaの採用はまだ行わない。

### 前回質問以外に進んだ実装と文書

| 確認先 | 今回取り込む進展 | 現在も別に残る条件 |
| --- | --- | --- |
| [#2324](https://github.com/hat47x/tei-platform/pull/2324)、[#1127の統合記録](https://github.com/hat47x/tei-platform/issues/1127#issuecomment-6484512293) | Web Targetにopt-inのRunner StoreとLookupSubmitが統合。実PGの試験記録がある | 出荷CLIの既定採用、真正なgrant、試験用Owner処理から正式業務への対応、Outcome写像は未決。実PG結果は既存記録であり今回の再実行ではない |
| [#2348の文書反映](https://github.com/hat47x/tei-platform/issues/1124#issuecomment-6486575297) | Accept後のledger失敗と旧Inspect後着の結合例がStudio鮮度設計へ統合 | 設計文書の統合。Consumer/GUI実装やOwner受入ではない |
| [#2372](https://github.com/hat47x/tei-platform/pull/2372)・[技術検証サンプルの現状](https://github.com/hat47x/tei-platform/blob/4fde7e46e73cb0f0eaa6694d655662d094842bb3/plan/design/TECH_VALIDATION_SAMPLES_2026-10.md) | 十サンプルの限定的な通し検証の統合状況を文書へ反映。#2327多言語運用、#2349外部採用、#2362規則設計の進展も含む | order-workflowは未統合。4・5は拡張工程だけ。実環境の確認や設計全体の採用とは別で、Java StoreやGenerator競合の証拠にも代用しない |
| [#2328](https://github.com/hat47x/tei-platform/pull/2328) | validateのJSON結果に読込み版fingerprintを追加 | 結果をある読込み版へ対応付けることであり、外部変更の常時検出や実Sourceの原子的snapshotではない |

Runner Storeの実装 `runner_order_submit_store.go`（blob `5e15ba8bbbd45dfd4615fd758d60e1af77e4a3c7`）も静的に確認した。これにより「直接SQLしか存在しない」という旧時点の説明を、現在の全体結論には使わない。一方、現在状態がsubmittedであることと、自要求の確定Receiptを取得できたことは別である。Commit応答喪失後のLookupと再Submitのnot-submittableを、同一OperationID・要求同一性・現在認可に沿って比較する問いを構成担当へ返す。既存の実DB試験の再実装は求めない。

### 今回の配信対象と範囲

| チャット | 追加質問・回収事項 |
| --- | --- |
| TEIデータ様式 | 非空Impactと未判定経路、Rule変更時の過去表示 |
| TEIハーネス#4 | S4-01を決定的に判別する同期点と実入力の観測 |
| TEI Lean活用のメリット | Providerが対象明細を欠落させた場合と条件付き定理 |
| TEI Go多言語対応調査 | Java Store取得物だけによる隔離実行 |
| TEI構成検討 | opt-in Runner統合後のLookupと再Submit、旧設計の更新範囲 |
| TEI汎用フレームワーク脱却構想 | 前回未確認の四問の回収。回答済みなら再作業不要 |

この表は問いの整理であり、全件送信や回答完了を示すものではない。受付と失敗は別の配信記録へ残す。今回は公開コード・Issue・文書の静的確認のみ。Go/Java/Lean/DB/UI、競合再現、正式なOwner受理を新規実行していない。


第2追補の配信結果：最新一覧で対象6件を照合し、3件ずつ送信を試みたが、全件が本文読込みタイムアウトで受付未確認となった。前回の5/6受付とは別のラウンドであり、今回は全件配信済み・追加回答回収済みとは扱わない。重複防止のため同一質問の連続再送はせず、質問と失敗結果を配信台帳へ保存した。
