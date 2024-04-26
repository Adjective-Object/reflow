package ansi

import (
	"bytes"
	"unicode"
	"unicode/utf8"

	"github.com/mattn/go-runewidth"
	"github.com/muesli/reflow/internal/statemachine"
)

// Buffer is a buffer aware of ANSI escape sequences.
type Buffer struct {
	bytes.Buffer
}

// PrintableRuneWidth returns the cell width of all printable runes in the
// buffer.
func (w Buffer) PrintableRuneWidth() int {
	return PrintableRuneWidth(w.String())
}

// PrintableRuneWidth returns the cell width of the given string.
func PrintableRuneWidth(s string) int {
	var n int
	stateMachine := statemachine.StateMachine{}
	for i := 0; i < len(s); i++ {
		var state statemachine.StateTransition
		if b := s[i]; b <= unicode.MaxASCII && b > 0x20 {
			// short-circuit for printable ASCII characters (most characters)
			state = stateMachine.Next(b)
			// all ASCII characters are 1 character wide
			if state.IsPrinting() {
				n += 1
			}
		} else {
			// collect the rune
			r, size := utf8.DecodeRuneInString(s[i:])
			j := i + size - 1
			// advance state by each byte
			for {
				// postcondition loop for performance
				state = stateMachine.Next(s[i])
				if i >= j {
					break
				}
				i++
			}
			// if we are in a printable state, count the rune width
			if state.IsPrinting() {
				n += runewidth.RuneWidth(r)
			}
		}

	}

	return n
}
