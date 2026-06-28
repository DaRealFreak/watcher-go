# Unified `watcher config` Command

**Date:** 2026-06-28
**Status:** Approved (design)
**Supersedes:** `2026-06-28-crawljob-cli-settings-design.md` (crawljob settings are absorbed into this command; that spec file is removed).

## Problem

watcher-go's persisted settings are scattered across many places and several CLI surfaces:

- **17 modules**, each with its own `Modules.<key>.*` settings, edited via `module <key> set/get/settings`.
- A global `crawljob.*` block (only editable by hand-editing YAML today).
- A `run.proxy_connection_limits` list-of-struct block, edited via a separate top-level `proxy-limits` command.
- Loose global scalars: `download.directory`, `database.path`, `watcher.sentry`.

There is **no central place** to see or change settings, and no single source of truth for "what can I configure and where." Users must remember which command edits what, and global blocks like `crawljob` require manual YAML edits. We want one command — `watcher config` — that centrally lists, reads, and writes all persisted settings.

## Decisions

1. **Replace, don't add.** `config` becomes the single settings command. The generic `module <key> set/get/settings` subcommands and the top-level `proxy-limits` command are removed; their functionality moves into `config` (with shared, not duplicated, logic). `module <key>` remains for module-specific *actions* (incl. proxy editing). `crawljob merge` remains (it is an action, not a setting).
2. **Registry-backed.** A new `internal/settings` package builds a runtime registry aggregating every settable key from the module factory and the global blocks. The reflection + typed-parse helpers currently in `internal/models/module_settings.go` move here (generalized); `module_settings.go` is deleted.
3. **Flat dotted addressing with real module keys.** Keys are a single flat namespace resolved by exact registry lookup: `crawljob.enabled`, `download.directory`, `pawchive.st.external_urls.download_external_items`. The real module key (`pawchive.st`) is used in the address; the registry maps it to the sanitized Viper key (`Modules.pawchive_st.…`). No `_`-sanitization is exposed to the user. `config list` prints the exact addressable key for every setting.
4. **Effective values.** `get`/`list` display the value that will actually be used (registry defaults applied when Viper is unset), so e.g. `crawljob.auto_start` reads `true` even before it is written.
5. **Scope:** module scalar + `[]string` settings, `crawljob.*`, `download.directory` (global + per-module override), `database.path`, `watcher.sentry`, and `run.proxy_connection_limits` (via a moved subcommand). Transient command flags (backup/restore toggles, `--force`, verbosity, sentry enable/disable *flags*, etc.) are excluded — they are per-invocation, not persisted preferences.

## Command surface

```
config list [filter]                  # every setting, grouped by source, with type + effective value
config get <key>                      # one setting's effective value
config set <key> <value>              # typed-parse, validate, persist (scalars; []string via comma)
config list-add <key> <value>         # append to a []string setting (idempotent, normalized where relevant)
config list-remove <key> <value>      # remove from a []string setting
config proxy-limits list|set|remove   # the run.proxy_connection_limits list-of-struct block
```

`config list` output is grouped: a `[global]` group, a `[crawljob]` group, and one `[module: <key>]` group per module, each line `<key>  <type>  <effective value>`. An optional substring `filter` narrows the listing. Read-only/complex entries (per-module `loopproxies`/`proxy`) are shown with a `(complex — edit via "module <key>" proxy commands)` note rather than a value-set affordance.

## Architecture

```
internal/settings/                 NEW package — the registry + reflection + typed parsing
  registry.go    — Registry type, Build(), Resolve(key), Entries()
  reflect.go     — schema walk + typed parse (moved/generalized from models/module_settings.go)
  globals.go     — explicit global entries + crawljob (reflected from jdownloader.Config) with defaults
  *_test.go      — unit tests

cmd/watcher/config.go              NEW — the `config` cobra command group
cmd/watcher/proxy_limits.go        EDIT — keep read/write/load helpers; expose subcommands for config to mount; drop the top-level registration
cmd/watcher/modules.go             EDIT — stop calling AddSettingsCommand
cmd/watcher/main.go                EDIT — register addConfigCommand(); remove addProxyLimitsCommand()
internal/models/module_settings.go REMOVE — logic moved to internal/settings
```

No import cycles: `internal/settings` imports `internal/modules` (factory), `internal/models` (Module type), and `internal/jdownloader` (Config); none of those import `internal/settings`. `cmd/watcher` imports `internal/settings`.

### Registry model

```go
// Kind classifies how a setting is edited.
type Kind int
const (
    KindScalar     Kind = iota // bool/string/int/uint/float — set via `config set`
    KindStringList             // []string — set/list-add/list-remove
    KindStructList             // []struct — read-only here (e.g. loopproxies); proxy_connection_limits handled by its own subcommand
)

type Entry struct {
    Key      string        // address/display key, e.g. "pawchive.st.external_urls.download_external_items"
    ViperKey string        // actual viper key, e.g. "Modules.pawchive_st.external_urls.download_external_items"
    Type     reflect.Type  // element type for scalars / []string
    Kind     Kind
    Group    string        // "global", "crawljob", or the module key
    ReadOnly bool          // true for KindStructList (and any complex type)
    Default  any           // optional; displayed when viper is unset (crawljob file/auto_start/auto_confirm)
}

type Registry struct { /* ordered entries + key->entry map */ }

func Build() *Registry                       // modules (factory) + crawljob + globals
func (r *Registry) Resolve(key string) (*Entry, bool)
func (r *Registry) Entries() []Entry         // stable order: global, crawljob, then modules alpha
func (r *Registry) EffectiveValue(e Entry) any // viper.IsSet(e.ViperKey) ? viper.Get : e.Default
```

- **Module entries:** for each `module` in `modules.GetModuleFactory().GetAllModules()`, walk `module.SettingsSchema` (recursive over `mapstructure` tags, same algorithm as today's `extractFromType`). Leaf scalar → `KindScalar`; `[]string` → `KindStringList`; `[]struct` (e.g. `loopproxies`) → `KindStructList` (ReadOnly). `Key = module.Key + "." + path`; `ViperKey = "Modules." + module.GetViperModuleKey() + "." + path`. Also register `<module.Key>.download.directory` (string scalar) as a per-module override.
- **crawljob entries:** walk `jdownloader.Config` under prefix `crawljob`; set `Default` for `file` (`./watcher-go.crawljob`), `auto_start` (true), `auto_confirm` (true). `blacklist` → `KindStringList`.
- **Global entries:** explicit list — `download.directory` (string), `database.path` (string), `watcher.sentry` (bool). `run.proxy_connection_limits` is NOT a generic entry; it is surfaced read-only in `list` with a pointer to `config proxy-limits`.

### Typed parsing

Generalized from `parseTypedValue`: supports `string`, `bool`, `int*`, `uint*`, `float*`, and `[]string` (comma-split). Non-string slices / structs are rejected with a message pointing to the right command or to manual YAML editing. `friendlyTypeName` is reused for messages and `list` output.

### `config` command behavior (cmd/watcher/config.go)

Thin cobra glue over the registry, persisting with `viper.Set(entry.ViperKey, …)` + `raven.CheckError(viper.WriteConfig())` (the established pattern):

- `set <key> <value>`: `Resolve`; if not found → unknown-key error listing how to discover (`config list`); if `ReadOnly`/`KindStructList` → error pointing to the structured command; else typed-parse against `entry.Type` and persist. For `KindStringList`, `set` replaces the whole list (comma-split) and the output suggests `list-add`/`list-remove` for incremental edits.
- `get <key>`: `Resolve` → print `EffectiveValue`. Unknown → error.
- `list [filter]`: iterate `Entries()`, group by `Group`, print aligned `key  type  value` (or the read-only note). Apply substring filter to the key if provided.
- `list-add` / `list-remove <key> <value>`: require `KindStringList`; read current slice (`viper.GetStringSlice`), apply add (idempotent; normalized for the crawljob blacklist via the same trim+lowercase rule `Blacklisted()` uses) or remove (reporting absent), persist. For non-list keys → error.
- `proxy-limits`: mount the `list`/`set`/`remove` subcommands built from the existing helpers in `proxy_limits.go`.

### proxy-limits move

`proxy_limits.go` keeps `readProxyLimitsList`, `writeProxyLimitsList`, and `loadProxyConnectionLimits` (the last is still called at startup in `main.go`). The cobra subcommands are exposed via a function (e.g. `proxyLimitsSubcommands() []*cobra.Command`) that `config.go` mounts under `config proxy-limits`. The top-level `addProxyLimitsCommand()` registration is removed from `main.go`.

## Error handling

- Unknown key → `unknown setting "<k>"; run "watcher config list" to see available settings`.
- `set` on read-only/struct key → `"<k>" is a <kind>; edit it via <pointer>` (per-module proxy commands, or `config proxy-limits`).
- `set`/`list-add`/`list-remove` type mismatch → `invalid value for <key> (expected <type>): <err>`.
- `list-add`/`list-remove` on a non-`[]string` key → error naming the key's kind.
- `list-add` already present / `list-remove` absent → friendly message, no write.
- `viper.WriteConfig()` failure → `raven.CheckError` (fatal; codebase convention).

## Testing

`internal/settings/*_test.go` (plain `testing`, no testify):

- **Registry build:** with the real factory (modules are registered via the standard blank-imports) assert representative entries exist with the correct `ViperKey` sanitization (e.g. `pawchive.st.external_urls.download_external_items` → `Modules.pawchive_st.external_urls.download_external_items`), correct `Kind` (a `[]string` like a blacklist → `KindStringList`; `loopproxies` → `KindStructList`/ReadOnly), and that crawljob + the three globals are present.
- **Resolve:** known key → entry; unknown → `false`.
- **Typed parse:** bool/string/int/[]string success; bad bool → error; struct/non-string-slice → rejected.
- **EffectiveValue:** crawljob defaults show through when unset (`auto_start` true, `file` default) and overrides win when `viper.Set` (`viper.Reset` between cases).
- **`[]string` add/remove:** append + dedupe + normalize (`Discord.GG` → `discord.gg`, second add no-op); remove case-insensitive; remove-absent reports false.

The cobra `config` glue is thin (like `crawljob merge`) → covered by `go build`, `config --help`, the registry unit tests, the already-proven proxy-limits logic, and a manual pass (`config list`, a `set`, a `list-add`, `proxy-limits set`).

## Migration (user-facing change)

- `module <key> set <path> <v>`  →  `config set <key>.<path> <v>`
- `module <key> get <path>`      →  `config get <key>.<path>`
- `module <key> settings`        →  `config list <key>` (filter)
- `proxy-limits set/list/remove` →  `config proxy-limits set/list/remove`

## Out of scope

- Fully-generic editing of arbitrary list-of-struct settings (only `proxy_connection_limits` gets structured editing, via the moved command; `loopproxies`/`proxy` remain editable through the existing per-module proxy commands).
- Migrating existing YAML config files (Viper keys are unchanged; only the CLI surface changes).
- Exposing transient command flags as persisted settings.
