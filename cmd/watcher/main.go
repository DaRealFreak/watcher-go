package main

import (
	"fmt"
	"github.com/kubernetes/klog"
	"github.com/spf13/cobra"
	"log"
	"os"
	"watcher-go/cmd/watcher/arguments"
	"watcher-go/cmd/watcher/database"
	"watcher-go/cmd/watcher/models"
	"watcher-go/cmd/watcher/modules"
)

type watcher struct {
	dbCon         *database.DbIO
	moduleFactory *modules.ModuleFactory
}

func init() {
	klog.InitFlags(nil)
}

func main() {
	watcher := NewWatcher()

	var rootCmd = &cobra.Command{
		Use:   "app",
		Short: "Watcher keeps track of all media items you want to track.",
		Long: "An application written in Go to keep track of items from multiple sources.\n" +
			"On every downloaded media file the current index will get updated so you'll never miss a tracked item",
	}
	rootCmd.PersistentFlags().StringP("author", "a", "DaRealFreak", "Author name for copyright attribution")

	// runs the main functionality to update all tracked items
	runCmd := &cobra.Command{
		Use:   "run",
		Short: "update all tracked items",
		Run: func(cmd *cobra.Command, args []string) {
			for _, item := range watcher.dbCon.GetTrackedItems(nil) {
				module := watcher.moduleFactory.GetModule(item.Module)
				if !module.IsLoggedIn() {
					watcher.loginToModule(module)
				}
				klog.Info(fmt.Sprintf("parsing item %s (current id: %s)", item.Uri, item.CurrentItem))
				module.Parse(item)
			}
		},
	}
	runCmd.Flags().StringVarP(&arguments.DownloadDirectory, "directory", "d", "./", "Download Directory (required)")

	// general add option
	addCmd := &cobra.Command{
		Use:   "add",
		Short: "add an item or account to the database",
		Long:  "option for the user to add accounts/items to the database for the main usage",
	}

	var url string
	var current string
	// add the item option, requires only the uri
	itemCmd := &cobra.Command{
		Use:   "item",
		Short: "adds an item to the database",
		Long:  "parses and adds the passed url into the tracked items if not already tracked",
		Run: func(cmd *cobra.Command, args []string) {
			watcher.AddItemByUri(url, current)
		},
	}
	itemCmd.Flags().StringVarP(&url, "url", "", "", "url of item you want to track (required)")
	itemCmd.Flags().StringVarP(&current, "current", "", "", "current item in case you don't want to download older items")
	_ = itemCmd.MarkFlagRequired("url")

	var username string
	var password string
	// add the account option, requires username, password and uri
	accountCmd := &cobra.Command{
		Use:   "account",
		Short: "adds an account to the database",
		Long:  "checks the passed url to assign the passed account/password to a module and save it to the database",
		Run: func(cmd *cobra.Command, args []string) {
			watcher.AddAccountByUri(url, username, password)
		},
	}
	accountCmd.Flags().StringVarP(&username, "username", "u", "", "username you want to add (required)")
	accountCmd.Flags().StringVarP(&password, "password", "p", "", "password of the user (required)")
	accountCmd.Flags().StringVarP(&url, "url", "", "", "url for the association of the account (required)")
	_ = accountCmd.MarkFlagRequired("username")
	_ = accountCmd.MarkFlagRequired("password")
	_ = accountCmd.MarkFlagRequired("url")

	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(addCmd)
	addCmd.AddCommand(itemCmd, accountCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	watcher.dbCon.CloseConnection()
}

func NewWatcher() *watcher {
	dbIO := database.NewConnection()
	watcher := watcher{
		dbCon:         dbIO,
		moduleFactory: modules.NewModuleFactory(dbIO),
	}
	return &watcher
}

// extract the module based on the uri and add account if not registered already
func (app watcher) AddAccountByUri(uri string, user string, password string) {
	module, err := app.moduleFactory.GetModuleFromUri(uri)
	app.checkError(err)

	app.dbCon.GetFirstOrCreateAccount(user, password, module)
}

// add item based on the uri and set it to the passed current item if not nil
func (app watcher) AddItemByUri(uri string, currentItem string) {
	module, err := app.moduleFactory.GetModuleFromUri(uri)
	app.checkError(err)

	trackedItem := app.dbCon.GetFirstOrCreateTrackedItem(uri, module)
	if currentItem != "" {
		app.dbCon.UpdateTrackedItem(trackedItem, currentItem)
	}
}

// login into the module
func (app watcher) loginToModule(module *models.Module) {
	klog.Info(fmt.Sprintf("logging in for module %s", module.Key()))
	account := app.dbCon.GetAccount(module)

	// no account available but module requires a login
	if account == nil && module.RequiresLogin() {
		log.Fatal(fmt.Sprintf("Module \"%s\" requires a login, but no account could be found", module.Key()))
	}

	// login into the module
	success := module.Login(account)
	if success {
		klog.Info("login successful")
	} else {
		if module.RequiresLogin() {
			log.Fatal(fmt.Sprintf("Module \"%s\" requires a login, but the login failed", module.Key()))
		} else {
			klog.Warning("login not successful")
		}
	}

}

func (app watcher) checkError(err error) {
	if err != nil {
		panic(err)
	}
}
