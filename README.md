# Watcher-go

[![build](https://github.com/DaRealFreak/watcher-go/actions/workflows/build.yml/badge.svg)](https://github.com/DaRealFreak/watcher-go/actions/workflows/build.yml)
[![tests](https://github.com/DaRealFreak/watcher-go/actions/workflows/tests.yml/badge.svg)](https://github.com/DaRealFreak/watcher-go/actions/workflows/tests.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/DaRealFreak/watcher-go)](https://goreportcard.com/report/github.com/DaRealFreak/watcher-go) 
![GitHub](https://img.shields.io/github/license/DaRealFreak/watcher-go)

Application to keep track of items from multiple sources with a local database.
It will download any detected item and update the index in the database on completion of the download.

## Dependencies

Optional dependencies (some functionality may not be accessible without):

- [ImageMagick](https://imagemagick.org/) - for image to animation conversions (currently only used by the pixiv module)
  Windows will try to execute `magick.exe convert`(default for a [chocolately](https://chocolatey.org/) installation),
  other operating systems `convert`
- [FFmpeg](https://ffmpeg.org/) - for video to animation conversion (currently only used by the pixiv module)
- [SQLite3](https://www.sqlite.org/index.html) - for restoring data from SQL files in backup archives

If ImageMagick and FFmpeg are not available a fallback is implemented to generate .gif animations using the golang
imaging libraries.
These libraries can only generate a GIF file with 256 colors, so it is not recommended.

## Usage

These root commands are currently available with following functionality:

```  
Available Commands:
  add                   add an item, account, OAuth2 client or cookie to the database
  backup                generates a backup of the current settings and database file
  completion            Generate the autocompletion script for the specified shell
  generate-autocomplete generates auto completion for Bash, Zsh and PowerShell
  help                  Help about any command
  list                  lists items or accounts from the database
  module                lists the module specific commands and settings
  restore               restores the current settings/database from the passed backup archive
  run                   update all tracked items or directly passed items
  update                update the application or an item/account/OAuth2 client/cookie in the database
```

### Global Flags

These flags are available for all commands and sub commands:

```
Global Flags:
      --config string               config file (default is ./.watcher.yaml)
      --database string             database file (default is ./watcher.db)
      --disable-sentry              disable sentry and don't send usage statistics/errors to the developer
      --enable-sentry               use sentry to send usage statistics/errors to the developer
      --log-disable-colors          disables colors even for tty terminals
      --log-disable-timestamp       removes the time info of the log entries, useful if output is logged with a timestamp already
      --log-force-colors            enforces colored output even for non-tty terminals
      --log-level-uppercase         transforms the log levels into upper case
      --log-timestamp-passed-time   uses the passed time since the program is running in seconds instead of a formatted time
  -v, --verbosity string            log level (debug, info, warn, error, fatal, panic (default "info")
```

The sentry is disabled by default and has to be enabled before errors will be sent to the sentry server.
You can do so by adding `--enable-sentry` to any command.
This setting will be active until `--disable-sentry` is being added to another execution.

### Running the application

Just running the command `watcher run` will run the main functionality of the application:  
checking all active sources and downloading updates.  
Running the application only for specific items is possible by running:  
`watcher run [flags] [url1] [url2] [url3] ...`

Flags for the `run` command are:

```
Flags:
  -d, --directory string   download directory (will be saved in config file)
  -x, --disable strings    url of module you want don't want to run
  -f, --force              forces to ignore previous progress
  -p, --parallel           run modules parallel
  -u, --url strings        url of module you want to run
```

You can specify the download directory, which is getting saved in a configuration file,
so you don't have to pass the argument every time.  
You can also specify which module to run by passing an example url to retrieve the module from.  
Modules can be run parallel to each other with the `--parallel` flag, causing each module
to run independently of each other, ignoring possible rate limits from other modules.  
It is also possible to run only specific modules by attaching the repeated flag `--url`.  
In case you want to disable modules from being run you can attach the repeated flag `--disable`.

### Adding Accounts/Items/OAuth2 Clients/Cookies

Accounts, tracked items, OAuth2 clients and cookies can be added by attaching to the add command (
f.e. `watcher add item`)

```
Available Commands:
  account     adds an account to the database
  cookie      adds a cookie to the database
  item        adds an item to the database
  oauth       adds an OAuth2 client to the database
```

Following flags are available for the `watcher add account` command

```
Flags:
  -P, --password string   password of the user (required)
  -u, --url string        url for the association of the account (required)
  -U, --user string       username you want to add (required)

```

Items can be added by executing following command:  
`watcher add item [url1] [url2] [url3] ...`

OAuth2 clients have the following flags and require either a client ID for the normal authentication or an access token
for a static token source:

```
Flags:
      --access-token string    OAuth2 access token
      --client-id string       OAuth2 client ID
      --client-secret string   OAuth2 client secret
      --refresh-token string   OAuth2 refresh token
  -u, --url string             url for the association of the OAuth2 client (required)
```

### List Accounts/OAuth2 Clients/Cookies/Items/Modules

To see what accounts, items, OAuth2 clients, cookies and modules are available you can add following sub commands to the
list command

```
Available Commands:
  accounts    displays all accounts
  all         displays modules, accounts and items in the database
  cookies     displays all cookies
  items       displays all items
  modules     shows all registered modules
  oauth       displays all OAuth2 clients

Flags:
  -u, --url string   url of module
``` 

You can attach a `--url` flag to specify the module of all sub commands in the list category.  
Watcher list items got the extra flag `--include-completed` if you also want to display completed items into the list.

### Updating Application/Accounts/Items/OAuth2 Clients/Cookies

The update sub command will check for available updates of the application and download it.
In case you want to update an account or tracked item you can add the following sub commands:

```
Available Commands:
  -           updates the application
  account     updates the saved account
  cookie      updates the cookie for a new value and expiration date
  item        updates the saved current item
  oauth       updates the saved OAuth2 client
```

accounts got the following flags:

```
Available Commands:
  disable     disable an account based on the username
  enable      enables an account based on the username

Flags:
  -P, --password string   new password (required)
  -u, --url string        url of module (required)
  -U, --user string       username (required)
```

items got the following flags:

```
Flags:
  -c, --current string     current item in case you don't want to download older items
  -f, --subfolder string   subfolder path for additional grouping
  -u, --url string         url of tracked item you want to update (required)
```

oauth got the following flags:

```
Available Commands:
  disable     disable an OAuth2 client based on the client ID or access token
  enable      enables an OAuth2 client based on the client ID or access token

Flags:
      --access-token string    OAuth2 access token
      --client-id string       OAuth2 client ID
      --client-secret string   OAuth2 client secret
      --refresh-token string   OAuth2 refresh token
  -u, --url string             url of module (required)
```

cookies got the following flags:

```
Available Commands:
  disable     disables the cookie matching to the passed module and name
  enable      enables the cookie matching to the passed module and name

Flags:
  -e, --expiration string   cookie expiration
  -N, --name string         cookie name (required)
  -u, --url string          url for the association of the cookie (required)
  -V, --value string        cookie value (required)
```

### Enabling/Disabling Accounts/Items/OAuth2 Clients/Cookies

You can also enable/disable accounts, OAuth2 clients, cookies and items individually with the update sub command.  
To enable accounts run `watcher update account enable`, to disable accounts `watcher update account disable`.

Accounts need the following flags:

```
Flags:
  -u, --url string    url of module (required)
  -U, --user string   username (required)
```

OAuth2 clients requires the following flags (either client ID or Access Token has to be passed to the function):

```
Flags:
      --access-token string   OAuth2 access token
      --client-id string      OAuth2 client ID
  -u, --url string            url of module (required)
```

Cookies require the following flags:

```
Flags:
  -N, --name string   cookie name (required)
  -u, --url string    url of module (required)
```

Similar to the accounts is the command for enabling items:  
`watcher update item enable [url1] [url2] [url3] ...`  
and the command for disabling items:  
`watcher update item disable [url1] [url2] [url3] ...`

No flags are required for enabling/disabling items.

### Enabling Auto Completion

Auto Completion can be generated for the terminals bash, powershell and zsh.
Simply run `watcher generate-autocomplete` with the following sub commands
to generate a script in `~/.watcher/completion` and printing you the command to activate it.

```
Available Commands:
  bash        generates auto completion for Bash
  powershell  generates auto completion for PowerShell
  zsh         generates auto completion for Zsh
```

### Backup Database/Settings

You can back up the database and the settings with the `backup [archive]` command.  
It is also possible to further specify more precisely what you want to back up using these sub commands.

```
Available Commands:
  accounts    generates a backup of the current accounts
  cookies     generates a backup of the current cookies
  items       generates a backup of the current items
  oauth       generates a backup of the current OAuth2 clients
  settings    generates a backup of the current settings
```

There are currently gzip, tar and zip archive formats supported which can be specified with the command flags

```
Flags:
      --gzip   use a gzip(.tar.gz) archive
      --tar    use a tar(.tar) archive
      --zip    use a zip(.zip) archive
      --sql    generate a .sql file
```

The `--sql` flags does not exist for the `backup settings` sub command, but for every other sub command.

### Restore Database/Settings

The generated archives from the `watcher backup` command can be directly used to restore the database/settings.  
As with the backup command it is possible to further specify what you want to restore using these flags:

```
Available Commands:
  accounts    restores the accounts table from the passed archive
  cookies     restores the cookies table from the passed archive
  items       restores the tracked_items table from the passed archive
  oauth       restores the oauth_clients table from the passed archive
  settings    restores the settings file from the passed archive
```

The binary backup from the database will be preferred over the .sql files in the archive (only used in full restore)
in case that the archive got manually modified (the backup command can either back up the binary file or .sql files,
not both). Neither the binary file nor the .sql files are further checked for invalid/corrupted data.

### Module Settings

Each module can bring custom commands and settings.
You can list all modules with custom commands/settings using `watcher module --help`.  
Due to the modular structure with later planned external module support I'd recommend checking the commands/setting
yourself.

## Development

Want to contribute? Great!  
I'm always glad hearing about bugs or pull requests.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details
