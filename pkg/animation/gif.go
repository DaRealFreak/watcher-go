package animation

import (
	"bytes"
	"fmt"
	log "github.com/sirupsen/logrus"
	_ "golang.org/x/image/bmp"
	"golang.org/x/image/draw"
	"image"
	"image/color/palette"
	"image/gif"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
)

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
	if len(fileData.Frames) != len(fileData.MsDelays) {
		return nil, fmt.Errorf("delays don't match the frame count")
	}

	outGif := &gif.GIF{}
	for i := 0; i <= len(fileData.Frames)-1; i++ {
		decodedImage, _, err := image.Decode(bytes.NewReader(fileData.Frames[i]))
		if err != nil {
			return nil, err
		}
		// create new paletted image and draw our decoded image into it
		imageFilePaletted := image.NewPaletted(decodedImage.Bounds(), palette.Plan9)
		draw.Draw(imageFilePaletted, imageFilePaletted.Rect, decodedImage, decodedImage.Bounds().Min, draw.Over)

		// append the new paletted image and the delay to the image
		outGif.Image = append(outGif.Image, imageFilePaletted)
		outGif.Delay = append(outGif.Delay, 0)
	}
	f := new(bytes.Buffer)
	if err := gif.EncodeAll(f, outGif); err != nil {
		return nil, err
	}
	return f.Bytes(), nil
}
