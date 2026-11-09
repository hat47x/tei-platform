# Process Plugin: Host管理Environment ProfileのReference（#1521）

このReferenceは`internal/plugin/runtime.HostProcessEnvironmentProfile`を利用し、Process Pluginの実子プロセスへ渡す環境変数をHostが明示的に組み立てる。

```go
profile := runtime.HostProcessEnvironmentProfile{
    SearchDirectories: []string{"/opt/tei/approved-bin"},
    HomeDirectory: "/var/empty/tei-plugin",
    TemporaryDirectory: "/var/tmp/tei-plugin",
    Additional: map[string]string{
        "TEI_PLUGIN_MODE": "reference",
    },
}
launcher, err := runtime.NewStdioProcessLauncherForHostProfile(profile)
if err != nil {
    return err
}
host := runtime.HostSandboxLauncher{
    Network: networkEnforcer,
    Filesystem: filesystemEnforcer,
    Secrets: secretInjector,
    Runner: launcher,
}
```

上例の`networkEnforcer`等はHostによる実装が必要であり、TEIがOS隔離を自動で設置する例ではない。エンドユーザー、Plugin Definition、Command payloadの入力から無審査で`Additional`やディレクトリを構成してはならない。

## 具体的な境界

- `Environment()`は`os.Environ()`や`os.Getenv()`を呼ばず、非nilの環境配列を返す。空ProfileでもHost変数は子へ継承されない。
- `SearchDirectories`は各要素が絶対パスであることを検証して`PATH`にする。相対パス、空要素、パス区切りを埋め込んだ単一要素を拒否する。値の順序はHostが指定した順序を保持。
- `HomeDirectory` / `TemporaryDirectory`はUnixで`HOME` / `TMPDIR`、Windowsで`USERPROFILE` / `TEMP`・`TMP`へ対応付ける。
- `Additional`をキー順に固定して重複・不正書式・NULを拒否。管理対象の`PATH`、`HOME`、`TMPDIR`、`TEMP`、`TMP`、`USERPROFILE`を直接上書きできない。エラーに環境変数の値を表示しない。
- 構成後のHost Profile値を変更しても、生成した`StdioProcessLauncher.Env`は変更されない。
- `StdioProcessLauncher`の既存の実子プロセス起動と、既存`HostSandboxLauncher`・Linux固定実行への接続をそのまま利用し、新ResolverやPlugin専用実行プロトコルは作らない。
- OSの環境自動追加（Windowsの特殊変数等）、ディレクトリ実在性・書込み可否・所有者・権限、実行ファイルの署名・固定、実SecretInjectorの配送・失効、ネットワーク/ファイルシステム隔離は保証しない。信用済みHostがそれぞれ確認する。
- `SearchDirectories`の`PATH`は子プロセス起動時の探索を限定するが、検証済み実行バイナリを固定する契約とは異なる。署名と実行ファイル固定は既存`PinnedLinuxProcess`の責務。
- 高リスクのSecretは`Additional`へ直接埋め込まず、HostのSecretInjectorとResource Authorityへ委ねる。

## 検証境界

`internal/plugin/runtime/process_environment_profile_test.go` は環境配列の決定性、OS別変数マッピング、重複・不正拒否、生成済みLauncherの固定、実Go子プロセスでのHost秘密Canary非継承、Host指定PATHからの実起動を確認する。プラグインとHostが同じOS権限を持つ場合の全面的な隔離、実IdP/Secret Provider、Windows/macOS実端末の試験は別途必要。
