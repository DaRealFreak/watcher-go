package database

import (
	"bufio"
	"io/ioutil"
	"log"
	"os"
	"testing"

	"github.com/DaRealFreak/watcher-go/pkg/modules"
	"github.com/DaRealFreak/watcher-go/pkg/raven"
	"github.com/spf13/viper"
)

// nolint: gochecknoglobals
var dbIO *DbIO

// TestMain is the constructor for the test functions
// creates temporary file to use as database file to prevent previous data influencing the tests
// and remove the database at the end again for a clean system
func TestMain(m *testing.M) {
	f, err := ioutil.TempFile("", "*.db")
	if err != nil {
		log.Fatal("couldn't create temporary database file for unit tests", err)
	}

	// close the file, set the database path and remove the temporary file
	_ = f.Close()
	viper.Set("Database.Path", f.Name())
	RemoveDatabase()

	// initialize the database
	dbIO = NewConnection()
	factory := modules.NewModuleFactory(dbIO)
	test := factory.GetAllModules()[0]
	dbIO.GetFirstOrCreateAccount("test_user", "test_pass", test)
	dbIO.GetFirstOrCreateTrackedItem(test.Key(), test)

	// run the unit tests
	code := m.Run()

	// destructor, close database connection and remove the file
	dbIO.CloseConnection()
	RemoveDatabase()
	os.Exit(code)
}

// test the add account function
func TestDumpTables(t *testing.T) {
	f := bufio.NewWriter(os.Stdout)
	raven.CheckError(dbIO.DumpTables(f, "accounts", "tracked_items"))
	raven.CheckError(f.Flush())
}
