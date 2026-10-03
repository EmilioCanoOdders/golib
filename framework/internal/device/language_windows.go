//go:build !js

package device

import (
	"syscall"
	"unsafe"
)

// muiLanguageName asks GetUserPreferredUILanguages for the languages' names,
// such as "es-ES", rather than their numbers.
const muiLanguageName = 0x8

var getUserPreferredUILanguages = syscall.NewLazyDLL("kernel32.dll").NewProc("GetUserPreferredUILanguages")

// SystemLanguages returns the languages Windows shows its own text in, most
// preferred first, as it names them, such as "es-ES": the user's display
// languages. It is empty if Windows says none.
func SystemLanguages() []string {
	if getUserPreferredUILanguages.Find() != nil {
		return nil
	}
	var count, size uint32
	// The first call says how long the list is, the second fills it in: the
	// names one after the other, each ending in a zero, and a zero after the
	// last.
	if ok, _, _ := getUserPreferredUILanguages.Call(muiLanguageName, uintptr(unsafe.Pointer(&count)), 0, uintptr(unsafe.Pointer(&size))); ok == 0 || size == 0 {
		return nil
	}
	buffer := make([]uint16, size)
	if ok, _, _ := getUserPreferredUILanguages.Call(muiLanguageName, uintptr(unsafe.Pointer(&count)), uintptr(unsafe.Pointer(&buffer[0])), uintptr(unsafe.Pointer(&size))); ok == 0 {
		return nil
	}
	var languages []string
	start := 0
	for i, c := range buffer {
		if c != 0 {
			continue
		}
		if i > start {
			languages = append(languages, syscall.UTF16ToString(buffer[start:i]))
		}
		start = i + 1
	}
	return languages
}
