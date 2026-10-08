package forms

import (
	"fmt"

	"github.com/ipoluianov/altpaint/paint"
	"github.com/ipoluianov/nui/ui"
	"github.com/ipoluianov/nui/ui/i18n"
)

// Strings are all the texts of the application. A misspelled field is a
// compile error; a field a language leaves empty falls back to English
// (strings_test.go checks that none is left, the maps included).
type Strings struct {
	Error string

	// Menus; & marks the letter of Alt+letter
	MenuFile        string
	MenuEdit        string
	MenuView        string
	MenuImage       string
	MenuLayers      string
	MenuAdjustments string
	MenuEffects     string
	MenuHelp        string

	// Commands
	CmdNew                 string
	CmdOpen                string
	CmdOpenRecent          string
	CmdSave                string
	CmdSaveAs              string
	CmdSaveAll             string
	CmdClose               string
	CmdCloseAll            string
	CmdExit                string
	CmdUndo                string
	CmdRedo                string
	CmdCut                 string
	CmdCopy                string
	CmdCopyMerged          string
	CmdPaste               string
	CmdPasteNewLayer       string
	CmdPasteNewImage       string
	CmdSelectAll           string
	CmdDeselect            string
	CmdInvertSelection     string
	CmdEraseSelection      string
	CmdFillSelection       string
	CmdZoomIn              string
	CmdZoomOut             string
	CmdZoomWindow          string
	CmdActualSize          string
	CmdPixelGrid           string
	CmdNextImage           string
	CmdPrevImage           string
	CmdCropToSelection     string
	CmdResize              string
	CmdCanvasSize          string
	CmdFlipHorizontal      string
	CmdFlipVertical        string
	CmdRotate90            string
	CmdRotate270           string
	CmdRotate180           string
	CmdFlatten             string
	CmdAddLayer            string
	CmdDeleteLayer         string
	CmdDuplicateLayer      string
	CmdMergeDown           string
	CmdImportFromFile      string
	CmdFlipLayerHorizontal string
	CmdFlipLayerVertical   string
	CmdLayerUp             string
	CmdLayerDown           string
	CmdLayerProperties     string
	NoRecentFiles          string
	ClearRecent            string

	// Panels
	PanelColors  string
	PanelLayers  string
	PanelHistory string
	Primary      string
	Secondary    string
	SwapColors   string
	ResetColors  string
	PaletteHint  string
	HistoryNew   string
	HistoryOpen  string

	// Tool bar
	BrushWidth     string
	Antialiasing   string
	AlphaBlending  string
	Shape          string
	Shapes         [3]string // rectangle, rounded rectangle, ellipse
	FillMode       string
	FillModes      [3]string // outline, filled, outline and fill
	GradientType   string
	Gradients      [5]string // by paint.GradientKind
	Tolerance      string
	FloodMode      string
	FloodModes     [2]string // contiguous, global
	Sampling       string
	SamplingModes  [2]string // layer, image
	SelectionMode  string
	SelectionModes [5]string // by paint.CombineMode
	FontSize       string
	Bold           string
	Italic         string
	Monospace      string
	ZoomWindow     string
	// The status bar: the size of the selection and of the image
	StatusSelection func(w, h int) string
	StatusImage     func(w, h int) string

	// Names of the tools, their hints in the status bar, by ToolID
	Tools     map[ToolID]string
	ToolHints map[ToolID]string
	// Names of the blend modes, by paint.BlendMode
	Blends [14]string
	// Names of the adjustments and effects, and of their parameters, by id
	Filters map[string]string
	Params  map[string]string
	// Names of the history steps that are not tools or filters
	History map[string]string

	// Images and files
	Untitled          func(n int) string
	Background        string
	LayerN            func(n int) string
	CopySuffix        string
	AllImages         string
	AllFiles          string
	OpenFailed        func(name, err string) string
	SaveFailed        func(name, err string) string
	Saved             func(name string) string
	SavedFlattened    string
	UnsavedTitle      string
	SaveChangesAsk    func(name string) string
	Save              string
	DontSave          string
	ClipboardEmpty    string
	ClipboardInternal string
	ExpandCanvasAsk   string
	CloneHint         string
	PixelGridHint     string

	// Dialogs
	Width                 string
	Height                string
	Pixels                string
	BackgroundLabel       string
	BackgroundWhite       string
	BackgroundSecondary   string
	BackgroundTransparent string
	ByPercentage          string
	KeepAspect            string
	Resampling            string
	Resamplings           [3]string // by paint.Resampling
	Anchor                string
	Name                  string
	Visible               string
	BlendMode             string
	Opacity               string

	// Settings, about, install
	Settings         string
	Help             string
	About            string
	Language         string
	LanguageSystem   string
	Theme            string
	ThemeDark        string
	ThemeLight       string
	JPEGQuality      string
	AboutTitle       func(app string) string
	Version          string
	Author           string
	License          string
	VisitWebsite     string
	Close            string
	Install          string
	Update           string
	Uninstall        string
	InstallAsk       func(dir string) string
	InstallFailed    func(err string) string
	Installed        string
	UninstallAsk     func(dir string) string
	Uninstalled      string
	UninstallRunning string
}

// ToolName returns the name of the tool
func (s *Strings) ToolName(id ToolID) string {
	return lookup(s.Tools, en.Tools, id, string(id))
}

// ToolHint returns what the tool does and how, for the status bar
func (s *Strings) ToolHint(id ToolID) string {
	return lookup(s.ToolHints, en.ToolHints, id, "")
}

// BlendName returns the name of the blend mode
func (s *Strings) BlendName(m paint.BlendMode) string {
	if m < 0 || int(m) >= len(s.Blends) {
		return ""
	}
	return s.Blends[m]
}

// GradientName returns the name of the gradient kind
func (s *Strings) GradientName(k paint.GradientKind) string {
	if k < 0 || int(k) >= len(s.Gradients) {
		return ""
	}
	return s.Gradients[k]
}

// FilterName returns the name of the adjustment or effect
func (s *Strings) FilterName(id string) string {
	return lookup(s.Filters, en.Filters, id, id)
}

// ParamName returns the name of a parameter of a filter
func (s *Strings) ParamName(id string) string {
	return lookup(s.Params, en.Params, id, id)
}

// HistoryName returns the name of a step of the history: a tool, a filter or a command
func (s *Strings) HistoryName(id string) string {
	if v, ok := s.History[id]; ok {
		return v
	}
	if v, ok := s.Tools[ToolID(id)]; ok {
		return v
	}
	if v, ok := s.Filters[id]; ok {
		return v
	}
	return lookup(en.History, en.History, id, id)
}

// lookup finds the text in the language, then in English, then gives def
func lookup[K comparable](m, base map[K]string, key K, def string) string {
	if v, ok := m[key]; ok {
		return v
	}
	if v, ok := base[key]; ok {
		return v
	}
	return def
}

var languages = []struct{ tag, name string }{
	{"en", "English"},
	{"ru", "Русский"},
}

var en = Strings{
	Error: "Error",

	MenuFile:        "&File",
	MenuEdit:        "&Edit",
	MenuView:        "&View",
	MenuImage:       "&Image",
	MenuLayers:      "&Layers",
	MenuAdjustments: "&Adjustments",
	MenuEffects:     "Effe&cts",
	MenuHelp:        "&Help",

	CmdNew:                 "New",
	CmdOpen:                "Open",
	CmdOpenRecent:          "Open Recent",
	CmdSave:                "Save",
	CmdSaveAs:              "Save As",
	CmdSaveAll:             "Save All",
	CmdClose:               "Close",
	CmdCloseAll:            "Close All",
	CmdExit:                "Exit",
	CmdUndo:                "Undo",
	CmdRedo:                "Redo",
	CmdCut:                 "Cut",
	CmdCopy:                "Copy",
	CmdCopyMerged:          "Copy Merged",
	CmdPaste:               "Paste",
	CmdPasteNewLayer:       "Paste into New Layer",
	CmdPasteNewImage:       "Paste into New Image",
	CmdSelectAll:           "Select All",
	CmdDeselect:            "Deselect All",
	CmdInvertSelection:     "Invert Selection",
	CmdEraseSelection:      "Erase Selection",
	CmdFillSelection:       "Fill Selection",
	CmdZoomIn:              "Zoom In",
	CmdZoomOut:             "Zoom Out",
	CmdZoomWindow:          "Zoom to Window",
	CmdActualSize:          "Actual Size",
	CmdPixelGrid:           "Pixel Grid",
	CmdNextImage:           "Next Image",
	CmdPrevImage:           "Previous Image",
	CmdCropToSelection:     "Crop to Selection",
	CmdResize:              "Resize",
	CmdCanvasSize:          "Canvas Size",
	CmdFlipHorizontal:      "Flip Horizontal",
	CmdFlipVertical:        "Flip Vertical",
	CmdRotate90:            "Rotate 90° Clockwise",
	CmdRotate270:           "Rotate 90° Counter-clockwise",
	CmdRotate180:           "Rotate 180°",
	CmdFlatten:             "Flatten",
	CmdAddLayer:            "Add New Layer",
	CmdDeleteLayer:         "Delete Layer",
	CmdDuplicateLayer:      "Duplicate Layer",
	CmdMergeDown:           "Merge Layer Down",
	CmdImportFromFile:      "Import from File",
	CmdFlipLayerHorizontal: "Flip Layer Horizontal",
	CmdFlipLayerVertical:   "Flip Layer Vertical",
	CmdLayerUp:             "Move Layer Up",
	CmdLayerDown:           "Move Layer Down",
	CmdLayerProperties:     "Layer Properties",
	NoRecentFiles:          "(none)",
	ClearRecent:            "Clear the List",

	PanelColors:  "Colors",
	PanelLayers:  "Layers",
	PanelHistory: "History",
	Primary:      "Primary",
	Secondary:    "Secondary",
	SwapColors:   "Swap colors",
	ResetColors:  "Black and white",
	PaletteHint:  "Click: primary color, right click: secondary",
	HistoryNew:   "New Image",
	HistoryOpen:  "Open Image",

	BrushWidth:      "Brush width:",
	Antialiasing:    "Antialiasing",
	AlphaBlending:   "Alpha blending",
	Shape:           "Shape:",
	Shapes:          [3]string{"Rectangle", "Rounded Rectangle", "Ellipse"},
	FillMode:        "Fill:",
	FillModes:       [3]string{"Outline", "Filled", "Outline and fill"},
	GradientType:    "Type:",
	Gradients:       [5]string{"Linear", "Linear (reflected)", "Diamond", "Radial", "Conical"},
	Tolerance:       "Tolerance, %:",
	FloodMode:       "Mode:",
	FloodModes:      [2]string{"Contiguous", "Global"},
	Sampling:        "Sampling:",
	SamplingModes:   [2]string{"Layer", "Image"},
	SelectionMode:   "Selection mode:",
	SelectionModes:  [5]string{"Replace", "Union (Ctrl)", "Exclude (Alt)", "Intersect (right click)", "Xor"},
	FontSize:        "Size:",
	Bold:            "Bold",
	Italic:          "Italic",
	Monospace:       "Monospace",
	ZoomWindow:      "Window",
	StatusSelection: func(w, h int) string { return fmt.Sprintf("Selection %d × %d", w, h) },
	StatusImage:     func(w, h int) string { return fmt.Sprintf("Image %d × %d", w, h) },

	Tools: map[ToolID]string{
		ToolRectSelect:    "Rectangle Select",
		ToolEllipseSelect: "Ellipse Select",
		ToolLassoSelect:   "Lasso Select",
		ToolMagicWand:     "Magic Wand",
		ToolMoveSelected:  "Move Selected Pixels",
		ToolMoveSelection: "Move Selection",
		ToolZoom:          "Zoom",
		ToolPan:           "Pan",
		ToolPaintBucket:   "Paint Bucket",
		ToolGradient:      "Gradient",
		ToolPaintbrush:    "Paintbrush",
		ToolEraser:        "Eraser",
		ToolPencil:        "Pencil",
		ToolColorPicker:   "Color Picker",
		ToolCloneStamp:    "Clone Stamp",
		ToolText:          "Text",
		ToolLine:          "Line",
		ToolShapes:        "Shapes",
	},
	ToolHints: map[ToolID]string{
		ToolRectSelect:    "Drag to select. Shift: square. Ctrl adds, Alt subtracts. A click deselects",
		ToolEllipseSelect: "Drag to select. Shift: circle. Ctrl adds, Alt subtracts. A click deselects",
		ToolLassoSelect:   "Draw around what to select. Ctrl adds, Alt subtracts",
		ToolMagicWand:     "Click to select an area of similar color. Ctrl adds, Alt subtracts, right click intersects",
		ToolMoveSelected:  "Drag to move the selected pixels",
		ToolMoveSelection: "Drag to move the selection outline",
		ToolZoom:          "Click to zoom in, right click to zoom out. Ctrl + wheel zooms with any tool",
		ToolPan:           "Drag to scroll. With any tool, hold Space and drag",
		ToolPaintBucket:   "Click to fill an area of similar color; right click fills with the secondary color",
		ToolGradient:      "Drag to draw a gradient from the primary to the secondary color. Shift: 15° steps",
		ToolPaintbrush:    "Drag to paint. [ and ] change the width",
		ToolEraser:        "Drag to erase to transparency",
		ToolPencil:        "Drag to draw one-pixel lines",
		ToolColorPicker:   "Click to pick the primary color, right click for the secondary",
		ToolCloneStamp:    "Ctrl + click sets the source, then paint to copy from it",
		ToolText:          "Click and type. Enter: new line, Esc: finish",
		ToolLine:          "Drag to draw a line. Shift: 15° steps",
		ToolShapes:        "Drag to draw a shape. Shift: square or circle",
	},
	Blends: [14]string{"Normal", "Multiply", "Additive", "Color Burn", "Color Dodge", "Reflect", "Glow",
		"Overlay", "Difference", "Negation", "Lighten", "Darken", "Screen", "Xor"},
	Filters: map[string]string{
		"AutoLevel":          "Auto-Level",
		"BlackAndWhite":      "Black and White",
		"BrightnessContrast": "Brightness / Contrast",
		"HueSaturation":      "Hue / Saturation",
		"InvertColors":       "Invert Colors",
		"Posterize":          "Posterize",
		"Sepia":              "Sepia",
		"GaussianBlur":       "Gaussian Blur",
		"MotionBlur":         "Motion Blur",
		"Sharpen":            "Sharpen",
		"AddNoise":           "Add Noise",
		"Pixelate":           "Pixelate",
		"Emboss":             "Emboss",
		"EdgeDetect":         "Edge Detect",
		"Vignette":           "Vignette",
	},
	Params: map[string]string{
		"Brightness":      "Brightness",
		"Contrast":        "Contrast",
		"Hue":             "Hue",
		"Saturation":      "Saturation",
		"Lightness":       "Lightness",
		"Levels":          "Levels",
		"Radius":          "Radius",
		"Angle":           "Angle",
		"Distance":        "Distance",
		"Amount":          "Amount",
		"Intensity":       "Intensity",
		"ColorSaturation": "Color saturation",
		"Coverage":        "Coverage",
		"CellSize":        "Cell size",
		"Density":         "Density",
	},
	History: map[string]string{
		"Deselect":            "Deselect All",
		"SelectAll":           "Select All",
		"InvertSelection":     "Invert Selection",
		"EraseSelection":      "Erase Selection",
		"FillSelection":       "Fill Selection",
		"Cut":                 "Cut",
		"Paste":               "Paste",
		"CropToSelection":     "Crop to Selection",
		"ResizeImage":         "Resize Image",
		"CanvasSize":          "Canvas Size",
		"FlipHorizontal":      "Flip Horizontal",
		"FlipVertical":        "Flip Vertical",
		"Rotate90":            "Rotate 90°",
		"Rotate180":           "Rotate 180°",
		"Rotate270":           "Rotate -90°",
		"Flatten":             "Flatten",
		"AddLayer":            "Add Layer",
		"DeleteLayer":         "Delete Layer",
		"DuplicateLayer":      "Duplicate Layer",
		"MergeLayerDown":      "Merge Layer Down",
		"MoveLayerUp":         "Move Layer Up",
		"MoveLayerDown":       "Move Layer Down",
		"FlipLayerHorizontal": "Flip Layer Horizontal",
		"FlipLayerVertical":   "Flip Layer Vertical",
		"LayerVisibility":     "Layer Visibility",
		"LayerProperties":     "Layer Properties",
		"ImportFromFile":      "Import from File",
	},

	Untitled:          func(n int) string { return fmt.Sprintf("Untitled %d", n) },
	Background:        "Background",
	LayerN:            func(n int) string { return fmt.Sprintf("Layer %d", n) },
	CopySuffix:        "copy",
	AllImages:         "All images",
	AllFiles:          "All files",
	OpenFailed:        func(name, err string) string { return fmt.Sprintf("Cannot open %s: %s", name, err) },
	SaveFailed:        func(name, err string) string { return fmt.Sprintf("Cannot save %s: %s", name, err) },
	Saved:             func(name string) string { return "Saved " + name },
	SavedFlattened:    "Saved, the layers merged into one: this file type has no layers",
	UnsavedTitle:      "Unsaved changes",
	SaveChangesAsk:    func(name string) string { return fmt.Sprintf("Save the changes to %s?", name) },
	Save:              "Save",
	DontSave:          "Don't Save",
	ClipboardEmpty:    "There is no image on the clipboard",
	ClipboardInternal: "Copied within AltPaint (the system clipboard is not available)",
	ExpandCanvasAsk:   "The image being pasted is larger than the canvas. Expand the canvas?",
	CloneHint:         "Ctrl + click first to set where to copy from",
	PixelGridHint:     "The pixel grid is shown from 600% zoom",

	Width:                 "Width:",
	Height:                "Height:",
	Pixels:                "pixels",
	BackgroundLabel:       "Background:",
	BackgroundWhite:       "White",
	BackgroundSecondary:   "Secondary color",
	BackgroundTransparent: "Transparent",
	ByPercentage:          "By percentage:",
	KeepAspect:            "Maintain aspect ratio",
	Resampling:            "Resampling:",
	Resamplings:           [3]string{"Best quality", "Bilinear", "Nearest neighbor"},
	Anchor:                "Anchor:",
	Name:                  "Name:",
	Visible:               "Visible",
	BlendMode:             "Blend mode:",
	Opacity:               "Opacity:",

	Settings:       "Settings",
	Help:           "Help",
	About:          "About",
	Language:       "Language:",
	LanguageSystem: "System",
	Theme:          "Theme:",
	ThemeDark:      "Dark",
	ThemeLight:     "Light",
	JPEGQuality:    "JPEG quality:",
	AboutTitle:     func(app string) string { return "About " + app },
	Version:        "Version",
	Author:         "Author",
	License:        "License",
	VisitWebsite:   "Visit Website",
	Close:          "Close",
	Install:        "Install",
	Update:         "Update",
	Uninstall:      "Uninstall",
	InstallAsk: func(dir string) string {
		return "Install AltPaint into " + dir + "?\n\nIt is added to the Start menu, the desktop and the list of installed apps, and opens in place of this copy."
	},
	InstallFailed: func(err string) string { return "Installation failed: " + err },
	Installed:     "AltPaint is installed",
	UninstallAsk: func(dir string) string {
		return "Uninstall AltPaint?\n\nThe settings stay in " + dir + "."
	},
	Uninstalled:      "AltPaint is uninstalled",
	UninstallRunning: "AltPaint is running. Close it and try again.",
}

var ru = Strings{
	Error: "Ошибка",

	MenuFile:        "&Файл",
	MenuEdit:        "&Правка",
	MenuView:        "&Вид",
	MenuImage:       "&Изображение",
	MenuLayers:      "С&лои",
	MenuAdjustments: "&Коррекция",
	MenuEffects:     "&Эффекты",
	MenuHelp:        "&Справка",

	CmdNew:                 "Создать",
	CmdOpen:                "Открыть",
	CmdOpenRecent:          "Недавние файлы",
	CmdSave:                "Сохранить",
	CmdSaveAs:              "Сохранить как",
	CmdSaveAll:             "Сохранить все",
	CmdClose:               "Закрыть",
	CmdCloseAll:            "Закрыть все",
	CmdExit:                "Выход",
	CmdUndo:                "Отменить",
	CmdRedo:                "Повторить",
	CmdCut:                 "Вырезать",
	CmdCopy:                "Копировать",
	CmdCopyMerged:          "Копировать со всех слоёв",
	CmdPaste:               "Вставить",
	CmdPasteNewLayer:       "Вставить в новый слой",
	CmdPasteNewImage:       "Вставить в новое изображение",
	CmdSelectAll:           "Выделить всё",
	CmdDeselect:            "Снять выделение",
	CmdInvertSelection:     "Инвертировать выделение",
	CmdEraseSelection:      "Стереть выделенное",
	CmdFillSelection:       "Залить выделенное",
	CmdZoomIn:              "Увеличить",
	CmdZoomOut:             "Уменьшить",
	CmdZoomWindow:          "По размеру окна",
	CmdActualSize:          "Реальный размер",
	CmdPixelGrid:           "Пиксельная сетка",
	CmdNextImage:           "Следующее изображение",
	CmdPrevImage:           "Предыдущее изображение",
	CmdCropToSelection:     "Обрезать по выделению",
	CmdResize:              "Размер изображения",
	CmdCanvasSize:          "Размер холста",
	CmdFlipHorizontal:      "Отразить по горизонтали",
	CmdFlipVertical:        "Отразить по вертикали",
	CmdRotate90:            "Повернуть на 90° по часовой",
	CmdRotate270:           "Повернуть на 90° против часовой",
	CmdRotate180:           "Повернуть на 180°",
	CmdFlatten:             "Свести слои",
	CmdAddLayer:            "Добавить слой",
	CmdDeleteLayer:         "Удалить слой",
	CmdDuplicateLayer:      "Дублировать слой",
	CmdMergeDown:           "Объединить с нижним",
	CmdImportFromFile:      "Импорт из файла",
	CmdFlipLayerHorizontal: "Отразить слой по горизонтали",
	CmdFlipLayerVertical:   "Отразить слой по вертикали",
	CmdLayerUp:             "Слой выше",
	CmdLayerDown:           "Слой ниже",
	CmdLayerProperties:     "Свойства слоя",
	NoRecentFiles:          "(нет)",
	ClearRecent:            "Очистить список",

	PanelColors:  "Цвета",
	PanelLayers:  "Слои",
	PanelHistory: "История",
	Primary:      "Основной",
	Secondary:    "Дополнительный",
	SwapColors:   "Поменять цвета местами",
	ResetColors:  "Чёрный и белый",
	PaletteHint:  "Щелчок: основной цвет, правый щелчок: дополнительный",
	HistoryNew:   "Новое изображение",
	HistoryOpen:  "Открытие изображения",

	BrushWidth:      "Ширина кисти:",
	Antialiasing:    "Сглаживание",
	AlphaBlending:   "Альфа-смешение",
	Shape:           "Фигура:",
	Shapes:          [3]string{"Прямоугольник", "Скруглённый прямоугольник", "Эллипс"},
	FillMode:        "Заливка:",
	FillModes:       [3]string{"Контур", "Заливка", "Контур и заливка"},
	GradientType:    "Тип:",
	Gradients:       [5]string{"Линейный", "Линейный (отражённый)", "Ромбовидный", "Радиальный", "Конический"},
	Tolerance:       "Допуск, %:",
	FloodMode:       "Режим:",
	FloodModes:      [2]string{"Смежные", "Глобально"},
	Sampling:        "Образец:",
	SamplingModes:   [2]string{"Слой", "Изображение"},
	SelectionMode:   "Режим выделения:",
	SelectionModes:  [5]string{"Заменить", "Добавить (Ctrl)", "Вычесть (Alt)", "Пересечь (правый щелчок)", "Исключающее ИЛИ"},
	FontSize:        "Размер:",
	Bold:            "Жирный",
	Italic:          "Курсив",
	Monospace:       "Моноширинный",
	ZoomWindow:      "Окно",
	StatusSelection: func(w, h int) string { return fmt.Sprintf("Выделение %d × %d", w, h) },
	StatusImage:     func(w, h int) string { return fmt.Sprintf("Изображение %d × %d", w, h) },

	Tools: map[ToolID]string{
		ToolRectSelect:    "Прямоугольное выделение",
		ToolEllipseSelect: "Эллиптическое выделение",
		ToolLassoSelect:   "Лассо",
		ToolMagicWand:     "Волшебная палочка",
		ToolMoveSelected:  "Перемещение выделенных пикселей",
		ToolMoveSelection: "Перемещение выделения",
		ToolZoom:          "Масштаб",
		ToolPan:           "Прокрутка",
		ToolPaintBucket:   "Заливка",
		ToolGradient:      "Градиент",
		ToolPaintbrush:    "Кисть",
		ToolEraser:        "Ластик",
		ToolPencil:        "Карандаш",
		ToolColorPicker:   "Пипетка",
		ToolCloneStamp:    "Клонирующий штамп",
		ToolText:          "Текст",
		ToolLine:          "Линия",
		ToolShapes:        "Фигуры",
	},
	ToolHints: map[ToolID]string{
		ToolRectSelect:    "Тяните, чтобы выделить. Shift: квадрат. Ctrl добавляет, Alt вычитает. Щелчок снимает выделение",
		ToolEllipseSelect: "Тяните, чтобы выделить. Shift: круг. Ctrl добавляет, Alt вычитает. Щелчок снимает выделение",
		ToolLassoSelect:   "Обведите то, что нужно выделить. Ctrl добавляет, Alt вычитает",
		ToolMagicWand:     "Щелчок выделяет область похожего цвета. Ctrl добавляет, Alt вычитает, правый щелчок пересекает",
		ToolMoveSelected:  "Тяните, чтобы переместить выделенные пиксели",
		ToolMoveSelection: "Тяните, чтобы переместить контур выделения",
		ToolZoom:          "Щелчок увеличивает, правый щелчок уменьшает. Ctrl + колесо работает с любым инструментом",
		ToolPan:           "Тяните, чтобы прокрутить. С любым инструментом: держите пробел и тяните",
		ToolPaintBucket:   "Щелчок заливает область похожего цвета; правый щелчок — дополнительным цветом",
		ToolGradient:      "Тяните, чтобы нарисовать градиент от основного цвета к дополнительному. Shift: шаг 15°",
		ToolPaintbrush:    "Тяните, чтобы рисовать. [ и ] меняют ширину",
		ToolEraser:        "Тяните, чтобы стирать до прозрачности",
		ToolPencil:        "Тяните, чтобы рисовать линии в один пиксель",
		ToolColorPicker:   "Щелчок берёт основной цвет, правый щелчок — дополнительный",
		ToolCloneStamp:    "Ctrl + щелчок задаёт источник, затем рисуйте, копируя с него",
		ToolText:          "Щёлкните и печатайте. Enter: новая строка, Esc: готово",
		ToolLine:          "Тяните, чтобы нарисовать линию. Shift: шаг 15°",
		ToolShapes:        "Тяните, чтобы нарисовать фигуру. Shift: квадрат или круг",
	},
	Blends: [14]string{"Обычный", "Умножение", "Сложение", "Затемнение основы", "Осветление основы", "Отражение", "Свечение",
		"Перекрытие", "Разница", "Отрицание", "Замена светлым", "Замена тёмным", "Экран", "Исключающее ИЛИ"},
	Filters: map[string]string{
		"AutoLevel":          "Автоуровень",
		"BlackAndWhite":      "Чёрно-белое",
		"BrightnessContrast": "Яркость / контраст",
		"HueSaturation":      "Тон / насыщенность",
		"InvertColors":       "Инвертировать цвета",
		"Posterize":          "Постеризация",
		"Sepia":              "Сепия",
		"GaussianBlur":       "Размытие по Гауссу",
		"MotionBlur":         "Размытие в движении",
		"Sharpen":            "Резкость",
		"AddNoise":           "Добавить шум",
		"Pixelate":           "Пикселизация",
		"Emboss":             "Тиснение",
		"EdgeDetect":         "Выделение краёв",
		"Vignette":           "Виньетка",
	},
	Params: map[string]string{
		"Brightness":      "Яркость",
		"Contrast":        "Контраст",
		"Hue":             "Тон",
		"Saturation":      "Насыщенность",
		"Lightness":       "Светлота",
		"Levels":          "Уровни",
		"Radius":          "Радиус",
		"Angle":           "Угол",
		"Distance":        "Расстояние",
		"Amount":          "Сила",
		"Intensity":       "Интенсивность",
		"ColorSaturation": "Насыщенность цвета",
		"Coverage":        "Покрытие",
		"CellSize":        "Размер ячейки",
		"Density":         "Плотность",
	},
	History: map[string]string{
		"Deselect":            "Снятие выделения",
		"SelectAll":           "Выделить всё",
		"InvertSelection":     "Инверсия выделения",
		"EraseSelection":      "Стирание выделенного",
		"FillSelection":       "Заливка выделенного",
		"Cut":                 "Вырезание",
		"Paste":               "Вставка",
		"CropToSelection":     "Обрезка по выделению",
		"ResizeImage":         "Размер изображения",
		"CanvasSize":          "Размер холста",
		"FlipHorizontal":      "Отражение по горизонтали",
		"FlipVertical":        "Отражение по вертикали",
		"Rotate90":            "Поворот на 90°",
		"Rotate180":           "Поворот на 180°",
		"Rotate270":           "Поворот на -90°",
		"Flatten":             "Сведение слоёв",
		"AddLayer":            "Новый слой",
		"DeleteLayer":         "Удаление слоя",
		"DuplicateLayer":      "Дублирование слоя",
		"MergeLayerDown":      "Объединение с нижним",
		"MoveLayerUp":         "Слой выше",
		"MoveLayerDown":       "Слой ниже",
		"FlipLayerHorizontal": "Отражение слоя по горизонтали",
		"FlipLayerVertical":   "Отражение слоя по вертикали",
		"LayerVisibility":     "Видимость слоя",
		"LayerProperties":     "Свойства слоя",
		"ImportFromFile":      "Импорт из файла",
	},

	Untitled:   func(n int) string { return fmt.Sprintf("Без имени %d", n) },
	Background: "Фон",
	LayerN:     func(n int) string { return fmt.Sprintf("Слой %d", n) },
	CopySuffix: "копия",
	AllImages:  "Все изображения",
	AllFiles:   "Все файлы",
	OpenFailed: func(name, err string) string {
		return fmt.Sprintf("Не удалось открыть %s: %s", name, err)
	},
	SaveFailed: func(name, err string) string {
		return fmt.Sprintf("Не удалось сохранить %s: %s", name, err)
	},
	Saved:             func(name string) string { return "Сохранено: " + name },
	SavedFlattened:    "Сохранено со сведёнными слоями: в этом формате слоёв нет",
	UnsavedTitle:      "Несохранённые изменения",
	SaveChangesAsk:    func(name string) string { return fmt.Sprintf("Сохранить изменения в %s?", name) },
	Save:              "Сохранить",
	DontSave:          "Не сохранять",
	ClipboardEmpty:    "В буфере обмена нет изображения",
	ClipboardInternal: "Скопировано внутри AltPaint (системный буфер обмена недоступен)",
	ExpandCanvasAsk:   "Вставляемое изображение больше холста. Увеличить холст?",
	CloneHint:         "Сначала Ctrl + щелчок, чтобы задать, откуда копировать",
	PixelGridHint:     "Пиксельная сетка видна при масштабе от 600%",

	Width:                 "Ширина:",
	Height:                "Высота:",
	Pixels:                "пикс.",
	BackgroundLabel:       "Фон:",
	BackgroundWhite:       "Белый",
	BackgroundSecondary:   "Дополнительный цвет",
	BackgroundTransparent: "Прозрачный",
	ByPercentage:          "В процентах:",
	KeepAspect:            "Сохранять пропорции",
	Resampling:            "Интерполяция:",
	Resamplings:           [3]string{"Наилучшее качество", "Билинейная", "Ближайший сосед"},
	Anchor:                "Привязка:",
	Name:                  "Имя:",
	Visible:               "Видимый",
	BlendMode:             "Режим наложения:",
	Opacity:               "Непрозрачность:",

	Settings:       "Настройки",
	Help:           "Справка",
	About:          "О программе",
	Language:       "Язык:",
	LanguageSystem: "Системный",
	Theme:          "Тема:",
	ThemeDark:      "Тёмная",
	ThemeLight:     "Светлая",
	JPEGQuality:    "Качество JPEG:",
	AboutTitle:     func(app string) string { return "О программе " + app },
	Version:        "Версия",
	Author:         "Автор",
	License:        "Лицензия",
	VisitWebsite:   "Открыть сайт",
	Close:          "Закрыть",
	Install:        "Установить",
	Update:         "Обновить",
	Uninstall:      "Удалить",
	InstallAsk: func(dir string) string {
		return "Установить AltPaint в " + dir + "?\n\nПрограмма появится в меню «Пуск», на рабочем столе и в списке установленных приложений и откроется вместо этой копии."
	},
	InstallFailed: func(err string) string { return "Не удалось установить: " + err },
	Installed:     "AltPaint установлен",
	UninstallAsk: func(dir string) string {
		return "Удалить AltPaint?\n\nНастройки останутся в " + dir + "."
	},
	Uninstalled:      "AltPaint удалён",
	UninstallRunning: "AltPaint запущен. Закройте его и повторите.",
}

var catalog = i18n.NewCatalog(en, map[string]Strings{
	"ru": ru,
})

// T returns the texts in the language of the application
func T() *Strings {
	return catalog.Get(ui.Language())
}

// SetLanguage switches the application to the language of the settings, "" - the system's
func SetLanguage(lang string) {
	if lang == "" {
		lang = ui.SystemLanguage()
	}
	ui.SetLanguage(lang)
}
