// Package duplication uses ImageMagick to check for similarity in pictures after calculating them down for performance
package duplication

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/DaRealFreak/watcher-go/internal/raven"
	"github.com/DaRealFreak/watcher-go/pkg/imaging"
	log "github.com/sirupsen/logrus"
)

// CheckForSimilarity uses image magick to check for image similarity
func CheckForSimilarity(file1 string, file2 string) (similarity float64, err error) {
	f1, err := getFileResourceReader(file1)
	defer raven.CheckClosureNonFatal(f1)

	if err != nil {
		return 0, err
	}

	f2, err := getFileResourceReader(file2)
	defer raven.CheckClosureNonFatal(f2)

	if err != nil {
		return 0, err
	}

	tmpFile1, err := copyToTempFile(f1)
	defer raven.CheckFileRemoval(tmpFile1)
	if err != nil {
		return 0, err
	}

	tmpFile2, err := copyToTempFile(f2)
	defer raven.CheckFileRemoval(tmpFile2)
	if err != nil {
		return 0, err
	}

	if err = resizeImage(tmpFile1.Name(), 400, 400); err != nil {
		return 0, err
	}

	if err = resizeImage(tmpFile2.Name(), 400, 400); err != nil {
		return 0, err
	}

	sim, err := getSimilarity(tmpFile1.Name(), tmpFile2.Name())

	return sim, err
}

// getSimilarity returns the similarity of the passed files in numerical percentage, the higher, the more similar
func getSimilarity(file1 string, file2 string) (similarity float64, err error) {
	executable, args := imaging.GetImageMagickEnv("compare")
	args = append(args, "-metric", "mse", file1, file2, "NULL:")
	log.Debugf("running command: %s %s", executable, strings.Join(args, " "))

	// ImageMagick compare returns 0 on similar images, 1 on dissimilar images, 2 on error according to the man page
	// since we want to return similarity we have to handle exit code 1 too which would be handled as error in go
	// #nosec
	stdout, stderr, err := executeCommand(exec.Command(executable, args...))

	// the buffer won't be nil even on a returned error
	//noinspection GoNilness
	out := stdout.String() + stderr.String()
	positiveMatchPattern := regexp.MustCompile(`[\d.]+ \(([\d.]+)\)`)
	matches := positiveMatchPattern.FindStringSubmatch(out)

	if len(matches) == 2 {
		similarityResult := matches[1]
		res, _ := strconv.ParseFloat(similarityResult, 64)

		return 1 - res, nil
	}

	combinedMsg := fmt.Sprint(err) + ": " + stderr.String()
	return 0, errors.New(combinedMsg)
}

// resizeImage uses ImageMagick to resize the passed file to the requested width x height ignoring the aspect ratio
func resizeImage(fileName string, width int, height int) error {
	executable, args := imaging.GetImageMagickEnv("convert")
	args = append(args, fileName, "-resize", fmt.Sprintf("%dx%d!", width, height), fileName)
	log.Debugf("running command: %s %s", executable, strings.Join(args, " "))

	cmd := exec.Command(executable, args...)
	err := cmd.Start()
	if err != nil {
		return err
	}

	return cmd.Wait()
}

// getFileResourceReader returns a reader of the file resource of either passed URL or local path
func getFileResourceReader(source string) (r *os.File, err error) {
	u, err := url.Parse(source)
	if err != nil {
		return nil, err
	}

	if u.Scheme == "http" || u.Scheme == "https" {
		var f *os.File
		f, err = os.CreateTemp("", ".*")
		if err != nil {
			return nil, err
		}

		var resp *http.Response
		resp, err = http.Get(source)
		if err != nil {
			return nil, err
		}

		if _, err = io.Copy(f, resp.Body); err != nil {
			return nil, err
		}

		return f, nil
	}

	return os.Open(source)
}

// copyToTempFile copies content of the read closer to a temporary file
func copyToTempFile(r io.Reader) (f *os.File, err error) {
	f, err = os.CreateTemp("", ".*")
	if err != nil {
		return nil, err
	}

	_, err = io.Copy(f, r)
	if err != nil {
		return nil, err
	}

	return f, nil
}

// executeCommand the command and returns the output/error
func executeCommand(cmd *exec.Cmd) (stdout bytes.Buffer, stderr bytes.Buffer, err error) {
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Start()
	if err != nil {
		return stdout, stderr, err
	}

	err = cmd.Wait()

	return stdout, stderr, err
}
