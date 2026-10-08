package forms

import (
	"bytes"
	"image"
	"image/color"
	"testing"

	"github.com/ipoluianov/altpaint/paint"
)

// Drawing again only the changed part of the view gives the same picture as
// drawing it all, at any zoom: the stroke does not leave seams
func TestRenderPartEqualsAll(t *testing.T) {
	for _, z := range []float64{0.07, 0.37, 1, 3} {
		cv := NewCanvasView(nil)
		cv.SetSize(500, 400)
		d := paint.NewDocument(900, 700, color.NRGBA{200, 220, 240, 255}, "Background")
		d.AddLayer("AddLayer", "Layer")
		tab := &DocTab{doc: d, zoom: z}
		cv.tab = tab
		cv.syncContent()
		cv.SetScrollX(cv.InnerWidth()/2 - 250)
		cv.SetScrollY(cv.InnerHeight()/2 - 200)
		if cv.Width() != 500 || cv.Height() != 400 {
			t.Fatalf("view size %dx%d", cv.Width(), cv.Height())
		}
		cv.render(image.Rect(0, 0, 500, 400))
		before := bytes.Clone(cv.buf.Pix)
		d.TakeViewDirty()

		s := d.BeginStroke(paint.StrokePaint)
		s.Color = color.NRGBA{200, 0, 0, 160}
		s.Width = 9
		s.Antialias = true
		s.MoveTo(paint.Point{X: 300, Y: 300})
		s.MoveTo(paint.Point{X: 520, Y: 410})
		s.End("Paintbrush")
		cv.render(cv.screenRect(d.TakeViewDirty()))
		part := bytes.Clone(cv.buf.Pix)
		if bytes.Equal(part, before) {
			t.Fatalf("zoom %v: the stroke is not seen", z)
		}

		cv.render(cv.buf.Rect)
		if !bytes.Equal(part, cv.buf.Pix) {
			t.Errorf("zoom %v: the part drawn differs from the whole", z)
		}
	}
}
