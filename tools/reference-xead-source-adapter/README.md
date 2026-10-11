# Source Adapter Reference

This command reads a caller-provided XML source file locally and can emit a bounded receipt or candidate evidence. It does not include or distribute external model files. Unknown elements and attributes remain unresolved; source observations do not become accepted TEI meaning.

## Inspect a local file

```bash
go run ./tools/reference-xead-source-adapter -format xead -input /path/to/local-model.xead
```

To exercise optional classification using synthetic data, pass the deliberately incomplete example contract:

```bash
go run ./tools/reference-xead-source-adapter \
  -format xead \
  -input /path/to/local-model.xead \
  -contract reference/xead-source-transcription-contract.synthetic.json
```

The example contract is not comprehensive, does not encode a particular Reference Model, and does not establish semantic acceptance or runtime compatibility. A consumer that needs broader classification must supply and review its own contract.

Observed source values and structural locators can be sensitive. Keep inputs, detailed observations, and generated local evidence outside the public repository. Public receipts omit source values, text, unknown schema names, and structural paths.
