package update

import (
	"fmt"
	"github.com/DaRealFreak/watcher-go/pkg/version"
	"github.com/blang/semver"
	"github.com/rhysd/go-github-selfupdate/selfupdate"
	log "github.com/sirupsen/logrus"
	"github.com/tcnksm/go-gitconfig"
	"os"
)

type updateChecker struct {
}

// returns a new update checker
func NewUpdateChecker() *updateChecker {
	return &updateChecker{}
}

// check if any new releases exist
func (u *updateChecker) CheckForAvailableUpdates() (updateAvailable bool, err error) {
	latest, found, err := selfupdate.DetectLatest(version.RepositoryUrl)
	if err != nil {
		log.Warning("error occurred while detecting version: ", err)
		return false, err
	}

	v := semver.MustParse(version.VERSION)
	if !found || latest.Version.LTE(v) {
		return false, nil
	}
	return true, nil
}

// update the application
func (u *updateChecker) UpdateApplication() (err error) {
	updateAvailable, err := u.CheckForAvailableUpdates()
	if !updateAvailable {
		fmt.Println("current version is the latest")
		return nil
	}

	// check for github token in environment or in git config if not set in environment
	// required for updates when repository is private
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		token, _ = gitconfig.GithubToken()
	}

	up, err := selfupdate.NewUpdater(selfupdate.Config{
		APIToken: token,
	})
	if err != nil {
		return err
	}

	fmt.Println("new version detected, updating...")
	// retrieve latest asset url again
	latest, _, err := up.DetectLatest(version.RepositoryUrl)
	if err != nil {
		log.Warning("error occurred while retrieving latest asset URLs: ", err)
		return err
	}

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("could not locate executable path")
	}
	if err := up.UpdateTo(latest, exe); err != nil {
		return err
	}
	log.Info("successfully updated to version " + latest.Version.String())
	return nil
}
