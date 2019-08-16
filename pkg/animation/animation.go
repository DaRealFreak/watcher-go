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
	"path/filepath"
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

	args := []string{"convert"}
	for i := 0; i <= len(fileData.Frames)-1; i++ {
		args = append(args, fmt.Sprintf("-delay %d %s", fileData.MsDelays[i]/10, fileData.FilePaths[i]))
	}
	args = append(args, "-loop 0", filepath.Join(fileData.WorkPath, "output.mkv"))

	// windows:
	// magick.exe convert -delay X image1 -delay Y image2 -delay Z image3 -loop 0 output.mkv
	// darwin/linux
	// convert -delay X image1 -delay Y image2 -delay Z image3 -loop 0 output.mkv
	// ToDo: implement ImageMagick conversion

	args = []string{"-y", "-i " + filepath.Join(fileData.WorkPath, "output.mkv"), "-lossless 1", "-loop 0", filepath.Join(fileData.WorkPath, "output.webp")}
	// ToDo: ffmpeg conversion

	// ToDo: read created webp and return it

	// clean up the created folder/files
	err := os.RemoveAll(fileData.WorkPath)
	if err != nil {
		log.Fatal(err)
	}
	return nil, nil
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
