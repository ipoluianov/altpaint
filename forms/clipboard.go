package forms

import (
	"bytes"
	"errors"
	"image"
	"image/png"

	"github.com/ipoluianov/altpaint/paint"
)

// The clipboard keeps the copied image in the application too: a paste
// works where the system clipboard cannot be reached (no xclip on Linux),
// and the system one is read first, so an image copied in another
// application is pasted when there is one.
var internalClip *image.RGBA

var errNoSystemClipboard = errors.New("the system clipboard is not available")

// clipboardWrite copies the image; an error tells it is only in the application
func clipboardWrite(img *image.RGBA) error {
	internalClip = paint.ToRGBA(img)
	var buf bytes.Buffer
	if err := png.Encode(&buf, internalClip); err != nil {
		return err
	}
	return systemClipboardWrite(buf.Bytes())
}

// clipboardRead returns the image of the system clipboard, or the one copied in the application
func clipboardRead() (*image.RGBA, error) {
	if data, err := systemClipboardRead(); err == nil && len(data) > 0 {
		if img, err := paint.DecodeImage(bytes.NewReader(data)); err == nil {
			return img, nil
		}
	}
	if internalClip == nil {
		return nil, errNoSystemClipboard
	}
	return internalClip, nil
}

// clipboardPeek returns the image copied in the application, without asking the system
func clipboardPeek() *image.RGBA {
	return internalClip
}
