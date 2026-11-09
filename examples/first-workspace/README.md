# 最初の意味変更を試す

この例は、大規模な`reference/workspace`に依存せず、一つの業務概念の定義・検証と変更の追跡を試すための最小限の作業領域です。必要なのはGoの開発ツールだけで、認証情報、データベース、外部プラグインは使いません。

**この例で確認するのは「業務モデルを一か所変更し、検証できること」までです。** 入力フォームの生成、データの保存、登録・参照・更新・削除、Web画面や業務アプリケーションの実行は扱いません。これらへの接続は#1127の後続段階で扱います。

## 1. 最小限の作業領域を確認する

リポジトリのルートから実行します。

```sh
go run ./cmd/tei validate -workspace examples/first-workspace
go run ./cmd/tei source-index -workspace examples/first-workspace -subject-kind semantic-data -subject-id intake.request
go run ./cmd/tei world -workspace examples/first-workspace
```

この例では、次の三つのファイルを正本として扱います。

- `tei.workspace.yaml`：管理するソースの起点を指定する
- `semantic/intake/request.tei`：受付依頼（Request）の識別子・概要・状態の意味を定義する
- `applications/request-log/application.tei`：業務アプリケーションの識別子と、利用する意味の定義を記す

`source-index`で`semantic-data:intake.request`を調べると、`semantic/intake/request.tei`の定義位置とソースの指紋値が返ります。`world`は、ファイルの物理的なパスを意味の正本に混ぜず、正規化された意味を表示します。

## 2. 作業領域をコピーして変更する

次の`cp`はUnix系シェルで使うコマンドです。Windows PowerShellでは後述の`Copy-Item`を使用できます。

```sh
cp -R examples/first-workspace /tmp/tei-first
```

Windows PowerShellでは次のコマンドでコピーしてください。それ以降の`/tmp/tei-first`は`$env:TEMP/tei-first`に読み替えます。

```powershell
Copy-Item -Path examples/first-workspace -Destination "$env:TEMP/tei-first" -Recurse
```

コピーした`/tmp/tei-first/semantic/intake/request.tei`の`data Request`へ、次の属性を一行追加します。

```tei
    attribute priority type std.code
```

変更後、次のコマンドを実行します。

```sh
go run ./cmd/tei validate -workspace /tmp/tei-first
go run ./cmd/tei source-index -workspace /tmp/tei-first -subject-kind semantic-attribute -subject-id intake.request.priority
go run ./cmd/tei world -workspace /tmp/tei-first
```

`intake.request.priority`が追加され、既存の`intake.request`と`request_log`の識別子は変わりません。ここで確認できるのは意味の追加だけです。データベースの列や画面が自動的に生成・更新されたわけではありません。

変更後は必ずソース索引を再生成してください。変更前の指紋値に基づく参照位置や内容を、現在の検証根拠として使い回してはいけません。編集するのは`.tei`の正本であり、`tei world`が出力したJSONではありません。

## 3. 不正な定義を検出して修正する

コピーした`request.tei`で、`priority`属性の型を一時的に`std.not_defined`へ変更し、検証を実行します。

```sh
go run ./cmd/tei validate -workspace /tmp/tei-first
```

`UNKNOWN_SEMANTIC_ATTRIBUTE_TYPE`と`semantic/intake/request.tei`の定義位置が表示され、終了コードが0以外になるのが期待結果です。型を推測して検証を通すことはしません。`std.code`へ戻し、検証が成功することを確認します。

参照先を誤記した場合も検証できます。例えば同じ`data Request`に`relation parent to intake.unknown_request one`を追加すると、`UNKNOWN_SEMANTIC_RELATION_TARGET`と`parent`の定義位置が表示されるはずです。`intake.unknown_request`を実在する`intake.request`へ訂正し、再検証します。この自己参照が検証できても、データの保存、選択画面、登録・参照・更新・削除の実行が成立したわけではありません。

参照先のデータ定義が存在していても、個々のデータを関連付ける`key`の属性型が異なれば不正です。次の型の不一致を試せます。

```tei
    relation parent to intake.request one
        key intake.request.summary -> intake.request.request_id
```

`summary`は`std.text`、`requestId`は`std.identifier`なので、通常の`tei validate`では`SEMANTIC_RELATION_KEY_TYPE_MISMATCH`が報告され、関係の定義元が示されることを期待します。参照元を`intake.request.request_id`へ訂正すると検証は成功します。ただし、**意味上の`key`を検証できても、物理データベースの外部キーや関連検索が作られた証拠にはなりません**。

意味上の操作でも、存在しない入力型や出力型は拒否されます。例えば`capability intake.request.submit { input request intake.unknown_request output intake.unknown_result }`を宣言すると、`UNKNOWN_SEMANTIC_CAPABILITY_INPUT`と`UNKNOWN_SEMANTIC_CAPABILITY_OUTPUT`が発生し、機能の定義位置が示されます。どちらも`intake.request`へ訂正すると、モデルの検証は成功します。ただし、操作の宣言だけで実行機能、権限、提供実装、画面が成立するわけではありません。

アプリケーションが参照する意味の定義名を誤記すると、`UNKNOWN_APPLICATION_SEMANTIC_NAMESPACE`とともにアプリケーションの定義位置が示されます。構文エラーの場合は`PARSE_ERROR`になります。どちらも元の定義を修正して再検証します。

## 第1段階で確認できることと次の課題

この例で扱うのは、(a) 独立した作業領域の作成、(b) 一か所の編集、(c) ソースの位置と正本となる意味の確認、(d) 不正な参照の検出と修正です。回帰テストは`go test ./internal/workspace -run TestFirstModelChange -count=1`で実行できます。実際の`tei`コマンドをビルドし、正常状態から編集、エラー、修正までを確認するテストは`go test ./cmd/tei -run TestFirstModelChangeCLIEndToEnd -count=1`です。

次の課題は、利用者が作った同じ作業領域から、標準的な登録・参照・更新・削除、検索画面、データを扱えるようにすること（#1127）です。その際の操作数、手入力する箇所、修正に要する時間を、外部の利用者にも測れるようにします。定義の検証が成功しても、業務で利用できるとまでは主張しません。
