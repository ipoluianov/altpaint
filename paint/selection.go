package paint

import (
	"image"
	"image/draw"
)

// Selection is the selected part of the document: a mask of its size, 255
// fully selected, 0 not selected, between - the soft edge of an
// antialiased shape. A selection is not changed once made: the
// operations return a new one.
type Selection struct {
	Mask   *image.Alpha
	bounds image.Rectangle // of the selected pixels

	outline []Segment
}

// CombineMode is how a new selection joins the current one
type CombineMode int

const (
	CombineReplace CombineMode = iota
	CombineUnion
	CombineExclude
	CombineIntersect
	CombineXor
)

// Segment is a piece of the border of the selection, between pixel corners;
// it is horizontal or vertical
type Segment struct {
	X1, Y1, X2, Y2 int
}

// NewSelection makes a selection of the mask; nil when nothing is selected
func NewSelection(mask *image.Alpha) *Selection {
	s := &Selection{Mask: mask}
	s.bounds = alphaBounds(mask)
	if s.bounds.Empty() {
		return nil
	}
	return s
}

// SelectRect returns the selection of the rectangle in a w x h document
func SelectRect(w, h int, r image.Rectangle) *Selection {
	mask := image.NewAlpha(image.Rect(0, 0, w, h))
	draw.Draw(mask, r, image.Opaque, image.Point{}, draw.Src)
	return NewSelection(mask)
}

// SelectMask returns the selection of the coverage mask (of any bounds) in a w x h document
func SelectMask(w, h int, cov *image.Alpha) *Selection {
	mask := image.NewAlpha(image.Rect(0, 0, w, h))
	draw.Draw(mask, cov.Rect, cov, cov.Rect.Min, draw.Src)
	return NewSelection(mask)
}

// Bounds returns the rectangle around the selected pixels
func (s *Selection) Bounds() image.Rectangle {
	return s.bounds
}

// Combine returns the selection that results from adding other to s by the
// mode; s may be nil (nothing selected), the result too
func Combine(s, other *Selection, mode CombineMode) *Selection {
	if mode == CombineReplace || s == nil && (mode == CombineUnion || mode == CombineXor) {
		return other
	}
	if s == nil {
		return nil // exclude from or intersect with nothing
	}
	if other == nil {
		if mode == CombineIntersect {
			return nil
		}
		return s
	}
	mask := image.NewAlpha(s.Mask.Rect)
	a, b := s.Mask.Pix, other.Mask.Pix
	for i := range mask.Pix {
		x, y := int(a[i]), int(b[i])
		var v int
		switch mode {
		case CombineUnion:
			v = max(x, y)
		case CombineExclude:
			v = x * (255 - y) / 255
		case CombineIntersect:
			v = min(x, y)
		case CombineXor:
			v = max(x, y) - min(x, y)
		}
		mask.Pix[i] = uint8(v)
	}
	return NewSelection(mask)
}

// Inverted returns the selection of what s does not select; nil s selects everything
func Inverted(w, h int, s *Selection) *Selection {
	if s == nil {
		return nil
	}
	mask := image.NewAlpha(s.Mask.Rect)
	for i, v := range s.Mask.Pix {
		mask.Pix[i] = 255 - v
	}
	return NewSelection(mask)
}

// Translated returns the selection moved by dx, dy
func (s *Selection) Translated(dx, dy int) *Selection {
	mask := image.NewAlpha(s.Mask.Rect)
	src := s.Mask
	r := s.bounds.Add(image.Pt(dx, dy)).Intersect(mask.Rect)
	for y := r.Min.Y; y < r.Max.Y; y++ {
		copy(mask.Pix[mask.PixOffset(r.Min.X, y):mask.PixOffset(r.Max.X, y)],
			src.Pix[src.PixOffset(r.Min.X-dx, y-dy):src.PixOffset(r.Max.X-dx, y-dy)])
	}
	return NewSelection(mask)
}

// Resized returns the selection in a document of the new size, placed at offset
func (s *Selection) Resized(w, h int, offset image.Point) *Selection {
	mask := image.NewAlpha(image.Rect(0, 0, w, h))
	draw.Draw(mask, s.Mask.Rect.Add(offset), s.Mask, image.Point{}, draw.Src)
	return NewSelection(mask)
}

// Outline returns the border of the selection: the edges between the pixels
// that are mostly selected and the ones that are not, joined into runs
func (s *Selection) Outline() []Segment {
	if s.outline != nil {
		return s.outline
	}
	m := s.Mask
	in := func(x, y int) bool {
		if x < m.Rect.Min.X || y < m.Rect.Min.Y || x >= m.Rect.Max.X || y >= m.Rect.Max.Y {
			return false
		}
		return m.Pix[m.PixOffset(x, y)] >= 128
	}
	b := s.bounds
	segs := make([]Segment, 0)
	// Horizontal edges: above each row of the bounds and below the last
	for y := b.Min.Y; y <= b.Max.Y; y++ {
		start := -1
		for x := b.Min.X; x <= b.Max.X; x++ {
			edge := x < b.Max.X && in(x, y) != in(x, y-1)
			if edge && start < 0 {
				start = x
			}
			if !edge && start >= 0 {
				segs = append(segs, Segment{start, y, x, y})
				start = -1
			}
		}
	}
	for x := b.Min.X; x <= b.Max.X; x++ {
		start := -1
		for y := b.Min.Y; y <= b.Max.Y; y++ {
			edge := y < b.Max.Y && in(x, y) != in(x-1, y)
			if edge && start < 0 {
				start = y
			}
			if !edge && start >= 0 {
				segs = append(segs, Segment{x, start, x, y})
				start = -1
			}
		}
	}
	s.outline = segs
	return segs
}

// alphaBounds returns the rectangle around the nonzero pixels of the mask
func alphaBounds(m *image.Alpha) image.Rectangle {
	r := m.Rect
	minX, minY, maxX, maxY := r.Max.X, r.Max.Y, r.Min.X-1, r.Min.Y-1
	for y := r.Min.Y; y < r.Max.Y; y++ {
		row := m.Pix[m.PixOffset(r.Min.X, y):m.PixOffset(r.Max.X, y)]
		first := -1
		for i, v := range row {
			if v != 0 {
				first = i
				break
			}
		}
		if first < 0 {
			continue
		}
		last := first
		for i := len(row) - 1; i > first; i-- {
			if row[i] != 0 {
				last = i
				break
			}
		}
		minX = min(minX, r.Min.X+first)
		maxX = max(maxX, r.Min.X+last)
		minY = min(minY, y)
		maxY = y
	}
	if maxX < minX {
		return image.Rectangle{}
	}
	return image.Rect(minX, minY, maxX+1, maxY+1)
}
