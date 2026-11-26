# Utter

On-device voice dictation for macOS. Press a key, speak, press it again: the text is recognised and cleaned up on your Mac and typed where your cursor is, about **0.44 s** after you stop (median).

Utter is a build of [Voxt](../Voxt) from [sgl-project/sglang-omni](https://github.com/sgl-project/sglang-omni). Its `cassie` branch adds about 20 commits on top of upstream (`git log upstream/main..cassie`); the rest of this repository's history is upstream's, kept so the branch can be rebased onto new upstream releases.

## What it changes

1. **Clean up while you speak, with a model on this Mac.** Upstream turns live clean-up off when the language model runs on `127.0.0.1`. Utter turns it on: each finished sentence goes to the model while you are still speaking, so little is left to clean when you stop.
2. **Reuse that work when only punctuation changed.** The final transcript often adds or drops a comma compared with the live one, which used to discard every cleaned sentence after it. Utter matches the two ignoring punctuation, white space and case.
3. **Large dictionaries.** Live clean-up keeps working with more than 30 dictionary terms; each request lists only the terms its own text mentions.
4. **A local-only app.** One speech model (Qwen3-ASR 0.6B 4bit), no cloud speech services, no Translate / Rewrite / Meetings / Notes pages, no update checks. Built with the `utter` build tag; without it the code is plain Voxt.

Every change has a test that fails with the change reverted.

## Results

M1 Pro, 16 GB. Five Chinese dictations of 7–39 s replayed through the real helper, three runs each. Time from releasing the key to inserted text, median.

| Clean-up model | Live clean-up off | Live clean-up on |
|---|---|---|
| Qwen3-4B-Instruct Q4_0 (llama.cpp) | 1.31 s | **0.57 s** |
| Qwen3-0.6B fine-tuned, Q8_0 ([voxt-polish](https://github.com/cassieliang6709/voxt-polish)) | 0.48 s | **0.44 s** |

With the 4B model, live clean-up halves the wait. With the fine-tuned 0.6B model the clean-up itself takes a fraction of a second, so most of the gain comes from the model; speech recognition is then most of what is left (0.28 s). Per-recording numbers and method: [UTTER.md](../UTTER.md) (Chinese).

## Install

Needs an Apple silicon Mac with macOS 15 or later.

1. Download `Utter-*.zip` from [Releases](https://github.com/cassieliang6709/utter/releases), unzip it and move `Utter.app` to `/Applications`.
2. The app is not notarized, so macOS blocks it on first launch. Allow it once:
   ```sh
   xattr -dr com.apple.quarantine /Applications/Utter.app
   ```
3. Open Utter, then **Settings → Speech models**: download Qwen3-ASR 0.6B (4bit), about 1.3 GB. The voice detector (2.4 MB) comes with it.
4. In **System Settings → Privacy & Security**, give Utter **Microphone** and **Accessibility** access. Accessibility lets the shortcut work everywhere and lets Utter type the text.
5. Press **Right ⌥** to start, speak, press it again to stop.

Without a language model Utter types the text as recognised. For clean-up, run any OpenAI-compatible server on this Mac, for example llama.cpp:

```sh
brew install llama.cpp
llama-server -hf unsloth/Qwen3-4B-Instruct-2507-GGUF:Q4_0 --host 127.0.0.1 --port 8090
```

Then add it in **Settings → Language models** as a custom OpenAI-compatible service with the address `http://127.0.0.1:8090/v1`, and choose it for Transcribe. Ollama and LM Studio work the same way.

Notes:

- Utter and the official Voxt share their data folder (`~/Library/Application Support/VoxtNew`): dictionary, history and settings carry over. They use the same default shortcut, so run one at a time.
- The app is ad hoc signed. After you rebuild it, macOS may forget its Accessibility access; turn it off and on again in System Settings.
- The fine-tuned 0.6B model is not published: it was trained on personal dictation. Its training and evaluation code is in [voxt-polish](https://github.com/cassieliang6709/voxt-polish).

## Build from source

Needs Go 1.27.4, Xcode, and the official [Voxt](https://github.com/sgl-project/sglang-omni/releases) installed in `/Applications` (its speech runtime is reused instead of building it, about 1 GB).

```sh
git clone -b cassie https://github.com/cassieliang6709/utter.git
cd utter/Voxt
scripts/build-helper.sh
scripts/build-utter.sh   # builds with -tags utter and installs /Applications/Utter.app
```

Tests:

```sh
scripts/check.sh
CGO_ENABLED=0 go test ./...
CGO_ENABLED=0 go test -tags utter -run TestUtterProfile .
```

## Credits

Voxt and SGLang-Omni are by the SGLang-Omni team, Apache-2.0; Utter keeps the same licence ([LICENSE](../LICENSE), [Voxt/NOTICE](../Voxt/NOTICE)). The live clean-up itself is upstream's ([#4592](https://github.com/sgl-project/sglang-omni/pull/4592)); the local-model switch was also proposed upstream in [#4643](https://github.com/sgl-project/sglang-omni/pull/4643).
