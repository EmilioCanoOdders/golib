package golib

import (
	"os"
	"slices"
	"strings"

	"golib/internal/device"
)

// SystemLanguages returns the languages the player's system is set to show
// its text in, most preferred first, as language tags: the language in
// lowercase and then, when the system says it, the script and the region,
// such as "es-ES", "en", "pt-BR" or "zh-Hans-CN". They are Windows's display
// languages, macOS's preferred languages, Linux's LANGUAGE and LC_ALL,
// LC_MESSAGES or LANG, and a browser's navigator.languages. Use it to start a
// game in the player's language until they choose one, with ChooseLanguage.
//
// It is empty when the system says none, and under golib shot, so that its
// pictures are the same on every machine: a game then starts in its own
// default. It works before Run.
func SystemLanguages() []string {
	if os.Getenv(shotFramesEnv) != "" {
		return nil
	}
	return languageTags(device.SystemLanguages())
}

// ChooseLanguage returns which of the languages a game has, by their tags,
// such as "es" and "en", to start in: the first of the system's languages
// it has, whole or by its language alone, so "es-MX" chooses "es", and "en"
// chooses "en-US" when the game has only that; or fallback when it has none
// of them.
//
//	lang := golib.ChooseLanguage([]string{"es", "en"}, "en")
func ChooseLanguage(have []string, fallback string) string {
	return chooseLanguage(SystemLanguages(), have, fallback)
}

// chooseLanguage is ChooseLanguage for the system's languages wanted.
func chooseLanguage(wanted, have []string, fallback string) string {
	tags := languageTags(have)
	for _, w := range wanted {
		if i := slices.Index(tags, w); i >= 0 {
			return have[i] // the whole tag
		}
	}
	for _, w := range wanted {
		base := languageBase(w)
		for i, t := range tags {
			if languageBase(t) == base {
				return have[i]
			}
		}
	}
	return fallback
}

// languageTags returns names of languages as tags, each once: "es_ES.UTF-8"
// and "es-es" are "es-ES". A name with no language is left out.
func languageTags(names []string) []string {
	var tags []string
	for _, name := range names {
		if tag := languageTag(name); tag != "" && !slices.Contains(tags, tag) {
			tags = append(tags, tag)
		}
	}
	return tags
}

// languageTag returns name as a language tag: an encoding after "." and a
// modifier after "@" dropped, "_" as "-", the language lowercase, a script
// of four letters capitalized, and a region of two letters in capitals.
func languageTag(name string) string {
	name, _, _ = strings.Cut(name, ".")
	name, _, _ = strings.Cut(name, "@")
	parts := strings.FieldsFunc(name, func(r rune) bool { return r == '-' || r == '_' })
	if len(parts) == 0 || len(parts[0]) < 2 || len(parts[0]) > 3 {
		return ""
	}
	parts[0] = strings.ToLower(parts[0])
	for i := 1; i < len(parts); i++ {
		switch len(parts[i]) {
		case 4:
			parts[i] = strings.ToUpper(parts[i][:1]) + strings.ToLower(parts[i][1:])
		case 2:
			parts[i] = strings.ToUpper(parts[i])
		}
	}
	return strings.Join(parts, "-")
}

// languageBase returns the language of a tag, without its script or region.
func languageBase(tag string) string {
	base, _, _ := strings.Cut(tag, "-")
	return base
}
