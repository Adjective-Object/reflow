package indent

import (
	"bytes"
	"io"

	"github.com/muesli/reflow/ansi"
	"github.com/muesli/reflow/internal/statemachine"
)

//////////////////////////////////////
//                                  //
// Root-Level Constructor Functions //
//                                  //
//////////////////////////////////////

// Common interface fronting AdvancedWriter and SimpleWriter
// for backwards-compatibility with existing code.
type Writer interface {
	Write([]byte) (int, error)
	WriteString(string) (int, error)
	WriteByte(byte) error
	Bytes() []byte
	String() string
}

// MakeSimpleWriter creates a new indent-writer instance, used to write content
// to an internal buffer, with the specified indent level.
func NewSimpleWriter(indent uint) SimpleWriter {
	return SimpleWriter{
		Indent: indent,
	}
}

// NewWriterPipe creates a new indent-writer instance, used to write content to
// the provided io.Writer, with the specified indent level.
//
// If you don't need indent functions, you should prefer using MakeSimpleWriter instead.
//
// See NewWriterPipe for an explanation
func NewWriter(indent uint, indentFunc IndentFunc) Writer {
	return NewWriterPipe(nil, indent, indentFunc)
}

// NewWriterPipe creates a new indent-writer instance, used to write content to
// the provided io.Writer, with the specified indent level.
//
// If you don't need io forwarding or indent functions, you should prefer using
// MakeSimpleWriter instead.
//
// Because this returns an interface type, the return value is always heap-allocated,
// whereas MakeSimpleWriter can be stack-allocated since it returns a concrete type.
//
// If this is used in an inner-loop function, this can lead to a lot of repeated heap
// allocations
func NewWriterPipe(w io.Writer, indent uint, indentFunc IndentFunc) Writer {
	if indentFunc == nil && w == nil {
		s := NewSimpleWriter(indent)
		return &s
	}
	return NewAdvancedWriter(w, indent, indentFunc)
}

///////////////////
//               //
// Simple Writer //
//               //
///////////////////

// A writer that writes to its own internal buffer, with a fixed indent level.
//
// This is the most efficient writer for most usecases, as it doesn't do runtime
// pointer indirection through a function pointer for each byte written.
type SimpleWriter struct {
	Indent     uint
	state      statemachine.AnsiState
	buf        bytes.Buffer
	skipIndent bool
}

// Bytes is shorthand for declaring a new default indent-writer instance,
// used to immediately indent a byte slice.
func Bytes(b []byte, indent uint) []byte {
	// Since the Writer is not returned, we can use a fully on-stack writer
	// and include a pointer into it in the ansiWriter.
	f := SimpleWriter{
		Indent: indent,
	}
	f.buf.Grow(len(b))

	_, _ = f.Write(b)

	return f.Bytes()
}

// String is shorthand for declaring a new default indent-writer instance,
// used to immediately indent a string.
func String(s string, indent uint) string {
	// Since the Writer is not returned, we can use a fully on-stack writer
	// and include a pointer into it in the ansiWriter.
	//
	// TODO: revisit heap escape here: it currently escapes becaues it's
	// referenced by ansiWriter, which shouldn't be escaping here..
	f := SimpleWriter{
		Indent: indent,
	}
	// preallocate buffer to speed up
	f.buf.Grow(len(s))

	_, _ = f.WriteString(s)

	return f.String()
}

// Write is used to write content to the indent buffer.
func (w *SimpleWriter) Write(b []byte) (int, error) {
	var i int
	for i := 0; i < len(b); i++ {
		if err := w.WriteByte(b[i]); err != nil {
			return i, err
		}
	}
	return i, nil
}

// Write is used to write content to the indent buffer.
func (w *SimpleWriter) WriteString(s string) (int, error) {
	for i := 0; i < len(s); i++ {
		if err := w.WriteByte(s[i]); err != nil {
			return i, err
		}
	}

	return len(s), nil
}

func (w *SimpleWriter) WriteByte(b byte) error {
	step := w.state.Next(b)
	if step.IsPrinting() {
		if !w.skipIndent {
			w.state.WriteResetSequence(&w.buf)
			for i := 0; i < int(w.Indent); i++ {
				if err := w.buf.WriteByte(' '); err != nil {
					return err
				}
			}

			w.skipIndent = true
			w.state.WriteRestoreSequence(&w.buf)
		}

		if b == '\n' {
			// end of current line
			w.skipIndent = false
		}
	}

	return w.buf.WriteByte(b)
}

// Bytes returns the indented result as a byte slice.
func (w *SimpleWriter) Bytes() []byte {
	return w.buf.Bytes()
}

// String returns the indented result as a string.
func (w *SimpleWriter) String() string {
	return w.buf.String()
}

//////////////////////
//                  //
// Advanced Writer  //
//                  //
//////////////////////

type IndentFunc func(w io.Writer)

// The "advanced" writer type, that allows for custom indentation functions
//
// Prefer using SimpleWriter where possible, as it's more efficient.
type AdvancedWriter struct {
	Indent     uint
	IndentFunc IndentFunc
	buf        bytes.Buffer
	state      statemachine.AnsiState
	Forward    io.Writer
	skipIndent bool
}

func NewAdvancedWriter(w io.Writer, indent uint, indentFunc IndentFunc) *AdvancedWriter {
	writer := &AdvancedWriter{
		Indent:     indent,
		IndentFunc: indentFunc,
		Forward:    w,
	}
	return writer
}

func (w *AdvancedWriter) Write(b []byte) (int, error) {
	var i int
	for i := 0; i < len(b); i++ {
		if err := w.WriteByte(b[i]); err != nil {
			return i, err
		}
	}
	return i, nil
}

func (w *AdvancedWriter) WriteString(s string) (int, error) {
	for i := 0; i < len(s); i++ {
		if err := w.WriteByte(s[i]); err != nil {
			return i, err
		}
	}

	return len(s), nil
}

func (w *AdvancedWriter) WriteByte(b byte) error {
	// buffer used to write single bytes to the Forwarded io.Writer
	var buf [1]byte
	step := w.state.Next(b)
	if step.IsPrinting() {
		if !w.skipIndent {
			if w.Forward != nil {
				if _, err := w.Forward.Write(w.state.ResetSequence()); err != nil {
					return err
				}
			} else {
				w.state.WriteResetSequence(&w.buf)
			}
			if w.IndentFunc != nil {
				// if we have an indent function, pass it a wrapped writer so we can
				// track any ansi transitions in the callback.
				var wrappedWriter *ansi.Writer
				if w.Forward != nil {
					wrappedWriter = ansi.NewWriterForState(w.state, w.Forward)
				} else {
					wrappedWriter = ansi.NewWriterForState(w.state, &w.buf)
				}
				for i := 0; i < int(w.Indent); i++ {
					w.IndentFunc(wrappedWriter)
				}
				// restore our internal state using the wrapped writer's state
				w.state = wrappedWriter.ExportState()
			} else {
				if w.Forward != nil {
					buf[0] = ' '
					for i := 0; i < int(w.Indent); i++ {
						if _, err := w.Forward.Write(buf[:]); err != nil {
							return err
						}
					}
				} else {
					for i := 0; i < int(w.Indent); i++ {
						w.buf.WriteByte(' ')
					}
				}
			}

			w.skipIndent = true
			if w.Forward != nil {
				if _, err := w.Forward.Write(w.state.RestoreSequence()); err != nil {
					return err
				}
			} else {
				w.state.WriteRestoreSequence(&w.buf)
			}
		}

		if b == '\n' {
			// end of current line
			w.skipIndent = false
		}
	}

	buf[0] = b
	if w.Forward != nil {
		_, err := w.Forward.Write(buf[:])
		return err
	} else {
		return w.buf.WriteByte(b)
	}
}

func (w *AdvancedWriter) Bytes() []byte {
	return w.buf.Bytes()
}

func (w *AdvancedWriter) String() string {
	return w.buf.String()
}
