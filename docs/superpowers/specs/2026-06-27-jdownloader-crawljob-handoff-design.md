# JDownloader `.crawljob` Handoff for External Links

**Date:** 2026-06-27
**Status:** Approved (design)

## Problem

Modules that scrape post-based services (pawchive, pixiv, kemono, patreon) frequently
find external download URLs (mega.nz, mediafire, etc.) embedded in posts and comments.
watcher-go has no module that can download these hosts, so today they are simply logged
via `slog.Warn` ("found external URL …") and dropped. The user manually copies these URLs
out of the log and feeds them into JDownloader by hand, into the correct per-post folder.

We want watcher-go to instead accumulate these unparseable links into a JDownloader
`.crawljob` file — one that records the correct destination folder per post — and provide a
command to push that file into JDownloader's Folder Watch directory so JDownloader downloads
each file straight into the right place, with no manual copying.

## Background: JDownloader Folder Watch / `.crawljob`

JDownloader ships a built-in **Folder Watch** extension that monitors a directory and ingests
any `.crawljob` file dropped into it. A `.crawljob` is a JSON array of package entries. Each
entry can specify the exact download folder, package name, the links, and whether to
auto-confirm (skip the LinkGrabber review) and auto-start. JDownloader's own plugins then
resolve/download the hosts (mega, mediafire, …). No authentication or network API is required —
we just write a file. This is why this approach was chosen over the MyJDownloader cloud API
(heavier: account auth, AES-encrypted payloads, no maintained Go client) and the deprecated
local HTTP API (unreliable).

Relevant `.crawljob` fields used:

| Field | Value | Meaning |
|-------|-------|---------|
| `packageName` | per-post string | package label in JDownloader |
| `downloadFolder` | absolute path | destination directory |
| `text` | newline-joined URLs | the links to add |
| `comment` | source post URL | provenance, shown in JDownloader |
| `enabled` | `"true"` | entry is active |
| `autoConfirm` | `"TRUE"`/`"FALSE"`/`"UNSET"` | skip LinkGrabber review |
| `autoStart` | `"TRUE"`/`"FALSE"`/`"UNSET"` | begin download immediately |
| `forcedStart` | `"UNSET"` | force past limits (left unset) |
| `extractAfterDownload` | `"UNSET"` | archive extraction (left unset) |

Note the `BooleanStatus` fields (`autoConfirm`, `autoStart`, `forcedStart`,
`extractAfterDownload`) use the string enum `TRUE`/`FALSE`/`UNSET`, while `enabled` is the
JSON string `"true"`.

## Decisions

These were resolved during brainstorming:

1. **Scope — shared facility.** The crawljob writer is a shared component used by all four
   modules (pawchive, pixiv, kemono, patreon), not pawchive-only. The external-URL handling is
   already duplicated across these modules, so a single shared sink avoids further duplication.
2. **Routing — only unparseable links.** The crawljob captures exactly the links watcher-go
   cannot download itself (the current "warn and drop" branch where `factory.CanParse` is
   false). Links a native module supports continue to download natively. No overlap, no double
   downloads.
3. **Lifecycle — accumulate locally, then move.** Links accumulate into a single local
   `.crawljob` file across runs (human-editable so the user can prune non-download links). A
   new command moves that file into JDownloader's folderwatch directory under a unique name;
   the local file is consumed (gone) and the next run starts a fresh one.
4. **Config — single global block.** One global `crawljob:` config section holds enable flag,
   file paths, blacklist, and auto flags. The existing per-module `external_urls` print/download
   toggles are unchanged.
5. **Granularity — one package per post.** Each post with unparseable links produces one
   package entry: its own `downloadFolder`, `packageName`, and `comment`.
6. **Blacklist — domain-suffix match, case-insensitive.** `mediafire.com` matches
   `www.mediafire.com`.
7. **Auto flags default TRUE, configurable.** `autoConfirm` and `autoStart` default to `TRUE`
   so a merge starts downloading immediately; both overridable via config.

## Architecture

```
internal/jdownloader/        NEW shared package
  crawljob.go                Crawljob storable struct + (de)serialization
  writer.go                  Writer singleton: Enabled/Blacklisted/Add, file read-modify-write
  config.go                  reads global "crawljob" config via Viper

cmd/watcher/crawljob.go      NEW `watcher crawljob merge` command

internal/modules/{pawchive,pixiv,kemono,patreon}/download.go
                             integration: route unparseable links to Writer.Add
```

### `internal/jdownloader/` package

**`Crawljob` struct** — marshals to one `.crawljob` array element. Fields per the table above.
A file is a `[]Crawljob`.

**Config** (read via Viper at top-level key `crawljob`):

```yaml
crawljob:
  enabled: true
  file: ./watcher-go.crawljob            # local accumulation file (default: ./watcher-go.crawljob)
  folderwatch_path: C:\Users\...\JDownloader\folderwatch
  blacklist: [discord.gg, patreon.com, t.me]
  auto_start: true                       # default true
  auto_confirm: true                     # default true
```

**`Writer`** — package-level singleton, `sync.Mutex`-guarded because the watcher may process
items in parallel (`Run.RunParallel`). API:

- `Enabled() bool` — true when `crawljob.enabled` is set.
- `Blacklisted(rawURL string) bool` — parses the URL host and returns true if the host equals
  or is a subdomain of any blacklist entry (case-insensitive suffix match on label boundaries).
- `Add(packageName, downloadFolder, sourceURL string, links []string) error`:
  1. Resolve `downloadFolder` to an absolute path via `filepath.Abs` (JDownloader requires
     absolute paths; watcher's download dir may be relative).
  2. Read the existing local file (empty slice if missing).
  3. Drop any link already present anywhere in the file (dedupe across all packages).
  4. If, after dedupe, no links remain, do nothing (no file write).
  5. Append exactly one new `Crawljob` entry containing the surviving links — one `Add` call
     produces one entry. (We do not merge into a same-named existing entry: step 3's dedupe
     already prevents duplicate links, and the tracked-item mechanism means a re-run won't
     re-emit an already-processed post's links.)
  6. Write the file back (pretty-printed JSON for human editability).

### Module integration

For each of the four modules:

1. **Gate change.** `getExternalLinks` currently early-returns unless
   `DownloadExternalItems || PrintExternalItems`. Add a third condition so links are still
   collected when `jdownloader.Writer.Enabled()` is true.
2. **Routing change.** In the per-post loop over external links, the branch where
   `!factory.CanParse(externalURL)` currently only logs `slog.Warn`. Change it to:
   - If `Writer.Enabled()` and not `Writer.Blacklisted(url)` → call `Writer.Add(...)` with the
     post's package name, the post's **absolute** download folder, the source post URL, and the
     link. Log an info that it was queued for JDownloader.
   - Else → keep today's `slog.Warn` (unchanged behavior when crawljob is disabled).
3. **Folder availability.** In pawchive, the per-post download folder is computed *after* the
   link loop (`download.go:327-337`). Move that computation up so it's available inside the loop
   for `downloadFolder`. Mirror the equivalent adjustment in the other three modules as needed.

Package name format: `"<module key> - <post id> - <post title>"` (title omitted if empty),
mirroring the existing post-folder naming.

### `watcher crawljob merge` command

Cobra command under `cmd/watcher/`, following the existing `cli.add*Command` pattern.

- Reads `crawljob.file` and `crawljob.folderwatch_path` (flags `--file` / `--folderwatch` may
  override).
- If the local file is missing or contains no entries → print a message and exit cleanly (no-op).
- If `folderwatch_path` is unset → error clearly.
- Otherwise **move** the local file to `<folderwatch_path>/watcher-go-<unix-ts>.crawljob`
  (unique name avoids clobbering and gives JDownloader a fresh job). After the move the local
  file no longer exists; the next scrape run starts a new one.

## Error handling

- Crawljob disabled → modules behave exactly as today (warn-and-drop).
- `folderwatch_path` not configured on merge → explicit error, non-zero exit.
- Local file missing/empty on merge → friendly no-op message, zero exit.
- Blacklisted URL → skipped (logged at debug/info).
- Relative download dir → resolved to absolute in `Add`.
- Concurrent module runs → all `Add` calls serialized via the writer mutex.
- Malformed/corrupt local file → return an error from `Add`/merge rather than silently
  overwriting; surface it so the user can inspect/edit.

## Testing

Project has working test infra (`internal/modules/pawchive/download_test.go`).

- `internal/jdownloader/writer_test.go`:
  - `Add` writes expected JSON; second `Add` with an overlapping link dedupes it.
  - `Add` resolves a relative `downloadFolder` to absolute.
  - `Add` with all-duplicate links is a no-op.
  - `Blacklisted` — suffix match (`www.mediafire.com` vs `mediafire.com`), case-insensitivity,
    non-match (`notmediafire.com` does not match `mediafire.com`).
- `cmd/watcher` (or package-level) merge test:
  - move places the file under a unique name in the target dir and removes the source.
  - empty/missing source → no-op, no error.
  - unset folderwatch path → error.
- pawchive `download_test.go` case: with crawljob enabled, an unparseable link (e.g.
  `mega.nz`) lands in the crawljob file with the correct absolute `downloadFolder`, while a
  link a native module can parse does **not**.

## Out of scope

- MyJDownloader cloud API integration.
- Changing how native modules download links they already support.
- Automatic triggering of the merge (it stays an explicit user command).
- Retry/verification of whether JDownloader actually completed downloads.
