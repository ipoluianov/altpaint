# Contributing

Bug reports and ideas are welcome in
[issues](https://github.com/ipoluianov/altpaint/issues). Please include the
AltPaint version (shown in About) and your operating system.

## Building

You need Go, see the version in [go.mod](go.mod). No cgo is required, so
any platform builds for any other.

```sh
go build .
go test ./...
```

The UI library is [nui](https://github.com/ipoluianov/nui). To work on both
at once, uncomment the `replace` line at the end of `go.mod`.

## Code layout

- `forms/` - the windows, dialogs and widgets; `forms/strings.go` holds
  all the texts
- `paint/` - the image model: layers, blend modes, history, selection,
  the drawing of the tools, adjustments, effects and file formats; it has
  no UI and is covered by tests
- `config/` - the settings
- `install/` - installing a downloaded copy on Windows
- `instance/` - keeps a single running copy
- `build/` - release builds and packages, `scripts/` - the Linux installer

## Pull requests

- Keep a change to one thing, and describe what it does and why.
- Every text goes into all the languages in `forms/strings.go`.
  `TestAllTranslated` fails on a missing one. A machine translation is
  fine, a native speaker can fix it later.
- Say on which systems you tried it.
