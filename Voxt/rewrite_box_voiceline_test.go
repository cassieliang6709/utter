package main

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// TestVoiceLineTail: a long spoken instruction keeps its newest words; a
// short one is shown whole.
func TestVoiceLineTail(t *testing.T) {
	if got := voiceLineTail("改写成正式一点"); got != "改写成正式一点" {
		t.Errorf("short line changed: %q", got)
	}
	long := strings.Repeat("前面的话", 100) + "最后说的这句"
	got := voiceLineTail(long)
	if !strings.HasPrefix(got, "…") || !strings.HasSuffix(got, "最后说的这句") {
		t.Errorf("long line %q: want an ellipsis and the newest words", got)
	}
	if n := utf8.RuneCountInString(got); n != voiceLineMaxRunes+1 {
		t.Errorf("long line has %d runes, want %d", n, voiceLineMaxRunes+1)
	}
}
