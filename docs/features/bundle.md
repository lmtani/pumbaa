# Bundle Creation

Package WDL workflows with dependencies into ZIP files.

---

## :rocket: Usage

```bash
pumbaa bundle create --workflow FILE --output FILE
```

---

## :flags: Flags

| Flag | Alias | Required | Description |
|------|:-----:|:--------:|-------------|
| `--workflow` | `-w` | :white_check_mark: | Main WDL file |
| `--output` | `-o` | :white_check_mark: | Output ZIP path |

---

## :gear: How It Works

```mermaid
flowchart LR
    A[Parse main WDL] --> B[Find imports]
    B --> C[Resolve paths]
    C --> D[Package to ZIP]
```

1. Parses main WDL file
2. Finds all `import` statements
3. Resolves import paths
4. Packages all files into ZIP

---

## :bulb: Example

Given workflow structure:

```
pipeline.wdl
tasks/
  alignment.wdl
  calling.wdl
```

=== "Create Bundle"

    ```bash
    pumbaa bundle create \
      --workflow pipeline.wdl \
      --output bundle.zip
    ```

=== "Submit with Bundle"

    ```bash
    pumbaa workflow submit \
      --workflow pipeline.wdl \
      --dependencies bundle.zip
    ```

---

## :books: See Also

- [:material-upload: Submit](submit.md) — Use bundles with submission
