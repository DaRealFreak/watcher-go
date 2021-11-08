package database

import (
	"database/sql"
	"fmt"
	"io"
	"strings"

	"github.com/DaRealFreak/watcher-go/internal/raven"
)

// tableSchema contains the useful columns of the sqlite_master table
type tableSchema struct {
	Name string
	Type string
	SQL  string
}

// DumpTables dumps the database tableSchema and the inserts to the passed writer
func (db *DbIO) DumpTables(writer io.Writer, tableNames ...string) (err error) {
	if _, err = writer.Write([]byte("BEGIN TRANSACTION;\n")); err != nil {
		return err
	}

	tableSchemas, err := db.getSchemas(tableNames...)
	if err != nil {
		return err
	}

	for _, currentTableSchema := range tableSchemas {
		if _, err = writer.Write([]byte("DROP TABLE IF EXISTS " + currentTableSchema.Name + ";\n")); err != nil {
			return err
		}

		if _, err = writer.Write([]byte(currentTableSchema.SQL + ";\n")); err != nil {
			return err
		}

		var inserts []string
		inserts, err = db.getTableRows(currentTableSchema.Name)
		if err != nil {
			return err
		}

		for _, insert := range inserts {
			if _, err = writer.Write([]byte(insert + "\n")); err != nil {
				return err
			}
		}
	}

	_, err = writer.Write([]byte("COMMIT;\n"))

	return err
}

// getTableRows returns the insert queries for the passed table
func (db *DbIO) getTableRows(tableName string) (inserts []string, err error) {
	// first get the column names
	columnNames, err := db.getPragmaTableInfo(tableName)
	if err != nil {
		return nil, err
	}

	// sqlite_master table contains the SQL CREATE statements for the database.
	columnSelects := make([]string, len(columnNames))
	for i, c := range columnNames {
		columnSelects[i] = fmt.Sprintf(`'||quote("%s")||'`, strings.Replace(c, `"`, `""`, -1))
	}

	// create insert queries with the pragma table info, so we can't use static queries here
	// #nosec
	q := fmt.Sprintf(`
		SELECT 'INSERT INTO "%s" VALUES(%s);' FROM "%s";
	`,
		tableName,
		strings.Join(columnSelects, ","),
		tableName,
	)

	stmt, err := db.connection.Prepare(q)
	if err != nil {
		return nil, err
	}

	defer raven.CheckClosure(stmt)

	rows, err := stmt.Query()
	if err != nil {
		return nil, err
	}

	defer raven.CheckClosure(rows)

	inserts = []string{}

	for rows.Next() {
		var insert string

		err = rows.Scan(&insert)
		if err != nil {
			return nil, err
		}

		inserts = append(inserts, insert)
	}

	err = rows.Err()

	return inserts, err
}

// getPragmaTableInfo returns the table_info PRAGMA query for the passed table
func (db *DbIO) getPragmaTableInfo(tableName string) (columnNames []string, err error) {
	// sqlite_master table contains the SQL CREATE statements for the database.
	q := `PRAGMA table_info("` + tableName + `")`

	stmt, err := db.connection.Prepare(q)
	if err != nil {
		return nil, err
	}

	defer raven.CheckClosure(stmt)

	rows, err := stmt.Query()
	if err != nil {
		return nil, err
	}

	defer raven.CheckClosure(rows)

	columnNames = []string{}

	for rows.Next() {
		var arr []interface{}

		for i := 0; i < 6; i++ {
			arr = append(arr, new(interface{}))
		}

		err = rows.Scan(arr...)
		if err != nil {
			return nil, err
		}

		columnNames = append(columnNames,
			func() (result string) {
				// check the type
				switch (*arr[1].(*interface{})).(type) {
				case string:
					result = (*arr[1].(*interface{})).(string)
				case []uint8:
					result = string((*arr[1].(*interface{})).([]uint8))
				}
				return
			}(),
		)
	}

	err = rows.Err()

	return columnNames, err
}

// getSchemas returns the available schemas, optional parameter names to specify f.e. tables
func (db *DbIO) getSchemas(names ...string) (schemas []*tableSchema, err error) {
	tableNames := make([]interface{}, len(names))
	for i, v := range names {
		tableNames[i] = v
	}

	query := `SELECT "name", "type", "sql"
			  FROM "sqlite_master"
			  WHERE "sql" NOT NULL
			  %s
			  ORDER BY "name"`
	if len(names) > 0 {
		query = fmt.Sprintf(query,
			"AND name IN ("+
				strings.TrimSuffix(
					strings.Repeat("?,", len(names)),
					",",
				)+
				")",
		)
	} else {
		query = fmt.Sprintf(query, "")
	}

	stmt, err := db.connection.Prepare(query)
	if err != nil {
		return nil, err
	}

	defer raven.CheckClosure(stmt)

	var rows *sql.Rows

	if len(names) > 0 {
		rows, err = stmt.Query(tableNames...)
	} else {
		rows, err = stmt.Query()
	}

	if err != nil {
		return nil, err
	}

	defer raven.CheckClosure(rows)

	schemas = []*tableSchema{}

	for rows.Next() {
		s := &tableSchema{}

		err = rows.Scan(&s.Name, &s.Type, &s.SQL)
		if err != nil {
			return nil, err
		}

		schemas = append(schemas, s)
	}

	err = rows.Err()

	return schemas, err
}
