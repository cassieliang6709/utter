package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/sgl-project/sglang-omni/Voxt/internal/llm"
	"github.com/sgl-project/sglang-omni/Voxt/internal/settings"
)

// TestAdoptPolish: only settings with no language model at all get the
// bundled model; a service the user set up, or a feature's own choice,
// stays as it is.
func TestAdoptPolish(t *testing.T) {
	s := settings.Defaults()
	if !adoptPolish(&s) || !usesBundledPolish(s) {
		t.Fatalf("empty settings: providers %+v, want the bundled model as custom", s.Providers.LLM)
	}
	if adoptPolish(&s) || len(s.Providers.LLM) != 1 {
		t.Errorf("adopted twice: %+v", s.Providers.LLM)
	}

	mine := settings.Defaults()
	mine.Providers.LLM = []settings.Provider{{ID: llm.Custom, Endpoint: "http://127.0.0.1:8090/v1"}}
	if adoptPolish(&mine) || usesBundledPolish(mine) {
		t.Errorf("a custom service of the user's own was taken over: %+v", mine.Providers.LLM)
	}

	chosen := settings.Defaults()
	chosen.Features.Transcribe.LLM = "openai"
	if adoptPolish(&chosen) {
		t.Errorf("Transcribe's own choice was overridden")
	}

	slash := settings.Defaults()
	slash.Providers.LLM = []settings.Provider{{ID: llm.Custom, Endpoint: polishEndpoint + "/"}}
	if !usesBundledPolish(slash) {
		t.Errorf("a trailing slash hides the bundled model")
	}
}

// TestDownloadVerified: the model is kept only with the right checksum;
// a wrong file or a failed request leaves nothing behind.
func TestDownloadVerified(t *testing.T) {
	body := []byte("not really a model, but bytes to check")
	sum := sha256.Sum256(body)
	good := hex.EncodeToString(sum[:])
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/model.gguf" {
			http.NotFound(w, r)
			return
		}
		http.ServeContent(w, r, "model.gguf", time.Time{}, bytes.NewReader(body))
	}))
	defer srv.Close()
	dir := t.TempDir()
	ctx := context.Background()

	path := filepath.Join(dir, "ok.gguf")
	var last int64
	if err := downloadVerified(ctx, srv.Client(), srv.URL+"/model.gguf", path, good, time.Second, func(done, _ int64) { last = done }); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(path); string(got) != string(body) || last != int64(len(body)) {
		t.Errorf("file %q, progress %d", got, last)
	}

	bad := filepath.Join(dir, "bad.gguf")
	if err := downloadVerified(ctx, srv.Client(), srv.URL+"/model.gguf", bad, "00", time.Second, nil); err == nil {
		t.Errorf("a wrong checksum was accepted")
	}
	missing := filepath.Join(dir, "missing.gguf")
	if err := downloadVerified(ctx, srv.Client(), srv.URL+"/nothing", missing, good, time.Second, nil); err == nil {
		t.Errorf("a 404 was accepted")
	}
	for _, p := range []string{bad, bad + ".part", missing} {
		if _, err := os.Stat(p); err == nil {
			t.Errorf("%s was left behind", filepath.Base(p))
		}
	}
}

// TestDownloadStallsThenGoesOn: a connection that stops bringing data, or a
// server that never answers, is dropped after the stall time instead of
// hanging; the bytes that came are kept and the next attempt asks only for
// the rest.
func TestDownloadStallsThenGoesOn(t *testing.T) {
	body := bytes.Repeat([]byte("0123456789"), 1000)
	sum := sha256.Sum256(body)
	good := hex.EncodeToString(sum[:])
	half := len(body) / 2
	var ranges []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ranges = append(ranges, r.Header.Get("Range"))
		if r.URL.Path == "/stall" {
			w.Header().Set("Content-Length", strconv.Itoa(len(body)))
			_, _ = w.Write(body[:half]) // then nothing, as a proxy that stops passing data
			w.(http.Flusher).Flush()
			<-r.Context().Done()
			return
		}
		http.ServeContent(w, r, "model.gguf", time.Time{}, bytes.NewReader(body))
	}))
	defer srv.Close()
	path := filepath.Join(t.TempDir(), "model.gguf")

	start := time.Now()
	err := downloadVerified(context.Background(), srv.Client(), srv.URL+"/stall", path, good, 200*time.Millisecond, nil)
	if err == nil || time.Since(start) > 5*time.Second {
		t.Fatalf("stalled download: err %v after %v, want an error soon", err, time.Since(start))
	}
	if fi, err := os.Stat(path + ".part"); err != nil || fi.Size() != int64(half) {
		t.Fatalf("the bytes that came were not kept: %v %v", fi, err)
	}
	// A server that never answers is dropped too.
	hang := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() }))
	defer hang.Close()
	start = time.Now()
	if err := downloadVerified(context.Background(), hang.Client(), hang.URL+"/model.gguf", path, good, 200*time.Millisecond, nil); err == nil || time.Since(start) > 5*time.Second {
		t.Fatalf("a server with no answer: err %v after %v", err, time.Since(start))
	}
	if err := downloadVerified(context.Background(), srv.Client(), srv.URL+"/model.gguf", path, good, time.Second, nil); err != nil {
		t.Fatal(err)
	}
	if want := "bytes=" + strconv.Itoa(half) + "-"; ranges[len(ranges)-1] != want {
		t.Errorf("second request asked %q, want %q", ranges[len(ranges)-1], want)
	}
	if got, _ := os.ReadFile(path); !bytes.Equal(got, body) {
		t.Errorf("the file was not put together right (%d bytes)", len(got))
	}
}
