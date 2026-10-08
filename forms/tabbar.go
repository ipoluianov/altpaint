package forms

import (
	"github.com/ipoluianov/nui/ui"
)

// TabBar is the row of the open images: a click shows one, the cross or a middle click closes it
type TabBar struct {
	ui.Widget
	main *MainForm

	hover      int
	hoverClose bool
	// The tabs as last laid out: their left edges and widths
	xs, ws []int
}

const (
	tabPadding   = 10
	tabCloseSize = 14
	tabMinWidth  = 70
	tabMaxWidth  = 220
)

func NewTabBar(main *MainForm) *TabBar {
	var c TabBar
	c.InitWidget()
	c.main = main
	c.hover = -1
	c.SetXExpandable(true)
	h := ui.ThemeRowHeight() + 6
	c.SetMinHeight(h)
	c.SetMaxHeight(h)
	c.SetOnPaint(c.paint)
	c.SetOnMouseDown(c.mouseDown)
	c.SetOnMouseMove(func(x, y int, mods ui.KeyModifiers) bool {
		i, onClose := c.tabAt(x)
		if i != c.hover || onClose != c.hoverClose {
			c.hover, c.hoverClose = i, onClose
			c.Form().Update()
		}
		return true
	})
	c.SetOnMouseLeave(func() {
		c.hover = -1
		c.Form().Update()
	})
	c.SetOnMouseDblClick(func(button ui.MouseButton, x, y int, mods ui.KeyModifiers) bool {
		if i, _ := c.tabAt(x); i < 0 && button == ui.MouseButtonLeft {
			main.newImage()
		}
		return true
	})
	return &c
}

// layout measures the tabs
func (c *TabBar) layout() {
	c.xs, c.ws = c.xs[:0], c.ws[:0]
	x := 0
	for _, t := range c.main.tabs {
		w, _, _ := ui.MeasureText(ui.ThemeFontFamily(), ui.ThemeFontSize(), c.main.tabTitle(t))
		w = max(tabMinWidth, min(tabMaxWidth, w+2*tabPadding+tabCloseSize+6))
		c.xs = append(c.xs, x)
		c.ws = append(c.ws, w)
		x += w + 1
	}
}

// tabAt returns the tab at x, and whether x is on its cross; -1 for none
func (c *TabBar) tabAt(x int) (int, bool) {
	c.layout()
	for i := range c.xs {
		if x >= c.xs[i] && x < c.xs[i]+c.ws[i] {
			closeX := c.xs[i] + c.ws[i] - tabPadding/2 - tabCloseSize
			return i, x >= closeX-2 && x < closeX+tabCloseSize+2
		}
	}
	return -1, false
}

func (c *TabBar) paint(cnv *ui.Canvas) {
	c.layout()
	pal := ui.CurrentPalette()
	h := c.Height()
	cnv.FillRect(0, h-1, c.Width(), 1, pal.Divider)
	cnv.SetFontFamily(ui.ThemeFontFamily())
	cnv.SetFontSize(ui.ThemeFontSize())
	cnv.SetVAlign(ui.VAlignCenter)
	for i, t := range c.main.tabs {
		x, w := c.xs[i], c.ws[i]
		active := i == c.main.cur
		switch {
		case active:
			cnv.FillRect(x, 0, w, h, ui.ThemeBackgroundColor(-2, "base"))
			cnv.FillRect(x, 0, w, 2, pal.Highlight)
		case i == c.hover:
			cnv.FillRect(x, 0, w, h-1, ui.ThemeBackgroundColor(4, ""))
		}
		cnv.SetColor(pal.WindowText)
		if !active {
			cnv.SetColor(colorMuted.get())
		}
		cnv.SetHAlign(ui.HAlignLeft)
		cnv.DrawText(x+tabPadding, 0, w-2*tabPadding-tabCloseSize, h, c.main.tabTitle(t))
		// The cross
		cx := x + w - tabPadding/2 - tabCloseSize
		cy := (h - tabCloseSize) / 2
		if i == c.hover && c.hoverClose {
			cnv.FillRect(cx-1, cy-1, tabCloseSize+2, tabCloseSize+2, ui.ThemeBackgroundColor(8, ""))
		}
		if active || i == c.hover {
			col := colorMuted.get()
			cnv.DrawLine(cx+4, cy+4, cx+tabCloseSize-4, cy+tabCloseSize-4, 1, col)
			cnv.DrawLine(cx+tabCloseSize-4, cy+4, cx+4, cy+tabCloseSize-4, 1, col)
		}
		cnv.FillRect(x+w, 4, 1, h-8, pal.Divider)
	}
}

func (c *TabBar) mouseDown(button ui.MouseButton, x, y int, mods ui.KeyModifiers) bool {
	i, onClose := c.tabAt(x)
	if i < 0 {
		return true
	}
	if button == ui.MouseButtonMiddle || onClose && button == ui.MouseButtonLeft {
		c.main.closeTab(i, nil)
		return true
	}
	c.main.selectTab(i)
	return true
}
