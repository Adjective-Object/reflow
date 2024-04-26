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
	i := 0
	for i < len(s) {
		var stateTrans statemachine.StateTransition
		var b = s[i]
		if b <= unicode.MaxASCII && b > 0x20 {
			// short-circuit for printable ASCII characters (most characters)
			stateTrans = stateMachine.Next(b)
			// all ASCII characters are 1 character wide
			if stateTrans.IsPrinting() {
				n += 1
			}
			i++
		} else {
			// collect the rune
			r, size := utf8.DecodeRuneInString(s[i:])
			j := i + size
			// advance state by each byte
			for {
				// postcondition loop for performance
				stateTrans = stateMachine.Next(s[i])
				i++
				if i >= j {
					break
				}
			}
			// if we are in a printable state, count the rune width
			// of the multibyte character
			if stateTrans.IsPrinting() {
				n += runewidth.RuneWidth(r)
			}
		}

	}

	return n
}
