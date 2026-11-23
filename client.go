package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/gorilla/websocket"
)

const streamingURL = "wss://streaming.assemblyai.com/v3/ws"

// Messages the server sends. See https://www.assemblyai.com/docs/streaming/message-sequence
type BeginMessage struct {
	ID        string `json:"id"`
	ExpiresAt int64  `json:"expires_at"`
}

type Word struct {
	Text        string  `json:"text"`
	Start       int     `json:"start"`
	End         int     `json:"end"`
	Confidence  float64 `json:"confidence"`
	WordIsFinal bool    `json:"word_is_final"`
}

type TurnMessage struct {
	TurnOrder  int    `json:"turn_order"`
	EndOfTurn  bool   `json:"end_of_turn"`
	Transcript string `json:"transcript"`
	Words      []Word `json:"words"`
}

type TerminationMessage struct {
	AudioDurationSeconds   float64 `json:"audio_duration_seconds"`
	SessionDurationSeconds float64 `json:"session_duration_seconds"`
}

type ErrorMessage struct {
	ErrorCode int    `json:"error_code"`
	Error     string `json:"error"`
}

// Client is a thin wrapper around the Real-time STT WebSocket.
// Only one goroutine may write to it at a time; gorilla/websocket
// does not allow concurrent writers.
type Client struct {
	conn *websocket.Conn
}

// Config holds the connection parameters sent in the WebSocket URL.
type Config struct {
	APIKey      string
	SampleRate  int
	SpeechModel string
	Keyterms    []string
	VoiceFocus  string // "near-field", "far-field", or "" to leave it off
}

// Connect opens the session. Connection parameters go in the query string;
// the API key goes in the Authorization header with no "Bearer" prefix.
func Connect(cfg Config) (*Client, error) {
	params := url.Values{}
	params.Set("speech_model", cfg.SpeechModel)
	params.Set("sample_rate", strconv.Itoa(cfg.SampleRate))
	params.Set("encoding", "pcm_s16le")
	if len(cfg.Keyterms) > 0 {
		// keyterms_prompt is ONE parameter holding a JSON-encoded array.
		terms, err := json.Marshal(cfg.Keyterms)
		if err != nil {
			return nil, err
		}
		params.Set("keyterms_prompt", string(terms))
	}
	if cfg.VoiceFocus != "" {
		params.Set("voice_focus", cfg.VoiceFocus)
	}

	header := http.Header{}
	header.Set("Authorization", cfg.APIKey)

	conn, resp, err := websocket.DefaultDialer.Dial(streamingURL+"?"+params.Encode(), header)
	if err != nil {
		if resp != nil {
			return nil, fmt.Errorf("connect failed (HTTP %d): %w", resp.StatusCode, err)
		}
		return nil, fmt.Errorf("connect failed: %w", err)
	}
	return &Client{conn: conn}, nil
}

// SendAudio sends one chunk of raw PCM as a binary frame.
// Each chunk must hold between 20 ms and 1000 ms of audio.
func (c *Client) SendAudio(pcm []byte) error {
	return c.conn.WriteMessage(websocket.BinaryMessage, pcm)
}

// Terminate asks the server to finalize the open turn and end the session.
// The server replies with a Termination message, then closes the socket.
func (c *Client) Terminate() error {
	return c.conn.WriteJSON(map[string]string{"type": "Terminate"})
}

func (c *Client) Close() error { return c.conn.Close() }

// ReadMessage blocks for the next server message and returns one of
// *BeginMessage, *TurnMessage, *TerminationMessage, *ErrorMessage,
// or nil for message types this program doesn't use (SpeechStarted, Warning).
func (c *Client) ReadMessage() (any, error) {
	_, data, err := c.conn.ReadMessage()
	if err != nil {
		return nil, err
	}
	var envelope struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, err
	}

	var msg any
	switch envelope.Type {
	case "Begin":
		msg = &BeginMessage{}
	case "Turn":
		msg = &TurnMessage{}
	case "Termination":
		msg = &TerminationMessage{}
	case "Error":
		msg = &ErrorMessage{}
	default:
		return nil, nil
	}
	if err := json.Unmarshal(data, msg); err != nil {
		return nil, err
	}
	return msg, nil
}
