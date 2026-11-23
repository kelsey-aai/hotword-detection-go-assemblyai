package main

import (
	"bytes"
	"encoding/binary"

	"github.com/gordonklaus/portaudio"
)

// recorder reads 16-bit mono PCM from the default input device.
type recorder struct {
	stream *portaudio.Stream
	in     []int16
}

// newRecorder opens the default microphone. framesPerBuffer sets how much
// audio each Read returns: sampleRate/10 frames is 100 ms.
func newRecorder(sampleRate, framesPerBuffer int) (*recorder, error) {
	in := make([]int16, framesPerBuffer)
	stream, err := portaudio.OpenDefaultStream(1, 0, float64(sampleRate), framesPerBuffer, in)
	if err != nil {
		return nil, err
	}
	return &recorder{stream: stream, in: in}, nil
}

// Read blocks until one buffer of audio is captured and returns it as
// little-endian PCM16 bytes, the format the streaming API expects.
func (r *recorder) Read() ([]byte, error) {
	if err := r.stream.Read(); err != nil {
		return nil, err
	}
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, r.in); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (r *recorder) Start() error { return r.stream.Start() }
func (r *recorder) Stop() error  { return r.stream.Stop() }
func (r *recorder) Close() error { return r.stream.Close() }
