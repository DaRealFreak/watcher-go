package animation

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
)

// create webp animated picture from the passed FileData
// ToDo: fallback to imaging library of golang of error
func (h *Helper) CreateAnimationWebp(fileData *FileData) (content []byte, err error) {
	// create mkv video from the file data first
	// since ImageMagick doesn't support animated webp pictures
	if _, err := h.createAnimationImageMagick(fileData, "mkv", false); err != nil {
		return nil, err
	}

	// convert from mkv to webp using ffmpeg since ImageMagick does not support animated webp yet
	args := []string{
		"-y",
		"-loglevel", "0",
		"-i", filepath.Join(fileData.WorkPath, h.outputFileName+".mkv"),
		"-lossless", "1",
		"-loop", "0",
		filepath.Join(fileData.WorkPath, h.outputFileName+".webp"),
	}
	log.Debugf("running command: ffmpeg %s", strings.Join(args, " "))
	if err := exec.Command("ffmpeg", args...).Run(); err != nil {
		return nil, err
	}

	// read file content to return it
	content, err = ioutil.ReadFile(filepath.Join(fileData.WorkPath, h.outputFileName+".webp"))
	if err != nil {
		return
	}

	// clean up the created folder/files
	err = os.RemoveAll(fileData.WorkPath)
	return
}
