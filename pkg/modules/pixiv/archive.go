package pixiv

import (
	"archive/zip"
	"io/ioutil"
)

func (m *pixiv) readZipFile(zf *zip.File) ([]byte, error) {
	f, err := zf.Open()
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ioutil.ReadAll(f)
}
