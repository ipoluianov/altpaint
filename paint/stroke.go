package paint

import (
	"image"
	"image/color"
	"math"
)

// StrokeKind is what a stroke does to the pixels it covers
type StrokeKind int

const (
	StrokePaint StrokeKind = iota // paints the color
	StrokeErase                   // makes the pixels transparent
	StrokeClone                   // copies the pixels from an offset (clone stamp)
)

// Stroke is a drag of a brush over a layer. The stroke keeps how much each
// pixel is covered by the brush so far (the most of all the dabs, not
// their sum), and the layer is redrawn from its state before the stroke
// with that coverage: a half-transparent color does not get darker where
// the stroke crosses itself.
type Stroke struct {
	doc    *Document
	layer  *Layer
	before *image.RGBA
	selB   *Selection
	cov    *image.Alpha
	dirty  image.Rectangle

	Kind      StrokeKind
	Color     color.NRGBA
	Width     float64
	Antialias bool
	// Blend off replaces the pixels with the color instead of painting over them
	Blend bool
	// Pixel strokes are one pixel wide and hard, like a pencil
	Pixel bool
	// CloneOffset is where the clone stamp takes the pixels from, relative to where it paints
	CloneOffset image.Point

	last    Point
	started bool
}

// BeginStroke starts a stroke on the current layer
func (d *Document) BeginStroke(kind StrokeKind) *Stroke {
	d.floatingDone()
	l := d.Layer()
	return &Stroke{
		doc:    d,
		layer:  l,
		before: CloneRGBA(l.Img),
		selB:   d.Sel,
		cov:    image.NewAlpha(l.Img.Rect),
		Kind:   kind,
		Width:  1,
		Blend:  true,
	}
}

// MoveTo adds the brush moved from the last point to p; the first call puts a dab at p
func (s *Stroke) MoveTo(p Point) {
	from := s.last
	if !s.started {
		from = p
		s.started = true
	}
	s.last = p
	var r image.Rectangle
	if s.Pixel {
		r = s.pixelLine(from, p)
	} else {
		r = s.capsule(from, p, s.Width/2)
	}
	if r.Empty() {
		return
	}
	s.render(r)
}

// capsule covers the pixels within radius of the segment a-b
func (s *Stroke) capsule(a, b Point, radius float64) image.Rectangle {
	radius = max(radius, 0.5)
	pad := radius + 1
	r := image.Rect(
		int(math.Floor(min(a.X, b.X)-pad)), int(math.Floor(min(a.Y, b.Y)-pad)),
		int(math.Ceil(max(a.X, b.X)+pad)), int(math.Ceil(max(a.Y, b.Y)+pad)),
	).Intersect(s.cov.Rect)
	dx, dy := b.X-a.X, b.Y-a.Y
	l2 := dx*dx + dy*dy
	for y := r.Min.Y; y < r.Max.Y; y++ {
		py := float64(y) + 0.5
		i := s.cov.PixOffset(r.Min.X, y)
		for x := r.Min.X; x < r.Max.X; x, i = x+1, i+1 {
			px := float64(x) + 0.5
			t := 0.0
			if l2 > 0 {
				t = max(0, min(1, ((px-a.X)*dx+(py-a.Y)*dy)/l2))
			}
			ex, ey := px-(a.X+t*dx), py-(a.Y+t*dy)
			dist := math.Sqrt(ex*ex + ey*ey)
			var c float64
			if s.Antialias {
				c = max(0, min(1, radius+0.5-dist))
			} else if dist <= radius {
				c = 1
			}
			if v := uint8(c*255 + 0.5); v > s.cov.Pix[i] {
				s.cov.Pix[i] = v
			}
		}
	}
	return r
}

// pixelLine covers the pixels of the line a-b, one pixel wide (Bresenham)
func (s *Stroke) pixelLine(a, b Point) image.Rectangle {
	x0, y0 := int(math.Floor(a.X)), int(math.Floor(a.Y))
	x1, y1 := int(math.Floor(b.X)), int(math.Floor(b.Y))
	r := image.Rect(min(x0, x1), min(y0, y1), max(x0, x1)+1, max(y0, y1)+1)
	dx, dy := abs(x1-x0), -abs(y1-y0)
	sx, sy := 1, 1
	if x0 > x1 {
		sx = -1
	}
	if y0 > y1 {
		sy = -1
	}
	e := dx + dy
	for {
		if image.Pt(x0, y0).In(s.cov.Rect) {
			s.cov.Pix[s.cov.PixOffset(x0, y0)] = 255
		}
		if x0 == x1 && y0 == y1 {
			break
		}
		if e2 := 2 * e; e2 >= dy {
			e += dy
			x0 += sx
		} else {
			e += dx
			y0 += sy
		}
	}
	return r.Intersect(s.cov.Rect)
}

func abs(v int) int {
	return max(v, -v)
}

// render draws the stroke on the layer within r
func (s *Stroke) render(r image.Rectangle) {
	sel := s.selB
	if sel != nil {
		r = r.Intersect(sel.Bounds())
	}
	if r.Empty() {
		return
	}
	s.dirty = s.dirty.Union(r)
	dst, b := s.layer.Img, s.before
	pc := Premul(s.Color)
	for y := r.Min.Y; y < r.Max.Y; y++ {
		ci := s.cov.PixOffset(r.Min.X, y)
		di := dst.PixOffset(r.Min.X, y)
		for x := r.Min.X; x < r.Max.X; x, ci, di = x+1, ci+1, di+4 {
			cov := uint32(s.cov.Pix[ci])
			if sel != nil {
				cov = cov * uint32(sel.Mask.Pix[sel.Mask.PixOffset(x, y)]) / 255
			}
			d := dst.Pix[di : di+4 : di+4]
			bp := b.Pix[di : di+4 : di+4]
			switch s.Kind {
			case StrokeErase:
				inv := 255 - cov
				for ch := range 4 {
					d[ch] = uint8((uint32(bp[ch])*inv + 127) / 255)
				}
			case StrokeClone:
				sx, sy := x+s.CloneOffset.X, y+s.CloneOffset.Y
				if !image.Pt(sx, sy).In(b.Rect) {
					copy(d, bp)
					continue
				}
				si := b.PixOffset(sx, sy)
				c := color.RGBA{b.Pix[si], b.Pix[si+1], b.Pix[si+2], b.Pix[si+3]}
				paintPixel(d, bp, c, cov, true)
			default:
				paintPixel(d, bp, pc, cov, s.Blend)
			}
		}
	}
	s.doc.Invalidate(r)
}

// End finishes the stroke and records it in the history under the name
func (s *Stroke) End(name string) {
	if s.dirty.Empty() {
		return
	}
	s.doc.PushPixels(name, s.layer, s.dirty, s.before, s.selB)
}
