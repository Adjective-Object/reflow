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
	return PrintableRuneWidthBytes(w.Buffer.Bytes())
}

func isSingleByteRune(b byte) bool {
	return b <= unicode.MaxASCII
}

// expects the input to already pass isSingleByteRune
func isPrintableSingleByteRune(b byte) bool {
	return b >= 0x20
}

// PrintableRuneWidth returns the cell width of the given string.
func PrintableRuneWidth(s string) int {
	var n int
	stateMachine := statemachine.StateMachine{}
	i := 0
	for i < len(s) {
		var stateTrans statemachine.StateTransition
		var b = s[i]
		if isSingleByteRune(b) {
			// short-circuit for printable ASCII characters (most characters)
			stateTrans = stateMachine.Next(b)
			// all ASCII characters are 1 character wide
			if stateTrans.IsPrinting() && isPrintableSingleByteRune(b) {
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

// PrintableRuneWidth returns the cell width of the given byte array,
// interpreted as a string.
//
// This is provided as a separate function from PrintableRuneWidth to
// allow for more efficient processing of byte arrays, as converting
// between byte arrays and strings requires copying the underlying buffer
func PrintableRuneWidthBytes(s []byte) int {
	var n int
	stateMachine := statemachine.StateMachine{}
	i := 0
	for i < len(s) {
		b := s[i]
		var stateTrans statemachine.StateTransition
		if isSingleByteRune(b) {
			// short-circuit for printable ASCII characters (most characters)
			stateTrans = stateMachine.Next(b)
			// all ASCII characters are 1 character wide
			if stateTrans.IsPrinting() && isPrintableSingleByteRune(b) {
				n += 1
			}
			i++
		} else {
			// collect the rune
			r, size := utf8.DecodeRune(s[i:])
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
