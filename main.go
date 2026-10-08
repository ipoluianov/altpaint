package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/ipoluianov/altpaint/app"
	"github.com/ipoluianov/altpaint/config"
	"github.com/ipoluianov/altpaint/forms"
	"github.com/ipoluianov/altpaint/install"
	"github.com/ipoluianov/altpaint/instance"
	"github.com/ipoluianov/nui/ui"
)

// fileArgs returns the files of the command line, made absolute: the
// running copy may have another working directory
func fileArgs() []string {
	var files []string
	for _, a := range os.Args[1:] {
		if strings.HasPrefix(a, "--") {
			continue
		}
		if abs, err := filepath.Abs(a); err == nil {
			a = abs
		}
		files = append(files, a)
	}
	return files
}

func main() {
	install.SetIcon(iconPNG)
	// Started from "Installed apps" to remove it
	if install.HasArg(install.UninstallArg) {
		uninstall(install.HasArg(install.QuietArg))
		return
	}

	// One copy runs: the images opened from the file manager go to its window
	files := fileArgs()
	inst, ok := instance.Acquire(config.ConfigDirectory(), files)
	if !ok {
		return // the running copy opens the files instead
	}
	defer inst.Close()

	config.LoadSettings()
	forms.SetLanguage(config.GetSettings().Language)
	forms.ApplyTheme(config.GetSettings().Theme)
	ui.SetAppIcon(appIcon())
	form := ui.NewForm()
	mainForm := forms.NewMainForm()
	form.Panel().SetPanelPadding(0)
	form.Panel().AddWidget(0, 0, mainForm)
	form.SetMenuBar(mainForm.BuildMenuBar())
	mainForm.AddShortcuts(form)
	maximized := mainForm.RestoreWindowState(form)
	form.OnClose = mainForm.OnWindowClose
	form.SetOnLanguageChanged(mainForm.ApplyLanguage)
	form.SetOnFilesDropped(func(files []string, x, y int) { mainForm.OpenFiles(files) })
	form.Show()
	inst.Serve(func(files []string) { form.Invoke(func() { mainForm.BringToFront(files) }) })
	// The form is handled by its own goroutine once shown
	form.Invoke(func() {
		if maximized {
			form.Maximize()
		}
		mainForm.OpenFiles(files)
		mainForm.EnsureDocument()
		mainForm.Activate()
		if install.HasArg(install.InstalledArg) {
			mainForm.ShowInstalled()
		}
	})
	form.Exec()

	// Installed from this copy: the installed one takes over, so the lock goes first
	if install.RelaunchPending() {
		inst.Close()
		if err := install.StartInstalled(); err != nil {
			install.Inform(app.DisplayName, err.Error())
		}
	}
}
