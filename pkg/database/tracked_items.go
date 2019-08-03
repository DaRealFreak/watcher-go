package database

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"watcher-go/pkg/models"
)

// retrieve all tracked items from the sqlite database
// if module is set limit the results use the passed module as restraint
func (db DbIO) GetTrackedItems(module models.ModuleInterface) []*models.TrackedItem {
	var items []*models.TrackedItem

	var rows *sql.Rows
	var err error
	if module == nil {
		rows, err = db.connection.Query("SELECT * FROM tracked_items WHERE NOT complete ORDER BY module, uid")
		db.checkErr(err)
	} else {
		stmt, err := db.connection.Prepare("SELECT * FROM tracked_items WHERE NOT complete AND module = ? ORDER BY uid")
		db.checkErr(err)

		rows, err = stmt.Query(module.Key())
		db.checkErr(err)
	}
	defer rows.Close()

	for rows.Next() {
		item := models.TrackedItem{}
		err = rows.Scan(&item.Id, &item.Uri, &item.CurrentItem, &item.Module, &item.Complete)
		db.checkErr(err)

		items = append(items, &item)
	}

	return items
}

// check if an item exists already, if not create it
// returns the already persisted or the newly created item
func (db DbIO) GetFirstOrCreateTrackedItem(uri string, module models.ModuleInterface) *models.TrackedItem {
	stmt, err := db.connection.Prepare("SELECT * FROM tracked_items WHERE uri = ? and module = ?")
	db.checkErr(err)

	rows, err := stmt.Query(uri, module.Key())
	db.checkErr(err)
	defer rows.Close()

	item := models.TrackedItem{}
	if rows.Next() {
		// item already persisted
		err = rows.Scan(&item.Id, &item.Uri, &item.CurrentItem, &item.Module, &item.Complete)
		db.checkErr(err)
	} else {
		// create the item and call the same function again
		db.CreateTrackedItem(uri, module)
		item = *db.GetFirstOrCreateTrackedItem(uri, module)
	}
	return &item
}

// inserts the passed uri and the module into the tracked_items table
func (db DbIO) CreateTrackedItem(uri string, module models.ModuleInterface) {
	stmt, err := db.connection.Prepare("INSERT INTO tracked_items (uri, module) VALUES (?, ?)")
	db.checkErr(err)
	defer stmt.Close()

	_, err = stmt.Exec(uri, module.Key())
	db.checkErr(err)
}

// update the current item column of the tracked item in the database
func (db DbIO) UpdateTrackedItem(trackedItem *models.TrackedItem, currentItem string) {
	stmt, err := db.connection.Prepare("UPDATE tracked_items SET current_item = ? WHERE uid = ?")
	db.checkErr(err)
	defer stmt.Close()

	_, err = stmt.Exec(currentItem, trackedItem.Id)
	db.checkErr(err)

	// update current item
	trackedItem.CurrentItem = currentItem
}

// change the complete status of the passed tracked item
func (db DbIO) ChangeTrackedItemCompleteStatus(trackedItem *models.TrackedItem, complete bool) {
	var completeInt int8
	if complete {
		completeInt = 1
	} else {
		completeInt = 0
	}
	stmt, err := db.connection.Prepare("UPDATE tracked_items SET complete = ? WHERE uid = ?")
	db.checkErr(err)
	defer stmt.Close()

	_, err = stmt.Exec(completeInt, trackedItem.Id)
	db.checkErr(err)

	trackedItem.Complete = complete
}
