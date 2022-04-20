package animation

// FileFormatWebm is the file extension for the WebM format
const FileFormatWebm = ".webm"

// CreateAnimationWebM tries to create an .mkv video with the passed file data using ImageMagick
func (h *Helper) CreateAnimationWebM(fData *FileData) (content []byte, err error) {
	return h.createAnimationImageMagick(fData, "webm", true)
}
