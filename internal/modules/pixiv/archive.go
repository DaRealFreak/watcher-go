package pixiv

import (
	"archive/zip"
	"io"

	"github.com/DaRealFreak/watcher-go/internal/raven"
)

// readZipFile reads from a zip file object
func (m *pixiv) readZipFile(zf *zip.File) ([]byte, error) {
	f, err := zf.Open()
	if err != nil {
		return nil, err
	}

	defer raven.CheckClosureNonFatal(f)

	return io.ReadAll(f)
}
