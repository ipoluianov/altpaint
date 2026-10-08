package paint

import (
	"image"
	"math"
)

// FloodMask returns the pixels of src similar to the one at seed: 255 for
// the ones within the tolerance (0..1) of its color, 0 for the others.
// Contiguous takes only the ones connected to the seed, otherwise all of
// them in the image. The mask has the bounds of src.
func FloodMask(src *image.RGBA, seed image.Point, tolerance float64, contiguous bool) *image.Alpha {
	r := src.Rect
	mask := image.NewAlpha(r)
	if !seed.In(r) {
		return mask
	}
	si := src.PixOffset(seed.X, seed.Y)
	ref := [4]int{int(src.Pix[si]), int(src.Pix[si+1]), int(src.Pix[si+2]), int(src.Pix[si+3])}
	// The distance in RGBA, squared, compared without the root
	limit := int(math.Round(tolerance * tolerance * 4 * 255 * 255))
	match := func(i int) bool {
		p := src.Pix[i : i+4 : i+4]
		d0, d1, d2, d3 := int(p[0])-ref[0], int(p[1])-ref[1], int(p[2])-ref[2], int(p[3])-ref[3]
		return d0*d0+d1*d1+d2*d2+d3*d3 <= limit
	}

	if !contiguous {
		for y := r.Min.Y; y < r.Max.Y; y++ {
			i := src.PixOffset(r.Min.X, y)
			m := mask.PixOffset(r.Min.X, y)
			for x := r.Min.X; x < r.Max.X; x, i, m = x+1, i+4, m+1 {
				if match(i) {
					mask.Pix[m] = 255
				}
			}
		}
		return mask
	}

	// Scanline fill: a run of the row is filled at once, and the rows
	// above and below it are searched for runs to fill next
	stack := []image.Point{seed}
	for len(stack) > 0 {
		p := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if mask.Pix[mask.PixOffset(p.X, p.Y)] != 0 || !match(src.PixOffset(p.X, p.Y)) {
			continue
		}
		x0 := p.X
		for x0 > r.Min.X && mask.Pix[mask.PixOffset(x0-1, p.Y)] == 0 && match(src.PixOffset(x0-1, p.Y)) {
			x0--
		}
		x1 := p.X
		for x1+1 < r.Max.X && mask.Pix[mask.PixOffset(x1+1, p.Y)] == 0 && match(src.PixOffset(x1+1, p.Y)) {
			x1++
		}
		for x := x0; x <= x1; x++ {
			mask.Pix[mask.PixOffset(x, p.Y)] = 255
		}
		for _, y := range []int{p.Y - 1, p.Y + 1} {
			if y < r.Min.Y || y >= r.Max.Y {
				continue
			}
			inRun := false
			for x := x0; x <= x1; x++ {
				ok := mask.Pix[mask.PixOffset(x, y)] == 0 && match(src.PixOffset(x, y))
				if ok && !inRun {
					stack = append(stack, image.Pt(x, y))
				}
				inRun = ok
			}
		}
	}
	return mask
}
