# Bundle Creation

Package WDL workflows with dependencies into ZIP files.

## Usage

```bash
pumbaa bundle create --workflow FILE --output FILE
```

## Flags

| Flag | Alias | Required | Description |
|------|-------|----------|-------------|
| `--workflow` | `-w` | Yes | Main WDL file |
| `--output` | `-o` | Yes | Output ZIP path |

## How It Works

1. Parses main WDL file
2. Finds all `import` statements
3. Resolves import paths
4. Packages all files into ZIP

## Example

Given workflow structure:
```
pipeline.wdl
tasks/
  alignment.wdl
  calling.wdl
```

Create bundle:
```bash
pumbaa bundle create \
  --workflow pipeline.wdl \
  --output bundle.zip
```

Submit with bundle:
```bash
pumbaa workflow submit \
  --workflow pipeline.wdl \
  --dependencies bundle.zip
```

## See Also

- [Submit](submit.md) - Use bundles with submission
