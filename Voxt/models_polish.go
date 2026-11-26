package main

// The Utter build's own clean-up model (-tags utter): a fine-tuned
// Qwen3-0.6B (Hugging Face liangyue36/qwen3-0.6b-dictation-polish-GGUF)
// served by voxt_polish_cpp, which the bundle carries in
// Contents/Resources/polish. Settings with no language model at all get
// it as the custom service at polishEndpoint; the model downloads on the
// first start and the engine runs while the app does. A custom service at
// any other address is the user's own: nothing downloads or runs. Other
// builds never start it (utterLockedUI).

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sgl-project/sglang-omni/Voxt/internal/core"
	"github.com/sgl-project/sglang-omni/Voxt/internal/i18n"
	"github.com/sgl-project/sglang-omni/Voxt/internal/llm"
	"github.com/sgl-project/sglang-omni/Voxt/internal/settings"
)

const (
	polishPort      = 18090
	polishModelName = "qwen3-0.6b-polish"
	polishModelFile = "qwen3-0.6b-polish-q8_0.gguf"
	polishModelPath = "liangyue36/qwen3-0.6b-dictation-polish-GGUF/resolve/e5921f110bae656f8ce60e0d176b034b90819ead/" + polishModelFile
	polishModelSHA  = "0bca63f64e65d2ccec61a756dd27f7c1306783fd367b634ae6a4733628bf7185"
	problemPolish   = "problem/polish"
	polishSource    = "polish"
)

// polishHosts are where the model is downloaded from, in order: Hugging
// Face, then its mirror for networks that cannot reach it.
var polishHosts = []string{"https://huggingface.co/", "https://hf-mirror.com/"}

var polishEndpoint = "http://127.0.0.1:" + strconv.Itoa(polishPort) + "/v1"

// adoptPolish gives settings with no language model set up the bundled
// clean-up model as the custom service; false when there was one.
func adoptPolish(s *settings.Settings) bool {
	if len(s.Providers.LLM) > 0 || s.Features.Transcribe.LLM != "" {
		return false
	}
	s.Providers.LLM = append(s.Providers.LLM, settings.Provider{ID: llm.Custom, Endpoint: polishEndpoint, Model: polishModelName})
	return true
}

// usesBundledPolish says whether the custom service is the bundled model.
func usesBundledPolish(s settings.Settings) bool {
	for _, p := range s.Providers.LLM {
		if p.ID == llm.Custom {
			return strings.TrimRight(strings.TrimSpace(p.Endpoint), "/") == polishEndpoint
		}
	}
	return false
}

// polishRun is the running bundled model: its download and engine.
type polishRun struct {
	cancel context.CancelFunc
	done   sync.WaitGroup
}

// startPolish adopts and starts the bundled clean-up model (main thread,
// once at start).
func (a *app) startPolish() {
	if !utterLockedUI || a.polish != nil {
		return
	}
	if s := a.prefs; adoptPolish(&s) {
		a.updateSettings(func(s *settings.Settings) { adoptPolish(s) })
	}
	if !usesBundledPolish(a.prefs) {
		return
	}
	exe, _ := os.Executable()
	res := filepath.Join(filepath.Dir(exe), "..", "Resources", "polish")
	bin, prompt := filepath.Join(res, "voxt_polish_cpp"), filepath.Join(res, "system_prompt.txt")
	if _, err := os.Stat(bin); err != nil {
		a.polishLog("warning", "no bundled clean-up engine: "+err.Error())
		return
	}
	model := filepath.Join(a.dataHome, "polish", polishModelFile)
	ctx, cancel := context.WithCancel(context.Background())
	run := &polishRun{cancel: cancel}
	a.polish = run
	run.done.Add(1)
	go func() {
		defer run.done.Done()
		if err := a.ensurePolishModel(ctx, model); err != nil {
			if ctx.Err() == nil {
				a.polishLog("error", "clean-up model download: "+err.Error())
				a.post(func() {
					a.setProblem(problemPolish, "download", &core.Notification{Severity: core.SeverityWarning,
						Title: i18n.T("Could not download the clean-up model"),
						Body:  i18n.T("Transcribe types the text as recognised until it is ready. Voxt tries again at its next start.")})
				})
			}
			return
		}
		a.post(func() { a.setProblem(problemPolish, "", nil) })
		runPolishEngine(ctx, bin, model, prompt, a.polishLog)
	}()
}

// stopPolish stops the download and the engine (quit).
func (a *app) stopPolish() {
	if run := a.polish; run != nil {
		run.cancel()
		run.done.Wait()
		a.polish = nil
	}
}

// ensurePolishModel downloads the model to path unless it is there,
// checked against polishModelSHA, showing its progress.
func (a *app) ensurePolishModel(ctx context.Context, path string) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	last := -1
	progress := func(done, total int64) {
		pct := int(done * 100 / max(total, 1))
		if pct/5 == last/5 {
			return
		}
		last = pct
		mb := total >> 20
		a.post(func() {
			a.setProblem(problemPolish, "downloading", &core.Notification{Severity: core.SeverityInfo,
				Title: i18n.T("Downloading the clean-up model"),
				Body:  i18n.T("%d%% of %d MB. Transcribe types the text as recognised until it is ready.", pct, mb)})
		})
	}
	// Attempts go on, Hugging Face and its mirror in turn, while they
	// bring bytes; six in a row that bring nothing give up until the next
	// start.
	var err error
	for attempt, idle := 0, 0; idle < 6; attempt++ {
		host := polishHosts[attempt%len(polishHosts)]
		before := partSize(path)
		if err = downloadVerified(ctx, http.DefaultClient, host+polishModelPath, path, polishModelSHA, polishStall, progress); err == nil || ctx.Err() != nil {
			return err
		}
		if partSize(path) > before {
			idle = 0
		} else {
			idle++
		}
		a.polishLog("warning", "clean-up model from "+host+": "+err.Error())
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(3 * time.Second):
		}
	}
	return err
}

// partSize is the size of what earlier attempts left of path's download.
func partSize(path string) int64 {
	fi, err := os.Stat(path + ".part")
	if err != nil {
		return 0
	}
	return fi.Size()
}

// polishStall is how long a download may receive nothing before it is
// dropped and tried again: a proxy that stops passing data keeps the
// connection open forever.
var polishStall = 30 * time.Second

// downloadVerified downloads url to path through path+".part", going on
// from what an earlier attempt left there (a Range request), and keeps the
// file only when its SHA-256 is sum. A connection that brings nothing for
// stall is dropped, its bytes kept for the next attempt; a file with the
// wrong checksum is deleted.
func downloadVerified(ctx context.Context, client *http.Client, url, path, sum string, stall time.Duration, progress func(done, total int64)) error {
	part := path + ".part"
	f, err := os.OpenFile(part, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	h := sha256.New()
	have, err := io.Copy(h, f)
	if err != nil {
		return err
	}
	reqCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	// The watchdog runs from the request on: a connection, a TLS
	// handshake or a redirect can hang as well as the body.
	var last atomic.Int64
	last.Store(time.Now().UnixNano())
	var stalled atomic.Bool
	go func() {
		t := time.NewTicker(stall / 4)
		defer t.Stop()
		for {
			select {
			case <-reqCtx.Done():
				return
			case <-t.C:
				if time.Since(time.Unix(0, last.Load())) > stall {
					stalled.Store(true)
					cancel()
					return
				}
			}
		}
	}()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	if have > 0 {
		req.Header.Set("Range", "bytes="+strconv.FormatInt(have, 10)+"-")
	}
	resp, err := client.Do(req)
	if stalled.Load() {
		return fmt.Errorf("no answer for %s", stall)
	}
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	switch {
	case resp.StatusCode == http.StatusPartialContent && have > 0:
	case resp.StatusCode == http.StatusOK:
		if err := f.Truncate(0); err != nil {
			return err
		}
		if _, err := f.Seek(0, io.SeekStart); err != nil {
			return err
		}
		h.Reset()
		have = 0
	default:
		return fmt.Errorf("%s: %s", url, resp.Status)
	}
	total := have + resp.ContentLength
	_, err = io.Copy(io.MultiWriter(f, h), &progressReader{r: resp.Body, total: total, done: have, last: &last, fn: progress})
	if stalled.Load() {
		return fmt.Errorf("no data for %s; kept %s to go on from", stall, filepath.Base(part))
	}
	if err != nil {
		return err
	}
	if err := checkSum(h, sum); err != nil {
		_ = f.Close()
		_ = os.Remove(part) // a wrong file is never kept, nor gone on from
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(part, path)
}

func checkSum(h hash.Hash, want string) error {
	if got := hex.EncodeToString(h.Sum(nil)); got != want {
		return fmt.Errorf("checksum %s, want %s", got, want)
	}
	return nil
}

type progressReader struct {
	r     io.Reader
	total int64
	done  int64
	last  *atomic.Int64 // the time of the last bytes, for the stall check
	fn    func(done, total int64)
}

func (p *progressReader) Read(b []byte) (int, error) {
	n, err := p.r.Read(b)
	p.done += int64(n)
	if n > 0 && p.last != nil {
		p.last.Store(time.Now().UnixNano())
	}
	if p.fn != nil && p.total > 0 {
		p.fn(p.done, p.total)
	}
	return n, err
}

// runPolishEngine runs the engine until ctx ends, starting it again after
// it exits; it gives up after three exits within a minute of starting
// (the port taken, a broken model).
func runPolishEngine(ctx context.Context, bin, model, prompt string, log func(level, msg string)) {
	quick := 0
	for ctx.Err() == nil && quick < 3 {
		start := time.Now()
		cmd := exec.CommandContext(ctx, bin, "--model", model, "--system-prompt", prompt,
			"--draft-max", "6", "--serve", "--port", strconv.Itoa(polishPort))
		cmd.Stdout, cmd.Stderr = polishLogWriter{log}, polishLogWriter{log}
		err := cmd.Run()
		if ctx.Err() != nil {
			return
		}
		log("warning", fmt.Sprintf("clean-up engine exited: %v", err))
		if time.Since(start) < time.Minute {
			quick++
		} else {
			quick = 0
		}
		select {
		case <-ctx.Done():
		case <-time.After(2 * time.Second):
		}
	}
	if quick >= 3 {
		log("error", "clean-up engine keeps exiting; not starting it again")
	}
}

type polishLogWriter struct{ log func(level, msg string) }

func (w polishLogWriter) Write(b []byte) (int, error) {
	for _, line := range strings.Split(strings.TrimSpace(string(b)), "\n") {
		if line != "" {
			w.log("debug", line)
		}
	}
	return len(b), nil
}

// polishLog writes a line about the bundled model to the app's log.
func (a *app) polishLog(level, msg string) {
	if lg := a.logs; lg != nil {
		lg.Append(core.LogEntry{Time: time.Now(), Level: core.Severity(level), Source: polishSource, Message: msg})
	}
}
