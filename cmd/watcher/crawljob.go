package watcher

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/DaRealFreak/watcher-go/internal/jdownloader"
	"github.com/DaRealFreak/watcher-go/internal/raven"
	"github.com/spf13/cobra"
)

// addCrawljobCommand adds the `crawljob` command group for the JDownloader handoff.
func (cli *CliApplication) addCrawljobCommand() {
	crawljobCmd := &cobra.Command{
		Use:   "crawljob",
		Short: "manage the JDownloader .crawljob handoff file",
	}

	var fileOverride, folderwatchOverride string
	mergeCmd := &cobra.Command{
		Use:   "merge",
		Short: "move the accumulated .crawljob file into JDownloader's folderwatch directory",
		Long: "moves the local .crawljob file (built up from external links watcher-go can't\n" +
			"download itself) into JDownloader's Folder Watch directory so JDownloader picks\n" +
			"it up and downloads each file into its post folder. The local file is consumed.",
		Run: func(_ *cobra.Command, _ []string) {
			cfg := jdownloader.LoadConfig()
			if fileOverride != "" {
				cfg.File = fileOverride
			}
			if folderwatchOverride != "" {
				cfg.FolderwatchPath = folderwatchOverride
			}

			movedTo, err := jdownloader.NewWriter(cfg).Merge(time.Now().Unix())
			raven.CheckError(err)

			if movedTo == "" {
				slog.Info("no crawljob entries to merge")
				return
			}
			slog.Info(fmt.Sprintf("merged crawljob into %s", movedTo))
		},
	}
	mergeCmd.Flags().StringVar(&fileOverride, "file", "", "override the local crawljob file path")
	mergeCmd.Flags().StringVar(&folderwatchOverride, "folderwatch", "", "override JDownloader's folderwatch directory")

	crawljobCmd.AddCommand(mergeCmd)
	cli.rootCmd.AddCommand(crawljobCmd)
}
