# CLI Settings for the crawljob Config

**Date:** 2026-06-28
**Status:** Approved (design)

## Problem

The JDownloader `.crawljob` handoff (see `2026-06-27-jdownloader-crawljob-handoff-design.md`)
is configured through a global `crawljob:` block in the YAML config. Today the only way to
change those values — enable/disable the handoff, point it at JDownloader's folderwatch
directory, manage the domain blacklist — is to hand-edit the YAML file. That is tedious and
error-prone. We want first-class CLI commands to read and write these settings, matching the
ergonomics watcher-go already provides for module settings and proxy limits.

## Scope

The `crawljob` global config block only. Six fields:

| Key | Type | Default |
|-----|------|---------|
| `enabled` | bool | false |
| `file` | string | `./watcher-go.crawljob` |
| `folderwatch_path` | string | (empty) |
| `blacklist` | []string | (empty) |
| `auto_start` | bool | true |
| `auto_confirm` | bool | true |

Not in scope: refactoring the existing module settings framework
(`internal/models/module_settings.go`); filesystem validation of paths; any other config block.

## Existing precedents

- **`proxy-limits`** (`cmd/watcher/proxy_limits.go`): a command group with `list`/`set`/`remove`
  subcommands that mutate Viper and persist via `raven.CheckError(viper.WriteConfig())`.
- **`module <key> set/get/settings`** (`internal/models/module_settings.go`): a reflection-based
  framework that walks a schema struct's `mapstructure` tags to provide typed set/get/list. It is
  bound to `Modules.<key>.<setting>` keys and the `Module` type, so it is **not** directly
  reusable for the global `crawljob` block — but its approach (reflect over the schema, typed
  parse) is the model this design follows in a self-contained way.

## Decisions

1. **Command shape:** `crawljob set/get/settings` for the scalar fields, plus dedicated
   `crawljob blacklist list|add|remove` for the list field (incremental list editing, mirroring
   `proxy-limits`).
2. **Implementation:** focused and self-contained. The settings logic lives in the
   `jdownloader` package with `jdownloader.Config` as the single source of truth (reflected over
   its `mapstructure` tags, so the command never drifts from the struct). The existing module
   settings framework is left untouched.
3. **Effective values:** `get`/`settings` read through `LoadConfig()` so they display the value
   that will actually be used (defaults applied), not the raw YAML.

## Architecture

```
internal/jdownloader/settings.go        NEW — pure, testable settings logic over Config
internal/jdownloader/settings_test.go   NEW — unit tests for the above
cmd/watcher/crawljob.go                  EXTEND — add set/get/settings + blacklist subcommands
```

### `internal/jdownloader/settings.go`

Pure functions; no cobra, no `viper.WriteConfig` (persistence stays in the command layer).

- `type Setting struct { Key string; Kind reflect.Kind; IsSlice bool; Value any }`
- `func Settings() []Setting` — reflect over the value returned by `LoadConfig()` (effective
  values, defaults applied), producing one `Setting` per `mapstructure`-tagged field, in a
  stable declaration order. Powers `settings` (all) and `get` (lookup by key). The `blacklist`
  field is included with `IsSlice = true`.
- `func ParseScalar(key, value string) (parsed any, err error)` — looks up the key's field type
  by reflecting over `Config`; returns an error for an unknown key, for a slice key (message:
  use the `blacklist` subcommands), or for a value that fails typed parsing. Supports `bool` and
  `string` today (the only scalar kinds present); structured the same way as the module
  framework's `parseTypedValue` so adding an int/uint field later is a one-line extension.
- `func ViperKey(key string) string` — returns `"crawljob." + key` (single place that knows the
  block prefix).
- Blacklist helpers (operate on a plain slice; caller reads from / writes to Viper):
  - `func NormalizeDomain(d string) string` — `strings.ToLower(strings.TrimSpace(d))`.
  - `func AddBlacklistDomain(list []string, domain string) (out []string, added bool)` —
    normalizes, returns `(list, false)` if already present (idempotent), else appends.
  - `func RemoveBlacklistDomain(list []string, domain string) (out []string, removed bool)` —
    case-insensitive match; returns `(list, false)` if absent.

  These match `Blacklisted()`'s normalization (it lowercases at match time), so stored entries
  stay clean and dedupe/removal behave intuitively.

### `cmd/watcher/crawljob.go` (extend `addCrawljobCommand`)

Thin Cobra glue beside the existing `merge` subcommand, mirroring `proxy-limits`:

- `set [key] [value]` (`cobra.ExactArgs(2)`): `parsed, err := jdownloader.ParseScalar(...)`;
  on error print it (and, for an unknown key, the valid key list); else
  `viper.Set(jdownloader.ViperKey(key), parsed)` → `raven.CheckError(viper.WriteConfig())` →
  print `set <key> = <value>`.
- `get [key]` (`ExactArgs(1)`): find the key in `jdownloader.Settings()`; print
  `<key> = <value>` or, for an unknown key, the valid list.
- `settings` (`NoArgs`): print every `jdownloader.Settings()` entry aligned as
  `<key>  <value>` (type shown in a trailing column), e.g. the example in the design.
- `blacklist` group:
  - `list` — print the current `crawljob.blacklist` (read via `viper.GetStringSlice`), or
    "no domains blacklisted".
  - `add [domain]` (`ExactArgs(1)`): read list, `AddBlacklistDomain`; if added,
    `viper.Set` + `WriteConfig` + print `added <domain>`; else print `<domain> already in blacklist`.
  - `remove [domain]` / alias `rm` (`ExactArgs(1)`): read list, `RemoveBlacklistDomain`; if
    removed, persist + print `removed <domain>`; else print `<domain> not in blacklist`.

The blacklist is written back as a plain `[]string` via `viper.Set`, matching how
`LoadConfig`/`Blacklisted` already consume `crawljob.blacklist`.

## Error handling

- Unknown key (`set`/`get`) → message + the list of valid keys.
- `set blacklist …` → error: `blacklist is a list; use "watcher crawljob blacklist add/remove/list"`.
- Bad scalar value → `invalid value for <key> (expected <type>): <err>`.
- `viper.WriteConfig()` failure → `raven.CheckError` (fatal; codebase convention).
- `blacklist add` already-present / `remove` absent → friendly message, no error, no write.

## Testing

`internal/jdownloader/settings_test.go` (plain `testing`, no testify):

- `ParseScalar`: `enabled true` → bool true; `file ./x` → string; unknown key → error; slice key
  `blacklist` → error mentioning the blacklist subcommands; `enabled notabool` → parse error.
- `Settings`: with `viper.Reset()` then selected `viper.Set` calls — asserts all six keys are
  present with correct `Kind`/`IsSlice`, and that effective values reflect defaults (e.g.
  `auto_start` true, `file` `./watcher-go.crawljob`) when unset and overrides when set.
- Blacklist helpers: `AddBlacklistDomain` appends + dedupes + normalizes (`Discord.GG` →
  `discord.gg`, second add is a no-op); `RemoveBlacklistDomain` removes case-insensitively and
  reports absent.

The Cobra glue is thin (like `merge`, which has no unit test) → covered by `go build`,
`crawljob --help`, the unit tests above, and manual `set`/`get`/`settings`/`blacklist` runs.

## Out of scope

- Refactoring/​generalizing the module settings framework.
- Filesystem validation of `file` / `folderwatch_path` (set leniently, like `proxy-limits`).
- Settings for any config block other than `crawljob`.
