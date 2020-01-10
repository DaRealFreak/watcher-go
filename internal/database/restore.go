package database

import (
	"os/exec"

	"github.com/spf13/viper"
)

// RestoreTableFromFile executes the passed SQL file, used primarily for restoration
// #nosec
func (db *DbIO) RestoreTableFromFile(fileName string) error {
	return exec.Command("sqlite3", viper.GetString("Database.Path"), ".read '"+fileName+"'").Run()
}
