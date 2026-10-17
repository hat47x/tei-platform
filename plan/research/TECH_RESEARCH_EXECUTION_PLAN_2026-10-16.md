# TEI 技術調査の実証計画 — 2026年10月16日

状態：**着手候補の計画**（実行・製品採用・受入済みを示さない）  
期間：2026-10-16〜11-13を目安。日付は調整可能であり、既存Issueの受理順・保護された統合条件を置き換えない。  
根拠の正本：[2026年10月の技術調査の統合方針](TECH_RESEARCH_INTEGRATION_2026-10.md)、[製品クリティカルパス](../workstreams/TEI_PRODUCT_CRITICAL_PATH_2026-10.md)、[現行実装状況](../../docs/status.md)。調査論文の結果はTEIでの実測ではない。

## 1. 今回の判断

**新しいAI生成器・状態管理DB・Wasm共通基盤の採用を急がない。** TEIの既存の意味・契約・受理・証拠の区別を保ち、一つの業務変更について「設計の妥当性、実際の操作、実行後の観測、後日の差分」を追跡する。そのうえで、手戻りや確認の負担を減らせるかを先に実証する。

次の四点を重点的に扱う。独立した新しい判定機構は設けず、既存のIssueで分担する。

1. **仕様・設計の保持**：Code of Theseusの示唆を踏まえ、実装が動くかだけでなく、元の責務・依存関係・業務上の意味が保たれるかを照合する。#982、#998、#1120、#1124、#1803、#1805。
2. **変更後の状態と操作の追跡**：VisStateCoderの局所更新と状態ずれの考え方を、既存のSource Index / Typed Reference / Scenario / Evidenceを参照する際に利用する。新しいProject State Graphを正本化しない。#1445、#1180、#997、#1824。
3. **人・AI共通のCapabilityと観測**：ANISのdiscoverability／granularity／observabilityを、TEIの既存Operation／Service Contract／Runtime Authorityに適用する。AI専用の第二Operationを作らない。#951、#1121、#1824、#259。
4. **検証器を検証する**：Oracle Integrity Auditの失敗注入・既知の正解・実行前後の状態保存を、既存のScenario/Evidence試験の品質評価に加える。検証器自身のPASSを業務上のPASSとみなさない。#1119、#1163、#1445。

既存の製品横断優先順位（受理済みSourceの実Web自身による拘束、同一R1からの実DB書込みと独立観測、三視点の受理）を追い越さない。軽量認知・Wasm/WIT・Studioはこの実業務の証明に**直列の前提として追加しない**。

## 2. 参照する最新研究（2026-10-16に要旨を再確認）

| 研究 | 主張された観測と適用限界 | TEIで検証する仮説 |
| --- | --- | --- |
| [The Code of Theseus（10/14提出、10/15改訂）](https://arxiv.org/abs/2610.20068) | 設計文書を与えてもheld-outテスト成功はほぼ変わらず（33.7%→35.8%）、再構築したモジュール構造F1は0.54→0.63。ソフトウェアの設計構造と動作は別軸 | Source/三視点/Capability/実装の責務関係を復元できるか。振舞いの適合と設計構造の保持を別に計測する |
| [VisStateCoder（10/13提出）](https://arxiv.org/abs/2610.17861) | 画面・ファイル・操作・描画証拠の関係を追跡して、更新の状態ずれを抑える。実験成績は研究の評価環境限定 | 差分によって実際に影響するUI部品・操作・Scenarioだけを再確認し、影響外を壊さないか |
| [ANIS（10/11提出）](https://arxiv.org/abs/2610.15765) | AI操作の発見しやすさ、まとまった操作単位、変更後の読取り確認を分ける。論文の環境ではトークン使用量約73%減 | 同じCapabilityをHuman UIとAgent向け契約で発見・呼出し・別の読取りで照合できるか |
| [When Agent Evaluation Becomes Unreliable（10/12提出）](https://arxiv.org/abs/2610.16067) | Oracle自身による評価前の状態改変で成功率が過大に見える実例。オフライン故障注入と状態不変条件で調査 | TEIの検証器に既知の正負ケースと故障注入を当て、検証処理が対象の状態・期待値を変えないか |
| [Verify Claims, Not Scores（10/1提出）](https://arxiv.org/abs/2610.01348) | 単一スコアより主張別の根拠、未検証、到達可能な範囲の評価を重視 | UI表示、操作成功、業務結果、独立Observation、受理・Authorityの各主張を別々に表示する |

これらの論文にある数値を、TEIの目標値や受理条件として持ち込まない。実装や実験の優先順位は、TEIの製品価値、利用者が確認する負担、検証できるSourceの有無から決める。

## 3. 取り組む共通の縦断事例

既存の受注管理`order_entry`を中心に、小さな変更を一つ追跡する。参考ケースに含めるのは、顧客参照と、明細の入力検証・行番号・承認規則など、**既存の宣言で確認できる要素だけ**とする。未受理の新規規則を実装済みとして扱わない。

```text
Requirement / original Source
  -> 三視点のTyped Reference / 仕様の欠落・矛盾
  -> 変更候補と最小差分（未確定はunresolved）
  -> 既存Shared Acceptance（受理するときだけ）
  -> 既存Operation/Capability/Providerを選択
  -> Human UI / Agent Interface / Go Web-Desktop Host
  -> Command と別のObservation
  -> Scenario・Evidence・exact Source / Target revision
  -> 変更後に影響する範囲だけ再検証
```

同じ事例について、少なくとも次の負例を用意する。未受理改変、古いSource版と意味が等価な別Byte版、権限不足、表示は成功したがCommand未実行、Command完了後のObservation失敗、誤ったOracle、他のUIやSourceを巻き込む局所修正。実環境で確認していない負例は「試験案」または「合成試験済み」にとどめる。

## 4. 作業順と完了の境界

### A. 10/16〜10/23：業務変更の縦断基準と証拠の信用度【最優先】

担当既存Issue：#1119、#1121、#1127、#1163、#1445、#1180、#267。

- 既存の`Source Revision → Target build → Action → Observation`の各Ownerと証拠の接続先を一枚の読取り専用表に整理する。旧E1合成8receiptと実Provider／独立DBの検証を混同しない。
- #1121で、受理済みR1を実Web自身が照合するR1正常／R0未受理／意味同値R2を**Web側で拒否**する試験へ接続。ラベル検査やハーネス検査は実Webの保証と別。
- #1127/#1163で同一Accepted Source版・同一Operationの実DB write→別実行／別接続readと、read失敗のN11を照合。Provider/Authorityの担当実装は各Ownerのままにする。
- #1445の既存POC-8〜13（Typed Impact、2-Workspace Source diff、Scenario、原本receipt SHA照合）を、**独立したテストとsource provenanceが届く範囲でのみ**検証計画に使う。
- #1163と協議して短命scratchのR0/R1全Sourceの保存範囲・個人情報除去・保持期間・checksumを決める。元のアーカイブからSource集合を推測復元しない。

**出口**：ケースごとにExact Source／Target／権限主体／Command／Observationが`supported / unsupported / unresolved / not-evaluated`で示され、未取得の証拠に具体的な担当と再実行方法があること。**Gate合格自体はこの計画で決めない**。

**Phase Aの具体的な受渡しと現存証拠の差異**：[一改訂の実行・観測証拠の受渡し（2026-10-16）](../workstreams/TEI_RESEARCH_PHASE_A_EVIDENCE_HANDOFF_2026-10-16.md)に、#1121の現行Web Host拘束、合成E1と別系統の実PostgreSQL E2a-1の不一致、8未達条件のOwner、既存検証入口を整理した。**正式なHost Current Accepted Sourceの正本提供と、同一order_entry実行のPostgreSQL→別invocation読取りが最初の未解決接合点**である。既存の歴史的receiptを製品G2の根拠へ合算しない。

### B. 10/23〜10/30：仕様の欠落・設計構造・局所修正の比較【高】

担当既存Issue：#982、#998、#1120、#1124、#1803、#1805、#1445。候補：既存REF-SPEC-01。

- 同じ`order_entry`事例について「欠落参照、曖昧な更新可否、矛盾する権限、古いRevision」をそれぞれ少なくとも1件用意し、元Source・Typed Reference・未解決事項までたどれることを確認する。
- 受理済み意味から生成する設計一覧の試作で、Data／Function／Business Procedure→Operation／実装の依存と責務を導出する。Codeの表面や識別子が一致するだけで意味が同じとしない。
- (a)従来のコード先行、(b)既存Sourceと型付き参照を用いる工程、(c)AI支援で根拠へ誘導する工程を、**同じ受入基準**で比較する。変更対象外のファイルに触れないか、既存試験の再合格が維持されるかを記録する。
- Code of Theseusから「設計構造も別に評価する」着想を採るが、TEIの唯一のモジュール構成への強制や過度な構造一致を求めない。

**出口**：4種類の仕様上の問題、Sourceと依存の追跡、最低1件の局所修正と負例について、検出／見落とし／誤警報を同じ評価表で比較する。構造上の改善だけを業務PASSと呼ばない。

### C. 10/30〜11/06：Human UI・Agent Interface・Studioの一貫性【高、Bと並行可能】

担当既存Issue：#1824、#997、#1000、#951、#1180、#1168、#1441。候補：既存REF-INTERACTION-01。

- TEIの一つの既存Operation/Capabilityについて、Human向けUIのボタン等と、AI向けの発見・引数Schema・呼出し・読み戻しを**同じ契約**から提示する。Agent向けの独立した業務意味・実行Authorityを作らない。
- 各操作の`UI部品 → Event → Contract → Command → Observation`を正負両方で試す。変更後の画面を再描画できるだけでは成功扱いにしない。
- Web標準＋Litを優先する共通部品、Goの共通操作、Web/ Wails v3 Desktop Host固有Adapterを区別する。既存React Web Referenceは当面の実測基準として保持。
- Studio（編集用Host）のWebView起動成功と、**生成された業務アプリのDesktop Target**の適合試験を混同しない。Aに続く既存#1824のPoC A/B/Cを流用し、所有PRと競合させない。
- 画面とAgent側の同一操作の記録から、未解決、権限拒否、Command結果、別read Observationを別々に表示する。失敗した画面部分に対応する変更だけ再検証し、変更対象外のUI/Scenarioへ回帰がないか確認。

**出口**：一つのCapabilityについてHuman UIとAgent呼出しの実契約・呼出結果・別Observationが辿れること。実Webブラウザー、実WebView、人の操作評価、合成UIテストを**個別に**記録する。

### D. 11/06〜11/13：同じ意味の可搬実行と軽量化採否【条件付き】

担当既存Issue：#1861、#259、#943、#973、#976、#1016、#1824。候補：既存REF-PORTABLE-01。

- 既存#1861の受注明細の**読み取り専用見積もりプレビュー**（既存23ベクトル）を比較基準にし、Go Native／Node JS、TinyGo等のWasm候補、必要なら実WIT Component＋異なる2 Hostを比較する。WITの採用は目的ではない。
- 同じ期待値を使い、正常値・業務拒否・Host import欠落・権限不足・Artifact不一致を区別する。ブラウザ動作はNode CLIの結果から推測しない。
- 成果物サイズ、初回起動、レイテンシ、ピークメモリ、依存ツール、実際の移植作業量を測る。測定前に共通の数値上限を課さない。
- HikeはGo製TEI CoreのコンパイラではなくGo似の別言語であり、費用を含めて比較できる場合のみ別候補とする。WASI 0.2/0.3とHost実装の対応も別々に確認する。
- 実Consumerと二つの実Hostで明らかな価値がなければ、Go Native＋Host Adapterを継続し、共通Wasm ABIの標準化を保留する。

**出口**：一つの固定入力集合、実Artifact SHA・ツール版・Host版・実行結果・資源使用量・対象外の記録をまとめ、`採用候補 / 継続調査 / 不採用`を決める。ただし正式標準化は該当Issue／ADRの判断による。

### E. 横断：CPU-first認知・セルフビルドは解除条件を満たしてから

担当：`ISSUE-000047`、#843、#876、#951、#943、#1067、#1445。

- まず既存`InspectSubject`・Source Index・Typed Referenceと決定的な候補探索で複数Applicationの共通化可能性を調べる。**SEI、Jev、FlyHash、LightGBMは候補順位・保留判断の比較用**であり、Meaning Acceptanceに昇格しない。
- 実際に許可された二つのApplication、契約が一致／不一致となる正負例、CPUだけの再現可能な基線が揃うまでは学習済みモデル・蒸留を開始しない。#843/#876の停止・資格条件を独断で解除しない。
- エージェントが実行を提案した場合、現在の権限、対象、実行結果、独立Observationと作業再開情報を追跡する。過去のチャットやAIの記憶をOwnerにはしない。
- TEIによるTEIのセルフビルドは、既存Build/Artifact/Plugin Lockの実契約で成立する範囲を伸ばし、全体の完成をA〜Dの前提にしない。

**出口**：CPU基線との差が測れる候補だけを、既存Issueの次段階へ昇格させる。モデルサイズ・学習コスト・トークン数だけを成功指標としない。

## 5. 検証方法と記録する値

結果を一つの平均スコアにまとめず、Source Revisionと対象環境をそろえて項目ごとに記録する。

| 軸 | 最小記録 | 禁止する読み替え |
| --- | --- | --- |
| 意味・設計 | 受理済み／候補／未解決、誤りの検出数と誤警報、Cross-view依存の説明 | 見た目やテスト通過からMeaning Acceptanceを推測 |
| 機能・UI | 操作を見つけられたか、Action/Commandの結果、拒否結果、画面での説明可能性 | UI描画成功＝業務完了 |
| 実行・証拠 | exact Source／Artifact版、実Target、現在の権限、別read Observation、Oracle由来と検証器の非改変性 | 合成receipt・自己生成Oracle・古い成功＝本番PASS |
| 開発負担 | 操作数、確認時間、AIの追加質問数、局所再テスト件数、無関係なSource更新、再作業 | 生成コード量やトークン削減だけで生産性向上 |
| 可搬性・資源 | 実行先、ABI、依存、起動、CPU/メモリ、バイナリサイズ、移植の工数 | Studio Host成功＝業務Target適合 |

**Oracle側の健全性試験**を各Referenceの少数の既知正負ケースに必ず含める。少なくとも、期待値を変えても検証器が異常を示さずに成功と判定しないこと、検証器が検査前後の対象状態を書き換えないこと、Observationを取得できないときにPASSとしないこと、異なる実行版の証拠を結合しないことを確認する。必要な実行環境がないときは`not-evaluated`とする。仕様検査に恣意的な大量の性能・品質数値閾値は追加しない。

## 6. 開発・統合規約

- **既存IssueがOwner**。開始時に現在のmain、進行中PR、同じPathの担当を照合し、同一パスの並行編集を避ける。この計画はIssueの権限や受入条件を変更しない。
- 各差分は**一つの具体的Consumerと失敗反例**に絞る。先にRead-only Projectionと反証テストを作り、正式なSchema、共通Core型、ResolverやGateを増やすのは実Consumerに不足が認められてから。
- Issue・PRに変更範囲／対象版／根拠と未実施を記録する。必要な局所試験・静的解析・実Host試験を選び、`AGENTS.md`に従いexact headで確かめる。全Repository-healthの一律PASSは要求しない。合成試験・実Host試験・人的受理を混同しない。
- マイルストーンごとに条件を確認しmainに統合したら不要なブランチを削除する。未達成なら無理に結合せず、原因と担当を明示する。
- 研究で効果が出ない場合や未知依存の閉包が証明できない場合は`unresolved`で停止し、比較方式の採用を保留する。既存の正式受理・製品Gateを研究結果で書き換えない。

## 7. 10月16日からの具体的な起票・引継ぎ

新しいIssueをむやみに増やさない。現在OPENの#1445、#1119、#1121、#1127、#1163、#982、#1824、#1861を基本入口にする。

1. **今週**：#1119/#1163で同一R1のProduct証拠と旧合成E1の不足一覧を再確認し、#1121/#1127のWeb版拘束・別readの実作業順をそろえる。#1445は既存差分・receipt照合のread-only表示に徹する。
2. **次週**：#982/#998/#1120にREF-SPEC-01の4反例と手動／AI支援比較表を受け渡す。別の仕様正本ではなく、出所と診断を示す試験資料にする。
3. **並行**：#1824で同一OperationをGoサービスとLitベースのWeb/Desktop共通UIから呼ぶ対象を一つに限定し、#997/#951でContractを再利用する。#1168で人の操作評価の収集方法を決める。
4. **実Consumerの準備後**：#1861で実Wasm Componentと二Hostが本当に必要かを比較する。Go/WASM/Host同値と認可境界の未検証を明示する。
5. **毎週金曜**：新論文と前週の再現結果を見比べ、実証で減った不確実性だけを計画に反映する。論文の流行や新技術の数で優先順位を自動変更しない。

## 8. 見直し条件

- **前倒し**：実WebのSource R0/R2拒否、実DB write→別read、Human/Agent同一Capabilityを同じ版で実証でき、参照済み契約と既存Gateの接続先が判明した。
- **保留**：原子的Sourceや独立Observationが未取得、Oracleを分離できない、実Host未検証、並行実装のOwnerが確定しない。
- **方向転換**：仕様検査が既存の決定的な方法より誤警報や作業負担を増やす、共通UIが案件別の変更を難しくする、Component化がHost Adapterより複雑になる、CPU認知が決定的な基線を上回らない。

**本計画の完成条件**：採用した研究方式の数ではなく、同じ受注事例について「受理済み意味・Source版・対象操作・実Provider・観測・失敗時の未解決」が追跡でき、変更を行う人の確認負担と影響外の再検証が実測で減ったかを説明できること。
