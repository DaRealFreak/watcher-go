package pixivapi

import (
	"encoding/json"
	"io"
	"net/http"
)

type PixivErrorHandler struct {
}

func (e PixivErrorHandler) CheckResponse(response *http.Response) (err error, fatal bool) {
	switch {
	case response.StatusCode == 400:
		var mobileAPIError MobileAPIError

		if content, readErr := io.ReadAll(response.Body); readErr == nil {
			if err = json.Unmarshal(content, &mobileAPIError); err == nil {
				if mobileAPIError.ErrorDetails.Message == `{"offset":["offset must be no more than 5000"]}` {
					return OffsetError{
						APIError: APIError{
							ErrorMessage: mobileAPIError.ErrorDetails.Message,
						},
					}, true
				}
			}
		}
	}

	return nil, false
}

func (e PixivErrorHandler) CheckDownloadedFileForErrors(writtenSize int64, responseHeader http.Header) error {
	return nil
}
