package forms

import (
	"fmt"
	"math"

	"github.com/ipoluianov/altpaint/app"
	"github.com/ipoluianov/altpaint/config"
	"github.com/ipoluianov/altpaint/install"
	"github.com/ipoluianov/nui/ui"
)

// StatusBar is the bottom row: the hint of the tool, where the mouse is, the
// sizes, the zoom, and the links as in the other AltBins utilities
type StatusBar struct {
	ui.Widget
	main *MainForm

	lblHint   *ui.Label
	lblCursor *statusCell
	lblSel    *statusCell
	lblSize   *statusCell
	zoom      *ui.ComboBox
	zoomIndex int // shown in the zoom box, -1 - none yet
	zoomFirst string

	// What the install link does: install, update or uninstall
	installStatus install.Status
}

// zoomPresets are offered in the zoom box, after "Window"
var zoomPresets = []float64{0.12, 0.25, 0.5, 0.66, 1, 1.5, 2, 3, 4, 8, 16, 32}

func NewStatusBar(main *MainForm) *StatusBar {
	var c StatusBar
	c.InitWidget()
	c.main = main
	c.SetPanelPadding(4)
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
	// The hint takes the free space and is cut when there is not enough
	c.lblHint = ui.NewLabel("")
	c.lblHint.SetXExpandable(true)
	c.lblHint.SetForegroundColor(colorMuted.get())
	add(c.lblHint)
	c.lblCursor = newStatusCell(90)
	add(c.lblCursor)
	gap(8)
	c.lblSel = newStatusCell(170)
	add(c.lblSel)
	gap(8)
	c.lblSize = newStatusCell(170)
	add(c.lblSize)
	gap(8)
	c.zoomIndex = -1
	c.zoom = ui.NewComboBox()
	c.zoom.SetMinWidth(90)
	c.zoom.SetOnSelectedIndexChanged(c.onZoom)
	add(c.zoom)
	gap(16)

	links := []*ui.Label{
		newLinkLabel(func() string { return T().Settings }, func() { main.ShowSettings() }),
		newLinkLabel(func() string { return T().Help }, func() { openDocs(&c, "help") }),
	}
	// A downloaded copy offers to install or update itself, the installed one to be removed
	c.installStatus = install.CurrentStatus()
	if c.installStatus != install.StatusNone {
		links = append(links, newLinkLabel(c.installText, c.onInstallLink))
	}
	links = append(links, newLinkLabel(func() string { return T().About }, c.onAbout))
	for i, lbl := range links {
		if i > 0 {
			gap(12)
		}
		add(lbl)
	}
	c.fillZoom()
	return &c
}

// fillZoom puts the zoom presets into the box, in the current language
func (c *StatusBar) fillZoom() {
	items := []string{T().ZoomWindow}
	for _, z := range zoomPresets {
		items = append(items, percent(z*100))
	}
	fillCombo(c.zoom, items...)
	c.zoomIndex = -1
}

func (c *StatusBar) onZoom() {
	i := c.zoom.SelectedIndex()
	if i == 0 {
		c.main.canvas.ZoomToWindow()
	} else if i > 0 && i <= len(zoomPresets) {
		cv := c.main.canvas
		cv.SetZoom(zoomPresets[i-1], cv.Width()/2, cv.Height()/2)
	}
	c.main.canvas.Focus()
}

// Refresh shows the state of the current image and the mouse
func (c *StatusBar) Refresh() {
	// A label lays the window out again when its text is set, so only a change is set
	if hint := T().ToolHint(tools.Tool); c.lblHint.Text() != hint {
		c.lblHint.SetText(hint)
		c.lblHint.SetForegroundColor(colorMuted.get())
	}
	d := c.main.doc()
	if d == nil {
		c.lblCursor.SetText("")
		c.lblSel.SetText("")
		c.lblSize.SetText("")
		return
	}
	cv := c.main.canvas
	if cv.mouseIn {
		p := cv.toImage(cv.mouseX, cv.mouseY)
		c.lblCursor.SetText(fmt.Sprintf("%d, %d", int(math.Floor(p.X)), int(math.Floor(p.Y))))
	} else {
		c.lblCursor.SetText("")
	}
	if dr := cv.drag; dr != nil && (dr.tool == ToolRectSelect || dr.tool == ToolEllipseSelect || dr.tool == ToolShapes) {
		r := selectRect(dr.start, dr.cur, dr.mods.Shift)
		c.lblSel.SetText(T().StatusSelection(r.Dx(), r.Dy()))
	} else if d.Sel != nil {
		b := d.Sel.Bounds()
		c.lblSel.SetText(T().StatusSelection(b.Dx(), b.Dy()))
	} else {
		c.lblSel.SetText("")
	}
	c.lblSize.SetText(T().StatusImage(d.W, d.H))
	// The box shows the zoom when it is one of the presets
	idx := -1
	z := cv.zoom()
	if cv.tab != nil && cv.tab.fit {
		idx = 0
	} else {
		for i, p := range zoomPresets {
			if math.Abs(p-z) < 1e-6 {
				idx = i + 1
			}
		}
	}
	first := T().ZoomWindow
	if idx < 0 {
		idx, first = 0, percent(z*100)
	}
	if c.zoom.SelectedIndex() != idx || c.zoomIndex != idx || c.zoomFirst != first {
		c.zoom.SetItemText(0, first)
		c.zoom.SetSelectedIndex(idx)
		c.zoomIndex, c.zoomFirst = idx, first
	}
}

// statusCell is a text of the status bar that changes often, e.g. with the
// mouse: it has a fixed width and is just repainted, as a label would lay
// the window out again on every change
type statusCell struct {
	ui.Widget
	text string
}

func newStatusCell(width int) *statusCell {
	var c statusCell
	c.InitWidget()
	c.SetMinWidth(width)
	c.SetMaxWidth(width)
	c.SetMinHeight(ui.ThemeControlHeight())
	c.SetMaxHeight(ui.ThemeControlHeight())
	c.SetOnPaint(func(cnv *ui.Canvas) {
		cnv.SetHAlign(ui.HAlignLeft)
		cnv.SetVAlign(ui.VAlignCenter)
		cnv.SetFontFamily(ui.ThemeFontFamily())
		cnv.SetFontSize(ui.ThemeFontSize())
		cnv.SetColor(ui.CurrentPalette().WindowText)
		cnv.DrawText(0, 0, c.Width(), c.Height(), c.text)
	})
	return &c
}

func (c *statusCell) SetText(text string) {
	if c.text == text {
		return
	}
	c.text = text
	if f := c.Form(); f != nil {
		f.Update()
	}
}

// newLinkLabel creates a hyperlink-style label with the text from text() that calls onClick on left click.
// It is underlined only under the mouse, so a row of links stays quiet.
func newLinkLabel(text func() string, onClick func()) *ui.Label {
	lbl := ui.NewLabel("")
	lbl.SetTextFunc(text)
	lbl.SetForegroundColor(colorLink.get())
	lbl.SetOnMouseEnter(func() { lbl.SetUnderline(true) })
	lbl.SetOnMouseLeave(func() { lbl.SetUnderline(false) })
	linkLabels = append(linkLabels, lbl)
	lbl.SetMouseCursor(ui.MouseCursorPointer)
	lbl.SetOnMouseDown(func(button ui.MouseButton, x int, y int, mods ui.KeyModifiers) bool {
		if button != ui.MouseButtonLeft {
			return false
		}
		onClick()
		return true
	})
	return lbl
}

// openDocs opens the docs on the site; campaign tells which place in the app the visit came from
func openDocs(parent ui.Widgeter, campaign string) {
	if err := app.OpenSiteURL(app.DocsURL, campaign); err != nil {
		ui.ShowMessageBox(parent, T().Error, err.Error())
	}
}

func (c *StatusBar) onAbout() {
	c.ShowDialog(NewAboutDialog())
}

// installText is the text of the install link for what it does now
func (c *StatusBar) installText() string {
	switch c.installStatus {
	case install.StatusUpdate:
		return T().Update
	case install.StatusUninstall:
		return T().Uninstall
	}
	return T().Install
}

func (c *StatusBar) onInstallLink() {
	if c.installStatus == install.StatusUninstall {
		c.onUninstall()
		return
	}
	c.onInstall()
}

// onInstall copies the application to ~/.altbins and registers it, then
// quits for the installed copy to start (see main). Over an older installed
// version it is the same: that one is replaced.
func (c *StatusBar) onInstall() {
	ui.ShowQuestionMessageBoxOKCancel(c, c.installText(), T().InstallAsk(install.Dir()), func() {
		c.main.requestExit(func() {
			if err := install.Install(); err != nil {
				ui.ShowMessageBox(c, T().Error, T().InstallFailed(err.Error()))
				return
			}
			install.RelaunchAfterExit()
			c.main.quit()
		})
	}, nil)
}

// onUninstall removes the installed copy; the settings stay.
// When that is this copy, it quits and its binary is deleted once it has.
func (c *StatusBar) onUninstall() {
	ui.ShowQuestionMessageBoxOKCancel(c, T().Uninstall, T().UninstallAsk(config.ConfigDirectory()), func() {
		installed := install.IsInstalledCopy()
		if installed {
			c.main.requestExit(func() {
				if err := install.Uninstall(); err != nil {
					ui.ShowMessageBox(c, T().Error, err.Error())
					return
				}
				c.main.quit()
			})
			return
		}
		if err := install.Uninstall(); err != nil {
			ui.ShowMessageBox(c, T().Error, err.Error())
			return
		}
		c.installStatus = install.CurrentStatus()
		ui.ShowToast(c, T().Uninstalled, ui.ToastSuccess)
	}, nil)
}
