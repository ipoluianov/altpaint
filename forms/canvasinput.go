package forms

import (
	"image"
	"image/color"
	"math"

	"github.com/ipoluianov/altpaint/paint"
	"github.com/ipoluianov/nui/ui"
)

// roundedRadius is the corner radius of the Rounded Rectangle shape
const roundedRadius = 16

func (c *CanvasView) mouseDown(button ui.MouseButton, x, y int, mods ui.KeyModifiers) bool {
	c.Focus()
	c.mouseX, c.mouseY = x, y
	d := c.doc()
	if d == nil || c.drag != nil {
		return true
	}
	p := c.toImage(x, y)
	dr := &dragState{button: button, tool: tools.Tool, mods: mods, start: p, cur: p,
		startMouse: image.Pt(x, y), scroll: image.Pt(c.ScrollX(), c.ScrollY())}
	if c.panHold {
		dr.tool = ToolPan
	}
	// nui tells the release of the left button only, so the other buttons
	// do what is done with a click
	if button != ui.MouseButtonLeft {
		switch dr.tool {
		case ToolZoom, ToolMagicWand, ToolPaintBucket, ToolColorPicker, ToolText:
		default:
			return true
		}
	}
	if dr.tool != ToolText {
		c.finishText()
	}
	primary := button != ui.MouseButtonRight
	col, other := tools.Primary, tools.Secondary
	if !primary {
		col, other = other, col
	}

	switch dr.tool {
	case ToolPan:
	case ToolZoom:
		dir := 1
		if !primary {
			dir = -1
		}
		c.ZoomStep(dir, true)
		return true
	case ToolRectSelect, ToolEllipseSelect:
	case ToolLassoSelect:
		dr.lasso = []paint.Point{p}
	case ToolMagicWand:
		c.magicWand(p, mods, button)
		return true
	case ToolMoveSelected:
		dr.move = d.BeginMove()
	case ToolMoveSelection:
		if d.Sel == nil {
			return true
		}
		dr.selStart = d.Sel
	case ToolPaintBucket:
		c.bucketFill(p, col)
		return true
	case ToolGradient:
		dr.edit = d.BeginEdit()
	case ToolPaintbrush, ToolEraser, ToolPencil:
		kind := paint.StrokePaint
		if dr.tool == ToolEraser {
			kind = paint.StrokeErase
		}
		s := d.BeginStroke(kind)
		s.Color = col
		s.Width = tools.Width
		s.Antialias = tools.Antialias
		s.Blend = tools.Blend
		s.Pixel = dr.tool == ToolPencil
		s.MoveTo(p)
		dr.stroke = s
	case ToolCloneStamp:
		if mods.Ctrl || mods.Cmd {
			src := image.Pt(int(math.Floor(p.X)), int(math.Floor(p.Y)))
			c.tab.cloneSource = &src
			c.tab.cloneOffset = nil
			return true
		}
		if c.tab.cloneSource == nil {
			ui.ShowToast(c, T().CloneHint, ui.ToastInfo)
			return true
		}
		if c.tab.cloneOffset == nil {
			off := c.tab.cloneSource.Sub(image.Pt(int(math.Floor(p.X)), int(math.Floor(p.Y))))
			c.tab.cloneOffset = &off
		}
		s := d.BeginStroke(paint.StrokeClone)
		s.Width = tools.Width
		s.Antialias = tools.Antialias
		s.CloneOffset = *c.tab.cloneOffset
		s.MoveTo(p)
		dr.stroke = s
	case ToolColorPicker:
		c.pickColor(p, primary)
		if !primary {
			return true
		}
	case ToolText:
		c.startText(p, col)
		return true
	case ToolLine, ToolShapes:
		dr.edit = d.BeginEdit()
	}
	c.drag = dr
	c.main.updateStatus()
	_ = other
	return true
}

func (c *CanvasView) mouseMove(x, y int, mods ui.KeyModifiers) bool {
	c.mouseX, c.mouseY = x, y
	c.mouseIn = true
	dr := c.drag
	if dr == nil {
		c.main.updateStatus()
		return true
	}
	d := c.doc()
	p := c.toImage(x, y)
	dr.cur = p
	dr.mods.Shift = mods.Shift
	if image.Pt(x, y) != dr.startMouse {
		dr.moved = true
	}
	primary := dr.button != ui.MouseButtonRight
	col, other := tools.Primary, tools.Secondary
	if !primary {
		col, other = other, col
	}

	switch dr.tool {
	case ToolPan:
		// The content moves with the mouse: the scroll changes the other way.
		// x is in the content, which moves itself, so the window position is used
		wx, wy := x-c.ScrollX(), y-c.ScrollY()
		sx0, sy0 := dr.startMouse.X-dr.scroll.X, dr.startMouse.Y-dr.scroll.Y
		c.SetScrollX(dr.scroll.X - (wx - sx0))
		c.SetScrollY(dr.scroll.Y - (wy - sy0))
	case ToolLassoSelect:
		last := dr.lasso[len(dr.lasso)-1]
		if math.Hypot(p.X-last.X, p.Y-last.Y)*c.zoom() >= 2 {
			dr.lasso = append(dr.lasso, p)
		}
	case ToolMoveSelected:
		dr.move.To(roundDelta(dr.start, p))
	case ToolMoveSelection:
		delta := roundDelta(dr.start, p)
		d.Sel = dr.selStart.Translated(delta.X, delta.Y)
		if d.Sel == nil {
			d.Sel = dr.selStart
		}
		d.Touch()
	case ToolGradient:
		a, b := dr.start, p
		if mods.Shift {
			b = snapAngle(a, b)
		}
		c0, c1 := col, other
		dr.edit.Draw(func(dst, before *image.RGBA) image.Rectangle {
			return paint.DrawGradient(dst, before, dr.edit.Sel(), tools.Gradient, a, b, c0, c1)
		})
	case ToolPaintbrush, ToolEraser, ToolPencil, ToolCloneStamp:
		dr.stroke.MoveTo(p)
	case ToolColorPicker:
		c.pickColor(p, primary)
	case ToolLine:
		a, b := dr.start, p
		if mods.Shift {
			b = snapAngle(a, b)
		}
		dr.edit.Draw(func(dst, before *image.RGBA) image.Rectangle {
			mask := paint.FillPolygons([]paint.Polygon{paint.LinePolygon(a, b, tools.Width)}, tools.Antialias, dst.Rect)
			return paint.FillMask(dst, before, mask, col, dr.edit.Sel(), tools.Blend)
		})
	case ToolShapes:
		r := shapeRect(dr.start, p, mods.Shift)
		dr.edit.Draw(func(dst, before *image.RGBA) image.Rectangle {
			return drawShape(dst, before, dr.edit.Sel(), r, col, other)
		})
	}
	c.main.updateStatus()
	return true
}

func (c *CanvasView) mouseUp(button ui.MouseButton, x, y int, mods ui.KeyModifiers) bool {
	dr := c.drag
	if dr == nil || dr.button != button {
		return true
	}
	c.drag = nil
	d := c.doc()
	name := string(dr.tool)
	switch dr.tool {
	case ToolRectSelect, ToolEllipseSelect:
		r := selectRect(dr.start, dr.cur, dr.mods.Shift).Intersect(d.Bounds())
		if !dr.moved || r.Empty() {
			if !dr.moved {
				d.Deselect("Deselect")
			}
			break
		}
		var sel *paint.Selection
		if dr.tool == ToolRectSelect {
			sel = paint.SelectRect(d.W, d.H, r)
		} else {
			poly := paint.EllipsePolygon(float64(r.Min.X), float64(r.Min.Y), float64(r.Max.X), float64(r.Max.Y))
			sel = paint.SelectMask(d.W, d.H, paint.FillPolygons([]paint.Polygon{poly}, tools.Antialias, d.Bounds()))
		}
		d.SetSelection(name, paint.Combine(d.Sel, sel, selMode(dr.mods, dr.button)))
	case ToolLassoSelect:
		if len(dr.lasso) < 3 {
			if !dr.moved {
				d.Deselect("Deselect")
			}
			break
		}
		mask := paint.FillPolygons([]paint.Polygon{dr.lasso}, tools.Antialias, d.Bounds())
		d.SetSelection(name, paint.Combine(d.Sel, paint.SelectMask(d.W, d.H, mask), selMode(dr.mods, dr.button)))
	case ToolMoveSelected:
		dr.move.End(name)
	case ToolMoveSelection:
		moved := d.Sel
		d.Sel = dr.selStart
		if moved != dr.selStart {
			d.SetSelection(name, moved)
		}
	case ToolGradient, ToolLine, ToolShapes:
		if dr.moved {
			dr.edit.Commit(name)
		} else {
			dr.edit.Cancel()
		}
	case ToolPaintbrush, ToolEraser, ToolPencil, ToolCloneStamp:
		dr.stroke.End(name)
	}
	c.main.docChanged()
	return true
}

// selMode returns how a new selection joins the current one: by the tool
// option, or Ctrl adds, Alt subtracts, the right button intersects
func selMode(mods ui.KeyModifiers, button ui.MouseButton) paint.CombineMode {
	switch {
	case mods.Ctrl || mods.Cmd:
		return paint.CombineUnion
	case mods.Alt:
		return paint.CombineExclude
	case button == ui.MouseButtonRight:
		return paint.CombineIntersect
	}
	return tools.SelMode
}

// roundDelta returns the move from a to b in whole pixels
func roundDelta(a, b paint.Point) image.Point {
	return image.Pt(int(math.Round(b.X-a.X)), int(math.Round(b.Y-a.Y)))
}

// snapAngle turns b around a to the nearest multiple of 15 degrees
func snapAngle(a, b paint.Point) paint.Point {
	dx, dy := b.X-a.X, b.Y-a.Y
	l := math.Hypot(dx, dy)
	ang := math.Round(math.Atan2(dy, dx)/(math.Pi/12)) * (math.Pi / 12)
	return paint.Point{X: a.X + l*math.Cos(ang), Y: a.Y + l*math.Sin(ang)}
}

// shapeRect returns the rectangle of a shape dragged from a to b, on the pixel corners
func shapeRect(a, b paint.Point, square bool) [4]float64 {
	r := selectRect(paint.Point{X: math.Floor(a.X), Y: math.Floor(a.Y)}, paint.Point{X: math.Floor(b.X), Y: math.Floor(b.Y)}, square)
	return [4]float64{float64(r.Min.X), float64(r.Min.Y), float64(r.Max.X + 1), float64(r.Max.Y + 1)}
}

// shapePolygon returns the shape of the Shapes tool in the rectangle
func shapePolygon(x0, y0, x1, y1 float64) paint.Polygon {
	switch tools.Shape {
	case shapeEllipse:
		return paint.EllipsePolygon(x0, y0, x1, y1)
	case shapeRoundedRectangle:
		return paint.RoundRectPolygon(x0, y0, x1, y1, roundedRadius)
	}
	return paint.RectPolygon(x0, y0, x1, y1)
}

// drawShape draws the shape of the tool options in r: the outline of the
// brush width inside it in col, the inside in col or fillCol
func drawShape(dst, before *image.RGBA, sel *paint.Selection, r [4]float64, col, fillCol color.NRGBA) image.Rectangle {
	x0, y0, x1, y1 := r[0], r[1], r[2], r[3]
	w := tools.Width
	outer := shapePolygon(x0, y0, x1, y1)
	var inner paint.Polygon
	if x1-x0 > 2*w && y1-y0 > 2*w {
		inner = shapePolygon(x0+w, y0+w, x1-w, y1-w)
	}
	var changed image.Rectangle
	// The inside first, then the outline over it, both from what was there
	switch tools.Fill {
	case fillInterior:
		mask := paint.FillPolygons([]paint.Polygon{outer}, tools.Antialias, dst.Rect)
		return paint.FillMask(dst, before, mask, col, sel, tools.Blend)
	case fillOutlineBoth:
		if inner != nil {
			mask := paint.FillPolygons([]paint.Polygon{inner}, tools.Antialias, dst.Rect)
			changed = paint.FillMask(dst, before, mask, fillCol, sel, tools.Blend)
		}
	}
	polys := []paint.Polygon{outer}
	if inner != nil {
		polys = append(polys, inner.Reversed())
	}
	mask := paint.FillPolygons(polys, tools.Antialias, dst.Rect)
	// The outline is painted over the inside just drawn, not over what was there
	return changed.Union(paint.FillMask(dst, dst, mask, col, sel, tools.Blend))
}

// sampleImage returns what the fill, the wand and the picker look at
func (c *CanvasView) sampleImage() *image.RGBA {
	if tools.SampleAll {
		return c.doc().Composite()
	}
	return c.doc().Layer().Img
}

func (c *CanvasView) magicWand(p paint.Point, mods ui.KeyModifiers, button ui.MouseButton) {
	d := c.doc()
	pt := image.Pt(int(math.Floor(p.X)), int(math.Floor(p.Y)))
	if !pt.In(d.Bounds()) {
		return
	}
	mask := paint.FloodMask(c.sampleImage(), pt, tools.Tolerance/100, !tools.Global)
	sel := paint.NewSelection(mask)
	d.SetSelection(string(ToolMagicWand), paint.Combine(d.Sel, sel, selMode(mods, button)))
	c.main.docChanged()
}

func (c *CanvasView) bucketFill(p paint.Point, col color.NRGBA) {
	d := c.doc()
	pt := image.Pt(int(math.Floor(p.X)), int(math.Floor(p.Y)))
	if !pt.In(d.Bounds()) {
		return
	}
	mask := paint.FloodMask(c.sampleImage(), pt, tools.Tolerance/100, !tools.Global)
	e := d.BeginEdit()
	e.Draw(func(dst, before *image.RGBA) image.Rectangle {
		return paint.FillMask(dst, before, mask, col, e.Sel(), tools.Blend)
	})
	e.Commit(string(ToolPaintBucket))
	c.main.docChanged()
}

func (c *CanvasView) pickColor(p paint.Point, primary bool) {
	src := c.sampleImage()
	pt := image.Pt(int(math.Floor(p.X)), int(math.Floor(p.Y)))
	if !pt.In(src.Rect) {
		return
	}
	col := paint.Unpremul(src.RGBAAt(pt.X, pt.Y))
	c.main.setColor(col, primary)
}

// ---- Text ----

func (c *CanvasView) startText(p paint.Point, col color.NRGBA) {
	c.finishText()
	d := c.doc()
	t := &textEdit{edit: d.BeginEdit(), at: image.Pt(int(math.Floor(p.X)), int(math.Floor(p.Y))), color: col}
	c.text = t
	c.main.setTyping(true)
	c.redrawText()
}

func (c *CanvasView) redrawText() {
	t := c.text
	style := tools.Text
	style.Antialias = tools.Antialias
	mask, lay := paint.TextMask(string(t.text), t.at, style)
	t.lay = lay
	if len(t.text) == 0 {
		t.edit.Clear()
		return
	}
	t.edit.Draw(func(dst, before *image.RGBA) image.Rectangle {
		return paint.FillMask(dst, before, mask, t.color, t.edit.Sel(), tools.Blend)
	})
}

// finishText commits the text being typed
func (c *CanvasView) finishText() {
	t := c.text
	if t == nil {
		return
	}
	c.text = nil
	c.main.setTyping(false)
	if len(t.text) > 0 {
		t.edit.Commit(string(ToolText))
		c.main.docChanged()
	}
}

// ToolOptionsChanged applies the changed options to what is being made
func (c *CanvasView) ToolOptionsChanged() {
	if c.text != nil {
		c.redrawText()
	}
	c.Update()
}

// FinishInteractive commits what a tool is making: the text, a drag
func (c *CanvasView) FinishInteractive() {
	c.finishText()
	if dr := c.drag; dr != nil {
		c.mouseUp(dr.button, c.mouseX, c.mouseY, dr.mods)
	}
}

func (c *CanvasView) char(ch rune, mods ui.KeyModifiers) bool {
	t := c.text
	if t == nil || mods.Ctrl || mods.Alt || mods.Cmd || ch < ' ' || ch == 0x7f {
		return false
	}
	t.text = append(t.text, ch)
	c.redrawText()
	return true
}

func (c *CanvasView) keyDown(key ui.Key, mods ui.KeyModifiers) bool {
	if t := c.text; t != nil {
		switch key {
		case ui.KeyEsc:
			c.finishText()
			return true
		case ui.KeyEnter:
			if mods.Ctrl {
				c.finishText()
			} else {
				t.text = append(t.text, '\n')
				c.redrawText()
			}
			return true
		case ui.KeyBackspace:
			if len(t.text) > 0 {
				t.text = t.text[:len(t.text)-1]
				c.redrawText()
			}
			return true
		}
		// The letters type: the key is not taken here, so the character
		// comes, and the tool keys are off while typing (see setTyping)
		return false
	}
	switch key {
	case ui.KeySpace:
		c.panHold = true
		return true
	case ui.KeyEsc:
		if dr := c.drag; dr != nil {
			switch {
			case dr.edit != nil:
				dr.edit.Cancel()
			case dr.stroke != nil:
				dr.stroke.End(string(dr.tool))
			case dr.move != nil:
				dr.move.End(string(dr.tool))
			case dr.selStart != nil:
				c.doc().Sel = dr.selStart
			}
			c.drag = nil
			c.Update()
			return true
		}
	}
	return false
}

func (c *CanvasView) keyUp(key ui.Key, mods ui.KeyModifiers) bool {
	if key == ui.KeySpace {
		c.panHold = false
		return true
	}
	return false
}

func (c *CanvasView) mouseWheel(dx, dy int) bool {
	if c.tab == nil {
		return true
	}
	mods := c.Form().KeyModifiers()
	if mods.Ctrl || mods.Cmd {
		if dy > 0 {
			c.ZoomStep(1, true)
		} else if dy < 0 {
			c.ZoomStep(-1, true)
		}
		return true
	}
	const step = 48
	if dy != 0 {
		c.SetScrollY(c.ScrollY() - dy*step)
	}
	if dx != 0 {
		c.SetScrollX(c.ScrollX() - dx*step)
	}
	return true
}
