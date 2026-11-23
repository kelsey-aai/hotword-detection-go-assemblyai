# Hotword detection in Go with AssemblyAI

A command-line voice assistant trigger written in Go. It streams your microphone to AssemblyAI's Real-time STT WebSocket, watches the transcript for a hotword such as "Jarvis", and prints the command you speak after it.

It uses AssemblyAI-STT-3.7-Realtime-Agent (`assemblyai-stt-3-7-realtime-agent`, $0.45/hr of session time), the model the docs recommend for voice agents. The hotword is passed as a keyterm, so the model is biased toward hearing it.

This is the companion repo for the tutorial: [How to build hotword (wake word) detection in Go with real-time speech-to-text](https://www.assemblyai.com/blog/hotword-detection-with-assemblyai-streaming-speech-to-text).

AssemblyAI's official SDKs are Python and JavaScript, so this repo calls the WebSocket API directly, exactly as the [streaming docs](https://www.assemblyai.com/docs/streaming/getting-started/transcribe-streaming-audio#using-the-websocket-api-directly) describe.

## What's in the repo

| File | What it does |
|------|--------------|
| `recorder.go` | Opens the default microphone with PortAudio and returns 100 ms chunks of 16-bit PCM |
| `client.go` | The WebSocket client: connection parameters, the `Begin` / `Turn` / `Termination` / `Error` messages, and `Terminate` |
| `hotword.go` | Whole-word hotword matching and a detector that fires once per turn |
| `main.go` | Wires it together: one goroutine reads messages, the main goroutine sends audio, Ctrl+C terminates the session cleanly |

## Quick start

Requires Go 1.21+ and the PortAudio C library.

```bash
# macOS
brew install portaudio
# Debian / Ubuntu
sudo apt-get install portaudio19-dev

git clone https://github.com/kelsey-aai/hotword-detection-go-assemblyai
cd hotword-detection-go-assemblyai

export ASSEMBLYAI_API_KEY=your_api_key   # https://www.assemblyai.com/dashboard/signup
go run . Jarvis
```

Say "Jarvis, turn on the lights." You'll see:

```text
Listening for "Jarvis". Press Ctrl+C to stop.
Session 5d2c... started

>>> Hotword detected. I'm listening.
Jarvis, turn on the lights.
>>> Command: "turn on the lights"
```

Multi-word hotwords work too: `go run . "hey jarvis"`.

## How it works

- Connection parameters go in the WebSocket URL: `speech_model`, `sample_rate` (your mic's native rate), `encoding=pcm_s16le`, `keyterms_prompt` (ONE parameter holding a JSON array, e.g. `["Jarvis"]`) and `voice_focus=far-field`.
- The API key goes in the `Authorization` header, with no `Bearer` prefix.
- Audio goes up as binary frames of 100 ms. Each frame must hold 20–1000 ms of audio.
- The detector wakes on the first partial transcript that contains the hotword, then reports the command when the turn is final.
- On Ctrl+C the program sends `{"type":"Terminate"}` and waits for the server's `Termination` message, so the last turn isn't dropped.

## Cost note

Real-time STT is billed on how long the WebSocket stays open, not on how much speech it hears. An always-on listener bills every hour it runs. For a device that listens 24/7, run a small on-device wake word engine first and open the AssemblyAI session only after it fires.

## Troubleshooting

- `pkg-config: exec: not found` or `portaudio.h: No such file`: install PortAudio (see Quick start).
- `connect failed (HTTP 401)`: `ASSEMBLYAI_API_KEY` is missing or wrong.
- The hotword is never detected: check your mic is the system default input, and try `near-field` if you're wearing a headset.

## Resources

- [Tutorial](https://www.assemblyai.com/blog/hotword-detection-with-assemblyai-streaming-speech-to-text)
- [Streaming quickstart](https://www.assemblyai.com/docs/streaming/getting-started/transcribe-streaming-audio)
- [Prompting and keyterms](https://www.assemblyai.com/docs/streaming/prompting-and-keyterms)
- [Voice Focus](https://www.assemblyai.com/docs/streaming/voice-focus)
- [Message sequence](https://www.assemblyai.com/docs/streaming/message-sequence)

## License

MIT
