package pixiv

import (
	"archive/zip"
	"io/ioutil"

	"github.com/DaRealFreak/watcher-go/pkg/raven"
)

// readZipFile reads from a zip file object
func (m *pixiv) readZipFile(zf *zip.File) ([]byte, error) {
	f, err := zf.Open()
	if err != nil {
		return nil, err
	}
	defer raven.CheckClosure(f)
	return ioutil.ReadAll(f)
}
