# 外部アプリの契約を意味として取り込み、案件で受理するまで(`external-adoption-authority`)

取込み担当者が、別のリポジトリのアプリSUIが持つ島階層の契約を固定した版のまま意味定義へ写し、拡張として制約を後付けし、意味の配布物として固定・取込みしたうえで、案件の受理担当者が受理するまでを一つの流れで行う技術検証用サンプルです。確かめたいのは、固定した外部版から受理記録まで同じ識別子と照合値が引き継がれ、写せなかった意味を示す損失ありの記録(`lossless=false`)が最後まで保たれるか、という点です。

- 状態: 設計案は提案段階です。通しの流れは[`tools/reference-external-adoption-authority`](../../tools/reference-external-adoption-authority/README.md)で、既存資産を変更せずに固定入力の複製で呼び、合成の信頼付与・署名者の鍵・ホストと論理時刻を使って実装し、試験しました。外部側(SUI)は固定表の複製による模擬で、実際のSUIリポジトリ、発行者、信頼ストア、製品の受理では確かめていません。各工程の実装、試験、実環境での確認の状況は、下の工程表で工程ごとに区別して示します。
- 設計の正本: [技術検証用サンプルの統合設計 4.9節](../../plan/design/TECH_VALIDATION_SAMPLES_2026-10.md#49-外部アプリの契約を意味として取り込み案件で受理するまでexternal-adoption-authority)。技術検証用サンプルの位置づけ、物語、受け渡しの詳細はそちらを参照してください。
- 各資産の説明と使い方は、それぞれのREADMEが正本です。この入口は工程との対応だけを示します。

## 工程

現状の区分は、[`tools/reference-external-adoption-authority`](../../tools/reference-external-adoption-authority/README.md)の通しの試験(`TestThroughExternalAdoptionAuthority`)の結果です。既存資産を呼べたか、呼べなかった理由は、そのREADMEの「既存資産を呼べるか」にあります。工程間は、版・照合値・識別子を前の記録の照合値でつないだ一つの記録(`chain-record.json`)で引き継ぎます。この記録の形はサンプル固有で、コアへ移す内容は決めていません。

| 工程 | 内容 | 前工程から受け取るもの | 使う既存資産 | 現状 |
| --- | --- | --- | --- | --- |
| 1 | 外部契約の意味への写像(影の検査) | なし(最初の工程)。SUI側で固定した版と依存blob4件、同じ意味で固定した入力 | [`reference/sui-adoption`](../sui-adoption/README.md)(固定入力。検査器`tools/reference-sui-adoption`の説明を含む)、[`tools/reference-sui-adoption`](../../tools/reference-sui-adoption/) | CLIを変更せずに実行して試験。外部側との照合はサンプル側(固定表の複製との照合) |
| 2 | 拡張の後付け、名前空間所有者の判定、意味の固定 | 工程1の名前空間と、制約の保持範囲の項目 | [`tools/reference-semantic-extension-catalog`](../../tools/reference-semantic-extension-catalog/README.md)、[`tools/reference-authority-evidence-trust`](../../tools/reference-authority-evidence-trust/)(工程2から4で共用する判定部品) | カタログのCLIと判定部品を変更せずに呼んで試験。信頼付与と権限の主張は合成 |
| 3 | 意味配布物の固定、署名の前検査、取込み | 工程2の名前空間の意味と意味の照合値、名前空間所有者の判定結果 | [`reference/semantic-distribution`](../semantic-distribution/README.md)、[`reference/semantic-build-bridge`](../semantic-build-bridge/README.md)、[`reference/semantic-composition-preflight`](../semantic-composition-preflight/README.md) | 三つの資産を変更せずに呼んで試験。署名者の鍵と信頼付与は合成、時刻は論理時刻 |
| 4 | 案件での受理と、現時点での再判定 | 工程3の型付き参照と意味の照合値、署名と名前空間所有者の指紋、工程1から運ばれた`lossless=false` | [`reference/semantic-composition-preflight`](../semantic-composition-preflight/README.md)(受理照合の互換入口) | 受理照合を変更せずに合成ホストで呼んで試験。パス取込みだけを扱うため、配布物との対応はサンプル側で照合 |

工程ごとの補足です。

- 工程1: 影の検査のCLIは固定版と依存blobを定数として報告へ書くだけです。報告の値を`reference/sui-adoption/README.md`の固定表の複製(外部側の模擬)と照合して止めるのはサンプル側で、SUIの実物とは照合していません。
- 工程2: 後付けしたのは循環検査(`tei.reference.sui-acyclic`)だけです。親の不在の意味は拡張でも埋めず、未解決のまま運びます。制約の評価はカタログのCLIに出ていないため、共有パッケージ(`hierarchy`、`acyclicgraph`)を直接呼んでいます。
- 工程3: 配布物は検証用の模擬配布物で、署名も検証用の形式です。意味の固定記録と配布物の固定記録は、構築計画の入力にダイジェストを並べる形でサンプル側が結んでいます。
- 工程4: 既存の受理照合(`InspectCurrentProjectMeaning`)はパス取込みだけを扱い、データのみの配布物を受理しません。利用側はパス取込みで提供元の複製を読み、その意味の照合値が配布物の値と同じことをサンプル側で照合します。受理は試験プロセス内の合成ホストで、製品の受理ではありません。

## 取り込むピーキーな要件と失敗注入

要件IDの意味は、設計文書の[2章 ピーキーな要件の目録](../../plan/design/TECH_VALIDATION_SAMPLES_2026-10.md#2-ピーキーな要件の目録)にあります。「固有」は目録にない、このサンプル固有の困難です。保つものと確かめ方の詳細は、4.9節の表を参照してください。どの注入も試験しました。注入の識別子と観測値は[`tools/reference-external-adoption-authority`](../../tools/reference-external-adoption-authority/README.md)の「失敗注入と合否」にあります。

| 要件 | 工程 | 注入する失敗 | 観測する合否 |
| --- | --- | --- | --- |
| P-GOV-01 | 1 | SUIの依存blob4件のうち1件が外部リポジトリ側で変わる | 工程1の報告が入力の不一致として止まる |
| P-INT-03 | 2 | 同じ名前空間を二つの提供元が自分のものと主張する | 曖昧として適格とされず、列挙順で一方を選ばない |
| P-OPS-02 | 2〜4 | 署名者の鍵が失効・交換される、または名前空間所有者の信頼付与が失効する | 判定時点で有効な信頼付与で判定し、過去の適格性を根拠にしない |
| 固有 | 1〜4 | 親の不在の意味を表せないまま「損失なし」と報告される | `lossless=false`が配布、受理、完了報告まで読み替えられずに運ばれる |
| 固有 | 4 | 利用側のソース受理は変わらないまま、提供元の名前空間の意味だけが変わる | 旧い受理に基づく照合が拒否され、ファイルの移動だけなら成立が保たれる |

## 通しの完了条件

次の五つは、[`tools/reference-external-adoption-authority`](../../tools/reference-external-adoption-authority/README.md)の`TestThroughExternalAdoptionAuthority`で成立を確かめました。実行方法は、そのREADMEの「実行」にあります(`python3 tools/verify-external-adoption-authority.py`を清浄な複製で実行)。

- 一つの固定したSUI版から、写像報告、意味の固定記録、配布物、受理記録まで、同じ識別子と照合値が引き継がれる。途中で版や配布物を差し替えると、後続の工程が入力の不一致として止まる。
- 写像で表せなかった意味は`lossless=false`として最後まで運ばれ、どの段階でも「損失なし」へ読み替えられない。
- 署名者と名前空間所有者の適格性は判定時点で有効な信頼付与で判定され、失効、鍵交換、曖昧、不在は工程2から4のどこでも適格とされない。
- 外部意味の受理は既存のソース受理と同じホスト判断で一対一に記録され、提供元の意味が変われば旧い受理は成立しない。
- 完了報告は、固定した版への適合、損失あり、合成データや模擬による確認、実環境での確認を区別して記し、現行の外部アプリへの適合を主張しない。

成立の範囲には限りがあります。外部側は固定表の複製による模擬で、信頼付与、署名者の鍵、信頼の時点、受理のホストは合成です。失効は一つのプロセスの中の時点の列で判定し、サーバ間の伝播は扱っていません。外部意味の受理は、既存の受理照合がパス取込みだけを扱うため、パス取込みの意味の照合値と配布物の意味の照合値の一致をサンプル側で結んでいます。

## 部分工程の診断(互換入口)

次の資産は、このサンプルの部分工程の単独の診断や共有の部品として、現在の位置と入口のまま残します。実行方法は各READMEを参照してください。

| 資産 | 対応する工程 | 役割 |
| --- | --- | --- |
| [`tools/reference-sui-adoption`](../../tools/reference-sui-adoption/)(実行方法は[`reference/sui-adoption`](../sui-adoption/README.md)) | 1 | 影の検査。階層検査の共有パッケージも提供する |
| [`tools/reference-authority-evidence-trust`](../../tools/reference-authority-evidence-trust/) | 2〜4 | 信頼付与の判定部品。他のツールが取り込む |
| [`reference/semantic-composition-preflight`](../semantic-composition-preflight/README.md) | 3・4 | 署名の前検査と受理照合 |

## 基盤への改善圧力

見出しだけを示します。現状、求める条件、確認方法は、4.9節の「基盤への改善圧力と現状」を参照してください。

- P-GOV-01: 外部版の固定を取得時に照合する手段がない(ツール)
- 固有: 写像の損失と未表現の制約を表す手段が足りない(コア)
- 固有: 二つの固定記録と配布の正式な形が接続されていない(プラグイン)
- 固有: 外部意味の案件での受理が未実装(実行時)
- P-OPS-02: 信頼付与の判定が二系統あり、運用が未実装(プラグイン)
- 固有: 第二の外部アプリで欠落が再現するかを比べる手段が限られる(検証根拠)

通しの実装で観測した不足と、サンプル側での扱いは、[`tools/reference-external-adoption-authority`](../../tools/reference-external-adoption-authority/README.md)の「基盤の不足」にあります。どの不足も、サンプル側の実装で通しを成立させただけで、基盤では解消していません。たとえば外部版の照合は固定表の複製との照合で、取得時の照合ではありません。署名者と名前空間所有者は同じ`authoritytrust.Verify`で判定していますが、それはサンプルが前検査のLookupを実装した結果で、前検査の判定自体は二系統のままです。通しの照合(`VerifyThrough`)と損失の記録もサンプル固有です。
