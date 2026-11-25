//go:build utter

package main

// utterLockedUI enables Utter's streamlined local-only UI:
//   - Sidebar keeps Home, History, Transcribe, Dictionary, Logs, Settings.
//   - Speech models lock to asr.fast (Qwen3-ASR 0.6B 4bit) + Silero VAD.
//   - Settings hide cloud updater, remote LLM APIs, unused feature toggles/shortcuts,
//     and auto-model pickers on Transcribe.
const utterLockedUI = true
