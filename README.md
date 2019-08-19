# Watcher-go
[![Go Report Card](https://goreportcard.com/badge/github.com/DaRealFreak/watcher-go)](https://goreportcard.com/report/github.com/DaRealFreak/watcher-go?style=flat-square)  ![GitHub](https://img.shields.io/github/license/DaRealFreak/watcher-go?style=flat-square)

Application to keep track of items from multiple sources with a local database. 
It will download any detected item and update the index in the database on completion of the download.


## Usage
There are currently 5 available root commands with following functionality:
```  
Available Commands:
  add         add an item or account to the database
  list        lists items or accounts from the database
  run         update all tracked items
  update      update the application or an item/account in the database
```

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
  all         displays accounts and items in the database
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
  account     updates the saved current item
  item        updates the saved current item
```

accounts got the following flags (password is not required on enable/disable sub command):
```
Available Commands:
  disable     disable an account based on the username
  enable      enables an account based on the username

Flags:
  -h, --help              help for account
  -p, --password string   new password (required)
      --url string        url of module (required)
  -u, --user string       username (required)
```

items got the following flags:
```
  -c, --current string   current item in case you don't want to download older items
      --url string       url of tracked item you want to update (required)
```

enabling/disabling items will come in later versions


## Development
Want to contribute? Great!  
I'm always glad hearing about bugs or pull requests.

## License
This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details
