package animation

import log "github.com/sirupsen/logrus"

// create gif animated picture from the passed FileData
func (h *Helper) CreateAnimationGif(fileData *FileData) (content []byte, err error) {
	log.Debugf("trying to create GIF animation with ImageMagick")
	content, err = h.createAnimationGifImageMagick(fileData)
	if err != nil {
		log.Debugf("trying to create GIF animation using go library fallback")
		content, err = h.createAnimationGifGo(fileData)
	}
	return
}

// create a high quality gif with ImageMagick
func (h *Helper) createAnimationGifImageMagick(fileData *FileData) (content []byte, err error) {
	return h.createAnimationImageMagick(fileData, "gif", true)
}

// fallback function to create a gif file with the golang image libraries
// quality suffers a lot (256 colors max f.e.), so ImageMagick conversion would be preferable
func (h *Helper) createAnimationGifGo(fileData *FileData) (content []byte, err error) {
	// ToDo: implement again
	return
}
