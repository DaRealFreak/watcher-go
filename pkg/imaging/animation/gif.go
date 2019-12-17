package animation

import (
	"bytes"
	"fmt"
	"image"
	"image/color/palette"
	"image/gif"

	log "github.com/sirupsen/logrus"
	"golang.org/x/image/draw"

	// imports for registering formats to image decoder
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	// more imports for registering formats to image decoder
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/webp"
)

const FILE_FORMAT_GIF = ".gif"

// CreateAnimationGif creates a .gif (Graphics Interchange Format) file from the passed fileData.
// If ImageMagick GIF creation fails we use a fallback method to retrieve a lower quality gif made with golang libraries
func (h *Helper) CreateAnimationGif(fData *FileData) (content []byte, err error) {
	log.Debugf("trying to create GIF animation with ImageMagick")

	content, err = h.createAnimationGifImageMagick(fData)
	if err != nil {
		log.Debugf("trying to create GIF animation using go library fallback")

		content, err = h.createAnimationGifGo(fData)
	}

	return
}

// createAnimationGifImageMagick tries to create a high quality GIF using ImageMagick
func (h *Helper) createAnimationGifImageMagick(fData *FileData) (content []byte, err error) {
	return h.createAnimationImageMagick(fData, "gif", true)
}

// createAnimationGifGo is a fallback function to create a GIF file with the golang image libraries
// quality suffers a lot (256 colors max f.e.), so ImageMagick conversion would be preferable
func (h *Helper) createAnimationGifGo(fData *FileData) (content []byte, err error) {
	if len(fData.Frames) != len(fData.MsDelays) {
		return nil, fmt.Errorf("delays don't match the frame count")
	}

	outGif := &gif.GIF{}

	for i := 0; i <= len(fData.Frames)-1; i++ {
		decodedImage, _, err := image.Decode(bytes.NewReader(fData.Frames[i]))
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
