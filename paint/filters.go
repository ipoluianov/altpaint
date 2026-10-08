package paint

import (
	"image"
	"image/color"
	"math"
	"math/rand/v2"
)

// Param is a number a filter takes, set in its dialog
type Param struct {
	// ID names the parameter; the UI shows its translation
	ID       string
	Min, Max float64
	Default  float64
}

// Filter is an adjustment or an effect: a change of all the pixels of the
// selected area of a layer
type Filter struct {
	// ID names the filter; the UI shows its translation
	ID     string
	Params []Param
	// Apply returns the area r of src changed; the result has the bounds r.
	// src has the whole layer, so the filters that look around a pixel see past r.
	Apply func(src *image.RGBA, r image.Rectangle, p []float64) *image.RGBA
}

// Defaults returns the default values of the parameters
func (f *Filter) Defaults() []float64 {
	v := make([]float64, len(f.Params))
	for i, p := range f.Params {
		v[i] = p.Default
	}
	return v
}

// ApplyFilter draws the filter applied to before on dst, within the
// selection; returns the changed rectangle. Used with an Edit, so it can be
// shown while the parameters are changed.
func ApplyFilter(dst, before *image.RGBA, sel *Selection, f *Filter, params []float64) image.Rectangle {
	r := before.Rect
	if sel != nil {
		r = r.Intersect(sel.Bounds())
	}
	if r.Empty() {
		return r
	}
	res := f.Apply(before, r, params)
	for y := r.Min.Y; y < r.Max.Y; y++ {
		di := dst.PixOffset(r.Min.X, y)
		ri := res.PixOffset(r.Min.X, y)
		for x := r.Min.X; x < r.Max.X; x, di, ri = x+1, di+4, ri+4 {
			s := uint32(255)
			if sel != nil {
				s = uint32(sel.Mask.Pix[sel.Mask.PixOffset(x, y)])
			}
			if s == 255 {
				copy(dst.Pix[di:di+4], res.Pix[ri:ri+4])
				continue
			}
			inv := 255 - s
			for ch := range 4 {
				dst.Pix[di+ch] = uint8((uint32(res.Pix[ri+ch])*s + uint32(before.Pix[di+ch])*inv + 127) / 255)
			}
		}
	}
	return r
}

// mapColors returns the area with every straight color changed by f
func mapColors(src *image.RGBA, r image.Rectangle, f func(c color.NRGBA) color.NRGBA) *image.RGBA {
	dst := image.NewRGBA(r)
	for y := r.Min.Y; y < r.Max.Y; y++ {
		si := src.PixOffset(r.Min.X, y)
		di := dst.PixOffset(r.Min.X, y)
		for x := r.Min.X; x < r.Max.X; x, si, di = x+1, si+4, di+4 {
			p := src.Pix[si : si+4 : si+4]
			if p[3] == 0 {
				continue
			}
			c := Premul(f(Unpremul(color.RGBA{p[0], p[1], p[2], p[3]})))
			dst.Pix[di], dst.Pix[di+1], dst.Pix[di+2], dst.Pix[di+3] = c.R, c.G, c.B, c.A
		}
	}
	return dst
}

func clamp8(v float64) uint8 {
	return uint8(max(0, min(255, math.Round(v))))
}

func luma(c color.NRGBA) float64 {
	return 0.299*float64(c.R) + 0.587*float64(c.G) + 0.114*float64(c.B)
}

// Adjustments are the color changes of the Adjustments menu
var Adjustments = []*Filter{
	{ID: "AutoLevel", Apply: autoLevel},
	{ID: "BlackAndWhite", Apply: func(src *image.RGBA, r image.Rectangle, p []float64) *image.RGBA {
		return mapColors(src, r, func(c color.NRGBA) color.NRGBA {
			v := clamp8(luma(c))
			return color.NRGBA{v, v, v, c.A}
		})
	}},
	{ID: "BrightnessContrast", Params: []Param{{"Brightness", -100, 100, 0}, {"Contrast", -100, 100, 0}},
		Apply: func(src *image.RGBA, r image.Rectangle, p []float64) *image.RGBA {
			b := p[0] * 255 / 100
			k := math.Tan((p[1] + 100) / 200 * math.Pi / 2) // 0 at -100, 1 at 0, steep at 100
			var lut [256]uint8
			for i := range lut {
				lut[i] = clamp8((float64(i)-127.5)*k + 127.5 + b)
			}
			return mapColors(src, r, func(c color.NRGBA) color.NRGBA {
				return color.NRGBA{lut[c.R], lut[c.G], lut[c.B], c.A}
			})
		}},
	{ID: "HueSaturation", Params: []Param{{"Hue", -180, 180, 0}, {"Saturation", 0, 200, 100}, {"Lightness", -100, 100, 0}},
		Apply: func(src *image.RGBA, r image.Rectangle, p []float64) *image.RGBA {
			return mapColors(src, r, func(c color.NRGBA) color.NRGBA {
				h, s, l := rgbToHSL(c)
				h = math.Mod(h+p[0]/360+1, 1)
				s = max(0, min(1, s*p[1]/100))
				if p[2] > 0 {
					l += (1 - l) * p[2] / 100
				} else {
					l += l * p[2] / 100
				}
				out := hslToRGB(h, s, l)
				out.A = c.A
				return out
			})
		}},
	{ID: "InvertColors", Apply: func(src *image.RGBA, r image.Rectangle, p []float64) *image.RGBA {
		return mapColors(src, r, func(c color.NRGBA) color.NRGBA {
			return color.NRGBA{255 - c.R, 255 - c.G, 255 - c.B, c.A}
		})
	}},
	{ID: "Posterize", Params: []Param{{"Levels", 2, 64, 16}},
		Apply: func(src *image.RGBA, r image.Rectangle, p []float64) *image.RGBA {
			n := max(2, math.Round(p[0]))
			var lut [256]uint8
			for i := range lut {
				lut[i] = clamp8(math.Round(float64(i)/255*(n-1)) * 255 / (n - 1))
			}
			return mapColors(src, r, func(c color.NRGBA) color.NRGBA {
				return color.NRGBA{lut[c.R], lut[c.G], lut[c.B], c.A}
			})
		}},
	{ID: "Sepia", Apply: func(src *image.RGBA, r image.Rectangle, p []float64) *image.RGBA {
		return mapColors(src, r, func(c color.NRGBA) color.NRGBA {
			v := luma(c)
			return color.NRGBA{clamp8(v * 1.07), clamp8(v * 0.89), clamp8(v * 0.67), c.A}
		})
	}},
}

// autoLevel stretches each channel to the full range, ignoring the rarest 0.5% at each end
func autoLevel(src *image.RGBA, r image.Rectangle, p []float64) *image.RGBA {
	var hist [3][256]int
	n := 0
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			i := src.PixOffset(x, y)
			if src.Pix[i+3] == 0 {
				continue
			}
			c := Unpremul(color.RGBA{src.Pix[i], src.Pix[i+1], src.Pix[i+2], src.Pix[i+3]})
			hist[0][c.R]++
			hist[1][c.G]++
			hist[2][c.B]++
			n++
		}
	}
	var luts [3][256]uint8
	cut := n / 200
	for ch := range 3 {
		lo, hi, sum := 0, 255, 0
		for lo < 255 && sum+hist[ch][lo] <= cut {
			sum += hist[ch][lo]
			lo++
		}
		sum = 0
		for hi > lo && sum+hist[ch][hi] <= cut {
			sum += hist[ch][hi]
			hi--
		}
		for i := range 256 {
			if hi > lo {
				luts[ch][i] = clamp8(float64(i-lo) * 255 / float64(hi-lo))
			} else {
				luts[ch][i] = uint8(i)
			}
		}
	}
	return mapColors(src, r, func(c color.NRGBA) color.NRGBA {
		return color.NRGBA{luts[0][c.R], luts[1][c.G], luts[2][c.B], c.A}
	})
}

// Effects are the filters of the Effects menu
var Effects = []*Filter{
	{ID: "GaussianBlur", Params: []Param{{"Radius", 0, 100, 2}}, Apply: func(src *image.RGBA, r image.Rectangle, p []float64) *image.RGBA {
		return gaussianBlur(src, r, p[0])
	}},
	{ID: "MotionBlur", Params: []Param{{"Angle", -180, 180, 25}, {"Distance", 1, 200, 10}}, Apply: motionBlur},
	{ID: "Sharpen", Params: []Param{{"Amount", 1, 20, 2}}, Apply: func(src *image.RGBA, r image.Rectangle, p []float64) *image.RGBA {
		blur := gaussianBlur(src, r, max(1, p[0]/2))
		dst := image.NewRGBA(r)
		for y := r.Min.Y; y < r.Max.Y; y++ {
			for x := r.Min.X; x < r.Max.X; x++ {
				si, bi := src.PixOffset(x, y), blur.PixOffset(x, y)
				a := src.Pix[si+3]
				for ch := range 3 {
					v := 2*float64(src.Pix[si+ch]) - float64(blur.Pix[bi+ch])
					dst.Pix[bi+ch] = uint8(max(0, min(float64(a), math.Round(v))))
				}
				dst.Pix[bi+3] = a
			}
		}
		return dst
	}},
	{ID: "AddNoise", Params: []Param{{"Intensity", 0, 100, 64}, {"ColorSaturation", 0, 400, 100}, {"Coverage", 0, 100, 100}}, Apply: addNoise},
	{ID: "Pixelate", Params: []Param{{"CellSize", 1, 100, 4}}, Apply: pixelate},
	{ID: "Emboss", Params: []Param{{"Angle", -180, 180, 0}}, Apply: func(src *image.RGBA, r image.Rectangle, p []float64) *image.RGBA {
		return relief(src, r, p[0], true)
	}},
	{ID: "EdgeDetect", Params: []Param{{"Angle", -180, 180, 45}}, Apply: func(src *image.RGBA, r image.Rectangle, p []float64) *image.RGBA {
		return relief(src, r, p[0], false)
	}},
	{ID: "Vignette", Params: []Param{{"Radius", 10, 200, 100}, {"Density", 0, 100, 70}}, Apply: vignette},
}

// FilterByID finds an adjustment or an effect
func FilterByID(id string) *Filter {
	for _, list := range [][]*Filter{Adjustments, Effects} {
		for _, f := range list {
			if f.ID == id {
				return f
			}
		}
	}
	return nil
}

// gaussianBlur approximates the Gaussian blur of the radius by three box
// blurs; the premultiplied colors are averaged, so the transparent pixels do
// not darken the edges. The area around r is read too.
func gaussianBlur(src *image.RGBA, r image.Rectangle, radius float64) *image.RGBA {
	if radius < 0.5 {
		return CropRGBA(src, r)
	}
	sigma := radius / 2
	boxes := boxSizes(sigma, 3)
	pad := 0
	for _, b := range boxes {
		pad += b
	}
	work := r.Inset(-pad).Intersect(src.Rect)
	w, h := work.Dx(), work.Dy()
	a := make([]float32, w*h*4)
	for y := range h {
		si := src.PixOffset(work.Min.X, work.Min.Y+y)
		for i := range w * 4 {
			a[y*w*4+i] = float32(src.Pix[si+i])
		}
	}
	b := make([]float32, len(a))
	for _, rad := range boxes {
		boxBlurH(a, b, w, h, rad)
		boxBlurV(b, a, w, h, rad)
	}
	dst := image.NewRGBA(r)
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			ai := ((y-work.Min.Y)*w + (x - work.Min.X)) * 4
			di := dst.PixOffset(x, y)
			al := min(255, max(0, a[ai+3]+0.5))
			dst.Pix[di+3] = uint8(al)
			for ch := range 3 {
				dst.Pix[di+ch] = uint8(min(al, max(0, a[ai+ch]+0.5)))
			}
		}
	}
	return dst
}

// boxSizes returns the radii of n box blurs that together approximate a Gaussian of sigma
func boxSizes(sigma float64, n int) []int {
	wIdeal := math.Sqrt(12*sigma*sigma/float64(n) + 1)
	wl := int(math.Floor(wIdeal))
	if wl%2 == 0 {
		wl--
	}
	wu := wl + 2
	mIdeal := (12*sigma*sigma - float64(n*wl*wl) - 4*float64(n*wl) - 3*float64(n)) / (-4*float64(wl) - 4)
	m := int(math.Round(mIdeal))
	sizes := make([]int, n)
	for i := range sizes {
		if i < m {
			sizes[i] = (wl - 1) / 2
		} else {
			sizes[i] = (wu - 1) / 2
		}
	}
	return sizes
}

// boxBlurH averages each pixel of src with rad pixels on each side into dst; the edges repeat
func boxBlurH(src, dst []float32, w, h, rad int) {
	if rad <= 0 {
		copy(dst, src)
		return
	}
	k := float32(1) / float32(2*rad+1)
	for y := range h {
		row := y * w * 4
		at := func(x, ch int) float32 { return src[row+max(0, min(w-1, x))*4+ch] }
		for ch := range 4 {
			var sum float32
			for x := -rad; x <= rad; x++ {
				sum += at(x, ch)
			}
			for x := range w {
				dst[row+x*4+ch] = sum * k
				sum += at(x+rad+1, ch) - at(x-rad, ch)
			}
		}
	}
}

func boxBlurV(src, dst []float32, w, h, rad int) {
	if rad <= 0 {
		copy(dst, src)
		return
	}
	k := float32(1) / float32(2*rad+1)
	for x := range w {
		at := func(y, ch int) float32 { return src[(max(0, min(h-1, y))*w+x)*4+ch] }
		for ch := range 4 {
			var sum float32
			for y := -rad; y <= rad; y++ {
				sum += at(y, ch)
			}
			for y := range h {
				dst[(y*w+x)*4+ch] = sum * k
				sum += at(y+rad+1, ch) - at(y-rad, ch)
			}
		}
	}
}

func motionBlur(src *image.RGBA, r image.Rectangle, p []float64) *image.RGBA {
	ang := p[0] * math.Pi / 180
	dist := max(1, p[1])
	n := int(math.Ceil(dist)) + 1
	dx, dy := math.Cos(ang), -math.Sin(ang)
	dst := image.NewRGBA(r)
	b := src.Rect
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			var sum [4]float64
			cnt := 0.0
			for i := range n {
				t := (float64(i)/float64(n-1) - 0.5) * dist
				sx, sy := int(math.Round(float64(x)+dx*t)), int(math.Round(float64(y)+dy*t))
				if sx < b.Min.X || sy < b.Min.Y || sx >= b.Max.X || sy >= b.Max.Y {
					continue
				}
				si := src.PixOffset(sx, sy)
				for ch := range 4 {
					sum[ch] += float64(src.Pix[si+ch])
				}
				cnt++
			}
			di := dst.PixOffset(x, y)
			for ch := range 4 {
				dst.Pix[di+ch] = clamp8(sum[ch] / cnt)
			}
		}
	}
	return dst
}

func addNoise(src *image.RGBA, r image.Rectangle, p []float64) *image.RGBA {
	intensity, sat, coverage := p[0]/100, p[1]/100, p[2]/100
	// The same noise every time the dialog redraws
	rnd := rand.New(rand.NewPCG(uint64(r.Min.X)*31+uint64(r.Min.Y), 7))
	return mapColors(src, r, func(c color.NRGBA) color.NRGBA {
		if rnd.Float64() >= coverage {
			return c
		}
		g := rnd.NormFloat64() * intensity * 64
		dr, dg, db := rnd.NormFloat64(), rnd.NormFloat64(), rnd.NormFloat64()
		k := intensity * 32 * sat
		return color.NRGBA{
			clamp8(float64(c.R) + g + dr*k),
			clamp8(float64(c.G) + g + dg*k),
			clamp8(float64(c.B) + g + db*k),
			c.A,
		}
	})
}

func pixelate(src *image.RGBA, r image.Rectangle, p []float64) *image.RGBA {
	cell := max(1, int(math.Round(p[0])))
	dst := image.NewRGBA(r)
	// The cells are aligned to the image, not to the selection
	for cy := r.Min.Y / cell * cell; cy < r.Max.Y; cy += cell {
		for cx := r.Min.X / cell * cell; cx < r.Max.X; cx += cell {
			cr := image.Rect(cx, cy, cx+cell, cy+cell).Intersect(src.Rect)
			var sum [4]int
			for y := cr.Min.Y; y < cr.Max.Y; y++ {
				for x := cr.Min.X; x < cr.Max.X; x++ {
					i := src.PixOffset(x, y)
					for ch := range 4 {
						sum[ch] += int(src.Pix[i+ch])
					}
				}
			}
			n := cr.Dx() * cr.Dy()
			fill := cr.Intersect(r)
			for y := fill.Min.Y; y < fill.Max.Y; y++ {
				for x := fill.Min.X; x < fill.Max.X; x++ {
					i := dst.PixOffset(x, y)
					for ch := range 4 {
						dst.Pix[i+ch] = uint8(sum[ch] / n)
					}
				}
			}
		}
	}
	return dst
}

// relief convolves with a directional 3x3 kernel: gray for emboss, in color around gray for edge detect
func relief(src *image.RGBA, r image.Rectangle, angle float64, gray bool) *image.RGBA {
	a := angle * math.Pi / 180
	cx, cy := math.Cos(a), -math.Sin(a)
	var k [3][3]float64
	for j := -1; j <= 1; j++ {
		for i := -1; i <= 1; i++ {
			k[j+1][i+1] = -(float64(i)*cx + float64(j)*cy)
		}
	}
	b := src.Rect
	at := func(x, y int) color.NRGBA {
		x, y = max(b.Min.X, min(b.Max.X-1, x)), max(b.Min.Y, min(b.Max.Y-1, y))
		i := src.PixOffset(x, y)
		return Unpremul(color.RGBA{src.Pix[i], src.Pix[i+1], src.Pix[i+2], src.Pix[i+3]})
	}
	dst := image.NewRGBA(r)
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			var s [3]float64
			for j := -1; j <= 1; j++ {
				for i := -1; i <= 1; i++ {
					c := at(x+i, y+j)
					w := k[j+1][i+1]
					if gray {
						s[0] += luma(c) * w
					} else {
						s[0] += float64(c.R) * w
						s[1] += float64(c.G) * w
						s[2] += float64(c.B) * w
					}
				}
			}
			o := at(x, y)
			var c color.NRGBA
			if gray {
				v := clamp8(128 + s[0])
				c = color.NRGBA{v, v, v, o.A}
			} else {
				c = color.NRGBA{clamp8(128 + s[0]), clamp8(128 + s[1]), clamp8(128 + s[2]), o.A}
			}
			pc := Premul(c)
			i := dst.PixOffset(x, y)
			dst.Pix[i], dst.Pix[i+1], dst.Pix[i+2], dst.Pix[i+3] = pc.R, pc.G, pc.B, pc.A
		}
	}
	return dst
}

// vignette darkens towards the corners of the image
func vignette(src *image.RGBA, r image.Rectangle, p []float64) *image.RGBA {
	b := src.Rect
	cx, cy := float64(b.Min.X+b.Max.X)/2, float64(b.Min.Y+b.Max.Y)/2
	rad := math.Hypot(float64(b.Dx()), float64(b.Dy())) / 2 * p[0] / 100
	density := p[1] / 100
	dst := image.NewRGBA(r)
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			d := math.Hypot(float64(x)+0.5-cx, float64(y)+0.5-cy) / rad
			k := 1 - density*smoothstep(0.3, 1, d)
			i := src.PixOffset(x, y)
			for ch := range 3 {
				dst.Pix[i+ch] = clamp8(float64(src.Pix[i+ch]) * k)
			}
			dst.Pix[i+3] = src.Pix[i+3]
		}
	}
	return dst
}

func smoothstep(e0, e1, x float64) float64 {
	t := max(0, min(1, (x-e0)/(e1-e0)))
	return t * t * (3 - 2*t)
}

func rgbToHSL(c color.NRGBA) (h, s, l float64) {
	r, g, b := float64(c.R)/255, float64(c.G)/255, float64(c.B)/255
	mx, mn := max(r, g, b), min(r, g, b)
	l = (mx + mn) / 2
	if mx == mn {
		return 0, 0, l
	}
	d := mx - mn
	if l > 0.5 {
		s = d / (2 - mx - mn)
	} else {
		s = d / (mx + mn)
	}
	switch mx {
	case r:
		h = (g - b) / d
		if g < b {
			h += 6
		}
	case g:
		h = (b-r)/d + 2
	default:
		h = (r-g)/d + 4
	}
	return h / 6, s, l
}

func hslToRGB(h, s, l float64) color.NRGBA {
	if s == 0 {
		v := clamp8(l * 255)
		return color.NRGBA{v, v, v, 255}
	}
	var q float64
	if l < 0.5 {
		q = l * (1 + s)
	} else {
		q = l + s - l*s
	}
	p := 2*l - q
	hue := func(t float64) float64 {
		t = math.Mod(t+1, 1)
		switch {
		case t < 1.0/6:
			return p + (q-p)*6*t
		case t < 0.5:
			return q
		case t < 2.0/3:
			return p + (q-p)*(2.0/3-t)*6
		}
		return p
	}
	return color.NRGBA{clamp8(hue(h+1.0/3) * 255), clamp8(hue(h) * 255), clamp8(hue(h-1.0/3) * 255), 255}
}
