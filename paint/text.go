package paint

import (
	"image"
	"strings"
	"sync"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/gobolditalic"
	"golang.org/x/image/font/gofont/goitalic"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// TextStyle is how the Text tool writes
type TextStyle struct {
	Size   float64 // in pixels
	Bold   bool
	Italic bool
	Mono   bool
	// Antialias smooths the edges of the letters
	Antialias bool
}

var (
	fontsMtx sync.Mutex
	fonts    = map[[3]bool]*opentype.Font{}
	faces    = map[faceKey]font.Face{}
)

type faceKey struct {
	style [3]bool
	size  float64
}

func (s TextStyle) face() font.Face {
	fontsMtx.Lock()
	defer fontsMtx.Unlock()
	style := [3]bool{s.Bold, s.Italic, s.Mono}
	key := faceKey{style, s.Size}
	if f, ok := faces[key]; ok {
		return f
	}
	f, ok := fonts[style]
	if !ok {
		data := goregular.TTF
		switch {
		case s.Mono:
			data = gomono.TTF
		case s.Bold && s.Italic:
			data = gobolditalic.TTF
		case s.Bold:
			data = gobold.TTF
		case s.Italic:
			data = goitalic.TTF
		}
		var err error
		f, err = opentype.Parse(data)
		if err != nil {
			return nil
		}
		fonts[style] = f
	}
	face, err := opentype.NewFace(f, &opentype.FaceOptions{Size: max(1, s.Size), DPI: 72, Hinting: font.HintingNone})
	if err != nil {
		return nil
	}
	faces[key] = face
	return face
}

// TextLayout tells where the lines of a text are: their baselines and the
// position of the caret after the text
type TextLayout struct {
	Bounds     image.Rectangle
	LineHeight int
	Caret      image.Rectangle // a thin rectangle where the next letter goes
}

// TextMask returns the coverage of the text written from the top left
// corner at, line by line, and where it is
func TextMask(text string, at image.Point, style TextStyle) (*image.Alpha, TextLayout) {
	face := style.face()
	var lay TextLayout
	if face == nil {
		return image.NewAlpha(image.Rectangle{}), lay
	}
	m := face.Metrics()
	ascent := m.Ascent.Ceil()
	lineH := m.Height.Ceil()
	lay.LineHeight = lineH
	lines := strings.Split(text, "\n")
	width := 0
	for _, line := range lines {
		width = max(width, font.MeasureString(face, line).Ceil())
	}
	// Italic letters lean out past their advance
	pad := int(style.Size/4) + 2
	b := image.Rect(at.X-pad, at.Y, at.X+width+pad, at.Y+lineH*len(lines)+pad)
	mask := image.NewAlpha(b)
	dr := &font.Drawer{Dst: mask, Src: image.Opaque, Face: face}
	for i, line := range lines {
		dr.Dot = fixed.P(at.X, at.Y+ascent+i*lineH)
		dr.DrawString(line)
	}
	if !style.Antialias {
		Threshold(mask)
	}
	last := lines[len(lines)-1]
	cx := at.X + font.MeasureString(face, last).Ceil()
	cy := at.Y + (len(lines)-1)*lineH
	lay.Caret = image.Rect(cx, cy, cx+max(1, int(style.Size/16)), cy+lineH)
	lay.Bounds = image.Rect(at.X, at.Y, at.X+width, at.Y+lineH*len(lines))
	return mask, lay
}
