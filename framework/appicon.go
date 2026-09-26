package golib

import (
	"os"
	"path/filepath"

	"golib/internal/device"
)

// showGameIcon gives the system the game's icon.png while a debug build runs,
// which on macOS puts it in the Dock: a debug build there is a bare
// executable, which the Dock shows with the generic icon for programs. The
// app golib dist makes on macOS carries the icon from the same file, and on
// Windows golib puts it inside every executable, so only debug builds read
// the file, and only macOS's backend uses it. A game without icon.png keeps
// the system's default.
func showGameIcon() {
	if distBuild {
		return
	}
	workDir, err := os.Getwd()
	if err != nil {
		return
	}
	exe, err := os.Executable()
	if err != nil {
		exe = "" // only the working directory counts
	}
	icon, err := os.ReadFile(filepath.Join(debugGameDir(workDir, exe, isDir), "icon.png"))
	if err != nil {
		return
	}
	device.SetAppIcon(icon)
}
