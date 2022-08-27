package session

import (
	"fmt"
	"net/http"
	"strconv"
)

type StatusError struct {
	error
	StatusCode int
}

func (e StatusError) Error() string {
	return fmt.Sprintf("unexpected status code: %d", e.StatusCode)
}

type WrittenSizeError struct {
	error
	Message string
}

func (e WrittenSizeError) Error() string {
	return e.Message
}

type Errorhandler interface {
	CheckResponse(response *http.Response) error
	CheckDownloadedFileForErrors(writtenSize int64, responseHeader http.Header) (err error)
}

type DefaultErrorHandler struct {
}

func (e DefaultErrorHandler) CheckResponse(response *http.Response) error {
	switch {
	case response.StatusCode < 400:
		return nil
	default:
		return StatusError{
			StatusCode: response.StatusCode,
		}
	}
}

func (e DefaultErrorHandler) CheckDownloadedFileForErrors(writtenSize int64, responseHeader http.Header) error {
	if val, ok := responseHeader["Content-Length"]; ok {
		fileSize, err := strconv.Atoi(val[0])
		if err == nil {
			if writtenSize != int64(fileSize) {
				return WrittenSizeError{
					Message: fmt.Sprintf(
						"written file size doesn't match the header content length value (%d != %d)",
						writtenSize, fileSize,
					),
				}
			}
		}
	}

	if writtenSize <= 0 {
		return WrittenSizeError{
			Message: "written content has a size of 0 bytes",
		}
	}

	return nil
}
