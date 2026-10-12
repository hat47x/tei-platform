# SUIの操作をTEI Actionへ接続する参照実装

この実装は、非業務アプリケーションの独自UIをTEI ApplicationのActionへ接続する際の境界を検証するものです。**本番用のCanvas接続、SUIの移行、永続化、実セッション認可は実装していません。**

## 所有する責務

| 関心事 | 所有者・現状 |
| --- | --- |
| ポインター操作、選択、ドラッグ中の表示、取消し | SUIの既存React Canvas。TEIには未接続 |
| ApplicationとActionの対応 | 参照用`Host` |
| 宣言済みCommandの解決 | `CanonicalCommandGate`。`driver.Resolver.ResolvePattern`へ接続可能 |
| 実行時の認可 | `CapabilityAuthorizer`。既存`security.authorize`を使用するが、実セッションは未接続 |
| Documentの識別、Revision照合、更新 | アプリケーション所有の`Store`。今回の実装はメモリ内の模擬保存 |
| SUIのDocumentV1、島への所属、履歴、SafeMode | SUIの既存実装。TEIには移植しない |

`Host.Commit`はApplication ID、**Resource ID**、Action ID、期待Revision、アプリケーション固有のPayloadを受け取ります。更新前にActionの対応と認可を確認し、保存側へ委譲します。**Hostだけでは同時更新を防げません。** Revisionの原子的な照合と更新はStoreの責務です。

## SUIの実コードとの対応

SUI `03_Implement/frontend/src/domain/types.ts` の`DocumentV1`では、Cardの座標は`card.x`、`card.y`です。`card.position`という構造ではありません。参照用`JSONDocumentStore`はこの座標形式に合わせ、`version: 1`と文書本体の`id`を確認します。Sourceは`card.meta.source`、保留は`card.holdState`、レビューは`reviewAttribution`などを保持します。未知のEdge種別も移動によって変更しません。

ただし、参照用StoreはSUIのTypeScript検証器を呼び出さず、島への自動所属やUndo/Redoも再現しません。構造が似ていることと、SUIのドメイン処理に接続したことを混同しないでください。

**旧SUI実装で確認した差分**（2026-10-10に確認した旧経路。PR #3222の新経路とは区別）：

- `src/canvas/CardView.tsx`の`handlePointerMove`は移動中の各イベントで`onMove`を呼びます。Drop時だけ呼び出すAPIではありません。
- `src/App.tsx`の`handleCardMove`は移動中に`setHistory`と`setIsDirty`を実行します。
- 同じ`App.tsx`では`pointerup`と`pointercancel`の両方で`commitCardDragSnapshot`を呼びます。現状の取消しを「永続状態に影響しない」と扱うことはできません。
- Drop時の処理はCardの移動だけでなく、島への所属変更を同じUndo操作にまとめます。単純な`x/y`更新だけでは動作互換になりません。

したがって、TEI接続時は既存のCanvasを置き換えるのではなく、**局所プレビュー／取消し／確定**を分けるSUI側の接続口を用意し、確定時にだけ`Host.Commit`へ最終座標とRevisionを渡す必要があります。認可拒否やRevision競合では、プレビューを破棄し、Documentを再取得します。SUIの既存島所属・Undo/Redo規則はSUI側のAction実装に残します。

## ブラウザー側の操作アダプター（参照実装）

`native-action-bridge.mjs` は、FrameworkやNode.js専用APIに依存しないES Moduleです。既存のReact Canvasを置き換えず、画面側が`begin`、`preview`、`cancel`、`drop`を呼び分ける接続口だけを提供します。

- `begin`: Application、Resource、Actionと期待Revisionを固定します。共通の`protocolVersion: "1"`を生成します。画面入力のApplication IDや権限を信用するという意味ではありません。Host側で再検証が必要です。
- `preview`: 最終候補のPayloadのコピーをローカルに保持します。永続化・TEI Actionは呼びません。
- `cancel`: プレビューを破棄します。書き込みは行いません。
- `drop`: 最後のPayloadで1回だけ`dispatchCommit`を呼び、処理中の二重操作を拒否します。拒否や競合を勝手に再実行せず、例外を呼出元へ戻します。

SUI側では [PR #3222](https://github.com/hat47x/sui-sensemaking/pull/3222) が2026-10-10にmainへ統合され、`CardView`の局所プレビュー、`pointerup`のみの確定コールバック、`CanvasShell`の`onCardMoveCommit`、そして`App.tsx`の確定処理への接続を追加しました。`domain/card_drag_commit.ts`は、従来の位置変更とドロップ先の島への所属変更を一つのDocument更新へまとめます。Appは`applyDocumentChange`を一回呼ぶため、Undo履歴も一単位になります。Cardのポインター操作開始時にはSUI Documentの参照とETagを保持し、確定時にいずれかが変化していれば操作を拒否します。これはSUI内部のローカル整合性ガードであり、TEIサーバーでのRevision照合や認可の代用にはなりません。既存の`onMove`経路は別の利用者との互換用に保持しています。

ただし、**SUI内の局所的な確定と、TEIランタイムのAction実行は別段階です。** このPRでは、SUIからTEIの`native-action-bridge.mjs`、`Host.Commit`、実セッション認可を呼び出していません。DocumentのRevisionを操作開始時に捕捉して送信する経路も未実装です。現行のSUI保存は引き続きSUI自身が所有しています。

現段階の変更量はSUIの既存3ファイル（App、CanvasShell、CardView）への編集と、新規ドメイン実装1ファイル・テスト2ファイルです。SUI側のVitest・typecheck・build・実ブラウザー確認は未実行のため、UIの動作互換が実証された状態ではありません。ブラウザーではTEI側のES Moduleを取り込めますが、`structuredClone`を利用できる実行環境が必要です。

## SUI利用側アダプターへの接続（Draft）

SUIの[PR #3226](https://github.com/hat47x/sui-sensemaking/pull/3226)では、`src/api/tei_card_move_action.ts`を追加しています。SUIの`DocumentV1`とETagを操作開始時点の条件とし、Card移動の確定座標だけを`sui.move`のPayloadとして渡します。成功後は、**SUI自身の既存Document取得経路**からDocumentとETagを再読込し、TEI Actionが返したRevision、Card座標、島への所属が一致した場合にだけSUIへ状態反映を委ねます。利用側が`dispatch`、`readDocument`、`isCurrent`、`applyConfirmed`の4つのポートを与える構造で、TEI CoreにSUI専用型を取り込みません。

SUI内の単体参照テストは14件PASS（Node.js 22.16.0、Git Blob SHA一致のファイル）。この確認はSUI実Canvasや本番のTEI Hostの連携試験ではありません。特に、サーバー側の`sui.move` Command実装・永続的なCAS、HTTP配備・セッション認可、およびReact App内で履歴・dirty・ETagを整合して取り込む実経路は未接続です。SUIの保存用ETagとTEIの応答Revisionを同一リソースの値として照合できる本番Adapterが確定するまで、UI側の接続を有効化しません。

### SUI受信実装の構文破損復旧・レビュー帰属（2026-10-11）

SUI側で追加修正を照合した際、`routes/docs.py`にAction受信関数の重複と`expectedRevision`フィールドの欠落があることを発見しました。これはコードをPRへ追加しただけでは気付きにくい重大な退行です。重複がないと確認できたコミット`d952d97`を基点に、Action受信経路のOrigin、Revision、保存、監査を再構築し、`_SuiCardMoveActionIntent`・`post_sui_card_move_action`・`_commit_sui_card_move_action`とRevisionフィールドが各一つであることを確認しました。ローカル実行用`03_Implement/scripts/verify_native_action_boundary.sh`には、pytestに先立って`compileall`／`ast.parse`で重要定義の欠損と重複を検出する静的ガードを追加しています。**スクリプトと正式統合テストは未実行です。**

SUIの既存`reviewAttribution`は、過去の人間レビューの帰属を示します。新規レビューを登録する通常PUTでは本人一致を厳密に確認する一方、Card移動は既存レビュー帰属を変えない操作であり、別の認可済み利用者にも許可されるべきです。ActionではReviewerの再申告検証を行わず、`updated.reviewAttribution == original.reviewAttribution`で改変不可を保証します。異なるActorによる移動で`reviewAttribution`・`critiqueInputs`・`reproposalDiffs`が保存されるSQLite回帰テストを追加しています。これも正式実行はまだです。

TEI HostからSUIへの実認証委譲は`CAPABILITY_GAPS.md`の「SUI native Action receiver: authority handoff gates」を要件とします。**実セッションとApplication実装の本人・Tenant・対象Resourceが整合しない構成では有効化しません。** Hostサービスアカウントをそのまま元の人間主体の代用にしないこと、SUI所有の保存契約をTEI Coreへ移植しないことを明示しています。

### SUI同時更新時のロック競合と通信断後の観察（2026-10-11）

SUI #3226の条件付きSQL claimは、実行中に同じDocumentへの別Writerが現れた場合、SQLite`SQLITE_BUSY`/`SQLITE_LOCKED`、PostgreSQL`40001`/`40P01`/`55P03`を既知の書込競合として扱い、SUI ActionおよびIf-Match付きPUTへ409を返します。これ以外のDB障害を勝手に競合扱いしません。Actionが行claim後に入力・ドメイン検証で拒否されたときは早期`db.rollback()`でロックを解放します。2つの活きたSQLite Sessionでのロック競合、ドメイン拒否後の新しい正常Action、PostgreSQLドライバーのSQLSTATE分類のテストを追加しました。独立したSQLite実験は実行しましたが**PR headの完全なSQLite/PostgreSQL統合テストはまだです**。

通信断でPOSTが確定したか不明なとき、SUIは`commit_outcome_unknown`として**自動再送しません**。追加された`observeSuiCardMoveAfterUnknownCommit`はSUIの権限付きDocument取得口から読み直して、`unchanged`・`converged`・`diverged`・`unresolved`を観察するだけです。`converged`は「結果と同じDocumentになっている」であって「このActionが成功した／この人が編集した」という確証ではありません。したがって監査帰属やUndo/dirtyを更新せず、React所有者で現在のセッション・ETag・Document同一性を再評価します。GitHub BlobからTypeScript型を除去した限定JS実行では35ケース／158アサーションが成功しました。正式Node、tsc、Vitestの合格とは区別します。

これらは**SUI Applicationが所有する具体化**です。TEI汎用HostへSUI固有の履歴・島・PostgreSQL SQLSTATE列挙を持ち込まず、実Subject委譲を確立するまでActionの本番経路を有効化しません。

### CAS Revisionの強度とSUI固有の成功監査（2026-10-11追補）

SUI [Draft PR #3226](https://github.com/hat47x/sui-sensemaking/pull/3226)のAction readbackを、SUI既存GETの互換的な弱ETag解釈から切り離しました。**SUI固有のApplicationプロファイル**として、Action要求・応答と再読込のRevisionは厳密な小文字64桁SHA-256であり、再読込のHTTP ETagには強い引用符付きの`"<hash>"`だけを認めます。弱い`W/"..."`や欠落・不正ETagを拒否し、取得はsame-origin、no-store、redirect=errorとしました。JavaScript／Python双方で、末尾改行と正規表現`# SUIの操作をTEI Actionへ接続する参照実装

この実装は、非業務アプリケーションの独自UIをTEI ApplicationのActionへ接続する際の境界を検証するものです。**本番用のCanvas接続、SUIの移行、永続化、実セッション認可は実装していません。**

## 所有する責務

| 関心事 | 所有者・現状 |
| --- | --- |
| ポインター操作、選択、ドラッグ中の表示、取消し | SUIの既存React Canvas。TEIには未接続 |
| ApplicationとActionの対応 | 参照用`Host` |
| 宣言済みCommandの解決 | `CanonicalCommandGate`。`driver.Resolver.ResolvePattern`へ接続可能 |
| 実行時の認可 | `CapabilityAuthorizer`。既存`security.authorize`を使用するが、実セッションは未接続 |
| Documentの識別、Revision照合、更新 | アプリケーション所有の`Store`。今回の実装はメモリ内の模擬保存 |
| SUIのDocumentV1、島への所属、履歴、SafeMode | SUIの既存実装。TEIには移植しない |

`Host.Commit`はApplication ID、**Resource ID**、Action ID、期待Revision、アプリケーション固有のPayloadを受け取ります。更新前にActionの対応と認可を確認し、保存側へ委譲します。**Hostだけでは同時更新を防げません。** Revisionの原子的な照合と更新はStoreの責務です。

## SUIの実コードとの対応

SUI `03_Implement/frontend/src/domain/types.ts` の`DocumentV1`では、Cardの座標は`card.x`、`card.y`です。`card.position`という構造ではありません。参照用`JSONDocumentStore`はこの座標形式に合わせ、`version: 1`と文書本体の`id`を確認します。Sourceは`card.meta.source`、保留は`card.holdState`、レビューは`reviewAttribution`などを保持します。未知のEdge種別も移動によって変更しません。

ただし、参照用StoreはSUIのTypeScript検証器を呼び出さず、島への自動所属やUndo/Redoも再現しません。構造が似ていることと、SUIのドメイン処理に接続したことを混同しないでください。

**旧SUI実装で確認した差分**（2026-10-10に確認した旧経路。PR #3222の新経路とは区別）：

- `src/canvas/CardView.tsx`の`handlePointerMove`は移動中の各イベントで`onMove`を呼びます。Drop時だけ呼び出すAPIではありません。
- `src/App.tsx`の`handleCardMove`は移動中に`setHistory`と`setIsDirty`を実行します。
- 同じ`App.tsx`では`pointerup`と`pointercancel`の両方で`commitCardDragSnapshot`を呼びます。現状の取消しを「永続状態に影響しない」と扱うことはできません。
- Drop時の処理はCardの移動だけでなく、島への所属変更を同じUndo操作にまとめます。単純な`x/y`更新だけでは動作互換になりません。

したがって、TEI接続時は既存のCanvasを置き換えるのではなく、**局所プレビュー／取消し／確定**を分けるSUI側の接続口を用意し、確定時にだけ`Host.Commit`へ最終座標とRevisionを渡す必要があります。認可拒否やRevision競合では、プレビューを破棄し、Documentを再取得します。SUIの既存島所属・Undo/Redo規則はSUI側のAction実装に残します。

## ブラウザー側の操作アダプター（参照実装）

`native-action-bridge.mjs` は、FrameworkやNode.js専用APIに依存しないES Moduleです。既存のReact Canvasを置き換えず、画面側が`begin`、`preview`、`cancel`、`drop`を呼び分ける接続口だけを提供します。

- `begin`: Application、Resource、Actionと期待Revisionを固定します。共通の`protocolVersion: "1"`を生成します。画面入力のApplication IDや権限を信用するという意味ではありません。Host側で再検証が必要です。
- `preview`: 最終候補のPayloadのコピーをローカルに保持します。永続化・TEI Actionは呼びません。
- `cancel`: プレビューを破棄します。書き込みは行いません。
- `drop`: 最後のPayloadで1回だけ`dispatchCommit`を呼び、処理中の二重操作を拒否します。拒否や競合を勝手に再実行せず、例外を呼出元へ戻します。

SUI側では [PR #3222](https://github.com/hat47x/sui-sensemaking/pull/3222) が2026-10-10にmainへ統合され、`CardView`の局所プレビュー、`pointerup`のみの確定コールバック、`CanvasShell`の`onCardMoveCommit`、そして`App.tsx`の確定処理への接続を追加しました。`domain/card_drag_commit.ts`は、従来の位置変更とドロップ先の島への所属変更を一つのDocument更新へまとめます。Appは`applyDocumentChange`を一回呼ぶため、Undo履歴も一単位になります。Cardのポインター操作開始時にはSUI Documentの参照とETagを保持し、確定時にいずれかが変化していれば操作を拒否します。これはSUI内部のローカル整合性ガードであり、TEIサーバーでのRevision照合や認可の代用にはなりません。既存の`onMove`経路は別の利用者との互換用に保持しています。

ただし、**SUI内の局所的な確定と、TEIランタイムのAction実行は別段階です。** このPRでは、SUIからTEIの`native-action-bridge.mjs`、`Host.Commit`、実セッション認可を呼び出していません。DocumentのRevisionを操作開始時に捕捉して送信する経路も未実装です。現行のSUI保存は引き続きSUI自身が所有しています。

現段階の変更量はSUIの既存3ファイル（App、CanvasShell、CardView）への編集と、新規ドメイン実装1ファイル・テスト2ファイルです。SUI側のVitest・typecheck・build・実ブラウザー確認は未実行のため、UIの動作互換が実証された状態ではありません。ブラウザーではTEI側のES Moduleを取り込めますが、`structuredClone`を利用できる実行環境が必要です。

## SUI利用側アダプターへの接続（Draft）

SUIの[PR #3226](https://github.com/hat47x/sui-sensemaking/pull/3226)では、`src/api/tei_card_move_action.ts`を追加しています。SUIの`DocumentV1`とETagを操作開始時点の条件とし、Card移動の確定座標だけを`sui.move`のPayloadとして渡します。成功後は、**SUI自身の既存Document取得経路**からDocumentとETagを再読込し、TEI Actionが返したRevision、Card座標、島への所属が一致した場合にだけSUIへ状態反映を委ねます。利用側が`dispatch`、`readDocument`、`isCurrent`、`applyConfirmed`の4つのポートを与える構造で、TEI CoreにSUI専用型を取り込みません。

SUI内の単体参照テストは14件PASS（Node.js 22.16.0、Git Blob SHA一致のファイル）。この確認はSUI実Canvasや本番のTEI Hostの連携試験ではありません。特に、サーバー側の`sui.move` Command実装・永続的なCAS、HTTP配備・セッション認可、およびReact App内で履歴・dirty・ETagを整合して取り込む実経路は未接続です。SUIの保存用ETagとTEIの応答Revisionを同一リソースの値として照合できる本番Adapterが確定するまで、UI側の接続を有効化しません。

の挙動に依存せず、63桁+改行も無効にします。この制約を**汎用TEI Hostの`expectedRevision`型や第二Applicationに押し付けません**。Inventory等の異なる保存リビジョン形式はそのApplicationの契約が定義します。

SUI固有のAction保存がDB commitに成功した後、SUI既存`AuditDispatcher`へ`apply`イベントを送信する処理を追加しました。検証済ActorとTenant/Document、`sui.move`、`committed`を記録し、Card本文・Source・生の権限主張はコピーしません。競合・拒否・失敗時には成功イベントを送らず、監査Sink障害でも**既に完了したコミットをHTTP 500の失敗と誤報しない**ようにします。ただし既存監査配送はfail-openであり、**永続監査outbox・復旧可能な配送保証は未実装**です。本番監査要件を満たすことを意味しません。READMEの確認はコード・テスト追加の事実であり、正式Python/SQLite/Vitest/Go/React検証は未完了です。

### SUI Action境界のHTTP保護とUndo状態計画（2026-10-11）

[SUI Draft PR #3226](https://github.com/hat47x/sui-sensemaking/pull/3226)では、Go参照HTTPプロファイルとSUI専用の受信口の差分を詰めました。明示されたHTTP OriginのScheme/Host同一性はBFF Cookieなしでも確認し、異なるOriginや`X-TEI-Action`欠落は`403 request_origin_denied`、非JSONは`415 unsupported_content_type`とします。Cookie認証では引き続きSUIのBFF CSRFミドルウェアを利用します。ETagに終端改行を付けた非正規入力は64文字の長さ制限によって拒否し、保存済みDocumentの形式不整合（`500 execution_failed`）と利用者のMove入力違反（`400 invalid_payload`）は分離しました。これらはSUI専用HTTPアダプターの処理であり、TEIのGo HostへSUI固有型を導入しません。

ブラウザー側では、成功応答を受け取った後の壊れたDocument再読込についても、確認済みの`committedRevision`を保持した`invalid_readback`を返します。サーバー側は既に処理した可能性があるため、旧Revisionでの自動再送は禁止します。

SUIへ`frontend/src/domain/confirmed_card_move_state.ts`と対応テストを追加しました。これはReact Canvasを置き換えるものではなく、SUI自身の履歴`past/present/future`、保存済みETagとdirtyを一体として計画する純粋関数です。サーバー確定後はdirty=falseとして直前のDocumentをUndo履歴に残し、後からUndoした操作は**新しいローカル編集**として再保存します。未保存編集、ReadOnly、保存処理中、テナントセッション変化、進行中ドラッグ、操作開始時のDocument/ETag不一致は拒否します。**複数React stateの更新原子性・実Canvasへの接続は未完成です。**

アダプターの補助的V8確認は25/25ケース、履歴計画関数の同確認は5/5ケース。ただしGitHub取得後に型注釈を除去し簡易Assertで実行した結果であり、Node/Vitest/TypeScript/Python/Goの正式テストPASSではありません。実証はSUIの`03_Implement/scripts/verify_native_action_boundary.sh`でexact-headをチェックアウトして行います。CIは停止中であり、両PRのDraftは維持します。

### Action送信前のDocument差分制約と検証入口（2026-10-10）

[SUI Draft PR #3226](https://github.com/hat47x/sui-sensemaking/pull/3226)に`isPureCardMove`を追加しました。元Documentと移動候補を比較し、**対象Cardのx/yと島への所属変更のみ**を許して、それ以外の情報変更が混入したリクエストは**Action dispatch前**に拒否します。これにより成功済みのSUI保存と画面の再取得結果が食い違う事例を抑制し、SUIのSource／Hold／Edge／レビュー／Affiliationの非改変を守ります。TEI HostがSUIのCardやIslandの意味論を解釈する必要はありません。

SUIの`Island.summaryText`がある場合にPydanticが`summaryReviewed=false`を補完する正規化にも対応しました。`true`など異なるレビュー状態は比較エラーとして保持します。

SUIへ`03_Implement/scripts/verify_native_action_boundary.sh`を追加し、**正確なHEADとクリーンな作業ツリー**を確認してPython SQLite・認可テスト、TypeScript型チェック、Vitest、Nodeアダプターテストを実行する手動検証経路を用意しました。追加実装のNodeテスト23件については、GitHub Blobから取得したソースをV8内で型注釈除去・簡易Assertionで評価する限定的な**23/23 PASS**を得ましたが、これは`node --test`も`tsc`も使わない補助検査であり、正式なNode／Vitest／Go／Pythonテスト完了を意味しません。SUI FastAPI実通信やTEI Go Hostの安全な連結は未完了です。

### SUI Actionの同期確定とPydantic正規化（2026-10-10）

SUI側の`applyConfirmed`ポートは`Promise<boolean>`を許さず、**同期的に状態反映が完了した場合の`boolean true`だけ**を承認するよう改訂しました。サーバーが確定した後のReact側競合を避けるには、SUIアプリがDocument参照、ETag、未保存編集の有無を同期的に再確認し、履歴・dirty・Undo/Redoを一貫して更新する必要があります。単純な`setState`要求だけを根拠に`true`を返す実装は許容しません。

また、SUI Pydanticが欠落した`Island.collapsed`を`false`で補完し、`shape`と`geometry`の片方からもう一方を導出する既知の保存時正規化に対応しました。Document全体照合によるSource、レビュー、Edge、Affiliation等の喪失検出は維持します。

SUI `src/api/tei_card_move_action.integration.test.ts`に、Actionポート・同一オリジンPOST・no-storeのETag付きGET・アプリ所有状態の同期確定を結ぶテストを追加しました（送信と受信はモックHTTP、実SUI FastAPI、TEI Go Host、React実画面は未接続）。**変更後のNode／Vitest／Pythonテストは未実行です。**

### 利用側安全性の追補（2026-10-10）

[SUI Draft PR #3226](https://github.com/hat47x/sui-sensemaking/pull/3226)では、Action成功後のDocument全体照合を追加し、座標・島所属だけが正しくてもSource／Hold／レビュー／未知Edge／Affiliationなどに損失があればローカル反映を拒否するようにしました。サーバー所有の`updatedAt`と任意の`null`／未指定の等価性は許容します。単一テナントの追加認可アダプターがない場合でも、明示的な読取り専用モードのAction更新は拒否します。

また**SUI固有の`POST /docs/{doc_id}/action-commit`はデフォルト無効**としました。`request.app.state.sui_native_action_v1_enabled is True`が明示されない限り404で拒否します。テストだけが局所的に有効化し、未検証の通常配備で新たな書込み経路を開きません。これらの追加修正後のPython／Vitest／Node／全体healthは未実行です。実TEI Hostの認可・呼出委譲・SUI React状態反映が完成するまで、本番有効化を認めません。

### SUIサーバー側のSUI所有コマンドと既存保存経路

[SUI PR #3226](https://github.com/hat47x/sui-sensemaking/pull/3226)へ、`card_move_command.py`（DocumentV1内のCard移動、島所属、Affiliation衝突時の拒否）と、既存`PUT /docs/{id}`／`GET /docs/{id}`／ETagを利用するSQLite結合テストを追加しました。フロントエンドの`getAuthoritativeDocument`は`cache:"no-store"`を指定し、セッション前提条件を維持してDocumentとETagを読み直します。

この段階は、SUIが自身のDomain MutationとDocument保存を所有する構造の確認です。**SUI側には`POST /docs/{doc_id}/action-commit`というApplication-owned v1受信経路を追加し、既存の`_authorize_request`、DocumentContentStoreの保存トランザクションとRevision Head CASへ接続しました。** ただし、TEI Go HostからそのSUI経路を実際に呼ぶ部分、Host側の実セッション権限照合・委譲と監査、React側のatomic Undo/dirty反映は**未完成**です。SUI側のAction endpointはTEI Go Hostの実装完了を意味しません。Python／SQLiteのAction受付結合テスト、Action JSON厳格性とBodyサイズ制限の試験、TypeScript側の同一オリジン通信テストを追加しましたが、**今回の追加分は実行未確認**です。SUIのテナント境界とルート一覧を静的に検査する既存のテストも実行が必要です。全体健全性検査も未実施です。SUIの現在のPUTに存在するテナント分離・SafeMode・レビュー帰属・アーカイブ判定の手続きを、独立したTEI Action経路から迂回してはいけません。

SUI側のTypeScriptとPythonで島境界判定を実装するため、同じ入力と期待状態を使う`card_move_parity_v1.json`を[SUI PR #3226](https://github.com/hat47x/sui-sensemaking/pull/3226)へ追加しました。矩形加入、通常移動、グリッド吸着、多角形境界の4ケースを双方のテストから参照します。**まだ実行結果は確認できていません。** 追加の敵対的ケース・性能比較・本番E2Eで差異を検出できるまで、UIからのAction確定を本番経路として有効にしません。

## SUI実接続の条件（2026-10-10時点）

SUI main の`App.tsx`は、`handleCardMoveBegin`で`DocumentV1`の参照と`docEtag`を捕捉し、`handleCardMoveCommit`から`commitCardDrag`を呼んで`applyDocumentChange`へ**ローカル変更**を渡しています。現行の`CanvasShell.onCardMoveCommit`は同期的な`void`コールバックです。このまま同じ確定処理にHTTP dispatchを足すと、サーバー側の拒否より先にSUIのUndo履歴・dirty状態を書き換える二重の更新経路ができてしまいます。

実接続では、次の境界を一つずつ確定します。

1. **保存先の一本化**：通常のSUIローカル編集と、試験用のTEIサーバー確定モードを明示的に分離する。後者では承認済みのサーバー応答を受ける前に`applyDocumentChange`を呼ばない。失敗時にローカル保存へ黙ってフォールバックしない。
2. **Revisionの同一性**：TEI要求の`expectedRevision`は実際にCASする保存先のRevisionを使用する。SUIの`docEtag`を採用する場合は、サーバー側のSUI保存Revisionと同一条件であることを実証する。値がない場合や、ドラッグ開始後に未保存編集が入った場合は送信しない。
3. **SUIの意味論の保持**：最終位置だけを書き換える模擬`JSONDocumentStore`は、島への所属変更とUndo/Redoを再現できない。本番アダプターではSUI所有の`commitCardDrag`相当の処理をSUIの権威ある保存境界で原子的に実行し、TEI HostにはCardやIslandの業務規則を実装しない。
4. **確定後の再取得**：成功時は保存側のRevisionとDocumentを再取得し、失敗・競合時はプレビューを破棄して差分を表示する。通信断の場合は成功／失敗を推測せず、保存先を照会する。
5. **保護の継承**：SUIのReadOnly、SafeMode、未レビュー情報、Export制限、テナント境界、監査を既存より弱めない。認証セッション・CSRF・認可・冪等性はTEI実配備側で検証する。

まず`SUI CardView → SUI commit callback → Action dispatcher → TEI Host → SUI所有の保存処理 → 再読込`を一本の接続テストで検証します。旧Revision、認可拒否、通信断、島所属、Undo、来歴保持を合否条件とします。**現時点では未接続・未検証**であり、この文書は実装完了の証拠ではありません。

## 確定結果の再照合（ブラウザー参照実装）

`confirmed-action-commit.mjs`は、UIがサーバーの書込み応答だけを根拠にDocumentを更新することを防ぐ、小さな接続部品です。`createNativeActionBridge`の送信先として組み込みます。

- `dispatchCommit(intent)`：`createJSONActionDispatcher`などで一度だけ確定要求を送ります。権限拒否・Revision競合でローカル状態は更新しません。
- `readResource(intent)`：成功後に、**同じ権威ある保存先**からキャッシュを使わずに取得します。アプリケーション側の読取アダプターは`{applicationID, resourceID, revision, value}`形式へ正規化します。`value`はアプリケーション所有の状態です。
- `isCurrent(intent)`：送信前と読取後に、操作開始時のローカル状態が現在も有効か検査します。**同期的に真偽値の`true`を返した場合だけ通過**させ、Promise、例外、その他の真と評価される値は拒否します。SUIであればDocument参照・ETag・未保存編集状態などを確認します。判定できない場合は成功扱いにしません。
- `applySnapshot(resource, intent)`：Resource IDとRevisionが確定応答と一致し、ローカル状態も変化していない場合にだけ、アプリケーション自身が確定済みSnapshotを反映します。この処理自身も**最新状態を原子的に再確認**する必要があります。反映が原子的に完了した場合に限り、同期・非同期を問わず**真偽値の`true`を明示的に返す**必要があります。例外、`false`、`undefined`などは反映未確認として扱います。

同じResourceに対する処理中の二重確定は拒否します。別Resourceは同時に処理できます。エラー`ActionConfirmationError`の`committedRevision`が非nullの場合、**サーバーは成功を応答済みですが、UIへの反映は未完了**です。読取失敗、Revisionの不一致、UIの状態変化、反映処理の失敗をサーバー側のロールバックとみなさず、旧Revisionで自動再送しません。通信断など、成功応答そのものを受け取れなかった場合も、保存先に変更済みの可能性があるため再照会が必要です。

SUIの`applyDocumentChange`はローカルUndo履歴とdirty状態を更新するため、そのままサーバー確定Snapshotの反映に使うことはできません。実接続では**サーバー確定Snapshotを履歴・dirty状態と整合させて取り込む別のSUI所有経路**が必要です。TEI側がSUIのUndoやIsland規則を解釈することはありません。

検証用の`confirmed-action-commit.test.mjs`（15ケース）と`action-browser-roundtrip.test.mjs`（4ケース）は、取消し・正常確定・失敗時の非反映・通信応答喪失・第二Applicationへの再利用を対象にします。後者のHTTPサーバーと保存処理は**テスト内の合成実装**であり、実Go Host・本番SUI保存経路・実Canvasの動作証拠ではありません。

## Web向けのJSON Action接続口（参照実装）

CLIや同一プロセス内の実行では、`Host.Commit`を直接呼べます。Webではオプションとして、`JSONCommitHandler`（Go標準`net/http`）と`json-action-dispatch.mjs`（ブラウザーの`fetch`）を利用できます。外部Pluginプロセスを必須にしません。

`JSONCommitHandler`は利用側がApplicationとActionの組合せに対応する`DecodePayload`を渡さない限り、503で拒否します。共通層の詳細契約は[`PROTOCOL.md`](PROTOCOL.md)にまとめ、要求は[`action-protocol-v1.schema.json`](action-protocol-v1.schema.json)、成功・失敗の応答は[`action-result-v1.schema.json`](action-result-v1.schema.json)に定義しています。認可は`Host.Authorizer`の責務であり、認証済みセッションから主体を取り出す実装が必要です。HTTP層は任意の主体IDを受信しません。JSONの未知フィールド、複数連結されたリクエスト、64KiB超の入力を拒否します。

要求例（既存の`native-action-bridge.mjs`の`dispatchCommit`に、`createJSONActionDispatcher("/actions/commit")`を渡す）。この外側の形式はSUI以外のアプリケーションでも共通です：

```json
{
  "protocolVersion": "1",
  "applicationID": "sui",
  "resourceID": "doc",
  "actionID": "sui.move",
  "expectedRevision": "r1",
  "payload": {"cardId": "a", "x": 5, "y": 6}
}
```

`POST`には`Content-Type: application/json`と`X-TEI-Action: commit`が必要です。許可された同一オリジンの要求だけを受け付け、クロスサイトの利用を禁止します。成功時は`{"revision":"..."}`、拒否時は`{"error":"..."}`を返します。`403 action_denied`と`409 revision_conflict`はブラウザーでも区別して通知できます。この同一オリジン検査とカスタムヘッダーは、**本番向けのCSRF対策・セッション検証・永続化を代替しません**。

HTTP接続口とブラウザー側Dispatcherの動作は模擬Store・模擬fetchを使ったテストの範囲です。HTTP→Host→SUI形式JSONDocumentStoreで正常更新と再送時の競合を確認する統合テストも追加しましたが、このテストはまだ実リポジトリで実行していません。SUIの本番サーバーにこのHandlerを登録した状態ではありません。

## 実行時の認証を伴うHTTPS委譲（追加の参照実装）

`actionboundary.HTTPSApplicationStore`を追加し、TEIのHostが認可を通したActionを、固定されたHTTPSの受信口へ送れるようにしました。宛先の選択は配備設定が所有し、Action本文で別サイトへ書込み先を変更できません。信頼できる実行文脈の本人と、対象アプリケーション向けに委譲された資格情報の本人が一致しない場合は送信しません。接続エラーや不正応答では成功・失敗を推測せず、再送もしません。

詳細な責務分担は[共通Action契約](PROTOCOL.md#アプリケーション所有の保存先へのhttps委譲)を参照してください。今回の検証はTLSテストサーバーへのHTTP往復と模擬資格情報を利用するものであり、**実際の認証済みSUIセッションから資格情報を取得する機構、TEI Hostへの本番登録、SUI BackendとのEnd-to-End接続は未実装**です。既定無効のSUI Action受信口を有効にしません。

## 実行

```sh
go test -race -count=1 ./tools/reference-sui-action-boundary/actionboundary
node --test tools/reference-sui-action-boundary/native-action-bridge.test.mjs
node --test tools/reference-sui-action-boundary/json-action-dispatch.test.mjs
node --test tools/reference-sui-action-boundary/confirmed-action-commit.test.mjs
node --test tools/reference-sui-action-boundary/action-browser-roundtrip.test.mjs
```

ローカルに再構成したファイルとGitHubのGit Blob SHAを照合したうえで、Node.js 22のテスト16件（操作アダプター6件、JSON Dispatcher 6件、共通契約4件）がすべてPASSしました。GoのHTTP Handlerとそのテスト2ファイルについても、Git Blob SHAが一致するコピーを用意し、模擬Host／模擬Document Storeを組み合わせて`go test -race`と`go vet`をPASSしました。

ただし、これらは**正確なPR headをチェックアウトしたTEI全体のテストではありません**。Canonical Command解決、実セッション認可、SUI本番保存機構の検証結果も含まれません。統合時は正確なコミットを固定し、`tools/repository-health-check.sh`全項目PASSと実行前後のclean状態を確認してください。

## 検証範囲と未実装

参照テストは正常更新、認可拒否、Revision競合、同時更新、Resource IDの不一致、未知Card、非有限座標、不正Document構造、無関係な情報の保持を対象にします。`standard_protocol_test.go`ではSUI形式のCard移動と、独立したInventory形式の在庫調整を同じHandler・Hostに渡すテストを追加しました。双方とも模擬Storeであり、汎用プラグインの本番動作を証明しません。

未実装：SUIの確定コールバックからTEIランタイムへのAction dispatch、操作開始時のRevision捕捉と権限照合、SUIのDocumentV1検証器とTEI保存機構の実連携、実セッションからのSubject解決、実ランタイム権限、ブラウザー配備、SafeModeとExport保護の統合確認、性能測定。

作業窓口はGitHub Issue #993、設計の正本はISSUE-000052-sui-reference-adoption-and-plugin-surface-evaluation.mdです。GitHub Issue #52は別件です。
