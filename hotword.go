package main

import (
	"strings"
	"unicode"
)

// normalize lowercases text and replaces punctuation with spaces, so
// "Jarvis," and "jarvis" compare equal.
func normalize(s string) []string {
	return strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '\''
	})
}

// findHotword reports whether the hotword (one or more words) appears in the
// transcript as whole words, and returns the text spoken after it.
// Matching whole words means "Jarvis" never fires on "Jarvisville".
func findHotword(transcript, hotword string) (found bool, command string) {
	words := normalize(transcript)
	target := normalize(hotword)
	if len(target) == 0 {
		return false, ""
	}
	for i := 0; i+len(target) <= len(words); i++ {
		match := true
		for j := range target {
			if words[i+j] != target[j] {
				match = false
				break
			}
		}
		if match {
			return true, strings.Join(words[i+len(target):], " ")
		}
	}
	return false, ""
}

// detector fires once per turn: on the first partial that contains the
// hotword (fast wake), then reports the command when the turn is final.
type detector struct {
	hotword   string
	firedTurn int // turn_order that already fired, -1 if none
}

func newDetector(hotword string) *detector {
	return &detector{hotword: hotword, firedTurn: -1}
}

type event struct {
	Wake    bool   // the hotword was just heard in this turn
	Command string // text after the hotword, set only on the final turn
	Final   bool
}

func (d *detector) onTurn(t *TurnMessage) (event, bool) {
	found, command := findHotword(t.Transcript, d.hotword)
	if !found {
		return event{}, false
	}
	ev := event{Final: t.EndOfTurn}
	if d.firedTurn != t.TurnOrder {
		d.firedTurn = t.TurnOrder
		ev.Wake = true
	}
	if t.EndOfTurn {
		ev.Command = command
	}
	if !ev.Wake && !ev.Final {
		return event{}, false // already woke on this turn; wait for the final
	}
	return ev, true
}
