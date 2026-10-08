package forms

import (
	"image"
	"image/color"
	"image/draw"
	"math"

	"github.com/ipoluianov/altpaint/paint"
	"github.com/ipoluianov/nui/ui"
)

// DocTab is an open image with the way it is shown
type DocTab struct {
	doc  *paint.Document
	zoom float64
	// fit: the zoom follows the window until the user zooms
	fit              bool
	scrollX, scrollY int
	// cloneOffset is where the Clone Stamp takes the pixels from, relative to where it paints
	cloneSource *image.Point
	cloneOffset *image.Point
}

// zoomLevels are the steps of zooming in and out
var zoomLevels = []float64{0.01, 0.02, 0.03, 0.05, 0.08, 0.12, 0.16, 0.25, 0.33, 0.5, 0.66,
	1, 1.5, 2, 3, 4, 5, 6, 7, 8, 10, 12, 16, 20, 24, 32, 48, 64}

const (
	// canvasMargin is the space around the image when it is larger than the window
	canvasMargin = 24
	checkerSize  = 8
	// The pixel grid is shown from this zoom
	gridMinZoom = 6
)

// viewKey is what the picture of the view depends on, besides the overlays
type viewKey struct {
	doc              *paint.Document
	zoom             float64
	scrollX, scrollY int
	w, h             int
	dark             bool
}

// CanvasView shows the current image and lets the tools work on it
type CanvasView struct {
	ui.Widget
	main *MainForm
	tab  *DocTab

	buf    *image.RGBA
	bufKey viewKey

	mouseIn          bool
	mouseX, mouseY   int // content coordinates
	panHold          bool
	ants             int // the phase of the marching ants
	lastAntsSelected bool

	drag *dragState
	text *textEdit
}

// dragState is a drag of the mouse with a tool
type dragState struct {
	button     ui.MouseButton
	tool       ToolID
	mods       ui.KeyModifiers
	start      paint.Point // image coordinates
	cur        paint.Point
	startMouse image.Point // content coordinates
	scroll     image.Point // the scroll at the start, for panning

	stroke   *paint.Stroke
	edit     *paint.Edit
	move     *paint.Move
	selStart *paint.Selection
	lasso    []paint.Point
	moved    bool
}

// textEdit is the text being typed with the Text tool
type textEdit struct {
	edit  *paint.Edit
	at    image.Point
	text  []rune
	color color.NRGBA
	lay   paint.TextLayout
}

func NewCanvasView(main *MainForm) *CanvasView {
	var c CanvasView
	c.InitWidget()
	c.SetTypeName("CanvasView")
	c.main = main
	c.SetXExpandable(true)
	c.SetYExpandable(true)
	c.SetCanBeFocused(true)
	c.SetAllowScroll(true, true)
	c.SetOnPaint(c.paint)
	c.SetOnMouseDown(c.mouseDown)
	c.SetOnMouseUp(c.mouseUp)
	c.SetOnMouseMove(c.mouseMove)
	c.SetOnMouseWheel(c.mouseWheel)
	c.SetOnMouseEnter(func() { c.mouseIn = true })
	c.SetOnMouseLeave(func() {
		c.mouseIn = false
		c.main.updateStatus()
	})
	c.SetOnKeyDown(c.keyDown)
	c.SetOnKeyUp(c.keyUp)
	c.SetOnChar(c.char)
	c.SetOnScrollChanged(func(x, y int) {
		if c.tab != nil {
			c.tab.scrollX, c.tab.scrollY = x, y
		}
	})
	c.AddTimer(120, c.timerAnts)
	return &c
}

// SetTab shows the image; nil shows nothing
func (c *CanvasView) SetTab(tab *DocTab) {
	c.FinishInteractive()
	c.tab = tab
	c.bufKey = viewKey{}
	if tab == nil {
		return
	}
	c.syncContent()
	c.SetScrollX(tab.scrollX)
	c.SetScrollY(tab.scrollY)
}

func (c *CanvasView) doc() *paint.Document {
	if c.tab == nil {
		return nil
	}
	return c.tab.doc
}

// ---- Geometry ----

// fitZoom returns the zoom that shows the whole image in the window, at most 100%
func (c *CanvasView) fitZoom() float64 {
	d := c.doc()
	w, h := c.Width()-2*canvasMargin, c.Height()-2*canvasMargin
	if d == nil || w <= 0 || h <= 0 {
		return 1
	}
	return min(1, float64(w)/float64(d.W), float64(h)/float64(d.H))
}

func (c *CanvasView) zoom() float64 {
	if c.tab == nil {
		return 1
	}
	if c.tab.fit {
		c.tab.zoom = c.fitZoom()
	}
	return c.tab.zoom
}

// contentSize returns the size of the scrolled area: the image and the margins, at least the window
func (c *CanvasView) contentSize() (int, int) {
	d := c.doc()
	if d == nil {
		return c.Width(), c.Height()
	}
	z := c.zoom()
	w := max(c.Width(), int(math.Ceil(float64(d.W)*z))+2*canvasMargin)
	h := max(c.Height(), int(math.Ceil(float64(d.H)*z))+2*canvasMargin)
	return w, h
}

// origin returns where the image starts in the content
func (c *CanvasView) origin() (float64, float64) {
	d := c.doc()
	cw, ch := c.contentSize()
	z := c.zoom()
	return math.Floor((float64(cw) - float64(d.W)*z) / 2), math.Floor((float64(ch) - float64(d.H)*z) / 2)
}

// toImage converts content coordinates to image ones
func (c *CanvasView) toImage(x, y int) paint.Point {
	ox, oy := c.origin()
	z := c.zoom()
	return paint.Point{X: (float64(x) - ox) / z, Y: (float64(y) - oy) / z}
}

// toContent converts image coordinates to content ones
func (c *CanvasView) toContent(p paint.Point) (float64, float64) {
	ox, oy := c.origin()
	z := c.zoom()
	return ox + p.X*z, oy + p.Y*z
}

// syncContent sets the scrolled area for the image and the zoom
func (c *CanvasView) syncContent() {
	w, h := c.contentSize()
	if c.InnerWidth() != w || c.InnerHeight() != h {
		c.SetInnerSize(w, h)
		if c.ScrollX() > w-c.Width() {
			c.SetScrollX(w - c.Width())
		}
		if c.ScrollY() > h-c.Height() {
			c.SetScrollY(h - c.Height())
		}
	}
}

// SetZoom zooms keeping the image point under (ax, ay) of the window (not the content) in place
func (c *CanvasView) SetZoom(z float64, ax, ay int) {
	if c.tab == nil {
		return
	}
	z = max(zoomLevels[0], min(zoomLevels[len(zoomLevels)-1], z))
	p := c.toImage(ax+c.ScrollX(), ay+c.ScrollY())
	c.tab.zoom = z
	c.tab.fit = false
	c.syncContent()
	x, y := c.toContent(p)
	c.SetScrollX(int(math.Round(x)) - ax)
	c.SetScrollY(int(math.Round(y)) - ay)
	c.main.updateStatus()
}

// ZoomStep zooms to the next level in (+1) or out (-1), at the mouse when it is over the image
func (c *CanvasView) ZoomStep(dir int, atMouse bool) {
	z := c.zoom()
	next := z
	if dir > 0 {
		for _, l := range zoomLevels {
			if l > z*1.001 {
				next = l
				break
			}
		}
	} else {
		for i := len(zoomLevels) - 1; i >= 0; i-- {
			if zoomLevels[i] < z*0.999 {
				next = zoomLevels[i]
				break
			}
		}
	}
	ax, ay := c.Width()/2, c.Height()/2
	if atMouse && c.mouseIn {
		ax, ay = c.mouseX-c.ScrollX(), c.mouseY-c.ScrollY()
	}
	c.SetZoom(next, ax, ay)
}

// ZoomToWindow fits the image into the window
func (c *CanvasView) ZoomToWindow() {
	if c.tab == nil {
		return
	}
	c.tab.fit = true
	c.syncContent()
	c.main.updateStatus()
}

// ZoomActual shows the image at 100%
func (c *CanvasView) ZoomActual() {
	c.SetZoom(1, c.Width()/2, c.Height()/2)
}

// ---- Painting ----

func (c *CanvasView) paint(cnv *ui.Canvas) {
	sx, sy := c.ScrollX(), c.ScrollY()
	w, h := c.Width(), c.Height()
	if c.tab == nil {
		cnv.FillRect(sx, sy, w, h, colorWorkspace.get())
		return
	}
	c.syncContent()
	d := c.doc()
	key := viewKey{d, c.zoom(), sx, sy, w, h, ui.IsDarkTheme}
	if key != c.bufKey || c.buf == nil {
		// Another image, zoom, scroll or size: all is drawn again
		d.TakeViewDirty()
		c.render(image.Rect(0, 0, w, h))
		c.bufKey = key
	} else if r := d.TakeViewDirty(); !r.Empty() {
		// Only what changed, e.g. under the brush
		c.render(c.screenRect(r))
	}
	blit(cnv, sx, sy, c.buf)
	c.paintOverlays(cnv)
}

// blit draws the opaque image at (x, y): copied, not blended, which is what
// takes most of the time of a frame otherwise
func blit(cnv *ui.Canvas, x, y int, img *image.RGBA) {
	if cnv.Scale() != 1 {
		cnv.DrawImage(x, y, img)
		return
	}
	dst := cnv.RGBA()
	at := image.Pt(x+cnv.TranslatedX(), y+cnv.TranslatedY())
	clip := image.Rect(cnv.ClipX(), cnv.ClipY(), cnv.ClipX()+cnv.ClipW(), cnv.ClipY()+cnv.ClipH())
	r := image.Rectangle{Min: at, Max: at.Add(img.Rect.Size())}.Intersect(clip).Intersect(dst.Rect)
	if r.Empty() {
		return
	}
	draw.Draw(dst, r, img, r.Min.Sub(at), draw.Src)
}

// screenRect returns the pixels of buf that show the area of the image
func (c *CanvasView) screenRect(r image.Rectangle) image.Rectangle {
	z := c.zoom()
	left, top := c.imageScreenOrigin()
	return image.Rect(
		int(math.Floor(left+float64(r.Min.X)*z))-1, int(math.Floor(top+float64(r.Min.Y)*z))-1,
		int(math.Ceil(left+float64(r.Max.X)*z))+1, int(math.Ceil(top+float64(r.Max.Y)*z))+1)
}

// imageScreenOrigin returns where the image starts in buf (in the window)
func (c *CanvasView) imageScreenOrigin() (float64, float64) {
	ox, oy := c.origin()
	return ox - float64(c.ScrollX()), oy - float64(c.ScrollY())
}

// render draws the area of buf (window coordinates): the image over the
// checkers, the workspace around it. Each pixel of buf depends only on its
// position, so drawing a part gives the same as drawing all.
func (c *CanvasView) render(area image.Rectangle) {
	w, h := c.Width(), c.Height()
	if c.buf == nil || c.buf.Rect.Dx() != max(1, w) || c.buf.Rect.Dy() != max(1, h) {
		c.buf = image.NewRGBA(image.Rect(0, 0, max(1, w), max(1, h)))
		area = c.buf.Rect
	}
	buf := c.buf
	area = area.Intersect(buf.Rect)
	if area.Empty() {
		return
	}
	d := c.doc()
	comp := d.Composite()
	z := c.zoom()
	left, top := c.imageScreenOrigin()
	il, it := int(math.Floor(left)), int(math.Floor(top))
	img := image.Rect(il, it, int(math.Ceil(left+float64(d.W)*z)), int(math.Ceil(top+float64(d.H)*z)))
	vis := img.Intersect(area)

	// The workspace: the rows above and below the image, and beside it
	ws := colorWorkspace.get()
	wsPix := [4]uint8{ws.R, ws.G, ws.B, 255}
	for y := area.Min.Y; y < area.Max.Y; y++ {
		x0, x1 := area.Min.X, area.Max.X
		if y >= vis.Min.Y && y < vis.Max.Y {
			fillRow(buf, y, x0, vis.Min.X, wsPix)
			fillRow(buf, y, vis.Max.X, x1, wsPix)
		} else {
			fillRow(buf, y, x0, x1, wsPix)
		}
	}
	if vis.Empty() {
		return
	}

	// The checkers: light or dark by the square of the column and of the row
	light, dark := uint32(colorCheckerLight.R), uint32(colorCheckerDark.R)
	colOdd := make([]bool, vis.Dx())
	for i := range colOdd {
		colOdd[i] = ((vis.Min.X+i-il)/checkerSize)%2 == 1
	}

	if z >= 1 {
		// Each pixel of the image is a square of the screen
		cols := make([]int, vis.Dx())
		for i := range cols {
			cols[i] = max(0, min(d.W-1, int((float64(vis.Min.X+i)-left)/z))) * 4
		}
		for y := vis.Min.Y; y < vis.Max.Y; y++ {
			iy := max(0, min(d.H-1, int((float64(y)-top)/z)))
			row := comp.Pix[iy*comp.Stride : iy*comp.Stride+d.W*4]
			out := buf.Pix[buf.PixOffset(vis.Min.X, y):buf.PixOffset(vis.Max.X, y)]
			rowOdd := ((y-it)/checkerSize)%2 == 1
			for i, si := range cols {
				p := row[si : si+4 : si+4]
				o := out[i*4 : i*4+4 : i*4+4]
				if p[3] == 255 {
					o[0], o[1], o[2], o[3] = p[0], p[1], p[2], 255
					continue
				}
				bg := light
				if colOdd[i] != rowOdd {
					bg = dark
				}
				inv := 255 - uint32(p[3])
				k := bg * inv / 255
				o[0], o[1], o[2], o[3] = uint8(uint32(p[0])+k), uint8(uint32(p[1])+k), uint8(uint32(p[2])+k), 255
			}
		}
		return
	}

	// Zoomed out: each pixel of the screen is the average of the pixels of
	// the image it covers
	type span struct{ a, b int }
	colSpan := make([]span, vis.Dx())
	for i := range colSpan {
		x := float64(vis.Min.X + i)
		a := max(0, min(d.W-1, int(math.Floor((x-left)/z))))
		b := max(a+1, min(d.W, int(math.Floor((x+1-left)/z))))
		colSpan[i] = span{a, b}
	}
	sums := make([]uint32, 4*d.W)
	for y := vis.Min.Y; y < vis.Max.Y; y++ {
		fy := float64(y)
		ya := max(0, min(d.H-1, int(math.Floor((fy-top)/z))))
		yb := max(ya+1, min(d.H, int(math.Floor((fy+1-top)/z))))
		// The sums of the columns of the rows this screen row covers
		x0, x1 := colSpan[0].a, colSpan[len(colSpan)-1].b
		clear(sums[x0*4 : x1*4])
		for iy := ya; iy < yb; iy++ {
			row := comp.Pix[iy*comp.Stride:]
			for j := x0 * 4; j < x1*4; j++ {
				sums[j] += uint32(row[j])
			}
		}
		out := buf.Pix[buf.PixOffset(vis.Min.X, y):buf.PixOffset(vis.Max.X, y)]
		rowOdd := ((y-it)/checkerSize)%2 == 1
		rows := uint32(yb - ya)
		for i, sp := range colSpan {
			var r, g, b, a uint32
			for j := sp.a * 4; j < sp.b*4; j += 4 {
				r += sums[j]
				g += sums[j+1]
				b += sums[j+2]
				a += sums[j+3]
			}
			n := rows * uint32(sp.b-sp.a)
			r, g, b, a = r/n, g/n, b/n, a/n
			bg := light
			if colOdd[i] != rowOdd {
				bg = dark
			}
			k := bg * (255 - a) / 255
			o := out[i*4 : i*4+4 : i*4+4]
			o[0], o[1], o[2], o[3] = uint8(r+k), uint8(g+k), uint8(b+k), 255
		}
	}
}

// fillRow fills the pixels x0..x1 of the row with the color
func fillRow(img *image.RGBA, y, x0, x1 int, col [4]uint8) {
	if x0 >= x1 {
		return
	}
	row := img.Pix[img.PixOffset(x0, y):img.PixOffset(x1, y)]
	copy(row[:4], col[:])
	for filled := 4; filled < len(row); filled *= 2 {
		copy(row[filled:], row[:filled])
	}
}

// paintOverlays draws what is over the image: the edge, the pixel grid, the selection, the tool
func (c *CanvasView) paintOverlays(cnv *ui.Canvas) {
	d := c.doc()
	z := c.zoom()
	ox, oy := c.origin()
	iw, ih := int(math.Round(float64(d.W)*z)), int(math.Round(float64(d.H)*z))
	edge := colorImageEdge.get()
	cnv.SetColor(edge)
	cnv.DrawRect(int(ox)-1, int(oy)-1, iw+2, ih+2)

	if z >= gridMinZoom && c.main.settingsPixelGrid() {
		c.paintGrid(cnv, ox, oy, z)
	}

	if d.Sel != nil {
		c.paintAnts(cnv, d.Sel.Outline())
	}
	c.paintTool(cnv)
}

// paintGrid draws the borders of the visible pixels
func (c *CanvasView) paintGrid(cnv *ui.Canvas, ox, oy, z float64) {
	d := c.doc()
	sx, sy := c.ScrollX(), c.ScrollY()
	gridColor := color.RGBA{128, 128, 128, 160}
	x0 := max(0, int((float64(sx)-ox)/z))
	x1 := min(d.W, int((float64(sx+c.Width())-ox)/z)+1)
	y0 := max(0, int((float64(sy)-oy)/z))
	y1 := min(d.H, int((float64(sy+c.Height())-oy)/z)+1)
	top, bottom := int(oy+float64(y0)*z), int(oy+float64(y1)*z)
	left, right := int(ox+float64(x0)*z), int(ox+float64(x1)*z)
	for x := x0; x <= x1; x++ {
		px := int(ox + float64(x)*z)
		cnv.DrawLine(px, top, px, bottom, 1, gridColor)
	}
	for y := y0; y <= y1; y++ {
		py := int(oy + float64(y)*z)
		cnv.DrawLine(left, py, right, py, 1, gridColor)
	}
}

// paintAnts draws the border of the selection as moving black and white dashes
func (c *CanvasView) paintAnts(cnv *ui.Canvas, segs []paint.Segment) {
	z := c.zoom()
	ox, oy := c.origin()
	sx, sy := float64(c.ScrollX()), float64(c.ScrollY())
	w, h := float64(c.Width()), float64(c.Height())
	white := color.RGBA{255, 255, 255, 255}
	black := color.RGBA{0, 0, 0, 255}
	const dash = 4
	for _, s := range segs {
		x1, y1 := math.Round(ox+float64(s.X1)*z), math.Round(oy+float64(s.Y1)*z)
		x2, y2 := math.Round(ox+float64(s.X2)*z), math.Round(oy+float64(s.Y2)*z)
		if max(x1, x2) < sx-1 || min(x1, x2) > sx+w+1 || max(y1, y2) < sy-1 || min(y1, y2) > sy+h+1 {
			continue
		}
		// Only the visible part is dashed
		x1, x2 = max(x1, sx-dash*2), min(x2, sx+w+dash*2)
		y1, y2 = max(y1, sy-dash*2), min(y2, sy+h+dash*2)
		cnv.DrawLine(int(x1), int(y1), int(x2), int(y2), 1, white)
		if y1 == y2 {
			start := int(x1) - ((int(x1)+int(y1)+c.ants)%(2*dash)+2*dash)%(2*dash)
			for x := start; x < int(x2); x += 2 * dash {
				a, b := max(x, int(x1)), min(x+dash, int(x2))
				if a < b {
					cnv.DrawLine(a, int(y1), b, int(y1), 1, black)
				}
			}
		} else {
			start := int(y1) - ((int(x1)+int(y1)+c.ants)%(2*dash)+2*dash)%(2*dash)
			for y := start; y < int(y2); y += 2 * dash {
				a, b := max(y, int(y1)), min(y+dash, int(y2))
				if a < b {
					cnv.DrawLine(int(x1), a, int(x1), b, 1, black)
				}
			}
		}
	}
}

// paintTool draws what the tool shows: the shape being selected, the brush, the text box
func (c *CanvasView) paintTool(cnv *ui.Canvas) {
	z := c.zoom()
	if dr := c.drag; dr != nil {
		switch dr.tool {
		case ToolRectSelect, ToolEllipseSelect:
			r := selectRect(dr.start, dr.cur, dr.mods.Shift)
			if dr.tool == ToolEllipseSelect {
				c.paintPolygon(cnv, paint.EllipsePolygon(float64(r.Min.X), float64(r.Min.Y), float64(r.Max.X), float64(r.Max.Y)))
			} else {
				c.paintAnts(cnv, rectSegments(r))
			}
		case ToolLassoSelect:
			c.paintPolyline(cnv, dr.lasso, false)
		case ToolGradient:
			c.paintPolyline(cnv, []paint.Point{dr.start, dr.cur}, true)
			for _, p := range []paint.Point{dr.start, dr.cur} {
				x, y := c.toContent(p)
				c.paintHandle(cnv, int(x), int(y))
			}
		}
	}
	if t := c.text; t != nil {
		b := t.lay.Bounds.Union(t.lay.Caret).Inset(-2)
		c.paintAnts(cnv, rectSegments(b))
		x, y := c.toContent(paint.Point{X: float64(t.lay.Caret.Min.X), Y: float64(t.lay.Caret.Min.Y)})
		cnv.FillRect(int(x), int(y), max(1, int(math.Round(float64(t.lay.Caret.Dx())*z))), int(float64(t.lay.Caret.Dy())*z), ui.CurrentPalette().Highlight)
	}
	if !c.mouseIn || c.drag != nil && c.drag.tool == ToolPan {
		return
	}
	p := c.toImage(c.mouseX, c.mouseY)
	switch tools.Tool {
	case ToolPaintbrush, ToolEraser, ToolCloneStamp:
		r := tools.Width / 2
		c.paintPolygon(cnv, paint.EllipsePolygon(p.X-r, p.Y-r, p.X+r, p.Y+r))
		if tools.Tool == ToolCloneStamp && c.tab.cloneSource != nil {
			src := paint.Point{X: float64(c.tab.cloneSource.X) + 0.5, Y: float64(c.tab.cloneSource.Y) + 0.5}
			if c.tab.cloneOffset != nil {
				src = paint.Point{X: p.X + float64(c.tab.cloneOffset.X), Y: p.Y + float64(c.tab.cloneOffset.Y)}
			}
			x, y := c.toContent(src)
			c.paintCross(cnv, int(x), int(y))
		}
	case ToolPencil:
		px, py := math.Floor(p.X), math.Floor(p.Y)
		c.paintPolygon(cnv, paint.RectPolygon(px, py, px+1, py+1))
	default:
		if c.drag == nil {
			c.paintCross(cnv, c.mouseX, c.mouseY)
		}
	}
}

// paintPolygon draws the closed outline of a shape given in image coordinates
func (c *CanvasView) paintPolygon(cnv *ui.Canvas, poly paint.Polygon) {
	c.paintPolyline(cnv, append(poly, poly[0]), true)
}

// paintPolyline draws the line in black with white around it, so it is seen on any image
func (c *CanvasView) paintPolyline(cnv *ui.Canvas, pts []paint.Point, thin bool) {
	if len(pts) < 2 {
		return
	}
	for pass := range 2 {
		col := color.RGBA{255, 255, 255, 200}
		if pass == 1 {
			col = color.RGBA{0, 0, 0, 255}
		}
		for i := 1; i < len(pts); i++ {
			x1, y1 := c.toContent(pts[i-1])
			x2, y2 := c.toContent(pts[i])
			if pass == 0 {
				cnv.DrawLine(int(x1)+1, int(y1)+1, int(x2)+1, int(y2)+1, 1, col)
			} else {
				cnv.DrawLine(int(x1), int(y1), int(x2), int(y2), 1, col)
			}
		}
	}
}

func (c *CanvasView) paintHandle(cnv *ui.Canvas, x, y int) {
	cnv.FillRect(x-4, y-4, 9, 9, color.RGBA{0, 0, 0, 255})
	cnv.FillRect(x-3, y-3, 7, 7, color.RGBA{255, 255, 255, 255})
}

func (c *CanvasView) paintCross(cnv *ui.Canvas, x, y int) {
	for pass := range 2 {
		col, d := color.RGBA{255, 255, 255, 255}, 1
		if pass == 1 {
			col, d = color.RGBA{0, 0, 0, 255}, 0
		}
		cnv.DrawLine(x-7+d, y+d, x-2+d, y+d, 1, col)
		cnv.DrawLine(x+3+d, y+d, x+8+d, y+d, 1, col)
		cnv.DrawLine(x+d, y-7+d, x+d, y-2+d, 1, col)
		cnv.DrawLine(x+d, y+3+d, x+d, y+8+d, 1, col)
	}
}

// rectSegments returns the edges of the rectangle, for the ants
func rectSegments(r image.Rectangle) []paint.Segment {
	return []paint.Segment{
		{X1: r.Min.X, Y1: r.Min.Y, X2: r.Max.X, Y2: r.Min.Y},
		{X1: r.Min.X, Y1: r.Max.Y, X2: r.Max.X, Y2: r.Max.Y},
		{X1: r.Min.X, Y1: r.Min.Y, X2: r.Min.X, Y2: r.Max.Y},
		{X1: r.Max.X, Y1: r.Min.Y, X2: r.Max.X, Y2: r.Max.Y},
	}
}

// selectRect returns the pixels between the two points; square makes it a square
func selectRect(a, b paint.Point, square bool) image.Rectangle {
	if square {
		dx, dy := b.X-a.X, b.Y-a.Y
		s := max(math.Abs(dx), math.Abs(dy))
		b = paint.Point{X: a.X + math.Copysign(s, dx), Y: a.Y + math.Copysign(s, dy)}
	}
	return image.Rect(int(math.Round(a.X)), int(math.Round(a.Y)), int(math.Round(b.X)), int(math.Round(b.Y))).Canon()
}

func (c *CanvasView) timerAnts() {
	d := c.doc()
	selected := d != nil && (d.Sel != nil || c.text != nil || c.drag != nil && c.drag.tool == ToolRectSelect)
	if selected {
		c.ants = (c.ants + 1) % 8
		c.Update()
	} else if c.lastAntsSelected {
		c.Update()
	}
	c.lastAntsSelected = selected
}

// Update repaints the window
func (c *CanvasView) Update() {
	if f := c.Form(); f != nil {
		f.Update()
	}
}
