# ADRの保存場所について

TEIのADRの正本は [`plan/adr/`](../../plan/adr/README.md) です。

ADR-0001〜0015は以前`docs/adr/`を正本としていましたが、本文はすでに`plan/adr/`へ移行済みです。旧URLや過去文書からのリンクを壊さないため、`docs/adr/`には各ADRの案内ファイルだけを互換入口として残しています。

設計判断の本文、新しいADR、現在の判断履歴は [`plan/adr/README.md`](../../plan/adr/README.md) を参照してください。

## このディレクトリの役割

- ADR本文の正本を置かない
- 新しいADRを追加しない
- ADR-0001〜0015の旧URLから`plan/adr/`の正本へ案内する
- 過去の参照互換を保つ

この互換入口を残していることはADR移行の未完了を意味しません。`docs/adr/`と`plan/adr/`を二重正本にはせず、設計判断のauthorityは`plan/adr/`だけにあります。
