package images

import (
	"io"
	"image"
	"image/jpeg"
	"github.com/nfnt/resize"
)

type encoder struct {
	r io.Reader
	i image.Image
}

func Encoder(r io.Reader) *encoder {
	return &encoder{r: r}
}

func (e *encoder) EncodeCover(f io.Writer) (err error) {
	if e.i == nil {
		e.i, _, err = image.Decode(e.r)
		if err != nil {
			return
		}
	}
	err = jpeg.Encode(f, e.i, nil)
	return
}

func (e *encoder) EncodeThumbnail(f io.Writer, maxWidth, maxHeight uint) (err error) {
	if e.i == nil {
		e.i, _, err = image.Decode(e.r)
		if err != nil {
			return
		}
	}

	ti := resize.Thumbnail(maxWidth, maxHeight, e.i, resize.Bicubic)
	err = jpeg.Encode(f, ti, nil)
	return
}
