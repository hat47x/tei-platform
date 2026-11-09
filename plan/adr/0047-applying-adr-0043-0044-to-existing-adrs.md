# ADR-0047: ADR-0043・ADR-0044を既存ADRへ適用する範囲を定める

- 状態: 提案中
- 日付: 2026-11-04
- 関連ADR: ADR-0005 / ADR-0013 / ADR-0019 / ADR-0021 / ADR-0031 / ADR-0035 / ADR-0039 / ADR-0042 / ADR-0043 / ADR-0044 / ADR-0008〜0015 / ADR-0026 / ADR-0027
- 関連Issue: #2556 / #2622 / #1223 / #974

## 背景

[ADR-0043](0043-target-native-artifacts-without-tei-host.md)は、更新範囲としてADR-0032とADR-0033だけを挙げた。しかし、それ以前のADRにも、TEIのSupervisorやHostが実行を監督する前提、「Host」を実行許可の主体とする表現、Studioの認証済み利用者を前提にする候補が残っている。

これらのADRは、判断の中核（権限の段階的な確立、Permissionの分離、証拠と権限の区別）が今も有効である。そのため「置き換え済み」にはせず、どの範囲で読むかを一か所で定める。採用済みADRは本文を書き換えない規則（[README](README.md)）に従い、本ADRで適用範囲を宣言する。

## 決定

1. **ADR-0005の対象を限定する。** Supervisor Ownership・fencing tokenの規則は、参照Process Host等、TEIが監督するDurable Invocationを選んだ実行環境に限って適用する。制御面で確立した権限と、実行側が実際に受理した世代を分ける原則は、すべての対象環境で維持する。所有記録の永続化（#2556）は、コアへ移さず、Plugin／対象環境の実装として扱う。
2. **「Host」は対象環境で副作用を行う主体と読む。** ADR-0019・ADR-0031・ADR-0006・ADR-0007の「Host」「TEI Application Runtime」は、実装事実の記述か、対象環境で副作用を行う主体（実行環境・Plugin・既存アプリケーション）として読む。共通Hostの存在を前提にしない。
3. **認証を前提にした規則は、認証Pluginを束縛した構成に限る。** 評価順序の認証段（ADR-0025、統合後は[ADR-0046](0046-business-lifecycle-as-standard-data-role.md)）と、Decision witnessのidentity候補（ADR-0042）がこれに当たる。ローカルのStudioは認証無しを原則とするため、「Studioの認証済み利用者」をDecision actorの根拠にしない。
4. **物理DB系ADRはRDB系Providerの実現規則として存続する。** ADR-0008〜0015・0026・0027は、#2619の「SQLは特殊経路、通常はData Operations＋Provider binding」のうちBinding / Realization層の規則として読む。Data Operationsの共通契約（#2622）へは持ち上げない。ADR-0021のkeylessな見出し・明細の取得は、Relationを一般的にたどれない場合の明示的な特殊経路として読む。
5. **ADR-0013のprovider enumは移行対象とする。** 閉じた列挙でProviderを選ぶ方式は、ADR-0032・#1223の契約中心のProvider解決へ実装を移す時点で置き換える。
6. **判断の中核は維持する。** 本ADRは、上記ADRの判断を取り消したり置き換え済みにしたりしない。

## 提案中ADRへの反映

本ADRと同時に、提案中のADR-0035・ADR-0039・ADR-0042は本文を次のとおり改訂する。

- ADR-0035：「Production Hostの現在のAuthority」を「対象環境で副作用を行う主体が強制する現在のAuthority」に改める。
- ADR-0039：対象をSQLからData Operation入力一般へ広げ、「Provider呼出し前に拒否する」とする。SQLのNULL bindはSQL realizationの表現として残す。
- ADR-0042：identity providerの候補から「Studio authenticated user context」を外し、共有環境で認証Pluginが提供する場合に限る。

## 採用しない案

- **該当ADRを個別に「置き換え済み」にする。** 判断の中核が有効なため、正本が失われる。
- **ADR-0019をADR-0044で置き換え済みにする。** ADR-0044自身がADR-0019に依存しており（Semantic Capability扱い、secret非保存）、取り消すものではないと明記している。
