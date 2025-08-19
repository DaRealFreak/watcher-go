package kemono

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/DaRealFreak/watcher-go/internal/raven"
	http "github.com/bogdanfinn/fhttp"
	log "github.com/sirupsen/logrus"
)

func (m *kemono) getTotalSize(url string) (int64, error) {
	// first try a HEAD request for Content-Length
	req, _ := http.NewRequest("HEAD", url, nil)
	resp, err := m.Session.GetClient().Do(req)
	if err != nil {
		return 0, err
	}

	if resp.StatusCode >= 400 {
		return 0, fmt.Errorf("bad status code: %d", resp.StatusCode)
	}

	if cl := resp.Header.Get("Content-Length"); cl != "" {
		return strconv.ParseInt(cl, 10, 64)
	}

	// fallback: HEAD with a tiny range request
	req.Header.Set("Range", "bytes=0-0")
	resp2, err := m.Session.GetClient().Do(req)
	if err != nil {
		return 0, err
	}

	if resp2.StatusCode >= 400 {
		return 0, fmt.Errorf("bad status code: %d", resp.StatusCode)
	}

	if cr := resp2.Header.Get("Content-Range"); cr != "" {
		// the format is "bytes 0-0/12345"
		parts := strings.SplitN(cr, "/", 2)
		if len(parts) == 2 {
			return strconv.ParseInt(parts[1], 10, 64)
		}
	}

	return 0, fmt.Errorf("unable to determine total file size")
}

func (m *kemono) downloadChunks(url, outFile string, chunkSize int64, retries int, delay time.Duration) error {
	total, err := m.getTotalSize(url)
	if err != nil {
		return err
	}
	log.WithField("module", m.Key).Debugf("total size: %d bytes", total)

	// remove an existing file if present
	if _, err = os.Stat(outFile); err == nil {
		if err = os.Remove(outFile); err != nil {
			return err
		}
	}

	// open file for appending
	f, err := os.OpenFile(outFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	defer raven.CheckClosure(f)

	client := m.Session.GetClient()
	var offset int64
	for offset < total {
		end := offset + chunkSize - 1
		if end >= total {
			end = total - 1
		}
		rangeHdr := fmt.Sprintf("bytes=%d-%d", offset, end)

		var success bool
		for attempt := 1; attempt <= retries; attempt++ {
			log.WithField("module", m.Key).Debugf("fetching bytes %d-%d (attempt %d)", offset, end, attempt)
			req, _ := http.NewRequest("GET", url, nil)
			req.Header.Set("Range", rangeHdr)

			resp, requestErr := client.Do(req)
			if requestErr != nil {
				log.WithField("module", m.Key).Warn(requestErr)
			} else {
				if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
					log.WithField("module", m.Key).Warnf("bad status code: %d", resp.StatusCode)
				} else {
					// write body to file
					if _, copyErr := io.Copy(f, resp.Body); copyErr != nil {
						log.WithField("module", m.Key).Warnf(
							"error writing chunk %d-%d: %v",
							offset,
							end,
							copyErr,
						)
					} else {
						success = true
						break
					}
				}
			}

			if attempt < retries {
				log.WithField("module", m.Key).Debugf("retrying in %s", delay)
				time.Sleep(delay)
			}
		}

		if !success {
			return fmt.Errorf("max retries reached, aborting")
		}

		offset = end + 1
	}

	log.WithField("module", m.Key).Infof("successfully downloaded chunks to file: %s", outFile)
	return nil
}
