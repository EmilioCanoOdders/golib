package golib

import (
	"slices"
	"testing"
)

// The systems name languages in their own ways; golib says them all as tags.
func TestSystemLanguagesAreTags(t *testing.T) {
	got := languageTags([]string{"es_ES.UTF-8", "es-es", "en_US@euro", "zh-hans-cn", "pt-BR", "C", "", "es-419", "x"})
	want := []string{"es-ES", "en-US", "zh-Hans-CN", "pt-BR", "es-419"}
	if !slices.Equal(got, want) {
		t.Errorf("the tags are %v, want %v", got, want)
	}
}

// A game starts in the first of the system's languages it has, whole or by
// the language alone, and in its fallback with none of them.
func TestChooseLanguage(t *testing.T) {
	for _, c := range []struct {
		wanted, have []string
		want         string
	}{
		{[]string{"es-ES", "en-US"}, []string{"es", "en"}, "es"},
		{[]string{"fr-FR", "en-GB"}, []string{"es", "en"}, "en"},
		{[]string{"pt-BR"}, []string{"pt-PT", "pt-BR"}, "pt-BR"},
		{[]string{"en"}, []string{"es", "en-US"}, "en-US"},
		{[]string{"de-DE"}, []string{"es", "en"}, "en"},
		{nil, []string{"es", "en"}, "en"},
	} {
		if got := chooseLanguage(c.wanted, c.have, "en"); got != c.want {
			t.Errorf("wanting %v from %v chose %q, want %q", c.wanted, c.have, got, c.want)
		}
	}
}

// Under golib shot there are no system languages, so its pictures repeat
// on any machine.
func TestShotsHaveNoSystemLanguage(t *testing.T) {
	t.Setenv(shotFramesEnv, "60")
	if got := SystemLanguages(); got != nil {
		t.Errorf("under golib shot the system speaks %v, want nothing", got)
	}
	if got := ChooseLanguage([]string{"es", "en"}, "es"); got != "es" {
		t.Errorf("under golib shot a game starts in %q, want its fallback", got)
	}
}
