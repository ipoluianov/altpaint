package paint

import "image"

// History is the list of the changes of a document, undone and redone in order.
//
// There are two kinds of steps. A pixel step keeps the area of a layer as it
// was before and after a change (a stroke, an effect). A state step keeps the
// list of the layers, their properties and image pointers, the size and the
// selection: adding or removing a layer, resizing. The layers are shared
// between the states, not copied: a pixel step changes the pixels of a layer
// in place, and since the steps are undone in order, the pixels are as the
// state expects them by the time it is restored.
type History struct {
	items   []*HistoryItem
	pos     int // items[:pos] are done
	savedAt int // pos when the document was saved, -1 when that state is gone
}

// HistoryItem is a step of the history
type HistoryItem struct {
	// Name is the id of what was done, e.g. "Paintbrush"; the UI shows its translation
	Name string
	undo func(d *Document)
	redo func(d *Document)
}

// Items returns the steps, the oldest first
func (h *History) Items() []*HistoryItem {
	return h.items
}

// Pos returns how many of the steps are done; the ones after it can be redone
func (h *History) Pos() int {
	return h.pos
}

func (h *History) CanUndo() bool { return h.pos > 0 }
func (h *History) CanRedo() bool { return h.pos < len(h.items) }

// historyLimit is how many steps are kept; the oldest go
const historyLimit = 100

func (d *Document) push(item *HistoryItem) {
	h := &d.History
	if h.savedAt > h.pos {
		h.savedAt = -1 // the saved state was undone and is replaced now
	}
	h.items = append(h.items[:h.pos], item)
	h.pos++
	if len(h.items) > historyLimit {
		n := len(h.items) - historyLimit
		h.items = append(h.items[:0:0], h.items[n:]...)
		h.pos -= n
		h.savedAt -= n
		if h.savedAt < 0 {
			h.savedAt = -1
		}
	}
	d.version++
}

// Undo undoes the last done step
func (d *Document) Undo() {
	d.dropFloating()
	if !d.History.CanUndo() {
		return
	}
	d.History.pos--
	d.History.items[d.History.pos].undo(d)
	d.version++
}

// Redo does again the first undone step
func (d *Document) Redo() {
	d.dropFloating()
	if !d.History.CanRedo() {
		return
	}
	d.History.items[d.History.pos].redo(d)
	d.History.pos++
	d.version++
}

// GoTo undoes or redoes the steps until pos of them are done
func (d *Document) GoTo(pos int) {
	for d.History.pos > pos && d.History.CanUndo() {
		d.Undo()
	}
	for d.History.pos < pos && d.History.CanRedo() {
		d.Redo()
	}
}

// PushPixels records a change of the area r of the layer: before is the area
// as it was (its bounds include r), the layer has it as it is now. selBefore
// is the selection before the change, d.Sel after it.
func (d *Document) PushPixels(name string, layer *Layer, r image.Rectangle, before *image.RGBA, selBefore *Selection) {
	d.floatingDone()
	d.push(d.pixelsItem(name, layer, r, before, selBefore))
	d.Invalidate(r)
}

// docState is the document without the pixels, see History
type docState struct {
	w, h    int
	layers  []*Layer
	props   []Layer // the fields of each layer, the image pointer included
	current int
	sel     *Selection
}

func (d *Document) state() docState {
	s := docState{w: d.W, h: d.H, layers: append([]*Layer(nil), d.Layers...), current: d.Current, sel: d.Sel}
	for _, l := range d.Layers {
		s.props = append(s.props, *l)
	}
	return s
}

func (d *Document) restore(s docState) {
	d.W, d.H = s.w, s.h
	d.Layers = append(d.Layers[:0:0], s.layers...)
	for i, l := range d.Layers {
		*l = s.props[i]
	}
	d.Current = s.current
	d.Sel = s.sel
	d.InvalidateAll()
}

// Do runs f, which changes the layers, their properties, the size or the
// selection, and records the change as a step. f must not change the pixels
// of the layers in place: it gives a layer a new image instead.
func (d *Document) Do(name string, f func()) {
	d.dropFloating()
	before := d.state()
	f()
	d.Current = max(0, min(d.Current, len(d.Layers)-1))
	after := d.state()
	d.push(&HistoryItem{
		Name: name,
		undo: func(d *Document) { d.restore(before) },
		redo: func(d *Document) { d.restore(after) },
	})
	d.InvalidateAll()
}

// SetCurrent makes the layer current; this is not a step of the history
func (d *Document) SetCurrent(i int) {
	if i >= 0 && i < len(d.Layers) && i != d.Current {
		d.dropFloating()
		d.Current = i
		d.version++
	}
}
