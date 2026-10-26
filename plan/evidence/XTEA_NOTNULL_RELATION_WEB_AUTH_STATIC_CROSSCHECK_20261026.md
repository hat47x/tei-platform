# X-TEA固定版：TableField NotNull／Relation任意性／DDLとWeb認証境界の静的対照（2026-10-26）

状態：**旧製品のSource Evidence・設計検討資料。TEIのAccepted Meaning、DDL同等性、実配布Webの安全性を認定しない。** 担当Issueは[#1126](https://github.com/hat47x/tei-platform/issues/1126)（ER／論理と物理）、[#1043](https://github.com/hat47x/tei-platform/issues/1043)（Web Session／認証）。上流の既知のlogin到達問題は[#2222](https://github.com/hat47x/tei-platform/issues/2222)を参照する。既存Issueにある調査コメントを再投稿する目的ではない。

## 1. 固定版と読取り範囲

- **XeadModeler M49固定ソース**：commit [68f1535b1c5dd80a327dfced37be401ecd3093c9](https://github.com/xead/XeadModeler/tree/68f1535b1c5dd80a327dfced37be401ecd3093c9)。`src/xeadModeler/Modeler.java` Git blob **c104e379862cf280a5944ed97c35cfe568c5f348**、過去に記録されたSHA-256 **7cbbefe8115b20e2ba13fcfdb21ef9c5ae596debf673b6afaf21ab6b20a3edef**。巨大ファイルの行指定APIが今回空本文を返したため、以下のModeler.java行番号は既存固定版調査コメントの照合結果であり、**今回の直接本文再取得は未成功**。
- 同commit `src/xeadModeler/DialogAddRelationshipOnDatamodel.java`：Git blob **06db0d68e9ebc4ff5b18828abc2de72ee6e34f3a**。**今回本文を直接再取得**し、438–446行・617–625行に`Relationship@Type`・`ExistWhen1/2`生成を確認。
- **XeadServer固定ソース**：commit [7947b52bdb56dc51d2d4c2ba80cba57eda7ece3d](https://github.com/xead/XeadServer/tree/7947b52bdb56dc51d2d4c2ba80cba57eda7ece3d)。`Service.java` blob **3795cc0ce35907bfa614e08da1229ca3b0f6ce4b**、`Session.java` blob **5e65654fad7ce6a6933335117ca86f951c149078**、`DBMethod.java` blob **f0431406deea70121ccc9b27fe794e6f6e02f1d0**、`web.xml` blob **6193108a8b960d46033e96fe7f0fdb96c8465a71**。今回関連行を直接再取得。
- **旧archive**：TEI既存[#1043コメント](https://github.com/hat47x/tei-platform/issues/1043#issuecomment-6222262422)にはarchive root **20dfb2a8b7e9cd0fd01cdb0031305455759c4653**、Server Service.java SHA-256 **D7157AFD2C17181E394C2C643D3465230D570EA8825A3A038845CE7D24F81A16**、Session.java SHA-256 **B58996F3964F822E02952C61CAF8B1927FD3AF9036A7C9B1AB860338EBFDFEFE**が記録済み。今回これらのバイトやdigestは再取得・再計算していない。
- Modeler M50 `xteamdl.jar`：記録済みSHA-256 **0305791a8f5a61291d436f7f04dc8ff1180fa79841bda88b01524d636fb9f04f**。M49とM50の個別メソッド同値、及びServerとModelerの配布版対応は**未照合**。

## 2. TableField：Help → UI → DOM保存 → CreateTable

固定版の該当位置を示す。詳細は既存の[#1126 UI・保存](https://github.com/hat47x/tei-platform/issues/1126#issuecomment-6217185431)、[#1126 Default](https://github.com/hat47x/tei-platform/issues/1126#issuecomment-6217211197)、[#1126 Field初期値](https://github.com/hat47x/tei-platform/issues/1126#issuecomment-6222101937)を正本の既往調査記録とする。

| ステップ | `Modeler.java` 行 | 原典の静的事実 | 未確認／限界 |
|---|---:|---|---|
| Help | 固定archiveの同梱Help、厳密なHTMLファイル名・行は今回未取得 | 既存#1126記録では、Not Nullを「未指定を許さない」、省略値と併用して設定すると説明 | Helpの生バイト／行番号・見出し・翻訳差は今回独立再確認していない |
| UI | 4757–4759、24023–24027、24080–24087 | Not Nullチェック欄を持ち、TableField属性に従いチェック状態を表示 | 実GUIでの選択・状態変化未実行 |
| Field新規追加 | 20434–20481 | `NotNull="true"`、`Default=""`を新Fieldに設定する。TableKeyFieldへのPK追加は別経路（25804–25837） | 初期NotNull=trueはPKメンバーであることの証明でない |
| DOM更新・保存 | 28574–28577、28613–28619 | NotNullのtrue/falseとDefault文字列をそれぞれTableField DOM属性へ反映 | Save-to-file後の再読込・実バイト差分未実行 |
| CreateTable列 | 10248–10253 | `Default`が非空なら`DEFAULT <値>`、`NotNull="true"`なら`NOT NULL`を付与 | 実DDL文字列の生成、DB方言による受理、既存データ検証は未実行 |
| PK制約 | 10281以降 | PK生成は列NotNull付与と別経路 | PK参加から`NotNull`属性を自動書換えすると推定しない |

**fixture件数の不一致を保留：** 旧#1引継ぎ記録には「522 TableField、true319、false203」、後続[#1126コメント](https://github.com/hat47x/tei-platform/issues/1126#issuecomment-6217211197)には「523 Field、true320、false203」と異なる集計がある。今回固定XMLを独立に再集計していないため、後者を正本として無条件採用しない。どちらも`Default`非空0という既往報告だが、個数の差を解消するには入力ZIP・XML digest、抽出XPath、重複・Element単位をそろえた再集計が必要。原典値やモデル本文は転載しない（#941）。

**Editor側は別モデル：** [#1126既存コメント](https://github.com/hat47x/tei-platform/issues/1126#issuecomment-6222189555)にはXeadEditor `DialogImportModel.java`でModeler `NotNull`をEditor `Nullable`へ反転写像する記録がある。これはModeler内部保存やTEIの物理制約の同値性を示さない。

## 3. Relationship端点の任意性とFKの独立判断

既存[#1126 optionalityコメント](https://github.com/hat47x/tei-platform/issues/1126#issuecomment-6222299793)で記録されたM49固定`Modeler.java`の位置：
- `getNotNullOfTableKey()` **25408–25430行**：Key内のTableKeyFieldが参照するTableFieldのNotNullを調べ、1つでもtrueでなければKey全体をnot-nullでないと判定する。
- **16495–16529行**：Relationshipの端点表示は反対側Keyのnot-null集約判定および対応する`ExistWhen`の非空条件に依存する。tooltip参照は16536–16566行。
- **10375–10430、10485–10510、10590行以降**：CreateTableのFK節はglobal FK出力設定、RelationshipのType、ExistWhen等を条件とする。確認された分岐には`getNotNullOfTableKey()`の集約結果がFK節出力条件として直接用いられない。
- 一方、今回直接確認した`DialogAddRelationshipOnDatamodel.java` **438–446、617–625行**は`Relationship@Type`を`REFFER`または導出したTypeとして登録し、`ExistWhen1`と`ExistWhen2`を別属性に生成する。

したがって少なくとも以下は**別の判断**である：

1. **Source fact**：`TableField@NotNull`、`TableKeyField`、`Relationship@Type`、`ExistWhen1/2`が原典にどう宣言されていたか。
2. **Legacy Modelerの関係図端点表示**：`getNotNullOfTableKey`と条件式によって選ばれる視覚表現。TEIのAcceptedな業務cardinality/optionalityの証明ではない。
3. **列DDL**：各Fieldごとの`NOT NULL`と`DEFAULT`。
4. **FK DDL**：Relationship種別、ExistWhen、生成フラグなどの別条件に従う物理参照整合性制約。

旧表示から「このRelationshipが任意だから当該列は必ずNULL許容」と推論しない。逆に列`NOT NULL`やPK存在のみから「業務関係が必須」や「FK節が生成された」と推論しない。Source→Candidate→Owner Acceptance→物理Realizationの各境界で曖昧さを保持する。

TEI現行責務は[#1126](https://github.com/hat47x/tei-platform/issues/1126)。原文文字列と欠落／空を区別した`ExistWhen1/2`のSource Fact保全は既存[#1126追補](https://github.com/hat47x/tei-platform/issues/1126#issuecomment-6390896918)、論理KeyとPK/FKを混同しない設計は[SEMANTIC_LOGICAL_IDENTITY_FD_AUTHORING.md](../design/SEMANTIC_LOGICAL_IDENTITY_FD_AUTHORING.md)、PostgreSQLの実DDL責務は[POSTGRES_CONSTRAINT_RENDERING_DESIGN_2026-10.md](../design/POSTGRES_CONSTRAINT_RENDERING_DESIGN_2026-10.md)／[#1803](https://github.com/hat47x/tei-platform/issues/1803)。新しいDDL engineやRelation Kind、Accepted ADRはここでは増やさない。

## 4. Login／password／sessionは関係・DDLとは独立（#1043、#2222）

固定[Service.java](https://github.com/xead/XeadServer/blob/7947b52bdb56dc51d2d4c2ba80cba57eda7ece3d/WEB-INF/src/xeadServer/Service.java)と[Session.java](https://github.com/xead/XeadServer/blob/7947b52bdb56dc51d2d4c2ba80cba57eda7ece3d/WEB-INF/src/xeadServer/Session.java)について今回直接再読取：

| 経路 | 位置 | 事実 | 推論／未確認 |
|---|---|---|---|
| Credential入力 | Service.init 69–82、doPost 114–157 | `USER/PASSWORD`は要求パラメータ優先、各値が空ならinit-paramへfallback。独自`new Session`を作成する | Helpの「固定」と異なる要求優先は既存[#1043コメント](https://github.com/hat47x/tei-platform/issues/1043#issuecomment-6222271711)。実稼働側のPolicy未確認 |
| DB loginと到達性 | Session.java 96、constructor 150–165、isAuthenticatedUser 166–238 | `userID=""`のfieldを先行`userID.matches("[0-9a-zA-Z]+")`で参照し、引数`user`の代入は成功分岐の後段。照合は入力Passwordをdigest化し`IDUSER/TXPASSWORD`条件のSQLで照合 | 通常初期化ではDB照合に到達しない静的推論。既存[#2222](https://github.com/hat47x/tei-platform/issues/2222)が互換ハーネスの限定修正を所有 |
| Password policyと期間 | Session.java 100–102、171–220、269–299 | 既定MD5、1回、salt無。HashFormatで方式設定。ユーザーレコードDTVALID/DTEXPIREを参照 | DTEXPIREはPassword変更期限の証拠でない。この経路にcomplexity・lockoutは確認されない。別の登録・変更経路、DB・IdPは未確認 |
| 失敗HTTP | Session constructor 150–165、Service.doPost 226–229 | falseなら共通メッセージ、例外をHTTP 500へ送る | DB未到達問題および容器生成の応答本文は未確認。TEIは業務deniedとExecution Errorを[ADR-0020](../adr/0020-capability-business-outcome-boundary.md)で分ける |
| Web Session vs DB Session | Session.writeLogAndStartSession 436–469、closeSession 1570–1626、Service.doPost 175–182 | 独自`NRSESSION`を採番しDBにACT/END等を記録。確認login経路に`getSession/changeSessionId/invalidate`なし | 独自Session生成はHTTP Session ID rotationでない。Session fixation成立や実Cookie設定は未実証 |
| web.xml／DBMethod | 固定web.xml 1–34、DBMethod.doPost 394–469、getConnectionForSession 620–686 | 同梱記述子はDBMethodだけをmappingしHTTP security-constraintやCookie secureを記述しない。DBMethodの`SESSION`は接続Mapのキーに使われる | 最終WAR、Tomcat/Proxy、通信・実認証、SAML別モジュールは未確認 |

既存[#1043コメント](https://github.com/hat47x/tei-platform/issues/1043#issuecomment-6222262422)に原文パラメータのScript引渡しと制限が記録済み。重複して認証Issueを作らない。TEIのWebセッション、CSRF、Secure Cookie、認証後のID再生成・認可は[WEB_APPLICATION_COMPOSITION_DESIGN.md](../design/WEB_APPLICATION_COMPOSITION_DESIGN.md)で別に扱う。[ADR-0019](../adr/0019-authentication-capability-boundary.md)は認証と認可の分離。**これらTEI側の仕様が上流で実装済みという意味ではない。**

## 5. 確認事実・推論・未確認と受入の引継ぎ

- **確認事実**：直接取得した`DialogAddRelationshipOnDatamodel.java`に`Type`と`ExistWhen1/2`の別属性生成がある。固定XeadServerのログイン・DB session・web.xml処理を再読取。#1126にはUI/保存/DDL/optional分離の固定行番号つき証拠が既記録。
- **既存証拠に基づく事実（このターン原ソース本文未再取得）**：巨大`Modeler.java`のTableField UI/DOM/DDL、getNotNullOfTableKey、FK分岐、および固定archive Helpの説明。
- **推論／提案**：旧表示と列制約からCanonical Relationのoptional/cardinality、あるいはFK enforcementを自動確定すべきでない。#1126でOwnerが意味判断し、DB差分/移行は既存[#1803](https://github.com/hat47x/tei-platform/issues/1803)へ委譲。
- **未確認**：Helpの正確なHTMLパス・行番号、M49/M50メソッド同等性、fixture件数不一致の解消、実GUI編集・保存後XML・実CreateTable/DB実行、実配布WAR/Cookie/Proxy/認証成功、原典データからAccepted MeaningへのOwner受理。
- **最小負例（未実行）**：同じKeyで1 FieldだけNotNullをfalseにした場合の図terminalとDDL列を独立比較。さらに`ExistWhen`のみ変更した場合の図terminal／FK節を別比較。元モデルは隔離複製を利用し、原本digest不変を保証。TEIではSource Factの欠落と明示空を混同せず、論理の未受理とDDL生成不可を独立に診断。
- **Issue分担**：#1126は論理意味／UI／Crossview、#1803は物理DDL／Migration、#1043はWeb Session・認証／認可、#2222はXeadServer認証前段不整合、#940は実旧Runtime観測、#941は資料利用条件。#1158はUI Adapter、#1174はCapability Effect Fence、#2020はDelivery Profileであり、以前の#1回答で「password専用」「Web認証監査専用」と割り当てた場合は誤り。

**検証実績：** 固定GitHubコードの読み取り、既存Issue本文・コメント・設計・採用済みADRの静的照合のみ。**未実施：** GUI、通信、認証、DB/DDL、ビルド、テスト、実現環境での比較。新規Issueを作る根拠は現時点でない。
