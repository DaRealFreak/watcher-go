package animation

// create mkv video from the passed frames
func (h *Helper) CreateAnimationMkv(fileData *FileData) (content []byte, err error) {
	return h.createAnimationImageMagick(fileData, "mkv", true)
}
