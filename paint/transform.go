package paint

import (
	"image"
	"image/draw"

	xdraw "golang.org/x/image/draw"
)

// Resampling is how the pixels are computed when an image is resized
type Resampling int

const (
	ResampleBest Resampling = iota // Catmull-Rom
	ResampleBilinear
	ResampleNearest
)

func (r Resampling) scaler() xdraw.Interpolator {
	switch r {
	case ResampleBilinear:
		return xdraw.BiLinear
	case ResampleNearest:
		return xdraw.NearestNeighbor
	}
	return xdraw.CatmullRom
}

// ResizeRGBA returns the image scaled to w x h
func ResizeRGBA(src *image.RGBA, w, h int, method Resampling) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	method.scaler().Scale(dst, dst.Rect, src, src.Rect, xdraw.Src, nil)
	return dst
}

// ResizeImage scales all the layers to w x h
func (d *Document) ResizeImage(name string, w, h int, method Resampling) {
	if w == d.W && h == d.H || w <= 0 || h <= 0 {
		return
	}
	d.Do(name, func() {
		for _, l := range d.Layers {
			l.Img = ResizeRGBA(l.Img, w, h, method)
		}
		d.W, d.H = w, h
		d.Sel = nil
	})
}

// Anchor is where the image stays when the canvas is resized: -1, 0, 1 for left/top, center, right/bottom
type Anchor struct {
	X, Y int
}

// ResizeCanvas changes the size of the image without scaling it: the
// layers are cut or extended with transparency (the bottom one with bg)
func (d *Document) ResizeCanvas(name string, w, h int, anchor Anchor, bg *image.Uniform) {
	if w == d.W && h == d.H || w <= 0 || h <= 0 {
		return
	}
	off := image.Pt(anchorOffset(d.W, w, anchor.X), anchorOffset(d.H, h, anchor.Y))
	d.Do(name, func() {
		for i, l := range d.Layers {
			img := image.NewRGBA(image.Rect(0, 0, w, h))
			if i == 0 && bg != nil {
				draw.Draw(img, img.Rect, bg, image.Point{}, draw.Src)
			}
			draw.Draw(img, l.Img.Rect.Add(off), l.Img, image.Point{}, draw.Src)
			l.Img = img
		}
		d.W, d.H = w, h
		if d.Sel != nil {
			d.Sel = d.Sel.Resized(w, h, off)
		}
	})
}

func anchorOffset(old, size, anchor int) int {
	switch {
	case anchor < 0:
		return 0
	case anchor > 0:
		return size - old
	}
	return (size - old) / 2
}

// CropToSelection cuts the image to the rectangle around the selection;
// the pixels outside of a selection of another shape become transparent
func (d *Document) CropToSelection(name string) {
	if d.Sel == nil {
		return
	}
	sel := d.Sel
	r := sel.Bounds()
	rectangular := true
	for y := r.Min.Y; y < r.Max.Y && rectangular; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			if sel.Mask.Pix[sel.Mask.PixOffset(x, y)] != 255 {
				rectangular = false
				break
			}
		}
	}
	d.Do(name, func() {
		for _, l := range d.Layers {
			img := image.NewRGBA(image.Rect(0, 0, r.Dx(), r.Dy()))
			if rectangular {
				draw.Draw(img, img.Rect, l.Img, r.Min, draw.Src)
			} else {
				draw.DrawMask(img, img.Rect, l.Img, r.Min, sel.Mask, r.Min, draw.Src)
			}
			l.Img = img
		}
		d.W, d.H = r.Dx(), r.Dy()
		d.Sel = nil
	})
}

// FlipRGBA returns the image mirrored left to right (horizontal) or top to bottom
func FlipRGBA(src *image.RGBA, horizontal bool) *image.RGBA {
	b := src.Rect
	dst := image.NewRGBA(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			sx, sy := x, y
			if horizontal {
				sx = b.Max.X - 1 - (x - b.Min.X)
			} else {
				sy = b.Max.Y - 1 - (y - b.Min.Y)
			}
			copy(dst.Pix[dst.PixOffset(x, y):dst.PixOffset(x, y)+4], src.Pix[src.PixOffset(sx, sy):src.PixOffset(sx, sy)+4])
		}
	}
	return dst
}

// RotateRGBA returns the image turned clockwise by quarter turns (1..3)
func RotateRGBA(src *image.RGBA, quarters int) *image.RGBA {
	quarters = ((quarters % 4) + 4) % 4
	w, h := src.Rect.Dx(), src.Rect.Dy()
	if quarters%2 == 1 {
		w, h = h, w
	}
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	sw, sh := src.Rect.Dx(), src.Rect.Dy()
	for y := range sh {
		for x := range sw {
			var dx, dy int
			switch quarters {
			case 0:
				dx, dy = x, y
			case 1:
				dx, dy = sh-1-y, x
			case 2:
				dx, dy = sw-1-x, sh-1-y
			case 3:
				dx, dy = y, sw-1-x
			}
			si := src.PixOffset(src.Rect.Min.X+x, src.Rect.Min.Y+y)
			di := dst.PixOffset(dx, dy)
			copy(dst.Pix[di:di+4], src.Pix[si:si+4])
		}
	}
	return dst
}

// FlipImage mirrors all the layers
func (d *Document) FlipImage(name string, horizontal bool) {
	d.Do(name, func() {
		for _, l := range d.Layers {
			l.Img = FlipRGBA(l.Img, horizontal)
		}
		d.Sel = flipSelection(d.Sel, horizontal)
	})
}

// FlipLayer mirrors the current layer
func (d *Document) FlipLayer(name string, horizontal bool) {
	d.Do(name, func() {
		l := d.Layer()
		l.Img = FlipRGBA(l.Img, horizontal)
	})
}

func flipSelection(s *Selection, horizontal bool) *Selection {
	if s == nil {
		return nil
	}
	return NewSelection(alphaFromRGBA(FlipRGBA(rgbaFromAlpha(s.Mask), horizontal)))
}

// RotateImage turns all the layers clockwise by quarter turns
func (d *Document) RotateImage(name string, quarters int) {
	d.Do(name, func() {
		for _, l := range d.Layers {
			l.Img = RotateRGBA(l.Img, quarters)
		}
		if quarters%2 != 0 {
			d.W, d.H = d.H, d.W
		}
		if d.Sel != nil {
			d.Sel = NewSelection(alphaFromRGBA(RotateRGBA(rgbaFromAlpha(d.Sel.Mask), quarters)))
		}
	})
}

// The selection masks are turned with the same code as the images
func rgbaFromAlpha(m *image.Alpha) *image.RGBA {
	img := image.NewRGBA(m.Rect)
	for i, v := range m.Pix {
		img.Pix[i*4+3] = v
	}
	return img
}

func alphaFromRGBA(img *image.RGBA) *image.Alpha {
	m := image.NewAlpha(img.Rect)
	for i := range m.Pix {
		m.Pix[i] = img.Pix[i*4+3]
	}
	return m
}

// AddLayer adds a transparent layer above the current one and makes it current
func (d *Document) AddLayer(name, layerName string) {
	d.Do(name, func() {
		l := NewLayer(layerName, d.W, d.H)
		d.insertLayer(d.Current+1, l)
		d.Current++
	})
}

// InsertLayer adds the layer above the current one and makes it current
func (d *Document) InsertLayer(name string, l *Layer) {
	d.Do(name, func() {
		d.insertLayer(d.Current+1, l)
		d.Current++
	})
}

func (d *Document) insertLayer(i int, l *Layer) {
	d.Layers = append(d.Layers[:i], append([]*Layer{l}, d.Layers[i:]...)...)
}

// DeleteLayer removes the current layer; the last one stays
func (d *Document) DeleteLayer(name string) {
	if len(d.Layers) < 2 {
		return
	}
	d.Do(name, func() {
		d.Layers = append(d.Layers[:d.Current:d.Current], d.Layers[d.Current+1:]...)
		d.Current = max(0, d.Current-1)
	})
}

// DuplicateLayer adds a copy of the current layer above it
func (d *Document) DuplicateLayer(name, copySuffix string) {
	d.Do(name, func() {
		l := d.Layer().Clone()
		l.Name += copySuffix
		d.insertLayer(d.Current+1, l)
		d.Current++
	})
}

// MergeDown draws the current layer on the one below it and removes it
func (d *Document) MergeDown(name string) {
	if d.Current == 0 {
		return
	}
	d.Do(name, func() {
		top, below := d.Layers[d.Current], d.Layers[d.Current-1]
		img := CloneRGBA(below.Img)
		if top.Visible {
			BlendLayer(img, top.Img, img.Rect, top.Opacity, top.Blend)
		}
		below.Img = img
		d.Layers = append(d.Layers[:d.Current:d.Current], d.Layers[d.Current+1:]...)
		d.Current--
	})
}

// Flatten merges all the layers into one
func (d *Document) Flatten(name string) {
	if len(d.Layers) < 2 {
		return
	}
	d.Do(name, func() {
		img := d.Flattened()
		bottom := d.Layers[0]
		bottom.Img = img
		bottom.Visible = true
		bottom.Opacity = 255
		bottom.Blend = BlendNormal
		d.Layers = []*Layer{bottom}
		d.Current = 0
	})
}

// MoveLayer moves the current layer up (+1) or down (-1) the stack
func (d *Document) MoveLayer(name string, dir int) {
	j := d.Current + dir
	if j < 0 || j >= len(d.Layers) {
		return
	}
	d.Do(name, func() {
		d.Layers[d.Current], d.Layers[j] = d.Layers[j], d.Layers[d.Current]
		d.Current = j
	})
}

// SetLayerProps changes the properties of the layer
func (d *Document) SetLayerProps(name string, l *Layer, layerName string, visible bool, opacity uint8, blend BlendMode) {
	if l.Name == layerName && l.Visible == visible && l.Opacity == opacity && l.Blend == blend {
		return
	}
	d.Do(name, func() {
		l.Name, l.Visible, l.Opacity, l.Blend = layerName, visible, opacity, blend
	})
}
