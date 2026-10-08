package paint

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/image/bmp"
	"golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

// NativeExt is the extension of the AltPaint format: the layers and their
// properties in a zip, each layer a PNG
const NativeExt = ".apn"

// Format is a file type the images are saved in
type Format struct {
	Name string
	Exts []string // the first is the one added to a new file
	// Layered formats keep the layers; the others save the flattened image
	Layered bool
}

// Formats are the file types offered when saving, the native one first
var Formats = []Format{
	{"AltPaint", []string{NativeExt}, true},
	{"PNG", []string{".png"}, false},
	{"JPEG", []string{".jpg", ".jpeg", ".jpe", ".jfif"}, false},
	{"BMP", []string{".bmp"}, false},
	{"GIF", []string{".gif"}, false},
	{"TIFF", []string{".tif", ".tiff"}, false},
}

// OpenExts are the extensions of the files that can be opened
var OpenExts = []string{NativeExt, ".png", ".jpg", ".jpeg", ".jpe", ".jfif", ".bmp", ".gif", ".tif", ".tiff", ".webp"}

// FormatOf returns the format for the extension of the path, nil when it is not known
func FormatOf(path string) *Format {
	ext := strings.ToLower(filepath.Ext(path))
	for i := range Formats {
		for _, e := range Formats[i].Exts {
			if e == ext {
				return &Formats[i]
			}
		}
	}
	return nil
}

// ErrUnknownFormat is returned when saving to an extension that is not known
var ErrUnknownFormat = errors.New("unknown file type")

// Open reads an image file; layerName names the layer of a flat image
func Open(path string, layerName string) (*Document, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var d *Document
	if bytes.HasPrefix(data, []byte("PK")) && strings.EqualFold(filepath.Ext(path), NativeExt) {
		d, err = decodeNative(data)
	} else {
		var img image.Image
		img, _, err = image.Decode(bytes.NewReader(data))
		if err == nil {
			d = NewDocumentFromImage(img, layerName)
		}
	}
	if err != nil {
		return nil, err
	}
	d.Path = path
	return d, nil
}

// DecodeImage reads an image of any of the known formats into premultiplied RGBA at (0, 0)
func DecodeImage(r io.Reader) (*image.RGBA, error) {
	img, _, err := image.Decode(r)
	if err != nil {
		return nil, err
	}
	return ToRGBA(img), nil
}

// ToRGBA converts an image to premultiplied RGBA at (0, 0)
func ToRGBA(img image.Image) *image.RGBA {
	b := img.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(dst, dst.Rect, img, b.Min, draw.Src)
	return dst
}

// SaveOptions are the settings of the formats that have them
type SaveOptions struct {
	JPEGQuality int // 1..100
}

// Save writes the document to the path in the format of its extension.
// The file is written next to it first and then put in its place.
func Save(d *Document, path string, opts SaveOptions) error {
	f := FormatOf(path)
	if f == nil {
		return fmt.Errorf("%w: %s", ErrUnknownFormat, filepath.Ext(path))
	}
	var buf bytes.Buffer
	if err := Encode(&buf, d, f, opts); err != nil {
		return err
	}
	return writeFileAtomic(path, buf.Bytes())
}

// Encode writes the document in the format
func Encode(w io.Writer, d *Document, f *Format, opts SaveOptions) error {
	if f.Layered {
		return encodeNative(w, d)
	}
	img := d.Flattened()
	switch f.Exts[0] {
	case ".png":
		return png.Encode(w, img)
	case ".jpg":
		q := opts.JPEGQuality
		if q <= 0 {
			q = 95
		}
		return jpeg.Encode(w, flattenOn(img, color.White), &jpeg.Options{Quality: min(100, q)})
	case ".bmp":
		return bmp.Encode(w, img)
	case ".gif":
		return gif.Encode(w, img, &gif.Options{NumColors: 256})
	case ".tif":
		return tiff.Encode(w, img, &tiff.Options{Compression: tiff.Deflate})
	}
	return ErrUnknownFormat
}

// flattenOn draws the image over the color, for the formats without transparency
func flattenOn(img *image.RGBA, bg color.Color) *image.RGBA {
	dst := image.NewRGBA(img.Rect)
	draw.Draw(dst, dst.Rect, image.NewUniform(bg), image.Point{}, draw.Src)
	draw.Draw(dst, dst.Rect, img, img.Rect.Min, draw.Over)
	return dst
}

// nativeDoc is document.json of the native format
type nativeDoc struct {
	Version int
	Width   int
	Height  int
	Current int
	Layers  []nativeLayer // the bottom one first
}

type nativeLayer struct {
	Name    string
	Visible bool
	Opacity uint8
	Blend   string
	File    string
}

var blendIDs = map[BlendMode]string{
	BlendNormal: "normal", BlendMultiply: "multiply", BlendAdditive: "additive", BlendColorBurn: "colorburn",
	BlendColorDodge: "colordodge", BlendReflect: "reflect", BlendGlow: "glow", BlendOverlay: "overlay",
	BlendDifference: "difference", BlendNegation: "negation", BlendLighten: "lighten", BlendDarken: "darken",
	BlendScreen: "screen", BlendXor: "xor",
}

func encodeNative(w io.Writer, d *Document) error {
	z := zip.NewWriter(w)
	doc := nativeDoc{Version: 1, Width: d.W, Height: d.H, Current: d.Current}
	for i, l := range d.Layers {
		name := fmt.Sprintf("layer%d.png", i)
		doc.Layers = append(doc.Layers, nativeLayer{Name: l.Name, Visible: l.Visible, Opacity: l.Opacity, Blend: blendIDs[l.Blend], File: name})
		// PNGs are compressed already
		fw, err := z.CreateHeader(&zip.FileHeader{Name: name, Method: zip.Store})
		if err != nil {
			return err
		}
		if err := png.Encode(fw, l.Img); err != nil {
			return err
		}
	}
	// A preview for file managers that look into the zip
	fw, err := z.CreateHeader(&zip.FileHeader{Name: "preview.png", Method: zip.Store})
	if err != nil {
		return err
	}
	if err := png.Encode(fw, d.Flattened()); err != nil {
		return err
	}
	js, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	fw, err = z.Create("document.json")
	if err != nil {
		return err
	}
	if _, err := fw.Write(js); err != nil {
		return err
	}
	return z.Close()
}

func decodeNative(data []byte) (*Document, error) {
	z, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}
	readFile := func(name string) ([]byte, error) {
		f, err := z.Open(name)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		return io.ReadAll(io.LimitReader(f, 1<<31))
	}
	js, err := readFile("document.json")
	if err != nil {
		return nil, err
	}
	var doc nativeDoc
	if err := json.Unmarshal(js, &doc); err != nil {
		return nil, err
	}
	if doc.Width <= 0 || doc.Height <= 0 || doc.Width > 1<<16 || doc.Height > 1<<16 || len(doc.Layers) == 0 {
		return nil, errors.New("invalid document")
	}
	d := &Document{W: doc.Width, H: doc.Height}
	for _, nl := range doc.Layers {
		bs, err := readFile(nl.File)
		if err != nil {
			return nil, err
		}
		img, err := png.Decode(bytes.NewReader(bs))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", nl.File, err)
		}
		l := NewLayer(nl.Name, d.W, d.H)
		draw.Draw(l.Img, l.Img.Rect, img, img.Bounds().Min, draw.Src)
		l.Visible = nl.Visible
		l.Opacity = nl.Opacity
		for mode, id := range blendIDs {
			if id == nl.Blend {
				l.Blend = mode
			}
		}
		d.Layers = append(d.Layers, l)
	}
	d.Current = max(0, min(doc.Current, len(d.Layers)-1))
	d.InvalidateAll()
	return d, nil
}

// writeFileAtomic writes to a temporary file and renames it over the target,
// so a crash leaves either the old file or the new one, never a torn one
func writeFileAtomic(path string, bs []byte) error {
	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	_, err = f.Write(bs)
	if err == nil {
		err = f.Sync()
	}
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err == nil {
		err = os.Rename(tmp, path)
	}
	if err != nil {
		os.Remove(tmp)
	}
	return err
}
