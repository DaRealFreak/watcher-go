package watcher

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/DaRealFreak/watcher-go/pkg/models"
)

// ListRegisteredModules lists all registered modules
func (app *Watcher) ListRegisteredModules() {
	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 8, 1, '\t', 0)
	_, _ = fmt.Fprintln(w, "ID\tModule Key\tRequires Login")

	for index, module := range app.ModuleFactory.GetAllModules() {
		_, _ = fmt.Fprintf(
			w,
			"%d\t%s\t%t\n",
			index,
			module.Key,
			module.RequiresLogin,
		)
	}

	_ = w.Flush()
}

// ListTrackedItems lists all tracked items with the option to limit it to a module
func (app *Watcher) ListTrackedItems(uri string, includeCompleted bool) {
	var trackedItems []*models.TrackedItem
	if uri == "" {
		trackedItems = app.DbCon.GetTrackedItems(nil, includeCompleted)
	} else {
		module := app.ModuleFactory.GetModuleFromURI(uri)
		trackedItems = app.DbCon.GetTrackedItems(module, includeCompleted)
	}

	// initialize tab writer
	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 8, 1, '\t', 0)
	_, _ = fmt.Fprintln(w, "ID\tModule\tUrl\tCurrent Item\tCompleted")

	for _, item := range trackedItems {
		_, _ = fmt.Fprintf(
			w,
			"%d\t%s\t%s\t%s\t%t\n",
			item.ID,
			item.Module,
			item.URI,
			item.CurrentItem,
			item.Complete,
		)
	}

	_ = w.Flush()
}

// ListAccounts lists all accounts with the option to limit it to a module
func (app *Watcher) ListAccounts(uri string) {
	var accounts []*models.Account
	if uri == "" {
		accounts = app.DbCon.GetAllAccounts(nil)
	} else {
		module := app.ModuleFactory.GetModuleFromURI(uri)
		accounts = app.DbCon.GetAllAccounts(module)
	}

	// initialize tab writer
	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 8, 1, '\t', 0)
	_, _ = fmt.Fprintln(w, "ID\tUsername\tPassword\tModule\tDisabled")

	for _, account := range accounts {
		_, _ = fmt.Fprintf(
			w,
			"%d\t%s\t%s\t%s\t%t\n",
			account.ID,
			account.Username,
			account.Password,
			account.Module,
			account.Disabled,
		)
	}

	_ = w.Flush()
}
