package animation

// FileFormatMkv is the file extension for the MKV format
const FileFormatMkv = ".mkv"

// CreateAnimationMkv tries to create an .mkv video with the passed file data using ImageMagick
func (h *Helper) CreateAnimationMkv(fData *FileData) (content []byte, err error) {
	return h.createAnimationImageMagick(fData, "mkv", true)
}
