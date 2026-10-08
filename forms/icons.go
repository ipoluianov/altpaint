package forms

import (
	"bytes"
	"embed"
	"image"
	"image/color"
	"image/draw"
	"image/png"

	"github.com/ipoluianov/nui/ui"
)

// icons/*.png are rendered from the matching icons/*.svg by scripts/render-icons.sh:
// <name>.png is 24x24 for the tool bars, <name>-16.png 16x16 for the menus
//
//go:embed icons/*.png
var iconsFS embed.FS

// The icons are drawn in one light color for the dark theme; in the light
// theme they are shown in this dark one
var iconColorLight = ui.ColorFromHex("#455a64")

// themedIcons set the icons again when the theme changes, see setIcon
var themedIcons []func()

// loadIcon returns the icon in the colors of the current theme
func loadIcon(name string) image.Image {
	data, err := iconsFS.ReadFile("icons/" + name + ".png")
	if err != nil {
		return nil
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil
	}
	if ui.IsDarkTheme {
		return img
	}
	return tintIcon(img, iconColorLight)
}

// setIcon sets the icon with set now and again on every theme change
func setIcon(name string, set func(img image.Image)) {
	apply := func() { set(loadIcon(name)) }
	apply()
	themedIcons = append(themedIcons, apply)
}

// tintIcon paints a one-color icon in col, keeping its transparency.
// Pixels of another color (an accent) are kept.
func tintIcon(img image.Image, col color.RGBA) image.Image {
	src := image.NewNRGBA(img.Bounds())
	draw.Draw(src, src.Rect, img, img.Bounds().Min, draw.Src)
	for i := 0; i < len(src.Pix); i += 4 {
		r, g, b := src.Pix[i], src.Pix[i+1], src.Pix[i+2]
		gray := max(r, g, b)-min(r, g, b) < 24
		if gray {
			src.Pix[i], src.Pix[i+1], src.Pix[i+2] = col.R, col.G, col.B
		}
	}
	return src
}
