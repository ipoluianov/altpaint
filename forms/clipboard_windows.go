package forms

import "golang.design/x/clipboard"

// On Windows the clipboard library works without cgo; nui initializes it

func systemClipboardWrite(data []byte) error {
	clipboard.Write(clipboard.FmtImage, data)
	return nil
}

func systemClipboardRead() ([]byte, error) {
	data := clipboard.Read(clipboard.FmtImage)
	if len(data) == 0 {
		return nil, errNoSystemClipboard
	}
	return data, nil
}
