# X-TEA知見の移行計画（2026-11）

- 状態: **計画案（未採用）**。作業の順序、成果物、受入、停止条件を整理します。Issueの作業条件や設計判断の採用を、この文書だけで決めません。
- 日付: 2026-11-01
- 入力: [X-TEAのローコード機構に関する知識目録](../research/XTEA_LOWCODE_KNOWLEDGE_INVENTORY_2026-11.md)、[X-TEA知見の受容アーキテクチャ](../design/XTEA_LOWCODE_RECEPTION_ARCHITECTURE_2026-11.md)、[X-TEA知見の受容要件](../../requirements/XTEA_LOWCODE_RECEPTION_REQUIREMENTS_2026-11.md)
- 関連: [#1127](https://github.com/hat47x/tei-platform/issues/1127)、[#1129](https://github.com/hat47x/tei-platform/issues/1129)、[#1130](https://github.com/hat47x/tei-platform/issues/1130)、[#1802](https://github.com/hat47x/tei-platform/issues/1802)、[#1803](https://github.com/hat47x/tei-platform/issues/1803)、[#2016](https://github.com/hat47x/tei-platform/issues/2016)、[#1973](https://github.com/hat47x/tei-platform/issues/1973)

## 1. 目的

X-TEAから回収したローコードの知見を、TEIの意味・対話契約・生成器・Plugin・作者環境・要求へ、**証拠を持って受け渡す**ための順序を決めます。

この計画が答えるのは、次の問いです。

- どの知見を、どの設計判断へ渡すか。
- どの順序なら、Coreを増やさずに定型の生産性を確かめられるか。
- 何を確認できたら「受け止めた」と言えるか。
- どこで外部効果が出て、どこから先を人に確認するか。

本文では、実行能力（Capability）を「能力」、正本から導出した表示・成果物（Projection）を「投影」と呼びます。受容は知識を受け止める設計上の行為、受理は意味を正本へ取り込む判断を指します。アプリケーションの型（Application Pattern）を「Pattern」、拡張を「Plugin」、設計・編集環境を「Studio」と呼びます。

```text
X-TEAの4アプリケーション（固定したcommit）
        ↓ 静的読解
知識目録（105項目、出典位置つき）
        ↓ 受容先の層へ
受容アーキテクチャ（D1〜D8、現行実装との差）
        ↓ 要求候補・受入条件
受容要件（AC-1〜AC-11、RQ-C1〜C3）
        ↓ 作業とゲート
移行計画（WP1〜WP10、G0〜G5）
```

## 2. 前提と制約

- X-TEAのソースは非公開の複製にあり、公開文書へパス・名称・集計を出しません。根拠は公開upstreamのcommitとファイル位置で示します。
- 参照モデル・手引書の本文は転載しません。利用条件の確認は[#941](https://github.com/hat47x/tei-platform/issues/941)の担当です。
- ソースの静的読解を、旧Runtimeの実行観測と呼ばない。実測は[#940](https://github.com/hat47x/tei-platform/issues/940)に譲ります。
- 意味の受理、要求の採用、権限、ライセンスは、開発作業の停止条件にしません。PoCとして案を実装・検証し、案の採否と版を記録して進めます（[ADR-0035](../adr/0035-autonomy-levels-and-verification-packets.md)のA3・A4）。人の確認を待つのは、対外公開や本番配備など、外部へ取り消せない効果を出す場合だけです。
- この計画の文書作業は隔離ブランチで行い、`main`を直接変更しません。実装を伴う作業は、対応Issueの条件に従います。

## 3. 現在地

| 成果物 | 状態 | 何を決めたか | 何を決めていないか |
| --- | --- | --- | --- |
| [知識目録](../research/XTEA_LOWCODE_KNOWLEDGE_INVENTORY_2026-11.md) | 105項目 | 機構・効果・破綻・根拠の一覧 | 全分岐、実行時の挙動、参照モデルでの使われ方 |
| [受容アーキテクチャ](../design/XTEA_LOWCODE_RECEPTION_ARCHITECTURE_2026-11.md) | 設計案 | 受容先の層、D1〜D8とD1.1・D1.2・D2.1・D3.1・D6.1の選択肢と推奨、現行実装との差、適合試験案 | D1〜D8の採用、パッケージの採否、フックのDSL |
| [受容要件](../../requirements/XTEA_LOWCODE_RECEPTION_REQUIREMENTS_2026-11.md) | 候補 | 既存RQの具体化（AC-1〜AC-11）、新規候補（RQ-C1〜C3） | 採用の可否、要求本文への反映 |
| 本計画 | 計画案 | 作業パッケージと順序、ゲート、停止条件 | 実施の割当、日程、実装ブランチ |
| [PoC PR](https://github.com/hat47x/tei-platform/pull/2521) | 実装案（Draft） | `interaction`、表示解決、画面HTML・印刷、Package、Semantic fingerprint、統合試験 | 意味の受理、mainへの統合、Commandの実行境界、集計・改ページ |

## 4. 作業パッケージ

### 4.1 既存Issueとの関係（2026-11-01時点）

| WP | 既存Issue（状態） | 既存が扱う範囲 | この計画で足りないこと |
| --- | --- | --- | --- |
| WP3 | [#1127](https://github.com/hat47x/tei-platform/issues/1127)、[#1124](https://github.com/hat47x/tei-platform/issues/1124)、[#997](https://github.com/hat47x/tei-platform/issues/997)（すべてopen） | 標準CRUD、三視点モデラー、UI構成キット | 対話契約の宣言と、Patternの組合せとしてのパッケージ |
| WP4 | [#1063](https://github.com/hat47x/tei-platform/issues/1063)、[#1129](https://github.com/hat47x/tei-platform/issues/1129)（open） | DSL、副作用規則 | フックの宣言・位相・効果の契約 |
| WP5 | [#1129](https://github.com/hat47x/tei-platform/issues/1129)、[#1174](https://github.com/hat47x/tei-platform/issues/1174)、[#2147](https://github.com/hat47x/tei-platform/issues/2147)（open） | 副作用、Commitフェンス、段階移行 | `delete`効果、結果の分類、読み取り専用の三段拒否 |
| WP6 | [#1130](https://github.com/hat47x/tei-platform/issues/1130)、[#1803](https://github.com/hat47x/tei-platform/issues/1803)（open） | バッチ・帳票、派生出力 | 出力投影の契約とテンプレートPlugin |
| WP7 | [#2016](https://github.com/hat47x/tei-platform/issues/2016)、[#2076](https://github.com/hat47x/tei-platform/issues/2076)（open） | 生成器の選抜・失効 | Patternパッケージの契約と版の束縛 |
| WP8 | なし | — | 暦・会計期間の意味と提供元（新規Issue候補） |
| WP9 | [#1802](https://github.com/hat47x/tei-platform/issues/1802)、[#1124](https://github.com/hat47x/tei-platform/issues/1124)、[#1125](https://github.com/hat47x/tei-platform/issues/1125)（open） | 機能移植の仕分け、三視点編集、業務DFD | 受理済み意味に対する診断の一覧と負例 |
| WP10 | なし | — | 文書の索引・引継ぎ（このブランチとPR） |

[#1804](https://github.com/hat47x/tei-platform/issues/1804)、[#1805](https://github.com/hat47x/tei-platform/issues/1805)、[#1806](https://github.com/hat47x/tei-platform/issues/1806)、[#1807](https://github.com/hat47x/tei-platform/issues/1807)、[#1808](https://github.com/hat47x/tei-platform/issues/1808)、[#1637](https://github.com/hat47x/tei-platform/issues/1637)、[#1973](https://github.com/hat47x/tei-platform/issues/1973)は完了済みです。完了した成果は前提として再利用しますが、新しい作業の担当にはしません。

各パッケージは、目的、内容、成果物、受入、依存、既存の担当、停止条件を持ちます。危険度はADR-0035の暫定運用に合わせます。

### WP1 知識目録の固定と点検

- **目的**: 後続の設計が参照する事実の一覧を、検証可能な状態で固定する。
- **内容**: 知識目録の根拠リンクを一件ずつ開き、ファイルと行が存在することを確認する。推論と事実の区分、非公開情報の混入、用語の揺れを点検する。
- **成果物**: 点検記録（知識目録の改訂、または別の作業記録）。
- **受入**: 根拠リンクが解決する。事実と推論の区分が全項目にある。非公開の名称・パス・集計がない。
- **依存**: なし。
- **危険度**: A0（読み取り・分析）。
- **停止条件**: 根拠の版が特定できない、または非公開情報の混入が疑われる場合は、該当項目を未確認へ戻す。

### WP2 受容アーキテクチャのレビューと設計判断

- **目的**: D1〜D8のうち、どれを採用し、どれを保留するかを決める。
- **内容**: D1（パッケージ）、D2（フック）、D3（項目の役割）、D4（配置）、D5（結果型）、D6（標準業務意味）、D7（帳票）、D8（診断）について、選択肢・推奨・撤回条件をレビューする。採用するものは設計判断記録の候補を作る。
- **成果物**: レビュー記録、採用した設計判断のADR案（採用は人）。
- **受入**: 各判断に、採用・保留・却下のいずれかと理由がある。保留には、何が分かれば決められるかがある。
- **依存**: WP1。
- **危険度**: A0（分析）。採用はA4。
- **停止条件**: Coreの要素種別の追加が必要になった場合は、パッケージの設計を見直すまで止める。

### WP3 最小の対話契約のPoC

- **目的**: 一覧と単票の二つのPatternで、項目の役割・表示意図・操作の宣言が、Coreを増やさずに成立するかを確かめる。
- **WP3a（先に行う）**: 既存の`order_record`と`customer_reference`を使い、項目の並び・参照入力・一覧から単票への遷移だけを宣言する。必須・更新禁止・値域は扱わない。Coreの属性を増やさず、対話契約の層だけを試せる。
- **WP3aの手順（案、未実施）**:
  1. 変更前の基準を取る。`go test ./internal/application ./internal/syntax`と`go run ./cmd/tei validate -workspace reference/workspace`の結果を記録する。
  2. `order_record`に、`orderId`、`customerId`、`orderedAt`、`status`、`total`の順序と、`customerId`から`customer_reference`への参照入力を宣言する。
  3. 構文と検証の負例を追加する。未知の項目、未知の参照先、項目の重複、型の不一致を拒否する。
  4. CLIとWebの投影で、項目の順序と参照入力が同じになることを確認する。表示だけを変えたときに意味の差分が出ないことを確認する。
  5. 変更候補は、`internal/application/model.go`の対話契約の型、`internal/syntax/application_pattern.go`の解析、`reference/workspace/applications/order-entry/patterns/order-record.tei`の宣言、構文・検証・投影の試験である。いずれもD1・D2.1の採用判断の後に着手する。
- **WP3b（D3.1の決定後）**: 必須・更新禁止・値域を、Attribute・Validation・Typeのどこへ置くかを決め、負例つきで試す。Coreの属性追加はA4に当たるため、決定前に実装しない。
- **成果物**: 参照用作業領域の小さな宣言、適合試験、負例、結果の記録。
- **受入**: AC-3、AC-4に対応する確認が取れる。Coreの要素種別を追加していない。表示の変更で意味の差分が出ない。
- **依存**: WP2のD1・D2.1・D3.1・D4の判断。
- **既存の担当**: [#1127](https://github.com/hat47x/tei-platform/issues/1127)、[#997](https://github.com/hat47x/tei-platform/issues/997)、[#1124](https://github.com/hat47x/tei-platform/issues/1124)。
- **危険度**: A2（隔離ブランチの変更）。
- **停止条件**: 項目の役割が既存のData Derivation・Validationと重複する場合は、新しい語彙を増やさず既存へ寄せる。参照入力か遷移が既存のPatternの組合せで表せない場合は、Coreの拡張ではなく、対話契約の設計を見直すまで止める。

### WP4 対話フックの契約PoC

- **目的**: 行レベルの検証・導出・参照入力の補助を、宣言と規則Pluginで表せるか確かめる。
- **内容**: フックの位相（読取前後、変更前後、削除前後）、範囲（行、レコード、項目）、読み取り、使用能力、生成物、失敗の種類を宣言する。規則Pluginを一つ用意し、未宣言能力の拒否、決定性、失敗の区別を試験する。画面と帳票で同じ結果になることを確認する。`Hook`型は現行コードに無く、`plugin.Capability`は提供元の能力を表す契約であるため、フックの契約にはそのまま使えない。宣言をApplicationの対話契約、実装契約をPluginの別契約に置く案（[受容アーキテクチャのD2.1](../design/XTEA_LOWCODE_RECEPTION_ARCHITECTURE_2026-11.md)のC）を第一候補として比較する。
- **成果物**: フック契約の案、規則Pluginの参照実装、負例、決定性の記録。
- **受入**: AC-2、AC-5に対応する確認が取れる。正本に任意コード・SQL・確定がない。
- **依存**: WP3。
- **既存の担当**: [#1063](https://github.com/hat47x/tei-platform/issues/1063)、[#1129](https://github.com/hat47x/tei-platform/issues/1129)。
- **危険度**: A2。
- **停止条件**: フックの種類が機能型の再実装になる場合は、Patternパッケージ側へ移す。

### WP5 能力の結果の分類と競合・読み取り専用のPoC

- **目的**: 0件・複数一致・競合・拒否・不明・未実行を、業務上の結末（Outcome）と実行の失敗（error）に分けて区別する。
- **内容**: 単票の更新・削除を対象に、更新カウンタを一つの実現として扱い、カウンタのない表でも契約を宣言できるようにする。既存の`Outcome`（`internal/semantic/model.go`）と呼出し境界のerrorを再利用し、第三の結果チャネルを増やさない。`DataEffect`が`delete`を持たない点をどう扱うかを決める。読み取り専用のデータソースへの書込みを、宣言・検証・実行の三段で拒否する。Commandの成功と観測を分ける。あわせて、メニュー非表示でも直接呼出しを拒否する操作権限の検査を、最小の負例として確認する。
- **成果物**: 結果型の案、負例（0件、複数、競合、読取専用、権限外の直接呼出し、例外）、観測の記録。
- **受入**: AC-5、AC-6、AC-10、AC-11に対応する確認が取れる。
- **依存**: WP3。
- **既存の担当**: [#1129](https://github.com/hat47x/tei-platform/issues/1129)、[#1174](https://github.com/hat47x/tei-platform/issues/1174)、[#2147](https://github.com/hat47x/tei-platform/issues/2147)。
- **危険度**: A2。
- **停止条件**: 既存のBusiness Effect Lifecycleと結果の分類が衝突する場合は、分類を統合するまで止める。

### WP6 親子と帳票の投影PoC

- **目的**: 一つの対話契約から、画面と帳票の複数の提示形態を出せるか確かめる。
- **内容**: 見出し明細の保守を対象に、画面と帳票の投影を作る。帳票は、テンプレート・項目の束縛・改ページ・繰返し・集計を宣言し、レイアウトはテンプレート側へ置く。成果物から入力の版を辿れるようにする。
- **成果物**: 投影の宣言、画面と帳票の一致試験、manifest、負例。
- **受入**: AC-7に対応する確認が取れる。テンプレートの変更で意味の差分が出ない。
- **依存**: WP3、WP5。
- **既存の担当**: [#1130](https://github.com/hat47x/tei-platform/issues/1130)、[#1803](https://github.com/hat47x/tei-platform/issues/1803)。
- **危険度**: A2。
- **停止条件**: 帳票の表現力が不足し、レイアウトを正本へ戻す必要が出た場合は、テンプレートの契約を見直す。

### WP7 パッケージ化の判断

- **目的**: WP3〜WP6で反復した組合せだけを、Patternパッケージとして切り出す。
- **内容**: 二つ以上の案件または組合せで同じ構造が繰り返されたかを確認する。繰り返しが確認できた場合だけ、パッケージの契約（構成、必須の対話契約、投影、適合試験、取り外し）を定義する。契約項目と合成規則は[受容アーキテクチャのD1.1](../design/XTEA_LOWCODE_RECEPTION_ARCHITECTURE_2026-11.md)を出発点にする。確認できない場合は、5 Kindの組合せのままにする。
- **成果物**: パッケージ化の判断記録、または見送りの記録。
- **受入**: AC-1に対応する確認が取れる。パッケージの追加がCoreの要素種別を要求しない。
- **依存**: WP3、WP6。
- **危険度**: A2。採用はA4。
- **停止条件**: パッケージ間の重複が8割を超える場合は、パッケージ化を見送り、組合せの標準化へ戻す。

### WP8 標準業務意味の研究

- **目的**: 暦・会計期間・税・通貨・採番のうち、どれを製品の意味として扱うかを決める材料を作る。
- **内容**: [受容アーキテクチャのD6.1](../design/XTEA_LOWCODE_RECEPTION_ARCHITECTURE_2026-11.md)の推奨に従い、暦と会計期間を意味として試し、営業日・休日・年度・月度・締め・期間の状態を、適用期間と評価時点を持つ宣言として表せるか調べる。税・通貨・採番は、PluginのCapabilityとDataの契約として試す。二つ以上の反復、生成器・帳票からの構造参照、実行先間の一致、業務上の可視性のいずれかを満たした場合だけ、意味へ昇格する。法制度・業種差の調査は、必要最小限に限る。
- **成果物**: 研究記録、意味の候補、提供元の候補、負例、昇格条件の判定記録。暦・会計期間の宣言の形と、税・通貨・採番の契約は[標準業務の受容設計](../design/XTEA_STANDARD_BUSINESS_SEMANTICS_2026-11.md)にある。
- **受入**: 適用期間の境界試験ができる。案件固有の規則をCoreの変更なしに差し替えられる。昇格の条件を満たした場合だけ、Coreの意味を追加している。
- **依存**: WP2のD6の判断。
- **危険度**: A0/A1。採用はA4。
- **停止条件**: 法制度・業種差の調査が発散する場合は、暦と会計期間だけに絞る。

### WP9 作者診断のPoC

- **目的**: 未定義参照、未被呼出し、未使用の表、三視点の抜け、対話契約の不整合を、受理済み意味の診断として出せるか確かめる。
- **内容**: 既存の検証の上に、三視点の行列と対話契約の検査を加える。診断は修正案を作れるが、自動で受理しない。負例で偽陽性・偽陰性を確認する。
- **成果物**: 診断の一覧、負例、修正案の受理境界の記録。
- **受入**: AC-9に対応する確認が取れる。作者環境の面ごとの検査と、生成物を未受理で置く境界はAC-12として分けて確認する（[作者環境の受容設計](../design/XTEA_AUTHORING_SURFACE_RECEPTION_2026-11.md)）。
- **依存**: WP3。
- **既存の担当**: [#1802](https://github.com/hat47x/tei-platform/issues/1802)、[#1124](https://github.com/hat47x/tei-platform/issues/1124)、[#1125](https://github.com/hat47x/tei-platform/issues/1125)。
- **危険度**: A2。
- **停止条件**: 診断の維持コストが見つかる問題の価値を上回る場合は、未定義参照と三視点の抜けに絞る。

### WP10 統合と引継ぎ

- **目的**: 文書・索引・Issueの参照をそろえ、次の担当が再開できる状態にする。
- **内容**: 設計・要件・計画の索引を更新する。関連Issueへ、成果物と未解決事項を記録する。採用された判断はADRへ、実装はIssueへ渡す。
- **成果物**: 更新された索引、Issueの記録、引継ぎの要約。
- **受入**: 索引から全成果物へ到達できる。未解決事項が、何が分かれば決まるかとともに残る。
- **依存**: WP1〜WP9のうち、実施したもの。
- **危険度**: A0/A2。
- **停止条件**: 並行作業と衝突する場合は、その変更だけを保留し、衝突の内容をPRへ記録して他の作業を続ける。

## 5. 順序とゲート

```text
WP1 知識目録の固定
  ↓ G0: 根拠リンク・区分・非公開情報の点検
WP2 受容アーキテクチャのレビュー
  ↓ G1: D1〜D8とRQ候補の採否（人）
WP3 最小の対話契約
  ↓ G2: Coreを増やさず、意味と表示が分かれた
  ├─ WP4 フック
  └─ WP5 結果型
        ↓ G3: 拡張点と結果型が、負例つきで成立した
WP6 親子と帳票
  ↓ G4: 同じ契約から複数の投影が出た
WP7 パッケージ化の判断
  ↓ G5: 反復が確認できた場合だけパッケージ化
WP8 標準業務意味（並行可）
WP9 作者診断（並行可）
  ↓
WP10 統合と引継ぎ
```

ゲートの意味は次のとおりです。

| ゲート | 何を確認するか | 通らない場合 |
| --- | --- | --- |
| G0 | 事実と推論の区分、根拠の解決、非公開情報の不在 | 該当項目を未確認へ戻す |
| G1 | 設計判断と要求候補の採否 | 保留のまま、WP3へ進まない |
| G2 | Coreの要素種別を増やさずに、意味と表示が分かれた | 語彙を減らすか、既存の仕組みへ寄せる |
| G3 | 宣言外の効果を拒否し、失敗を区別した | フックと結果型の設計を見直す |
| G4 | 同じ契約から複数の提示形態が出た | 帳票の契約を見直す |
| G5 | 組合せの反復が確認できた | パッケージ化を見送る |

## 6. 検証の方針

- 各パッケージは、**合成fixture**と**負例**を持つ。負例は、境界を破る入力（未宣言能力、読取専用への書込み、0件更新、複数一致、版の不一致）を含める。
- 同じ意味と評価時点から、同じ結果を返すことを、再実行で確認する。
- 画面・帳票・CLIの結果を比較する場合は、意味と検証結果を比較し、見た目の一致を意味の一致と呼ばない。
- 生成物は、入力の版（意味、対話契約、テンプレート、生成器）をmanifestに記録する。
- 合成の成功を、実Provider・実DB・本番の証拠にしない。実環境の確認は、別のIssueと検証根拠で行う。
- 旧Runtimeとの比較は、[#940](https://github.com/hat47x/tei-platform/issues/940)の観測が得られてから行う。ソースの読解を比較結果と呼ばない。

## 6.1 文書の検証記録（2026-11-01、同日再点検）

4文書を対象に、機械的に確認できる範囲を点検しました。

| 項目 | 結果 | 手段と限界 |
| --- | --- | --- |
| X-TEAの出典参照 | 152件。ファイルの欠落0、行範囲の超過0 | `D:`/`E:`/`M:`/`S:`のファイル存在と行数を照合。内容の意味的な正しさは別に読む |
| 現行TEIの参照 | 21件。ファイルの欠落0、行範囲の超過0 | `internal/`のGoファイル存在と行数を照合 |
| 文書PRの状態 | #2519、CI全項目pass、`MERGEABLE`/`CLEAN` | 2026-11-02時点。Draftのまま、mainへは未統合 |
| PoC PRの状態 | #2521、CI全項目pass、`MERGEABLE`/`CLEAN` | 2026-11-02時点。`go test ./internal/application ./internal/applicationview ./internal/semantic ./internal/syntax ./internal/validation ./cmd/tei`が成功 |
| 相対リンク | 切れ0 | リンク先のパス存在を確認。アンカーと外部URLの到達は未確認 |
| 表の列数 | 不整合0 | 連続する表の列数を比較 |
| 改行 | LFで統一 | CRLFの混入を確認 |
| 非公開情報 | ローカルパス・非公開名・集計の混入0 | 既知のパターンを検索。未知の混入は人が確認する |
| Issueの状態 | 参照したIssueの状態を2026-11-01時点で確認 | 状態は変わり得る。統合の直前に再確認する |

対象は4文書、1,573行です。知識項目104件のうち、設計判断を動かす58件を付録Aで対応づけ、残りは層の割当（RQ/CORE/PAT/GEN/PLG/STU/EVI）を持ちます。見出しの重複は0です。

この記録は、文書の参照が解決することを示すもので、記述の意味が正しいことや、設計が採用されたことを示しません。

## 7. リスクと撤回

| リスク | 現れ方 | 対処 | 撤回の条件 |
| --- | --- | --- | --- |
| 過剰な抽象化 | パッケージ・フック・役割の語彙が実装より複雑になる | [#1973](https://github.com/hat47x/tei-platform/issues/1973)の基準で、何が可能になったかを問う | 語彙を減らしても同じ仕事ができる |
| パッケージの増殖 | 組合せごとにパッケージが増える | 二つ以上の反復を条件にする | 重複が8割を超える |
| フックの複雑化 | フックの種類が機能型の再実装になる | 位相と範囲を限定する | Patternパッケージ側へ移す |
| 標準意味の範囲拡大 | 法制度・業種差の調査が終わらない | 暦と会計期間に絞る | 製品の約束にしない |
| ソース読解の過信 | 静的読解を実行観測として扱う | 事実と推論を分ける | 実行観測が得られない項目は未確認に戻す |
| 非公開情報の混入 | パス・名称・集計が文書へ入る | 公開upstreamの位置だけを書く | 混入が疑われる記述を削除する |
| 並行作業との衝突 | 既存Issueの設計と矛盾する | 統合前に最新のIssue・PRを確認する | 矛盾が解けない場合は統合しない |

## 8. この計画でやらないこと

- X-TEAの画面・書式・機能型の再現。
- 旧Runtimeの実行結果の取得（[#940](https://github.com/hat47x/tei-platform/issues/940)）。
- 参照モデル・手引書の内容の転載（[#941](https://github.com/hat47x/tei-platform/issues/941)）。
- 要求本文の、製品としての最終的な採否（PoCの作業では案を更新して進める）。
- 実装コードの一括変更。実装は、対応Issueの受入条件に従って小さく行う。
- 新しい判定段階（Gate）の追加。既存のGateと混同しない。

## 9. 次の判断

人が決める必要がある事項です。

1. `RQ-C1`〜`RQ-C3`の採否と、AC-1〜AC-11のうちどれを既存要求へ加えるか。
2. D1（パッケージ）・D2（フック）・D3（項目の役割）の採用範囲。最初のPoCをどこまでにするか。
3. D6（標準業務意味）の対象を、暦と会計期間に絞るか、税・通貨・採番まで含めるか。
4. 既存の参照モデル・旧資産を、受容試験のfixtureに使える範囲（[#941](https://github.com/hat47x/tei-platform/issues/941)）。
5. この計画を、既存Issueのどれへ結び付けるか（新規Issueが必要か）。

### 9.1 新規Issue候補（案、未起票）

上の判断の後で必要になる作業の候補です。必要性が確認できるまで、GitHubへ起票しません。

| 候補 | 目的 | 主な範囲 | 受入の手掛かり | 依存 |
| --- | --- | --- | --- | --- |
| N-1 対話契約とパッケージ | Patternの5種を変えずに、項目の並び・参照入力・遷移を宣言し、組合せをパッケージとして再利用する | 構文、検証、CLI/Web投影、取り外し | 付録BのP-1・P-2、R-1・R-2 | D1・D2.1・D3.1の採否。最小PoCは[#2521](https://github.com/hat47x/tei-platform/pull/2521) |
| N-2 対話フック | 宣言と規則Pluginで、行検証・導出・参照補助を表す | 位相・範囲・効果、Plugin契約、未宣言能力の拒否 | 付録BのH-1〜H-4 | N-1、D2.1 |
| N-3 結果の分類と削除効果 | 業務上の結末と実行の失敗の分担、0件・複数・競合・読取専用・権限外、`delete`効果 | 結果型、負例、観測 | 付録BのO-1〜O-5 | D5 |
| N-4 出力投影と帳票 | 同じ対話契約から画面と帳票を出す | 投影宣言、テンプレートPlugin、manifest | 付録BのT-1〜T-3 | N-1、[#1803](https://github.com/hat47x/tei-platform/issues/1803)、[#2016](https://github.com/hat47x/tei-platform/issues/2016) |
| N-5 暦・会計期間 | 適用期間と評価時点を持つ暦・期間の意味 | 意味、提供元、境界試験 | D6.1の受入条件 | D6.1の採否 |
| N-6 受理済み意味の診断 | 未定義参照、未被呼出し、三視点の抜け、配置の重複 | 診断一覧、負例、修正案の受理境界 | 付録BのG-1〜G-4 | N-1、[#1802](https://github.com/hat47x/tei-platform/issues/1802) |

## 10. 関連

- [X-TEAのローコード機構に関する知識目録](../research/XTEA_LOWCODE_KNOWLEDGE_INVENTORY_2026-11.md)
- [X-TEA知見の受容アーキテクチャ](../design/XTEA_LOWCODE_RECEPTION_ARCHITECTURE_2026-11.md)
- [X-TEA知見の受容要件](../../requirements/XTEA_LOWCODE_RECEPTION_REQUIREMENTS_2026-11.md)
- [機能移植対応表](../design/AUTHORING_TOOL_FEATURE_TRANSFER_MAP_2026-10.md)
- [生成AIネイティブなローコード開発基盤の製品戦略](../design/GENERATIVE_LOWCODE_PRODUCT_STRATEGY_2026-10.md)
- [開発計画](../README.md)
- [TEI製品要求](../../requirements/TEI_PRODUCT_REQUIREMENTS.md)

## 11. 引継ぎの要点

**解析の結論.** X-TEAの生産性は、8種の閉じた機能型、実行時解釈、暗黙の標準業務ライブラリの組合せから来ています。そのままTEIへ移すと、機能型の特権化、正本への任意コード混入、標準規則の不可視化を招きます。知見は「何を実現していたか」として回収し、実現方式は受容しません。

**受容の核.**

- 機能型は、5種のPatternの組合せと対話契約で表す（D1）。
- フックは宣言だけを正本に置き、本体は規則Pluginが供給する（D2・D2.1）。
- 項目の意味・対話・表示を分ける（D3・D3.1）。
- 取引の境界をCommandに置き、業務上の結末と実行の失敗を分ける（D5）。
- 標準業務意味は、暦と会計期間から始める（D6.1）。
- 帳票は同じ対話契約からの投影とし、生成器のreceiptに版を残す（D7）。

**成果物.**

| 成果物 | 内容 |
| --- | --- |
| 知識目録 | 105項目、出典152件、読んだ範囲と未読範囲 |
| 受容アーキテクチャ | D1〜D8とD1.1・D1.2・D2.1・D3.1・D6.1、付録A（対応）・B（試験）・C（現行文法との差） |
| 受容要件 | AC-1〜AC-11、RQ-C1〜C3、既存要求との重複確認 |
| 移行計画 | WP1〜WP10、G0〜G5、Issue候補N-1〜N-6、この引継ぎ |
| 文書PR | [#2519](https://github.com/hat47x/tei-platform/pull/2519)（`e0ea7565f` でmainへ統合） |
| PoC PR | [#2521](https://github.com/hat47x/tei-platform/pull/2521)（`53f5bb425` でmainへ統合） |
| 追加の実装PR | [#2679](https://github.com/hat47x/tei-platform/pull/2679) Query入力、[#2682](https://github.com/hat47x/tei-platform/pull/2682) Command実行境界、[#2683](https://github.com/hat47x/tei-platform/pull/2683) フック宣言、[#2684](https://github.com/hat47x/tei-platform/pull/2684) 未使用属性、[#2685](https://github.com/hat47x/tei-platform/pull/2685) 印刷受領票、[#2686](https://github.com/hat47x/tei-platform/pull/2686) フック実行境界、[#2687](https://github.com/hat47x/tei-platform/pull/2687) layout、[#2688](https://github.com/hat47x/tei-platform/pull/2688) 統合試験、[#2689](https://github.com/hat47x/tei-platform/pull/2689) 実装状態、[#2691](https://github.com/hat47x/tei-platform/pull/2691) role、[#2693](https://github.com/hat47x/tei-platform/pull/2693) commandビュー、[#2696](https://github.com/hat47x/tei-platform/pull/2696) Packageビュー、[#2697](https://github.com/hat47x/tei-platform/pull/2697) Applicationビュー、[#2698](https://github.com/hat47x/tei-platform/pull/2698) 取引境界、[#2700](https://github.com/hat47x/tei-platform/pull/2700) 削除効果、[#2701](https://github.com/hat47x/tei-platform/pull/2701) 帳票テンプレート、[#2703](https://github.com/hat47x/tei-platform/pull/2703) 期間投影、[#2705](https://github.com/hat47x/tei-platform/pull/2705) 参照実装、[#2712](https://github.com/hat47x/tei-platform/pull/2712) 参照実行、[#2713](https://github.com/hat47x/tei-platform/pull/2713) SQL参照検査、[#2718](https://github.com/hat47x/tei-platform/pull/2718) 期間表示、[#2719](https://github.com/hat47x/tei-platform/pull/2719) on-delete、[#2723](https://github.com/hat47x/tei-platform/pull/2723) 閉期間拒否、[#2724](https://github.com/hat47x/tei-platform/pull/2724) 削除ポリシー検査、[#2727](https://github.com/hat47x/tei-platform/pull/2727) 参照Store、[#2729](https://github.com/hat47x/tei-platform/pull/2729) effectcommit方針、[#2733](https://github.com/hat47x/tei-platform/pull/2733) Studioビュー、[#2741](https://github.com/hat47x/tei-platform/pull/2741) Studio renderer、[#2747](https://github.com/hat47x/tei-platform/pull/2747) Studio Command、[#2751](https://github.com/hat47x/tei-platform/pull/2751) ビュー表、[#2753](https://github.com/hat47x/tei-platform/pull/2753) 削除ポリシー実行、[#2755](https://github.com/hat47x/tei-platform/pull/2755) Studio Package、[#2757](https://github.com/hat47x/tei-platform/pull/2757) 必須属性、[#2759](https://github.com/hat47x/tei-platform/pull/2759) 値域、[#2762](https://github.com/hat47x/tei-platform/pull/2762) 重複create、[#2764](https://github.com/hat47x/tei-platform/pull/2764) 実装状況、[#2766](https://github.com/hat47x/tei-platform/pull/2766) キー指定、[#2768](https://github.com/hat47x/tei-platform/pull/2768) 期間状態CLI、[#2770](https://github.com/hat47x/tei-platform/pull/2770) 必須属性修正、[#2773](https://github.com/hat47x/tei-platform/pull/2773) effectcommitアダプタ、[#2799](https://github.com/hat47x/tei-platform/pull/2799) 複合キー、[#2803](https://github.com/hat47x/tei-platform/pull/2803) Studio期間実行（すべてmainへ統合） |

**実装済みの範囲（PoC）.**

| 範囲 | 状態 | 参照 |
| --- | --- | --- |
| 対話契約（`item`・`lookup`・`navigate`・`command`） | 実装・検証・CLI投影 | 付録C.5、`internal/application` |
| 表示解決（型・表示名・入力部品・必須・更新禁止・値域） | 実装 | 付録C.5、`internal/applicationview` |
| 画面HTML・印刷用の用紙面 | 実装 | 付録C.5、`view -format html/print` |
| header-detail（見出し・明細） | 実装 | 付録C.5 |
| 一覧（browse）ビュー | 実装 | 付録C.5 |
| 一覧の絞り込み（項目・演算子） | 実装 | 付録C.5 |
| Query入力と絞り込みの結びつけ | 実装 | 付録C.5 |
| Query入力のSQL参照検査 | 実装 | 付録C.5 |
| 一覧の並び順（`sort`） | 実装 | 付録C.5 |
| 一覧のページ件数（`page-size`） | 実装 | 付録C.5 |
| Commandの起動計画（入力の過不足） | 実装 | 付録C.5 |
| command Patternのビュー | 実装 | 付録C.5 |
| Commandの実行境界（Executor契約） | 実装 | 付録C.5 |
| Commandの取引境界（Unit of Work） | 実装 | 付録C.5 |
| 削除の効果（`delete`） | 実装 | 付録C.5 |
| 参照削除の振る舞い（`on-delete`） | 実装 | 付録C.5 |
| 削除効果の参照ポリシー検査 | 実装 | 付録C.5 |
| Commandの参照実装 | 実装 | 付録C.5 |
| Commandからeffectcommitへのアダプタ | 実装 | 付録C.5 |
| Command実行前の認可契約 | 実装 | 付録C.5 |
| CLI参照実行の認可AllowList | 実装 | 付録C.5 |
| Command効果の宣言照合 | 実装 | 付録C.5 |
| Command実行の記録契約 | 実装 | 付録C.5 |
| CLI実行記録 | 実装 | 付録C.5 |
| 適用操作の照合 | 実装（`Differences`を含む） | 付録C.5 |
| 診断の修正案 | 実装 | 付録C.5 |
| 能力の宣言許可の照合 | 実装 | 付録C.5 |
| CLIの失敗符号 | 実装 | 付録C.5 |
| 読み取り専用datasourceの事前検査 | 実装 | 付録C.5 |
| 画面・帳票の版 | 実装 | 付録C.5 |
| フックの宣言外呼出し | 実装 | 付録C.5 |
| フックの呼出しのCommand境界への委譲 | 実装 | 付録C.5、`internal/hook`（[#3108](https://github.com/hat47x/tei-platform/pull/3108)） |
| フック実行からの宣言制限呼出し元の配線 | 実装（参照） | 付録C.5、`internal/hook`、[確認記録](../evidence/XTEA_LOWCODE_HOOK_RUNNER_OUTCOME_2026-11.md) |
| フックの不成立と実行失敗の区別 | 実装 | 付録C.5、`internal/hook`、[確認記録](../evidence/XTEA_LOWCODE_HOOK_RUNNER_OUTCOME_2026-11.md) |
| 判定と能力呼出しの連結（`Guarded`） | 実装（参照） | 付録C.5、`internal/hook` |
| 能力から`command` Patternの解決（`CommandPatternFor`） | 実装 | 付録C.5、`internal/hook` |
| フック宣言の画面・印刷面への反映 | 実装 | 付録C.5、`internal/applicationview`、[確認記録](../evidence/XTEA_LOWCODE_HOOK_PRINT_SURFACE_2026-11.md) |
| 参照Storeの拒否符号 | 実装 | 付録C.5 |
| 作者環境の面ごとの検査（AC-12） | 設計のみ | 作者環境の受容設計§3.1〜3.4 |
| 派生出力の版・未解決（AC-7） | 設計のみ | 派生出力の受容設計§3.1〜3.5 |
| 標準業務の宣言（RQ-C3） | 一部実装（暦・期間キー・税・通貨・採番） | 標準業務の受容設計§3.1〜3.7、§5.1 |
| 参照データ記憶域（MemoryStore） | 実装 | 付録C.5 |
| 参照Storeの削除ポリシー実行 | 実装 | 付録C.5 |
| 参照Storeのキー対応削除 | 実装 | 付録C.5 |
| 参照Storeの複数段カスケード | 実装 | 付録C.5 |
| 参照Storeの削除ロールバック | 実装 | 付録C.5 |
| interactionなしPatternのビュー | 実装 | 付録C.5 |
| 参照Storeの必須属性検査 | 実装 | 付録C.5 |
| 参照Storeの値域検査 | 実装 | 付録C.5 |
| 参照Storeの重複create拒否 | 実装 | 付録C.5 |
| 参照Storeのキー指定と更新禁止 | 実装 | 付録C.5 |
| 参照Storeの複合キー | 実装 | 付録C.5 |
| 参照Storeの候補キー一意性 | 実装 | 付録C.5 |
| 候補キーの型正規化比較 | 実装 | 付録C.5 |
| Commandの参照実行（CLI） | 実装 | 付録C.5 |
| Command計画の期間表示 | 実装 | 付録C.5 |
| 期間の閉塞によるCommand拒否 | 実装 | 付録C.5 |
| CLI参照実行の期間状態 | 実装 | 付録C.5 |
| フックの宣言（位相・読み書き・失敗時） | 実装 | 付録C.5 |
| フックの範囲（行・レコード・項目）の宣言と拒否 | 実装（参照） | 付録C.5、`internal/application`・`internal/validation`・`internal/hook`（行単位は`RunRows`、ホストからの呼出しは未実装）、[確認記録](../evidence/XTEA_LOWCODE_HOOK_SCOPE_2026-11.md) |
| 行単位フックの`uses`の判定の一本化 | 実装（参照） | 付録C.5、`internal/hook`、[確認記録](../evidence/XTEA_LOWCODE_HOOK_ROW_USES_PARITY_2026-11.md) |
| 行単位フックの`uses`の実行（`RunRowCalls`） | 実装（参照） | 付録C.5、`internal/hook`、[確認記録](../evidence/XTEA_LOWCODE_HOOK_ROW_USES_CALLS_2026-11.md) |
| Data型パラメータへ渡す属性の規約 | 確認済み（限定なし） | 付録C.5、`internal/validation`・`internal/hook`、[確認記録](../evidence/XTEA_LOWCODE_DATA_PARAM_CONVENTION_2026-11.md) |
| `after-*`へ到達する実行経路（フック層） | 実装（参照） | 付録C.5、`internal/hook`、[確認記録](../evidence/XTEA_LOWCODE_HOOK_AFTER_CHANGE_2026-11.md) |
| `after-*`へ到達するCLI配線（`-reference-store`） | 実装（参照） | 付録C.5、`cmd/tei`、[確認記録](../evidence/XTEA_LOWCODE_HOOK_AFTER_CHANGE_CLI_2026-11.md) |
| `after-*`拒否時の効果の取り消し | 実装（参照・既定経路は未到達） | 付録C.5、`cmd/tei`、[確認記録](../evidence/XTEA_LOWCODE_HOOK_AFTER_CHANGE_REVERT_2026-11.md) |
| Data型パラメータの結び付け検査 | 未決（意味の設計判断） | 付録C.5、`internal/validation`、[確認記録](../evidence/XTEA_LOWCODE_DATA_PARAM_CONVENTION_2026-11.md) |
| 複合キーの表現 | 確認済み（複数パラメータ） | 付録C.5、`internal/validation`・`internal/command`・`internal/hook`、[確認記録](../evidence/XTEA_LOWCODE_COMPOSITE_KEY_EXPRESSION_2026-11.md) |
| 同時適用の整合（参照実装） | 対象外と確認 | 付録C.5、`internal/command`・`cmd/tei`、[確認記録](../evidence/XTEA_LOWCODE_OUTCOME_SHARED_EFFECT_2026-11.md) |
| プロセスを跨ぐ結末の検査 | 確認済み | 付録C.5、`internal/command`・`cmd/tei`、[確認記録](../evidence/XTEA_LOWCODE_OUTCOME_CROSS_PROCESS_2026-11.md) |
| `uses`呼出しの効果の補償 | 不要と確認 | 付録C.5、`internal/hook`・`internal/command`、[確認記録](../evidence/XTEA_LOWCODE_HOOK_USES_COMPENSATION_2026-11.md) |
| Studio経路での`after-*`拒否時の取り消し | 実装（参照・既定経路は未到達） | 付録C.5、`internal/studiohost`、[確認記録](../evidence/XTEA_LOWCODE_HOOK_AFTER_CHANGE_STUDIO_REVERT_2026-11.md) |
| 規則Pluginの発見・束縛・版の契約 | 未決（設計文書がD2採用時に決める。境界と差込み口は実装済み） | 付録C.5、`internal/hook`、[確認記録](../evidence/XTEA_LOWCODE_RULE_PLUGIN_CONTRACT_2026-11.md) |
| `after-*`へ到達するStudio配線 | 実装（参照・UIからは任意指定で呼ぶ） | 付録C.5、`internal/studiohost`・`internal/hook`、[確認記録](../evidence/XTEA_LOWCODE_HOOK_AFTER_CHANGE_STUDIO_2026-11.md) |
| 実行時の値変換 | 不要と確認 | 付録C.5、`internal/validation`、[確認記録](../evidence/XTEA_LOWCODE_VALUE_CONVERSION_2026-11.md)（受理条件が同一の下位原始型を要求するため、変換を要する組は通らない） |
| 行単位のフックの実行（`RunRows`・`applyReachableRows`） | 実装（参照） | 付録C.5、`internal/hook`（二つの行経路が併存し`uses`の扱いが違う。`Runner.Rows`を設定するホストは未実装）、[確認記録](../evidence/XTEA_LOWCODE_HOOK_ROW_REACHABILITY_2026-11.md) |
| フックの位相の重複宣言の拒否 | 実装 | 付録C.5、`internal/validation` |
| フックの位相順序と途中停止の記録（`RunSequence`） | 実装（参照） | 付録C.5、`internal/hook`、[確認記録](../evidence/XTEA_LOWCODE_HOOK_PHASE_SEQUENCE_2026-11.md) |
| フックの宣言した読み取りだけの授受（`ReadsFor`） | 実装 | 付録C.5、`internal/hook`、[確認記録](../evidence/XTEA_LOWCODE_HOOK_DECLARED_READS_2026-11.md) |
| 位相ごとの読み取り範囲（`RunSequence`） | 実装（参照） | 付録C.5、`internal/hook`、[確認記録](../evidence/XTEA_LOWCODE_HOOK_PER_PHASE_READS_2026-11.md) |
| 提案された値の宣言照合（`ProducesFor`） | 実装 | 付録C.5、`internal/hook`、[確認記録](../evidence/XTEA_LOWCODE_HOOK_DECLARED_PRODUCES_2026-11.md) |
| フック宣言の機能定義書への反映 | 実装 | 付録C.5、`internal/artifact`、[確認記録](../evidence/XTEA_LOWCODE_HOOK_FUNCTION_DEFINITION_2026-11.md) |
| CLI参照実行からのフック呼出し | 実装（参照） | 付録C.5、`cmd/tei`、[確認記録](../evidence/XTEA_LOWCODE_HOOK_CLI_WIRING_2026-11.md) |
| Studio参照実行からのフック呼出し | 実装（参照） | 付録C.5、`internal/studiohost`、[確認記録](../evidence/XTEA_LOWCODE_HOOK_STUDIO_WIRING_2026-11.md) |
| フック位相の到達範囲（`RunReachable`） | 実装（参照） | 付録C.5、`internal/hook`、[確認記録](../evidence/XTEA_LOWCODE_HOOK_REACHABLE_PHASES_2026-11.md) |
| フックの結末名とCommand拒否名の対応 | 実装（参照） | 付録C.5、`internal/hook`、[確認記録](../evidence/XTEA_LOWCODE_HOOK_OUTCOME_VOCABULARY_2026-11.md) |
| フックが名乗る結末名の語彙照合 | 実装（参照） | 付録C.5、`internal/hook`、[確認記録](../evidence/XTEA_LOWCODE_HOOK_OUTCOME_CHECK_2026-11.md) |
| 到達できる位相の経路での提案の照合（`RunReachable`） | 実装（参照） | 付録C.5、`internal/hook`、[確認記録](../evidence/XTEA_LOWCODE_HOOK_REACHABLE_PROPOSES_2026-11.md) |
| フックの提案のCLI・Studio表示 | 実装（参照） | 付録C.5、`cmd/tei`、`cmd/tei-studio`、[確認記録](../evidence/XTEA_LOWCODE_HOOK_SHOW_PROPOSALS_2026-11.md) |
| 宣言した能力を呼ばないまま実行済みと報告しない（`RunReachable`） | 実装（参照） | 付録C.5、`internal/hook`、[確認記録](../evidence/XTEA_LOWCODE_HOOK_DECLARED_CALL_HONESTY_2026-11.md) |
| 宣言した能力を実際に呼ぶ配線（`RunDeclaredCalls`） | 実装（参照） | 付録C.5、`internal/hook`、`cmd/tei`、`internal/studiohost`、[確認記録](../evidence/XTEA_LOWCODE_HOOK_DECLARED_CALL_WIRING_2026-11.md) |
| フックの`reads`から能力のパラメータへの対応（`BindInputsFromReads`） | 実装（参照） | 付録C.5、`internal/hook`、[確認記録](../evidence/XTEA_LOWCODE_HOOK_READS_TO_PARAMS_2026-11.md) |
| 能力入力の結び付けの型検査 | 実装 | 付録C.5、`internal/validation`、[確認記録](../evidence/XTEA_LOWCODE_HOOK_BINDING_TYPE_CHECK_2026-11.md) |
| 能力入力の結び付けのData対応 | 実装 | 付録C.5、`internal/validation`、[確認記録](../evidence/XTEA_LOWCODE_HOOK_BINDING_DATA_MATCH_2026-11.md) |
| 位相ごとのフック実装の供給（`Runner.ByPhase`） | 実装（参照） | 付録C.5、`internal/hook`、[確認記録](../evidence/XTEA_LOWCODE_HOOK_PHASE_IMPLEMENTATION_2026-11.md) |
| 複数の結末を持つ能力の未確定の報告 | 実装（参照） | 付録C.5、`internal/command`、[確認記録](../evidence/XTEA_LOWCODE_MULTI_OUTCOME_2026-11.md) |
| フック結末名とCommand拒否名の対応の実出力 | 実装（参照） | 付録C.5、`internal/hook`、`cmd/tei`、`internal/studiohost`、[確認記録](../evidence/XTEA_LOWCODE_OUTCOME_CORRESPONDENCE_USE_2026-11.md) |
| フックが使う能力の`command` Pattern不在の拒否 | 実装 | 付録C.5、`internal/validation` |
| フックの実行境界（reject/warn） | 実装 | 付録C.5 |
| 未使用属性の表示 | 実装 | 付録C.5 |
| 属性の役割（role） | 実装 | 付録C.5 |
| 項目の並び（layout） | 実装 | 付録C.5 |
| 印刷の受領票 | 実装 | 付録C.5 |
| 帳票テンプレートの版 | 実装 | 付録C.5 |
| Packageのビュー | 実装 | 付録C.5 |
| Applicationのビュー | 実装 | 付録C.5 |
| StudioホストのビューAPI | 実装 | 付録C.5 |
| Studio rendererのビュー表示 | 実装 | 付録C.5 |
| StudioからのCommand計画・参照実行 | 実装 | 付録C.5 |
| Studioからの期間状態付き参照実行 | 実装 | 付録C.5 |
| Studioの認可AllowList | 実装 | 付録C.5 |
| Studio実行記録 | 実装 | 付録C.5 |
| 既存認可規則との接続 | 実装（`CommandAuthorizer`） | 付録C.5 |
| Studioのビュー要約表示 | 実装 | 付録C.5 |
| StudioのPackageビュー | 実装 | 付録C.5 |
| Applicationビューの期間 | 実装 | 付録C.5 |
| 印刷の改ページ（`page-break`） | 実装 | 付録C.5 |
| Package（宣言・検証・版束縛・衝突・ビュー表示） | 実装 | 付録C.5、`internal/application`・`internal/validation` |
| Semantic fingerprint（投影の意味の版） | 実装 | 付録C.5、`internal/semantic` |
| 統合試験（Semantic→Application→Package→view/print） | 実装 | `cmd/tei/product_slice_test.go` |
| 残る未実装 | 実Provider接続、集計の実データ計算、削除の効果、帳票テンプレートの版管理、フックの実PluginとCLI・Studioへの配線 | 付録C.5、D5・D7 |

**文書の読み方.**

1. 知識目録で、機構・効果・破綻・根拠を見る。
2. 受容アーキテクチャの§1.1で判断の要約を見て、D1〜D8の該当節を読む。
3. 受容要件で、受入条件と要求候補を見る。
4. この計画で、作業パッケージ・ゲート・停止条件を見る。
5. 付録Aで対応、付録Bで試験、付録Cで現行文法との差を見る。

**統合の経緯.** 仮承認の下で統合を進めた過程で、PR番号の取り違えにより[#2690](https://github.com/hat47x/tei-platform/pull/2690)・[#2699](https://github.com/hat47x/tei-platform/pull/2699)・[#2702](https://github.com/hat47x/tei-platform/pull/2702)（いずれも期間機能の文書・実装）を意図せず統合しました。内容を点検し、期間機能の範囲を出ていないことを確認しています。取り消しはしていません。

**人の確認が必要な事項.** §9の5項目です。利用者の指示により、これらは**仮承認**として扱い、実装と統合まで進めました。要求の最終採否、D1〜D8の採用範囲、D6の対象、fixtureの利用条件、Issueへの結び付けは、別の担当の判断として残ります。この判断は開発作業を止める条件ではありません。既定案で進め、判断が変わったときに戻せる形にします。

**直近の検証.** `main`の健全性CIは、`#2773`の`2f7c775e7`、`#2799`の`605599a6d`、`#2803`の`080caada4`で成功しています。変更したGoパッケージの単体・統合試験、`go build ./...`、`go vet`、`gofmt -l`、Studioの`node --test`も各PRで確認しています。

**未検証.** 旧Runtimeの実行観測（[#940](https://github.com/hat47x/tei-platform/issues/940)）、実DBでの動作、実Providerの接続、配布JARとの同一性、参照モデルの内容（[#941](https://github.com/hat47x/tei-platform/issues/941)）です。文書の記述は静的読解に基づきます。

**次の一歩.** 文書と実装はmainへ統合済みです。次は実Providerの接続、[#940](https://github.com/hat47x/tei-platform/issues/940)の実行観測、統合後のCI完了確認です。実Providerは既存の`internal/security/effectcommit`へ接続し、Commandの`UnitOfWork`を`effectcommit.Runner`へ委譲する方針です。

## 12. レビューの手順

1. 知識目録の§1〜§3で、対象と根拠の範囲を見る。
2. 受容アーキテクチャの§1.1で判断の要約を見て、採用候補のD1・D2.1・D3.1・D6.1の節を読む。
3. 受容要件の§2でACと既存要求の対応、§3でRQ-C1〜C3を見る。
4. この計画の§4〜§7で、作業・ゲート・リスクを見る。
5. 付録Aで知識項目との対応、付録Bで適合試験、付録Cで現行文法との差を見る。

**判断する箇所.** §9の5項目です。特に、D1（パッケージ）、D3.1（必須・更新禁止・値域）、D6.1（暦から始める範囲）は、後続の実装範囲を決めます。

**再現する検証.** 文書のリンクと出典は§6.1の記録に従います。PoCは`go test ./internal/syntax ./internal/validation ./internal/crossview`と`go build ./...`で確認できます。

**PRの関係.** 文書は[#2519](https://github.com/hat47x/tei-platform/pull/2519)、対話契約のPoCは[#2521](https://github.com/hat47x/tei-platform/pull/2521)です。PoCは文書のD1・D3.1の最小部分を実装し、`main`には未統合です。

**統合の条件.** 要求・設計の採否は開発作業の停止条件にしません。PRは、変更範囲の検証結果を本文へ記録し、Draftなら解除してから統合します。CIとレビューは、実行できた範囲を記録します。Draftのまま放置せず、統合できる状態になった時点で解除します。






















































