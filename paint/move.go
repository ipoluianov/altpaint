package paint

import (
	"image"
	"image/draw"
)

// floating are the selected pixels lifted off the layer by the Move
// Selected tool or pasted: they can be moved around without leaving a
// trace until something else is done. The layer always shows them where
// they are; base is the layer without them.
type floating struct {
	layer  *Layer
	base   *image.RGBA     // the layer without the lifted pixels
	pixels *image.RGBA     // the lifted pixels at their original place
	sel    *Selection      // the selection at the original place
	offset image.Point     // where they are now, relative to the original place
	drawn  image.Rectangle // where they were drawn on the layer last
}

// Move is a drag of the Move Selected tool
type Move struct {
	doc    *Document
	before *image.RGBA // the layer at the start of the drag
	selB   *Selection
	start  image.Point // offset at the start
	dirty  image.Rectangle
}

// BeginMove starts moving the selected pixels of the current layer (all of
// it when nothing is selected). The first move lifts them.
func (d *Document) BeginMove() *Move {
	l := d.Layer()
	f := d.floating
	if f == nil || f.layer != l {
		d.dropFloating()
		sel := d.Sel
		if sel == nil {
			sel = SelectRect(d.W, d.H, d.Bounds())
		}
		r := sel.Bounds()
		pixels := image.NewRGBA(r)
		draw.DrawMask(pixels, r, l.Img, r.Min, sel.Mask, r.Min, draw.Src)
		base := CloneRGBA(l.Img)
		// The hole: what is lifted is taken out of the base
		for y := r.Min.Y; y < r.Max.Y; y++ {
			for x := r.Min.X; x < r.Max.X; x++ {
				inv := 255 - uint32(sel.Mask.Pix[sel.Mask.PixOffset(x, y)])
				i := base.PixOffset(x, y)
				for ch := range 4 {
					base.Pix[i+ch] = uint8((uint32(base.Pix[i+ch])*inv + 127) / 255)
				}
			}
		}
		f = &floating{layer: l, base: base, pixels: pixels, sel: sel, drawn: r}
		d.floating = f
	}
	return &Move{doc: d, before: CloneRGBA(l.Img), selB: d.Sel, start: f.offset}
}

// To moves the pixels by delta from where they were at the start of the drag
func (m *Move) To(delta image.Point) {
	f := m.doc.floating
	if f == nil {
		return
	}
	f.offset = m.start.Add(delta)
	m.dirty = m.dirty.Union(f.drawn) // where they leave
	m.doc.redrawFloating()
	m.dirty = m.dirty.Union(f.drawn)
	if m.doc.Sel != nil {
		m.dirty = m.dirty.Union(m.doc.Sel.Bounds())
	}
}

// End records the move in the history
func (m *Move) End(name string) {
	f := m.doc.floating
	if f == nil || m.dirty.Empty() {
		return
	}
	// The floating pixels stay after the step is recorded
	m.doc.push(m.doc.pixelsItem(name, f.layer, m.dirty, m.before, m.selB))
	m.doc.Invalidate(m.dirty)
}

// redrawFloating draws the layer as the base with the floating pixels at their offset
func (d *Document) redrawFloating() {
	f := d.floating
	r := f.pixels.Rect.Add(f.offset).Intersect(f.layer.Img.Rect)
	area := r.Union(f.drawn)
	PasteRGBA(f.layer.Img, CropRGBA(f.base, area))
	draw.Draw(f.layer.Img, r, f.pixels, r.Min.Sub(f.offset), draw.Over)
	d.Invalidate(area)
	f.drawn = r
	if f.offset == (image.Point{}) {
		d.Sel = f.sel
	} else {
		d.Sel = f.sel.Translated(f.offset.X, f.offset.Y)
	}
}

// Float makes the pixels floating on the current layer at their bounds, as
// pasted: they cover the layer until moved away. Recorded as a step.
func (d *Document) Float(name string, pixels *image.RGBA) {
	d.dropFloating()
	l := d.Layer()
	before := CloneRGBA(l.Img)
	selB := d.Sel
	r := pixels.Rect.Intersect(l.Img.Rect)
	mask := image.NewAlpha(pixels.Rect)
	draw.Draw(mask, mask.Rect, image.Opaque, image.Point{}, draw.Src)
	sel := SelectMask(d.W, d.H, mask)
	if sel == nil {
		return
	}
	f := &floating{layer: l, base: CloneRGBA(l.Img), pixels: pixels, sel: sel, drawn: r}
	d.floating = f
	d.redrawFloating()
	d.push(d.pixelsItem(name, l, pixels.Rect.Intersect(l.Img.Rect), before, selB))
}

// pixelsItem makes the history step of PushPixels without recording it
func (d *Document) pixelsItem(name string, layer *Layer, r image.Rectangle, before *image.RGBA, selBefore *Selection) *HistoryItem {
	r = r.Intersect(layer.Img.Rect)
	b := CropRGBA(before, r)
	a := CropRGBA(layer.Img, r)
	selAfter := d.Sel
	return &HistoryItem{
		Name: name,
		undo: func(d *Document) {
			PasteRGBA(layer.Img, b)
			d.Sel = selBefore
			d.Invalidate(r)
		},
		redo: func(d *Document) {
			PasteRGBA(layer.Img, a)
			d.Sel = selAfter
			d.Invalidate(r)
		},
	}
}

// HasFloating tells whether there are lifted or pasted pixels being moved
func (d *Document) HasFloating() bool {
	return d.floating != nil
}

// floatingDone fixes the floating pixels where they are: the layer already
// shows them, so they are just forgotten
func (d *Document) floatingDone() {
	d.floating = nil
}

func (d *Document) dropFloating() {
	d.floating = nil
}

// Deselect fixes the floating pixels and records the selection change
func (d *Document) Deselect(name string) {
	if d.Sel == nil {
		d.floatingDone()
		return
	}
	d.Do(name, func() { d.Sel = nil })
}

// SetSelection records the change of the selection as a step
func (d *Document) SetSelection(name string, s *Selection) {
	d.Do(name, func() { d.Sel = s })
}
