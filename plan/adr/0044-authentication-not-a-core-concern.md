# ADR-0044: 認証をTEIコアの規定・実現対象としない

> **位置づけ（2026-11-03）**：[ADR-0019](0019-authentication-capability-boundary.md)の「認証・認可をSemantic Capabilityとして扱う」分離と、[ADR-0043](0043-target-native-artifacts-without-tei-host.md)の「対象環境へ適応する」方針を、認証について具体化するものであり、これらを取り消すものではない。

- **状態**：採用（設計方針、2026-11-03）。実装済み・検証済みという意味ではない。
- **判断主体**：TEI Owner（2026-11-03）。
- **関連**：[ADR-0019](0019-authentication-capability-boundary.md)、[ADR-0043](0043-target-native-artifacts-without-tei-host.md)、`plan/design/LEGACY_AUTH_V1_DESIGN.md`、`plan/design/DESKTOP_STUDIO_HOST_INTEGRATION_DESIGN.md`、[製品要求](../../requirements/TEI_PRODUCT_REQUIREMENTS.md) RQ-019。
- **更新範囲**：認証をTEIコアが規定・実現する可能性を閉じ、Studioのローカル動作とランタイム認証の位置づけを明示する。ADR-0019のSemantic Capability扱い、ADR-0043の対象環境適応、RQ-019の権限・Resource境界を取り消すものではない。

## 背景

TEI Studioは、Sourceの参照・検証・編集提案・差分確認・受理をローカルのWorkspaceに対して行う開発者向けのAuthoring Clientである。ローカル実行に認証を要求すると、単一利用者の開発環境にIdPや認証基盤の用意を強制し、意味の編集と検証というStudioの価値を損なう。

一方、業務Applicationへの組み込みや共有環境への接続では認証・認可が必要になる。その方式はDatabase、LDAP、IdP、独自API、既存関数など組織・対象環境ごとに異なり、TEIコアに固定すると、ADR-0019が排除した物理方式のCanonical持ち込みをやり直すことになる。ADR-0043が決めたとおり、認証・認可はそれぞれの契約、提供主体、境界、発生時点に応じて解決する事項であり、共通プロセスや共通ランタイムへの集中配置を要求しない。

## 決定

1. **TEI Studioはローカル環境では認証無しで動作するのを原則とする。** ローカルWorkspaceに対する参照・検証・編集提案・受理等の操作に、ログインやcredentialを要求しない。
2. **ランタイムの認証はPlugin依存とし、多様な挙動を切り替え可能にする。** 認証・認可はADR-0019のとおりSemantic Capabilityとして表現し、実現はprocess Plugin、既存IdP、既存関数等のRealization / Binding側に置く。同一のCapability identityを保ったまま実現方式を交換できる。
3. **認証はTEIのコアが規定・実現すべき事項ではない。** `core/`、CLI、生成器は認証機構を必須依存・内蔵としない。認証を要求するかどうか、どの方式を使うかは、利用側の環境・契約（Host、Plugin、対象環境のPolicy）が決める。
4. **認証を必要としない運用と、Pluginで認証する運用を、同一の意味モデル・成果物から選択できる。** 新しい認証方式の追加は、コアの変更ではなくPlugin / Bindingの追加として行う。システムの運用に応じて柔軟に拡張できることを要件とする。
5. **認証が無いことを権限の無制約と混同しない。** RQ-019の権限・Resource境界、secretをCanonical definition・Durable artifactへ保存しない制約は、認証無しのローカル実行でもそのまま適用する。

## 既存判断との関係

- **ADR-0019を否定しない。** 「業務上の意味は意味層に、実現は実装層に」という同ADRの分離を、Studio（ローカル・認証無し）とランタイム（Plugin認証）の両側で完結させることが本ADRである。コアが認証を実装しないことは、認証をSemantic Capabilityとして表現する理由の裏面である。
- **ADR-0043の具体化である。** 同ADR決定3の「認証、認可…はそれぞれの契約、提供主体、境界、発生時点に応じて解決する」を、Studioとランタイムの具体的位置づけとして明文化した。
- **複数認証方式の接続規則を実証する作業（issue #1043等）は継続する。** これはコア実装ではなく、接続規則とPlugin契約の実証として読む。
- **LEGACY_AUTH_V1_DESIGNの「まだ固定しないこと」は変更しない。** `std.auth.*`の標準名、OAuth / OIDC / SAML等の網羅的model、標準User schemaは、本ADRによっても固定しない。方式の標準化は、複数の実現方式で同じ契約が成立するEvidenceを得てから別途判断する。

## 採用しない案

- **認証機構をTEIコアへ内蔵する。** 対象環境の既存ライフサイクル・認証認可と重複し、ADR-0043が排除した集中配置をやり直すため採用しない。
- **Studioのローカル動作にログインを必須化する。** 単一利用者の開発環境に認証基盤の用意を強制するため採用しない。
- **SAML等の特定認証方式を製品要件とする。** 方式の固定はADR-0019と同様にEvidenceが足りないため採用しない。

## 検証と未完了の範囲

- Studio（Wails Host PoC、CLI経由のAuthoring）がローカルWorkspaceに対して認証手続きなしで動作することを、既存の起動・Authoring E2Eで確認する。
- 認証Capabilityを持つPluginと持たないPluginの双方で、同一Workspace・同一成果物の受理・生成が成立することをReferenceで示す。
- 認証方式の追加がコアの変更ではなくPlugin / Bindingの追加で行えることを、LEGACY_AUTH_V1_DESIGNの実現方式交換の確認と合わせて示す。
