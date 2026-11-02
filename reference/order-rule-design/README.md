# 受注規則の設計(order-rule-design)

設計者が旧システムの設計資料を局所で読み、受理済みの受注の業務手順・データ・機能と突き合わせて、単価を正の値に限る規則の変更候補を作る技術検証用サンプルです。旧資料に由来する未解決を受理済みの意味へ混ぜずに運び、出所と署名つきの決定を付けた候補を、受理前のまま先行サンプル`order-workflow`(設計文書の4.1節)の共通受理へ渡せるかを確かめます。このサンプルは受理も正本の書き換えも行いません。

- 状態: 設計案は提案段階です。通しの流れは[`tools/reference-order-rule-design`](../../tools/reference-order-rule-design/README.md)で、既存資産を変更せずに合成の旧資料・複製した作業領域・試験用の署名鍵・論理時刻を使って実装し、試験しました。受理は行わず、共通受理へはpreviewとreviewだけで提出して止めます。`order-workflow`への提出は未接続です。実際の旧設計資料、実際の署名者の権限、実環境では確かめていません。各工程の状況は、次の工程表で工程ごとに区別して示します。
- 設計の正本: [技術検証用サンプルの統合設計](../../plan/design/TECH_VALIDATION_SAMPLES_2026-10.md)の4.2節。技術検証用サンプルの位置づけは同じ文書の1.2節、要件の目録は2章、資産の対応表は6章にあります。
- 各資産の使い方と実装状況は、それぞれのREADMEが正本です。このREADMEは工程との対応を案内するだけで、内容を写しません。

## 工程

「現状」は、工程に置いた既存資産の範囲で判断した区分です。工程の間の受け渡しは、通しの試験(`TestThroughOrderRuleDesign`)が一つの記録の連鎖として確かめます。出所状態と未解決の件数、候補の識別、基準版、先頭コミットと木、確認の照合値、署名文面の値が、工程をまたいで同じ値かを照合します。既存資産が呼べたか、呼べなかった理由は、そのREADMEの「既存資産を呼べるか」にあります。

| 区分 | 意味 |
| --- | --- |
| 実装あり | コードはあるが、資産の中に試験が見当たらない |
| 合成データで試験 | 合成データ、検証用の作業領域、模擬の提供元を使う試験が資産にある。このREADMEの作成時に試験は実行していない |
| 実環境で確認 | 実際の提供元や運用環境で確かめた記録がある |
| 足場なし | 工程を担う既存資産がない |
| 未確認 | 資産のREADMEから状況を確かめられなかった |

| 工程 | 内容 | 前工程から受け取るもの | 使う既存資産 | 現状 |
| --- | --- | --- | --- | --- |
| 1 | 旧設計資料の読取りと出所状態 | なし(利用者の資料と合成の転写契約) | [reference-xead-cross-view](../../tools/reference-xead-cross-view/README.md)、[reference-xead-source-adapter](../../tools/reference-xead-source-adapter/README.md)、共有入力の[合成の転写契約](../xead-source-transcription-contract.synthetic.json) | 合成データで試験 |
| 2 | 受理済み作業領域との突合と評価対象の特定 | 出所状態と未解決の件数 | [reference-business-dfd](../../tools/reference-business-dfd/README.md)、[reference-subsystem-cud](../../tools/reference-subsystem-cud/README.md)、共有入力の[business-system-design](../business-system-design/README.md)と[受注業務の作業領域](../workspace/) | 合成データで試験 |
| 3 | 変更案の作成 | 評価対象、未解決の一覧、工程1の検証根拠 | [business-spec-templates](../business-spec-templates/README.md)、[studio-authoring](../studio-authoring/README.md) | 合成データで試験 |
| 4 | 候補版の受渡しと再構築 | 工程3の候補 | [reference-tool-neutral-development](../../tools/reference-tool-neutral-development/README.md) | 合成データで試験 |
| 5 | 影響範囲と過去の検証根拠の再利用可否 | 工程4の候補版と基準版 | [reference-change-assurance](../../tools/reference-change-assurance/README.md) | 合成データで試験 |
| 6 | 審査、実行前確認、署名つき決定 | 工程5の値と、工程1・2の出所状態と未解決の一覧 | [reference-authoring-witness-openssh](../../tools/reference-authoring-witness-openssh/README.md)、[reference-spec-review](../../tools/reference-spec-review/README.md)、[reference-purpose-preflight](../../tools/reference-purpose-preflight/README.md) | 合成データで試験 |

補足します。

- 工程4は、検証用の作業領域を使い捨ての複製で再構築します。通しの検証はサンプル独自の受領記録(先頭コミット、木、確認したmain)を使い、`qualify.sh`の資格確認の受領記録は、全体の健全性検査を要するため呼びません。受領記録は、検証したコミットに限った記録です。
- 工程6の署名の探針(`pressure-test.sh`)は、固定のダミー文面に署名します。通しの検証はこれとは別に、候補の照合値、審査の照合値、基準版、理由の照合値を入れた文面へ`tools/internal/opensshwitness`で署名し、共通受理の再計算値と照合します。署名の決定の値は受理前の提出への署名で、受理の記録ではありません。
- 共通受理(`tools/reference-shared-acceptance`)へは`SubmissionBundle`を提出して止めます。提出先は`Submitter`で差し替えられ、`order-workflow`への接続は未接続です。差し替え点は[`tools/reference-order-rule-design`のREADME](../../tools/reference-order-rule-design/README.md)にあります。
- 工程6の後段にある共通受理は、どのサンプルにも吸収しない共通の基盤([reference-shared-acceptance](../../tools/reference-shared-acceptance/README.md))が担います。受け手の`order-workflow`は先行ブランチにあり、`main`には未統合です。

## 取り込むピーキーな要件と失敗注入

注入する失敗と合否の観測の詳細は、設計文書の4.2節にあります。要件の内容は2章の要件目録を参照してください。「固有」は目録にない、このサンプル固有の困難です。

| 要件 | 工程 | 注入する失敗(要点) | 観測する合否(要点) |
| --- | --- | --- | --- |
| 固有 | 1・3・6 | 旧資料に契約が扱えない記述が混ざる。資料が差し替わる | 出所状態と未解決の件数が署名の理由まで同じ値で残る |
| 固有 | 2・5・6 | 確認器ごとに対象の規則が異なる | 対象外の「指摘なし」を問題なしの根拠にしない |
| 固有 | 5 | 試験受理の版や過去の根拠を、受理済みと取り違える | 再利用候補を合格や実行許可として扱わない |
| P-WF-01 | 4〜6 | 署名の後に基準版や`main`が進む。署名を流用する | 再計算値の不一致で署名を止め、流用を拒否する |
| P-MIX-01 | 5 | 規則の変更を三つの実行先のうち二つにだけ派生させる | 派生していない実行先が再確認の範囲に残る |
| P-WF-02 | 5・6 | 旧版R0で開始した受注が未完了のまま残る | 実行中の案件と版が影響範囲に並び、扱いを決めずに署名しない |

## 通しの完了条件

通しの検証器(`tools/reference-order-rule-design`、実行は`python3 tools/verify-order-rule-design.py`を清浄な複製で)が、次の条件を別々の確認の足し算ではなく一つの記録の連鎖で確かめます。「五つ目」だけは、`order-workflow`がmainに未統合のため、共通受理(`tools/reference-shared-acceptance`)の再計算との一致までしか確かめられていません。

- 工程3の候補の照合値と基準版、工程4の先頭コミットと木、工程5の確認の照合値、工程6の署名文面の値が、一つの候補について一致する。途中で一つを変えると、その工程が拒否または不一致として検出する。
- 出所状態と未解決の件数が、候補、審査パケット、署名の理由に同じ値で現れる。
- 工程2・5・6の確認器が同じ候補版に対する出力を返し、対象外や入力不足も一覧に残る。「指摘なし」だけを集約した一覧は失敗とする。
- 工程5の後に基準版を進めると署名されず、工程4の後に`main`を進めると受領記録の再照合が失敗する。
- 署名文面の値が、`order-workflow`が共通受理へ渡す値と一致する。`order-workflow`の統合までは検証できない。
- 全工程を通して、受注業務の作業領域、`reference/world.json`、プラグインの固定情報、受理記録が変わらない。試験受理は使い捨ての複製だけで行う。

成立状況: 一つ目から四つ目と六つ目は、合成の旧資料と試験用の署名鍵で成立しました。一つ目の途中の識別の差し替えは、各工程の記録の単独の確認は通り、通しだけが不合格になる陰性対照で確かめています。五つ目の`order-workflow`との一致は未検証です(「受理は未接続」)。出所状態の三値は、合成資料では三値とも作れますが、実際の旧設計資料で得られる値は未確認です。

## 部分工程の診断(互換入口)

次の資産は、工程の一つの境界を単独で確かめる診断として、位置と入口を変えずに残します。実行方法は各資産のREADMEを参照してください。前の工程の出力を受け取る経路は、どれも持っていません(通しの検証器が工程の間をつなぎます)。

| 工程 | 資産 | 確かめる境界 |
| --- | --- | --- |
| 1 | [reference-xead-source-inspect](../../tools/reference-xead-source-inspect/README.md) | 資料の事前検査と集計 |
| 2 | [reference-business-procedure-check](../../tools/reference-business-procedure-check/)(READMEなし。[tools/README.md](../../tools/README.md)に掲載) | 業務手順と担当・機能・データの整合 |
| 2 | [reference-procedure-formalization](../../tools/reference-procedure-formalization/README.md) | 手順書から作った候補の不足と矛盾を質問にする |
| 3 | [reference-rule-candidate-discovery](../../tools/reference-rule-candidate-discovery/README.md) | 実例から規則の仮説を複数示す |
| 3 | [reference-exploratory-prototyping](../../tools/reference-exploratory-prototyping/README.md) | 試作の反復と部分採用 |
| 3 | [reference-authoring-auxiliary](../../tools/reference-authoring-auxiliary/README.md) | 設計の補助情報の置き場の分類 |
| 4 | [reference-external-coordination](../../tools/reference-external-coordination/README.md) | 外部との作業の受渡し |
| 4 | [reference-chat-cli-host](../../tools/reference-chat-cli-host/README.md) | 資格確認の前の事前診断 |
| 5 | [reference-procedure-expansion](../../tools/reference-procedure-expansion/README.md) | 業務手順の実現先ごとの展開(P-MIX-01の足場) |

## 基盤への改善圧力

詳細(現状、求める条件、確認方法)は設計文書の4.2節「基盤への改善圧力と現状」にあります。

| 要件 | 不足 |
| --- | --- |
| 固有 | 出所と確認履歴が、審査入力と受理境界まで一本につながらない |
| 固有 | 旧資料由来の未解決が、各段階で同じ形で残ることを確かめていない |
| 固有 | 確認器が自分の対象外を出力に示さない |
| 固有 | 認識状態(確認の鮮度)が審査の入力に接続されていない |
| P-WF-01 | 署名による決定者の確認が、共通受理の決定に接続されていない |
| 固有 | 共通受理の経路と直接編集の経路が同じ意味に至ることの比較がない |
| P-MIX-01 | 規則の変更の影響範囲に、実行先ごとの派生が入らない |
| P-WF-02 | 実行中の案件の版を、改訂の影響として設計側へ返す経路がない |
