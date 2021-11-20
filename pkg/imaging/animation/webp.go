package animation

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/DaRealFreak/watcher-go/internal/raven"
	log "github.com/sirupsen/logrus"
)

// FileFormatWebp is the file extension for the WEBP format
const FileFormatWebp = ".webp"

// CreateAnimationWebp tries create a .webp animated picture from the passed file data
func (h *Helper) CreateAnimationWebp(fData *FileData) (content []byte, err error) {
	// create mkv video from the file data first
	// since ImageMagick doesn't support animated webp pictures
	if _, err = h.createAnimationImageMagick(fData, "mkv", false); err != nil {
		return nil, err
	}

	// convert from mkv to webp using ffmpeg since ImageMagick does not support animated webp yet
	args := []string{
		"-y",
		"-loglevel", "0",
		"-i", filepath.Join(fData.WorkPath, h.outputFileName+".mkv"),
		"-lossless", "1",
		"-loop", "0",
		filepath.Join(fData.WorkPath, h.outputFileName+".webp"),
	}
	log.Debugf("running command: ffmpeg %s", strings.Join(args, " "))

	// #nosec
	if err = exec.Command("ffmpeg", args...).Run(); err != nil {
		return nil, err
	}

	// read file content to return it
	content, err = ioutil.ReadFile(filepath.Join(fData.WorkPath, h.outputFileName+".webp"))
	if err != nil {
		return
	}

	raven.CheckError(os.Chdir(fData.PreviousPath))

	// clean up the created folder/files
	defer raven.CheckPathRemoval(fData.WorkPath)

	return content, err
}
