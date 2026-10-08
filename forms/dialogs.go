package forms

import (
	"image/color"
	"math"

	"github.com/ipoluianov/altpaint/app"
	"github.com/ipoluianov/altpaint/config"
	"github.com/ipoluianov/altpaint/paint"
	"github.com/ipoluianov/nui/ui"
)

// maxImageSide limits the sizes typed in the dialogs
const maxImageSide = 32768

// dialogButtons adds a row of buttons at the right of the panel
func dialogButtons(panel *ui.Panel, buttons ...*ui.Button) {
	panel.AddWidget(0, 0, ui.NewHSpacer())
	for i, b := range buttons {
		panel.AddWidget(0, i+1, b)
	}
}

// dialogLayout adds the content panel, a spacer and the buttons panel to the dialog
func dialogLayout(c *ui.DialogContent) (content, buttons *ui.Panel) {
	content = ui.NewPanel()
	c.AddWidget(0, 0, content)
	c.AddWidget(1, 0, ui.NewVSpacer())
	buttons = ui.NewPanel()
	c.AddWidget(2, 0, buttons)
	return content, buttons
}

// intBox returns a NumBox for a whole number
func intBox(minV, maxV, value float64) *ui.NumBox {
	nb := ui.NewNumBox()
	nb.SetDecimals(0)
	nb.SetMin(minV)
	nb.SetMax(maxV)
	nb.SetValue(value)
	nb.SetMinWidth(110)
	return nb
}

func numValue(nb *ui.NumBox) int {
	return int(math.Round(nb.Value()))
}

// okCancel makes the OK and Cancel buttons of a dialog
func okCancel(c *ui.DialogContent, buttons *ui.Panel, onOK func()) (ok, cancel *ui.Button) {
	ok = ui.NewButton(ui.UIText().OK)
	ok.SetOnClick(onOK)
	cancel = ui.NewButton(ui.UIText().Cancel)
	cancel.SetOnClick(func() { c.Form().RequestClose() })
	dialogButtons(buttons, ok, cancel)
	return ok, cancel
}

// showDialog sets the title, the size and the buttons of a dialog when it is shown
func showDialog(c *ui.DialogContent, title string, w, h int, ok, cancel *ui.Button, focus ui.Widgeter) {
	c.OnDialogShow = func() {
		c.Form().SetTitle(title)
		c.Form().SetSize(w, h)
		c.Form().MoveToCenterOfParent()
		c.Form().SetAcceptButton(ok)
		c.Form().SetCancelButton(cancel)
		if focus != nil {
			focus.Focus()
		}
	}
}

// ---- New image ----

func NewNewImageDialog(w, h int, onAccept func(w, h int, bg color.NRGBA)) *ui.DialogContent {
	var c ui.DialogContent
	c.InitWidget()
	content, buttons := dialogLayout(&c)
	nbW := intBox(1, maxImageSide, float64(w))
	nbH := intBox(1, maxImageSide, float64(h))
	bg := ui.NewComboBox()
	bg.AddItem(T().BackgroundWhite, color.NRGBA{255, 255, 255, 255})
	bg.AddItem(T().BackgroundSecondary, tools.Secondary)
	bg.AddItem(T().BackgroundTransparent, color.NRGBA{})
	bg.SetSelectedIndex(0)
	content.AddWidget(0, 0, ui.NewLabel(T().Width))
	content.AddWidget(0, 1, nbW)
	content.AddWidget(0, 2, ui.NewLabel(T().Pixels))
	content.AddWidget(1, 0, ui.NewLabel(T().Height))
	content.AddWidget(1, 1, nbH)
	content.AddWidget(1, 2, ui.NewLabel(T().Pixels))
	content.AddWidget(2, 0, ui.NewLabel(T().BackgroundLabel))
	content.AddWidget(2, 1, bg)
	content.AddWidget(0, 3, ui.NewHSpacer())
	ok, cancel := okCancel(&c, buttons, func() {
		w, h := numValue(nbW), numValue(nbH)
		col, _ := bg.SelectedItemData().(color.NRGBA)
		c.Form().Close()
		c.RunInParent(func() { onAccept(w, h, col) })
	})
	showDialog(&c, T().CmdNew, 380, 220, ok, cancel, nbW)
	return &c
}

// ---- Resize image ----

func NewResizeDialog(w, h int, onAccept func(w, h int, method paint.Resampling)) *ui.DialogContent {
	var c ui.DialogContent
	c.InitWidget()
	content, buttons := dialogLayout(&c)
	nbPct := intBox(1, 10000, 100)
	nbW := intBox(1, maxImageSide, float64(w))
	nbH := intBox(1, maxImageSide, float64(h))
	keep := ui.NewCheckbox(T().KeepAspect)
	keep.SetChecked(true)
	method := ui.NewComboBox()
	for i, name := range T().Resamplings {
		method.AddItem(name, paint.Resampling(i))
	}
	method.SetSelectedIndex(0)

	updating := false
	set := func(f func()) {
		if updating {
			return
		}
		updating = true
		f()
		updating = false
	}
	nbPct.SetOnChanged(func() {
		set(func() {
			p := nbPct.Value() / 100
			nbW.SetValue(math.Max(1, math.Round(float64(w)*p)))
			nbH.SetValue(math.Max(1, math.Round(float64(h)*p)))
		})
	})
	nbW.SetOnChanged(func() {
		set(func() {
			if keep.Checked() {
				nbH.SetValue(math.Max(1, math.Round(nbW.Value()*float64(h)/float64(w))))
			}
		})
	})
	nbH.SetOnChanged(func() {
		set(func() {
			if keep.Checked() {
				nbW.SetValue(math.Max(1, math.Round(nbH.Value()*float64(w)/float64(h))))
			}
		})
	})

	content.AddWidget(0, 0, ui.NewLabel(T().ByPercentage))
	content.AddWidget(0, 1, nbPct)
	content.AddWidget(0, 2, ui.NewLabel("%"))
	content.AddWidget(1, 0, ui.NewLabel(T().Width))
	content.AddWidget(1, 1, nbW)
	content.AddWidget(1, 2, ui.NewLabel(T().Pixels))
	content.AddWidget(2, 0, ui.NewLabel(T().Height))
	content.AddWidget(2, 1, nbH)
	content.AddWidget(2, 2, ui.NewLabel(T().Pixels))
	content.AddWidget(3, 1, keep)
	content.AddWidget(4, 0, ui.NewLabel(T().Resampling))
	content.AddWidget(4, 1, method)
	content.AddWidget(0, 3, ui.NewHSpacer())
	ok, cancel := okCancel(&c, buttons, func() {
		nw, nh := numValue(nbW), numValue(nbH)
		m, _ := method.SelectedItemData().(paint.Resampling)
		c.Form().Close()
		c.RunInParent(func() { onAccept(nw, nh, m) })
	})
	showDialog(&c, T().CmdResize, 420, 300, ok, cancel, nbPct)
	return &c
}

// ---- Canvas size ----

// anchorBox is a 3x3 grid to choose where the image stays when the canvas is resized
type anchorBox struct {
	ui.Widget
	anchor paint.Anchor
}

const anchorCell = 26

func newAnchorBox() *anchorBox {
	var c anchorBox
	c.InitWidget()
	c.SetMinSize(anchorCell*3+1, anchorCell*3+1)
	c.SetMaxSize(anchorCell*3+1, anchorCell*3+1)
	c.SetOnPaint(func(cnv *ui.Canvas) {
		pal := ui.CurrentPalette()
		for j := range 3 {
			for i := range 3 {
				x, y := i*anchorCell, j*anchorCell
				cnv.FillRect(x, y, anchorCell+1, anchorCell+1, pal.Border)
				col := pal.Button
				if i-1 == c.anchor.X && j-1 == c.anchor.Y {
					col = pal.Highlight
				}
				cnv.FillRect(x+1, y+1, anchorCell-1, anchorCell-1, col)
			}
		}
	})
	c.SetOnMouseDown(func(button ui.MouseButton, x, y int, mods ui.KeyModifiers) bool {
		i, j := x/anchorCell, y/anchorCell
		if i >= 0 && i < 3 && j >= 0 && j < 3 {
			c.anchor = paint.Anchor{X: i - 1, Y: j - 1}
			c.Form().Update()
		}
		return true
	})
	return &c
}

func NewCanvasSizeDialog(w, h int, onAccept func(w, h int, anchor paint.Anchor)) *ui.DialogContent {
	var c ui.DialogContent
	c.InitWidget()
	content, buttons := dialogLayout(&c)
	nbW := intBox(1, maxImageSide, float64(w))
	nbH := intBox(1, maxImageSide, float64(h))
	keep := ui.NewCheckbox(T().KeepAspect)
	updating := false
	nbW.SetOnChanged(func() {
		if !updating && keep.Checked() {
			updating = true
			nbH.SetValue(math.Max(1, math.Round(nbW.Value()*float64(h)/float64(w))))
			updating = false
		}
	})
	nbH.SetOnChanged(func() {
		if !updating && keep.Checked() {
			updating = true
			nbW.SetValue(math.Max(1, math.Round(nbH.Value()*float64(w)/float64(h))))
			updating = false
		}
	})
	anchor := newAnchorBox()
	content.AddWidget(0, 0, ui.NewLabel(T().Width))
	content.AddWidget(0, 1, nbW)
	content.AddWidget(0, 2, ui.NewLabel(T().Pixels))
	content.AddWidget(1, 0, ui.NewLabel(T().Height))
	content.AddWidget(1, 1, nbH)
	content.AddWidget(1, 2, ui.NewLabel(T().Pixels))
	content.AddWidget(2, 1, keep)
	content.AddWidget(3, 0, ui.NewLabel(T().Anchor))
	content.AddWidget(3, 1, anchor)
	content.AddWidget(0, 3, ui.NewHSpacer())
	ok, cancel := okCancel(&c, buttons, func() {
		nw, nh := numValue(nbW), numValue(nbH)
		a := anchor.anchor
		c.Form().Close()
		c.RunInParent(func() { onAccept(nw, nh, a) })
	})
	showDialog(&c, T().CmdCanvasSize, 400, 330, ok, cancel, nbW)
	return &c
}

// ---- Layer properties ----

// NewLayerPropertiesDialog edits the layer: preview shows the values while
// they change, accept and cancel end the dialog
func NewLayerPropertiesDialog(l *paint.Layer, preview, accept func(name string, visible bool, opacity uint8, blend paint.BlendMode), cancelled func()) *ui.DialogContent {
	var c ui.DialogContent
	c.InitWidget()
	content, buttons := dialogLayout(&c)
	name := ui.NewTextBox()
	name.SetText(l.Name)
	visible := ui.NewCheckbox(T().Visible)
	visible.SetChecked(l.Visible)
	blend := ui.NewComboBox()
	for _, m := range paint.BlendModes() {
		blend.AddItem(T().BlendName(m), m)
	}
	blend.SetSelectedIndex(int(l.Blend))
	opacity := ui.NewSlider()
	opacity.SetRange(0, 255)
	opacity.SetStep(1)
	opacity.SetValue(float64(l.Opacity))
	opacity.SetXExpandable(true)
	nbOpacity := intBox(0, 255, float64(l.Opacity))
	nbOpacity.SetMinWidth(80)

	values := func() (string, bool, uint8, paint.BlendMode) {
		m, _ := blend.SelectedItemData().(paint.BlendMode)
		return name.Text(), visible.Checked(), uint8(numValue(nbOpacity)), m
	}
	update := func() { preview(values()) }
	opacity.SetOnValueChanged(func() {
		nbOpacity.SetValue(opacity.Value())
		update()
	})
	nbOpacity.SetOnChanged(func() {
		opacity.SetValue(nbOpacity.Value())
		update()
	})
	visible.SetOnStateChanged(update)
	blend.SetOnSelectedIndexChanged(update)

	content.AddWidget(0, 0, ui.NewLabel(T().Name))
	content.AddWidget(0, 1, name)
	content.AddWidget(1, 1, visible)
	content.AddWidget(2, 0, ui.NewLabel(T().BlendMode))
	content.AddWidget(2, 1, blend)
	content.AddWidget(3, 0, ui.NewLabel(T().Opacity))
	row := ui.NewPanel()
	row.SetPanelPadding(0)
	row.AddWidget(0, 0, opacity)
	row.AddWidget(0, 1, nbOpacity)
	content.AddWidget(3, 1, row)

	done := false
	ok, cancel := okCancel(&c, buttons, func() {
		done = true
		n, v, o, m := values()
		c.Form().Close()
		c.RunInParent(func() { accept(n, v, o, m) })
	})
	c.OnDialogReject = func() bool {
		if !done {
			done = true
			c.RunInParent(cancelled)
		}
		return true
	}
	showDialog(&c, T().CmdLayerProperties, 460, 250, ok, cancel, name)
	return &c
}

// ---- Filter ----

// NewFilterDialog asks for the parameters of the filter: preview shows the
// result as they change, accept or cancel end it
func NewFilterDialog(f *paint.Filter, preview func(params []float64), accept func(params []float64), cancelled func()) *ui.DialogContent {
	var c ui.DialogContent
	c.InitWidget()
	content, buttons := dialogLayout(&c)
	params := f.Defaults()
	for i, p := range f.Params {
		slider := ui.NewSlider()
		slider.SetRange(p.Min, p.Max)
		slider.SetStep(1)
		slider.SetValue(p.Default)
		slider.SetXExpandable(true)
		slider.SetMinWidth(220)
		nb := intBox(p.Min, p.Max, p.Default)
		nb.SetMinWidth(80)
		slider.SetOnValueChanged(func() {
			nb.SetValue(slider.Value())
			params[i] = slider.Value()
			preview(params)
		})
		nb.SetOnChanged(func() {
			if slider.Value() != nb.Value() {
				slider.SetValue(nb.Value())
				params[i] = nb.Value()
				preview(params)
			}
		})
		content.AddWidget(i, 0, ui.NewLabel(T().ParamName(p.ID)))
		content.AddWidget(i, 1, slider)
		content.AddWidget(i, 2, nb)
	}
	done := false
	ok, cancel := okCancel(&c, buttons, func() {
		done = true
		c.Form().Close()
		c.RunInParent(func() { accept(params) })
	})
	c.OnDialogReject = func() bool {
		if !done {
			done = true
			c.RunInParent(cancelled)
		}
		return true
	}
	showDialog(&c, T().FilterName(f.ID), 480, 120+len(f.Params)*44, ok, cancel, nil)
	return &c
}

// ---- Save changes ----

const (
	answerCancel = iota
	answerYes
	answerNo
)

// showSaveChanges asks whether to save the image named name: Save, Don't Save or Cancel
func showSaveChanges(parent *MainForm, name string, onAnswer func(answer int)) {
	var c ui.DialogContent
	c.InitWidget()
	content, buttons := dialogLayout(&c)
	lbl := ui.NewLabel(T().SaveChangesAsk(name))
	content.AddWidget(0, 0, lbl)
	answered := false
	answer := func(a int) func() {
		return func() {
			answered = true
			c.Form().Close()
			c.RunInParent(func() { onAnswer(a) })
		}
	}
	yes := ui.NewButton(T().Save)
	yes.SetOnClick(answer(answerYes))
	no := ui.NewButton(T().DontSave)
	no.SetOnClick(answer(answerNo))
	cancel := ui.NewButton(ui.UIText().Cancel)
	cancel.SetOnClick(answer(answerCancel))
	dialogButtons(buttons, yes, no, cancel)
	c.OnDialogReject = func() bool {
		if !answered {
			answered = true
			c.RunInParent(func() { onAnswer(answerCancel) })
		}
		return true
	}
	w, _, _ := ui.MeasureText(ui.ThemeFontFamily(), ui.ThemeFontSize(), lbl.Text())
	showDialog(&c, T().UnsavedTitle, max(440, min(w+60, 800)), 150, yes, cancel, yes)
	parent.ShowDialog(&c)
}

// ---- Settings ----

// NewSettingsDialog edits the application options
func NewSettingsDialog(settings config.Settings, onAccept func(config.Settings)) *ui.DialogContent {
	var c ui.DialogContent
	c.InitWidget()
	content, buttons := dialogLayout(&c)

	cmbLanguage := ui.NewComboBox()
	cmbLanguage.AddItem(T().LanguageSystem, "")
	cmbLanguage.SetSelectedIndex(0)
	for i, l := range languages {
		cmbLanguage.AddItem(l.name, l.tag)
		if l.tag == settings.Language {
			cmbLanguage.SetSelectedIndex(i + 1)
		}
	}
	cmbTheme := ui.NewComboBox()
	cmbTheme.AddItem(T().ThemeDark, themeDark)
	cmbTheme.AddItem(T().ThemeLight, themeLight)
	cmbTheme.SetSelectedIndex(0)
	if settings.Theme == themeLight {
		cmbTheme.SetSelectedIndex(1)
	}
	nbJPEG := intBox(1, 100, float64(settings.JPEGQuality))
	chkGrid := ui.NewCheckbox(T().CmdPixelGrid)
	chkGrid.SetChecked(settings.PixelGrid)

	content.AddWidget(0, 0, ui.NewLabel(T().Language))
	content.AddWidget(0, 1, cmbLanguage)
	content.AddWidget(1, 0, ui.NewLabel(T().Theme))
	content.AddWidget(1, 1, cmbTheme)
	content.AddWidget(2, 0, ui.NewLabel(T().JPEGQuality))
	content.AddWidget(2, 1, nbJPEG)
	content.AddWidget(3, 1, chkGrid)
	content.AddWidget(0, 2, ui.NewHSpacer())

	ok, cancel := okCancel(&c, buttons, func() {
		s := config.GetSettings()
		s.Language, _ = cmbLanguage.SelectedItemData().(string)
		s.Theme, _ = cmbTheme.SelectedItemData().(string)
		s.JPEGQuality = numValue(nbJPEG)
		s.PixelGrid = chkGrid.Checked()
		c.Form().Close()
		c.RunInParent(func() { onAccept(s) })
	})
	showDialog(&c, T().Settings, 440, 260, ok, cancel, nil)
	return &c
}

// ---- About ----

func NewAboutDialog() *ui.DialogContent {
	var c ui.DialogContent
	c.InitWidget()
	content, buttons := dialogLayout(&c)
	lines := []string{
		T().Version + " " + app.Version,
		T().Author + " " + app.Author,
		app.Copyright(),
		T().License + " " + app.License,
		app.Website,
	}
	lblName := ui.NewLabel(app.DisplayName)
	lblName.SetFontSize(24)
	lblName.SetTextAlign(ui.HAlignCenter)
	content.AddWidget(0, 0, lblName)
	for i, s := range lines {
		l := ui.NewLabel(s)
		l.SetTextAlign(ui.HAlignCenter)
		content.AddWidget(i+1, 0, l)
	}
	btnWebsite := ui.NewButton(T().VisitWebsite)
	btnWebsite.SetOnClick(func() {
		if err := app.OpenSiteURL(app.Website, "about_dialog"); err != nil {
			ui.ShowMessageBox(&c, T().Error, err.Error())
		}
	})
	btnClose := ui.NewButton(T().Close)
	btnClose.SetOnClick(func() { c.Form().Close() })
	dialogButtons(buttons, btnWebsite, btnClose)
	showDialog(&c, T().AboutTitle(app.DisplayName), 400, 300, btnClose, btnClose, btnClose)
	return &c
}
