package update

import (
	"fmt"
	"github.com/DaRealFreak/watcher-go/pkg/version"
	"github.com/blang/semver"
	"github.com/rhysd/go-github-selfupdate/selfupdate"
	log "github.com/sirupsen/logrus"
	"os"
)

type updateChecker struct {
}

// returns a new update checker
func NewUpdateChecker() *updateChecker {
	return &updateChecker{}
}

// check if any new releases exist and ask if we should update the binary
func (u *updateChecker) CheckForAvailableUpdates() (err error) {
	latest, found, err := selfupdate.DetectLatest(version.RepositoryUrl)
	if err != nil {
		log.Warning("error occurred while detecting version: ", err)
		return err
	}

	v := semver.MustParse(version.VERSION)
	if !found || latest.Version.LTE(v) {
		log.Trace("current version is the latest")
		return nil
	}

	fmt.Printf("a new version (%s) is available, do you want to update? (y/n)\n", latest.Version.String())
	// user doesn't want to update
	if !u.askYesNoAnswer() {
		return nil
	}

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("could not locate executable path")
	}
	if err := selfupdate.UpdateTo(latest.AssetURL, exe); err != nil {
		return err
	}
	log.Info("Successfully updated to version " + latest.Version.String())
	return nil
}

func (u *updateChecker) askYesNoAnswer() bool {
	var response string
	_, err := fmt.Scanln(&response)
	if err != nil {
		log.Fatal(err)
	}
	if response != "y" && response != "n" {
		fmt.Println("Invalid input, please write y or n")
		return u.askYesNoAnswer()
	} else {
		return response == "y"
	}
}
