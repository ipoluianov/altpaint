package forms

import (
	"image/color"

	"github.com/ipoluianov/nui/ui"
)

// The color themes offered in the settings (config.Settings.Theme)
const (
	themeDark  = ""
	themeLight = "light"
)

// themeColors is a color of the application in the dark and the light theme
type themeColors struct {
	dark, light color.RGBA
}

// get returns the color for the current theme
func (c themeColors) get() color.RGBA {
	if ui.IsDarkTheme {
		return c.dark
	}
	return c.light
}

var (
	colorLink  = themeColors{ui.ColorFromHex("#3d8bf2"), ui.ColorFromHex("#1d6fb5")}
	colorMuted = themeColors{ui.ColorFromHex("#8c8c8c"), ui.ColorFromHex("#6b6b6b")}
	// The area around the image
	colorWorkspace = themeColors{ui.ColorFromHex("#1b1b1b"), ui.ColorFromHex("#c8c8c8")}
	colorImageEdge = themeColors{ui.ColorFromHex("#000000"), ui.ColorFromHex("#808080")}
	// The squares that show transparency
	colorCheckerLight = color.RGBA{0xff, 0xff, 0xff, 0xff}
	colorCheckerDark  = color.RGBA{0xcc, 0xcc, 0xcc, 0xff}
)

// linkLabels are recolored when the theme changes, see newLinkLabel
var linkLabels []*ui.Label

// themeListeners are called when the theme changes
var themeListeners []func()

// ApplyTheme switches the application to the theme of the settings and
// repaints the open windows
func ApplyTheme(theme string) {
	if theme == themeLight {
		ui.ApplyLightTheme()
	} else {
		ui.ApplyDarkTheme()
	}
	for _, lbl := range linkLabels {
		lbl.SetForegroundColor(colorLink.get())
	}
	for _, setIcon := range themedIcons {
		setIcon()
	}
	for _, f := range themeListeners {
		f()
	}
}
