package database

import (
	"github.com/spf13/viper"
	"os/exec"
)

// RestoreTableFromFile executes the passed SQL file, used primarily for restoration
func (db *DbIO) RestoreTableFromFile(fileName string) error {
	return exec.Command("sqlite3", viper.GetString("Database.Path"), ".read '"+fileName+"'").Run()
}
