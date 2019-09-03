package formatter

import (
	"testing"
	"time"

	"github.com/mattn/go-colorable"
	"github.com/sirupsen/logrus"
)

func TestFormat(t *testing.T) {
	lvl, _ := logrus.ParseLevel("debug")
	logrus.SetLevel(lvl)
	// set custom text formatter for the logger
	logrus.StandardLogger().Formatter = &Formatter{
		DisableColors:            false,
		ForceColors:              false,
		DisableTimestamp:         false,
		UseUppercaseLevel:        false,
		UseTimePassedAsTimestamp: false,
		TimestampFormat:          time.StampMilli,
		PadAllLogEntries:         true,
	}
	logrus.SetOutput(colorable.NewColorableStdout())
	logrus.Debug("test")
	logrus.Info("test")
	logrus.WithField("module", "sankakucomplex.com").Info("test")
	logrus.WithField("module", "deviantart.com").Info("test")
	logrus.WithField("module", "pixiv.com").Info("test")
	logrus.WithField("module", "e-hentai.org").Info("test")
	logrus.WithFields(logrus.Fields{"module": "e-hentai.org", "test": "123"}).Info("test multi module")
	logrus.Warn("test")
	logrus.Error("test")
}
