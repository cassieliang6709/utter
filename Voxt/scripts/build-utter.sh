#!/usr/bin/env bash
# Build and install Utter.app (streamlined local-only UI + stripped omni runtime).
# Keeps only qwen3_asr_server + silero_vad_server + shared MLX/soxr libs in omni/bin/
# and removes the 6 unused ASR server binaries (Whisper, Parakeet, SenseVoice, Cohere,
# Nemotron, MOSS).
set -euo pipefail

VOXT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$VOXT_DIR"

OMNI_SRC="/Applications/Voxt.app/Contents/Resources/omni"
OMNI_DST="$VOXT_DIR/resources/darwin-arm64/omni"

if [ ! -d "$OMNI_DST/bin" ] && [ -d "$OMNI_SRC/bin" ]; then
  mkdir -p "$OMNI_DST"
  cp -R "$OMNI_SRC/." "$OMNI_DST/"
fi

# Strip the 6 unused ASR server binaries (Utter is locked to Qwen3-ASR 0.6B + Silero VAD).
UNUSED_SERVERS=(
  cohere_transcribe_server
  moss_transcribe_diarize_server
  nemotron_asr_server
  parakeet_server
  sensevoice_server
  whisper_server
)
for bin in "${UNUSED_SERVERS[@]}"; do
  rm -f "$OMNI_DST/bin/$bin"
done

echo "=== omni/bin contents kept for Utter ==="
ls -lh "$OMNI_DST/bin"

echo "=== Building Utter.app with -tags=utter ==="
GOFLAGS="-tags=utter" go tool mygo build -skip-dmg

echo "=== Installing to /Applications/Utter.app ==="
osascript -e 'tell application "Utter" to quit' 2>/dev/null || true
sleep 0.5
pkill -f "/Applications/Utter.app/Contents/MacOS/Utter" 2>/dev/null || true
sleep 0.3
rm -rf /Applications/Utter.app
ditto build/darwin-arm64/Utter.app /Applications/Utter.app
touch /Applications/Utter.app
echo "Installed /Applications/Utter.app ($(du -sh /Applications/Utter.app | awk '{print $1}'))"
