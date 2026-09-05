# Cromwell Hosts

Register Cromwell servers under short names and select them by name.

<div class="grid cards" markdown>

-   :material-server-network: **Type the URL once**

    Register a server, then refer to it as `prod` or `local`

-   :material-swap-horizontal: **Switch per command**

    `--host prod` works on every command, and so does `CROMWELL_HOST=prod`

-   :material-heart-pulse: **Know what is up**

    `pumbaa host check` pings every registered server

</div>

## :material-rocket-launch: Quick Start

```bash
pumbaa host add local http://localhost:8000
pumbaa host add prod https://cromwell.example.com

pumbaa --host prod workflow query     # one command against prod
pumbaa host use prod                  # or make it the default
```

The first host registered becomes the default, so a single-server setup needs
no further configuration.

## :material-console: Commands

| Command | Description |
|---------|-------------|
| `pumbaa host` | List registered hosts (same as `host list`) |
| `pumbaa host add <alias> <url>` | Register a server; `--default` also makes it the default |
| `pumbaa host use <alias>` | Make a registered host the default |
| `pumbaa host remove <alias>` | Remove a host (the default moves to another one) |
| `pumbaa host check [alias]` | Check reachability — of one host, or all of them |

```
$ pumbaa host list

Cromwell hosts
─────────────────────────────────────────
     ALIAS  URL
 →*  local  http://localhost:8000
     prod   https://cromwell.example.com

ℹ → in use now   * default
```

The two markers differ whenever `--host` or `CROMWELL_HOST` overrides the
default, which is exactly when it matters.

## :material-help-circle: How a host is resolved

Anywhere a host is accepted — the `--host` flag, the `CROMWELL_HOST`
environment variable — the value may be an alias or a URL:

1. A **registered alias** always wins.
2. Otherwise, anything carrying a scheme, a dot, a colon or a slash is treated
   as a **URL** (`localhost:8000` becomes `http://localhost:8000`).
3. Anything else is a typo, and is reported as one — with the registered hosts
   and the nearest match:

```
$ pumbaa --host prd workflow query
Error: unknown host "prd" (did you mean "prod"?); registered hosts: local, prod.
Register it with `pumbaa host add prd <url>`, or pass a full URL
```

Aliases are bare words: letters, digits, `-` and `_`. That restriction is what
keeps a reference unambiguous.

With nothing given, the default is the alias marked with `*`, then the
single-server `cromwell_host` setting, then `http://localhost:8000`.

## :material-file-cog: Where it lives

Hosts are stored in `~/.pumbaa/config.yaml`:

```yaml
hosts:
    local: http://localhost:8000
    prod: https://cromwell.example.com
default_host: prod
```

The alias also names the server in the dashboard header and in the
[local run history](history.md), which is keyed by host — the same workflow ID
on two servers is two different runs.

## :material-lightbulb: See also

- [Local run history](history.md) — what was submitted, per host
- [Configuration](../getting-started/configuration.md)
