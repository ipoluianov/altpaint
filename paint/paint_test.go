package paint

import (
	"bytes"
	"image"
	"image/color"
	"path/filepath"
	"testing"
)

var (
	red   = color.NRGBA{255, 0, 0, 255}
	white = color.NRGBA{255, 255, 255, 255}
)

func pixel(img *image.RGBA, x, y int) color.RGBA {
	return img.RGBAAt(x, y)
}

func TestStrokeUndoRedo(t *testing.T) {
	d := NewDocument(20, 20, white, "Background")
	s := d.BeginStroke(StrokePaint)
	s.Color = red
	s.Width = 3
	s.MoveTo(Point{5.5, 5.5})
	s.MoveTo(Point{15.5, 5.5})
	s.End("Paintbrush")

	if got := pixel(d.Layer().Img, 10, 5); got != (color.RGBA{255, 0, 0, 255}) {
		t.Fatalf("stroke pixel = %v", got)
	}
	if !d.Modified() {
		t.Fatal("the document must be modified after a stroke")
	}
	d.Undo()
	if got := pixel(d.Layer().Img, 10, 5); got != (color.RGBA{255, 255, 255, 255}) {
		t.Fatalf("after undo = %v", got)
	}
	if d.Modified() {
		t.Fatal("undone to the saved state: not modified")
	}
	d.Redo()
	if got := pixel(d.Composite(), 10, 5); got != (color.RGBA{255, 0, 0, 255}) {
		t.Fatalf("composite after redo = %v", got)
	}
}

// A half-transparent stroke crossing itself is not darker where it crosses
func TestStrokeDoesNotAccumulate(t *testing.T) {
	d := NewDocument(20, 20, color.NRGBA{}, "Layer")
	s := d.BeginStroke(StrokePaint)
	s.Color = color.NRGBA{0, 0, 0, 128}
	s.Width = 5
	s.MoveTo(Point{10, 10})
	s.MoveTo(Point{11, 10})
	s.MoveTo(Point{10, 10})
	s.End("Paintbrush")
	if a := pixel(d.Layer().Img, 10, 10).A; a != 128 {
		t.Fatalf("alpha where the stroke crosses itself = %d, want 128", a)
	}
}

func TestSelectionLimitsPainting(t *testing.T) {
	d := NewDocument(20, 20, white, "Background")
	d.SetSelection("Select", SelectRect(20, 20, image.Rect(0, 0, 10, 20)))
	s := d.BeginStroke(StrokePaint)
	s.Color = red
	s.Width = 4
	s.MoveTo(Point{2, 10})
	s.MoveTo(Point{18, 10})
	s.End("Paintbrush")
	if got := pixel(d.Layer().Img, 5, 10); got.G != 0 {
		t.Fatalf("inside the selection = %v, want red", got)
	}
	if got := pixel(d.Layer().Img, 15, 10); got.G != 255 {
		t.Fatalf("outside the selection = %v, want white", got)
	}
	// Undo of the stroke, then of the selection
	d.Undo()
	d.Undo()
	if d.Sel != nil {
		t.Fatal("the selection must be undone")
	}
}

func TestLayersAndBlend(t *testing.T) {
	d := NewDocument(4, 4, color.NRGBA{100, 100, 100, 255}, "Background")
	d.AddLayer("Add Layer", "Layer 2")
	if len(d.Layers) != 2 || d.Current != 1 {
		t.Fatalf("layers %d, current %d", len(d.Layers), d.Current)
	}
	e := d.BeginEdit()
	e.Draw(func(dst, before *image.RGBA) image.Rectangle {
		m := image.NewAlpha(dst.Rect)
		for i := range m.Pix {
			m.Pix[i] = 255
		}
		return FillMask(dst, before, m, color.NRGBA{128, 128, 128, 255}, nil, true)
	})
	e.Commit("Fill")
	d.SetLayerProps("Layer Properties", d.Layer(), "Layer 2", true, 255, BlendMultiply)
	// 100 * 128 / 255 = 50
	if got := pixel(d.Composite(), 1, 1); got.R != 50 {
		t.Fatalf("multiply = %v, want 50", got)
	}
	d.MergeDown("Merge Layer Down")
	if len(d.Layers) != 1 || pixel(d.Layer().Img, 1, 1).R != 50 {
		t.Fatalf("merged: %d layers, %v", len(d.Layers), pixel(d.Layer().Img, 1, 1))
	}
	d.Undo() // merge
	d.Undo() // props
	if got := pixel(d.Composite(), 1, 1); got.R != 128 {
		t.Fatalf("normal after undo = %v, want 128", got)
	}
	d.Undo() // fill
	d.Undo() // add layer
	if len(d.Layers) != 1 {
		t.Fatalf("after undo: %d layers", len(d.Layers))
	}
	d.GoTo(4)
	if len(d.Layers) != 1 || pixel(d.Layer().Img, 1, 1).R != 50 {
		t.Fatal("redo all must give the merged layer again")
	}
}

func TestOutlineHasHole(t *testing.T) {
	outer := RectPolygon(0, 0, 10, 10)
	inner := RectPolygon(2, 2, 8, 8).Reversed()
	m := FillPolygons([]Polygon{outer, inner}, true, image.Rect(0, 0, 10, 10))
	if m.AlphaAt(1, 5).A != 255 || m.AlphaAt(5, 5).A != 0 {
		t.Fatalf("ring: edge %d, center %d", m.AlphaAt(1, 5).A, m.AlphaAt(5, 5).A)
	}
}

func TestFloodFill(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	for y := range 10 {
		img.SetRGBA(5, y, color.RGBA{0, 0, 0, 255}) // a wall
	}
	m := FloodMask(img, image.Pt(1, 1), 0, true)
	if m.AlphaAt(4, 9).A != 255 || m.AlphaAt(5, 5).A != 0 || m.AlphaAt(7, 5).A != 0 {
		t.Fatal("contiguous fill must stop at the wall")
	}
	m = FloodMask(img, image.Pt(1, 1), 0, false)
	if m.AlphaAt(7, 5).A != 255 {
		t.Fatal("global fill must reach past the wall")
	}
}

func TestMoveSelected(t *testing.T) {
	d := NewDocument(10, 10, color.NRGBA{}, "Layer")
	d.Layer().Img.SetRGBA(2, 2, color.RGBA{255, 0, 0, 255})
	d.SetSelection("Select", SelectRect(10, 10, image.Rect(2, 2, 3, 3)))
	m := d.BeginMove()
	m.To(image.Pt(3, 0))
	m.To(image.Pt(5, 1))
	m.End("Move Selected")
	img := d.Layer().Img
	if img.RGBAAt(2, 2).A != 0 || img.RGBAAt(7, 3).R != 255 {
		t.Fatalf("moved: old %v, new %v", img.RGBAAt(2, 2), img.RGBAAt(7, 3))
	}
	if d.Sel.Bounds() != image.Rect(7, 3, 8, 4) {
		t.Fatalf("selection = %v", d.Sel.Bounds())
	}
	d.Undo()
	if img.RGBAAt(2, 2).R != 255 || img.RGBAAt(7, 3).A != 0 {
		t.Fatal("undo must put the pixel back")
	}
}

func TestTransforms(t *testing.T) {
	d := NewDocument(3, 2, color.NRGBA{}, "Layer")
	d.Layer().Img.SetRGBA(0, 0, color.RGBA{255, 0, 0, 255})
	d.RotateImage("Rotate", 1)
	if d.W != 2 || d.H != 3 || d.Layer().Img.RGBAAt(1, 0).R != 255 {
		t.Fatalf("rotated %dx%d, %v", d.W, d.H, d.Layer().Img.RGBAAt(1, 0))
	}
	d.ResizeCanvas("Canvas Size", 4, 5, Anchor{1, 1}, nil)
	if d.Layer().Img.RGBAAt(3, 2).R != 255 {
		t.Fatal("the canvas must grow on the left and top")
	}
	d.SetSelection("Select", SelectRect(4, 5, image.Rect(3, 2, 4, 3)))
	d.CropToSelection("Crop")
	if d.W != 1 || d.H != 1 || d.Layer().Img.RGBAAt(0, 0).R != 255 {
		t.Fatal("crop must leave the pixel")
	}
	d.GoTo(0)
	if d.W != 3 || d.H != 2 {
		t.Fatalf("undo all: %dx%d", d.W, d.H)
	}
}

func TestFiltersKeepSize(t *testing.T) {
	d := NewDocument(16, 12, color.NRGBA{200, 120, 40, 255}, "Layer")
	for _, list := range [][]*Filter{Adjustments, Effects} {
		for _, f := range list {
			e := d.BeginEdit()
			e.Draw(func(dst, before *image.RGBA) image.Rectangle {
				return ApplyFilter(dst, before, nil, f, f.Defaults())
			})
			e.Commit(f.ID)
		}
	}
	if d.History.Pos() != len(Adjustments)+len(Effects) {
		t.Fatalf("steps %d", d.History.Pos())
	}
}

func TestNativeRoundTrip(t *testing.T) {
	d := NewDocument(8, 6, white, "Background")
	d.AddLayer("Add Layer", "Top")
	d.Layer().Img.SetRGBA(3, 3, color.RGBA{0, 0, 128, 128})
	d.SetLayerProps("Props", d.Layer(), "Top", true, 200, BlendScreen)
	path := filepath.Join(t.TempDir(), "x"+NativeExt)
	if err := Save(d, path, SaveOptions{}); err != nil {
		t.Fatal(err)
	}
	d2, err := Open(path, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(d2.Layers) != 2 || d2.Layers[1].Blend != BlendScreen || d2.Layers[1].Opacity != 200 || d2.Layers[1].Name != "Top" {
		t.Fatalf("layers: %+v", d2.Layers[1])
	}
	if !bytes.Equal(d2.Layers[1].Img.Pix, d.Layers[1].Img.Pix) {
		t.Fatal("the pixels must be the same")
	}
	for _, ext := range []string{".png", ".jpg", ".bmp", ".gif", ".tiff"} {
		p := filepath.Join(t.TempDir(), "x"+ext)
		if err := Save(d, p, SaveOptions{}); err != nil {
			t.Fatalf("%s: %v", ext, err)
		}
		d3, err := Open(p, "Background")
		if err != nil || d3.W != 8 || d3.H != 6 {
			t.Fatalf("%s: %v", ext, err)
		}
	}
}

func TestText(t *testing.T) {
	m, lay := TextMask("Hi\nthere", image.Pt(5, 5), TextStyle{Size: 20, Antialias: true})
	if lay.Bounds.Dy() != 2*lay.LineHeight || m.Rect.Empty() {
		t.Fatalf("layout %+v", lay)
	}
	covered := 0
	for _, v := range m.Pix {
		if v > 0 {
			covered++
		}
	}
	if covered == 0 {
		t.Fatal("no text drawn")
	}
}
