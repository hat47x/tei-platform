# ADR-0045: 生成器と生成物を、版付き成果物・ライフサイクル・独立証拠の組として扱う

- 状態: 提案中
- 日付: 2026-11-04
- 統合元: [ADR-0034](0034-generator-as-derived-artifact.md) / [ADR-0037](0037-generator-lifecycle-separation.md) / [ADR-0040](0040-generator-revocation-approval-separation.md) / [ADR-0041](0041-generated-artifact-provenance-boundaries.md)（いずれも提案中）
- 関連ADR: ADR-0022 / ADR-0032 / ADR-0036 / ADR-0042 / ADR-0043
- 関連Issue: #2016（closed）/ #2595 / #2513 / #2514 / #1445 / #1016

## 背景

ADR-0034・0037・0040・0041は、どれも#2016の参照実装PoCから段階的に書き起こされた。後の版が前の版の未決を引き継いで書き直しており、「証拠は実行許可にならない」「意味の受理はA4として人へ戻す」という同じ原則が4件に分散している。receiptをArtifact/Build Evidenceへどう載せるかという未決も三重に重なっている。

また、ADR-0034の「Target Profile」は、ADR-0032のComposition Profile、ADR-0036のDelivery Profileと並ぶ三つ目のProfile概念になり、選択・検証の仕組みが並立するおそれがある。4件には「Hostが現在許可している」「既存Host Authority」という表現も残っており、[ADR-0043](0043-target-native-artifacts-without-tei-host.md)（TEI独自Hostを前提としない）と語がずれている。

本ADRは、4件の中核規則を保ったまま一つの判断へ統合する。

## 決定

1. **生成器は版付き成果物とする。** 生成器は、入力契約のidentity/digest、対象、source digest、生成手段、適合試験版を持つ。生成物はその派生物であり、意味の正本ではない（ADR-0034）。
2. **由来を問わず、事前に固定したfixtureで選抜する。** 候補がテンプレート・手書き・生成AI（メタ生成を含む）のいずれでも、同じ期待値fixtureで選ぶ。fixtureは#2619に沿い、第一級のテストデータとして保持する。
3. **4つの版を分ける。** 意味・生成器・生成物・実行許可の版を分ける。生成器の状態は `active / superseded / revoked` とし、reconcileは影響の報告にとどめ、再生成・再選抜を自動では行わない。生成AI候補は契約digestへ束縛する（ADR-0037）。
4. **失効の承認はregistryの編集と分ける。** 承認は別のrecordに固定し、機械検証するのは対象の一致だけとする。承認者のAuthorityの解決は[ADR-0042](0042-decision-witness-authority-qualified-boundary.md)のwitness規則に委ねる（ADR-0040）。
5. **証拠は種類ごとに別のreceiptへ残す。** 選抜・署名・artifact保存・audit・再現性を分け、単一の「信頼済み」flagを作らない（ADR-0041）。
6. **どの証拠も、意味の受理にも実行許可にもならない。** 実行許可は、対象環境で副作用を行う主体（実行環境・Plugin・既存アプリケーション）が強制するものであり、TEI Hostの許可を前提にしない（ADR-0043決定5・6）。
7. **対象Profileを独立の概念にしない。** 生成先の違いは、ADR-0043の対象環境の記述と、ADR-0032のComposition Profile（#1016）の宣言で表す。ADR-0036のDelivery Profileも、その宣言の一つの見方として扱う。

## 統合元ADRの扱い

本ADRを採用した時点で、ADR-0034・0037・0040・0041を「置き換え済み」とする。4件はいずれも提案中であり、採用済みADRの書換え禁止規則には触れない。統合元の背景・参照実装の記述は、履歴として各ファイルに残す。

## 採否の判断

ADR-0034は「#2016で効果が確認できない場合、本ADRは採用しない」としていた。#2016はclosedであり、その結果を根拠に本ADRを採用するかどうかは、意味・権限に関わる判断（A4）としてOwnerが決める。

## 検証と未完了の範囲

- #2595（Generator Selectionの実評価Source束縛）で、決定2・3の選抜と版の束縛を実Sourceで確かめる。
- 別言語の生成器（#2513 Java）で、決定1の成果物識別が言語に依存しないことを確かめる。
- 独立Oracleの対照（#2514）で、fixtureと生成実装の独立性を確かめる。
