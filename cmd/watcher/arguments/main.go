package arguments

import "flag"

var DownloadDirectory *string
var Uri *string
var Account *string
var Password *string
var CurrentItem *string

func init() {
	// used for download/default function
	DownloadDirectory = flag.String("directory", "./", "directory to download the media files into")
	// used for adding accounts/items
	Uri = flag.String("uri", "", "uri for the tracked item or account")
	// used for adding accounts
	Account = flag.String("account", "", "account to be added")
	Password = flag.String("password", "", "password for the added account")
	// used for adding items
	CurrentItem = flag.String("currentItem", "", "current item for the tracked item")
	flag.Parse()
}
