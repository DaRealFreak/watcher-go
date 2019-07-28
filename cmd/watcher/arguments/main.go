package arguments

import "flag"

var downloadDirectory *string

func InitFlags() {
	downloadDirectory = flag.String("directory", "./", "directory to download the media files into")
	flag.Parse()
}

func GetDownloadDirectory() string {
	return *downloadDirectory
}
