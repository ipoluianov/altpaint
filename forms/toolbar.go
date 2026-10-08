package forms

import (
	"fmt"
	"math"

	"github.com/ipoluianov/altpaint/paint"
	"github.com/ipoluianov/nui/ui"
)

// ToolBar is the row under the menu: the main commands, then the options of the current tool
type ToolBar struct {
	ui.Widget
	main *MainForm

	groups map[int][]ui.Widgeter

	width     *ui.NumBox
	antialias *ui.Checkbox
	blend     *ui.Checkbox
	shape     *ui.ComboBox
	fill      *ui.ComboBox
	gradient  *ui.ComboBox
	tolerance *ui.NumBox
	flood     *ui.ComboBox
	sampling  *ui.ComboBox
	selMode   *ui.ComboBox
	fontSize  *ui.NumBox
	bold      *ui.Checkbox
	italic    *ui.Checkbox
	mono      *ui.Checkbox
	toolName  *ui.Label

	// The check boxes with their texts, sized to fit them in each language
	checks []checkText

	// updating is set while the controls are set from the state, so their callbacks do nothing
	updating bool
}

type checkText struct {
	cb   *ui.Checkbox
	text func() string
}

func NewToolBar(main *MainForm) *ToolBar {
	var c ToolBar
	c.InitWidget()
	c.main = main
	c.SetPanelPadding(2)
	c.groups = make(map[int][]ui.Widgeter)
	col := 0
	add := func(w ui.Widgeter) {
		c.AddWidget(0, col, w)
		col++
	}
	gap := func(width int) {
		s := ui.NewSpace()
		s.SetSize(width, 0)
		add(s)
	}

	commands := []struct {
		icon string
		tip  func() string
		f    func()
	}{
		{"new", func() string { return T().CmdNew + " (Ctrl+N)" }, main.newImage},
		{"open", func() string { return T().CmdOpen + " (Ctrl+O)" }, main.openDialog},
		{"save", func() string { return T().CmdSave + " (Ctrl+S)" }, func() { main.save(main.tab(), nil) }},
		{"", nil, nil},
		{"cut", func() string { return T().CmdCut + " (Ctrl+X)" }, main.cut},
		{"copy", func() string { return T().CmdCopy + " (Ctrl+C)" }, main.copy},
		{"paste", func() string { return T().CmdPaste + " (Ctrl+V)" }, main.paste},
		{"crop", func() string { return T().CmdCropToSelection + " (Ctrl+Shift+X)" }, main.cropToSelection},
		{"deselect", func() string { return T().CmdDeselect + " (Ctrl+D)" }, main.deselect},
		{"", nil, nil},
		{"undo", func() string { return T().CmdUndo + " (Ctrl+Z)" }, main.undo},
		{"redo", func() string { return T().CmdRedo + " (Ctrl+Y)" }, main.redo},
	}
	for _, cmd := range commands {
		if cmd.f == nil {
			gap(10)
			continue
		}
		btn := ui.NewToolButton(nil, "", cmd.f)
		btn.SetButtonSize(toolButtonSize, toolButtonSize)
		btn.SetFlat(true)
		setIcon(cmd.icon, btn.SetImage)
		btn.SetTooltipFunc(cmd.tip)
		add(btn)
	}
	gap(16)
	sep := ui.NewSpace()
	sep.SetSize(1, toolButtonSize-8)
	sep.SetOnPaint(func(cnv *ui.Canvas) { cnv.FillRect(0, 0, 1, sep.Height(), ui.CurrentPalette().Divider) })
	add(sep)
	gap(12)

	c.toolName = ui.NewLabel("")
	add(c.toolName)
	gap(12)

	label := func(text func() string) *ui.Label {
		l := ui.NewLabel("")
		l.SetTextFunc(text)
		return l
	}
	// A group is hidden with its gap, so the shown ones stay together
	group := func(opt int, ws ...ui.Widgeter) {
		s := ui.NewSpace()
		s.SetSize(10, 0)
		ws = append(ws, s)
		for _, w := range ws {
			add(w)
		}
		c.groups[opt] = ws
	}

	c.selMode = c.combo(func(cb *ui.ComboBox) { tools.SelMode = paint.CombineMode(cb.SelectedIndex()) })
	group(optSelMode, label(func() string { return T().SelectionMode }), c.selMode)

	c.width = c.numBox(1, 1000, 0, func(v float64) { tools.Width = v })
	group(optWidth, label(func() string { return T().BrushWidth }), c.width)

	c.shape = c.combo(func(cb *ui.ComboBox) { tools.Shape = cb.SelectedIndex() })
	group(optShape, label(func() string { return T().Shape }), c.shape)

	c.fill = c.combo(func(cb *ui.ComboBox) { tools.Fill = cb.SelectedIndex() })
	group(optFill, label(func() string { return T().FillMode }), c.fill)

	c.gradient = c.combo(func(cb *ui.ComboBox) { tools.Gradient = paint.GradientKind(cb.SelectedIndex()) })
	group(optGradient, label(func() string { return T().GradientType }), c.gradient)

	c.fontSize = c.numBox(4, 500, 0, func(v float64) { tools.Text.Size = v })
	c.bold = c.check(func() string { return T().Bold }, func(v bool) { tools.Text.Bold = v })
	c.italic = c.check(func() string { return T().Italic }, func(v bool) { tools.Text.Italic = v })
	c.mono = c.check(func() string { return T().Monospace }, func(v bool) { tools.Text.Mono = v })
	group(optFont, label(func() string { return T().FontSize }), c.fontSize, c.bold, c.italic, c.mono)

	c.tolerance = c.numBox(0, 100, 0, func(v float64) { tools.Tolerance = v })
	group(optTolerance, label(func() string { return T().Tolerance }), c.tolerance)

	c.flood = c.combo(func(cb *ui.ComboBox) { tools.Global = cb.SelectedIndex() == 1 })
	group(optFlood, label(func() string { return T().FloodMode }), c.flood)

	c.sampling = c.combo(func(cb *ui.ComboBox) { tools.SampleAll = cb.SelectedIndex() == 1 })
	group(optSampling, label(func() string { return T().Sampling }), c.sampling)

	c.antialias = c.check(func() string { return T().Antialiasing }, func(v bool) { tools.Antialias = v })
	group(optAntialias, c.antialias)

	c.blend = c.check(func() string { return T().AlphaBlending }, func(v bool) { tools.Blend = v })
	group(optBlend, c.blend)

	add(ui.NewHSpacer())
	c.fillCombos()
	return &c
}

func (c *ToolBar) combo(onChange func(cb *ui.ComboBox)) *ui.ComboBox {
	cb := ui.NewComboBox()
	cb.SetOnSelectedIndexChanged(func() {
		if !c.updating {
			onChange(cb)
			c.main.toolOptionsChanged()
		}
	})
	return cb
}

func (c *ToolBar) numBox(minV, maxV float64, decimals int, onChange func(v float64)) *ui.NumBox {
	nb := ui.NewNumBox()
	nb.SetDecimals(decimals)
	nb.SetMin(minV)
	nb.SetMax(maxV)
	nb.SetMinWidth(70)
	nb.SetMaxWidth(80)
	nb.SetOnChanged(func() {
		if !c.updating {
			onChange(nb.Value())
			c.main.toolOptionsChanged()
		}
	})
	return nb
}

func (c *ToolBar) check(text func() string, onChange func(v bool)) *ui.Checkbox {
	cb := ui.NewCheckbox("")
	cb.SetTextFunc(text)
	cb.SetXExpandable(false)
	c.checks = append(c.checks, checkText{cb, text})
	cb.SetOnStateChanged(func() {
		if !c.updating {
			onChange(cb.Checked())
			c.main.toolOptionsChanged()
		}
	})
	return cb
}

// fillCombos puts the texts of the current language into the combo boxes
func (c *ToolBar) fillCombos() {
	c.updating = true
	defer func() { c.updating = false }()
	fillCombo(c.selMode, T().SelectionModes[:]...)
	fillCombo(c.shape, T().Shapes[:]...)
	fillCombo(c.fill, T().FillModes[:]...)
	var gradients []string
	for _, k := range paint.GradientKinds {
		gradients = append(gradients, T().GradientName(k))
	}
	fillCombo(c.gradient, gradients...)
	fillCombo(c.flood, T().FloodModes[:]...)
	fillCombo(c.sampling, T().SamplingModes[:]...)
	// A check box is 150 wide by default: it is made as wide as its text
	for _, ct := range c.checks {
		w, _, _ := ui.MeasureText(ui.ThemeFontFamily(), ui.ThemeFontSize(), ct.text())
		ct.cb.SetMinWidth(w + 32)
		ct.cb.SetMaxWidth(w + 32)
	}
	c.Refresh()
}

// fillCombo replaces the items of the combo box, keeping the chosen one
func fillCombo(cb *ui.ComboBox, items ...string) {
	if cb.ItemCount() == len(items) {
		for i, s := range items {
			cb.SetItemText(i, s)
		}
		return
	}
	sel := max(0, cb.SelectedIndex())
	for _, s := range items {
		cb.AddItem(s, nil)
	}
	cb.SetSelectedIndex(min(sel, len(items)-1))
}

// Refresh shows the options of the current tool and their values
func (c *ToolBar) Refresh() {
	c.updating = true
	defer func() { c.updating = false }()
	def := toolDefOf(tools.Tool)
	for opt, ws := range c.groups {
		for _, w := range ws {
			if v, ok := w.(interface{ SetVisible(bool) }); ok {
				v.SetVisible(def.opts&opt != 0)
			}
		}
	}
	c.toolName.SetText(T().ToolName(tools.Tool))
	setNum := func(nb *ui.NumBox, v float64) {
		if nb.Value() != v {
			nb.SetValue(v)
		}
	}
	setNum(c.width, tools.Width)
	setNum(c.tolerance, tools.Tolerance)
	setNum(c.fontSize, tools.Text.Size)
	c.antialias.SetChecked(tools.Antialias)
	c.blend.SetChecked(tools.Blend)
	c.bold.SetChecked(tools.Text.Bold)
	c.italic.SetChecked(tools.Text.Italic)
	c.mono.SetChecked(tools.Text.Mono)
	c.selMode.SetSelectedIndex(int(tools.SelMode))
	c.shape.SetSelectedIndex(tools.Shape)
	c.fill.SetSelectedIndex(tools.Fill)
	c.gradient.SetSelectedIndex(int(tools.Gradient))
	c.flood.SetSelectedIndex(b2i(tools.Global))
	c.sampling.SetSelectedIndex(b2i(tools.SampleAll))
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

// percent formats a share as "45%"
func percent(v float64) string {
	if v < 10 && v != math.Trunc(v) {
		return fmt.Sprintf("%.1f%%", v)
	}
	return fmt.Sprintf("%.0f%%", v)
}
