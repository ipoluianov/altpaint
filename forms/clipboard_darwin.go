package forms

import (
	"bytes"
	"context"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// On macOS the clipboard is reached through AppleScript, as the clipboard
// library needs cgo there

const clipboardTimeout = 5 * time.Second

func systemClipboardWrite(data []byte) error {
	f, err := os.CreateTemp("", "altpaint-*.png")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	f.Close()
	path, _ := filepath.Abs(f.Name())
	ctx, cancel := context.WithTimeout(context.Background(), clipboardTimeout)
	defer cancel()
	script := `set the clipboard to (read (POSIX file "` + strings.ReplaceAll(path, `"`, `\"`) + `") as «class PNGf»)`
	return exec.CommandContext(ctx, "osascript", "-e", script).Run()
}

func systemClipboardRead() ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), clipboardTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, "osascript", "-e", "the clipboard as «class PNGf»").Output()
	if err != nil {
		return nil, err
	}
	// The result is «data PNGf89504E47...»
	s := strings.TrimSpace(string(out))
	s = strings.TrimPrefix(s, "«data PNGf")
	s = strings.TrimSuffix(s, "»")
	data, err := hex.DecodeString(s)
	if err != nil || !bytes.HasPrefix(data, []byte("\x89PNG")) {
		return nil, errNoSystemClipboard
	}
	return data, nil
}
