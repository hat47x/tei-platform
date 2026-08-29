# Contributing to TEI

TEI is currently an experimental architecture project.

Code contributions are welcome, but they are not the only useful form of participation. In particular, the project values:

- reports from trying the reference workspace or CLI,
- examples where an existing enterprise system does not fit the current model,
- migration / database / deployment edge cases,
- comparisons with established low-code, model-driven, workflow, or schema tools,
- published design notes and operational findings,
- small reproducible examples that expose an architectural mismatch.

Before making a large implementation contribution, opening an issue with the intended boundary and use case is usually more useful than immediately adding a new Core primitive.

## Design preference

TEI generally prefers:

- explicit semantics over runtime guessing,
- small kernels over accumulating special kinds,
- native formats where they are already strong,
- deterministic generation where AI is unnecessary,
- observed facts over treating command acknowledgements as truth,
- provider boundaries over hard-coding one technology into semantic definitions.

## Development

```bash
go test ./...
go vet ./...
go run ./cmd/tei validate -workspace reference/workspace
```

The reference workspace intentionally contains historical contract warnings.
