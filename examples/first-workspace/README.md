# First Model Change — 最初の意味変更を試す

この例は、巨大な`reference/workspace`に依存せずに、一つの業務概念を定義・検証し、意味変更を追跡する最小のWorkspaceです。必要なのはGo toolchainだけです。認証情報、DB、外部Pluginは不要です。

**達成するのは「業務モデルを一か所変更して検証できる」段階までです。** フォーム生成、データ保存、CRUD、Web画面や実際の業務アプリを動かす例ではありません。これらへの接続は#1127の後続段階として扱います。

## 1. 既存の最小Workspaceを確認する

リポジトリのルートから実行します。

```sh
go run ./cmd/tei validate -workspace examples/first-workspace
go run ./cmd/tei source-index -workspace examples/first-workspace -subject-kind semantic-data -subject-id intake.request
go run ./cmd/tei world -workspace examples/first-workspace
```

この例の正本は次の3ファイルです。

- `tei.workspace.yaml`: 管理するSource rootの指定
- `semantic/intake/request.tei`: 受付依頼（Request）のID・概要・状態の意味
- `applications/request-log/application.tei`: 業務ApplicationのIDと使用するSemantic

`source-index`の`semantic-data:intake.request`は、`semantic/intake/request.tei`の定義位置とSource fingerprintを返します。`world`は物理的なSource pathを正本に混ぜず、正規化された意味を表示します。

## 2. 自分のWorkspaceで最初の変更を行う

以下の`cp`はUnix系シェルの例です（ほかの環境では同等のディレクトリコピーを使用）。

```sh
cp -R examples/first-workspace /tmp/tei-first
```

`/tmp/tei-first/semantic/intake/request.tei`の`data Request`に、属性を一行追加します。

```tei
    attribute priority type std.code
```

もう一度次を実行します。

```sh
go run ./cmd/tei validate -workspace /tmp/tei-first
go run ./cmd/tei source-index -workspace /tmp/tei-first -subject-kind semantic-attribute -subject-id intake.request.priority
go run ./cmd/tei world -workspace /tmp/tei-first
```

`intake.request.priority`が新たに現れ、既存の`intake.request`と`request_log`のIDは変わりません。この段階では意味の追加が確認できるだけで、DB列や画面が自動生成・更新されたわけではありません。

## 3. 失敗と修復を確認する

コピーした`request.tei`の`priority`属性の型を一時的に`std.not_defined`へ変更し、Validationを実行します。

```sh
go run ./cmd/tei validate -workspace /tmp/tei-first
```

`UNKNOWN_SEMANTIC_ATTRIBUTE_TYPE`のエラーと、`semantic/intake/request.tei`の定義位置が表示され、非0終了になることが期待結果です。「うまく推定して通す」ことはしません。型を`std.code`に戻してValidationが成功することを確認します。

Applicationが参照するSemanticの名前を誤記した場合は、`UNKNOWN_APPLICATION_SEMANTIC_NAMESPACE`としてApplication定義の位置を示します。構文エラーは`PARSE_ERROR`となります。いずれも元の定義を修復したうえで再検証します。

## 第1段階の完了と次の課題

この例は、(a) 独立Workspaceの作成、(b) 一か所の編集、(c) Source位置とCanonical意味の確認、(d) 不正な参照の検出・修復までを扱います。回帰テストは`go test ./internal/workspace -run TestFirstModelChange -count=1`で実行できます。実際の`tei` CLIをbuildして、正常→編集→エラー→修復の操作を確認するテストは`go test ./cmd/tei -run TestFirstModelChangeCLIEndToEnd -count=1`です。

次に必要なのは、同じ利用者作成Workspaceから標準的なCRUD・検索画面とデータを扱えるようにすること（#1127）、その際の操作数・手入力箇所・修復時間を外部の利用者にも測定可能にすることです。Validationの成功を業務利用可能という主張へ拡張しません。
