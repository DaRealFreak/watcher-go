package animation

import (
	"bytes"
	"fmt"
	"github.com/google/uuid"
	"github.com/mitchellh/go-homedir"
	_ "golang.org/x/image/webp"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
)

type FileData struct {
	Frames    [][]byte
	MsDelays  []int
	FilePaths []string
	WorkPath  string
}

func CreateWebpAnimation(fileData *FileData) ([]byte, error) {
	if len(fileData.Frames) != len(fileData.MsDelays) {
		return nil, fmt.Errorf("delays don't match the frame count")
	}
	dumpFramesForImageMagick(fileData)

	var executable string
	var args []string
	if runtime.GOOS == "windows" {
		executable = "magick"
		args = append(args, "convert")
	} else {
		executable = "convert"
	}
	for i := 0; i <= len(fileData.Frames)-1; i++ {
		args = append(args, "-delay", strconv.Itoa(fileData.MsDelays[i]/10), fileData.FilePaths[i])
	}
	args = append(args, "-loop", "0", filepath.Join(fileData.WorkPath, "output.mkv"))

	// ToDo: fallback to imaging library of golang of error
	if err := exec.Command(executable, args...).Run(); err != nil {
		log.Fatal(err)
	}

	// convert from mkv to webp due to ImageMagick not supporting animated webp yet
	args = []string{"-y", "-i", filepath.Join(fileData.WorkPath, "output.mkv"), "-lossless", "1", "-loop", "0", filepath.Join(fileData.WorkPath, "output.webp")}
	if err := exec.Command("ffmpeg", args...).Run(); err != nil {
		log.Fatal(err)
	}

	// read file content to return it
	content, err := ioutil.ReadFile(filepath.Join(fileData.WorkPath, "output.webp"))
	if err != nil {
		log.Fatal(err)
	}

	// clean up the created folder/files
	if err := os.RemoveAll(fileData.WorkPath); err != nil {
		log.Fatal(err)
	}
	return content, nil
}

// dump all frames into a unique folder for the ImageMagick conversion
func dumpFramesForImageMagick(fData *FileData) {
	uuid4, err := uuid.NewUUID()
	if err != nil {
		log.Fatal(err)
	}

	// convert to the absolute path
	homeDir, err := homedir.Dir()
	if err != nil {
		log.Fatal(err)
	}
	fData.WorkPath = filepath.Join(homeDir, ".watcher", uuid4.String())

	// create the directory
	if err := os.MkdirAll(fData.WorkPath, os.ModePerm); err != nil {
		log.Fatal(err)
	}

	// dump frames into folder and append created file paths into the file data
	for index, frame := range fData.Frames {
		// guess the image format for the file extension
		fType, err := guessImageFormat(bytes.NewReader(frame))
		fPath := filepath.Join(fData.WorkPath, fmt.Sprintf("%d.%s", index, fType))
		if err != nil {
			log.Fatal(err)
		}
		err = ioutil.WriteFile(fPath, frame, 0644)
		if err != nil {
			log.Fatal(err)
		}
		fData.FilePaths = append(fData.FilePaths, fPath)
	}
}

// guess image format from gif/jpeg/png/webp
func guessImageFormat(r io.Reader) (format string, err error) {
	_, format, err = image.DecodeConfig(r)
	return
}
