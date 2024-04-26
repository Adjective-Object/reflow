package ansi

import (
	"bytes"

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

	bi := 0
	for i, r := range s {
		var state statemachine.StateTransition
		if r <= 0x7F {
			state = stateMachine.Next(s[i])
		} else {
			for ; bi <= i; bi++ {
				state = stateMachine.Next(s[bi])
			}
		}
		if state.IsPrinting() {
			n += runewidth.RuneWidth(r)
		}
	}

	return n
}
