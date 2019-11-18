# Watcher-go
[![Go Report Card](https://goreportcard.com/badge/github.com/DaRealFreak/watcher-go)](https://goreportcard.com/report/github.com/DaRealFreak/watcher-go)  ![GitHub](https://img.shields.io/github/license/DaRealFreak/watcher-go) ![build](https://github.com/DaRealFreak/watcher-go/workflows/build/badge.svg?branch=master) ![tests](https://github.com/DaRealFreak/watcher-go/workflows/tests/badge.svg?branch=master)

Application to keep track of items from multiple sources with a local database. 
It will download any detected item and update the index in the database on completion of the download.

## Dependencies
Optional dependencies (some functionality may not be accessible without):
- [ImageMagick](https://imagemagick.org/) - for image to animation conversions (currently only used by the pixiv module).  
Windows will try to execute `magick.exe convert`(default for a [chocolately](https://chocolatey.org/) installation), other operating systems `convert`
- [FFmpeg](https://ffmpeg.org/) - for video to animation conversion (currently only used by the pixiv module)
- [SQLite3](https://www.sqlite.org/index.html) - for restoring data from SQL files in backup archives

If ImageMagick and FFmpeg are not available a fallback is implemented to generate .gif animations using the golang imaging libraries.
These libraries can only generate a GIF file with 256 colors, so it is not recommended.

## Usage
These root commands are currently available with following functionality:
```  
Available Commands:
  add                   add an item or account to the database
  backup                generates a backup of the current settings and database file
  generate-autocomplete generates auto completion for Bash, Zsh and PowerShell
  help                  Help about any command
  list                  lists items or accounts from the database
  module                lists the module specific commands and settings
  restore               restores the current settings/database from the passed backup archive
  run                   update all tracked items or directly passed items
  update                update the application or an item/account in the database
```

### Global Flags
These flags are available for all commands and sub commands:
```
Flags:
      --config                      config file (default is ./.watcher.yaml)
      --database string             database file (default is ./watcher.db)
  -v, --verbosity                   log level (debug, info, warn, error, fatal, panic (default "info")
      --version                     version for watcher
      --disable-sentry              disable sentry and don't send usage statistics/errors to the developer
      --enable-sentry               use sentry to send usage statistics/errors to the developer
      --log-disable-colors          disables colors even for tty terminals
      --log-force-colors            enforces colored output even for non-tty terminals
      --log-disable-timestamp       removes the time info of the log entries, useful if output is logged with a timestamp already
      --log-timestamp-passed-time   uses the passed time since the program is running in seconds instead of a formatted time
      --log-level-uppercase         transforms the log levels into upper case
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
  -u, --url strings        url of module you want to run
  -x, --disable strings    url of module you want don't want to run
  -p, --parallel           run modules parallel
```

You can specify the download directory, which is getting saved in a configuration file,
so you don't have to pass the argument every time.  
You can also specify which module to run by passing an example url to retrieve the module from.  
Modules can be run parallel to each other with the `--parallel` flag, causing each module
to run independently from each other, ignoring possible rate limits from other modules.  
It is also possible to run only specific modules by attaching the repeated flag `--url`.  
In case you want to disable modules from being run you can attach the repeated flag `--disable`.

### Adding Accounts/Items
Accounts and tracked items be added by attaching to the add command (f.e. `watcher add item`)
```
Available Commands:
  account     adds an account to the database
  item        adds an item to the database
```

Following flags are available for the `watcher add account` command
```
Flags:
  -h, --help              help for account
  -p, --password string   password of the user (required)
      --url string        url for the association of the account (required)
  -u, --user string       username you want to add (required)
```

Items can be added by executing following command:  
`watcher add item [url1] [url2] [url3] ...`

### List Accounts/Items/Modules 
To see what accounts, items and modules are available you can add following sub commands to the list command
```
Available Commands:
  all         displays modules, accounts and items in the database
  accounts    displays all accounts
  items       displays all items
  modules     shows all registered modules
``` 

watcher list items got the extra flag `--include-completed` if you also want to display completed items into the list.

### Updating Application/Accounts/Items
The update sub command will check for available updates of the application and download it.
In case you want to update an account or tracked item you can add the following sub commands:
```
Available Commands:
  -           updates the application
  account     updates the saved account
  item        updates the saved current item
```

accounts got the following flags:
```
Available Commands:
  disable     disable an account based on the username
  enable      enables an account based on the username

Flags:
  -p, --password string   new password (required)
      --url string        url of module (required)
  -u, --user string       username (required)
```

items got the following flags:
```
  -c, --current string   current item in case you don't want to download older items
      --url string       url of tracked item you want to update (required)
```

### Enabling/Disabling Accounts/Items
You can also enable/disable accounts and items individually with the update sub command.  
To enable accounts run `watcher update account enable`, to disable accounts `watcher update account disable`.  

Accounts need the following flags:
```
Flags:
      --url string    url of module (required)
  -u, --user string   username (required)
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
You can backup the database and the settings with the `backup [archive]` command.  
It is also possible to further specify more precisely what you want to backup using these sub commands.
```
Available Commands:
  accounts    generates a backup of the current accounts
  items       generates a backup of the current items
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
  items       restores the tracked_items table from the passed archive
  settings    restores the settings file from the passed archive
```

The binary backup from the database will be preferred over the .sql files in the archive (only used in full restore)
in case that the archive got manually modified (the backup command can either backup the binary file or .sql files,
not both). Neither the binary file nor the .sql files are further checked for invalid/corrupted data.

### Module Settings
Each module can bring custom commands and settings.
You can list all modules with custom commands/settings using `watcher module --help`.  
Due to the modular structure with later planned external module support I'd recommend checking the commands/setting yourself.

## Development
Want to contribute? Great!  
I'm always glad hearing about bugs or pull requests.

## License
This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details
