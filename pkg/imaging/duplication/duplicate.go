package duplication

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
)

// CheckForSimilarity uses image magick to check for image similarity
func CheckForSimilarity(file1 string, file2 string) (similarity float32, err error) {
	f1, err := getFileResourceReader(file1)
	if err != nil {
		return 0, err
	}

	f2, err := getFileResourceReader(file2)
	if err != nil {
		return 0, err
	}

	tmpFile1, err := copyToTempFile(f1)
	if err != nil {
		return 0, err
	}

	tmpFile2, err := copyToTempFile(f2)
	if err != nil {
		return 0, err
	}

	fmt.Println(tmpFile1.Name(), tmpFile2.Name())
	return 0, nil
}

// getFileResourceReader returns a reader of the file resource of either passed URL or local path
func getFileResourceReader(source string) (r io.ReadCloser, err error) {
	u, err := url.Parse(source)
	if err != nil {
		return nil, err
	}

	if u.Scheme == "http" || u.Scheme == "https" {
		resp, err := http.Get(source)
		if err != nil {
			return nil, err
		}

		return resp.Body, nil
	}

	return os.Open(source)
}

// copyToTempFile copies content of the read closer to a temporary file
func copyToTempFile(r io.ReadCloser) (f *os.File, err error) {
	f, err = ioutil.TempFile("", ".*")
	if err != nil {
		return nil, err
	}

	_, err = io.Copy(f, r)
	if err != nil {
		return nil, err
	}

	return f, nil
}
