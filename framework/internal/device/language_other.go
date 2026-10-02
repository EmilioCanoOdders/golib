//go:build !js && !darwin && !windows

package device

import (
	"os"
	"strings"
)

// SystemLanguages returns the languages Linux shows its own text in, most
// preferred first, as the environment names them, such as "es_ES.UTF-8":
// LANGUAGE's list, separated by colons, which GNU programs read first, and
// then the locale of messages, LC_ALL, LC_MESSAGES or LANG, whichever is set
// first. "C" and "POSIX", which mean no language, are left out.
func SystemLanguages() []string {
	var languages []string
	for _, name := range strings.Split(os.Getenv("LANGUAGE"), ":") {
		languages = append(languages, name)
	}
	for _, variable := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if value := os.Getenv(variable); value != "" {
			languages = append(languages, value)
			break
		}
	}
	out := languages[:0]
	for _, name := range languages {
		if base, _, _ := strings.Cut(name, "."); name != "" && base != "C" && base != "POSIX" {
			out = append(out, name)
		}
	}
	return out
}
