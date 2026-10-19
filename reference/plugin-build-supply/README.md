# プラグインの構築意図から自己構築・成果物の固定まで(`plugin-build-supply`)

プラグイン作成者が、構築意図の宣言から構築器の解決、構築の試行の証跡化、成果物の保管、構築器自身の自己構築、構築済みプロセスプラグインの呼出しまでを一つの流れで行う技術検証用サンプルです。確かめたいのは、工程1で確定した構築計画の照合値と成果物のダイジェストが全工程で同一に引き継がれるか、失敗や一回きりの成功を再現性の成立と取り違えないか、という点です。

- 状態: 設計案は提案段階です。このREADMEは入口であり、通しの流れは実装していません。各工程の実装、試験、実環境での確認の状況は、下の工程表で工程ごとに区別して示します。
- 設計の正本: [技術検証用サンプルの統合設計 4.7節](../../plan/design/TECH_VALIDATION_SAMPLES_2026-10.md#47-プラグインの構築意図から自己構築成果物の固定までplugin-build-supply)。技術検証用サンプルの位置づけ、物語、受け渡しの詳細はそちらを参照してください。
- 各資産の説明と使い方は、それぞれのREADMEが正本です。この入口は工程との対応だけを示します。

## 工程

現状の区分は、各資産のREADMEの記述から判断したものです。この入口を書いた時点では、どの試験も実行していません。

| 工程 | 内容 | 前工程から受け取るもの | 使う既存資産 | 現状 |
| --- | --- | --- | --- | --- |
| 1 | 構築意図の宣言と構築器の契約の解決 | なし(最初の工程)。作成者が書いた宣言と、ソースとカタログの観測値 | [`tools/reference-build-authoring-intent`](../../tools/reference-build-authoring-intent/README.md)、[`tools/reference-builder-contract-resolution`](../../tools/reference-builder-contract-resolution/README.md) | 合成データで試験 |
| 2 | 構築の試行の証跡化と成果物の保管 | 工程1の確定計画、構築器の識別、解決の固定記録 | [`tools/reference-build-attempt-evidence`](../../tools/reference-build-attempt-evidence/README.md)、[`tools/reference-build-artifact-store`](../../tools/reference-build-artifact-store/README.md) | 合成データで試験 |
| 3 | 構築器の自己構築と、二回構築・段階の収束・ホスト登録 | 工程1・2の計画と証跡。ただし現在の実行器は計画を自分で組み立て、工程1・2の参照実装を使わない | [`reference/plugin-self-build`](../plugin-self-build/README.md)(入力の正本と実行器の説明)、[`tools/reference-plugin-self-build`](../../tools/reference-plugin-self-build/)、[`tools/reference-native-plugin-builder`](../../tools/reference-native-plugin-builder/)、[`tools/reference-plugin-package-builder`](../../tools/reference-plugin-package-builder/)、[`tools/reference-transform-runtime-engine`](../../tools/reference-transform-runtime-engine/README.md)、[`tools/reference-transform-seed-builder`](../../tools/reference-transform-seed-builder/) | 実装あり |
| 4 | 構築済みプロセスプラグインの呼出し確認と、識別の通し照合 | 工程3の成果物と固定情報、工程1の計画の照合値、工程2の保管庫から再取得したバイト列 | [`tools/reference-process-plugin`](../../tools/reference-process-plugin/)(構築対象かつ呼出し対象) | 未確認 |

工程ごとの補足です。

- 工程3: `reference/plugin-self-build`のREADMEは、実行器を候補の実装とし、実際の構築の検証根拠や段階の収束の成立は後続としています。そのため「実装あり」にとどめます。
- 工程4: 自己構築の既存の試験が、構築済みの成果物を旧通信方式で実際に呼んでいるかは、READMEから確かめられません。識別を突き合わせる通しの照合は、まだ足場がありません。

## 取り込むピーキーな要件と失敗注入

要件IDの意味は、設計文書の[2章 ピーキーな要件の目録](../../plan/design/TECH_VALIDATION_SAMPLES_2026-10.md#2-ピーキーな要件の目録)にあります。「固有」は目録にない、このサンプル固有の困難です。保つものと確かめ方の詳細は、4.7節の表を参照してください。

| 要件 | 工程 | 注入する失敗 | 観測する合否 |
| --- | --- | --- | --- |
| P-GOV-01 | 1 | 提供元が複数あって曖昧になる。解決後に提供元が増える、または信頼が失効する | 列挙順で選ばずに停止し、固定記録が変わらず、失効後の利用が拒否される |
| P-JOB-01 | 2・3 | 構築の失敗で部分的な出力が残る。二回構築が一致しない。一回しかない実行を二回と主張する | 失敗や不一致の記録から、成功や再現性が導かれない |
| P-JOB-01 | 2 | 同じ確定計画で二つの試行を同時に起動し、一方を強制終了する | 試行ごとに別の発生識別子で証跡が残り、同じ試行を二度数えない |
| 固有 | 3 | ソースやツールチェーンの版の食い違い、汚れた作業ツリー、暗黙のキャッシュの混入 | 閉包を固定できなければ停止する |
| 固有 | 1〜4 | 工程間で受け渡す識別が食い違ったまま、各工程の単独確認だけが通る | 計画の照合値と成果物のダイジェストが全工程で同一でなければ、通しとして不合格になる |

## 通しの完了条件

現在、通しの完了条件はどれも成立していません。工程1・2が工程3と接続されておらず、通しの照合もないためです。

- 工程1で確定した計画の照合値が、工程2の証跡、工程3の検証根拠と固定情報まで同じ値で引き継がれる。途中で計画や識別を再用意しない。
- 保管庫から再取得した成果物のダイジェストが、二回構築のダイジェストとホスト登録時のダイジェストと同一である。
- 構築器自身の構築から配布一式とTEI-native実現の生成までが、同じエンジンの識別と構築器のダイジェストを計画に束縛したまま、段階の収束まで通る。
- 構築済みのプロセスプラグインを旧通信方式で呼べ、その成果物が工程1から3の識別と同一である。どの工程かが欠けたら、通しの完了とせず、欠けた工程を示す。

## 部分工程の診断(互換入口)

次の資産は、このサンプルの部分工程の単独の診断として、現在の位置と入口のまま残します。実行方法は各READMEを参照してください。

| 資産 | 対応する工程 | 役割 |
| --- | --- | --- |
| [`tools/reference-plugin-self-build`](../../tools/reference-plugin-self-build/)(説明は[`reference/plugin-self-build`](../plugin-self-build/README.md)) | 3 | 自己構築の単独診断。通しの核でもある |
| [`tools/reference-transform-runtime-engine-build`](../../tools/reference-transform-runtime-engine-build/README.md) | 3 | 変換エンジンの実構築と出力の確認 |
| [`tools/reference-generation-graph`](../../tools/reference-generation-graph/README.md) | 1 | 段階つきの変換グラフの検査 |
| [`tools/reference-generator-selection`](../../tools/reference-generator-selection/README.md) | 1 | 生成器の候補を同じ期待値で比べて選ぶ |
| [`tools/reference-target-profile`](../../tools/reference-target-profile/README.md) | 3 | 同じ契約を複数の実行先の型で動かして比べる |
| [`tools/reference-plugin-bundle`](../../tools/reference-plugin-bundle/README.md)、[`plugins/reference-bundle-alpha`](../../plugins/reference-bundle-alpha/README.md)、[`plugins/reference-bundle-beta`](../../plugins/reference-bundle-beta/README.md) | 工程外 | 二つの論理プラグインを一つの実行ファイルへ結合する診断。構築済みの成果物とは接続していない |
| [`tools/reference-build-execution-binding`](../../tools/reference-build-execution-binding/README.md) | 工程外 | 構築計画の型の分割を示す設計候補。基盤への圧力としてだけ追跡する |

## 基盤への改善圧力

見出しだけを示します。現状、求める条件、確認方法は、4.7節の「基盤への改善圧力と現状」を参照してください。

- 固有: 構築意図から計画を導く前段と、自己構築の実行器が接続されていない(コア)
- 固有: 構築計画の最初の版が、実行、成果物集合、主体を分けられない(コア)
- P-JOB-01: 構築試行の観測をホストが供給せず、失敗の分類が参照実装内の語彙にとどまる(実行時)
- 固有: 保管庫が実際の構築器の出力と接続されていない(データ)
- 固有: 変換の実行エンジンの識別が、実際にリンクされた実体の識別ではない(実行時)
- 固有: 自己構築の収束が初期の構築器のGo実装に依存し、ホストの署名と信頼の起点がコアの正式な型でない(ツール)
- 固有: 通しの完了を一つの証拠として示す手段がない(検証根拠)
