# AltPaint

An image editor: layers with blend modes, undo with a history list,
selections, the familiar tools, adjustments and effects. It runs on Linux, Windows and macOS as a single
binary, without cgo.

Website and documentation: https://altbins.pro/altpaint/

> This is the first prototype. The tools, the file formats and the menus
> are in place; see [what is not there yet](#not-yet).

## Features

- **Layers.** Add, delete, duplicate, merge down, reorder, hide, import from
  a file. Each layer has an opacity and one of 14 blend modes (Multiply,
  Screen, Overlay, Color Dodge, Difference, Xor...).
- **History.** Every change is a step in the History panel; a click on a
  step goes back or forward to it. Ctrl+Z / Ctrl+Y as usual. The last 100
  steps of each image are kept.
- **Selections.** Rectangle, ellipse, lasso and magic wand, combined by
  replace, union (Ctrl), exclude (Alt), intersect or xor. The tools and
  the effects work inside the selection. Move the selected pixels or just
  the outline; crop, invert, erase or fill the selection.
- **Tools.** Paintbrush, eraser, pencil, paint bucket (contiguous or global,
  with tolerance), gradient (linear, reflected, diamond, radial, conical),
  color picker, clone stamp, text, line and shapes (rectangle, rounded
  rectangle, ellipse; outline, filled or both). Antialiasing and alpha
  blending can be turned off. Single-letter keys choose the tools (B, E,
  P, F, G, K, L, T, O, S, M, Z, H); pressed again, S, M and O go to the
  next tool of the group.
- **Colors.** Primary and secondary color with transparency, a color
  picker with a hue/saturation square, the 48-color palette (click for the
  primary, right click for the secondary), X swaps them, D resets.
- **Image.** Resize (with the resampling), canvas size with an anchor,
  crop to selection, flip, rotate, flatten.
- **Adjustments.** Auto-Level, Black and White, Brightness / Contrast, Hue /
  Saturation, Invert Colors, Posterize, Sepia.
- **Effects.** Gaussian Blur, Motion Blur, Sharpen, Add Noise, Pixelate,
  Emboss, Edge Detect, Vignette. The result is shown on the image while
  the parameters are changed.
- **Files.** Opens PNG, JPEG, BMP, GIF, TIFF and WebP; saves PNG, JPEG,
  BMP, GIF, TIFF and its own layered `.apn` (a zip of PNG layers and a
  JSON description). Several images are open at once, in tabs. Images can
  be dropped on the window.
- **View.** Zoom from 1% to 6400% (Ctrl + wheel, Ctrl+Plus/Minus, the zoom
  tool), zoom to window, the pixel grid, panning with Space + drag.
- **Interface.** Light and dark themes, English and Russian, the window
  layout and the tool options are remembered.
- **One copy at a time.** An image opened from the file manager goes to a
  new tab of the window that is already open.

## Installation

Download a build for your system from the
[releases](https://github.com/ipoluianov/altpaint/releases):

- **Windows:** `altpaint.exe`.
- **macOS** (Apple Silicon): the `.dmg` image.
- **Linux** (x86-64 and ARM64): a `.deb` or `.rpm` package, or a user-level
  install (no root) that picks the build for your machine:

  ```sh
  curl -fsSL https://github.com/ipoluianov/altpaint/releases/latest/download/linux-install.sh | bash
  ```

Settings are stored in `~/.altbins/.altpaint/`. Removing the application
leaves them in place.

### Windows

The exe is not signed, so on the first start SmartScreen may say "Windows
protected your PC". Click **More info**, then **Run anyway**.

A downloaded copy offers to install itself into `%USERPROFILE%\.altbins`
with the **Install** link in the status bar. It adds AltPaint to the Start
menu, the desktop and "Installed apps". When a newer version is started
from a download, the link reads **Update**. Remove AltPaint from "Installed
apps" or with the **Uninstall** link.

### macOS

Open the `.dmg` and drag AltPaint to Applications.

### Linux

- **Clipboard.** Images are copied to the system clipboard with `xclip`
  (X11) or `wl-clipboard` (Wayland). Without them, copy and paste work
  within AltPaint only.
- **File dialogs** need `zenity` or `kdialog`.
- **Display.** AltPaint needs X11, and runs under Wayland through XWayland,
  which GNOME and KDE have. glibc-based distributions only (not Alpine).

## Not yet

- The right and middle mouse buttons act on a click only: the UI library
  reports the release of the left button only, so dragging with the right
  button (painting with the secondary color) is not there yet.
- No handles to edit a shape or a text after it is drawn, no rotate/zoom of
  a layer, no curves, levels or plugins.
- The text tool uses the Go fonts, not the fonts of the system.
- The interface is in English and Russian; the other languages of the
  AltBins utilities are to come.

## Building

You need Go (see the version in [go.mod](go.mod)). No cgo is required.

```sh
go build .
go test ./...
```

`build/all.sh` (or `build/all.bat`) builds every platform into `bin/`,
together with the deb/rpm packages, the dmg and the installer script.
The toolbar icons are drawn in `forms/icons/*.svg` and rendered by
`scripts/render-icons.sh`.

## License

[MIT](LICENSE) © Ivan Poluianov
