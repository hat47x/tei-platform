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
