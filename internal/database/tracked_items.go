package database

import (
	"database/sql"
	"fmt"

	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/raven"

	// import for side effects
	_ "github.com/mattn/go-sqlite3"
)

func (db *DbIO) createTrackedItemsTable(connection *sql.DB) (err error) {
	sqlStatement := `
		CREATE TABLE tracked_items
		(
			uid           INTEGER PRIMARY KEY AUTOINCREMENT,
			uri           VARCHAR(255) DEFAULT '',
			subfolder     VARCHAR(255) DEFAULT '',
			current_item  VARCHAR(255) DEFAULT '',
			module        VARCHAR(255) DEFAULT '' NOT NULL ,
			last_modified DATETIME 	   DEFAULT strftime('%s', 'now'),
			complete      BOOLEAN      DEFAULT FALSE NOT NULL
		);
	`
	_, err = connection.Exec(sqlStatement)

	return err
}

// GetTrackedItems retrieves all tracked items from the sqlite database
// if module is set limit the results use the passed module as restraint
func (db *DbIO) GetTrackedItems(module models.ModuleInterface, includeCompleted bool) []*models.TrackedItem {
	var (
		items []*models.TrackedItem
		rows  *sql.Rows
		err   error
	)

	if module == nil {
		if includeCompleted {
			rows, err = db.connection.Query("SELECT uid, uri, subfolder, current_item, module, last_modified, complete FROM tracked_items ORDER BY module, uid")
		} else {
			rows, err = db.connection.Query("SELECT uid, uri, subfolder, current_item, module, last_modified, complete FROM tracked_items WHERE NOT complete ORDER BY module, uid")
		}
	} else {
		var stmt *sql.Stmt

		if includeCompleted {
			stmt, err = db.connection.Prepare("SELECT uid, uri, subfolder, current_item, module, last_modified, complete FROM tracked_items WHERE module = ? ORDER BY uid")
		} else {
			stmt, err = db.connection.Prepare("SELECT uid, uri, subfolder, current_item, module, last_modified, complete FROM tracked_items WHERE NOT complete AND module = ? ORDER BY uid")
		}
		defer raven.CheckClosure(stmt)
		raven.CheckError(err)

		rows, err = stmt.Query(module.ModuleKey())
	}

	raven.CheckError(err)

	defer raven.CheckClosure(rows)

	for rows.Next() {
		item := models.TrackedItem{}

		err = rows.Scan(&item.ID, &item.URI, &item.SubFolder, &item.CurrentItem, &item.Module, &item.LastModified, &item.Complete)
		raven.CheckError(err)

		items = append(items, &item)
	}

	return items
}

// GetTrackedItemsByDomain retrieves all tracked items from the sqlite database based on the domain
func (db *DbIO) GetTrackedItemsByDomain(domain string, includeCompleted bool) []*models.TrackedItem {
	var (
		items []*models.TrackedItem
		rows  *sql.Rows
		err   error
	)

	var stmt *sql.Stmt

	if includeCompleted {
		stmt, err = db.connection.Prepare("SELECT uid, uri, subfolder, current_item, module, last_modified, complete FROM tracked_items WHERE uri LIKE ? ORDER BY uid")
	} else {
		stmt, err = db.connection.Prepare("SELECT uid, uri, subfolder, current_item, module, last_modified, complete FROM tracked_items WHERE NOT complete AND uri LIKE ? ORDER BY uid")
	}
	defer raven.CheckClosure(stmt)
	raven.CheckError(err)

	rows, err = stmt.Query(fmt.Sprintf("%%%s%%", domain))
	raven.CheckError(err)

	defer raven.CheckClosure(rows)

	for rows.Next() {
		item := models.TrackedItem{}

		err = rows.Scan(&item.ID, &item.URI, &item.SubFolder, &item.CurrentItem, &item.Module, &item.LastModified, &item.Complete)
		raven.CheckError(err)

		items = append(items, &item)
	}

	return items
}

// GetFirstOrCreateTrackedItem checks if an item exists already, else creates it
// returns the already persisted or the newly created item
func (db *DbIO) GetFirstOrCreateTrackedItem(uri string, subFolder string, module models.ModuleInterface) *models.TrackedItem {
	stmt, err := db.connection.Prepare("SELECT uid, uri, subfolder, current_item, module, last_modified, complete FROM tracked_items WHERE uri = ? and subfolder = ? and module = ?")
	defer raven.CheckClosure(stmt)
	raven.CheckError(err)

	rows, QueryErr := stmt.Query(uri, subFolder, module.ModuleKey())
	raven.CheckError(QueryErr)

	defer raven.CheckClosure(rows)

	item := models.TrackedItem{}

	if rows.Next() {
		// item already persisted
		err = rows.Scan(&item.ID, &item.URI, &item.SubFolder, &item.CurrentItem, &item.Module, &item.LastModified, &item.Complete)
		raven.CheckError(err)
	} else {
		// create the item and call the same function again
		db.CreateTrackedItem(uri, subFolder, module)
		item = *db.GetFirstOrCreateTrackedItem(uri, subFolder, module)
	}

	return &item
}

// CreateTrackedItem inserts the passed uri and the module into the tracked_items table
func (db *DbIO) CreateTrackedItem(uri string, subFolder string, module models.ModuleInterface) {
	stmt, err := db.connection.Prepare("INSERT INTO tracked_items (uri, subfolder, module) VALUES (?, ?, ?)")
	raven.CheckError(err)

	defer raven.CheckClosure(stmt)

	_, err = stmt.Exec(uri, subFolder, module.ModuleKey())
	raven.CheckError(err)
}

// UpdateTrackedItem updates the current item column of the tracked item in the database
// also sets the complete status to false to check it on the next check cycle
func (db *DbIO) UpdateTrackedItem(trackedItem *models.TrackedItem, currentItem string) {
	stmt, err := db.connection.Prepare("UPDATE tracked_items SET current_item = ?, last_modified = strftime('%s', 'now'), complete = ? WHERE uid = ?")
	raven.CheckError(err)

	defer raven.CheckClosure(stmt)

	_, err = stmt.Exec(currentItem, 0, trackedItem.ID)
	raven.CheckError(err)

	// update current item
	trackedItem.CurrentItem = currentItem
}

// ChangeTrackedItemCompleteStatus changes the complete status of the passed tracked item in the database
func (db *DbIO) ChangeTrackedItemCompleteStatus(trackedItem *models.TrackedItem, complete bool) {
	var completeInt int8
	if complete {
		completeInt = 1
	} else {
		completeInt = 0
	}

	stmt, err := db.connection.Prepare("UPDATE tracked_items SET last_modified = strftime('%s', 'now'), complete = ? WHERE uid = ?")
	raven.CheckError(err)

	defer raven.CheckClosure(stmt)

	_, err = stmt.Exec(completeInt, trackedItem.ID)
	raven.CheckError(err)

	trackedItem.Complete = complete
}

func (db *DbIO) ChangeTrackedItemUri(trackedItem *models.TrackedItem, uri string) {
	stmt, err := db.connection.Prepare("UPDATE tracked_items SET uri = ? WHERE uid = ?")
	raven.CheckError(err)

	defer raven.CheckClosure(stmt)

	_, err = stmt.Exec(uri, trackedItem.ID)
	raven.CheckError(err)

	trackedItem.URI = uri
}

func (db *DbIO) ChangeTrackedItemSubFolder(trackedItem *models.TrackedItem, subFolder string) {
	stmt, err := db.connection.Prepare("UPDATE tracked_items SET subfolder = ? WHERE uid = ?")
	raven.CheckError(err)

	defer raven.CheckClosure(stmt)

	_, err = stmt.Exec(subFolder, trackedItem.ID)
	raven.CheckError(err)

	trackedItem.SubFolder = subFolder
}
