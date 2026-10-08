package paint

import (
	"image"
	"image/color"
	"math"
)

// GradientKind is the shape of a gradient
type GradientKind int

const (
	GradientLinear GradientKind = iota
	GradientLinearReflected
	GradientDiamond
	GradientRadial
	GradientConical
)

// GradientKinds lists the kinds in the order they are offered
var GradientKinds = []GradientKind{GradientLinear, GradientLinearReflected, GradientDiamond, GradientRadial, GradientConical}

// DrawGradient fills the selection (or the whole image) of dst with the
// gradient from c0 at a to c1 at b, painted over before. Returns the changed rectangle.
func DrawGradient(dst, before *image.RGBA, sel *Selection, kind GradientKind, a, b Point, c0, c1 color.NRGBA) image.Rectangle {
	r := dst.Rect
	if sel != nil {
		r = r.Intersect(sel.Bounds())
	}
	dx, dy := b.X-a.X, b.Y-a.Y
	l2 := dx*dx + dy*dy
	l := math.Sqrt(l2)
	angle0 := math.Atan2(dy, dx)
	at := func(px, py float64) float64 {
		if l2 == 0 {
			return 0
		}
		vx, vy := px-a.X, py-a.Y
		switch kind {
		case GradientLinearReflected:
			return math.Abs((vx*dx + vy*dy) / l2)
		case GradientDiamond:
			// The distance along the line and across it, the larger one
			along := math.Abs(vx*dx+vy*dy) / l2
			across := math.Abs(vx*dy-vy*dx) / l2
			return along + across
		case GradientRadial:
			return math.Sqrt(vx*vx+vy*vy) / l
		case GradientConical:
			ang := math.Atan2(vy, vx) - angle0
			ang = math.Mod(ang+4*math.Pi, 2*math.Pi)
			if ang > math.Pi {
				ang = 2*math.Pi - ang
			}
			return ang / math.Pi
		}
		return (vx*dx + vy*dy) / l2
	}
	// Straight colors are mixed, then premultiplied
	mix := func(t float64) color.RGBA {
		t = max(0, min(1, t))
		lerp := func(u, v uint8) uint8 { return uint8(float64(u) + (float64(v)-float64(u))*t + 0.5) }
		return Premul(color.NRGBA{lerp(c0.R, c1.R), lerp(c0.G, c1.G), lerp(c0.B, c1.B), lerp(c0.A, c1.A)})
	}
	var lut [1024]color.RGBA
	for i := range lut {
		lut[i] = mix(float64(i) / float64(len(lut)-1))
	}
	for y := r.Min.Y; y < r.Max.Y; y++ {
		di := dst.PixOffset(r.Min.X, y)
		for x := r.Min.X; x < r.Max.X; x, di = x+1, di+4 {
			cov := uint32(255)
			if sel != nil {
				cov = uint32(sel.Mask.Pix[sel.Mask.PixOffset(x, y)])
			}
			t := max(0, min(1, at(float64(x)+0.5, float64(y)+0.5)))
			c := lut[int(t*float64(len(lut)-1)+0.5)]
			paintPixel(dst.Pix[di:di+4:di+4], before.Pix[di:di+4:di+4], c, cov, true)
		}
	}
	return r
}
