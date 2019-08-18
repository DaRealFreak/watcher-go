package animation

import (
	"bytes"
	"fmt"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	_ "golang.org/x/image/webp"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

type FileData struct {
	Frames          [][]byte
	MsDelays        []int
	FilePaths       []string
	WorkPath        string
	ConvertedFrames bool
}

type Helper struct {
	outputDirectory string
	outputFileName  string
}

// retrieve animation helper with standard settings
func NewAnimationHelper() *Helper {
	return &Helper{
		outputFileName:  "output",
		outputDirectory: os.TempDir(),
	}
}

// internal function to create mkv video from the passed frames with the option
// to not remove the temporary folder for further conversions from mkv to another format
// (useful for webp/fliff animated image formats which are not directly supported by ImageMagick yet)
func (h *Helper) createAnimationImageMagick(fileData *FileData, fileExtension string, deleteAfterConversion bool) (content []byte, err error) {
	if len(fileData.Frames) != len(fileData.MsDelays) {
		return nil, fmt.Errorf("delays don't match the frame count")
	}

	if len(fileData.FilePaths) == 0 {
		if err := h.dumpFramesForImageMagick(fileData); err != nil {
			return nil, err
		}
	}

	executable, args := h.getImageMagickEnv()
	for i := 0; i <= len(fileData.Frames)-1; i++ {
		args = append(args, "-delay", strconv.Itoa(fileData.MsDelays[i]/10), fileData.FilePaths[i])
	}
	args = append(args, "-loop", "0", filepath.Join(fileData.WorkPath, h.outputFileName+"."+fileExtension))

	log.Debugf("running command: %s %s", executable, strings.Join(args, " "))
	if err := exec.Command(executable, args...).Run(); err != nil {
		if fileData.ConvertedFrames {
			// fallback failed, return the error
			return nil, err
		} else {
			// force convert frames from the source format with ImageMagick to PNG
			// since FFmpeg had problems to convert some image formats into video formats
			return h.imageFormatFallback(fileData, fileExtension, deleteAfterConversion)
		}
	}

	// read file content to return it
	content, err = ioutil.ReadFile(filepath.Join(fileData.WorkPath, h.outputFileName+"."+fileExtension))
	if err != nil {
		return nil, err
	}

	// option to keep converted mkv for further conversions
	if deleteAfterConversion {
		// clean up the created folder/files
		err = os.RemoveAll(fileData.WorkPath)
	}
	return content, err
}

// dump all frames into a unique folder for the ImageMagick conversion
func (h *Helper) dumpFramesForImageMagick(fData *FileData) (err error) {
	uuid4, err := uuid.NewUUID()
	if err != nil {
		return err
	}

	// create custom folder in the temporary directory
	fData.WorkPath = filepath.Join(h.outputDirectory, uuid4.String())

	// create the directory
	if err := os.MkdirAll(fData.WorkPath, os.ModePerm); err != nil {
		return err
	}

	// reset file paths to allow multiple conversions of one FileData struct
	fData.FilePaths = []string{}
	// dump frames into folder and append created file paths into the file data
	for index, frame := range fData.Frames {
		// guess the image format for the file extension
		fType, err := h.guessImageFormat(bytes.NewReader(frame))
		fPath := filepath.Join(fData.WorkPath, fmt.Sprintf("%d.%s", index, fType))
		if err != nil {
			return err
		}
		err = ioutil.WriteFile(fPath, frame, 0644)
		if err != nil {
			return err
		}
		fData.FilePaths = append(fData.FilePaths, fPath)
	}
	return nil
}

// guess image format from gif/jpeg/png/webp
func (h *Helper) guessImageFormat(r io.Reader) (format string, err error) {
	_, format, err = image.DecodeConfig(r)
	return
}

// FFmpeg has sometimes problems to convert images to videos from different image formats
// so convert frames to PNG with ImageMagick
func (h *Helper) imageFormatFallback(fData *FileData, fileExtension string, deleteAfterConversion bool) ([]byte, error) {
	log.Debug("using image format fallback to PNG")
	for i := 0; i <= len(fData.Frames)-1; i++ {
		executable, args := h.getImageMagickEnv()
		newFilePath := filepath.Join(fData.WorkPath, strconv.Itoa(i)+".png")
		args = append(args, fData.FilePaths[i], newFilePath)
		fData.FilePaths[i] = newFilePath
		log.Debugf("running command: %s %s", executable, strings.Join(args, " "))
		if err := exec.Command(executable, args...).Run(); err != nil {
			return nil, err
		}
	}
	fData.ConvertedFrames = true
	return h.createAnimationImageMagick(fData, fileExtension, deleteAfterConversion)
}

func (h *Helper) getImageMagickEnv() (executable string, args []string) {
	if runtime.GOOS == "windows" {
		executable = "magick"
		args = append(args, "convert")
	} else {
		executable = "convert"
	}
	return
}
