package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gordonklaus/portaudio"
)

// AssemblyAI-STT-3.7-Realtime-Agent is the model the docs recommend for voice agents.
const speechModel = "assemblyai-stt-3-7-realtime-agent"

func main() {
	logger := log.New(os.Stderr, "", log.Ltime)

	if len(os.Args) < 2 {
		logger.Fatal(`usage: go run . <hotword>   e.g. go run . Jarvis`)
	}
	hotword := os.Args[1]

	apiKey := os.Getenv("ASSEMBLYAI_API_KEY")
	if apiKey == "" {
		logger.Fatal("set the ASSEMBLYAI_API_KEY environment variable")
	}

	if err := portaudio.Initialize(); err != nil {
		logger.Fatal(err)
	}
	defer portaudio.Terminate()

	device, err := portaudio.DefaultInputDevice()
	if err != nil {
		logger.Fatal(err)
	}
	sampleRate := int(device.DefaultSampleRate)
	framesPerBuffer := sampleRate / 10 // 100 ms per chunk

	client, err := Connect(Config{
		APIKey:      apiKey,
		SampleRate:  sampleRate,
		SpeechModel: speechModel,
		Keyterms:    []string{hotword}, // bias recognition toward the hotword
		VoiceFocus:  "far-field",       // laptop and room mics; use "near-field" for a headset
	})
	if err != nil {
		logger.Fatal(err)
	}
	defer client.Close()

	rec, err := newRecorder(sampleRate, framesPerBuffer)
	if err != nil {
		logger.Fatal(err)
	}
	defer rec.Close()

	// One goroutine reads every server message. It closes `done` when the
	// session ends (Termination, error, or a closed socket).
	done := make(chan struct{})
	go readLoop(client, newDetector(hotword), logger, done)

	if err := rec.Start(); err != nil {
		logger.Fatal(err)
	}
	fmt.Printf("Listening for %q. Press Ctrl+C to stop.\n", hotword)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	// The main goroutine is the only writer on the socket.
loop:
	for {
		select {
		case <-sigs:
			break loop
		case <-done:
			break loop
		default:
		}
		pcm, err := rec.Read() // blocks for 100 ms of audio
		if err != nil {
			logger.Printf("mic read: %v", err)
			continue
		}
		if err := client.SendAudio(pcm); err != nil {
			logger.Printf("send: %v", err)
			break
		}
	}

	_ = rec.Stop()
	fmt.Println("\nStopping...")

	// Terminate finalizes the open turn. Wait for the server's Termination
	// message instead of sleeping, so the last turn isn't lost.
	if err := client.Terminate(); err == nil {
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			logger.Println("no Termination message after 5 s; closing")
		}
	}
}

func readLoop(client *Client, det *detector, logger *log.Logger, done chan<- struct{}) {
	defer close(done)
	for {
		msg, err := client.ReadMessage()
		if err != nil {
			logger.Printf("connection closed: %v", err)
			return
		}
		switch m := msg.(type) {
		case *BeginMessage:
			fmt.Printf("Session %s started\n", m.ID)

		case *TurnMessage:
			if m.EndOfTurn {
				fmt.Printf("\r%-80s\n", m.Transcript)
			} else {
				fmt.Printf("\r%-80.80s", m.Transcript)
			}
			if ev, ok := det.onTurn(m); ok {
				if ev.Wake {
					fmt.Println("\n>>> Hotword detected. I'm listening.")
				}
				if ev.Final && ev.Command != "" {
					fmt.Printf(">>> Command: %q\n", ev.Command)
					// Hand ev.Command to your assistant logic here.
				}
			}

		case *ErrorMessage:
			logger.Printf("server error %d: %s", m.ErrorCode, m.Error)

		case *TerminationMessage:
			fmt.Printf("Session ended: %.0f s of audio, %.0f s billed session time\n",
				m.AudioDurationSeconds, m.SessionDurationSeconds)
			return
		}
	}
}
