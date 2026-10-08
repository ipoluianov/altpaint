package forms

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"time"
)

// The system clipboard is reached through wl-clipboard on Wayland and xclip
// on X11, as nui talks to X11 without cgo and has no image clipboard

const clipboardTimeout = 3 * time.Second

func clipboardTool(write bool) (string, []string) {
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		if write {
			if p, err := exec.LookPath("wl-copy"); err == nil {
				return p, []string{"--type", "image/png"}
			}
		} else if p, err := exec.LookPath("wl-paste"); err == nil {
			return p, []string{"--no-newline", "--type", "image/png"}
		}
	}
	if p, err := exec.LookPath("xclip"); err == nil {
		if write {
			return p, []string{"-selection", "clipboard", "-t", "image/png", "-i"}
		}
		return p, []string{"-selection", "clipboard", "-t", "image/png", "-o"}
	}
	return "", nil
}

func systemClipboardWrite(data []byte) error {
	tool, args := clipboardTool(true)
	if tool == "" {
		return errNoSystemClipboard
	}
	ctx, cancel := context.WithTimeout(context.Background(), clipboardTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, tool, args...)
	cmd.Stdin = bytes.NewReader(data)
	return cmd.Run()
}

func systemClipboardRead() ([]byte, error) {
	tool, args := clipboardTool(false)
	if tool == "" {
		return nil, errNoSystemClipboard
	}
	ctx, cancel := context.WithTimeout(context.Background(), clipboardTimeout)
	defer cancel()
	return exec.CommandContext(ctx, tool, args...).Output()
}
