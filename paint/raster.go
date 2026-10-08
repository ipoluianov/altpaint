package paint

import (
	"image"
	"image/color"
	"math"

	"golang.org/x/image/vector"
)

// Point is a position in the image, in pixels; (0.5, 0.5) is the center of the first pixel
type Point struct {
	X, Y float64
}

// Polygon is a closed path
type Polygon []Point

// FillPolygons returns the coverage of the polygons: 255 inside, the edge
// pixels partly covered when aa is on. A polygon going the other way cuts a
// hole in the ones it is inside of (an outline is a shape with a hole).
// The mask covers bounds, which is clipped to clip.
func FillPolygons(polys []Polygon, aa bool, clip image.Rectangle) *image.Alpha {
	b := polygonsBounds(polys).Intersect(clip)
	if b.Empty() {
		return image.NewAlpha(image.Rectangle{})
	}
	z := vector.NewRasterizer(b.Dx(), b.Dy())
	ox, oy := float32(b.Min.X), float32(b.Min.Y)
	for _, p := range polys {
		if len(p) < 3 {
			continue
		}
		z.MoveTo(float32(p[0].X)-ox, float32(p[0].Y)-oy)
		for _, q := range p[1:] {
			z.LineTo(float32(q.X)-ox, float32(q.Y)-oy)
		}
		z.ClosePath()
	}
	mask := image.NewAlpha(b)
	z.Draw(mask, b, image.Opaque, image.Point{})
	if !aa {
		Threshold(mask)
	}
	return mask
}

// Threshold makes a soft mask hard: more than half covered is covered
func Threshold(m *image.Alpha) {
	for i, v := range m.Pix {
		if v >= 128 {
			m.Pix[i] = 255
		} else {
			m.Pix[i] = 0
		}
	}
}

func polygonsBounds(polys []Polygon) image.Rectangle {
	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	for _, p := range polys {
		for _, q := range p {
			minX, minY = min(minX, q.X), min(minY, q.Y)
			maxX, maxY = max(maxX, q.X), max(maxY, q.Y)
		}
	}
	if minX > maxX {
		return image.Rectangle{}
	}
	return image.Rect(int(math.Floor(minX)), int(math.Floor(minY)), int(math.Ceil(maxX)), int(math.Ceil(maxY)))
}

// RectPolygon returns the rectangle from (x0, y0) to (x1, y1)
func RectPolygon(x0, y0, x1, y1 float64) Polygon {
	return Polygon{{x0, y0}, {x1, y0}, {x1, y1}, {x0, y1}}
}

// EllipsePolygon returns the ellipse in the rectangle, approximated finely enough for its size
func EllipsePolygon(x0, y0, x1, y1 float64) Polygon {
	cx, cy := (x0+x1)/2, (y0+y1)/2
	rx, ry := math.Abs(x1-x0)/2, math.Abs(y1-y0)/2
	n := max(16, int(math.Ceil(math.Max(rx, ry)*1.5)))
	n = min(n, 2048)
	p := make(Polygon, n)
	for i := range p {
		a := 2 * math.Pi * float64(i) / float64(n)
		p[i] = Point{cx + rx*math.Cos(a), cy + ry*math.Sin(a)}
	}
	return p
}

// RoundRectPolygon returns the rectangle with the corners rounded by radius
func RoundRectPolygon(x0, y0, x1, y1, radius float64) Polygon {
	x0, x1 = min(x0, x1), max(x0, x1)
	y0, y1 = min(y0, y1), max(y0, y1)
	radius = min(radius, (x1-x0)/2, (y1-y0)/2)
	if radius <= 0 {
		return RectPolygon(x0, y0, x1, y1)
	}
	n := max(4, int(math.Ceil(radius)))
	var p Polygon
	corner := func(cx, cy, a0 float64) {
		for i := 0; i <= n; i++ {
			a := a0 + math.Pi/2*float64(i)/float64(n)
			p = append(p, Point{cx + radius*math.Cos(a), cy + radius*math.Sin(a)})
		}
	}
	corner(x1-radius, y0+radius, -math.Pi/2)
	corner(x1-radius, y1-radius, 0)
	corner(x0+radius, y1-radius, math.Pi/2)
	corner(x0+radius, y0+radius, math.Pi)
	return p
}

// Reversed returns the polygon going the other way, to cut a hole
func (p Polygon) Reversed() Polygon {
	r := make(Polygon, len(p))
	for i, q := range p {
		r[len(p)-1-i] = q
	}
	return r
}

// LinePolygon returns the line from a to b of the width, with flat ends
func LinePolygon(a, b Point, width float64) Polygon {
	dx, dy := b.X-a.X, b.Y-a.Y
	l := math.Hypot(dx, dy)
	if l == 0 {
		return RectPolygon(a.X-width/2, a.Y-width/2, a.X+width/2, a.Y+width/2)
	}
	nx, ny := -dy/l*width/2, dx/l*width/2
	return Polygon{{a.X + nx, a.Y + ny}, {b.X + nx, b.Y + ny}, {b.X - nx, b.Y - ny}, {a.X - nx, a.Y - ny}}
}

// FillMask paints the color over the area of the layer image under the
// mask, starting from before (an image with the same pixels as dst had,
// at least in the mask bounds). sel limits it to the selection (nil - no
// limit). blend off replaces the pixels instead of painting over them.
// Returns the changed rectangle.
func FillMask(dst, before *image.RGBA, mask *image.Alpha, col color.NRGBA, sel *Selection, blend bool) image.Rectangle {
	r := mask.Rect.Intersect(dst.Rect)
	if sel != nil {
		r = r.Intersect(sel.Bounds())
	}
	pc := Premul(col)
	for y := r.Min.Y; y < r.Max.Y; y++ {
		mi := mask.PixOffset(r.Min.X, y)
		di := dst.PixOffset(r.Min.X, y)
		for x := r.Min.X; x < r.Max.X; x, mi, di = x+1, mi+1, di+4 {
			cov := uint32(mask.Pix[mi])
			if sel != nil {
				cov = cov * uint32(sel.Mask.Pix[sel.Mask.PixOffset(x, y)]) / 255
			}
			paintPixel(dst.Pix[di:di+4:di+4], before.Pix[di:di+4:di+4], pc, cov, blend)
		}
	}
	return r
}

// paintPixel sets d to b with the premultiplied color c painted over it with the coverage cov (0..255)
func paintPixel(d, b []uint8, c color.RGBA, cov uint32, blend bool) {
	if cov == 0 {
		copy(d, b)
		return
	}
	if !blend {
		// Replace: the color takes cov of the pixel
		inv := 255 - cov
		d[0] = uint8((uint32(c.R)*cov + uint32(b[0])*inv + 127) / 255)
		d[1] = uint8((uint32(c.G)*cov + uint32(b[1])*inv + 127) / 255)
		d[2] = uint8((uint32(c.B)*cov + uint32(b[2])*inv + 127) / 255)
		d[3] = uint8((uint32(c.A)*cov + uint32(b[3])*inv + 127) / 255)
		return
	}
	sr := uint32(c.R) * cov / 255
	sg := uint32(c.G) * cov / 255
	sb := uint32(c.B) * cov / 255
	sa := uint32(c.A) * cov / 255
	inv := 255 - sa
	d[0] = uint8(sr + (uint32(b[0])*inv+127)/255)
	d[1] = uint8(sg + (uint32(b[1])*inv+127)/255)
	d[2] = uint8(sb + (uint32(b[2])*inv+127)/255)
	d[3] = uint8(sa + (uint32(b[3])*inv+127)/255)
}
