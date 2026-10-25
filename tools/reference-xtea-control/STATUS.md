# 現状と引継ぎ

X-TEA制御面の現在地を、到達済みの経路と残作業の性質に分けて示します。詳細なコマンドは[README.md](README.md)、入口の一覧は[ROUTES.md](ROUTES.md)を参照してください。

## 到達したゴール

TEIの参照モデル再現確認で必要になるX-TEAの入力・観測を、GUIの座標操作に依存せず、生成AIが同じ手順で再実行できる状態になりました。

- Modeler / Editor / Driverは、起動時JavaエージェントとシナリオJSONで操作する。
- `.xead`は`xtea_cli.py`で直接読み、テーブル・フィールド・キー・機能・入出力・呼び出し関係をJSONで取得する。
- 実行した版、コマンド、結果、未確認事項を残せる。
- 別の担当者は`verify-all.ps1`で合成試験と実動試験を再実行できる。
- XeadServerはTEIの参考対象外とし、この制御面の対象に含めない。

## 到達済みの経路

| コンポーネント | 確認済み | 主な証拠 |
| --- | --- | --- |
| Modeler | 読み取り、About、システム名変更と保存、テーブル名変更と保存、フィールド名変更・追加・削除・型変更・並べ替えと保存、バックアップ | `scenarios/modeler-*.json` |
| Editor | 読み取り、システム名変更と保存、テーブル名変更と保存、テーブル一覧から編集画面を開く、テーブル追加・削除、機能追加・機能ID変更と保存、バックアップ | `scenarios/editor-*.json` |
| Driver | ログイン、XF100、XF300、XF200の更新・作成・削除、失敗時rollback | `scenarios/driver-*.json` |
| モデルCLI | システム、テーブル、フィールド、キー、機能種別、入出力CRUD、呼び出し関係、2モデルの差分、参照検査のJSON出力 | `xtea_cli.py`、`tests/test_model_cli.py` |
| XeadServer | **対象外**。`xtea_server.py`と`server-harness/`は過去の参考資産 | `tests/test_live_*.py`、`tests/test_tomcat_war.py` |

`verify-all.ps1`の既定実行ではSwing E2E 10件とモデルCLI 7件を確認します。XeadServerの参考試験は`-IncludeXeadServer`を付けた場合だけ実行し、TEIの比較・受入には使いません。

## 残作業の性質

残作業は、制御面の未実装ではなく、次のいずれかに分類できます。

1. **元のUIに存在しない操作**
   - XF200の登録・更新・削除は操作時に確定し、利用者が任意に実行するトランザクションrollbackボタンはありません。
   - 失敗時の内部rollbackは確認済みです。
2. **TEIの参考対象外**
   - XeadServerは対象外です。`DBMethod` / `Service` / WAR / Tomcat / JNDI / 認証は、TEIの比較・受入・新規検証に使いません。
   - 既存のコードと試験は過去の調査資産であり、`-IncludeXeadServer`を付けた場合だけ参考実行します。
3. **別Issueの機能移植**
   - Modeler / Editorのテーブル名変更は確認済みです。要素追加・削除・並べ替え・一括置換・機能編集は、制御面ではなく#1802配下の機能移植として扱います。
   - Editorのフィールド一覧は`Field` / `TableField`の版差により空になるため、#2199で扱います。

## 引継ぎの入口

1. [README.md](README.md)の「使い方」と「一括検証」を読む。
2. `upstream.json`の版を確認し、`clone-xtea`でX-TEAソースを取得する。
3. `verify-all.ps1`を実行する。実動試験はREADMEの環境変数を設定する。
4. 変更したら`README.md`と`ROUTES.md`を更新し、実行した版、コマンド、結果、未確認事項をPRへ書く。

画面キャプチャには参照モデルの内容が含まれ得ます。公開時は#941の条件に従い、`build/`と`artifacts/`をコミットしません。
