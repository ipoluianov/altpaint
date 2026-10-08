package forms

import (
	"image"
	"image/color"
	"image/draw"

	"github.com/ipoluianov/altpaint/paint"
	"github.com/ipoluianov/nui/ui"
)

// withDoc runs f on the current image after the tool has finished what it
// was making, then shows the changes. Nothing happens without an image.
func (c *MainForm) withDoc(f func(d *paint.Document)) {
	d := c.doc()
	if d == nil {
		return
	}
	c.canvas.FinishInteractive()
	f(d)
	c.docChanged()
}

// ---- Edit ----

func (c *MainForm) undo() {
	c.withDoc(func(d *paint.Document) { d.Undo() })
}

func (c *MainForm) redo() {
	c.withDoc(func(d *paint.Document) { d.Redo() })
}

func (c *MainForm) historyGoTo(pos int) {
	c.withDoc(func(d *paint.Document) { d.GoTo(pos) })
}

func (c *MainForm) selectAll() {
	c.withDoc(func(d *paint.Document) {
		d.SetSelection("SelectAll", paint.SelectRect(d.W, d.H, d.Bounds()))
	})
}

func (c *MainForm) deselect() {
	c.withDoc(func(d *paint.Document) { d.Deselect("Deselect") })
}

func (c *MainForm) invertSelection() {
	c.withDoc(func(d *paint.Document) {
		if d.Sel == nil {
			return
		}
		d.SetSelection("InvertSelection", paint.Inverted(d.W, d.H, d.Sel))
	})
}

// eraseSelection makes the selected pixels of the layer transparent
func (c *MainForm) eraseSelection() {
	c.withDoc(func(d *paint.Document) {
		if d.Sel == nil {
			return
		}
		e := d.BeginEdit()
		e.Draw(func(dst, before *image.RGBA) image.Rectangle {
			return paint.FillMask(dst, before, d.Sel.Mask, color.NRGBA{}, nil, false)
		})
		e.Commit("EraseSelection")
	})
}

// fillSelection fills the selection (or the whole layer) with the primary color
func (c *MainForm) fillSelection() {
	c.withDoc(func(d *paint.Document) {
		e := d.BeginEdit()
		e.Draw(func(dst, before *image.RGBA) image.Rectangle {
			mask := image.NewAlpha(d.Bounds())
			draw.Draw(mask, mask.Rect, image.Opaque, image.Point{}, draw.Src)
			return paint.FillMask(dst, before, mask, tools.Primary, e.Sel(), true)
		})
		e.Commit("FillSelection")
	})
}

// selectedPixels returns the selected pixels of the image (the layer, or
// all the layers merged), cut to the selection; the bounds stay where they are
func selectedPixels(d *paint.Document, merged bool) *image.RGBA {
	src := d.Layer().Img
	if merged {
		src = d.Composite()
	}
	r := d.SelBounds()
	img := image.NewRGBA(r)
	if d.Sel == nil {
		draw.Draw(img, r, src, r.Min, draw.Src)
	} else {
		draw.DrawMask(img, r, src, r.Min, d.Sel.Mask, r.Min, draw.Src)
	}
	return img
}

func (c *MainForm) copySelection(merged bool) {
	c.withDoc(func(d *paint.Document) {
		img := selectedPixels(d, merged)
		if err := clipboardWrite(img); err != nil {
			ui.ShowToast(c, T().ClipboardInternal, ui.ToastInfo)
		}
	})
}

func (c *MainForm) copy()       { c.copySelection(false) }
func (c *MainForm) copyMerged() { c.copySelection(true) }

func (c *MainForm) cut() {
	d := c.doc()
	if d == nil {
		return
	}
	c.copy()
	if d.Sel == nil {
		c.withDoc(func(d *paint.Document) {
			e := d.BeginEdit()
			e.Draw(func(dst, before *image.RGBA) image.Rectangle {
				paint.ClearRect(dst, dst.Rect)
				return dst.Rect
			})
			e.Commit("Cut")
		})
		return
	}
	c.eraseSelection()
}

// pasteTarget is where a pasted image goes: the top left corner of what is seen of the image
func (c *MainForm) pasteTarget(d *paint.Document) image.Point {
	cv := c.canvas
	p := cv.toImage(cv.ScrollX(), cv.ScrollY())
	return image.Pt(max(0, min(d.W-1, int(p.X))), max(0, min(d.H-1, int(p.Y))))
}

// paste puts the image of the clipboard on the current layer as floating
// pixels, selected, with the Move Selected tool to place them
func (c *MainForm) paste() {
	c.pasteInto(false)
}

func (c *MainForm) pasteIntoNewLayer() {
	c.pasteInto(true)
}

func (c *MainForm) pasteInto(newLayer bool) {
	d := c.doc()
	if d == nil {
		c.pasteIntoNewImage()
		return
	}
	img, err := clipboardRead()
	if err != nil || img == nil {
		ui.ShowToast(c, T().ClipboardEmpty, ui.ToastInfo)
		return
	}
	place := func() {
		c.withDoc(func(d *paint.Document) {
			if newLayer {
				d.AddLayer("AddLayer", T().LayerN(len(d.Layers)+1))
			}
			at := c.pasteTarget(d)
			if img.Rect.Dx() > d.W-at.X || img.Rect.Dy() > d.H-at.Y {
				at = image.Point{}
			}
			pixels := image.NewRGBA(image.Rectangle{Min: at, Max: at.Add(img.Rect.Size())})
			draw.Draw(pixels, pixels.Rect, img, img.Rect.Min, draw.Src)
			d.Float("Paste", pixels)
		})
		c.selectTool(ToolMoveSelected)
	}
	if img.Rect.Dx() <= d.W && img.Rect.Dy() <= d.H {
		place()
		return
	}
	// A larger image: the canvas can be made larger for it
	ui.ShowQuestionMessageBoxYesNo(c, T().CmdPaste, T().ExpandCanvasAsk, func() {
		c.withDoc(func(d *paint.Document) {
			d.ResizeCanvas("CanvasSize", max(d.W, img.Rect.Dx()), max(d.H, img.Rect.Dy()), paint.Anchor{X: -1, Y: -1}, nil)
		})
		place()
	}, place)
}

func (c *MainForm) pasteIntoNewImage() {
	img, err := clipboardRead()
	if err != nil || img == nil {
		ui.ShowToast(c, T().ClipboardEmpty, ui.ToastInfo)
		return
	}
	c.untitled++
	d := paint.NewDocumentFromImage(img, T().Background)
	d.Name = T().Untitled(c.untitled)
	c.addTab(d)
}

// ---- Image ----

func (c *MainForm) cropToSelection() {
	c.withDoc(func(d *paint.Document) { d.CropToSelection("CropToSelection") })
}

func (c *MainForm) resizeImage() {
	d := c.doc()
	if d == nil {
		return
	}
	c.canvas.FinishInteractive()
	c.ShowDialog(NewResizeDialog(d.W, d.H, func(w, h int, method paint.Resampling) {
		c.withDoc(func(d *paint.Document) { d.ResizeImage("ResizeImage", w, h, method) })
	}))
}

func (c *MainForm) canvasSize() {
	d := c.doc()
	if d == nil {
		return
	}
	c.canvas.FinishInteractive()
	c.ShowDialog(NewCanvasSizeDialog(d.W, d.H, func(w, h int, anchor paint.Anchor) {
		c.withDoc(func(d *paint.Document) {
			// The new area of the bottom layer gets the secondary color
			bg := image.NewUniform(paint.Premul(tools.Secondary))
			d.ResizeCanvas("CanvasSize", w, h, anchor, bg)
		})
	}))
}

func (c *MainForm) flipImage(horizontal bool) {
	name := "FlipVertical"
	if horizontal {
		name = "FlipHorizontal"
	}
	c.withDoc(func(d *paint.Document) { d.FlipImage(name, horizontal) })
}

func (c *MainForm) rotateImage(quarters int) {
	names := map[int]string{1: "Rotate90", 2: "Rotate180", 3: "Rotate270"}
	c.withDoc(func(d *paint.Document) { d.RotateImage(names[quarters], quarters) })
}

func (c *MainForm) flatten() {
	c.withDoc(func(d *paint.Document) { d.Flatten("Flatten") })
}

// ---- Layers ----

func (c *MainForm) addLayer() {
	c.withDoc(func(d *paint.Document) { d.AddLayer("AddLayer", T().LayerN(len(d.Layers)+1)) })
}

func (c *MainForm) deleteLayer() {
	c.withDoc(func(d *paint.Document) { d.DeleteLayer("DeleteLayer") })
}

func (c *MainForm) duplicateLayer() {
	c.withDoc(func(d *paint.Document) { d.DuplicateLayer("DuplicateLayer", " "+T().CopySuffix) })
}

func (c *MainForm) mergeDown() {
	c.withDoc(func(d *paint.Document) { d.MergeDown("MergeLayerDown") })
}

func (c *MainForm) moveLayer(dir int) {
	name := "MoveLayerUp"
	if dir < 0 {
		name = "MoveLayerDown"
	}
	c.withDoc(func(d *paint.Document) { d.MoveLayer(name, dir) })
}

func (c *MainForm) flipLayer(horizontal bool) {
	name := "FlipLayerVertical"
	if horizontal {
		name = "FlipLayerHorizontal"
	}
	c.withDoc(func(d *paint.Document) { d.FlipLayer(name, horizontal) })
}

func (c *MainForm) selectLayer(i int) {
	c.withDoc(func(d *paint.Document) { d.SetCurrent(i) })
}

func (c *MainForm) toggleLayerVisible(i int) {
	c.withDoc(func(d *paint.Document) {
		l := d.Layers[i]
		d.SetLayerProps("LayerVisibility", l, l.Name, !l.Visible, l.Opacity, l.Blend)
	})
}

func (c *MainForm) layerProperties() {
	d := c.doc()
	if d == nil {
		return
	}
	c.canvas.FinishInteractive()
	l := d.Layer()
	orig := *l
	c.ShowDialog(NewLayerPropertiesDialog(l,
		func(name string, visible bool, opacity uint8, blend paint.BlendMode) {
			// Shown while the dialog is open, without a step
			l.Name, l.Visible, l.Opacity, l.Blend = name, visible, opacity, blend
			d.InvalidateAll()
			c.docChanged()
		},
		func(name string, visible bool, opacity uint8, blend paint.BlendMode) {
			l.Name, l.Visible, l.Opacity, l.Blend = orig.Name, orig.Visible, orig.Opacity, orig.Blend
			c.withDoc(func(d *paint.Document) { d.SetLayerProps("LayerProperties", l, name, visible, opacity, blend) })
		},
		func() {
			l.Name, l.Visible, l.Opacity, l.Blend = orig.Name, orig.Visible, orig.Opacity, orig.Blend
			d.InvalidateAll()
			c.docChanged()
		}))
}

// importFromFile adds the image of a file as a new layer
func (c *MainForm) importFromFile() {
	if c.doc() == nil {
		return
	}
	c.canvas.FinishInteractive()
	c.Form().ShowOpenFileDialog(ui.OpenFileDialogOptions{Title: T().CmdImportFromFile, Filters: openFilters()},
		func(paths []string, err error) {
			if err != nil {
				c.showError(err)
				return
			}
			for _, p := range paths {
				src, err := paint.Open(p, "")
				if err != nil {
					c.showError(err)
					continue
				}
				c.withDoc(func(d *paint.Document) {
					l := paint.NewLayer(fileTitle(p), d.W, d.H)
					draw.Draw(l.Img, l.Img.Rect, src.Flattened(), image.Point{}, draw.Src)
					d.InsertLayer("ImportFromFile", l)
				})
			}
		})
}

// ---- Adjustments and effects ----

// runFilter applies the filter to the selection of the current layer; a
// filter with parameters asks for them first and shows the result as they change
func (c *MainForm) runFilter(f *paint.Filter) {
	d := c.doc()
	if d == nil {
		return
	}
	c.canvas.FinishInteractive()
	e := d.BeginEdit()
	apply := func(params []float64) {
		e.Draw(func(dst, before *image.RGBA) image.Rectangle {
			return paint.ApplyFilter(dst, before, e.Sel(), f, params)
		})
		c.Form().Update()
	}
	if len(f.Params) == 0 {
		apply(nil)
		e.Commit(f.ID)
		c.docChanged()
		return
	}
	apply(f.Defaults())
	c.ShowDialog(NewFilterDialog(f, apply, func(params []float64) {
		apply(params)
		e.Commit(f.ID)
		c.docChanged()
	}, func() {
		e.Cancel()
		c.docChanged()
	}))
}

// ---- View ----

func (c *MainForm) togglePixelGrid() {
	s := c.settingsPixelGridToggle()
	if s && c.canvas.zoom() < gridMinZoom {
		ui.ShowToast(c, T().PixelGridHint, ui.ToastInfo)
	}
	c.Form().Update()
}
