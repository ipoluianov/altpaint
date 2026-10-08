// Package paint is the image model of AltPaint: a document of layers, its
// history, the selection, and the drawing, adjustments and effects done on
// it. It knows nothing of the windows; everything runs on the UI thread.
package paint

import (
	"image"
	"image/color"
	"image/draw"
)

// Layer is one picture of the stack. Its pixels are premultiplied RGBA of
// the document size, starting at (0, 0).
type Layer struct {
	Name    string
	Visible bool
	Opacity uint8
	Blend   BlendMode
	Img     *image.RGBA
}

// NewLayer returns a transparent visible layer
func NewLayer(name string, w, h int) *Layer {
	return &Layer{Name: name, Visible: true, Opacity: 255, Img: image.NewRGBA(image.Rect(0, 0, w, h))}
}

// Clone returns a copy of the layer with its own pixels
func (l *Layer) Clone() *Layer {
	c := *l
	c.Img = CloneRGBA(l.Img)
	return &c
}

// Document is an image being edited: a stack of layers, the bottom one first
type Document struct {
	W, H    int
	Layers  []*Layer
	Current int
	// Sel is the selection, nil when nothing is selected (the tools work on the whole layer).
	// A selection is never changed in place: a new one replaces it, so the history can keep it.
	Sel *Selection

	History History

	// Path is the file the document was opened from or saved to, "" for a new one
	Path string
	// Name is shown for a new document, e.g. "Untitled 2"
	Name string

	composite *image.RGBA
	dirty     image.Rectangle
	version   int
	// viewDirty is what changed since the view last drew the image, see TakeViewDirty
	viewDirty image.Rectangle

	// floating holds the pixels lifted by the Move Selected tool, see move.go
	floating *floating
}

// NewDocument returns a document with one layer filled with bg
func NewDocument(w, h int, bg color.Color, layerName string) *Document {
	d := &Document{W: w, H: h}
	l := NewLayer(layerName, w, h)
	if _, _, _, a := bg.RGBA(); a > 0 {
		draw.Draw(l.Img, l.Img.Rect, image.NewUniform(bg), image.Point{}, draw.Src)
	}
	d.Layers = []*Layer{l}
	d.InvalidateAll()
	return d
}

// NewDocumentFromImage returns a document with the image as its only layer
func NewDocumentFromImage(img image.Image, layerName string) *Document {
	b := img.Bounds()
	d := &Document{W: b.Dx(), H: b.Dy()}
	l := NewLayer(layerName, d.W, d.H)
	draw.Draw(l.Img, l.Img.Rect, img, b.Min, draw.Src)
	d.Layers = []*Layer{l}
	d.InvalidateAll()
	return d
}

func (d *Document) Bounds() image.Rectangle {
	return image.Rect(0, 0, d.W, d.H)
}

// Layer returns the current layer
func (d *Document) Layer() *Layer {
	return d.Layers[d.Current]
}

// LayerIndex returns the index of the layer, -1 when it is not in the stack
func (d *Document) LayerIndex(l *Layer) int {
	for i, x := range d.Layers {
		if x == l {
			return i
		}
	}
	return -1
}

// Version changes with every change of the document, so the views know when to repaint
func (d *Document) Version() int {
	return d.version
}

// Touch marks the document changed without changing the pixels (a property, the selection)
func (d *Document) Touch() {
	d.version++
}

// Invalidate marks the area of the composite image to be drawn again
func (d *Document) Invalidate(r image.Rectangle) {
	r = r.Intersect(d.Bounds())
	d.dirty = d.dirty.Union(r)
	d.viewDirty = d.viewDirty.Union(r)
	d.version++
}

// InvalidateAll marks the whole composite image to be drawn again
func (d *Document) InvalidateAll() {
	d.dirty = d.Bounds()
	d.viewDirty = d.Bounds()
	d.version++
}

// TakeViewDirty returns the area of the image that changed since the last
// call, so a view draws again only that part
func (d *Document) TakeViewDirty() image.Rectangle {
	r := d.viewDirty
	d.viewDirty = image.Rectangle{}
	return r
}

// Composite returns the layers drawn one over another; it is updated where
// they changed since the last call. The image must not be changed.
func (d *Document) Composite() *image.RGBA {
	if d.composite == nil || d.composite.Rect != d.Bounds() {
		d.composite = image.NewRGBA(d.Bounds())
		d.dirty = d.Bounds()
	}
	if !d.dirty.Empty() {
		r := d.dirty
		d.dirty = image.Rectangle{}
		ClearRect(d.composite, r)
		for _, l := range d.Layers {
			if l.Visible && l.Opacity > 0 {
				BlendLayer(d.composite, l.Img, r, l.Opacity, l.Blend)
			}
		}
	}
	return d.composite
}

// Flattened returns a copy of the composite image
func (d *Document) Flattened() *image.RGBA {
	return CloneRGBA(d.Composite())
}

// SelAlpha returns how much the pixel is selected: 255 everywhere when nothing is
func (d *Document) SelAlpha(x, y int) uint8 {
	if d.Sel == nil {
		return 255
	}
	return d.Sel.Mask.AlphaAt(x, y).A
}

// SelBounds returns the selected area, the whole image when nothing is selected
func (d *Document) SelBounds() image.Rectangle {
	if d.Sel == nil {
		return d.Bounds()
	}
	return d.Sel.Bounds()
}

// Modified tells whether there are changes since the document was opened or saved
func (d *Document) Modified() bool {
	return d.History.pos != d.History.savedAt
}

// MarkSaved remembers the current state as the saved one
func (d *Document) MarkSaved() {
	d.History.savedAt = d.History.pos
	d.version++
}

// CloneRGBA returns a copy of the image
func CloneRGBA(src *image.RGBA) *image.RGBA {
	dst := &image.RGBA{
		Pix:    make([]uint8, len(src.Pix)),
		Stride: src.Stride,
		Rect:   src.Rect,
	}
	copy(dst.Pix, src.Pix)
	return dst
}

// CropRGBA returns a copy of the area of the image; its bounds are r
func CropRGBA(src *image.RGBA, r image.Rectangle) *image.RGBA {
	r = r.Intersect(src.Rect)
	dst := image.NewRGBA(r)
	for y := r.Min.Y; y < r.Max.Y; y++ {
		copy(dst.Pix[dst.PixOffset(r.Min.X, y):dst.PixOffset(r.Max.X, y)], src.Pix[src.PixOffset(r.Min.X, y):src.PixOffset(r.Max.X, y)])
	}
	return dst
}

// PasteRGBA copies src into dst at the bounds of src
func PasteRGBA(dst *image.RGBA, src *image.RGBA) {
	r := src.Rect.Intersect(dst.Rect)
	for y := r.Min.Y; y < r.Max.Y; y++ {
		copy(dst.Pix[dst.PixOffset(r.Min.X, y):dst.PixOffset(r.Max.X, y)], src.Pix[src.PixOffset(r.Min.X, y):src.PixOffset(r.Max.X, y)])
	}
}

// ClearRect makes the area transparent
func ClearRect(img *image.RGBA, r image.Rectangle) {
	r = r.Intersect(img.Rect)
	for y := r.Min.Y; y < r.Max.Y; y++ {
		clear(img.Pix[img.PixOffset(r.Min.X, y):img.PixOffset(r.Max.X, y)])
	}
}

// Premul returns the color premultiplied, as the layers keep it
func Premul(c color.NRGBA) color.RGBA {
	a := uint32(c.A)
	return color.RGBA{
		R: uint8((uint32(c.R)*a + 127) / 255),
		G: uint8((uint32(c.G)*a + 127) / 255),
		B: uint8((uint32(c.B)*a + 127) / 255),
		A: c.A,
	}
}

// Unpremul returns the straight color of a premultiplied one
func Unpremul(c color.RGBA) color.NRGBA {
	if c.A == 0 {
		return color.NRGBA{}
	}
	if c.A == 255 {
		return color.NRGBA{c.R, c.G, c.B, 255}
	}
	a := uint32(c.A)
	return color.NRGBA{
		R: uint8(min(255, (uint32(c.R)*255+a/2)/a)),
		G: uint8(min(255, (uint32(c.G)*255+a/2)/a)),
		B: uint8(min(255, (uint32(c.B)*255+a/2)/a)),
		A: c.A,
	}
}
