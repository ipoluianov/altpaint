package forms

import (
	"image/color"
	"path/filepath"

	"github.com/ipoluianov/altpaint/app"
	"github.com/ipoluianov/altpaint/config"
	"github.com/ipoluianov/altpaint/paint"
	"github.com/ipoluianov/nui/ui"
)

// MainForm is the window of AltPaint: the menu, the tool bar, the tabs of
// the images, the tools on the left, the image, the panels on the right
// (colors, layers, history) and the status bar
type MainForm struct {
	ui.Widget

	toolBar   *ToolBar
	tabBar    *TabBar
	toolsPane *ToolsPanel
	canvas    *CanvasView
	colors    *ColorsPanel
	layers    *LayersPanel
	history   *HistoryPanel
	status    *StatusBar
	splitter  *ui.Splitter
	rightPane *ui.Panel

	tabs []*DocTab
	cur  int

	// untitled numbers the new images: "Untitled 1", "Untitled 2"
	untitled int
	// lastVersion is the version of the current image the panels show
	lastVersion int
	// quitting: the window is closing, after the questions about the unsaved images
	quitting bool
}

const (
	defaultWindowWidth  = 1280
	defaultWindowHeight = 860
	defaultPanelsWidth  = 290
	// The image keeps at least this width when the panels are widened
	canvasMinWidth = 300
)

var lastCreatedMainWidget *MainForm

func NewMainForm() *MainForm {
	var c MainForm
	c.InitWidget()
	c.cur = -1
	lastCreatedMainWidget = &c
	loadTools(config.GetSettings().Tools)

	c.SetPanelPadding(0)
	c.SetCellPadding(0)
	c.toolBar = NewToolBar(&c)
	c.AddWidget(0, 0, c.toolBar)
	c.tabBar = NewTabBar(&c)
	c.AddWidget(1, 0, c.tabBar)

	body := ui.NewPanel()
	body.SetPanelPadding(0)
	body.SetCellPadding(0)
	body.SetXExpandable(true)
	body.SetYExpandable(true)
	c.toolsPane = NewToolsPanel(c.selectTool)
	body.AddWidget(0, 0, c.toolsPane)

	c.canvas = NewCanvasView(&c)
	c.canvas.SetMinWidth(canvasMinWidth)

	c.rightPane = ui.NewPanel()
	c.rightPane.SetPanelPadding(0)
	c.colors = NewColorsPanel(&c)
	c.rightPane.AddWidget(0, 0, c.sectionTitle(func() string { return T().PanelColors }))
	c.rightPane.AddWidget(1, 0, c.colors)
	c.layers = NewLayersPanel(&c)
	c.layers.SetYExpandable(true)
	c.rightPane.AddWidget(2, 0, c.sectionTitle(func() string { return T().PanelLayers }))
	c.rightPane.AddWidget(3, 0, c.layers)
	c.history = NewHistoryPanel(&c)
	c.history.SetYExpandable(true)
	c.rightPane.AddWidget(4, 0, c.sectionTitle(func() string { return T().PanelHistory }))
	c.rightPane.AddWidget(5, 0, c.history)

	c.splitter = ui.NewHSplitter()
	c.splitter.SetWidgets(c.canvas, c.rightPane)
	c.splitter.SetSecondSize(defaultPanelsWidth)
	c.splitter.SetXExpandable(true)
	c.splitter.SetYExpandable(true)
	body.AddWidget(0, 1, c.splitter)
	c.AddWidget(2, 0, body)

	c.status = NewStatusBar(&c)
	c.AddWidget(3, 0, c.status)

	themeListeners = append(themeListeners, func() {
		c.canvas.bufKey = viewKey{}
		clear(c.layers.thumbs)
	})
	c.AddTimer(100, c.timerRefresh)
	c.refreshTool()
	c.colors.Refresh()
	return &c
}

// sectionTitle is the heading of a panel on the right
func (c *MainForm) sectionTitle(text func() string) *ui.Label {
	l := ui.NewLabel("")
	l.SetTextFunc(text)
	l.SetForegroundColor(colorMuted.get())
	themeListeners = append(themeListeners, func() { l.SetForegroundColor(colorMuted.get()) })
	return l
}

// tab returns the current image tab, nil when no image is open
func (c *MainForm) tab() *DocTab {
	if c.cur < 0 || c.cur >= len(c.tabs) {
		return nil
	}
	return c.tabs[c.cur]
}

// doc returns the current image, nil when none is open
func (c *MainForm) doc() *paint.Document {
	if t := c.tab(); t != nil {
		return t.doc
	}
	return nil
}

// tabTitle is the name of the image on its tab, with a star when it has unsaved changes
func (c *MainForm) tabTitle(t *DocTab) string {
	name := docName(t.doc)
	if t.doc.Modified() {
		name += " *"
	}
	return name
}

// docName is the file name of the image, or the name of a new one
func docName(d *paint.Document) string {
	if d.Path != "" {
		return filepath.Base(d.Path)
	}
	return d.Name
}

// addTab opens the image in a new tab and shows it
func (c *MainForm) addTab(d *paint.Document) {
	c.canvas.FinishInteractive()
	t := &DocTab{doc: d, zoom: 1, fit: true}
	c.tabs = append(c.tabs, t)
	c.selectTab(len(c.tabs) - 1)
}

// selectTab shows the image of the tab
func (c *MainForm) selectTab(i int) {
	if i < 0 || i >= len(c.tabs) {
		return
	}
	c.cur = i
	c.canvas.SetTab(c.tabs[i])
	c.docChanged()
	c.canvas.Focus()
}

// docChanged updates what shows the current image: the title, the panels, the status
func (c *MainForm) docChanged() {
	if d := c.doc(); d != nil {
		c.lastVersion = d.Version()
	}
	c.UpdateTitle()
	c.layers.Refresh()
	c.history.Refresh()
	c.updateStatus()
	if f := c.Form(); f != nil {
		f.Update()
	}
}

// timerRefresh follows the changes of the image made by the tools
func (c *MainForm) timerRefresh() {
	d := c.doc()
	if d != nil && d.Version() != c.lastVersion && c.canvas.drag == nil {
		c.docChanged()
	}
}

func (c *MainForm) updateStatus() {
	c.status.Refresh()
}

// UpdateTitle shows the name of the current image in the window title
func (c *MainForm) UpdateTitle() {
	f := c.Form()
	if f == nil {
		return
	}
	title := app.DisplayName
	if t := c.tab(); t != nil {
		title = c.tabTitle(t) + " - " + title
	}
	f.SetTitle(title)
}

// ---- Tools and colors ----

// selectTool makes the tool current; choosing the current one again does nothing
func (c *MainForm) selectTool(id ToolID) {
	if tools.Tool != id {
		c.canvas.FinishInteractive()
		tools.Tool = id
	}
	c.refreshTool()
	c.canvas.Focus()
}

// selectToolByKey chooses the tool of the key; the next one with the same key if one is current
func (c *MainForm) selectToolByKey(key string) {
	var same []ToolID
	for _, t := range toolDefs {
		if t.key == key {
			same = append(same, t.id)
		}
	}
	if len(same) == 0 {
		return
	}
	next := same[0]
	for i, id := range same {
		if id == tools.Tool {
			next = same[(i+1)%len(same)]
		}
	}
	c.selectTool(next)
}

func (c *MainForm) refreshTool() {
	c.toolsPane.SetTool(tools.Tool)
	c.toolBar.Refresh()
	c.updateStatus()
}

// toolOptionsChanged is called when an option of the tool bar is changed
func (c *MainForm) toolOptionsChanged() {
	c.canvas.ToolOptionsChanged()
}

// setColor sets the primary or the secondary color
func (c *MainForm) setColor(col color.NRGBA, primary bool) {
	if primary {
		tools.Primary = col
	} else {
		tools.Secondary = col
	}
	if t := c.canvas.text; t != nil && primary {
		t.color = col
		c.canvas.redrawText()
	}
	c.colors.Refresh()
	c.Form().Update()
}

func (c *MainForm) swapColors() {
	tools.Primary, tools.Secondary = tools.Secondary, tools.Primary
	c.colors.Refresh()
	c.Form().Update()
}

func (c *MainForm) resetColors() {
	tools.Primary = color.NRGBA{0, 0, 0, 255}
	tools.Secondary = color.NRGBA{255, 255, 255, 255}
	c.colors.Refresh()
	c.Form().Update()
}

func (c *MainForm) changeWidth(dir int) {
	w := tools.Width
	step := max(1, float64(int(w/10)))
	tools.Width = max(1, min(1000, w+float64(dir)*step))
	c.toolBar.Refresh()
	c.Form().Update()
}

// ---- Settings and the window ----

func (c *MainForm) settingsPixelGrid() bool {
	return config.GetSettings().PixelGrid
}

// ApplySettings saves the settings and applies what they change in the window
func (c *MainForm) ApplySettings(s config.Settings) {
	languageChanged := s.Language != config.GetSettings().Language
	themeChanged := s.Theme != config.GetSettings().Theme
	if err := config.SetSettings(s); err != nil {
		ui.ShowMessageBox(c, T().Error, err.Error())
	}
	if languageChanged {
		SetLanguage(s.Language)
	}
	if themeChanged {
		ApplyTheme(s.Theme)
	}
	c.Form().Update()
}

// ShowSettings opens the settings dialog
func (c *MainForm) ShowSettings() {
	c.ShowDialog(NewSettingsDialog(config.GetSettings(), c.ApplySettings))
}

// ApplyLanguage updates the texts that do not follow the language by themselves
func (c *MainForm) ApplyLanguage() {
	c.toolBar.fillCombos()
	c.status.fillZoom()
	c.docChanged()
}

// RestoreWindowState applies the saved window layout before the form is shown.
// Returns whether the window should be maximized once shown.
func (c *MainForm) RestoreWindowState(form *ui.Form) (maximized bool) {
	form.SetSize(defaultWindowWidth, defaultWindowHeight)
	state, ok := config.LoadWindowState()
	if !ok {
		return false
	}
	form.SetSize(state.Width, state.Height)
	// 0,0 means the position was not known (the window manager did not report it)
	if state.X != 0 || state.Y != 0 {
		form.Move(state.X, state.Y)
	}
	if state.PanelsWidth > 0 {
		c.splitter.SetSecondSize(state.PanelsWidth)
	}
	return state.Maximized
}

// SaveWindowState remembers the window layout and the tools for the next start
func (c *MainForm) SaveWindowState() {
	form := c.Form()
	state, _ := config.LoadWindowState()
	state.Maximized = form.IsMaximized()
	// The size of a maximized window is the screen size: keep the normal one
	if !state.Maximized {
		state.X, state.Y = form.Position()
		state.Width, state.Height = form.Size()
	}
	state.PanelsWidth = c.splitter.SecondSize()
	config.SaveWindowState(state)
	config.UpdateSettings(func(s *config.Settings) { s.Tools = saveTools() })
}

// Activate puts the focus on the image
func (c *MainForm) Activate() {
	c.canvas.Focus()
}

// BringToFront shows the window on top of the others. Called when the
// application is started again, with the files that start was given.
func (c *MainForm) BringToFront(files []string) {
	raiseWindow(c.Form())
	c.OpenFiles(files)
}

// ShowInstalled tells that this copy has just been installed and started in place of the downloaded one
func (c *MainForm) ShowInstalled() {
	ui.ShowToast(c, T().Installed, ui.ToastSuccess)
}
