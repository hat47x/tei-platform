# 受注の画面経路と配備単位(order-web-delivery)

受注担当者が、未確定の受注を画面から閲覧し、確定し、確定結果を別の観測で確かめるまでを通し、その経路を画面とAPIの二つの配備単位に分けても主体と版が食い違わないかを確かめる技術検証用サンプルです。権限の失効、他の担当者による先行の確定、確定応答が届かない結果不明を、業務の成功と取り違えないことが焦点です。拡張として、同じ受注を会員向けの注文履歴として公開し、他人の行と社内向けの項目を返さないこと、会員の照会の急増から社内の確定を守ることを加えます。

- 状態: 設計案は提案段階です。このREADMEは入口であり、通しの検証は未実装です。各工程の実装・試験・実環境での確認の状況は、次の工程表で工程ごとに区別して示します。
- 設計の正本: [技術検証用サンプルの統合設計](../../plan/design/TECH_VALIDATION_SAMPLES_2026-10.md)の4.4節。技術検証用サンプルの位置づけは同じ文書の1.2節、要件の目録は2章、資産の対応表は6章、複合シナリオC6は3.3節にあります。
- 前提: 先行サンプル`order-workflow`(設計文書の4.1節)が受理した版R1と、残る未確定の受注を入力にします。`order-workflow`は`main`に未統合で、この接続は未成立です。
- 各資産の使い方と実装状況は、それぞれのREADMEが正本です。このREADMEは工程との対応を案内するだけで、内容を写しません。

## 工程

「現状」は、工程に置いた既存資産の範囲で判断した区分です。工程の間の受け渡しは、すべての工程で未接続です。現在の第一の配信先は合成の実行時データで動き、受理版R1や実DBの受注を読みません。

| 区分 | 意味 |
| --- | --- |
| 実装あり | コードはあるが、資産の中に試験が見当たらない |
| 合成データで試験 | 合成データ、検証用の作業領域、模擬の提供元を使う試験が資産にある。このREADMEの作成時に試験は実行していない |
| 実環境で確認 | 実際の提供元や運用環境で確かめた記録がある |
| 足場なし | 工程を担う既存資産がない |
| 未確認 | 資産のREADMEから状況を確かめられなかった |

| 工程 | 内容 | 前工程から受け取るもの | 使う既存資産 | 現状 |
| --- | --- | --- | --- | --- |
| 1 | 閲覧の入力(旧認証認可の引継ぎと公開読込の行スコープ) | なし(`order-workflow`の受理版R1の識別子と、未確定の受注の対象ID) | [web-api-contract](../web-api-contract/README.md)、共有入力の[driver-scenarios](../driver-scenarios/README.md)と[legacy-auth](../legacy-auth/)(READMEなし) | 合成データで試験 |
| 2 | 保護された確定要求の経路 | 確定操作、認可済みの主体、読込計画の照合値 | 部分診断の[web-composition](../web-composition/README.md)、共有入力の[legacy-auth](../legacy-auth/) | 合成データで試験 |
| 3 | 案件別の画面の重ね合わせ | 工程2の保護された確定境界 | 部分診断の[ui-customization](../ui-customization/README.md) | 合成データで試験 |
| 4 | 閲覧から確定と別観測の通し | 工程1〜3の成果と、受理版と対象ID | 部分診断の[reference-first-web-target](../../tools/reference-first-web-target/README.md) | 合成データで試験 |
| 5 | 配備単位の分離と固定投影の一致判定 | 工程4の確定経路と投影の記録 | [runtime-packaging](../runtime-packaging/README.md)、[reference-frozen-manifest-target](../../tools/reference-frozen-manifest-target/README.md)、[reference-target-package-evidence](../../tools/reference-target-package-evidence/README.md) | 合成データで試験 |
| 6 | 会員向けの注文履歴の公開と負荷の分離(拡張) | 工程1の読込計画と、工程5の二つの配備単位 | なし | 足場なし |

補足します。

- 工程1の資産は、実PostgreSQLでの行スコープの試験を条件付きで持ちます。稼働中のDB、実際の認可の正本、HTTPの処理とは未接続です。
- 工程2〜4は、工程に組み込む資産がなく、部分診断として残す資産だけで境界を確かめています。工程2は実際の認証基盤やCookie、CSRFの試験を含みません。工程3は実際の描画を実証していません。
- 工程4の資産は判定段階Cが版とパスを固定しており、先行ブランチも変更しています。このサンプルの作業では触れません。
- 工程6には、鮮度とキャッシュの再利用条件を合成で扱う部分診断だけがあります。会員向けの射影、流量制御、配備単位をまたぐ剥奪は未着手です。

## 取り込むピーキーな要件と失敗注入

注入する失敗と合否の観測の詳細は、設計文書の4.4節にあります。要件の内容は2章の要件目録を参照してください。「固有」は目録にない、このサンプル固有の困難です。

| 要件 | 工程 | 注入する失敗(要点) | 観測する合否(要点) |
| --- | --- | --- | --- |
| P-OPS-02 | 2・4 | 確定の直前にアカウントが失効する、主体の結び付きが変わる | 副作用の直前までに拒否し、応答・監査・観測が一致する |
| 固有 | 4 | 閲覧の間に、別の担当者が同じ受注を先に確定する | 閲覧した版と現在の版の食い違いで拒否する |
| P-TX-02 | 4 | 確定の応答が届かない | 別の観測で状態を確かめ、再送を重複確定として拒否する |
| P-B2C-01 | 1 | 検索、並べ替え、件数から禁止項目や他者の行を推測する | 禁止項目と行スコープの欠落を実行前に拒否する |
| 固有 | 3 | 案件の画面変更が基準部品を書き換える | 基準部品と業務の正本が変わらず、同じ確定境界を使う |
| P-MIX-03 | 5 | 分割後の成果物が固定した投影と食い違う | 一致しない成果物を配信しない |
| P-B2C-01 | 6 | 会員が他人の注文を照会し、許可外の項目を更新する(拡張) | 拒否され、射影にない項目が応答に一つもない |
| P-B2C-02 | 6 | 会員の照会を急増させながら社内の確定を流す(拡張) | 社内の応答時間が目標内に収まり、会員側は429で劣化する |
| P-DB-01 | 6 | キャッシュとDBの更新順を入れ替える(拡張) | 確定の判断は正本で再検証し、表示に取得時点を示す |
| P-OPS-02 | 6 | 一方の配備単位で剥奪し、他方から照会する(拡張) | 宣言した時間内に拒否され、縮小後の行が見えない |

## 通しの完了条件

通しの完了条件は、現在は一つも成立していません。画面経路が受理版R1と実際の受注を読む接続がなく、工程をつなぐ記録もないためです。

- 同一の受理版、主体、対象IDで、閲覧、案件画面からの確定、失効による拒否、確定応答を捨てた後の別観測、分割した二つの成果物での再実行までが、一本の記録で連鎖する。受理版、読込計画の照合値、受注版、成果物のダイジェストのどれか一つを省くと連鎖が切れる。
- 確定の途中でアカウントの失効や主体の変更が起きても副作用の直前までに拒否され、拒否が応答、監査、観測で一致する。
- 他者が先に確定した受注や結果不明の確定について、古い閲覧からの確定を通さず、再送で二重に確定しない。
- 案件の画面変更を加えても、基準部品と業務の正本は変わらず、標準画面と案件画面が同じ確定境界で同じ結果になる。
- 設計時の読込器を持たない配信先が、独立に固定した投影と一致する場合に限って起動する。
- 画面経路が扱う受注が、`order-workflow`の受理版と、その検証用の複製に残る未確定の受注に由来する。
- (拡張)会員向けの経路が他人の注文と社内向けの項目を返さず、照会の急増でも社内の確定の応答時間が目標内に収まり、配備単位をまたいで剥奪が宣言した時間内に効く。

## 部分工程の診断(互換入口)

次の資産は、工程の一つの境界を単独で確かめる診断として、位置と入口を変えずに残します。実行方法は各資産のREADMEを参照してください。判定段階Cと、R7A・R7Bが固定する根拠取得器群は、このサンプルに吸収しません(設計文書の5章)。

| 工程 | 資産 | 確かめる境界 |
| --- | --- | --- |
| 1 | [reference-scoped-query-preflight](../../tools/reference-scoped-query-preflight/README.md) | 行スコープを物理SQLへ適用できるかの実行前判定(P-B2C-01) |
| 1 | [reference-derivation-disclosure](../../tools/reference-derivation-disclosure/README.md) | 導出結果の開示境界 |
| 2 | [web-composition](../web-composition/README.md) | 必須保護の不足と三つの境界での再検証 |
| 3 | [ui-customization](../ui-customization/README.md) | 案件別の画面の重ね合わせ |
| 3 | [reference-runtime-change-instruction](../../tools/reference-runtime-change-instruction/README.md) | 画面上の変更の指示を見た目・業務規則・実行権限に分ける |
| 4 | [reference-first-web-target](../../tools/reference-first-web-target/README.md) | 閲覧・確定・観測の第一の配信先 |
| 4 | [derivation-projection](../derivation-projection/README.md)、[reference-derivation-projection](../../tools/reference-derivation-projection/README.md) | 導出項目のプレビュー、評価値、確定保存値の区別 |
| 4 | [derivation-validation](../derivation-validation/README.md)、[reference-derived-validation](../../tools/reference-derived-validation/README.md) | 導出の失敗と業務制約の不成立の区別 |
| 5 | [reference-first-web-two-unit-e1](../../tools/reference-first-web-two-unit-e1/README.md) | 画面とAPIの二つの成果物 |
| 5 | [reference-first-web-frozen-world-generate](../../tools/reference-first-web-frozen-world-generate/)、[reference-first-web-frozen-world-verify](../../tools/reference-first-web-frozen-world-verify/)(どちらもREADMEなし) | 固定した全体モデルの生成と検証 |
| 5 | [reference-first-web-target-manifest-capture](../../tools/reference-first-web-target-manifest-capture/README.md)、[reference-runtime-snapshot](../../tools/reference-runtime-snapshot/README.md) | 投影の採取と固定した投影の読取り |
| 5 | [reference-first-web-host-admission](../../tools/reference-first-web-host-admission/)(READMEなし) | 配備単位の選択と、ホストの現在の受理との照合 |
| 6 | [reference-derivation-cache](../../tools/reference-derivation-cache/README.md) | 導出値の鮮度とキャッシュの再利用条件(P-DB-01) |

## 基盤への改善圧力

詳細(現状、求める条件、確認方法)は設計文書の4.4節「基盤への改善圧力と現状」にあります。

| 要件 | 不足 |
| --- | --- |
| 固有 | 画面経路が`order-workflow`の受理版と受注を同じ版で読む接続がない |
| 固有 | 読込計画と案件画面の解決器を、実行体が消費していない |
| P-B2C-01 | 行スコープを認可の正本から導き、実際の提供元で強制した証拠がない |
| P-OPS-02 | 実際の認証基盤、Cookie、CSRFと、主体の結び付きの管理・失効の伝播が未成立 |
| 固有 | 配備単位ごとの依存閉包がプラグインの固定情報と接続されていない |
| 固有 | 固定した値の出所が、独立した受理・署名の権限と接続されていない |
| 固有 | 通しの完了を流れとして記録する枠組みがない |
| P-B2C-01 | 利用者の種類ごとの射影を、同じ読込計画から導く仕組みがない |
| P-B2C-02 | 流量制御と資源の分離がない |
| P-DB-01 | 会員画面に返す値の鮮度を宣言する手段がない |
