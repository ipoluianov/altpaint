package forms

import (
	"image"

	"github.com/ipoluianov/altpaint/config"
	"github.com/ipoluianov/altpaint/paint"
	"github.com/ipoluianov/nui/ui"
)

// BuildMenuBar makes the main menu with its shortcuts
func (c *MainForm) BuildMenuBar() *ui.MenuBar {
	bar := ui.NewMenuBar()
	t := T
	item := func(m *ui.ContextMenu, text func() string, shortcut string, icon string, f func()) *ui.ContextMenuItem {
		it := m.AddItem(text(), f)
		it.SetTextFunc(text)
		if shortcut != "" {
			it.SetShortcut(shortcut)
		}
		if icon != "" {
			setIcon(icon+"-16", func(img image.Image) { it.SetImage(img) })
		}
		return it
	}
	menu := func(text func() string) *ui.ContextMenu {
		m := ui.NewContextMenu(nil)
		bar.AddMenuItem(text(), m).SetTextFunc(text)
		return m
	}

	// ---- File ----
	file := menu(func() string { return t().MenuFile })
	item(file, func() string { return t().CmdNew + "..." }, "Mod+N", "new", c.newImage)
	item(file, func() string { return t().CmdOpen + "..." }, "Mod+O", "open", c.openDialog)
	recent := ui.NewContextMenu(nil)
	recentItem := file.AddItemWithSubmenu(t().CmdOpenRecent, recent)
	recentItem.SetTextFunc(func() string { return t().CmdOpenRecent })
	file.SetOnShow(func() {
		recent.RemoveAllItems()
		files := recentFiles()
		for _, f := range files {
			recent.AddItem(f, func() { c.OpenFile(f) })
		}
		if len(files) == 0 {
			recent.AddItem(t().NoRecentFiles, nil)
		} else {
			recent.AddSeparator()
			recent.AddItem(t().ClearRecent, func() {
				config.UpdateSettings(func(s *config.Settings) { s.RecentFiles = nil })
			})
		}
	})
	file.AddSeparator()
	item(file, func() string { return t().CmdSave }, "Mod+S", "save", func() { c.save(c.tab(), nil) })
	item(file, func() string { return t().CmdSaveAs + "..." }, "Mod+Shift+S", "", func() { c.saveAs(c.tab(), nil) })
	item(file, func() string { return t().CmdSaveAll }, "Mod+Alt+S", "", c.saveAll)
	file.AddSeparator()
	item(file, func() string { return t().CmdClose }, "Mod+W", "", func() { c.closeTab(c.cur, nil) })
	item(file, func() string { return t().CmdCloseAll }, "Mod+Shift+W", "", func() { c.closeAll(nil) })
	file.AddSeparator()
	item(file, func() string { return t().CmdExit }, "Alt+F4", "", func() { c.requestExit(c.quit) })

	// ---- Edit ----
	edit := menu(func() string { return t().MenuEdit })
	item(edit, func() string { return t().CmdUndo }, "Mod+Z", "undo", c.undo)
	item(edit, func() string { return t().CmdRedo }, "Mod+Y", "redo", c.redo)
	edit.AddSeparator()
	item(edit, func() string { return t().CmdCut }, "Mod+X", "cut", c.cut)
	item(edit, func() string { return t().CmdCopy }, "Mod+C", "copy", c.copy)
	item(edit, func() string { return t().CmdCopyMerged }, "Mod+Shift+C", "", c.copyMerged)
	item(edit, func() string { return t().CmdPaste }, "Mod+V", "paste", c.paste)
	item(edit, func() string { return t().CmdPasteNewLayer }, "Mod+Shift+V", "", c.pasteIntoNewLayer)
	item(edit, func() string { return t().CmdPasteNewImage }, "Mod+Alt+V", "", c.pasteIntoNewImage)
	edit.AddSeparator()
	item(edit, func() string { return t().CmdSelectAll }, "Mod+A", "", c.selectAll)
	item(edit, func() string { return t().CmdDeselect }, "Mod+D", "deselect", c.deselect)
	item(edit, func() string { return t().CmdInvertSelection }, "Mod+I", "", c.invertSelection)
	edit.AddSeparator()
	item(edit, func() string { return t().CmdEraseSelection }, "Delete", "", c.eraseSelection)
	item(edit, func() string { return t().CmdFillSelection }, "Backspace", "", c.fillSelection)

	// ---- View ----
	view := menu(func() string { return t().MenuView })
	item(view, func() string { return t().CmdZoomIn }, "Mod+Plus", "", func() { c.canvas.ZoomStep(1, false) })
	item(view, func() string { return t().CmdZoomOut }, "Mod+Minus", "", func() { c.canvas.ZoomStep(-1, false) })
	item(view, func() string { return t().CmdZoomWindow }, "Mod+B", "", c.canvas.ZoomToWindow)
	item(view, func() string { return t().CmdActualSize }, "Mod+0", "", c.canvas.ZoomActual)
	view.AddSeparator()
	item(view, func() string {
		if config.GetSettings().PixelGrid {
			return "✓ " + t().CmdPixelGrid
		}
		return t().CmdPixelGrid
	}, "Mod+Shift+G", "", c.togglePixelGrid)
	view.AddSeparator()
	item(view, func() string { return t().CmdNextImage }, "Mod+Tab", "", func() { c.cycleTab(1) })
	item(view, func() string { return t().CmdPrevImage }, "Mod+Shift+Tab", "", func() { c.cycleTab(-1) })

	// ---- Image ----
	img := menu(func() string { return t().MenuImage })
	item(img, func() string { return t().CmdCropToSelection }, "Mod+Shift+X", "crop", c.cropToSelection)
	item(img, func() string { return t().CmdResize + "..." }, "Mod+R", "", c.resizeImage)
	item(img, func() string { return t().CmdCanvasSize + "..." }, "Mod+Shift+R", "", c.canvasSize)
	img.AddSeparator()
	item(img, func() string { return t().CmdFlipHorizontal }, "", "", func() { c.flipImage(true) })
	item(img, func() string { return t().CmdFlipVertical }, "", "", func() { c.flipImage(false) })
	img.AddSeparator()
	item(img, func() string { return t().CmdRotate90 }, "Mod+H", "", func() { c.rotateImage(1) })
	item(img, func() string { return t().CmdRotate270 }, "Mod+G", "", func() { c.rotateImage(3) })
	item(img, func() string { return t().CmdRotate180 }, "Mod+J", "", func() { c.rotateImage(2) })
	img.AddSeparator()
	item(img, func() string { return t().CmdFlatten }, "Mod+Shift+F", "", c.flatten)

	// ---- Layers ----
	layers := menu(func() string { return t().MenuLayers })
	item(layers, func() string { return t().CmdAddLayer }, "Mod+Shift+N", "layer-add", c.addLayer)
	item(layers, func() string { return t().CmdDeleteLayer }, "Mod+Shift+Delete", "layer-delete", c.deleteLayer)
	item(layers, func() string { return t().CmdDuplicateLayer }, "Mod+Shift+D", "layer-duplicate", c.duplicateLayer)
	item(layers, func() string { return t().CmdMergeDown }, "Mod+M", "layer-merge", c.mergeDown)
	item(layers, func() string { return t().CmdImportFromFile + "..." }, "", "", c.importFromFile)
	layers.AddSeparator()
	item(layers, func() string { return t().CmdFlipLayerHorizontal }, "", "", func() { c.flipLayer(true) })
	item(layers, func() string { return t().CmdFlipLayerVertical }, "", "", func() { c.flipLayer(false) })
	layers.AddSeparator()
	item(layers, func() string { return t().CmdLayerUp }, "Mod+Alt+Up", "layer-up", func() { c.moveLayer(1) })
	item(layers, func() string { return t().CmdLayerDown }, "Mod+Alt+Down", "layer-down", func() { c.moveLayer(-1) })
	layers.AddSeparator()
	item(layers, func() string { return t().CmdLayerProperties + "..." }, "F4", "layer-props", c.layerProperties)

	// ---- Adjustments and effects ----
	adjustmentKeys := map[string]string{
		"AutoLevel": "Mod+Shift+L", "BlackAndWhite": "Mod+Shift+B", "BrightnessContrast": "",
		"HueSaturation": "Mod+Shift+U", "InvertColors": "Mod+Shift+I", "Posterize": "Mod+Shift+P", "Sepia": "Mod+Shift+E",
	}
	adj := menu(func() string { return t().MenuAdjustments })
	for _, f := range paint.Adjustments {
		item(adj, func() string { return filterMenuText(f) }, adjustmentKeys[f.ID], "", func() { c.runFilter(f) })
	}
	eff := menu(func() string { return t().MenuEffects })
	for _, f := range paint.Effects {
		item(eff, func() string { return filterMenuText(f) }, "", "", func() { c.runFilter(f) })
	}

	// ---- Help ----
	help := menu(func() string { return t().MenuHelp })
	item(help, func() string { return t().Help }, "F1", "", func() { openDocs(c, "help_f1") })
	item(help, func() string { return t().Settings + "..." }, "", "", c.ShowSettings)
	help.AddSeparator()
	item(help, func() string { return t().About + "..." }, "", "", func() { c.ShowDialog(NewAboutDialog()) })
	return bar
}

// filterMenuText is the menu item of a filter: "..." when it asks for parameters
func filterMenuText(f *paint.Filter) string {
	if len(f.Params) > 0 {
		return T().FilterName(f.ID) + "..."
	}
	return T().FilterName(f.ID)
}

// AddShortcuts adds the keys that are not in the menu: the tools, the colors, the brush width
func (c *MainForm) AddShortcuts(form *ui.Form) {
	c.addTypingShortcuts(form)
	form.AddShortcut("Mod+NumPlus", func() { c.canvas.ZoomStep(1, false) })
	form.AddShortcut("Mod+NumMinus", func() { c.canvas.ZoomStep(-1, false) })
	form.AddShortcut("Mod+F4", func() { c.closeTab(c.cur, nil) })
}

// typingShortcuts are the keys without modifiers, which type when the Text tool writes
func typingShortcuts(c *MainForm) map[string]func() {
	m := map[string]func(){
		"X": c.swapColors,
		"D": c.resetColors,
		"[": func() { c.changeWidth(-1) },
		"]": func() { c.changeWidth(1) },
	}
	for _, t := range toolDefs {
		m[t.key] = func() { c.selectToolByKey(t.key) }
	}
	return m
}

func (c *MainForm) addTypingShortcuts(form *ui.Form) {
	for key, f := range typingShortcuts(c) {
		form.AddShortcut(key, f)
	}
}

// setTyping turns off the keys that type while the Text tool writes, and back on
func (c *MainForm) setTyping(on bool) {
	form := c.Form()
	if form == nil {
		return
	}
	if on {
		for key := range typingShortcuts(c) {
			form.RemoveShortcut(key)
		}
		return
	}
	c.addTypingShortcuts(form)
}

// cycleTab shows the next (+1) or the previous (-1) image
func (c *MainForm) cycleTab(dir int) {
	if n := len(c.tabs); n > 1 {
		c.canvas.FinishInteractive()
		c.selectTab((c.cur + dir + n) % n)
	}
}

// settingsPixelGridToggle turns the pixel grid on or off and returns whether it is on
func (c *MainForm) settingsPixelGridToggle() bool {
	on := false
	config.UpdateSettings(func(s *config.Settings) {
		s.PixelGrid = !s.PixelGrid
		on = s.PixelGrid
	})
	return on
}
