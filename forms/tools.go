package forms

import (
	"image/color"

	"github.com/ipoluianov/altpaint/config"
	"github.com/ipoluianov/altpaint/paint"
	"github.com/ipoluianov/nui/ui"
)

// ToolID names a tool; it is also the id of its history steps and its name in the strings
type ToolID string

const (
	ToolRectSelect    ToolID = "RectangleSelect"
	ToolEllipseSelect ToolID = "EllipseSelect"
	ToolLassoSelect   ToolID = "LassoSelect"
	ToolMagicWand     ToolID = "MagicWand"
	ToolMoveSelected  ToolID = "MoveSelected"
	ToolMoveSelection ToolID = "MoveSelection"
	ToolZoom          ToolID = "Zoom"
	ToolPan           ToolID = "Pan"
	ToolPaintBucket   ToolID = "PaintBucket"
	ToolGradient      ToolID = "Gradient"
	ToolPaintbrush    ToolID = "Paintbrush"
	ToolEraser        ToolID = "Eraser"
	ToolPencil        ToolID = "Pencil"
	ToolColorPicker   ToolID = "ColorPicker"
	ToolCloneStamp    ToolID = "CloneStamp"
	ToolText          ToolID = "Text"
	ToolLine          ToolID = "Line"
	ToolShapes        ToolID = "Shapes"
)

// Option groups of the tool bar a tool shows
const (
	optWidth = 1 << iota
	optAntialias
	optBlend
	optShape
	optFill
	optGradient
	optTolerance
	optFlood
	optSampling
	optSelMode
	optFont
)

// toolDef describes a tool of the tools panel
type toolDef struct {
	id   ToolID
	icon string
	// key selects the tool; pressed again, it goes to the next tool with the same key
	key  string
	opts int
}

// toolDefs are the tools in the order of the tools panel, two in a row
var toolDefs = []toolDef{
	{ToolRectSelect, "tool-rect-select", "S", optSelMode},
	{ToolMoveSelected, "tool-move-selected", "M", 0},
	{ToolLassoSelect, "tool-lasso", "S", optSelMode | optAntialias},
	{ToolMoveSelection, "tool-move-selection", "M", 0},
	{ToolEllipseSelect, "tool-ellipse-select", "S", optSelMode | optAntialias},
	{ToolZoom, "tool-zoom", "Z", 0},
	{ToolMagicWand, "tool-magic-wand", "S", optSelMode | optTolerance | optFlood | optSampling},
	{ToolPan, "tool-pan", "H", 0},
	{ToolPaintBucket, "tool-bucket", "F", optTolerance | optFlood | optSampling | optBlend | optAntialias},
	{ToolGradient, "tool-gradient", "G", optGradient},
	{ToolPaintbrush, "tool-brush", "B", optWidth | optAntialias | optBlend},
	{ToolEraser, "tool-eraser", "E", optWidth | optAntialias},
	{ToolPencil, "tool-pencil", "P", optBlend},
	{ToolColorPicker, "tool-picker", "K", optSampling},
	{ToolCloneStamp, "tool-clone", "L", optWidth | optAntialias},
	{ToolText, "tool-text", "T", optFont | optAntialias | optBlend},
	{ToolLine, "tool-line", "O", optWidth | optAntialias | optBlend},
	{ToolShapes, "tool-shapes", "O", optWidth | optAntialias | optBlend | optShape | optFill},
}

func toolDefOf(id ToolID) toolDef {
	for _, t := range toolDefs {
		if t.id == id {
			return t
		}
	}
	return toolDefs[0]
}

// Shape kinds of the Shapes tool
const (
	shapeRectangle = iota
	shapeRoundedRectangle
	shapeEllipse
)

// Fill modes of the Shapes tool
const (
	fillOutline     = iota // the outline in the primary color
	fillInterior           // filled with the primary color
	fillOutlineBoth        // the outline in the primary, filled with the secondary color
)

// ToolState is the current tool, its options and the colors, shared by all the images
type ToolState struct {
	Tool      ToolID
	Primary   color.NRGBA
	Secondary color.NRGBA
	Width     float64
	Antialias bool
	Blend     bool
	Shape     int
	Fill      int
	Gradient  paint.GradientKind
	Tolerance float64 // 0..100
	Global    bool    // flood: the whole image, not just the contiguous area
	SampleAll bool    // the fill, the wand and the picker look at the image, not the layer
	SelMode   paint.CombineMode
	Text      paint.TextStyle
}

// tools is the state of the tools
var tools ToolState

// loadTools restores the tools from the settings
func loadTools(s config.ToolSettings) {
	parse := func(hex string, def color.NRGBA) color.NRGBA {
		c, ok := ui.ParseHexColor(hex)
		if !ok {
			return def
		}
		return color.NRGBA(c)
	}
	tools = ToolState{
		Tool:      ToolID(s.Tool),
		Primary:   parse(s.Primary, color.NRGBA{0, 0, 0, 255}),
		Secondary: parse(s.Secondary, color.NRGBA{255, 255, 255, 255}),
		Width:     max(1, s.Width),
		Antialias: s.Antialias,
		Blend:     s.Blend,
		Shape:     s.Shape,
		Fill:      s.Fill,
		Gradient:  paint.GradientKind(s.Gradient),
		Tolerance: max(0, min(100, s.Tolerance)),
		Global:    s.FloodMode == 1,
		SampleAll: s.SampleAll,
		Text: paint.TextStyle{
			Size:      max(4, s.FontSize),
			Bold:      s.Bold,
			Italic:    s.Italic,
			Mono:      s.Mono,
			Antialias: s.Antialias,
		},
	}
	if toolDefOf(tools.Tool).id != tools.Tool {
		tools.Tool = ToolPaintbrush
	}
}

// saveTools returns the tools as the settings keep them
func saveTools() config.ToolSettings {
	hex := func(c color.NRGBA) string { return ui.ColorToHex(color.RGBA(c)) }
	flood := 0
	if tools.Global {
		flood = 1
	}
	return config.ToolSettings{
		Tool:      string(tools.Tool),
		Primary:   hex(tools.Primary),
		Secondary: hex(tools.Secondary),
		Width:     tools.Width,
		Antialias: tools.Antialias,
		Blend:     tools.Blend,
		Fill:      tools.Fill,
		Shape:     tools.Shape,
		Gradient:  int(tools.Gradient),
		Tolerance: tools.Tolerance,
		FloodMode: flood,
		SampleAll: tools.SampleAll,
		FontSize:  tools.Text.Size,
		Bold:      tools.Text.Bold,
		Italic:    tools.Text.Italic,
		Mono:      tools.Text.Mono,
	}
}

// defaultPalette is the palette: 48 colors, 16 in a row
var defaultPalette = func() []color.NRGBA {
	hexes := []string{
		"#000000", "#404040", "#FF0000", "#FF6A00", "#FFD800", "#B6FF00", "#4CFF00", "#00FF21",
		"#00FF90", "#00FFFF", "#0094FF", "#0026FF", "#4800FF", "#B200FF", "#FF00DC", "#FF006E",
		"#FFFFFF", "#808080", "#7F0000", "#7F3300", "#7F6A00", "#5B7F00", "#267F00", "#007F0E",
		"#007F46", "#007F7F", "#004A7F", "#00137F", "#21007F", "#57007F", "#7F006E", "#7F0037",
		"#A0A0A0", "#303030", "#FF7F7F", "#FFB27F", "#FFE97F", "#DAFF7F", "#A5FF7F", "#7FFF8E",
		"#7FFFC5", "#7FFFFF", "#7FC9FF", "#7F92FF", "#A17FFF", "#D67FFF", "#FF7FED", "#FF7FB6",
	}
	p := make([]color.NRGBA, len(hexes))
	for i, h := range hexes {
		p[i] = color.NRGBA(ui.ColorFromHex(h))
	}
	return p
}()
