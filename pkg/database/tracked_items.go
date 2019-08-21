package database

import (
	"database/sql"

	"github.com/DaRealFreak/watcher-go/pkg/models"
	"github.com/DaRealFreak/watcher-go/pkg/raven"

	// import for side effects
	_ "github.com/mattn/go-sqlite3"
)

// GetTrackedItems retrieves all tracked items from the sqlite database
// if module is set limit the results use the passed module as restraint
func (db DbIO) GetTrackedItems(module models.ModuleInterface, includeCompleted bool) []*models.TrackedItem {
	var items []*models.TrackedItem
	var rows *sql.Rows
	var err error

	if module == nil {
		if includeCompleted {
			rows, err = db.connection.Query("SELECT * FROM tracked_items ORDER BY module, uid")
		} else {
			rows, err = db.connection.Query("SELECT * FROM tracked_items WHERE NOT complete ORDER BY module, uid")
		}
	} else {
		var stmt *sql.Stmt
		if includeCompleted {
			stmt, err = db.connection.Prepare("SELECT * FROM tracked_items WHERE module = ? ORDER BY uid")
		} else {
			stmt, err = db.connection.Prepare("SELECT * FROM tracked_items WHERE NOT complete AND module = ? ORDER BY uid")
		}
		raven.CheckError(err)
		rows, err = stmt.Query(module.Key())
	}
	raven.CheckError(err)
	defer raven.CheckRowClosure(rows)

	for rows.Next() {
		item := models.TrackedItem{}
		err = rows.Scan(&item.ID, &item.URI, &item.CurrentItem, &item.Module, &item.Complete)
		raven.CheckError(err)

		items = append(items, &item)
	}

	return items
}

// GetFirstOrCreateTrackedItem checks if an item exists already, else creates it
// returns the already persisted or the newly created item
func (db DbIO) GetFirstOrCreateTrackedItem(uri string, module models.ModuleInterface) *models.TrackedItem {
	stmt, err := db.connection.Prepare("SELECT * FROM tracked_items WHERE uri = ? and module = ?")
	raven.CheckError(err)

	rows, err := stmt.Query(uri, module.Key())
	raven.CheckError(err)
	defer raven.CheckRowClosure(rows)

	item := models.TrackedItem{}
	if rows.Next() {
		// item already persisted
		err = rows.Scan(&item.ID, &item.URI, &item.CurrentItem, &item.Module, &item.Complete)
		raven.CheckError(err)
	} else {
		// create the item and call the same function again
		db.CreateTrackedItem(uri, module)
		item = *db.GetFirstOrCreateTrackedItem(uri, module)
	}
	return &item
}

// CreateTrackedItem inserts the passed uri and the module into the tracked_items table
func (db DbIO) CreateTrackedItem(uri string, module models.ModuleInterface) {
	stmt, err := db.connection.Prepare("INSERT INTO tracked_items (uri, module) VALUES (?, ?)")
	raven.CheckError(err)
	defer raven.CheckStatementClosure(stmt)

	_, err = stmt.Exec(uri, module.Key())
	raven.CheckError(err)
}

// UpdateTrackedItem updates the current item column of the tracked item in the database
// also sets the complete status to false to check it on the next check cycle
func (db DbIO) UpdateTrackedItem(trackedItem *models.TrackedItem, currentItem string) {
	stmt, err := db.connection.Prepare("UPDATE tracked_items SET current_item = ?, complete = ? WHERE uid = ?")
	raven.CheckError(err)
	defer raven.CheckStatementClosure(stmt)

	_, err = stmt.Exec(currentItem, 0, trackedItem.ID)
	raven.CheckError(err)

	// update current item
	trackedItem.CurrentItem = currentItem
}

// ChangeTrackedItemCompleteStatus changes the complete status of the passed tracked item in the database
func (db DbIO) ChangeTrackedItemCompleteStatus(trackedItem *models.TrackedItem, complete bool) {
	var completeInt int8
	if complete {
		completeInt = 1
	} else {
		completeInt = 0
	}
	stmt, err := db.connection.Prepare("UPDATE tracked_items SET complete = ? WHERE uid = ?")
	raven.CheckError(err)
	defer raven.CheckStatementClosure(stmt)

	_, err = stmt.Exec(completeInt, trackedItem.ID)
	raven.CheckError(err)

	trackedItem.Complete = complete
}
