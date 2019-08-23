# Watcher-go
[![Go Report Card](https://goreportcard.com/badge/github.com/DaRealFreak/watcher-go)](https://goreportcard.com/report/github.com/DaRealFreak/watcher-go)  ![GitHub](https://img.shields.io/github/license/DaRealFreak/watcher-go)

Application to keep track of items from multiple sources with a local database. 
It will download any detected item and update the index in the database on completion of the download.

## Dependencies
For image to animation conversions (currently only used by the pixiv module) this application is using
[ImageMagick](https://imagemagick.org/) and [FFmpeg](https://ffmpeg.org/).  
The executed command on Windows for ImageMagick is `magick.exe convert`,
which is the default for a [chocolately](https://chocolatey.org/) installation.  
For all other operating systems the commands `ffmpeg` and `convert` are executed.  
As fallback if ImageMagick and FFmpeg are not available the golang imaging libraries are used.
The libraries can only generate a GIF file with 256 colors, so it is not recommended.


## Usage
There are currently 5 available root commands with following functionality:
```  
Available Commands:
  add                   add an item or account to the database
  list                  lists items or accounts from the database
  run                   update all tracked items
  update                update the application or an item/account in the database
  generate-autocomplete generates auto completion for Bash, Zsh and PowerShell
```

### Global Flags
These flags are available for all commands and sub commands:
```
Flags:
      --config             config file (default is ./.watcher.yaml)
  -v, --verbosity          log level (debug, info, warn, error, fatal, panic (default "info")
      --version            version for watcher
      --disable-sentry     disable sentry and don't send usage statistics/errors to the developer
      --enable-sentry      use sentry to send usage statistics/errors to the developer
      --log-force-colors   enforces colored output even for non-tty terminals
      --log-force-format   enforces formatted output even for non-tty terminals
```

The sentry is disabled by default and has to be enabled before errors will be sent to the sentry server.
You can do so by adding `--enable-sentry` to any command.
This setting will be active until `--disable-sentry` is being added to another execution.

### Running the application
Just running the command `watcher run` will run the main functionality of the application:  
checking all active sources and downloading updates  

Options for this command are:
```
Flags:
  -d, --directory string   download directory (will be saved in config file)
  -u, --url string         url of module you want to run
```

You can specify the download directory, which is getting saved in a configuration file,
so you don't have to pass the argument every time.  
You can also specify which module to run by passing an example url to retrieve the module from.

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
  -u, --username string   username you want to add (required)
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

No flags are required for to enable/disable items.

### Enabling Auto Completion
Auto Completion can be generated for the terminals bash, powershell and zsh.
Simply run `watcher generate-autocomplete` with the following sub commands
to generate a script in `~/.watcher/completion` and printing you the command to active it.

```
Available Commands:
  bash        generates auto completion for Bash
  powershell  generates auto completion for PowerShell
  zsh         generates auto completion for Zsh
```


## Development
Want to contribute? Great!  
I'm always glad hearing about bugs or pull requests.

## License
This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details
