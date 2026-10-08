package forms

import (
	"errors"
	"image/color"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/ipoluianov/altpaint/config"
	"github.com/ipoluianov/altpaint/paint"
	"github.com/ipoluianov/nui/ui"
)

// Size of the image AltPaint starts with
const (
	defaultImageWidth  = 800
	defaultImageHeight = 600
)

// newDocument makes a new image and opens it
func (c *MainForm) newDocument(w, h int, bg color.NRGBA) {
	c.untitled++
	d := paint.NewDocument(w, h, bg, T().Background)
	d.Name = T().Untitled(c.untitled)
	c.addTab(d)
}

// EnsureDocument opens a new image when none is open
func (c *MainForm) EnsureDocument() {
	if len(c.tabs) == 0 {
		c.newDocument(defaultImageWidth, defaultImageHeight, color.NRGBA{255, 255, 255, 255})
	}
}

// newImage asks for the size of a new image; the size of the image on the clipboard is offered
func (c *MainForm) newImage() {
	c.canvas.FinishInteractive()
	w, h := defaultImageWidth, defaultImageHeight
	if d := c.doc(); d != nil {
		w, h = d.W, d.H
	}
	if img := clipboardPeek(); img != nil {
		w, h = img.Rect.Dx(), img.Rect.Dy()
	}
	c.ShowDialog(NewNewImageDialog(w, h, func(w, h int, bg color.NRGBA) {
		c.newDocument(w, h, bg)
	}))
}

// openFilters are the file types of the open dialog
func openFilters() []ui.FileDialogFilter {
	var all []string
	for _, e := range paint.OpenExts {
		all = append(all, "*"+e)
	}
	return []ui.FileDialogFilter{
		{DisplayName: T().AllImages, Patterns: all},
		{DisplayName: T().AllFiles, Patterns: []string{"*"}},
	}
}

func (c *MainForm) openDialog() {
	c.canvas.FinishInteractive()
	c.Form().ShowOpenFileDialog(ui.OpenFileDialogOptions{Title: T().CmdOpen, AllowMultiple: true, Filters: openFilters()},
		func(paths []string, err error) {
			if err != nil {
				c.showError(err)
				return
			}
			c.OpenFiles(paths)
		})
}

// OpenFiles opens the images, each in its tab; an image already open is shown
func (c *MainForm) OpenFiles(paths []string) {
	for _, p := range paths {
		c.OpenFile(p)
	}
}

// OpenFile opens the image file in a new tab. A new untouched image is
// replaced, so opening a file at the start does not leave an empty tab.
func (c *MainForm) OpenFile(path string) {
	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}
	for i, t := range c.tabs {
		if t.doc.Path == path {
			c.selectTab(i)
			return
		}
	}
	d, err := paint.Open(path, T().Background)
	if err != nil {
		c.showError(errors.New(T().OpenFailed(filepath.Base(path), err.Error())))
		return
	}
	config.AddRecentFile(path)
	if len(c.tabs) == 1 && c.tabs[0].doc.Path == "" && !c.tabs[0].doc.Modified() && len(c.tabs[0].doc.History.Items()) == 0 {
		c.tabs = c.tabs[:0]
		c.cur = -1
	}
	c.addTab(d)
}

// saveFilters are the file types of the save dialog, the format of the image first
func saveFilters() []ui.FileDialogFilter {
	var fs []ui.FileDialogFilter
	for _, f := range paint.Formats {
		var pats []string
		for _, e := range f.Exts {
			pats = append(pats, "*"+e)
		}
		fs = append(fs, ui.FileDialogFilter{DisplayName: f.Name, Patterns: pats})
	}
	return fs
}

// save writes the image to its file, or asks where when it has none; done
// gets whether it was saved
func (c *MainForm) save(t *DocTab, done func(ok bool)) {
	if t == nil {
		return
	}
	c.canvas.FinishInteractive()
	if t.doc.Path == "" || paint.FormatOf(t.doc.Path) == nil {
		c.saveAs(t, done)
		return
	}
	c.writeFile(t, t.doc.Path, done)
}

// saveAs asks where to save the image and saves it there
func (c *MainForm) saveAs(t *DocTab, done func(ok bool)) {
	if t == nil {
		return
	}
	c.canvas.FinishInteractive()
	d := t.doc
	name := docName(d)
	if d.Path == "" || paint.FormatOf(d.Path) == nil {
		ext := ".png"
		if len(d.Layers) > 1 {
			ext = paint.NativeExt
		}
		name = strings.TrimSuffix(name, filepath.Ext(name)) + ext
	}
	opts := ui.SaveFileDialogOptions{Title: T().CmdSaveAs, DefaultFileName: name, Filters: saveFilters()}
	if d.Path != "" {
		opts.DefaultDirectory = filepath.Dir(d.Path)
	}
	c.Form().ShowSaveFileDialog(opts, func(path string, err error) {
		if err != nil {
			c.showError(err)
		}
		if err != nil || path == "" {
			if done != nil {
				done(false)
			}
			return
		}
		if paint.FormatOf(path) == nil {
			path += ".png"
		}
		c.writeFile(t, path, done)
	})
}

// writeFile saves the image to the path in the format of its extension
func (c *MainForm) writeFile(t *DocTab, path string, done func(ok bool)) {
	d := t.doc
	f := paint.FormatOf(path)
	err := paint.Save(d, path, paint.SaveOptions{JPEGQuality: config.GetSettings().JPEGQuality})
	if err != nil {
		c.showError(errors.New(T().SaveFailed(filepath.Base(path), err.Error())))
		if done != nil {
			done(false)
		}
		return
	}
	d.Path = path
	d.MarkSaved()
	config.AddRecentFile(path)
	if f != nil && !f.Layered && len(d.Layers) > 1 {
		ui.ShowToast(c, T().SavedFlattened, ui.ToastInfo)
	} else {
		ui.ShowToast(c, T().Saved(filepath.Base(path)), ui.ToastSuccess)
	}
	c.docChanged()
	if done != nil {
		done(true)
	}
}

// saveAll saves the images with changes, one after another
func (c *MainForm) saveAll() {
	var next func(i int)
	next = func(i int) {
		for ; i < len(c.tabs); i++ {
			if c.tabs[i].doc.Modified() {
				c.save(c.tabs[i], func(ok bool) {
					if ok {
						next(i + 1)
					}
				})
				return
			}
		}
	}
	next(0)
}

// closeTab closes the image, asking to save it when it has changes; done
// gets whether it was closed
func (c *MainForm) closeTab(i int, done func(closed bool)) {
	if i < 0 || i >= len(c.tabs) {
		return
	}
	c.canvas.FinishInteractive()
	t := c.tabs[i]
	finish := func(closed bool) {
		if done != nil {
			done(closed)
		}
	}
	remove := func() {
		idx := slices.Index(c.tabs, t)
		if idx < 0 {
			finish(true)
			return
		}
		c.tabs = slices.Delete(c.tabs, idx, idx+1)
		switch {
		case len(c.tabs) == 0:
			c.cur = -1
			c.canvas.SetTab(nil)
			c.docChanged()
		case c.cur >= idx:
			c.selectTab(max(0, c.cur-1))
		default:
			c.selectTab(c.cur)
		}
		finish(true)
	}
	if !t.doc.Modified() {
		remove()
		return
	}
	c.selectTab(i)
	showSaveChanges(c, docName(t.doc), func(answer int) {
		switch answer {
		case answerYes:
			c.save(t, func(ok bool) {
				if ok {
					remove()
				} else {
					finish(false)
				}
			})
		case answerNo:
			remove()
		default:
			finish(false)
		}
	})
}

// closeAll closes all the images; done gets whether all were closed
func (c *MainForm) closeAll(done func(closed bool)) {
	if len(c.tabs) == 0 {
		if done != nil {
			done(true)
		}
		return
	}
	c.closeTab(len(c.tabs)-1, func(closed bool) {
		if !closed {
			if done != nil {
				done(false)
			}
			return
		}
		c.closeAll(done)
	})
}

// requestExit asks about the unsaved images and runs then when they are all saved or dropped
func (c *MainForm) requestExit(then func()) {
	c.canvas.FinishInteractive()
	var next func(i int)
	next = func(i int) {
		for ; i < len(c.tabs); i++ {
			if c.tabs[i].doc.Modified() {
				t := c.tabs[i]
				c.selectTab(i)
				showSaveChanges(c, docName(t.doc), func(answer int) {
					switch answer {
					case answerYes:
						c.save(t, func(ok bool) {
							if ok {
								next(i + 1)
							}
						})
					case answerNo:
						next(i + 1)
					}
				})
				return
			}
		}
		then()
	}
	next(0)
}

// OnWindowClose is called when the window is being closed: the unsaved images are asked about first
func (c *MainForm) OnWindowClose() bool {
	if c.quitting {
		return true
	}
	c.requestExit(c.quit)
	return false
}

// quit closes the window without asking
func (c *MainForm) quit() {
	c.quitting = true
	c.SaveWindowState()
	c.Form().Close()
}

func (c *MainForm) showError(err error) {
	ui.ShowMessageBox(c, T().Error, err.Error())
}

// recentFiles returns the recent files that still exist
func recentFiles() []string {
	var files []string
	for _, f := range config.GetSettings().RecentFiles {
		if _, err := os.Stat(f); err == nil {
			files = append(files, f)
		}
	}
	return files
}

// fileTitle is the file name without the extension, e.g. for a layer made of the file
func fileTitle(path string) string {
	base := filepath.Base(path)
	return strings.TrimSuffix(base, filepath.Ext(base))
}
