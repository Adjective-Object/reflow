package ansi

import (
	"io"
	"unicode/utf8"

	"github.com/muesli/reflow/internal/statemachine"
)

type Writer struct {
	Forward io.Writer

	state    statemachine.AnsiState
	lastStep statemachine.StateTransition
}

func NewWriterForState(
	state statemachine.AnsiState,
	forward io.Writer,
) *Writer {
	return &Writer{
		Forward: forward,
		state:   state,
	}
}

// Write is used to write content to the ANSI buffer.
func (w *Writer) Write(b []byte) (int, error) {
	for i, c := range b {
		if err := w.WriteByte(c); err != nil {
			return i, err
		}
	}
	return len(b), nil
}

// WriteString is used to write content to the ANSI buffer.
func (w *Writer) WriteString(s string) (int, error) {
	for i := 0; i < len(s); i++ {
		if err := w.WriteByte(s[i]); err != nil {
			return i, err
		}
	}
	return len(s), nil
}

// WriteRune is used to write content to the ANSI buffer.
func (w *Writer) WriteRune(r rune) (int, error) {
	var runeBuf [4]byte
	n := utf8.EncodeRune(runeBuf[:], r)
	for i := 0; i < n; i++ {
		if err := w.WriteByte(runeBuf[i]); err != nil {
			return i, err
		}
	}
	return n, nil
}

// WriteByte is used to write content to the ANSI buffer.
func (w *Writer) WriteByte(b byte) error {
	w.lastStep = w.state.Next(b).StateTransition
	_, err := w.Forward.Write([]byte{b})
	return err
}

func (w *Writer) LastSequence() string {
	return string(w.state.ResetSequence())
}

// Resets the current ansi state to a neutral ansi state
//
// The current ansi state can be restored by calling RestoreAnsi
func (w *Writer) ResetAnsi() {
	if !w.state.IsDirty() {
		return
	}
	_, _ = w.Forward.Write(w.state.ResetSequence())
}

// Restores the saved ansi state
func (w *Writer) RestoreAnsi() {
	_, _ = w.Forward.Write(w.state.RestoreSequence())
}

// Clears the internal state of the ansistate
func (w *Writer) ClearAnsi() {
	w.state.ClearState()
}

// Check if the writer is currently in a printable sequence
// of characters (e.g. not in an ANSI escape sequence)
func (w *Writer) IsPrinting() bool {
	return w.lastStep.IsPrinting()
}

func (w *Writer) ExportState() statemachine.AnsiState {
	return w.state
}
