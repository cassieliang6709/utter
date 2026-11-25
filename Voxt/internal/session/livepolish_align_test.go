package session

import (
	"fmt"
	"strings"
	"testing"

	"github.com/sgl-project/sglang-omni/Voxt/internal/compose"
	"github.com/sgl-project/sglang-omni/Voxt/internal/store"
)

// TestLivePolishAlign: the final text counts as the live text when only
// punctuation, white space or case differ, with what the Final adds after
// it; any other difference keeps the final text.
func TestLivePolishAlign(t *testing.T) {
	for _, c := range []struct{ live, final, want string }{
		{"看得更细一些，就本质上。", "看得更细一些。就本质上。", "看得更细一些，就本质上。"},
		{"你看，两个里面。", "你看两个里面。", "你看，两个里面。"},
		{"Hello there. How are you?", "Hello there, how are you?", "Hello there. How are you?"},
		{"第一句。第二", "第一句，第二句。", "第一句。第二句。"},
		{"Hello there.", "Hello there. How are you?", "Hello there. How are you?"},
		{"第一句。", "第一句", "第一句。"},
		{"第一句", "第一句。", "第一句。"},
		{"How are", "How are you?", "How are you?"},
		{"How are you?", "How are you doing?", "How are you doing?"},
		{"Hi. How are you?", "Hi, how are you doing?", "Hi. how are you doing?"},
		{"Hi. How are you?", "Hi. How are you doing?", "Hi. How are you doing?"},
		{"一些，就本质上。为什么慢", "一些。就本质上，为什么慢？还有", "一些，就本质上。为什么慢？还有"},
		{"Meet Tuesday. No,", "Meet Tuesday. No, Thursday.", "Meet Tuesday. No, Thursday."},
		// words differ, or the Final is shorter: no match
		{"Hello their.", "Hello there.", "Hello there."},
		{"第一句。第二句。", "第一句。", "第一句。"},
		{"", "第一句。", "第一句。"},
	} {
		lp := &livePolish{text: c.live}
		if got := lp.align(c.final); got != c.want {
			t.Errorf("align(%q) with live %q = %q, want %q", c.final, c.live, got, c.want)
		}
	}
}

// TestLivePolishPunctuationOnly: a Final that changes a comma of the live
// text reuses the live clean-up, and Stop asks nothing more.
func TestLivePolishPunctuationOnly(t *testing.T) {
	model := newLiveLLM()
	r := newEarlyUnitRun(t, model, nil)
	r.settings.Features.Transcribe.RecleanSentences = 0
	live := "看得更细一些，就本质上。为什么这么慢？"
	r.hearLive(live)
	waitLive(t, r, live)
	got, err := liveCleanUp(r, "看得更细一些。就本质上。为什么这么慢？")
	if err != nil {
		t.Fatal(err)
	}
	if got != "["+live+"]" {
		t.Errorf("result %q, want the live clean-up's answer", got)
	}
	if n := len(model.requests()); n != 1 {
		t.Errorf("%d requests, want only the one while recording", n)
	}
}

// TestLivePolishLargeGlossary: a dictionary of more than
// compose.GlossaryAllUpTo terms still cleans up while recording, and each
// request lists the terms its own text mentions.
func TestLivePolishLargeGlossary(t *testing.T) {
	var terms []store.Term
	for i := range compose.GlossaryAllUpTo + 5 {
		terms = append(terms, store.Term{Term: fmt.Sprintf("Term%d", i), Enabled: true})
	}
	model := newLiveLLM()
	r := newEarlyUnitRun(t, model, terms)
	r.settings.Dictionary.Enabled, r.settings.Dictionary.Glossary = true, true
	live := "Ask Term7 about it."
	r.hearLive(live)
	waitLive(t, r, live)
	if _, err := liveCleanUp(r, live); err != nil {
		t.Fatal(err)
	}
	reqs := model.requests()
	if len(reqs) != 1 {
		t.Fatalf("%d requests, want only the one while recording", len(reqs))
	}
	user := reqs[0].Messages[len(reqs[0].Messages)-1].Content
	if !strings.Contains(user, "<glossary>\n- Term7\n</glossary>") || strings.Contains(user, "Term8") {
		t.Errorf("user message %q, want the glossary of the terms it mentions", user)
	}
}
