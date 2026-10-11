# 移動済み文書の参照調査 — 2026年10月9日

## 対象と結果

対象は`main`のコミット`aaf3994a0f47f2ac18eb310a95a6131e953d36fc`。Gitツリーにある`plan/`直下の移動案内は**22件**（監査記録20件、機能設計2件）で、すべての移動先が存在した。これは**移動先の存在確認**であり、旧パスが参照されなくなった証明ではない。

| 参照の区分 | 確認した場所と扱い |
| --- | --- |
| 現行の参照（修正済み） | [業務手順の独立レビュー依頼](../../plan/BUSINESS_PROCEDURE_BLOCKER_B_INDEPENDENT_REVIEW_HANDOFF_2026-09.md)は、品質モデルの正本`plan/design/`を参照している。 |
| 並行作業の対象 | [業務システム設計の読み方ガイド](../business-system-design-reading-guide.md)には旧パスへのリンクがあり、PR #1059が同じ文書を編集している。競合を避け、この調査では改修しない。 |
| 履歴上の記述 | [D4・O1の検討](../../plan/BUSINESS_PROCEDURE_D4_O1_2026-09.md)、[第二業務資料の検討](../../plan/BUSINESS_PROCEDURE_SECOND_CORPUS_PRESSURE_2026-09.md)、[引継ぎの検討](../../plan/BUSINESS_PROCEDURE_HANDOFF_2026-09.md)には当時の統合時点のパスが記録されている。現在の参照先と取り違えず、機械的に置換しない。 |
| 未調査・未確定 | それ以外の追跡対象ファイル、生成物、リポジトリ外からの参照は無参照を証明できていない。 |

上表は確認できた例であり、すべての参照元の一覧ではない。監査本文の正本は`docs/audits/`、機能設計の正本は`plan/design/`に置く。現在のGate判定や受入条件の権限を、本調査へ移さない。

## 移動案内を削除する前の条件

1. 対象コミットと検索対象を固定し、Markdownのリンク以外にプログラム中の文字列、検査手順、生成設定、成果物からの参照も調べる。
2. 現行の内部参照を正本へ切り替え、歴史上の旧パス表記は意味を保つかどうかを個別に判断する。
3. 検査方法と確認できなかった範囲を残す。内部の検索結果が0件でも、外部参照を含む完全な無参照証明にはしない。
4. 参照先の互換性と並行PRへの影響を確かめる。外部参照を確認できない間は移動案内を保持する。

**今回は移動案内を削除しない。** 信頼できる完全なチェックアウトでのリポジトリ全体の検査も行っていない。

## 追加調査（2026年10月10日）

### 確認した範囲

参照元の点検は`main@0c762cfd103f895c9c2c180e3e911a256858b600`のGitツリーを対象とし、移動案内22件のファイル名を検索した。Markdown文書474件のうち、次の**214件**を取得して調べた。

| 対象 | 確認件数 |
| --- | ---: |
| `plan/issue/`直下のMarkdown | 86 |
| `docs/`直下・`plan/design/`直下・`requirements/`直下のMarkdown | 37 |
| `plan/`直下の業務設計・定義・設計統合・Lane B・上流・第二アプリケーション関連の本文（`BUSINESS*`、`DEFINITION*`、`DERIVED*`、`DESIGN_SYNTHESIS*`、`LANE_B*`、`UPSTREAM*`、`SECOND_APPLICATION*`、移動案内を除く） | 91 |
| **合計** | **214** |

取得失敗は0件。ファイル名の文字列出現と、リンク先として旧パスを使っているかは区別した。

### 確認した参照の分類

- **旧パスに直接向かうMarkdownリンク**：この214件では[業務システム設計の読み方ガイド](../business-system-design-reading-guide.md)で1箇所を確認した。移動先の正本は`plan/design/BUSINESS_PROCEDURE_DESIGN_QUALITY_MODEL_2026-09.md`。同文書を編集するPR #1059に[修正箇所を連絡済み](https://github.com/hat47x/tei-platform/pull/1059#issuecomment-6082495357)であり、本PRでは重複編集しない。
- **正本への参照**：[Issue 000042](../../plan/issue/000042-business-procedure-cross-view-reference.md)、[Issue 000048](../../plan/issue/000048-business-procedure-contract-adr-readiness.md)、[Issue 000050](../../plan/issue/000050-second-realistic-handoff-corpus.md)では移動済み文書名が現れるが、対象リンクは正本パスを参照している。同一のファイル名が見つかっただけで旧パス参照と判定しない。
- **履歴上のパス**：[D4・O1の検討](../../plan/BUSINESS_PROCEDURE_D4_O1_2026-09.md)、[第二業務資料の検討](../../plan/BUSINESS_PROCEDURE_SECOND_CORPUS_PRESSURE_2026-09.md)、[引継ぎの検討](../../plan/BUSINESS_PROCEDURE_HANDOFF_2026-09.md)には当時の統合・改名の説明として旧`plan/`パスがある。現行のリンクではないため、自動置換しない。
- **文書名だけの参照**：移動済み文書の名前だけを記した箇所もある。リンク・実行パス・説明文のどれかを文脈で判断する。

### 未確認範囲と削除判断

今回調べたのはMarkdown474件中214件だけであり、**残り260件のMarkdownは未検査**。Go、JavaScript、設定、生成物、外部リポジトリ等も、参照の不存在を確認していない。GitHubコード検索に結果が出なくても網羅的な無参照証明とはしない。

現行の正本・GateやEvidenceの権限は移さず、**旧パスへの移動案内22件を引き続き保持する**。本調査をRepository Health実行のPASSや安全削除の許可として扱わない。

## 継続調査（2026年10月10日、Markdown全件照合）

### 対象と手順

基準は`main@02e4160a529d69835916193aec83b5f932fef46e`（Git tree `864bcee2afeda9954f4594b084fb78c81432b870`）。追跡対象のMarkdownは**479件**。

前段の214件は`main@0c762cfd103f895c9c2c180e3e911a256858b600`で点検した。今回は対象外だった263件を上記の基準で取得し、基準間で新設された設計文書2件と、既調査範囲で更新のあった9件を追加点検した。これにより479件の各ファイル名について移動済み文書22件の名前の出現を確認した。**すべてのリンクや文字列参照の意味を機械的に検証したという意味ではない**。

今回取得したMarkdownでの取得失敗は0件。検出した出現を、現行参照・正本への参照・過去の統合履歴・移動案内自身のリンクに分けて扱った。

### 正本へ直接変更した現行参照

- [業務システム設計の読み方ガイド](../business-system-design-reading-guide.md)：PR #1059はマージ済みだが、設計品質モデルを旧`plan/`から参照していたため、`plan/design/`へ修正する。
- Actor ContextのR3AP、R3B、R3C、R3D、R3J：受入機構と認可対象の監査について、`plan/`内の移動案内ではなく`docs/audits/`の監査正本を参照する。
- [業務手順の検討](../../plan/ORDER_ENTRY_BUSINESS_PROCEDURE_REFERENCE_EXPERIMENT.md)と[参照モデルの根拠引継ぎ](../../plan/REFERENCE_PATTERN_EVIDENCE_HANDOFF_2026-09.md)：上流設計・記憶分離の監査記録を正本へ向ける。
- [第二業務資料の監査](LANE_B_REALISTIC_BUSINESS_REFERENCE_CORPUS_AUDIT_2026-09.md)：RQ-002の監査記録を`docs/audits/`の正本へ向ける。
- [Reference索引](../../reference/README.md)と[SUI移管のReference案内](../../reference/sui-adoption/README.md)：SUI移管設計を`plan/design/`の正本へ向ける。

これらは本PR内の文書11件、該当文字列12箇所を修正する。監査内容・設計判断・受入権限は変更しない。

### 残す参照と未確認範囲

- 旧パスが当時の作業記録として記載されている箇所は、現在の実行用リンクと区別して維持する。
- 監査索引の相対リンクや移動案内から正本へのリンクは、本来の役割を果たしているため削除しない。
- **Go、JavaScript、シェル、設定・生成物、他リポジトリ、外部利用者の参照は無参照と判定できていない**。Markdown479件の点検のみを互換性廃止の証明へ格上げしない。
- 前段の「PR #1059が編集中」という記述は前段実施時点の履歴である。上記基準時点では同PRはマージ済みだった。

22件の移動案内を削除する判断は引き続き保留する。Repository Healthの実行結果・Gate・Evidenceの状態を本調査によって変更しない。
