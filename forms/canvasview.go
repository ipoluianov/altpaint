package forms

import (
	"image"
	"image/color"
	"math"

	"github.com/ipoluianov/altpaint/paint"
	"github.com/ipoluianov/nui/ui"
	xdraw "golang.org/x/image/draw"
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
	version          int
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
	key := viewKey{c.doc(), c.doc().Version(), c.zoom(), sx, sy, w, h, ui.IsDarkTheme}
	if key != c.bufKey || c.buf == nil {
		c.render(sx, sy, w, h)
		c.bufKey = key
	}
	cnv.DrawImage(sx, sy, c.buf)
	c.paintOverlays(cnv)
}

// render draws the visible part of the image over the checkers into buf
func (c *CanvasView) render(sx, sy, w, h int) {
	if c.buf == nil || c.buf.Rect.Dx() != w || c.buf.Rect.Dy() != h {
		c.buf = image.NewRGBA(image.Rect(0, 0, max(1, w), max(1, h)))
	}
	buf := c.buf
	ws := colorWorkspace.get()
	for i := 0; i < len(buf.Pix); i += 4 {
		buf.Pix[i], buf.Pix[i+1], buf.Pix[i+2], buf.Pix[i+3] = ws.R, ws.G, ws.B, 255
	}
	d := c.doc()
	comp := d.Composite()
	z := c.zoom()
	ox, oy := c.origin()
	// The image on the screen (the buf coordinates)
	left, top := ox-float64(sx), oy-float64(sy)
	vis := image.Rect(int(math.Floor(left)), int(math.Floor(top)),
		int(math.Ceil(left+float64(d.W)*z)), int(math.Ceil(top+float64(d.H)*z))).Intersect(buf.Rect)
	if vis.Empty() {
		return
	}
	checker := func(x, y int) uint32 {
		if ((x-int(left))/checkerSize+(y-int(top))/checkerSize)%2 == 0 {
			return uint32(colorCheckerLight.R)
		}
		return uint32(colorCheckerDark.R)
	}
	put := func(di int, p []uint8, x, y int) {
		bg := checker(x, y)
		inv := 255 - uint32(p[3])
		buf.Pix[di] = uint8(uint32(p[0]) + bg*inv/255)
		buf.Pix[di+1] = uint8(uint32(p[1]) + bg*inv/255)
		buf.Pix[di+2] = uint8(uint32(p[2]) + bg*inv/255)
		buf.Pix[di+3] = 255
	}

	if z >= 1 {
		// Each pixel of the image is a square of the screen
		cols := make([]int, vis.Dx())
		for i := range cols {
			ix := int((float64(vis.Min.X+i) - left) / z)
			cols[i] = max(0, min(d.W-1, ix)) * 4
		}
		for y := vis.Min.Y; y < vis.Max.Y; y++ {
			iy := max(0, min(d.H-1, int((float64(y)-top)/z)))
			row := comp.Pix[iy*comp.Stride:]
			di := buf.PixOffset(vis.Min.X, y)
			for i, si := range cols {
				put(di+i*4, row[si:si+4:si+4], vis.Min.X+i, y)
			}
		}
		return
	}

	// Zoomed out: the visible part is scaled down with filtering
	src := image.Rect(int(math.Floor((float64(vis.Min.X)-left)/z)), int(math.Floor((float64(vis.Min.Y)-top)/z)),
		int(math.Ceil((float64(vis.Max.X)-left)/z)), int(math.Ceil((float64(vis.Max.Y)-top)/z))).Intersect(comp.Rect)
	dst := image.Rect(int(math.Round(left+float64(src.Min.X)*z)), int(math.Round(top+float64(src.Min.Y)*z)),
		int(math.Round(left+float64(src.Max.X)*z)), int(math.Round(top+float64(src.Max.Y)*z)))
	if dst.Empty() {
		return
	}
	tmp := image.NewRGBA(dst)
	xdraw.BiLinear.Scale(tmp, dst, comp, src, xdraw.Src, nil)
	dst = dst.Intersect(buf.Rect)
	for y := dst.Min.Y; y < dst.Max.Y; y++ {
		di := buf.PixOffset(dst.Min.X, y)
		ti := tmp.PixOffset(dst.Min.X, y)
		for x := dst.Min.X; x < dst.Max.X; x, di, ti = x+1, di+4, ti+4 {
			put(di, tmp.Pix[ti:ti+4:ti+4], x, y)
		}
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
