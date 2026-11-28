//go:build utter

package main

import (
	"slices"
	"testing"

	"github.com/sgl-project/sglang-omni/Voxt/internal/i18n"
	"github.com/sgl-project/sglang-omni/Voxt/internal/models"
	"github.com/sgl-project/sglang-omni/Voxt/internal/settings"
)

func TestUtterProfileSidebarStreamlined(t *testing.T) {
	tr := i18n.NewTranslator("en")
	nav := buildNavViewState(tr, navViewStateInput{
		Current:  "/home",
		Features: settings.Defaults().Features,
	})
	var paths []string
	for _, g := range nav.Groups {
		for _, p := range g.Pages {
			paths = append(paths, p.Path)
		}
	}
	want := []string{"/home", "/history", "/dictionary", "/logs", "/settings"} // Transcribe is under Settings since Voxt 2.1.4
	if !slices.Equal(paths, want) {
		t.Fatalf("sidebar paths = %v, want %v", paths, want)
	}
}

func TestUtterProfileLockedModels(t *testing.T) {
	tr := i18n.NewTranslator("en")
	catalog := models.Catalog()
	status := map[string]models.Status{}
	for _, m := range catalog {
		status[m.ID] = models.Status{Installed: m.ID == models.ASRFast}
	}
	state := buildModelsViewState(tr, modelsViewStateInput{
		Catalog:  catalog,
		Status:   status,
		Settings: settings.Defaults(),
	})
	var ids []string
	for _, card := range state.ASR {
		for _, row := range card.Rows {
			ids = append(ids, row.ID)
		}
	}
	want := []string{models.ASRFast}
	if !slices.Equal(ids, want) {
		t.Fatalf("visible ASR models = %v, want %v", ids, want)
	}
	if state.Dictation != nil {
		t.Fatalf("expected macOS Dictation card hidden in Utter build")
	}
	if len(state.Hidden) != 0 {
		t.Fatalf("expected no hidden-model drawer in Utter build, got %d", len(state.Hidden))
	}
}

func TestUtterProfileSettingsAndShortcutsStreamlined(t *testing.T) {
	tr := i18n.NewTranslator("en")
	// Language models stays: the local polish service's address is set there.
	sections := buildSettingsSections(tr)
	if !slices.ContainsFunc(sections, func(s settingsSectionCatalogEntry) bool { return s.id == "language-models" }) {
		t.Fatalf("the Language models section is gone from the Utter build")
	}
	sc := buildSettingsShortcutsViewState(tr, settingsShortcutsStateInput{
		Features: settings.Defaults().Features,
	})
	for _, row := range sc.Rows {
		switch row.Feature {
		case "translate", "rewrite", "note":
			t.Fatalf("unexpected shortcut row %q in Utter build", row.Feature)
		}
	}
}

// TestUtterProfileOnboarding: the setup guide offers the locked speech
// model and passes over Translate and Rewrite, both ways.
func TestUtterProfileOnboarding(t *testing.T) {
	a := &app{}
	if got := a.obSpeech(); got != models.ASRFast {
		t.Errorf("the guide offers %s, want %s", got, models.ASRFast)
	}
	a.ob.step = obTranscribe
	if a.obGo(obTranslate); a.ob.step != obMore {
		t.Errorf("Continue from Transcribe went to step %d, want More", a.ob.step)
	}
	if a.obGo(obRewrite); a.ob.step != obTranscribe {
		t.Errorf("Back from More went to step %d, want Transcribe", a.ob.step)
	}
	titles := buildOnboardingViewState(i18n.NewTranslator("en"), onboardingViewStateInput{Open: true, Step: obTranscribe}).StepTitles
	if titles[obTranslate] != "" || titles[obRewrite] != "" || titles[obTranscribe] == "" {
		t.Errorf("breadcrumbs %q", titles)
	}
}

// TestUtterProfileHotkeysNoNote: the utter build has no Notes page, so a
// tap of Right ⌘ (the note shortcut's default) must not reach the helper;
// it started note recordings out of nowhere. Transcribe, Translate and
// Rewrite keep their shortcuts.
func TestUtterProfileHotkeysNoNote(t *testing.T) {
	a := &app{prefs: settings.Defaults()}
	var ids []string
	for _, b := range a.hotkeyPlan().Bindings {
		ids = append(ids, b.ID)
	}
	if slices.Contains(ids, "note") {
		t.Errorf("note binding in the utter build: %v", ids)
	}
	for _, want := range []string{"transcribe/0", "translate/0", "rewrite/0"} {
		if !slices.Contains(ids, want) {
			t.Errorf("no %s binding: %v", want, ids)
		}
	}
}
