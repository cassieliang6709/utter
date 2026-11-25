package main

import (
	"strings"

	"github.com/sgl-project/sglang-omni/Voxt/internal/models"
	"github.com/sgl-project/sglang-omni/Voxt/internal/settings"
)

// utterFeatureHidden reports whether feature or page id (without leading "/")
// is hidden in Utter's streamlined UI build, or hidden by user settings.
func utterFeatureHidden(f settings.Features, id string) bool {
	id = strings.TrimPrefix(id, "/")
	if f.IsHidden(id) {
		return true
	}
	if !utterLockedUI {
		return false
	}
	switch id {
	case settings.HideTranslate,
		settings.HideRewrite,
		settings.HideNotes,
		settings.HideMeetings,
		settings.HideAppEnhancement,
		"snippets":
		return true
	}
	return false
}

// utterSpeechModelAllowed reports whether a local speech model is shown in
// Utter's locked-model UI (only Qwen3-ASR 0.6B 4bit; Silero VAD stays internal).
func utterSpeechModelAllowed(id string) bool {
	if !utterLockedUI {
		return true
	}
	return id == models.ASRFast || id == models.VADSilero
}
