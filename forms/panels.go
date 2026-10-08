package forms

import (
	"image"
	"image/color"

	"github.com/ipoluianov/altpaint/paint"
	"github.com/ipoluianov/nui/ui"
	xdraw "golang.org/x/image/draw"
)

// toolButtonSize is the size of the buttons of the tools panel and the small tool bars
const toolButtonSize = 34

// ToolsPanel is the column of the tools on the left, two in a row
type ToolsPanel struct {
	ui.Widget
	buttons map[ToolID]*ui.ToolButton
}

func NewToolsPanel(onSelect func(id ToolID)) *ToolsPanel {
	var c ToolsPanel
	c.InitWidget()
	c.SetPanelPadding(2)
	c.SetCellPadding(1)
	c.buttons = make(map[ToolID]*ui.ToolButton)
	for i, t := range toolDefs {
		btn := ui.NewToolButton(nil, "", func() { onSelect(t.id) })
		btn.SetButtonSize(toolButtonSize, toolButtonSize)
		btn.SetFlat(true)
		setIcon(t.icon, btn.SetImage)
		btn.SetTooltipFunc(func() string { return T().ToolName(t.id) + " (" + t.key + ")" })
		c.buttons[t.id] = btn
		c.AddWidget(i/2, i%2, btn)
	}
	c.AddWidget(len(toolDefs)/2+1, 0, ui.NewVSpacer())
	return &c
}

// SetTool shows the tool as chosen
func (c *ToolsPanel) SetTool(id ToolID) {
	for tid, btn := range c.buttons {
		btn.SetChecked(tid == id)
	}
}

// ---- Colors ----

// ColorsPanel shows the primary and the secondary colors and the palette
type ColorsPanel struct {
	ui.Widget
	main      *MainForm
	primary   *ui.ColorPicker
	secondary *ui.ColorPicker
}

const (
	paletteColumns = 16
	paletteCell    = 13
)

func NewColorsPanel(main *MainForm) *ColorsPanel {
	var c ColorsPanel
	c.InitWidget()
	c.main = main
	c.SetPanelPadding(4)

	// The labels are above the colors, so a long translation does not widen the panel
	lblPrimary := ui.NewLabel("")
	lblPrimary.SetXExpandable(true)
	lblPrimary.SetTextFunc(func() string { return T().Primary })
	lblSecondary := ui.NewLabel("")
	lblSecondary.SetXExpandable(true)
	lblSecondary.SetTextFunc(func() string { return T().Secondary })
	c.primary = ui.NewColorPicker()
	c.primary.SetAlphaEnabled(true)
	c.primary.SetOnColorChanged(func(col color.RGBA) { main.setColor(color.NRGBA(col), true) })
	c.secondary = ui.NewColorPicker()
	c.secondary.SetAlphaEnabled(true)
	c.secondary.SetOnColorChanged(func(col color.RGBA) { main.setColor(color.NRGBA(col), false) })

	btnSwap := ui.NewToolButton(nil, "", main.swapColors)
	btnSwap.SetButtonSize(28, 28)
	btnSwap.SetFlat(true)
	setIcon("swap", btnSwap.SetImage)
	btnSwap.SetTooltipFunc(func() string { return T().SwapColors + " (X)" })
	btnReset := ui.NewToolButton(nil, "", main.resetColors)
	btnReset.SetButtonSize(28, 28)
	btnReset.SetFlat(true)
	setIcon("reset-colors", btnReset.SetImage)
	btnReset.SetTooltipFunc(func() string { return T().ResetColors + " (D)" })

	grid := ui.NewPanel()
	grid.SetPanelPadding(0)
	c.primary.SetMinWidth(120)
	c.secondary.SetMinWidth(120)
	grid.AddWidget(0, 0, lblPrimary)
	grid.AddWidget(1, 0, c.primary)
	grid.AddWidget(1, 1, btnSwap)
	grid.AddWidget(2, 0, lblSecondary)
	grid.AddWidget(3, 0, c.secondary)
	grid.AddWidget(3, 1, btnReset)
	c.AddWidget(0, 0, grid)

	palette := &ui.Widget{}
	palette.InitWidget()
	rows := (len(defaultPalette) + paletteColumns - 1) / paletteColumns
	palette.SetMinSize(paletteColumns*paletteCell+1, rows*paletteCell+1)
	palette.SetMaxSize(paletteColumns*paletteCell+1, rows*paletteCell+1)
	palette.SetOnPaint(func(cnv *ui.Canvas) {
		for i, col := range defaultPalette {
			x, y := i%paletteColumns*paletteCell, i/paletteColumns*paletteCell
			cnv.FillRect(x, y, paletteCell, paletteCell, ui.CurrentPalette().Border)
			cnv.FillRect(x+1, y+1, paletteCell-1, paletteCell-1, color.RGBA(col))
		}
	})
	palette.SetTooltipFunc(func() string { return T().PaletteHint })
	palette.SetOnMouseDown(func(button ui.MouseButton, x, y int, mods ui.KeyModifiers) bool {
		i := y/paletteCell*paletteColumns + x/paletteCell
		if x < paletteColumns*paletteCell && i >= 0 && i < len(defaultPalette) {
			main.setColor(defaultPalette[i], button != ui.MouseButtonRight)
		}
		return true
	})
	c.AddWidget(1, 0, palette)
	return &c
}

// Refresh shows the current colors
func (c *ColorsPanel) Refresh() {
	if color.NRGBA(c.primary.Color()) != tools.Primary {
		c.primary.SetColor(color.RGBA(tools.Primary))
	}
	if color.NRGBA(c.secondary.Color()) != tools.Secondary {
		c.secondary.SetColor(color.RGBA(tools.Secondary))
	}
}

// ---- Layers ----

const (
	layerRowHeight = 44
	layerThumbSize = 36
	layerCheckSize = 14
)

// LayersPanel lists the layers of the image, the top one first
type LayersPanel struct {
	ui.Widget
	main *MainForm
	list *ui.Widget

	thumbs map[*paint.Layer]layerThumb
}

type layerThumb struct {
	version int
	img     image.Image
}

func NewLayersPanel(main *MainForm) *LayersPanel {
	var c LayersPanel
	c.InitWidget()
	c.main = main
	c.SetPanelPadding(0)
	c.thumbs = make(map[*paint.Layer]layerThumb)

	c.list = &ui.Widget{}
	c.list.InitWidget()
	c.list.SetXExpandable(true)
	c.list.SetYExpandable(true)
	c.list.SetAllowScroll(false, true)
	c.list.SetAutoFillBackground(true)
	c.list.SetRole("base")
	c.list.SetOnPaint(c.paint)
	c.list.SetOnMouseDown(c.mouseDown)
	c.list.SetOnMouseDblClick(func(button ui.MouseButton, x, y int, mods ui.KeyModifiers) bool {
		if button == ui.MouseButtonLeft && x > layerCheckSize+12 {
			main.layerProperties()
		}
		return true
	})
	c.AddWidget(0, 0, c.list)

	bar := ui.NewPanel()
	bar.SetPanelPadding(0)
	bar.SetCellPadding(0)
	buttons := []struct {
		icon string
		tip  func() string
		f    func()
	}{
		{"layer-add", func() string { return T().CmdAddLayer + " (Ctrl+Shift+N)" }, main.addLayer},
		{"layer-delete", func() string { return T().CmdDeleteLayer + " (Ctrl+Shift+Del)" }, main.deleteLayer},
		{"layer-duplicate", func() string { return T().CmdDuplicateLayer + " (Ctrl+Shift+D)" }, main.duplicateLayer},
		{"layer-merge", func() string { return T().CmdMergeDown + " (Ctrl+M)" }, main.mergeDown},
		{"layer-up", func() string { return T().CmdLayerUp }, func() { main.moveLayer(1) }},
		{"layer-down", func() string { return T().CmdLayerDown }, func() { main.moveLayer(-1) }},
		{"layer-props", func() string { return T().CmdLayerProperties + " (F4)" }, main.layerProperties},
	}
	for i, b := range buttons {
		btn := ui.NewToolButton(nil, "", b.f)
		btn.SetButtonSize(30, 30)
		btn.SetFlat(true)
		setIcon(b.icon, btn.SetImage)
		btn.SetTooltipFunc(b.tip)
		bar.AddWidget(0, i, btn)
	}
	bar.AddWidget(0, len(buttons), ui.NewHSpacer())
	c.AddWidget(1, 0, bar)
	return &c
}

// Refresh lays the list out again for the layers of the current image
func (c *LayersPanel) Refresh() {
	d := c.main.doc()
	n := 0
	if d != nil {
		n = len(d.Layers)
	}
	c.list.SetInnerSize(c.list.Width(), n*layerRowHeight)
	// Thumbnails of the layers that are gone
	if d != nil && len(c.thumbs) > 64 {
		clear(c.thumbs)
	}
}

// rowLayer returns the index of the layer at the row (the top layer is row 0)
func rowLayer(d *paint.Document, row int) int {
	return len(d.Layers) - 1 - row
}

func (c *LayersPanel) paint(cnv *ui.Canvas) {
	d := c.main.doc()
	if d == nil {
		return
	}
	pal := ui.CurrentPalette()
	w := c.list.Width()
	for row := range d.Layers {
		i := rowLayer(d, row)
		l := d.Layers[i]
		y := row * layerRowHeight
		if i == d.Current {
			cnv.FillRect(0, y, w, layerRowHeight, pal.Selection)
		}
		// The visibility box
		cy := y + (layerRowHeight-layerCheckSize)/2
		cnv.SetColor(pal.Border)
		cnv.DrawRect(6, cy, layerCheckSize, layerCheckSize)
		if l.Visible {
			cnv.FillRect(9, cy+3, layerCheckSize-6, layerCheckSize-6, pal.Highlight)
		}
		// The thumbnail on checkers
		tx, ty := layerCheckSize+14, y+(layerRowHeight-layerThumbSize)/2
		cnv.DrawImage(tx, ty, c.thumb(d, l))
		cnv.SetColor(pal.Border)
		cnv.DrawRect(tx-1, ty-1, layerThumbSize+2, layerThumbSize+2)
		// The name, and what is not usual about the layer
		cnv.SetColor(pal.Text)
		if !l.Visible {
			cnv.SetColor(pal.DisabledText)
		}
		cnv.SetHAlign(ui.HAlignLeft)
		cnv.SetVAlign(ui.VAlignCenter)
		cnv.SetFontFamily(ui.ThemeFontFamily())
		cnv.SetFontSize(ui.ThemeFontSize())
		name := l.Name
		if l.Opacity < 255 || l.Blend != paint.BlendNormal {
			name += "  ·"
			if l.Blend != paint.BlendNormal {
				name += " " + T().BlendName(l.Blend)
			}
			if l.Opacity < 255 {
				name += " " + percent(float64(l.Opacity)*100/255)
			}
		}
		nx := tx + layerThumbSize + 10
		cnv.DrawText(nx, y, w-nx-4, layerRowHeight, name)
		cnv.FillRect(0, y+layerRowHeight-1, w, 1, pal.Divider)
	}
}

// thumb returns the small picture of the layer, made again when it changed
func (c *LayersPanel) thumb(d *paint.Document, l *paint.Layer) image.Image {
	t, ok := c.thumbs[l]
	if ok && t.version == d.Version() {
		return t.img
	}
	if ok && c.main.canvas.drag != nil {
		return t.img // not while drawing: it is redrawn when the drag ends
	}
	t = layerThumb{version: d.Version(), img: makeThumb(l.Img, layerThumbSize)}
	c.thumbs[l] = t
	return t.img
}

// makeThumb scales the image into a size x size square over checkers
func makeThumb(src *image.RGBA, size int) image.Image {
	w, h := src.Rect.Dx(), src.Rect.Dy()
	scale := min(float64(size)/float64(w), float64(size)/float64(h))
	tw, th := max(1, int(float64(w)*scale)), max(1, int(float64(h)*scale))
	// A few pixels of the source for each one of the thumbnail: it is made
	// on every change, and the layers can be large
	small := image.NewRGBA(image.Rect(0, 0, tw, th))
	xdraw.ApproxBiLinear.Scale(small, small.Rect, src, src.Rect, xdraw.Src, nil)
	dst := image.NewRGBA(image.Rect(0, 0, size, size))
	ws := colorWorkspace.get()
	for i := 0; i < len(dst.Pix); i += 4 {
		dst.Pix[i], dst.Pix[i+1], dst.Pix[i+2], dst.Pix[i+3] = ws.R, ws.G, ws.B, 255
	}
	ox, oy := (size-tw)/2, (size-th)/2
	for y := range th {
		for x := range tw {
			bg := uint32(colorCheckerLight.R)
			if (x/4+y/4)%2 == 1 {
				bg = uint32(colorCheckerDark.R)
			}
			si := small.PixOffset(x, y)
			di := dst.PixOffset(ox+x, oy+y)
			inv := 255 - uint32(small.Pix[si+3])
			for ch := range 3 {
				dst.Pix[di+ch] = uint8(uint32(small.Pix[si+ch]) + bg*inv/255)
			}
		}
	}
	return dst
}

func (c *LayersPanel) mouseDown(button ui.MouseButton, x, y int, mods ui.KeyModifiers) bool {
	d := c.main.doc()
	if d == nil {
		return true
	}
	row := y / layerRowHeight
	if row < 0 || row >= len(d.Layers) {
		return true
	}
	i := rowLayer(d, row)
	if x < layerCheckSize+12 && button == ui.MouseButtonLeft {
		c.main.toggleLayerVisible(i)
		return true
	}
	c.main.selectLayer(i)
	return true
}

// ---- History ----

const historyRowHeight = 24

// HistoryPanel lists the steps of the history of the current image; a click goes back or forward to a step
type HistoryPanel struct {
	ui.Widget
	main *MainForm
	list *ui.Widget
}

func NewHistoryPanel(main *MainForm) *HistoryPanel {
	var c HistoryPanel
	c.InitWidget()
	c.main = main
	c.SetPanelPadding(0)
	c.list = &ui.Widget{}
	c.list.InitWidget()
	c.list.SetXExpandable(true)
	c.list.SetYExpandable(true)
	c.list.SetAllowScroll(false, true)
	c.list.SetAutoFillBackground(true)
	c.list.SetRole("base")
	c.list.SetOnPaint(c.paint)
	c.list.SetOnMouseDown(func(button ui.MouseButton, x, y int, mods ui.KeyModifiers) bool {
		d := main.doc()
		if d == nil || button != ui.MouseButtonLeft {
			return true
		}
		row := y / historyRowHeight
		if row >= 0 && row <= len(d.History.Items()) {
			main.historyGoTo(row)
		}
		return true
	})
	c.AddWidget(0, 0, c.list)

	bar := ui.NewPanel()
	bar.SetPanelPadding(0)
	bar.SetCellPadding(0)
	undo := ui.NewToolButton(nil, "", main.undo)
	setIcon("undo", undo.SetImage)
	undo.SetTooltipFunc(func() string { return T().CmdUndo + " (Ctrl+Z)" })
	redo := ui.NewToolButton(nil, "", main.redo)
	setIcon("redo", redo.SetImage)
	redo.SetTooltipFunc(func() string { return T().CmdRedo + " (Ctrl+Y)" })
	for i, b := range []*ui.ToolButton{undo, redo} {
		b.SetButtonSize(30, 30)
		b.SetFlat(true)
		bar.AddWidget(0, i, b)
	}
	bar.AddWidget(0, 2, ui.NewHSpacer())
	c.AddWidget(1, 0, bar)
	return &c
}

// Refresh lays the list out for the history and shows its last done step
func (c *HistoryPanel) Refresh() {
	d := c.main.doc()
	n := 1
	if d != nil {
		n += len(d.History.Items())
	}
	h := n * historyRowHeight
	if c.list.InnerHeight() != h {
		c.list.SetInnerSize(c.list.Width(), h)
		if d != nil {
			// The current step stays in sight
			y := d.History.Pos() * historyRowHeight
			if y+historyRowHeight > c.list.ScrollY()+c.list.Height() {
				c.list.SetScrollY(y + historyRowHeight - c.list.Height())
			}
		}
	}
}

func (c *HistoryPanel) paint(cnv *ui.Canvas) {
	d := c.main.doc()
	if d == nil {
		return
	}
	pal := ui.CurrentPalette()
	w := c.list.Width()
	cnv.SetHAlign(ui.HAlignLeft)
	cnv.SetVAlign(ui.VAlignCenter)
	cnv.SetFontFamily(ui.ThemeFontFamily())
	cnv.SetFontSize(ui.ThemeFontSize())
	pos := d.History.Pos()
	names := []string{T().HistoryOpen}
	if d.Path == "" {
		names[0] = T().HistoryNew
	}
	for _, item := range d.History.Items() {
		names = append(names, T().HistoryName(item.Name))
	}
	for row, name := range names {
		y := row * historyRowHeight
		if row == pos {
			cnv.FillRect(0, y, w, historyRowHeight, pal.Selection)
		}
		cnv.SetColor(pal.Text)
		if row > pos {
			cnv.SetColor(pal.DisabledText) // undone, can be redone
		}
		cnv.DrawText(8, y, w-12, historyRowHeight, name)
	}
}
