# 現状と引継ぎ

X-TEA制御面の現在地を、到達済みの経路と残作業の性質に分けて示します。詳細なコマンドは[README.md](README.md)、入口の一覧は[ROUTES.md](ROUTES.md)を参照してください。

## 到達したゴール

TEIの参照モデル再現確認で必要になるX-TEAの入力・観測を、GUIの座標操作に依存せず、生成AIが同じ手順で再実行できる状態になりました。

- Modeler / Editor / Driverは、起動時JavaエージェントとシナリオJSONで操作する。
- XeadServerは、HTTPクライアントで操作する。
- 実行した版、コマンド、結果、未確認事項を残せる。
- 別の担当者は`verify-all.ps1`で合成試験と実動試験を再実行できる。

## 到達済みの経路

| コンポーネント | 確認済み | 主な証拠 |
| --- | --- | --- |
| Modeler | 読み取り、About、システム名変更と保存、テーブル名変更と保存、バックアップ | `scenarios/modeler-save-copy.json`、`scenarios/modeler-edit-table-name.json` |
| Editor | 読み取り、システム名変更と保存、バックアップ | `scenarios/editor-save-copy.json` |
| Driver | ログイン、XF100、XF300、XF200の更新・作成・削除、失敗時rollback | `scenarios/driver-*.json` |
| Server | `DBMethod`のSELECT/COUNT/INSERT/UPDATE/DELETE/COMMIT/ROLLBACK/CALL/IP、`Service`のGET/POST、Jetty/Tomcat | `tests/test_live_*.py`、`tests/test_tomcat_war.py` |

`verify-all.ps1`の実動環境変数付き実行で、Swing E2E 9件、サーバ契約4件、実DBMethod 3件、実Service 2件、Tomcat WAR 1件、外部Tomcat 1件が成功しています。

## 残作業の性質

残作業は、制御面の未実装ではなく、次のいずれかに分類できます。

1. **元のUIに存在しない操作**
   - XF200の登録・更新・削除は操作時に確定し、利用者が任意に実行するトランザクションrollbackボタンはありません。
   - 失敗時の内部rollbackは確認済みです。
2. **元のサーブレットが使わない経路**
   - `DBMethod`と`Service`は`.xeaf`から自前のDataSourceを作るため、JNDI DataSourceは実行経路に含まれません。
   - 外部TomcatへのWAR配備は運用上の確認であり、HTTPの入力経路そのものではありません。
3. **別Issueの機能移植**
   - Modeler / Editorのテーブル名変更は確認済みです。要素追加・削除・並べ替え・一括置換・機能編集は、制御面ではなく#1802配下の機能移植として扱います。
4. **上流の修正判断**
   - `Service`の認証前`userID`検査は上流`7947b52`の挙動です。ハーネスはビルド時だけ引数`user`へ置き換えています。上流の修正版確認または差分適用の判断が必要です。

## 引継ぎの入口

1. [README.md](README.md)の「使い方」と「一括検証」を読む。
2. `upstream.json`の版を確認し、`clone-xtea`でX-TEAソースを取得する。
3. `verify-all.ps1`を実行する。実動試験はREADMEの環境変数を設定する。
4. 変更したら`README.md`と`ROUTES.md`を更新し、実行した版、コマンド、結果、未確認事項をPRへ書く。

画面キャプチャには参照モデルの内容が含まれ得ます。公開時は#941の条件に従い、`build/`と`artifacts/`をコミットしません。
