package paint

import "image"

// Edit is a change of the current layer that is shown while it is being
// made and drawn again on every move: a shape being dragged, a gradient,
// text being typed. Each Draw starts from the layer as it was before the edit.
type Edit struct {
	doc    *Document
	layer  *Layer
	before *image.RGBA
	selB   *Selection
	prev   image.Rectangle // drawn by the last Draw
	dirty  image.Rectangle // drawn by any Draw
}

// BeginEdit starts an edit of the current layer
func (d *Document) BeginEdit() *Edit {
	d.floatingDone()
	l := d.Layer()
	return &Edit{doc: d, layer: l, before: CloneRGBA(l.Img), selB: d.Sel}
}

// Layer returns the layer being edited
func (e *Edit) Layer() *Layer { return e.layer }

// Before returns the layer image as it was before the edit; it must not be changed
func (e *Edit) Before() *image.RGBA { return e.before }

// Sel returns the selection the edit is limited to, nil - none
func (e *Edit) Sel() *Selection { return e.selB }

// Draw undoes what the last Draw did and calls f to draw on the layer
// image dst; before has the pixels as they were. f returns the rectangle
// it changed.
func (e *Edit) Draw(f func(dst, before *image.RGBA) image.Rectangle) {
	if !e.prev.Empty() {
		PasteRGBA(e.layer.Img, CropRGBA(e.before, e.prev))
		e.doc.Invalidate(e.prev)
	}
	r := f(e.layer.Img, e.before).Intersect(e.layer.Img.Rect)
	e.prev = r
	e.dirty = e.dirty.Union(r)
	e.doc.Invalidate(r)
}

// Clear undoes what the last Draw did
func (e *Edit) Clear() {
	e.Draw(func(dst, before *image.RGBA) image.Rectangle { return image.Rectangle{} })
}

// Commit records the edit in the history under the name
func (e *Edit) Commit(name string) {
	if e.prev.Empty() {
		return
	}
	e.doc.PushPixels(name, e.layer, e.dirty, e.before, e.selB)
}

// Cancel puts the layer back as it was before the edit
func (e *Edit) Cancel() {
	e.Clear()
}
