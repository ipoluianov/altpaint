package paint

import (
	"image"
	"image/color"
	"image/draw"
	"sync"
)

// BlendMode is how a layer mixes with the layers under it
type BlendMode int

// The modes in the order they are offered
const (
	BlendNormal BlendMode = iota
	BlendMultiply
	BlendAdditive
	BlendColorBurn
	BlendColorDodge
	BlendReflect
	BlendGlow
	BlendOverlay
	BlendDifference
	BlendNegation
	BlendLighten
	BlendDarken
	BlendScreen
	BlendXor
	blendModeCount
)

// BlendModes lists the modes in the order they are offered
func BlendModes() []BlendMode {
	modes := make([]BlendMode, blendModeCount)
	for i := range modes {
		modes[i] = BlendMode(i)
	}
	return modes
}

// blendFuncs mix a straight channel of the layer under (b) and of the layer over it (s)
var blendFuncs = [blendModeCount]func(b, s int) int{
	BlendMultiply: func(b, s int) int { return (b*s + 127) / 255 },
	BlendAdditive: func(b, s int) int { return min(255, b+s) },
	BlendColorBurn: func(b, s int) int {
		if b == 255 {
			return 255
		}
		if s == 0 {
			return 0
		}
		return max(0, 255-(255-b)*255/s)
	},
	BlendColorDodge: func(b, s int) int {
		if b == 0 {
			return 0
		}
		if s == 255 {
			return 255
		}
		return min(255, b*255/(255-s))
	},
	BlendReflect: func(b, s int) int {
		if s == 255 {
			return 255
		}
		return min(255, b*b/(255-s))
	},
	BlendGlow: func(b, s int) int {
		if b == 255 {
			return 255
		}
		return min(255, s*s/(255-b))
	},
	BlendOverlay: func(b, s int) int {
		if b < 128 {
			return 2 * b * s / 255
		}
		return 255 - 2*(255-b)*(255-s)/255
	},
	BlendDifference: func(b, s int) int { return max(b-s, s-b) },
	BlendNegation: func(b, s int) int {
		d := 255 - b - s
		return 255 - max(d, -d)
	},
	BlendLighten: func(b, s int) int { return max(b, s) },
	BlendDarken:  func(b, s int) int { return min(b, s) },
	BlendScreen:  func(b, s int) int { return b + s - (b*s+127)/255 },
	BlendXor:     func(b, s int) int { return b ^ s },
}

// blendTables are the blend functions computed for all the pairs of values, made when first needed
var (
	blendTablesMtx sync.Mutex
	blendTables    [blendModeCount]*[256 * 256]uint8
)

func blendTable(mode BlendMode) *[256 * 256]uint8 {
	blendTablesMtx.Lock()
	defer blendTablesMtx.Unlock()
	if t := blendTables[mode]; t != nil {
		return t
	}
	t := new([256 * 256]uint8)
	f := blendFuncs[mode]
	for b := range 256 {
		for s := range 256 {
			t[b<<8|s] = uint8(max(0, min(255, f(b, s))))
		}
	}
	blendTables[mode] = t
	return t
}

// BlendLayer draws the area r of the layer image src over dst with the opacity and the mode
func BlendLayer(dst, src *image.RGBA, r image.Rectangle, opacity uint8, mode BlendMode) {
	r = r.Intersect(dst.Rect).Intersect(src.Rect)
	if r.Empty() || opacity == 0 {
		return
	}
	if mode == BlendNormal || mode < 0 || mode >= blendModeCount {
		var mask image.Image
		if opacity < 255 {
			mask = image.NewUniform(color.Alpha{opacity})
		}
		draw.DrawMask(dst, r, src, r.Min, mask, image.Point{}, draw.Over)
		return
	}
	table := blendTable(mode)
	op := uint32(opacity)
	for y := r.Min.Y; y < r.Max.Y; y++ {
		di := dst.PixOffset(r.Min.X, y)
		si := src.PixOffset(r.Min.X, y)
		for x := r.Min.X; x < r.Max.X; x, di, si = x+1, di+4, si+4 {
			sa := uint32(src.Pix[si+3]) * op / 255
			if sa == 0 {
				continue
			}
			d := dst.Pix[di : di+4 : di+4]
			s := src.Pix[si : si+4 : si+4]
			ba := uint32(d[3])
			if ba == 0 {
				d[0] = uint8(uint32(s[0]) * op / 255)
				d[1] = uint8(uint32(s[1]) * op / 255)
				d[2] = uint8(uint32(s[2]) * op / 255)
				d[3] = uint8(sa)
				continue
			}
			ssa := uint32(s[3]) // the alpha the color of the source is premultiplied with
			for ch := range 3 {
				cs := uint32(s[ch]) * op / 255 // premultiplied by sa
				cb := uint32(d[ch])
				// Straight values for the blend function
				sb := min(255, uint32(s[ch])*255/ssa)
				bb := min(255, cb*255/ba)
				m := uint32(table[bb<<8|sb])
				v := ((255-ba)*cs + (255-sa)*cb + sa*ba*m/255) / 255
				d[ch] = uint8(min(v, 255))
			}
			d[3] = uint8(sa + ba - (sa*ba+127)/255)
		}
	}
}
