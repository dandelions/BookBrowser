// +build imgvips

// Implements libvips image processing. Inexplicably, this ended up being slower than the reference implementation
// so it's disabled by default. Gah.
package images

/*
// Commented out so that gomod doesn't pull in a ton of unnecessary dependencies.
import (
	"io"
	"gopkg.in/h2non/bimg.v1"
	"io/ioutil"
)

type encoder struct {
	r io.Reader
	i *bimg.Image
}

func Encoder(r io.Reader) *encoder {
	return &encoder{r: r}
}

func (e *encoder) readImage() error {
	buf, err := ioutil.ReadAll(e.r)
	if err != nil {
		return err
	}
	e.i = bimg.NewImage(buf)
	if e.i.Type() != "jpeg" {
		buf, err := e.i.Process(bimg.Options{Type: bimg.JPEG, Quality: 75})
		if err != nil {
			return err
		}
		e.i = bimg.NewImage(buf)
	}
	return nil
}

func (e *encoder) EncodeCover(f io.Writer) (err error) {
	if e.i == nil {
		if err = e.readImage(); err != nil {
			return
		}
	}

	_, err = f.Write(e.i.Image())
	return
}

func (e *encoder) EncodeThumbnail(f io.Writer, maxWidth, maxHeight uint) (err error) {
	if e.i == nil {
		if err = e.readImage(); err != nil {
			return
		}
	}

	buf, err := e.i.Process(bimg.Options{Width: int(maxWidth), Height: int(maxHeight), Quality: 75})
	_, err = f.Write(buf)
	return
}

func init() {
	bimg.VipsCacheSetMax(0)
	bimg.VipsCacheSetMaxMem(0)
}
*/
