package animation

import (
	"bytes"
	"fmt"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"

	"image"

	// import for registering bmp format to image decoder
	_ "golang.org/x/image/bmp"

	// import for registering webp format to image decoder
	_ "golang.org/x/image/webp"

	// import for registering gif format to image decoder
	_ "image/gif"
	// import for registering jpeg format to image decoder
	_ "image/jpeg"
	// import for registering png format to image decoder
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

// FileData is the object to create and handle possible steps during the animation creation process
// f.e. in case frames have to get converted to a different file format
type FileData struct {
	Frames          [][]byte
	MsDelays        []int
	FilePaths       []string
	WorkPath        string
	ConvertedFrames bool
}

// Helper contains the output settings and encapsulates the animation creation functions
type Helper struct {
	outputDirectory string
	outputFileName  string
}

// NewAnimationHelper returns a Helper struct with the default settings
func NewAnimationHelper() *Helper {
	return &Helper{
		outputFileName:  "output",
		outputDirectory: os.TempDir(),
	}
}

// createAnimationImageMagick is an internal function to create an mkv video from the passed frames with the option
// to not remove the temporary folder for further conversions from mkv to another format
// This option is useful for webp/fliff animated image formats which are not directly supported by ImageMagick yet
func (h *Helper) createAnimationImageMagick(fData *FileData, fExt string, del bool) (content []byte, err error) {
	if len(fData.Frames) != len(fData.MsDelays) {
		return nil, fmt.Errorf("delays don't match the frame count")
	}

	if len(fData.FilePaths) == 0 {
		if err := h.dumpFramesForImageMagick(fData); err != nil {
			return nil, err
		}
	}

	executable, args := h.getImageMagickEnv()
	for i := 0; i <= len(fData.Frames)-1; i++ {
		args = append(args,
			"-delay",
			// don't ask me about the conversion rate, was the result of trying to approach
			// the best length on multiple long ugoira works
			fmt.Sprintf("%0.2f", float64(fData.MsDelays[i])/13),
			fData.FilePaths[i],
		)
	}
	args = append(args, "-loop", "0", filepath.Join(fData.WorkPath, h.outputFileName+"."+fExt))

	log.Debugf("running command: %s %s", executable, strings.Join(args, " "))
	if err := exec.Command(executable, args...).Run(); err != nil {
		if fData.ConvertedFrames {
			// fallback failed, return the error
			return nil, err
		}
		// force convert frames from the source format with ImageMagick to PNG
		// since FFmpeg had problems to convert some image formats into video formats
		return h.imageFormatFallback(fData, fExt, del)
	}

	// read file content to return it
	content, err = ioutil.ReadFile(filepath.Join(fData.WorkPath, h.outputFileName+"."+fExt))
	if err != nil {
		return nil, err
	}

	// option to keep converted mkv for further conversions
	if del {
		// clean up the created folder/files
		err = os.RemoveAll(fData.WorkPath)
	}
	return content, err
}

// dumpFramesForImageMagick dumps all frames into a unique folder for the ImageMagick conversion
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

// guessImageFormat returns the guessed image format from the registered image encodings
func (h *Helper) guessImageFormat(r io.Reader) (format string, err error) {
	_, format, err = image.DecodeConfig(r)
	return
}

// imageFormatFallback implements a fallback method for FFmpeg, since it sometimes has problems
// to convert images to videos from different image formats.
// So we convert frames to PNG with ImageMagick and try to create the video again
func (h *Helper) imageFormatFallback(fData *FileData, fExt string, del bool) ([]byte, error) {
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
	return h.createAnimationImageMagick(fData, fExt, del)
}

// getImageMagickEnv retrieves the executable path and possible arguments for ImageMagick
// based on the OS it is running on
func (h *Helper) getImageMagickEnv() (executable string, args []string) {
	if runtime.GOOS == "windows" {
		executable = "magick"
		args = append(args, "convert")
	} else {
		executable = "convert"
	}
	return
}
